package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cloud.google.com/go/pubsub"
)

type GmailNotification struct {
	EmailAddress string `json:"emailAddress"`
	HistoryID    uint64 `json:"historyId"`
}

func StartListener(ctx context.Context, projectID, subscriptionID string, handler func(historyID uint64)) error {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("pubsub client error: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing Pub/Sub client: %v", err)
		}
	}()

	sub := client.Subscription(subscriptionID)

	log.Println("Pub/Sub listener started...")

	// TODO: improve this logic
	// Track processed historyIDs to avoid duplicates
	// Gmail sends multiple notifications for the same historyID
	processedHistory := make(map[uint64]bool)

	return sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {

		notification, err := parseNotification(m.Data)
		if err != nil {
			log.Printf("Error parsing notification: %v, raw data: %s", err, string(m.Data))
			m.Ack()
			return
		}

		// Check if already processed
		if processedHistory[notification.HistoryID] {
			log.Printf("HistoryID %d already processed, skipping", notification.HistoryID)
			m.Ack()
			return
		}

		log.Printf("New notification - %s (historyID: %d)", notification.EmailAddress, notification.HistoryID)

		// Mark as processed
		processedHistory[notification.HistoryID] = true

		// Call your handler
		handler(notification.HistoryID)

		m.Ack()
	})
}

func parseNotification(data []byte) (*GmailNotification, error) {
	var notification GmailNotification
	err := json.Unmarshal(data, &notification)
	return &notification, err
}
