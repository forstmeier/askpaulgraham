package cnt

import (
	"context"
	"encoding/xml"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

var _ Contenter = &Client{}

// Client implements the cnt.Contenter interface.
type Client struct{}

// New generates a pointer instance of Client.
func New() *Client {
	return &Client{}
}

// GetItems implements the cnt.Contenter.GetItems method.
func (c *Client) GetItems(ctx context.Context, address string) ([]ItemXML, error) {
	response, err := http.Get(address)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	rss := &RSSXML{}
	decoder := xml.NewDecoder(response.Body)
	if err := decoder.Decode(&rss); err != nil {
		return nil, err
	}

	return rss.Channel.Items, nil
}

// GetText implements the cnt.Contenter.GetText method.
func (c *Client) GetText(ctx context.Context, address string) (*string, error) {
	response, err := http.Get(address)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, err
	}

	text := document.Find("tbody").First().Text()
	return &text, nil
}
