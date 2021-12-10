package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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

func (m *mockHelper) sendRequest(method, url string, body io.Reader, payload interface{}) error {
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
	getObjectOutput *s3.GetObjectOutput
	getObjectError  error
	putObjectError  error
}

func (m *mockS3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return m.getObjectOutput, m.getObjectError
}

func (m *mockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return nil, m.putObjectError
}

func TestGetSummary(t *testing.T) {
	getSummaryErr := errors.New("mock get summary error")
	summary := "mock summary"

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

func TestSetAnswer(t *testing.T) {
	getFilesErr := errors.New("mock get files error")
	deleteFileErr := errors.New("mock delete file error")
	getObjectErr := errors.New("mock get object error")
	setAnswersErr := errors.New("mock set answers error")
	putObjectErr := errors.New("mock put object error")

	tests := []struct {
		description     string
		responses       []response
		getObjectOutput *s3.GetObjectOutput
		getObjectError  error
		putObjectError  error
		error           error
	}{
		{
			description: "error getting files",
			responses: []response{
				{
					body:  nil,
					error: getFilesErr,
				},
			},
			getObjectOutput: nil,
			getObjectError:  nil,
			putObjectError:  nil,
			error:           getFilesErr,
		},
		{
			description: "error deleting file",
			responses: []response{
				{
					body:  []byte(fmt.Sprintf(`{"data": [{"id": "mock_id", "filename": %q}]}`, answersFile)),
					error: nil,
				},
				{
					body:  nil,
					error: deleteFileErr,
				},
			},
			getObjectOutput: nil,
			getObjectError:  nil,
			putObjectError:  nil,
			error:           deleteFileErr,
		},
		{
			description: "error getting object",
			responses: []response{
				{
					body:  []byte(fmt.Sprintf(`{"data": [{"id": "mock_id", "filename": %q}]}`, answersFile)),
					error: nil,
				},
				{
					body:  nil,
					error: nil,
				},
			},
			getObjectOutput: nil,
			getObjectError:  getObjectErr,
			putObjectError:  nil,
			error:           getObjectErr,
		},
		{
			description: "error setting answers",
			responses: []response{
				{
					body:  []byte(fmt.Sprintf(`{"data": [{"id": "mock_id", "filename": %q}]}`, answersFile)),
					error: nil,
				},
				{
					body:  nil,
					error: nil,
				},
				{
					body:  nil,
					error: setAnswersErr,
				},
			},
			getObjectOutput: &s3.GetObjectOutput{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"text": "first text", "metadata": "first_id"}
		{"text": "second text", "metadata": "second_id"}`))),
			},
			getObjectError: nil,
			putObjectError: nil,
			error:          setAnswersErr,
		},
		{
			description: "error putting object",
			responses: []response{
				{
					body:  []byte(fmt.Sprintf(`{"data": [{"id": "mock_id", "filename": %q}]}`, answersFile)),
					error: nil,
				},
				{
					body:  nil,
					error: nil,
				},
				{
					body:  nil,
					error: nil,
				},
			},
			getObjectOutput: &s3.GetObjectOutput{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"text": "first text", "metadata": "first_id"}
		{"text": "second text", "metadata": "second_id"}`))),
			},
			getObjectError: nil,
			putObjectError: putObjectErr,
			error:          putObjectErr,
		},
		{
			description: "successful invocation",
			responses: []response{
				{
					body:  []byte(fmt.Sprintf(`{"data": [{"id": "mock_id", "filename": %q}]}`, answersFile)),
					error: nil,
				},
				{
					body:  nil,
					error: nil,
				},
				{
					body:  nil,
					error: nil,
				},
			},
			getObjectOutput: &s3.GetObjectOutput{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"text": "first text", "metadata": "first_id"}
		{"text": "second text", "metadata": "second_id"}`))),
			},
			getObjectError: nil,
			putObjectError: nil,
			error:          nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			h := &mockHelper{
				t:         t,
				responses: test.responses,
			}

			s3C := &mockS3Client{
				getObjectOutput: test.getObjectOutput,
				getObjectError:  test.getObjectError,
				putObjectError:  test.putObjectError,
			}

			c := &Client{
				helper:     h,
				bucketName: "bucket_name",
				s3Client:   s3C,
			}

			err := c.SetAnswer(context.Background(), "mock_id", "mock answer")
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
					body:  []byte(fmt.Sprintf(`{"data": [{"id": "mock_id", "filename": %q}]}`, answersFile)),
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
					body:  []byte(fmt.Sprintf(`{"data": [{"id": "mock_id", "filename": %q}]}`, answersFile)),
					error: nil,
				},
				{
					body:  []byte(`{"answers": ["answer"]}`),
					error: nil,
				},
			},
			answers: []string{"answer"},
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
