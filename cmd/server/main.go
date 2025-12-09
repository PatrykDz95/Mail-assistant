package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"
	"mailassist/internal/app"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
}

const (
	NumberOfEmailsToDownload int64 = 50
	NumberOfWorkers          int   = 5
)

func main() {
	ctx := context.Background()

	newApp, err := app.NewApp(ctx)
	if err != nil {
		log.Fatal(err)
	}

	newApp.StartWorkerPool(NumberOfWorkers)

	if err := newApp.ProcessInitialEmails(NumberOfEmailsToDownload); err != nil {
		log.Fatalf("Failed to process initial emails: %v", err)
	}

	if err := newApp.StartPubSubListener(); err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
}
