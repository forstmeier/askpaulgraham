package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const answersFilename = "answers.jsonl"

const (
	summaryModel = "curie"
	answersModel = "curie"
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
	// PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
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
	Prompt           string  `json:"prompt"`
	MaxTokens        int     `json:"max_tokens"`
	Temperature      float64 `json:"temperature"`
	TopP             float64 `json:"top_p"`
	FrequencyPenalty float64 `json:"frequency_penalty"`
	PresencePenalty  float64 `json:"presence_penalty"`
}

type getSummaryRespJSON struct {
	Choices []getSummaryRespChoiceJSON `json:"choices"`
	Created int                        `json:"created"`
}

type getSummaryRespChoiceJSON struct {
	Text string `json:"text"`
}

// GetSummary implements the nlp.NLPer.GetSummary method.
func (c *Client) GetSummary(ctx context.Context, text string) (*string, error) {
	// check to work within OpenAI token limits
	wordCount := strings.Split(text, " ")
	if len(wordCount) > 1500 {
		message := "Surpassed maximum word count permitted by OpenAI"
		return &message, nil
	}

	data, err := json.Marshal(getSummaryReqJSON{
		Prompt:           text + "\n\ntl;dr",
		MaxTokens:        60,
		Temperature:      0.3,
		TopP:             1.0,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
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

	return &responseBody.Choices[0].Text, nil
}

type getFilesRespJSON struct {
	Data []getFilesRespDataJSON `json:"data"`
}

type getFilesRespDataJSON struct {
	ID   string `json:"id"`
	Name string `json:"filename"`
}

type answerJSON struct {
	Text     string `json:"text"`
	Metadata string `json:"metadata"`
}

// SetAnswer implements the nlp.NLPer.SetAnswer method.
func (c *Client) SetAnswer(ctx context.Context, id, answer string) error {
	getAnswersResp, err := c.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: &c.bucketName,
		Key:    aws.String(answersFilename),
	})
	if err != nil {
		return err
	}

	answers := []answerJSON{
		{
			Text:     answer,
			Metadata: id,
		},
	}
	decoder := json.NewDecoder(getAnswersResp.Body)
	for decoder.More() {
		var answer answerJSON
		if err := decoder.Decode(&answer); err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if answer.Metadata != id {
			answers = append(answers, answer)
		}
	}

	answersBody := bytes.Buffer{}
	encoder := json.NewEncoder(&answersBody)
	for _, answer := range answers {
		if err := encoder.Encode(answer); err != nil {
			return err
		}
	}

	openAIAnswersBody := bytes.Buffer{}
	multipartWriter := multipart.NewWriter(&openAIAnswersBody)

	var fileWriter, purposeWriter io.Writer

	purposeWriter, err = multipartWriter.CreateFormField("purpose")
	if err != nil {
		return err
	}

	_, err = io.Copy(purposeWriter, strings.NewReader("answers"))
	if err != nil {
		return err
	}

	fileWriter, err = multipartWriter.CreateFormFile("file", answersFilename)
	if err != nil {
		return err
	}

	_, err = io.Copy(fileWriter, &answersBody)
	if err != nil {
		return err
	}

	multipartWriter.Close()

	if err := c.helper.sendRequest(
		http.MethodPost,
		"https://api.openai.com/v1/files",
		&openAIAnswersBody,
		nil,
		map[string]string{
			"Content-Type": multipartWriter.FormDataContentType(),
		},
	); err != nil {
		return err
	}

	return nil
}

type getAnswerReqJSON struct {
	Model           string     `json:"model"`
	Question        string     `json:"question"`
	Examples        [][]string `json:"examples"`
	ExamplesContext string     `json:"examples_context"`
	File            string     `json:"file"`
}

type getAnswersRespJSON struct {
	Answers []string `json:"answers"`
}

// GetAnswers implements the nlp.NLPer.GetAnswer method.
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
		if file.Name == answersFilename {
			fileID = file.ID
		}
	}

	data := getAnswerReqJSON{
		Model:    answersModel,
		Question: question,
		Examples: [][]string{
			{
				"What is the best way to start a company?",
				"How do I get users for my startup product?",
			},
		},
		ExamplesContext: "Build something that solves a problem you have",
		File:            fileID,
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	answers := getAnswersRespJSON{}
	if err := c.helper.sendRequest(
		http.MethodPost,
		"https://api.openai.com/v1/answers",
		bytes.NewReader(dataBytes),
		&answers,
		map[string]string{
			"Content-Type": "application/json",
		},
	); err != nil {
		return nil, err
	}

	return answers.Answers, nil
}
