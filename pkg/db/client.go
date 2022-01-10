package db

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/forstmeier/askpaulgraham/pkg/dct"
)

const documentsFilename = "documents.jsonl"

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
	UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
}

// GetIDs implements the db.Databaser.GetIDs method
// using AWS DynamoDB and returns a slice of the IDs
// of the items stored in the "summaries" table.
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
// method using AWS DynamoDB and returns a slice of structs
// representing the rows stored in the "summaries" table.
func (c *Client) GetSummaries(ctx context.Context) ([]Summary, error) {
	scanOutput, err := c.dynamoDBClient.Scan(&dynamodb.ScanInput{
		TableName: &c.summariesTableName,
	})
	if err != nil {
		return nil, err
	}

	datas := make([]Summary, len(scanOutput.Items))
	for i, item := range scanOutput.Items {
		number, err := strconv.Atoi(*item["number"].N)
		if err != nil {
			return nil, err
		}

		datas[i] = Summary{
			ID:      *item["id"].S,
			URL:     *item["url"].S,
			Title:   *item["title"].S,
			Summary: *item["summary"].S,
			Number:  number,
		}
	}

	return datas, nil
}

// StoreSummaries implements the db.Databaser.StoreSummaries
// method using AWS DynamoDB and stores the provided slice of
// structs in the "summaries" table.
func (c *Client) StoreSummaries(ctx context.Context, summaries []Summary) error {
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
						"number": {
							N: aws.String(strconv.Itoa(summary.Number)),
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
// method using AWS S3 and stores the provided text as
// a Markdown file.
//
// This Markdown file is not used by the application,
func (c *Client) StoreText(ctx context.Context, id, text string) error {
	_, err := c.s3Client.PutObject(&s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(strings.NewReader(text)),
		Bucket: &c.bucketName,
		Key:    aws.String(id + ".md"),
	})

	return err
}

// GetDocuments implements the db.Databaser.GetDocuments
// method using AWS S3 and returns a slice of structs
// representing the rows in the documents.jsonl file.
func (c *Client) GetDocuments(ctx context.Context) ([]dct.Document, error) {
	response, err := c.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: &c.bucketName,
		Key:    aws.String(documentsFilename),
	})
	if err != nil {
		return nil, err
	}

	documents := []dct.Document{}
	decoder := json.NewDecoder(response.Body)
	for decoder.More() {
		var document dct.Document
		if err := decoder.Decode(&document); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		documents = append(documents, document)
	}

	return documents, nil
}

// StoreDocuments implements the db.Databaser.StoreDocuments
// method using AWS S3 and stores the provided slice of structs
// representing the documents.jsonl file and replaces it in storage.
func (c *Client) StoreDocuments(ctx context.Context, documents []dct.Document) error {
	documentsBody := bytes.Buffer{}

	encoder := json.NewEncoder(&documentsBody)
	for _, document := range documents {
		if err := encoder.Encode(document); err != nil {
			return err
		}
	}

	_, err := c.s3Client.PutObject(&s3.PutObjectInput{
		Bucket: &c.bucketName,
		Key:    aws.String(documentsFilename),
		Body:   bytes.NewReader(documentsBody.Bytes()),
	})
	if err != nil {
		return err
	}

	return nil
}

// StoreQuestion implements the db.Databaser.StoreQuestion
// method using AWS DynamoDB and stores the received user
// question in the "questions" table.
func (c *Client) StoreQuestion(ctx context.Context, id, question string) error {
	now := time.Now().String()
	_, err := c.dynamoDBClient.PutItem(&dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"id": {
				S: &id,
			},
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

// StoreAnswer implements the db.Databaser.StoreAnswer
// method using AWS DynamoDB and stores the received answer
// generated by OpenAI in the "questions" table.
func (c *Client) StoreAnswer(ctx context.Context, id, answer string) error {
	_, err := c.dynamoDBClient.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":answer": {
				S: aws.String(answer),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: &id,
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set answer = :answer"),
		TableName:        &c.questionsTableName,
	})
	if err != nil {
		return err
	}

	return nil
}
