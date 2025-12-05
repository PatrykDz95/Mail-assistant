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
	NumberOfEmailsToDownload int64 = 52
	NumberOfWorkers          int   = 5
)

func main() {
	ctx := context.Background()

	application, err := app.NewApp(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	if err := application.ProcessInitialEmails(NumberOfEmailsToDownload); err != nil {
		log.Fatalf("Failed to process initial emails: %v", err)
	}

	application.StartWorkerPool(NumberOfWorkers)

	if err := application.StartPubSubListener(); err != nil {
		log.Fatalf("Pub/Sub listener error: %v", err)
	}
}
