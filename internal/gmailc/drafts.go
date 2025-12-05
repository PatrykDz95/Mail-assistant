package gmailc

import (
	"encoding/base64"
	"fmt"

	"google.golang.org/api/gmail/v1"
)

func (c *Client) CreateReplyDraft(email *Email, replyBody string) error {
	raw := fmt.Sprintf(
		"To: %s\r\nSubject: Re: %s\r\n\r\n%s",
		email.From, email.Subject, replyBody,
	)

	encoded := base64.URLEncoding.EncodeToString([]byte(raw))

	_, err := c.Srv.Users.Drafts.Create("me", &gmail.Draft{
		Message: &gmail.Message{
			Raw: encoded,
		},
	}).Do()

	return err
}
