package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"mailassist/internal/application/email"
	"mailassist/internal/infrastructure/config"
	"mailassist/internal/infrastructure/gmail"
	"mailassist/internal/infrastructure/llm"
	"mailassist/internal/infrastructure/persistence/sqlite"
	"mailassist/internal/infrastructure/pubsub"
	pubsubHandler "mailassist/internal/interfaces/pubsub"
	"mailassist/internal/interfaces/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	repo, err := sqlite.NewEmailRepository(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to create repository: %v", err)
	}
	defer func() {
		if err := repo.Close(); err != nil {
			log.Printf("Failed to close repository: %v", err)
		}
	}()

	llmClient, err := llm.NewClient()
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	gmailService, err := gmail.NewService(ctx)
	if err != nil {
		log.Fatalf("Failed to create Gmail service: %v", err)
	}

	gmailClient := gmail.NewClient(gmailService)

	if err := gmailClient.InitLabels(); err != nil {
		log.Fatalf("Failed to initialize labels: %v", err)
	}

	// Enable Gmail watch for push notifications
	if err := gmailClient.EnableWatch(ctx, cfg.TopicName); err != nil {
		log.Printf("Warning: Failed to enable watch: %v", err)
	}

	classifyUC := email.NewClassifyEmailUseCase(repo, llmClient, gmailClient)

	pool := worker.NewPool(cfg.NumWorkers, classifyUC)
	pool.Start(ctx)
	defer pool.Shutdown()

	// Pub/Sub subscriber
	subscriber, err := pubsub.NewSubscriber(ctx, cfg.GoogleCloudProject, cfg.SubscriptionID)
	if err != nil {
		log.Fatalf("Failed to create subscriber: %v", err)
	}
	defer func() {
		if err := subscriber.Close(); err != nil {
			log.Printf("Failed to close subscriber: %v", err)
		}
	}()

	// Pub/Sub handler
	handler := pubsubHandler.NewHandler(pool, gmailClient)

	// Process initial emails
	log.Printf("Processing initial batch of %d emails...", cfg.InitialEmailsToFetch)
	if err := processInitialEmails(ctx, gmailClient, pool, cfg.InitialEmailsToFetch); err != nil {
		log.Printf("Warning: Failed to process initial emails: %v", err)
	}

	// Start Pub/Sub listener in background
	go func() {
		log.Println("Starting Pub/Sub listener...")
		if err := subscriber.Listen(ctx, func(historyID uint64) {
			handler.HandleNotification(ctx, historyID)
		}); err != nil && ctx.Err() == nil {
			log.Printf("Pub/Sub listener error: %v", err)
		}
	}()

	log.Println("MailAssist is running. Press Ctrl+C to stop.")

	<-ctx.Done()
	log.Println("Shutting down gracefully...")
}

func processInitialEmails(ctx context.Context, gmailClient *gmail.Client, pool *worker.Pool, maxResults int64) error {
	messageIDs, err := gmailClient.ListMessagesFromInbox(ctx, maxResults)
	if err != nil {
		return err
	}

	log.Printf("Found %d messages to process", len(messageIDs))

	for _, msgID := range messageIDs {
		pool.Submit(worker.EmailJob{GmailID: msgID})
	}

	return nil
}
