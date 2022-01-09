package nlp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/forstmeier/askpaulgraham/pkg/dct"
)

func TestNew(t *testing.T) {
	client := New(session.New(), "api_key", "bucket_name")
	if client == nil {
		t.Errorf("incorrect client, received: %v", client)
	}
}

type mockHelper struct {
	t         *testing.T
	responses []response
}

type response struct {
	body  []byte
	error error
}

func (m *mockHelper) sendRequest(method, url string, body io.Reader, payload interface{}, headers map[string]string) error {
	if len(m.responses) == 0 {
		m.t.Fatal("no mock responses")
	}

	resp := response{}
	resp, m.responses = m.responses[0], m.responses[1:]

	if resp.error != nil {
		return resp.error
	}
	if resp.body != nil {
		if err := json.Unmarshal(resp.body, payload); err != nil {
			return err
		}
	}

	return nil
}

type mockS3Client struct {
	mockGetObjectOutput *s3.GetObjectOutput
	mockGetObjectError  error
}

func (m *mockS3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return m.mockGetObjectOutput, m.mockGetObjectError
}

func TestGetSummary(t *testing.T) {
	getSummaryErr := errors.New("mock get summary error")
	summary := "Mock Summary"

	tests := []struct {
		description string
		responses   []response
		summary     *string
		error       error
	}{
		{
			description: "error getting summary",
			responses: []response{
				{
					body:  nil,
					error: getSummaryErr,
				},
			},
			summary: nil,
			error:   getSummaryErr,
		},
		{
			description: "successful invocation",
			responses: []response{
				{
					body:  []byte(fmt.Sprintf(`{"choices": [{"text": %q}]}`, summary)),
					error: nil,
				},
			},
			summary: &summary,
			error:   nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			h := &mockHelper{
				t:         t,
				responses: test.responses,
			}

			c := &Client{
				helper: h,
			}

			summary, err := c.GetSummary(context.Background(), "mock text")
			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}

			if test.summary != nil && *summary != *test.summary {
				t.Errorf("incorrect summary, received: %v, expected: %v", summary, test.summary)
			}
		})
	}
}

func TestSetDocuments(t *testing.T) {
	setDocumentsErr := errors.New("mock set answers error")

	tests := []struct {
		description string
		responses   []response
		error       error
	}{
		{
			description: "error setting documents",
			responses: []response{
				{
					body:  nil,
					error: setDocumentsErr,
				},
			},
			error: setDocumentsErr,
		},
		{
			description: "successful invocation",
			responses: []response{
				{
					body:  nil,
					error: nil,
				},
			},
			error: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			c := &Client{
				helper: &mockHelper{
					t:         t,
					responses: test.responses,
				},
				bucketName: "bucket_name",
			}

			err := c.SetDocuments(context.Background(), []dct.Document{
				{
					Text:     "mock answer",
					Metadata: "mock_id",
				},
			})
			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}
		})
	}
}

func TestGetAnswers(t *testing.T) {
	getFilesErr := errors.New("mock get files error")
	getAnswersErr := errors.New("mock get answers error")

	tests := []struct {
		description string
		responses   []response
		answers     []string
		error       error
	}{
		{
			description: "error getting file",
			responses: []response{
				{
					body:  nil,
					error: getFilesErr,
				},
			},
			answers: nil,
			error:   getFilesErr,
		},
		{
			description: "error getting answers",
			responses: []response{
				{
					body:  []byte(fmt.Sprintf(`{"data": [{"id": "mock_id", "filename": %q}]}`, documentsFilename)),
					error: nil,
				},
				{
					body:  nil,
					error: getAnswersErr,
				},
			},
			answers: nil,
			error:   getAnswersErr,
		},
		{
			description: "successful invocation",
			responses: []response{
				{
					body:  []byte(fmt.Sprintf(`{"data": [{"id": "mock_id", "filename": %q}]}`, documentsFilename)),
					error: nil,
				},
				{
					body:  []byte(`{"answers": [" answer "]}`),
					error: nil,
				},
			},
			answers: []string{"Answer"},
			error:   nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			h := &mockHelper{
				t:         t,
				responses: test.responses,
			}

			c := &Client{
				helper: h,
			}

			answers, err := c.GetAnswers(context.Background(), "question")
			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}
			if !reflect.DeepEqual(answers, test.answers) {
				t.Errorf("incorrect answers, received: %v, expected: %v", answers, test.answers)
			}
		})
	}
}
