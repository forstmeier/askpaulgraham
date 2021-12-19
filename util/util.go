package util

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"github.com/forstmeier/askpaulgraham/pkg/db"
)

// Config represents the config.json file.
type Config struct {
	AWS    AWS    `json:"aws"`
	OpenAI OpenAI `json:"openai"`
}

// AWS represents aws config.json file field.
type AWS struct {
	DynamoDB DynamoDB `json:"dynamodb"`
	S3       S3       `json:"s3"`
}

// DynamoDB represents dynamodb config.json file field.
type DynamoDB struct {
	QuestionsTableName string `json:"questions_table_name"`
	SummariesTableName string `json:"summaries_table_name"`
}

// S3 represents s3 config.json file field.
type S3 struct {
	DataBucketName string `json:"data_bucket_name"`
}

// OpenAI represents openai config.json file field.
type OpenAI struct {
	APIKey string `json:"api_key"`
}

// Log provides a basic wrapper to format log output.
func Log(key string, value interface{}) {
	logMessage(key, value)
}

// SendResponse is a helper function for sendin g responses
// to API Gateway.
func SendResponse(statusCode int, payload interface{}, message string) (events.APIGatewayProxyResponse, error) {
	var body interface{}

	switch payloadValue := payload.(type) {
	case error:
		body = struct {
			Error string `json:"error"`
		}{
			Error: payload.(error).Error(),
		}

	case []db.Summary:
		body = struct {
			Message   string       `json:"message"`
			Summaries []db.Summary `json:"summaries"`
		}{
			Message:   "success",
			Summaries: payloadValue,
		}

	case string:
		body = struct {
			Message string `json:"message"`
			Answer  string `json:"answer"`
		}{
			Message: "success",
			Answer:  payloadValue,
		}

	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		logMessage("MARSHAL_RESPONSE_PAYLOAD_ERROR", err.Error())
		return events.APIGatewayProxyResponse{
			StatusCode:      http.StatusInternalServerError,
			Body:            fmt.Sprintf(`{"error": %q}`, err),
			IsBase64Encoded: false,
		}, err
	}

	logMessage("RESPONSE_BODY", string(bodyBytes))
	return events.APIGatewayProxyResponse{
		StatusCode:      statusCode,
		Body:            string(bodyBytes),
		IsBase64Encoded: false,
	}, nil
}

func logMessage(key string, value interface{}) {
	log.Printf(`{"%s": "%+v"}`, key, value)
}

// GetIDFromURL returns the ID from the essay URL path.
func GetIDFromURL(url string) string {
	_, file := path.Split(url)
	id := strings.Replace(file, ".html", "", -1)
	return id
}
