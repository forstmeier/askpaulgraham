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
	mockScanOutput          *dynamodb.ScanOutput
	mockScanError           error
	mockBatchWriteItemError error
	mockPutItemError        error
}

func (m *mockDynamoDBClient) Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
	return m.mockScanOutput, m.mockScanError
}

func (m *mockDynamoDBClient) BatchWriteItem(input *dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error) {
	return nil, m.mockBatchWriteItemError
}

func (m *mockDynamoDBClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return nil, m.mockPutItemError
}

type mockS3Client struct {
	mockPutObjectError error
}

func (m *mockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return nil, m.mockPutObjectError
}

func TestNew(t *testing.T) {
	client := New(session.New(), "bucket_name", "questions_table_name", "summaries_table_name")
	if client == nil {
		t.Errorf("incorrect client, received: %v", client)
	}
}

func TestGetIDs(t *testing.T) {
	mockScanErr := errors.New("mock scan error")

	tests := []struct {
		description    string
		mockScanOutput *dynamodb.ScanOutput
		mockScanError  error
		ids            []string
		error          error
	}{
		{
			description:    "error scanning table",
			mockScanOutput: nil,
			mockScanError:  mockScanErr,
			ids:            nil,
			error:          mockScanErr,
		},
		{
			description: "successful invocation",
			mockScanOutput: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					{
						"id": {
							S: aws.String("mock_id"),
						},
					},
				},
			},
			mockScanError: nil,
			ids:           []string{"mock_id"},
			error:         nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			d := &mockDynamoDBClient{
				mockScanOutput: test.mockScanOutput,
				mockScanError:  test.mockScanError,
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

func TestGetData(t *testing.T) {
	mockScanErr := errors.New("mock scan error")

	tests := []struct {
		description    string
		mockScanOutput *dynamodb.ScanOutput
		mockScanError  error
		datas          []Data
		error          error
	}{
		{
			description:    "error scanning table",
			mockScanOutput: nil,
			mockScanError:  mockScanErr,
			datas:          nil,
			error:          mockScanErr,
		},
		{
			description: "successful invocation",
			mockScanOutput: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					{
						"id": {
							S: aws.String("mock_id"),
						},
						"url": {
							S: aws.String("mock_url"),
						},
						"title": {
							S: aws.String("mock_title"),
						},
						"summary": {
							S: aws.String("mock_summary"),
						},
					},
				},
			},
			mockScanError: nil,
			datas: []Data{
				{
					ID:      "mock_id",
					URL:     "mock_url",
					Title:   "mock_title",
					Summary: "mock_summary",
				},
			},
			error: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			d := &mockDynamoDBClient{
				mockScanOutput: test.mockScanOutput,
				mockScanError:  test.mockScanError,
			}

			c := &Client{
				dynamoDBClient: d,
			}

			datas, err := c.GetSummaries(context.Background())

			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}

			if !reflect.DeepEqual(datas, test.datas) {
				t.Errorf("incorrect data, received: %v, expected: %v", datas, test.datas)
			}
		})
	}
}

func TestStoreSummaries(t *testing.T) {
	mockBatchWriteItemErr := errors.New("mock batch write item error")

	tests := []struct {
		description             string
		mockBatchWriteItemError error
		error                   error
	}{
		{
			description:             "error putting item",
			mockBatchWriteItemError: mockBatchWriteItemErr,
			error:                   mockBatchWriteItemErr,
		},
		{
			description:             "successful invocation",
			mockBatchWriteItemError: nil,
			error:                   nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			d := &mockDynamoDBClient{
				mockBatchWriteItemError: test.mockBatchWriteItemError,
			}

			c := &Client{
				dynamoDBClient: d,
			}

			err := c.StoreSummaries(context.Background(), []Data{
				{
					ID:      "mock_id",
					URL:     "url.com",
					Title:   "title",
					Summary: "short summary",
				},
			})

			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}
		})
	}
}

func TestStoreText(t *testing.T) {
	mockPutObjectErr := errors.New("mock put object error")

	tests := []struct {
		description        string
		mockPutObjectError error
		error              error
	}{
		{
			description:        "error putting object",
			mockPutObjectError: mockPutObjectErr,
			error:              mockPutObjectErr,
		},
		{
			description:        "successful invocation",
			mockPutObjectError: nil,
			error:              nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			s := &mockS3Client{
				mockPutObjectError: test.mockPutObjectError,
			}

			c := &Client{
				s3Client: s,
			}

			err := c.StoreText(context.Background(), "mock_id", "full text")

			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}
		})
	}
}

func TestStoreQuestion(t *testing.T) {
	mockPutItemErr := errors.New("mock put item error")

	tests := []struct {
		description      string
		mockPutItemError error
		error            error
	}{
		{
			description:      "error putting item",
			mockPutItemError: mockPutItemErr,
			error:            mockPutItemErr,
		},
		{
			description:      "successful invocation",
			mockPutItemError: nil,
			error:            nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			d := &mockDynamoDBClient{
				mockPutItemError: test.mockPutItemError,
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
