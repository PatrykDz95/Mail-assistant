package gmailc

import (
	"context"
	"fmt"
	"google.golang.org/api/gmail/v1"
	"time"
)

// TODO: find a better way to handle Gmail history delays
func (c *Client) FetchNewMessagesSince(ctx context.Context, historyID uint64) ([]*gmail.History, error) {

	var historyItems []*gmail.History

	// Gmail writes history asynchronously, so allow retries
	for attempt := 1; attempt <= 5; attempt++ {

		resp, err := c.Srv.Users.History.List("me").
			StartHistoryId(historyID).
			HistoryTypes("messageAdded").
			Context(ctx).
			Do()

		if err != nil {
			return nil, fmt.Errorf("gmail history list error: %w", err)
		}

		// If Gmail history is available -> return it
		if len(resp.History) > 0 {
			return resp.History, nil
		}

		// No history yet -> Gmail is lat, so retry with backoff
		time.Sleep(time.Duration(attempt*80) * time.Millisecond)
	}

	// Still empty after retries, not an error
	// Gmail often delays write or sends events early.
	return historyItems, nil
}

func (c *Client) ExtractMessageIDs(histories []*gmail.History) []string {
	var ids []string

	for _, h := range histories {
		for _, added := range h.MessagesAdded {
			if added.Message != nil && !isDraft(added.Message) {
				ids = append(ids, added.Message.Id)
			}
		}
	}

	return ids
}

func isDraft(msg *gmail.Message) bool {
	for _, labelID := range msg.LabelIds {
		if labelID == "DRAFT" {
			return true
		}
	}
	return false
}
