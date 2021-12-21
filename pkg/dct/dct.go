package dct

// Document represents a row in the documents.jsonl file.
type Document struct {
	Text     string `json:"text"`
	Metadata string `json:"metadata"`
}
