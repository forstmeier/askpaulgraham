package cnt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
)

func TestGetItems(t *testing.T) {
	tests := []struct {
		description string
		body        string
		response    []ItemXML
		error       error
	}{
		{
			description: "error getting data from server",
			body:        "---",
			response:    nil,
			error:       io.EOF,
		},
		{
			description: "successful invocation",
			body: `<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:dc="http://purl.org/dc/elements/1.1/">
	<channel>
		<title>Paul Graham: Essays</title>
		<link>http://www.paulgraham.com/</link>
		<description>Scraped feed provided by aaronsw.com</description>
		<item>
		<link>http://www.paulgraham.com/goodtaste.html</link>
		<title>Is There Such a Thing as Good Taste?</title>
		</item>
	</channel>
</rss>`,
			response: []ItemXML{
				{
					Link:  "http://www.paulgraham.com/goodtaste.html",
					Title: "Is There Such a Thing as Good Taste?",
				},
			},
			error: nil,
		},
	}

	client := &Client{}

	urlPath := "/path.rss"

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc(urlPath, func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, test.body)
			})

			server := httptest.NewServer(mux)

			response, err := client.GetItems(context.Background(), server.URL+urlPath)
			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}

			if !reflect.DeepEqual(response, test.response) {
				t.Errorf("incorrect response, received: %+v, expected: %+v", response, test.response)
			}
		})
	}
}

func TestGetText(t *testing.T) {
	tests := []struct {
		description string
		body        string
		response    *string
		error       error
	}{
		{
			description: "successful invocation",
			body: `<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN">
<html><body><table><tbody><tr><td><table><tbody><tr><td><font>Example text right here.</font></td></tr></tbody></table></td></tr></tbody></table></body></html>`,
			response: aws.String("Example text right here."),
			error:    nil,
		},
	}

	client := &Client{}

	urlPath := "/path.html"

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc(urlPath, func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, test.body)
			})

			server := httptest.NewServer(mux)

			response, err := client.GetText(context.Background(), ItemXML{
				Link: server.URL + urlPath,
			})
			if err != test.error {
				t.Errorf("incorrect error, received: %v, expected: %v", err, test.error)
			}

			if !reflect.DeepEqual(response, test.response) {
				t.Errorf("incorrect response, received: %+v, expected: %+v", response, test.response)
			}
		})
	}
}
