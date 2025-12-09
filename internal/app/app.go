package app

import (
	"context"
	"fmt"
	"log"
	"os"

	"mailassist/internal/gmailc"
	"mailassist/internal/llm"
	"mailassist/internal/pubsub"
	"mailassist/internal/store"
	"mailassist/internal/worker"
)

type App struct {
	ctx         context.Context
	gmailClient *gmailc.Client
	llmClient   *llm.Client
	store       *store.Store
	workerPool  *worker.Pool
}

func NewApp(ctx context.Context) (*App, error) {
	// Gmail service
	gSrv, err := gmailc.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("gmail service error: %w", err)
	}
	gmailClient := &gmailc.Client{Srv: gSrv}

	// SQLite
	st, err := store.NewStore("mailai.db")
	if err != nil {
		return nil, fmt.Errorf("sqlite error: %w", err)
	}

	// LLM client
	llmClient, err := llm.NewClient()
	if err != nil {
		return nil, fmt.Errorf("llm client error: %w", err)
	}

	return &App{
		ctx:         ctx,
		gmailClient: gmailClient,
		llmClient:   llmClient,
		store:       st,
	}, nil
}

func (a *App) StartWorkerPool(numWorkers int) {
	a.workerPool = worker.NewPool(numWorkers, a.llmClient, a.gmailClient, a.store)
	a.workerPool.Start()
	log.Printf("Worker pool started with %d workers", numWorkers)
}

func (a *App) ProcessInitialEmails(maxResults int64) error {
	if a.workerPool == nil {
		return fmt.Errorf("worker pool not started – call StartWorkerPool first")
	}

	log.Printf("Processing initial batch of %d emails...", maxResults)

	listRes, err := a.gmailClient.Srv.Users.Messages.List("me").
		LabelIds("INBOX").
		MaxResults(maxResults).
		Do()

	if err != nil {
		return fmt.Errorf("cannot list messages: %w", err)
	}

	if len(listRes.Messages) == 0 {
		log.Println("No messages found.")
		return nil
	}

	for _, m := range listRes.Messages {
		id := m.Id

		already, err := a.store.AlreadyProcessed(id)
		if err != nil {
			log.Printf("DB check error: %v", err)
			continue
		}
		if already {
			continue
		}

		email, err := a.gmailClient.FetchByID(id)
		if err != nil {
			log.Printf("Fetch error: %v", err)
			continue
		}

		if email.Body == "" {
			continue
		}

		a.workerPool.Submit(worker.Job{
			ID:      email.ID,
			Subject: email.Subject,
			Body:    email.Body,
		})
	}

	log.Println("Initial emails submitted to worker pool")
	return nil
}

func (a *App) StartPubSubListener() error {
	if a.workerPool == nil {
		return fmt.Errorf("worker pool not started – call StartWorkerPool first")
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return fmt.Errorf("GOOGLE_CLOUD_PROJECT not set")
	}

	subID := os.Getenv("SUBSCRIPTION_ID")
	if subID == "" {
		return fmt.Errorf("SUBSCRIPTION_ID not set")
	}

	topicName := fmt.Sprintf("projects/%s/topics/gmail-topic", projectID)
	if err := a.gmailClient.EnableWatch(a.ctx, topicName); err != nil {
		return fmt.Errorf("enable watch error: %w", err)
	}

	log.Println("Starting Pub/Sub listener...")

	return pubsub.StartListener(a.ctx, projectID, subID, a.handlePubSubNotification)
}

func (a *App) handlePubSubNotification(historyID uint64) {
	hist, err := a.gmailClient.FetchNewMessagesSince(a.ctx, historyID)
	if err != nil {
		log.Printf("History fetch error: %v", err)
		return
	}
	if hist == nil {
		return
	}

	ids := a.gmailClient.ExtractMessageIDs(hist)
	if len(ids) == 0 {
		return
	}

	for _, id := range ids {

		already, err := a.store.AlreadyProcessed(id)
		if err != nil {
			log.Printf("DB check error: %v", err)
			continue
		}
		if already {
			continue
		}

		email, err := a.gmailClient.FetchByID(id)
		if err != nil {
			log.Printf("Fetch error: %v", err)
			continue
		}

		if email.Body == "" {
			continue
		}

		a.workerPool.Submit(worker.Job{
			ID:      email.ID,
			Subject: email.Subject,
			Body:    email.Body,
		})
	}
}
