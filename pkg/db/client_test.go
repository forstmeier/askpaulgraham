package db

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/forstmeier/askpaulgraham/pkg/dct"
)

type mockDynamoDBClient struct {
	mockScanOutput          *dynamodb.ScanOutput
	mockScanError           error
	mockBatchWriteItemError error
	mockPutItemError        error
	mockUpdateItemError     error
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

func (m *mockDynamoDBClient) UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	return nil, m.mockUpdateItemError
}

type mockS3Client struct {
	mockGetObjectOutput *s3.GetObjectOutput
	mockGetObjectError  error
	mockPutObjectError  error
}

func (m *mockS3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return m.mockGetObjectOutput, m.mockGetObjectError
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
			c := &Client{
				dynamoDBClient: &mockDynamoDBClient{
					mockScanOutput: test.mockScanOutput,
					mockScanError:  test.mockScanError,
				},
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

func TestGetSummaries(t *testing.T) {
	mockScanErr := errors.New("mock scan error")

	tests := []struct {
		description    string
		mockScanOutput *dynamodb.ScanOutput
		mockScanError  error
		summaries      []Summary
		error          error
	}{
		{
			description:    "error scanning table",
			mockScanOutput: nil,
			mockScanError:  mockScanErr,
			summaries:      nil,
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
						"number": {
							N: aws.String("1"),
						},
					},
				},
			},
			mockScanError: nil,
			summaries: []Summary{
				{
					ID:      "mock_id",
					URL:     "mock_url",
					Title:   "mock_title",
					Summary: "mock_summary",
					Number:  1,
				},
			},
			error: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			c := &Client{
				dynamoDBClient: &mockDynamoDBClient{
					mockScanOutput: test.mockScanOutput,
					mockScanError:  test.mockScanError,
				},
			}

			summaries, err := c.GetSummaries(context.Background())

			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}

			if !reflect.DeepEqual(summaries, test.summaries) {
				t.Errorf("incorrect summaries, received: %v, expected: %v", summaries, test.summaries)
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
			c := &Client{
				dynamoDBClient: &mockDynamoDBClient{
					mockBatchWriteItemError: test.mockBatchWriteItemError,
				},
			}

			err := c.StoreSummaries(context.Background(), []Summary{
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
			c := &Client{
				s3Client: &mockS3Client{
					mockPutObjectError: test.mockPutObjectError,
				},
			}

			err := c.StoreText(context.Background(), "mock_id", "full text")

			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}
		})
	}
}

func TestGetDocuments(t *testing.T) {
	mockGetObjectErr := errors.New("mock get object error")

	tests := []struct {
		description         string
		mockGetObjectOutput *s3.GetObjectOutput
		mockGetObjectError  error
		documents           []dct.Document
		error               error
	}{
		{
			description:         "error getting object",
			mockGetObjectOutput: nil,
			mockGetObjectError:  mockGetObjectErr,
			documents:           nil,
			error:               mockGetObjectErr,
		},
		{
			description: "successful invocation",
			mockGetObjectOutput: &s3.GetObjectOutput{
				Body: aws.ReadSeekCloser(strings.NewReader(`{"text": "example text", "metadata": "example metadata"}`)),
			},
			mockGetObjectError: nil,
			documents: []dct.Document{
				{
					Text:     "example text",
					Metadata: "example metadata",
				},
			},
			error: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			c := &Client{
				s3Client: &mockS3Client{
					mockGetObjectOutput: test.mockGetObjectOutput,
					mockGetObjectError:  test.mockGetObjectError,
				},
			}

			documents, err := c.GetDocuments(context.Background())

			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}

			if !reflect.DeepEqual(documents, test.documents) {
				t.Errorf("incorrect documents, received: %v, expected: %v", documents, test.documents)
			}
		})
	}
}

func TestStoreDocuments(t *testing.T) {
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
			c := &Client{
				s3Client: &mockS3Client{
					mockPutObjectError: test.mockPutObjectError,
				},
			}

			err := c.StoreDocuments(context.Background(), []dct.Document{
				{
					Text:     "example text",
					Metadata: "example metadata",
				},
			})

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
			c := &Client{
				dynamoDBClient: &mockDynamoDBClient{
					mockPutItemError: test.mockPutItemError,
				},
			}

			err := c.StoreQuestion(context.Background(), "id", "question")

			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}
		})
	}
}

func TestStoreAnswer(t *testing.T) {
	mockUpdateItemErr := errors.New("mock update item error")

	tests := []struct {
		description         string
		mockUpdateItemError error
		error               error
	}{
		{
			description:         "error updating item",
			mockUpdateItemError: mockUpdateItemErr,
			error:               mockUpdateItemErr,
		},
		{
			description:         "successful invocation",
			mockUpdateItemError: nil,
			error:               nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			c := &Client{
				dynamoDBClient: &mockDynamoDBClient{
					mockUpdateItemError: test.mockUpdateItemError,
				},
			}

			err := c.StoreAnswer(context.Background(), "id", "answer")

			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}
		})
	}
}
