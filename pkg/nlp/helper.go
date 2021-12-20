package nlp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type helper interface {
	sendRequest(method, url string, body io.Reader, payload interface{}, headers map[string]string) error
}

var _ helper = &help{}

type help struct {
	apiKey     string
	httpClient http.Client
}

func (h *help) sendRequest(method, url string, body io.Reader, payload interface{}, headers map[string]string) error {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.apiKey))
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("nlp: %s", body)
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&payload); err != nil {
		return err
	}

	return nil
}
