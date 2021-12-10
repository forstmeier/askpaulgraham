package nlp

import "context"

// NLPer defines the methods for interacting with the
// OpenAI natural language processing API.
type NLPer interface {
	GetSummary(ctx context.Context, text string) (*string, error)
	SetAnswer(ctx context.Context, id, answer string) error
	GetAnswers(ctx context.Context, question string) ([]string, error)
}
