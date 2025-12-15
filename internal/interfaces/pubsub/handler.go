package pubsub

import (
	"context"
	"log"

	"mailassist/internal/interfaces/worker"
)

type Handler struct {
	pool         *worker.Pool
	gmailFetcher HistoryFetcher
}

type HistoryFetcher interface {
	FetchNewMessagesSince(ctx context.Context, historyID uint64) ([]string, error)
}

func NewHandler(pool *worker.Pool, gmailFetcher HistoryFetcher) *Handler {
	return &Handler{
		pool:         pool,
		gmailFetcher: gmailFetcher,
	}
}

func (h *Handler) HandleNotification(ctx context.Context, historyID uint64) {
	messageIDs, err := h.gmailFetcher.FetchNewMessagesSince(ctx, historyID)
	if err != nil {
		log.Printf("Fetch history error: %v", err)
		return
	}

	if len(messageIDs) == 0 {
		log.Printf("No new messages in historyID: %d", historyID)
		return
	}

	log.Printf("Found %d new message(s) in historyID: %d", len(messageIDs), historyID)

	for _, msgID := range messageIDs {
		h.pool.Submit(worker.EmailJob{GmailID: msgID})
	}
}
