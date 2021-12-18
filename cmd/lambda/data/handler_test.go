package main

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/forstmeier/askpaulgraham/pkg/cnt"
	"github.com/forstmeier/askpaulgraham/pkg/db"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

type mockCntClient struct {
	mockGetItemsOutput []cnt.ItemXML
	mockGetItemsError  error
	mockGetTextOutput  *string
	mockGetTextError   error
}

func (m *mockCntClient) GetItems(ctx context.Context, address string) ([]cnt.ItemXML, error) {
	return m.mockGetItemsOutput, m.mockGetItemsError
}

func (m *mockCntClient) GetText(ctx context.Context, address string) (*string, error) {
	return m.mockGetTextOutput, m.mockGetTextError
}

type mockDBClient struct {
	mockGetIDsOutput      []string
	mockGetIDsError       error
	mockStoreSummaryError error
	mockStoreTextError    error
}

func (m *mockDBClient) GetIDs(ctx context.Context) ([]string, error) {
	return m.mockGetIDsOutput, m.mockGetIDsError
}

func (m *mockDBClient) GetData(ctx context.Context) ([]db.Data, error) {
	return nil, nil
}

func (m *mockDBClient) StoreSummary(ctx context.Context, id, url, title, summary string) error {
	return m.mockStoreSummaryError
}

func (m *mockDBClient) StoreText(ctx context.Context, id, text string) error {
	return m.mockStoreTextError
}

func (m *mockDBClient) StoreQuestion(ctx context.Context, question string) error {
	return nil
}

type mockNLPClient struct {
	mockGetSummaryOutput *string
	mockGetSummaryError  error
	mockSetAnswerError   error
}

func (m *mockNLPClient) GetSummary(ctx context.Context, text string) (*string, error) {
	return m.mockGetSummaryOutput, m.mockGetSummaryError
}

func (m *mockNLPClient) SetAnswer(ctx context.Context, id, answer string) error {
	return m.mockSetAnswerError
}

func (m *mockNLPClient) GetAnswers(ctx context.Context, question string) ([]string, error) {
	return nil, nil
}

