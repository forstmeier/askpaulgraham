package db

import "context"

// Databaser defines methods for interacting with the
// storage layer of the application.
type Databaser interface {
	GetIDs(ctx context.Context) ([]string, error)
	StoreData(ctx context.Context, id, url, title, summary, text string) error
	StoreQuestion(ctx context.Context, question string) error
}
