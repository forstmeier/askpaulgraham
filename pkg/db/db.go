package db

import (
	"context"

	"github.com/forstmeier/askpaulgraham/pkg/dct"
)

// Databaser defines methods for interacting with the
// storage layer of the application.
type Databaser interface {
	GetIDs(ctx context.Context) ([]string, error)
	GetSummaries(ctx context.Context) ([]Summary, error)
	StoreSummaries(ctx context.Context, summaries []Summary) error
	StoreText(ctx context.Context, id, text string) error
	GetDocuments(ctx context.Context) ([]dct.Document, error)
	StoreDocuments(ctx context.Context, answers []dct.Document) error
	StoreQuestion(ctx context.Context, id, question string) error
	StoreAnswer(ctx context.Context, id, answer string) error
}

// Summary represents a row in the summaries table.
type Summary struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Number  int    `json:"number"`
}
