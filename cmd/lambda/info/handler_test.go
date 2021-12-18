package main

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"github.com/forstmeier/askpaulgraham/pkg/db"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

type mockDBClient struct {
	mockGetDataOutput       []db.Data
	mockGetDataError        error
	mockStoreQuestionsError error
}

func (m *mockDBClient) GetIDs(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockDBClient) GetSummaries(ctx context.Context) ([]db.Data, error) {
	return m.mockGetDataOutput, m.mockGetDataError
}

func (m *mockDBClient) StoreSummaries(ctx context.Context, summaries []db.Data) error {
	return nil
}

func (m *mockDBClient) StoreText(ctx context.Context, id, text string) error {
	return nil
}

func (m *mockDBClient) StoreQuestion(ctx context.Context, question string) error {
	return m.mockStoreQuestionsError
}

type mockNLPClient struct {
	mockGetAnswersOutput []string
	mockGetAnswersError  error
}

func (m *mockNLPClient) GetSummary(ctx context.Context, text string) (*string, error) {
	return nil, nil
}

func (m *mockNLPClient) SetAnswer(ctx context.Context, id, answer string) error {
	return nil
}

func (m *mockNLPClient) GetAnswers(ctx context.Context, question string) ([]string, error) {
	return m.mockGetAnswersOutput, m.mockGetAnswersError
}

func Test_handler(t *testing.T) {
	tests := []struct {
		description             string
		request                 events.APIGatewayProxyRequest
		mockGetDataOutput       []db.Data
		mockGetDataError        error
		mockStoreQuestionsError error
		mockGetAnswersOutput    []string
		mockGetAnswersError     error
		statusCode              int
		body                    string
	}{
		{
			description: "unsupported http method",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPut,
			},
			statusCode: http.StatusMethodNotAllowed,
			body:       `{"error":"method 'PUT' not allowed"}`,
		},
		{
			description: "error getting data",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
			},
			mockGetDataOutput: nil,
			mockGetDataError:  errors.New("mock get data error"),
			statusCode:        http.StatusInternalServerError,
			body:              `{"error":"mock get data error"}`,
		},
		{
			description: "successful get invocation",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
			},
			mockGetDataOutput: []db.Data{
				{
					ID:      "mock_id",
					URL:     "mock_url",
					Title:   "mock_title",
					Summary: "mock_summary",
				},
			},
			mockGetDataError: nil,
			statusCode:       http.StatusOK,
			body:             `{"message":"success","summaries":[{"id":"mock_id","url":"mock_url","title":"mock_title","summary":"mock_summary"}]}`,
		},
		{
			description: "error storing question",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Body:       `{"question":"mock_question"}`,
			},
			mockStoreQuestionsError: errors.New("mock store questions error"),
			statusCode:              http.StatusInternalServerError,
			body:                    `{"error":"mock store questions error"}`,
		},
		{
			description: "error getting answers",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Body:       `{"question":"mock_question"}`,
			},
			mockStoreQuestionsError: nil,
			mockGetAnswersOutput:    nil,
			mockGetAnswersError:     errors.New("mock get answers error"),
			statusCode:              http.StatusInternalServerError,
			body:                    `{"error":"mock get answers error"}`,
		},
		{
			description: "successful post invocation",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Body:       `{"question":"mock_question"}`,
			},
			mockStoreQuestionsError: nil,
			mockGetAnswersOutput:    []string{"mock_answer"},
			mockGetAnswersError:     nil,
			statusCode:              http.StatusOK,
			body:                    `{"message":"success","answer":"mock_answer"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			d := &mockDBClient{
				mockGetDataOutput:       test.mockGetDataOutput,
				mockGetDataError:        test.mockGetDataError,
				mockStoreQuestionsError: test.mockStoreQuestionsError,
			}

			n := &mockNLPClient{
				mockGetAnswersOutput: test.mockGetAnswersOutput,
				mockGetAnswersError:  test.mockGetAnswersError,
			}

			handlerFunc := handler(d, n)

			response, _ := handlerFunc(context.Background(), test.request)

			if response.StatusCode != test.statusCode {
				t.Errorf("incorrect status code, received: %d, expected: %d", response.StatusCode, test.statusCode)
			}

			if response.Body != test.body {
				t.Errorf("incorrect body, received: %q, expected: %q", response.Body, test.body)
			}
		})
	}
}
