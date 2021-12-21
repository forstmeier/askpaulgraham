package nlp

import (
	"context"

	"github.com/forstmeier/askpaulgraham/pkg/dct"
)

// NLPer defines the methods for interacting with the
// OpenAI natural language processing API.
type NLPer interface {
	GetSummary(ctx context.Context, text string) (*string, error)
	SetDocuments(ctx context.Context, documents []dct.Document) error
	GetAnswers(ctx context.Context, question string) ([]string, error)
}
