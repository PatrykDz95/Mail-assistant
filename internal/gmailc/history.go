package gmailc

import (
	"context"
	"fmt"

	"google.golang.org/api/gmail/v1"
)

func (c *Client) FetchNewMessagesSince(ctx context.Context, historyID uint64) ([]*gmail.History, error) {

	resp, err := c.Srv.Users.History.List("me").
		StartHistoryId(historyID).
		HistoryTypes("messageAdded").
		Context(ctx).
		Do()

	if err != nil {
		return nil, fmt.Errorf("gmail history list error: %w", err)
	}

	return resp.History, nil
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

// isDraft checks if a message is a draft by looking at its labels
func isDraft(msg *gmail.Message) bool {
	for _, labelID := range msg.LabelIds {
		if labelID == "DRAFT" {
			return true
		}
	}
	return false
}
