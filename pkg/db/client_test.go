package db

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
)

type mockDynamoDBClient struct {
	scanOutput   *dynamodb.ScanOutput
	scanError    error
	putItemError error
}

func (m *mockDynamoDBClient) Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
	return m.scanOutput, m.scanError
}

func (m *mockDynamoDBClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return nil, m.putItemError
}

type mockS3Client struct {
	putObjectError error
}

func (m *mockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return nil, m.putObjectError
}

func TestNew(t *testing.T) {
	client := New(session.New(), "bucket_name", "table_name")
	if client == nil {
		t.Errorf("incorrect client, received: %v", client)
	}
}

func TestGetIDs(t *testing.T) {
	scanErr := errors.New("mock scan error")

	tests := []struct {
		description string
		scanOutput  *dynamodb.ScanOutput
		scanError   error
		ids         []string
		error       error
	}{
		{
			description: "error scanning table",
			scanOutput:  nil,
			scanError:   scanErr,
			ids:         nil,
			error:       scanErr,
		},
		{
			description: "successful invocation",
			scanOutput: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					{
						"id": {
							S: aws.String("mock_id"),
						},
					},
				},
			},
			scanError: nil,
			ids:       []string{"mock_id"},
			error:     nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			d := &mockDynamoDBClient{
				scanOutput: test.scanOutput,
				scanError:  test.scanError,
			}

			c := &Client{
				dynamoDBClient: d,
			}

			ids, err := c.GetIDs(context.Background())

			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}

			if !reflect.DeepEqual(ids, test.ids) {
				t.Errorf("incorrect ids, received: %v, expected: %v", ids, test.ids)
			}
		})
	}
}

func TestStoreData(t *testing.T) {
	putItemErr := errors.New("mock put item error")
	putObjectErr := errors.New("mock put object error")

	tests := []struct {
		description    string
		putItemError   error
		putObjectError error
		error          error
	}{
		{
			description:    "error putting item",
			putItemError:   putItemErr,
			putObjectError: nil,
			error:          putItemErr,
		},
		{
			description:    "error putting object",
			putItemError:   nil,
			putObjectError: putObjectErr,
			error:          putObjectErr,
		},
		{
			description:    "successful invocation",
			putItemError:   nil,
			putObjectError: nil,
			error:          nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			d := &mockDynamoDBClient{
				putItemError: test.putItemError,
			}

			s := &mockS3Client{
				putObjectError: test.putObjectError,
			}

			c := &Client{
				dynamoDBClient: d,
				s3Client:       s,
			}

			err := c.StoreData(context.Background(), "mock_id", "url.com", "title", "short summary", "full text")

			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}
		})
	}
}

func TestStoreQuestion(t *testing.T) {
	putItemErr := errors.New("mock put item error")

	tests := []struct {
		description  string
		putItemError error
		error        error
	}{
		{
			description:  "error putting item",
			putItemError: putItemErr,
			error:        putItemErr,
		},
		{
			description:  "successful invocation",
			putItemError: nil,
			error:        nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			d := &mockDynamoDBClient{
				putItemError: test.putItemError,
			}

			c := &Client{
				dynamoDBClient: d,
			}

			err := c.StoreQuestion(context.Background(), "question")

			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}
		})
	}
}
