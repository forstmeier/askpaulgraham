package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const answersFile = "paul_graham_answers.jsonl"

const (
	summaryModel = "davinci"
	answersModel = "curie"
)

var errNon200StatusCode = errors.New("nlp: non-200 status code response")

var _ NLPer = &Client{}

// Client implements the nlp.NLPer interface.
type Client struct {
	helper     helper
	bucketName string
	s3Client   s3Client
}

type s3Client interface {
	GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
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
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature"`
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
	data, err := json.Marshal(getSummaryReqJSON{
		Prompt:      text,
		Temperature: 0.7,
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
	getFilesRespBody := getFilesRespJSON{}
	if err := c.helper.sendRequest(
		http.MethodGet,
		"https://api.openai.com/v1/files",
		nil,
		&getFilesRespBody,
	); err != nil {
		return err
	}

	fileID := ""
	for _, file := range getFilesRespBody.Data {
		if file.Name == answersFile {
			fileID = file.ID
		}
	}

	if fileID != "" {
		if err := c.helper.sendRequest(
			http.MethodDelete,
			fmt.Sprintf("https://api.openai.com/v1/files/%s", fileID),
			nil,
			nil,
		); err != nil {
			return err
		}
	}

	getAnswersResp, err := c.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: &c.bucketName,
		Key:    aws.String(answersFile),
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

	var answersBuffer bytes.Buffer
	bufferEncoder := json.NewEncoder(&answersBuffer)
	for _, answer := range answers {
		if err := bufferEncoder.Encode(answer); err != nil {
			return err
		}
	}

	var answersForm bytes.Buffer
	writer := multipart.NewWriter(&answersForm)
	part, err := writer.CreateFormFile("file", answersFile)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, &answersBuffer)
	if err != nil {
		return err
	}

	if err := writer.WriteField("purpose", "answers"); err != nil {
		return err
	}

	if err := c.helper.sendRequest(
		http.MethodPost,
		"https://api.openai.com/v1/files",
		&answersForm,
		nil,
	); err != nil {
		return err
	}

	_, err = c.s3Client.PutObject(&s3.PutObjectInput{
		Bucket: &c.bucketName,
		Key:    aws.String(answersFile),
		Body:   bytes.NewReader(answersBuffer.Bytes()),
	})
	if err != nil {
		return err
	}

	return nil
}

type getAnswerReqJSON struct {
	Model           string   `json:"model"`
	Question        string   `json:"question"`
	Examples        []string `json:"examples"`
	ExamplesContext string   `json:"examples_context"`
	File            string   `json:"file"`
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
	); err != nil {
		return nil, err
	}

	fileID := ""
	for _, file := range getFilesRespBody.Data {
		if file.Name == answersFile {
			fileID = file.ID
		}
	}

	data := getAnswerReqJSON{
		Model:    answersModel,
		Question: question,
		Examples: []string{
			"Build something that solves a problem you have",
			"Charge others for the solution you built",
		},
		ExamplesContext: "What is the best way to start a company?",
		File:            fileID,
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	answers := getAnswersRespJSON{}
	if err := c.helper.sendRequest(
		http.MethodGet,
		"https://api.openai.com/v1/answers",
		bytes.NewReader(dataBytes),
		&answers,
	); err != nil {
		return nil, err
	}

	return answers.Answers, nil
}
