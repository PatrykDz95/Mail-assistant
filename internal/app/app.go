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
	labelID     string
}

func NewApp(ctx context.Context) (*App, error) {

	gSrv, err := gmailc.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("gmail service error: %w", err)
	}
	gmailClient := &gmailc.Client{Srv: gSrv}

	st, err := store.NewStore("mailai.db")
	if err != nil {
		return nil, fmt.Errorf("sqlite error: %w", err)
	}

	llmClient, err := llm.NewClient()
	if err != nil {
		return nil, fmt.Errorf("llm client error: %w", err)
	}

	labelID, err := gmailClient.EnsureLabel("Processed by MailAI")
	if err != nil {
		return nil, fmt.Errorf("cannot ensure label: %w", err)
	}

	return &App{
		ctx:         ctx,
		gmailClient: gmailClient,
		llmClient:   llmClient,
		store:       st,
		labelID:     labelID,
	}, nil
}

func (a *App) ProcessInitialEmails(maxResults int64) error {
	log.Printf("Processing initial batch of %d emails...", maxResults)

	listRes, err := a.gmailClient.Srv.Users.Messages.List("me").
		LabelIds("INBOX").
		MaxResults(maxResults).
		Do()
	if err != nil {
		return fmt.Errorf("cannot list messages: %w", err)
	}

	if len(listRes.Messages) == 0 {
		log.Println("No new messages found.")
		return nil
	}

	for _, m := range listRes.Messages {
		if err := a.processEmail(m.Id); err != nil {
			log.Printf("Error processing %s: %v", m.Id, err)
			continue
		}
	}

	log.Println("Initial email processing completed")
	return nil
}

// processEmail processes a single email
func (a *App) processEmail(messageID string) error {
	// Check if already processed
	already, err := a.store.AlreadyProcessed(messageID)
	if err != nil {
		return fmt.Errorf("db error: %w", err)
	}
	if already {
		log.Printf("Message %s already processed, skip", messageID)
		return nil
	}

	// Fetch email
	email, err := a.gmailClient.FetchByID(messageID)
	if err != nil {
		return fmt.Errorf("fetch error: %w", err)
	}

	// Skip empty emails
	if email.Body == "" {
		log.Printf("Empty body for %s, skipping", email.ID)
		return nil
	}

	// Analyze with LLM
	res, err := a.llmClient.AnalyzeEmail(a.ctx, email.Subject, email.Body)
	if err != nil {
		return fmt.Errorf("llm error: %w", err)
	}

	// Add label
	if err := a.gmailClient.AddLabelToMessage(email.ID, res.Label); err != nil {
		log.Printf("AddLabel error: %v", err)
	}
	log.Printf("Added label %s to email %s", res.Label, email.From)

	// Create draft reply for action_needed emails
	if res.Category == "action_needed" {
		if err := a.gmailClient.CreateReplyDraft(email, res.Reply); err != nil {
			log.Printf("Draft error: %v", err)
		} else {
			log.Println("Draft created")
		}
	}

	if err := a.store.SaveEmail(&store.EmailRecord{
		GmailID:  email.ID,
		From:     email.From,
		Subject:  email.Subject,
		Body:     email.Body,
		Category: res.Category,
		Label:    res.Label,
		Draft:    res.Reply,
	}); err != nil {
		return fmt.Errorf("db save error: %w", err)
	}

	log.Printf("OK: %s – category=%s, label=%s", email.ID, res.Category, res.Label)
	return nil
}

func (a *App) StartWorkerPool(numWorkers int) {
	a.workerPool = worker.NewPool(numWorkers, a.llmClient, a.gmailClient, a.store, a.labelID)
	a.workerPool.Start()
	log.Printf("Worker pool started with %d workers", numWorkers)
}

func (a *App) StartPubSubListener() error {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return fmt.Errorf("GOOGLE_CLOUD_PROJECT is not set")
	}

	subscriptionID := os.Getenv("SUBSCRIPTION_ID")
	if subscriptionID == "" {
		return fmt.Errorf("SUBSCRIPTION_ID is not set")
	}

	topicName := fmt.Sprintf("projects/%s/topics/gmail-topic", projectID)
	if err := a.gmailClient.EnableWatch(a.ctx, topicName); err != nil {
		return fmt.Errorf("enable watch error: %w", err)
	}

	log.Println("Starting Pub/Sub listener...")
	return pubsub.StartListener(a.ctx, projectID, subscriptionID, a.handlePubSubNotification)
}

// handlePubSubNotification handles incoming Pub/Sub notifications
func (a *App) handlePubSubNotification(historyID uint64) {
	hist, err := a.gmailClient.FetchNewMessagesSince(a.ctx, historyID)
	if err != nil {
		log.Printf("History fetch error: %v", err)
		return
	}

	ids := a.gmailClient.ExtractMessageIDs(hist)

	if len(ids) == 0 {
		log.Printf("No new messages in this history")
		return
	}

	log.Printf("Found %d message(s) in history", len(ids))

	for _, id := range ids {
		// Check if already processed
		already, err := a.store.AlreadyProcessed(id)
		if err != nil {
			log.Printf("DB check error for %s: %v", id, err)
			continue
		}
		if already {
			log.Printf("Message %s already processed, skipping", id)
			continue
		}

		// Fetch message
		msg, err := a.gmailClient.FetchByID(id)
		if err != nil {
			log.Printf("Fetch error for %s: %v", id, err)
			continue
		}

		log.Printf("New email FROM: %s | SUBJECT: %s", msg.From, msg.Subject)

		// Submit to worker pool
		a.workerPool.Submit(worker.Job{
			ID:      msg.ID,
			Subject: msg.Subject,
			Body:    msg.Body,
		})
	}
}
