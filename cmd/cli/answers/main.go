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

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/forstmeier/askpaulgraham/pkg/cnt"
	"github.com/forstmeier/askpaulgraham/pkg/db"
	"github.com/forstmeier/askpaulgraham/util"
)

const (
	answerFilename  = "answer.json"
	answersFilename = "answers.jsonl"
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

	if *size == singleSize && *postID == "" {
		log.Fatal("error invalid arguments: argument 'id' is required for 'single' operation")
	}

	config := util.Config{}
	configContent, err := ioutil.ReadFile("etc/config/config.json")
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}
	if err := toml.Unmarshal(configContent, &config); err != nil {
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

	if *action == getAction {
		if *size == singleSize {
			postURL := fmt.Sprintf("http://www.paulgraham.com/%s.html", *postID)
			text, err := cntClient.GetText(ctx, postURL)
			if err != nil {
				log.Fatalf("error getting text: %v", err)
			}

			bodyBytes, err := json.Marshal(db.Answer{
				Metadata: *postID,
				Text:     *text,
			})
			if err != nil {
				log.Fatalf("error marshalling answer: %v", err)
			}

			if err := os.WriteFile(answerFilename, bodyBytes, 0644); err != nil {
				log.Fatalf("error writing answer file: %v", err)
			}

		} else if *size == bulkSize {
			items, err := cntClient.GetItems(ctx, "http://www.aaronsw.com/2002/feeds/pgessays.rss")
			if err != nil {
				log.Fatalf("error getting items: %v", err)
			}

			answersBody := bytes.Buffer{}
			encoder := json.NewEncoder(&answersBody)
			for _, item := range items {
				if strings.Contains(item.Link, "1638975042") {
					continue
				}

				text, err := cntClient.GetText(ctx, item.Link)
				if err != nil {
					log.Fatalf("error getting text: %v", err)
				}

				id := util.GetIDFromURL(item.Link)
				if err := encoder.Encode(db.Answer{
					Text:     *text,
					Metadata: id,
				}); err != nil {
					log.Fatalf("error encoding answer: %v", err)
				}
			}

			if err := os.WriteFile(answersFilename, answersBody.Bytes(), 0644); err != nil {
				log.Fatalf("error writing answers file: %v", err)
			}
		}

	} else if *action == setAction {
		filename := ""
		if *size == singleSize {
			filename = answerFilename
		} else if *size == bulkSize {
			filename = answersFilename
		}

		answers := []db.Answer{}

		bodyBytes, err := os.ReadFile(filename)
		if err != nil {
			log.Fatalf("error reading file: %v", err)
		}

		if *size == singleSize {
			storedAnswers, err := dbClient.GetAnswers(ctx)
			if err != nil {
				log.Fatalf("error getting stored answers file: %v", err)
			}

			answer := db.Answer{}
			if err := json.Unmarshal(bodyBytes, &answer); err != nil {
				log.Fatalf("error unmarshalling local answer file: %v", err)
			}

			for i, storedAnswer := range storedAnswers {
				if storedAnswer.Metadata == answer.Metadata {
					break
				} else if len(storedAnswers) == i+1 {
					answers = append(answers, storedAnswer)
				}
			}

		} else if *size == bulkSize {
			decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
			for decoder.More() {
				answer := db.Answer{}
				if err := decoder.Decode(&answer); err == io.EOF {
					break
				} else if err != nil {
					log.Fatalf("error decoding answer: %v", err)
				}

				answers = append(answers, answer)
			}
		}

		if err := dbClient.StoreAnswers(ctx, answers); err != nil {
			log.Fatalf("error storing answers file: %v", err)
		}
	}
}
