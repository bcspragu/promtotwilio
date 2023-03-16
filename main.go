package main

import (
	"log"
	"net/http"
	"os"
)

type options struct {
	AccountSid string
	AuthToken  string
	Receiver   string
	Sender     string
}

func main() {
	opts := options{
		AccountSid: os.Getenv("SID"),
		AuthToken:  os.Getenv("TOKEN"),
		Receiver:   os.Getenv("RECEIVER"),
		Sender:     os.Getenv("SENDER"),
	}

	if opts.AccountSid == "" || opts.AuthToken == "" || opts.Sender == "" {
		log.Fatal("'SID', 'TOKEN' and 'SENDER' environment variables need to be set")
	}

	err := http.ListenAndServe(":8080", NewServer(&opts))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
