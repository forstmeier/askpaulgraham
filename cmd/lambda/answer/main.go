package main

import (
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/forstmeier/askpaulgraham/pkg/db"
	"github.com/forstmeier/askpaulgraham/pkg/nlp"
)

func main() {
	newSession := session.New()

	dbClient := db.New(
		newSession,
		os.Getenv("BUCKET_NAME"),
		os.Getenv("QUESTIONS_TABLE_NAME"),
		os.Getenv("SUMMARIES_TABLE_NAME"),
	)

	nlpClient := nlp.New(
		newSession,
		os.Getenv("API_KEY"),
		os.Getenv("BUCKET_NAME"),
	)

	lambda.Start(handler(dbClient, nlpClient))
}