func Test_handler(t *testing.T) {
	getItemsErr := errors.New("mock get items error")
	getIDsErr := errors.New("mock get ids error")
	getTextErr := errors.New("mock get text error")
	getSummaryErr := errors.New("mock get summary error")
	storeSummaryErr := errors.New("mock store summary error")
	storeTextErr := errors.New("mock store text error")
	setAnswerErr := errors.New("mock set answer error")

	mockText := "full text"
	mockSummary := "summary"

	tests := []struct {
		description           string
		mockGetItemsOutput    []cnt.ItemXML
		mockGetItemsError     error
		mockGetIDsOutput      []string
		mockGetIDsError       error
		mockGetTextOutput     *string
		mockGetTextError      error
		mockGetSummaryOutput  *string
		mockGetSummaryError   error
		mockStoreSummaryError error
		mockStoreTextError    error
		mockSetAnswerError    error
		error                 error
	}{
		{
			description:        "error getting items",
			mockGetItemsOutput: nil,
			mockGetItemsError:  getItemsErr,
			error:              getItemsErr,
		},
		{
			description:        "error getting ids",
			mockGetItemsOutput: []cnt.ItemXML{},
			mockGetItemsError:  nil,
			mockGetIDsOutput:   nil,
			mockGetIDsError:    getIDsErr,
			error:              getIDsErr,
		},
		{
			description: "error getting text",
			mockGetItemsOutput: []cnt.ItemXML{
				{
					Link: "http://www.paulgraham.com/new_id.html",
				},
			},
			mockGetItemsError: nil,
			mockGetIDsOutput:  []string{"old_id"},
			mockGetIDsError:   nil,
			mockGetTextOutput: nil,
			mockGetTextError:  getTextErr,
			error:             getTextErr,
		},
		{
			description: "error getting summary",
			mockGetItemsOutput: []cnt.ItemXML{
				{
					Link: "http://www.paulgraham.com/new_id.html",
				},
			},
			mockGetItemsError:    nil,
			mockGetIDsOutput:     []string{"old_id"},
			mockGetIDsError:      nil,
			mockGetTextOutput:    &mockText,
			mockGetTextError:     nil,
			mockGetSummaryOutput: nil,
			mockGetSummaryError:  getSummaryErr,
			error:                getSummaryErr,
		},
		{
			description: "error storing summary",
			mockGetItemsOutput: []cnt.ItemXML{
				{
					Link: "http://www.paulgraham.com/new_id.html",
				},
			},
			mockGetItemsError:     nil,
			mockGetIDsOutput:      []string{"old_id"},
			mockGetIDsError:       nil,
			mockGetTextOutput:     &mockText,
			mockGetTextError:      nil,
			mockGetSummaryOutput:  &mockSummary,
			mockGetSummaryError:   nil,
			mockStoreSummaryError: storeSummaryErr,
			error:                 storeSummaryErr,
		},
		{
			description: "error storing text",
			mockGetItemsOutput: []cnt.ItemXML{
				{
					Link: "http://www.paulgraham.com/new_id.html",
				},
			},
			mockGetItemsError:     nil,
			mockGetIDsOutput:      []string{"old_id"},
			mockGetIDsError:       nil,
			mockGetTextOutput:     &mockText,
			mockGetTextError:      nil,
			mockGetSummaryOutput:  &mockSummary,
			mockGetSummaryError:   nil,
			mockStoreSummaryError: nil,
			mockStoreTextError:    storeTextErr,
			error:                 storeTextErr,
		},
		{
			description: "error setting answer",
			mockGetItemsOutput: []cnt.ItemXML{
				{
					Link: "http://www.paulgraham.com/new_id.html",
				},
			},
			mockGetItemsError:     nil,
			mockGetIDsOutput:      []string{"old_id"},
			mockGetIDsError:       nil,
			mockGetTextOutput:     &mockText,
			mockGetTextError:      nil,
			mockGetSummaryOutput:  &mockSummary,
			mockGetSummaryError:   nil,
			mockStoreSummaryError: nil,
			mockStoreTextError:    nil,
			mockSetAnswerError:    setAnswerErr,
			error:                 setAnswerErr,
		},
		{
			description: "successful invocation",
			mockGetItemsOutput: []cnt.ItemXML{
				{
					Link: "http://www.paulgraham.com/new_id.html",
				},
			},
			mockGetItemsError:     nil,
			mockGetIDsOutput:      []string{"old_id"},
			mockGetIDsError:       nil,
			mockGetTextOutput:     &mockText,
			mockGetTextError:      nil,
			mockGetSummaryOutput:  &mockSummary,
			mockGetSummaryError:   nil,
			mockStoreSummaryError: nil,
			mockStoreTextError:    nil,
			mockSetAnswerError:    nil,
			error:                 nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			c := &mockCntClient{
				mockGetItemsOutput: test.mockGetItemsOutput,
				mockGetItemsError:  test.mockGetItemsError,
				mockGetTextOutput:  test.mockGetTextOutput,
				mockGetTextError:   test.mockGetTextError,
			}

			d := &mockDBClient{
				mockGetIDsOutput:      test.mockGetIDsOutput,
				mockGetIDsError:       test.mockGetIDsError,
				mockStoreSummaryError: test.mockStoreSummaryError,
				mockStoreTextError:    test.mockStoreTextError,
			}

			n := &mockNLPClient{
				mockGetSummaryOutput: test.mockGetSummaryOutput,
				mockGetSummaryError:  test.mockGetSummaryError,
				mockSetAnswerError:   test.mockSetAnswerError,
			}

			handlerFunc := handler(c, d, n, "rss_url")

			err := handlerFunc(context.Background())

			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}
		})
	}
}
