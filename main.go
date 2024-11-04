package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gorilla/mux"
	"github.com/jamespearly/loggly"
	"github.com/joho/godotenv"
)

type Comment struct {
	ID     string `json:"id"`
	Author string `json:"author"`
	Body   string `json:"body"`
	Score  int64  `json:"score"`
}

var client *loggly.ClientType
var db *dynamodb.DynamoDB

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	logglyToken := os.Getenv("LOGGLY_TOKEN")
	client = loggly.New(logglyToken)

	session, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		log.Fatal("Couldn't create an AWS session", err)
	}

	db = dynamodb.New(session)

	router := mux.NewRouter()

	router.HandleFunc("/kparajul/status", statusHandler).Methods(http.MethodGet)
	router.HandleFunc("/kparajul/all", allHandler).Methods(http.MethodGet)
	router.HandleFunc("/kparajul/search", searchHandler).Methods(http.MethodGet)

	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	port := ":8080"

	http.ListenAndServe(port, router)

}

func notFoundHandler(writer http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(writer, fmt.Sprintf("Method not allowed: %d", http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		logFunc(req, http.StatusMethodNotAllowed)
	} else if req.URL.Path != "/kparajul/status" && req.URL.Path != "/kparajul/all" && req.URL.Path != "/kparajul/search" {
		http.Error(writer, fmt.Sprintf("Not Found: %d", http.StatusNotFound), http.StatusNotFound)
		logFunc(req, http.StatusNotFound)
	} else {
		http.Error(writer, fmt.Sprintf("Not Found: %d", http.StatusNotFound), http.StatusNotFound)
		logFunc(req, http.StatusNotFound)
	}
}

func statusHandler(writer http.ResponseWriter, req *http.Request) {
	result, er := db.DescribeTable(&dynamodb.DescribeTableInput{TableName: aws.String("kparajul_Reddit_Comments")})
	if er != nil {
		client.EchoSend("error scanning database", er.Error())
		logFunc(req, http.StatusInternalServerError)
	}
	itemCount := int(*result.Table.ItemCount)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	err := json.NewEncoder(writer).Encode(itemCount)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		logFunc(req, http.StatusInternalServerError)
		return
	}
	logFunc(req, http.StatusOK)

}

func allHandler(writer http.ResponseWriter, req *http.Request) {
	result, er := db.Scan(&dynamodb.ScanInput{TableName: aws.String("kparajul_Reddit_Comments"), ProjectionExpression: aws.String("id, author, body, score")})
	if er != nil {
		client.EchoSend("error scanning database", er.Error())
		logFunc(req, http.StatusInternalServerError)
	}

	//this syntax is difficult, I took help from chatGPT to figure out syntax to parse content
	var comments []Comment
	for _, item := range result.Items {
		var comment Comment

		if id, ok := item["id"]; ok && id.S != nil {
			comment.ID = *id.S
		}
		if author, ok := item["author"]; ok && author.S != nil {
			comment.Author = *author.S
		}
		if body, ok := item["body"]; ok && body.S != nil {
			comment.Body = *body.S
		}
		if score, ok := item["score"]; ok && score.N != nil {
			comment.Score, _ = strconv.ParseInt(*score.N, 10, 64)
		}
		comments = append(comments, comment)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	err := json.NewEncoder(writer).Encode(comments)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		logFunc(req, http.StatusInternalServerError)
		return
	}
	logFunc(req, http.StatusOK)

}

func searchHandler(writer http.ResponseWriter, req *http.Request) {
	parameters := req.URL.Query()
	id := parameters.Get("id")
	score := parameters.Get("score")

	var filterExpression string
	expressionAttribute := make(map[string]*dynamodb.AttributeValue)

	if id != "" && score == "" {
		filterExpression = "id = :id"
		expressionAttribute[":id"] = &dynamodb.AttributeValue{S: aws.String(id)}
	} else if score != "" && id == "" {
		filterExpression = "score = :score"
		expressionAttribute[":score"] = &dynamodb.AttributeValue{N: aws.String(score)}
	} else if id != "" && score != "" {
		filterExpression = "id = :id AND score = :score"
		expressionAttribute[":id"] = &dynamodb.AttributeValue{S: aws.String(id)}
		expressionAttribute[":score"] = &dynamodb.AttributeValue{N: aws.String(score)}
	} else {
		http.Error(writer, "Please provide id or score", http.StatusBadRequest)
		return
	}

	result, er := db.Scan(&dynamodb.ScanInput{
		TableName:                 aws.String("kparajul_Reddit_Comments"),
		FilterExpression:          aws.String(filterExpression),
		ExpressionAttributeValues: expressionAttribute,
		ProjectionExpression:      aws.String("id, author, body, score")})
	if er != nil {
		client.EchoSend("error scanning database", er.Error())
		logFunc(req, http.StatusInternalServerError)
	}
	var comments []Comment
	for _, item := range result.Items {
		var comment Comment

		if idx, ok := item["id"]; ok && idx.S != nil {
			comment.ID = *idx.S
		}
		if author, ok := item["author"]; ok && author.S != nil {
			comment.Author = *author.S
		}
		if body, ok := item["body"]; ok && body.S != nil {
			comment.Body = *body.S
		}
		if scorex, ok := item["score"]; ok && scorex.N != nil {
			comment.Score, _ = strconv.ParseInt(*scorex.N, 10, 64)
		}
		comments = append(comments, comment)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	err := json.NewEncoder(writer).Encode(comments)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		logFunc(req, http.StatusInternalServerError)
		return
	}
	logFunc(req, http.StatusOK)

}

func logFunc(req *http.Request, statusCode int) {
	if req.URL.Path == "/favicon.ico" {
		return
	}
	response := fmt.Sprintf("Method type: %s, sourse IP address: %s, request path: %s, HTTP status code: %d", req.Method, req.RemoteAddr, req.URL.Path, statusCode)
	client.EchoSend("info", response)
}
