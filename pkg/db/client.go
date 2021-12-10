package db

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
)

var _ Databaser = &Client{}

// Client implements the db.Databaser interface using
// AWS S3 and AWS DynamoDB.
type Client struct {
	bucketName     string
	tableName      string
	dynamoDBClient dynamoDBClient
	s3Client       s3Client
}

// New generates a Client pointer instance.
func New(newSession *session.Session, bucketName, tableName string) *Client {
	return &Client{
		bucketName:     bucketName,
		tableName:      tableName,
		dynamoDBClient: dynamodb.New(newSession),
		s3Client:       s3.New(newSession),
	}
}

type s3Client interface {
	PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

type dynamoDBClient interface {
	Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error)
	PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
}

// GetIDs implements the db.Databaser.GetIDs method
// using AWS DynamoDB.
func (c *Client) GetIDs(ctx context.Context) ([]string, error) {
	scanOutput, err := c.dynamoDBClient.Scan(&dynamodb.ScanInput{
		TableName: &c.tableName,
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

// StoreData implements the db.Databaser.StoreData
// method using AWS S3 and AWS DynamoDB.
func (c *Client) StoreData(ctx context.Context, id, url, title, summary, text string) error {
	_, err := c.dynamoDBClient.PutItem(&dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"id": {
				S: &id,
			},
			"url": {
				S: &url,
			},
			"title": {
				S: &title,
			},
			"summary": {
				S: &summary,
			},
		},
		TableName: &c.tableName,
	})
	if err != nil {
		return err
	}

	_, err = c.s3Client.PutObject(&s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(strings.NewReader(text)),
		Bucket: &c.bucketName,
		Key:    aws.String(id + ".md"),
	})

	return err
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
		TableName: &c.tableName,
	})
	if err != nil {
		return err
	}

	return nil
}
