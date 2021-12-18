package db

import "context"

// Databaser defines methods for interacting with the
// storage layer of the application.
type Databaser interface {
	GetIDs(ctx context.Context) ([]string, error)
	GetData(ctx context.Context) ([]Data, error)
	StoreSummary(ctx context.Context, id, url, title, summary string) error
	StoreText(ctx context.Context, id, text string) error
	StoreQuestion(ctx context.Context, question string) error
}

// Data represents a row in the database.
type Data struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}
