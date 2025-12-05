package gmailc

import (
	"encoding/base64"
	"strings"

	"google.golang.org/api/gmail/v1"
)

type Email struct {
	ID      string
	From    string
	Subject string
	Body    string
}

func (c *Client) FetchByID(id string) (*Email, error) {
	msg, err := c.Srv.Users.Messages.Get("me", id).Format("FULL").Do()
	if err != nil {
		return nil, err
	}

	return &Email{
		ID:      id,
		From:    header(msg, "From"),
		Subject: header(msg, "Subject"),
		Body:    extractBody(msg),
	}, nil
}

func header(msg *gmail.Message, name string) string {
	for _, h := range msg.Payload.Headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

func extractBody(msg *gmail.Message) string {
	if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
		d, _ := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
		return string(d)
	}

	for _, p := range msg.Payload.Parts {
		if p.MimeType == "text/plain" && p.Body != nil && p.Body.Data != "" {
			d, _ := base64.URLEncoding.DecodeString(p.Body.Data)
			return string(d)
		}
	}
	return ""
}
