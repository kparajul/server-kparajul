package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/jamespearly/loggly"
	"github.com/joho/godotenv"
)

type Response struct {
	Time    string `json:"time"`
	StatusC int    `json:"status_code"`
}

var client *loggly.ClientType

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	logglyToken := os.Getenv("LOGGLY_TOKEN")
	client = loggly.New(logglyToken)

	router := mux.NewRouter()

	router.HandleFunc("/kparajul/status", statusHandler).Methods(http.MethodGet)

	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	port := ":8080"

	http.ListenAndServe(port, router)

}

func notFoundHandler(writer http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(writer, fmt.Sprintf("Method not allowed: %d", http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		logFunc(req, http.StatusMethodNotAllowed)
	} else if req.URL.Path != "/kparajul/status" {
		http.Error(writer, fmt.Sprintf("Not Found: %d", http.StatusNotFound), http.StatusNotFound)
		logFunc(req, http.StatusNotFound)
	} else {
		http.Error(writer, fmt.Sprintf("Not Found: %d", http.StatusNotFound), http.StatusNotFound)
		logFunc(req, http.StatusNotFound)
	}

}

func statusHandler(writer http.ResponseWriter, req *http.Request) {
	location, er := time.LoadLocation("UTC")
	if er != nil {
		client.EchoSend("error", er.Error())
	}
	response := Response{Time: time.Now().In(location).Format(time.RFC1123Z), StatusC: http.StatusOK}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	err := json.NewEncoder(writer).Encode(response)
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
