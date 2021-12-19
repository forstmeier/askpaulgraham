package db

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
)

const answersFilename = "answers.jsonl"

var _ Databaser = &Client{}

// Client implements the db.Databaser interface using
// AWS S3 and AWS DynamoDB.
type Client struct {
	bucketName         string
	questionsTableName string
	summariesTableName string
	dynamoDBClient     dynamoDBClient
	s3Client           s3Client
}

// New generates a Client pointer instance.
func New(newSession *session.Session, bucketName, questionsTableName, summariesTableName string) *Client {
	return &Client{
		bucketName:         bucketName,
		questionsTableName: questionsTableName,
		summariesTableName: summariesTableName,
		dynamoDBClient:     dynamodb.New(newSession),
		s3Client:           s3.New(newSession),
	}
}

type s3Client interface {
	GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

type dynamoDBClient interface {
	Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error)
	BatchWriteItem(input *dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error)
	PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
}

// GetIDs implements the db.Databaser.GetIDs method
// using AWS DynamoDB.
func (c *Client) GetIDs(ctx context.Context) ([]string, error) {
	scanOutput, err := c.dynamoDBClient.Scan(&dynamodb.ScanInput{
		TableName: &c.summariesTableName,
	})
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(scanOutput.Items))
	for i, item := range scanOutput.Items {
		ids[i] = *item["id"].S
	}

	return ids, nil
}

// GetSummaries implements the db.Databaser.GetSummaries
// method using AWS DynamoDB.
func (c *Client) GetSummaries(ctx context.Context) ([]Data, error) {
	scanOutput, err := c.dynamoDBClient.Scan(&dynamodb.ScanInput{
		TableName: &c.summariesTableName,
	})
	if err != nil {
		return nil, err
	}

	datas := make([]Data, len(scanOutput.Items))
	for i, item := range scanOutput.Items {
		datas[i] = Data{
			ID:      *item["id"].S,
			URL:     *item["url"].S,
			Title:   *item["title"].S,
			Summary: *item["summary"].S,
		}
	}

	return datas, nil
}

// StoreSummaries implements the db.Databaser.StoreSummaries
// method using AWS DynamoDB.
func (c *Client) StoreSummaries(ctx context.Context, summaries []Data) error {
	chunk := 25
	for i := 0; i < len(summaries); i += chunk {
		end := i + chunk
		if end > len(summaries) {
			end = len(summaries)
		}

		putRequests := []*dynamodb.WriteRequest{}
		for _, summary := range summaries[i:end] {
			putRequests = append(putRequests, &dynamodb.WriteRequest{
				PutRequest: &dynamodb.PutRequest{
					Item: map[string]*dynamodb.AttributeValue{
						"id": {
							S: aws.String(summary.ID),
						},
						"url": {
							S: aws.String(summary.URL),
						},
						"title": {
							S: aws.String(summary.Title),
						},
						"summary": {
							S: aws.String(summary.Summary),
						},
					},
				},
			})
		}

		_, err := c.dynamoDBClient.BatchWriteItem(&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				c.summariesTableName: putRequests,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// StoreText implements the db.Databaser.StoreText
// method using AWS S3.
func (c *Client) StoreText(ctx context.Context, id, text string) error {
	_, err := c.s3Client.PutObject(&s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(strings.NewReader(text)),
		Bucket: &c.bucketName,
		Key:    aws.String(id + ".md"),
	})

	return err
}

// GetAnswers implements the db.Databaser.GetAnswers
// method using AWS S3.
func (c *Client) GetAnswers(ctx context.Context) ([]Answer, error) {
	response, err := c.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: &c.bucketName,
		Key:    aws.String(answersFilename),
	})
	if err != nil {
		return nil, err
	}

	answers := []Answer{}
	decoder := json.NewDecoder(response.Body)
	for decoder.More() {
		var answer Answer
		if err := decoder.Decode(&answer); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		answers = append(answers, answer)
	}

	return answers, nil
}

// StoreAnswers implements the db.Databaser.StoreAnswers
// method using AWS S3.
func (c *Client) StoreAnswers(ctx context.Context, answers []Answer) error {
	answersBody := bytes.Buffer{}

	encoder := json.NewEncoder(&answersBody)
	for _, answer := range answers {
		if err := encoder.Encode(answer); err != nil {
			return err
		}
	}

	_, err := c.s3Client.PutObject(&s3.PutObjectInput{
		Bucket: &c.bucketName,
		Key:    aws.String(answersFilename),
		Body:   bytes.NewReader(answersBody.Bytes()),
	})
	if err != nil {
		return err
	}

	return nil
}

// StoreQuestion implements the db.Databaser.StoreQuestion
// method using AWS DynamoDB.
func (c *Client) StoreQuestion(ctx context.Context, question string) error {
	now := time.Now().String()
	_, err := c.dynamoDBClient.PutItem(&dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"question": {
				S: &question,
			},
			"timestamp": {
				S: &now,
			},
		},
		TableName: &c.questionsTableName,
	})
	if err != nil {
		return err
	}

	return nil
}
