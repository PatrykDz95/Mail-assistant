package gmailc

import (
	"context"
	"fmt"
	"google.golang.org/api/gmail/v1"
)

func (c *Client) EnableWatch(ctx context.Context, topic string) error {

	req := &gmail.WatchRequest{
		TopicName: topic,
	}

	resp, err := c.Srv.Users.Watch("me", req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("gmail watch error: %w", err)
	}

	fmt.Println("Watch enabled. Expiration:", resp.Expiration)

	return nil
}
