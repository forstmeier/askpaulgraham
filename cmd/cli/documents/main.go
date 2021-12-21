//+build !test

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/forstmeier/askpaulgraham/pkg/cnt"
	"github.com/forstmeier/askpaulgraham/pkg/db"
	"github.com/forstmeier/askpaulgraham/pkg/nlp"
	"github.com/forstmeier/askpaulgraham/util"
)

const (
	documentFilename  = "document.json"
	documentsFilename = "documents.jsonl"
)

const (
	getAction  = "get"
	setAction  = "set"
	singleSize = "single"
	bulkSize   = "bulk"
)

func main() {
	ctx := context.Background()

	action := flag.String("action", "get", `action to perform ("get" or "set")`)
	size := flag.String("size", "single", `size of the action ("single" or "bulk")`)
	postID := flag.String("id", "", "blog post id")

	flag.Parse()

	if *action != getAction && *action != setAction {
		log.Fatalf("invalid action: %s", *action)
	}

	if *size != singleSize && *size != bulkSize {
		log.Fatalf("invalid size: %s", *size)
	}

	if *size == singleSize && *action == getAction && *postID == "" {
		log.Fatal("error invalid arguments: argument 'id' is required for 'single' 'get' operation")
	}

	config := util.Config{}
	configContent, err := ioutil.ReadFile("etc/config/config.json")
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}
	if err := json.Unmarshal(configContent, &config); err != nil {
		log.Fatalf("error unmarshalling config file: %v", err)
	}

	newSession, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})
	if err != nil {
		log.Fatalf("error creating aws session: %v", err)
	}

	cntClient := cnt.New()
	dbClient := db.New(
		newSession,
		config.AWS.S3.DataBucketName,
		config.AWS.DynamoDB.QuestionsTableName,
		config.AWS.DynamoDB.SummariesTableName,
	)
	nlpClient := nlp.New(
		newSession,
		config.OpenAI.APIKey,
		config.AWS.S3.DataBucketName,
	)

	if *action == getAction {
		if *size == singleSize {
			postURL := fmt.Sprintf("http://www.paulgraham.com/%s.html", *postID)
			text, err := cntClient.GetText(ctx, postURL)
			if err != nil {
				log.Fatalf("error getting text: %v", err)
			}

			bodyBytes, err := json.Marshal(db.Document{
				Metadata: *postID,
				Text:     *text,
			})
			if err != nil {
				log.Fatalf("error marshalling document: %v", err)
			}

			if err := os.WriteFile(documentFilename, bodyBytes, 0644); err != nil {
				log.Fatalf("error writing document file: %v", err)
			}

		} else if *size == bulkSize {
			items, err := cntClient.GetItems(ctx, "http://www.aaronsw.com/2002/feeds/pgessays.rss")
			if err != nil {
				log.Fatalf("error getting items: %v", err)
			}

			documentsBody := bytes.Buffer{}
			encoder := json.NewEncoder(&documentsBody)
			for _, item := range items {
				if strings.Contains(item.Link, "1638975042") {
					continue
				}

				text, err := cntClient.GetText(ctx, item.Link)
				if err != nil {
					log.Fatalf("error getting text: %v", err)
				}

				id := util.GetIDFromURL(item.Link)
				if err := encoder.Encode(db.Document{
					Text:     *text,
					Metadata: id,
				}); err != nil {
					log.Fatalf("error encoding document: %v", err)
				}
			}

			if err := os.WriteFile(documentsFilename, documentsBody.Bytes(), 0644); err != nil {
				log.Fatalf("error writing documents file: %v", err)
			}
		}

	} else if *action == setAction {
		filename := ""
		if *size == singleSize {
			filename = documentFilename
		} else if *size == bulkSize {
			filename = documentsFilename
		}

		documents := []db.Document{}

		bodyBytes, err := os.ReadFile(filename)
		if err != nil {
			log.Fatalf("error reading file: %v", err)
		}

		if *size == singleSize {
			storedDocuments, err := dbClient.GetDocuments(ctx)
			if err != nil {
				log.Fatalf("error getting stored documents file: %v", err)
			}

			document := db.Document{}
			if err := json.Unmarshal(bodyBytes, &document); err != nil {
				log.Fatalf("error unmarshalling local document file: %v", err)
			}

			documents = append(documents, document)
			for _, storedDocument := range storedDocuments {
				if storedDocument.Metadata != document.Metadata {
					documents = append(documents, storedDocument)
				}
			}

			if err := dbClient.StoreText(ctx, document.Metadata, document.Text); err != nil {
				log.Fatalf("error storing markdown text file: %v", err)
			}

			if err := nlpClient.SetAnswer(ctx, document.Metadata, document.Text); err != nil {
				log.Fatalf("error setting answer: %v", err)
			}

		} else if *size == bulkSize {
			decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
			for decoder.More() {
				document := db.Document{}
				if err := decoder.Decode(&document); err == io.EOF {
					break
				} else if err != nil {
					log.Fatalf("error decoding document: %v", err)
				}

				documents = append(documents, document)
			}

			items, err := cntClient.GetItems(ctx, "http://www.aaronsw.com/2002/feeds/pgessays.rss")
			if err != nil {
				log.Fatalf("error getting items: %v", err)
			}

			itemIDsMap := map[string]struct{}{}
			for _, item := range items {
				if strings.Contains(item.Link, "1638975042") {
					continue
				}

				id := util.GetIDFromURL(item.Link)
				itemIDsMap[id] = struct{}{}
			}

			for _, document := range documents {
				if _, ok := itemIDsMap[document.Metadata]; !ok {
					if err := nlpClient.SetAnswer(ctx, document.Metadata, document.Text); err != nil {
						log.Fatalf("error setting answer: %v", err)
					}
				}
			}
		}

		if err := dbClient.StoreDocuments(ctx, documents); err != nil {
			log.Fatalf("error storing documents file: %v", err)
		}
	}
}
