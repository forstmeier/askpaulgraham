package db

import "context"

// Databaser defines methods for interacting with the
// storage layer of the application.
type Databaser interface {
	GetIDs(ctx context.Context) ([]string, error)
	GetSummaries(ctx context.Context) ([]Summary, error)
	StoreSummaries(ctx context.Context, summaries []Summary) error
	StoreText(ctx context.Context, id, text string) error
	GetAnswers(ctx context.Context) ([]Answer, error)
	StoreAnswers(ctx context.Context, answers []Answer) error
	StoreQuestion(ctx context.Context, id, question string) error
	StoreAnswer(ctx context.Context, id, answer string) error
}

// Summary represents a row in the summaries table.
type Summary struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// Answer represents a row in the answers.jsonl file.
type Answer struct {
	Text     string `json:"text"`
	Metadata string `json:"metadata"`
}
