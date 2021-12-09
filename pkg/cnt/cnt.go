package cnt

import "context"

// Contenter defines methods for interacting with the
// root blog content.
type Contenter interface {
	GetItems(ctx context.Context, address string) ([]Item, error)
	GetText(ctx context.Context, item Item) (*string, error)
}

// RSS represents the target RSS feed.
type RSS struct {
	Channel Channel `xml:"channel"`
}

// Channel represents the list of items in the RSS feed.
type Channel struct {
	Items []Item `xml:"item"`
}

// Item represents an object in the target RSS feed.
type Item struct {
	Link  string `xml:"link"`
	Title string `xml:"title"`
}
