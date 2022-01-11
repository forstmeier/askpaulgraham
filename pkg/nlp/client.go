package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/forstmeier/askpaulgraham/pkg/dct"
)

const documentsFilename = "documents.jsonl"

const (
	summaryModel = "curie"
	answersModel = "curie"
)

const (
	maxContextTokenLength = 2049 // most OpenAI model max
	maxTokens             = 60
	temperature           = 0.50
)

var _ NLPer = &Client{}

// Client implements the nlp.NLPer interface.
type Client struct {
	helper     helper
	bucketName string
	s3Client   s3Client
}

type s3Client interface {
	GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
}

// New generates a pointer instance of Client.
func New(newSession *session.Session, apiKey, bucketName string) *Client {
	return &Client{
		helper: &help{
			apiKey:     apiKey,
			httpClient: http.Client{},
		},
		bucketName: bucketName,
		s3Client:   s3.New(newSession),
	}
}

type getSummaryReqJSON struct {
	Prompt           string   `json:"prompt"`
	MaxTokens        int      `json:"max_tokens"`
	Temperature      float64  `json:"temperature"`
	TopP             float64  `json:"top_p"`
	FrequencyPenalty float64  `json:"frequency_penalty"`
	PresencePenalty  float64  `json:"presence_penalty"`
	Stop             []string `json:"stop"`
}

type getSummaryRespJSON struct {
	Choices []getSummaryRespChoiceJSON `json:"choices"`
	Created int                        `json:"created"`
}

type getSummaryRespChoiceJSON struct {
	Text string `json:"text"`
}

// GetSummary implements the nlp.NLPer.GetSummary method
// and generates a summary of the provided text with OpenAI.
func (c *Client) GetSummary(ctx context.Context, text string) (*string, error) {
	// approximate check to work within OpenAI token limits
	characters := len(text)
	if (characters / 4) > (maxContextTokenLength - maxTokens) {
		message := "Surpassed maximum word count permitted by OpenAI."
		return &message, nil
	}

	data, err := json.Marshal(getSummaryReqJSON{
		Prompt:           text + "\n\ntl;dr:",
		MaxTokens:        maxTokens,
		Temperature:      temperature,
		TopP:             1.0,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
		Stop:             []string{".", "<|endoftext|>"},
	})
	if err != nil {
		return nil, err
	}

	responseBody := getSummaryRespJSON{}
	if err := c.helper.sendRequest(
		http.MethodPost,
		fmt.Sprintf("https://api.openai.com/v1/engines/%s/completions", summaryModel),
		bytes.NewReader(data),
		&responseBody,
		map[string]string{
			"Content-Type": "application/json",
		},
	); err != nil {
		return nil, err
	}

	summary := formatString(responseBody.Choices[0].Text) + "."

	return &summary, nil
}

type getFilesRespJSON struct {
	Data []getFilesRespDataJSON `json:"data"`
}

type getFilesRespDataJSON struct {
	ID   string `json:"id"`
	Name string `json:"filename"`
}

type documentJSON struct {
	Text     string `json:"text"`
	Metadata string `json:"metadata"`
}

// SetDocuments implements the nlp.NLPer.SetDocuments method
// and stores the provided slice of structs representing the
// documents.jsonl file in OpenAI.
func (c *Client) SetDocuments(ctx context.Context, documents []dct.Document) error {
	documentsBody := bytes.Buffer{}
	encoder := json.NewEncoder(&documentsBody)
	re, err := regexp.Compile(`\w\.\w`)
	if err != nil {
		return err
	}

	for _, document := range documents {
		text := re.ReplaceAllStringFunc(document.Text, replaceFunc)
		for _, paragraph := range strings.Split(text, ".\n") {
			if paragraph == "" {
				continue
			}

			if err := encoder.Encode(dct.Document{
				Text:     paragraph,
				Metadata: document.Metadata,
			}); err != nil {
				return err
			}
		}
	}

	openAIDocumentsBody := bytes.Buffer{}
	multipartWriter := multipart.NewWriter(&openAIDocumentsBody)

	var fileWriter, purposeWriter io.Writer

	purposeWriter, err = multipartWriter.CreateFormField("purpose")
	if err != nil {
		return err
	}

	_, err = io.Copy(purposeWriter, strings.NewReader("answers"))
	if err != nil {
		return err
	}

	fileWriter, err = multipartWriter.CreateFormFile("file", documentsFilename)
	if err != nil {
		return err
	}

	_, err = io.Copy(fileWriter, &documentsBody)
	if err != nil {
		return err
	}

	multipartWriter.Close()

	if err := c.helper.sendRequest(
		http.MethodPost,
		"https://api.openai.com/v1/files",
		&openAIDocumentsBody,
		nil,
		map[string]string{
			"Content-Type": multipartWriter.FormDataContentType(),
		},
	); err != nil {
		return err
	}

	return nil
}

func replaceFunc(input string) string {
	return strings.Replace(input, ".", ".\n", -1)
}

type getAnswerReqJSON struct {
	Model           string     `json:"model"`
	Question        string     `json:"question"`
	Examples        [][]string `json:"examples"`
	ExamplesContext string     `json:"examples_context"`
	File            string     `json:"file"`
	Temperature     float64    `json:"temperature"`
	MaxTokens       int        `json:"max_tokens"`
	Stop            []string   `json:"stop"`
}

type getAnswersRespJSON struct {
	Answers []string `json:"answers"`
}

// GetAnswers implements the nlp.NLPer.GetAnswer method
// and generates answers to the provided question using OpenAI.
func (c *Client) GetAnswers(ctx context.Context, question string) ([]string, error) {
	getFilesRespBody := getFilesRespJSON{}
	if err := c.helper.sendRequest(
		http.MethodGet,
		"https://api.openai.com/v1/files",
		nil,
		&getFilesRespBody,
		map[string]string{
			"Content-Type": "application/json",
		},
	); err != nil {
		return nil, err
	}

	fileID := ""
	for _, file := range getFilesRespBody.Data {
		if file.Name == documentsFilename {
			fileID = file.ID
		}
	}

	data := getAnswerReqJSON{
		Model:    answersModel,
		Question: question,
		Examples: [][]string{
			{
				"What is the secret to a successful startup?",
				"What you need to succeed in a startup is not expertise in startups. What you need is expertise in your own users.",
			},
			{
				"What do I do to grow my company?",
				"The way to make your startup grow, is to make something users really love.",
			},
		},
		ExamplesContext: "Users are the most important thing to a startup.",
		File:            fileID,
		Temperature:     0.65,
		MaxTokens:       24,
		Stop: []string{
			"\n---",
			"\n===",
			".",
		},
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	responseBody := getAnswersRespJSON{}
	if err := c.helper.sendRequest(
		http.MethodPost,
		"https://api.openai.com/v1/answers",
		bytes.NewReader(dataBytes),
		&responseBody,
		map[string]string{
			"Content-Type": "application/json",
		},
	); err != nil {
		return nil, err
	}

	answers := []string{}
	for _, answer := range responseBody.Answers {
		answer = formatString(answer)
		answers = append(answers, answer)
	}

	return answers, nil
}

func formatString(input string) string {
	if input == "" || len(input) < 2 {
		return input
	}

	input = strings.TrimSpace(input)
	input = strings.ToUpper(string(input[0])) + string(input[1:])

	return input
}
