package cnt

import "context"

// Contenter defines methods for interacting with the
// root blog content.
type Contenter interface {
	GetItems(ctx context.Context, address string) ([]ItemXML, error)
	GetText(ctx context.Context, address string) (*string, error)
}

// RSSXML represents the target RSS feed.
type RSSXML struct {
	Channel ChannelXML `xml:"channel"`
}

// ChannelXML represents the list of items in the RSS feed.
type ChannelXML struct {
	Items []ItemXML `xml:"item"`
}

// ItemXML represents an object in the target RSS feed.
type ItemXML struct {
	Link  string `xml:"link"`
	Title string `xml:"title"`
}
