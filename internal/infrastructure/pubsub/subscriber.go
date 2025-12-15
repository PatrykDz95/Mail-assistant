package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cloud.google.com/go/pubsub"
)

// Notification represents Gmail Pub/Sub notification
type Notification struct {
	EmailAddress string `json:"emailAddress"`
	HistoryID    uint64 `json:"historyId"`
}

// Subscriber handles Pub/Sub messages
type Subscriber struct {
	client         *pubsub.Client
	subscriptionID string
	processedIDs   map[uint64]bool
}

// NewSubscriber creates a new Pub/Sub subscriber
func NewSubscriber(ctx context.Context, projectID, subscriptionID string) (*Subscriber, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create pubsub client: %w", err)
	}

	return &Subscriber{
		client:         client,
		subscriptionID: subscriptionID,
		processedIDs:   make(map[uint64]bool),
	}, nil
}

// Listen starts listening for Pub/Sub messages
func (s *Subscriber) Listen(ctx context.Context, handler func(historyID uint64)) error {
	sub := s.client.Subscription(s.subscriptionID)

	log.Println("Pub/Sub listener started...")

	return sub.Receive(ctx, func(_ context.Context, m *pubsub.Message) {
		notification, err := parseNotification(m.Data)
		if err != nil {
			log.Printf("Parse notification error: %v", err)
			m.Ack()
			return
		}

		// Avoid duplicate processing
		if s.processedIDs[notification.HistoryID] {
			m.Ack()
			return
		}

		log.Printf("New notification - %s (historyID: %d)", notification.EmailAddress, notification.HistoryID)

		s.processedIDs[notification.HistoryID] = true
		handler(notification.HistoryID)

		m.Ack()
	})
}

// Close closes the Pub/Sub client
func (s *Subscriber) Close() error {
	return s.client.Close()
}

func parseNotification(data []byte) (*Notification, error) {
	var n Notification
	if err := json.Unmarshal(data, &n); err != nil {
		return nil, fmt.Errorf("unmarshal notification: %w", err)
	}
	return &n, nil
}
