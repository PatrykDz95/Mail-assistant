package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/gmail/v1"
	"mailassist/internal/domain/email"
)

// Client implements Gmail operations (adapter)
type Client struct {
	Srv      *gmail.Service
	LabelIDs map[string]string
}

// NewClient creates a new Gmail client
func NewClient(srv *gmail.Service) *Client {
	return &Client{
		Srv:      srv,
		LabelIDs: make(map[string]string),
	}
}

// labelMap maps domain labels to Gmail label names
// TODO: check if this makes sense
var labelMap = map[email.Label]string{
	email.LabelNewsletter:   "Newsletter",
	email.LabelPrivate:      "Private",
	email.LabelBusiness:     "Business",
	email.LabelPayments:     "Payments",
	email.LabelActionNeeded: "Action Needed",
	email.LabelJunk:         "Junk",
}

func (c *Client) InitLabels() error {
	// Fetch existing labels
	list, err := c.Srv.Users.Labels.List("me").Do()
	if err != nil {
		return err
	}

	// Fill existing labels
	for _, l := range list.Labels {
		c.LabelIDs[l.Name] = l.Id
	}

	// Ensure required labels exist
	for _, gmailName := range labelMap {
		if _, ok := c.LabelIDs[gmailName]; ok {
			continue
		}

		// Try create the label
		created, err := c.Srv.Users.Labels.Create("me", &gmail.Label{Name: gmailName}).Do()
		if err != nil {
			if strings.Contains(err.Error(), "Label name exists or conflicts") {
				log.Printf("Label %q already exists (409 conflict), continuing", gmailName)
				continue
			}
			return err
		}

		c.LabelIDs[gmailName] = created.Id
	}

	return nil
}

func (c *Client) FetchEmail(ctx context.Context, messageID string) (*email.Email, error) {
	msg, err := c.Srv.Users.Messages.Get("me", messageID).Format("FULL").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("gmail get message: %w", err)
	}

	return email.NewEmail(
		messageID,
		extractHeader(msg, "From"),
		extractHeader(msg, "Subject"),
		extractBody(msg),
	), nil
}

func (c *Client) ApplyLabel(ctx context.Context, messageID string, label email.Label) error {
	gmailName := labelMap[label]
	labelID := c.LabelIDs[gmailName]

	if labelID == "" {
		return fmt.Errorf("label ID not found for %q", gmailName)
	}

	_, err := c.Srv.Users.Messages.Modify("me", messageID, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{labelID},
	}).Context(ctx).Do()

	return err
}

func (c *Client) CreateDraft(ctx context.Context, recipient, subject, body string) error {
	raw := fmt.Sprintf(
		"To: %s\r\nSubject: %s\r\n\r\n%s",
		recipient, subject, body,
	)

	encoded := base64.URLEncoding.EncodeToString([]byte(raw))

	_, err := c.Srv.Users.Drafts.Create("me", &gmail.Draft{
		Message: &gmail.Message{
			Raw: encoded,
		},
	}).Context(ctx).Do()

	return err
}

func (c *Client) FetchNewMessagesSince(ctx context.Context, historyID uint64) ([]string, error) {
	var messageIDs []string

	resp, err := c.Srv.Users.History.List("me").
		StartHistoryId(historyID).
		HistoryTypes("messageAdded").
		Context(ctx).
		Do()

	if err != nil {
		return nil, fmt.Errorf("gmail history list: %w", err)
	}

	for _, h := range resp.History {
		for _, added := range h.MessagesAdded {
			if added.Message != nil && !isDraft(added.Message) {
				messageIDs = append(messageIDs, added.Message.Id)
			}
		}
	}

	return messageIDs, nil
}

// EnableWatch enables Gmail push notifications
func (c *Client) EnableWatch(ctx context.Context, topicName string) error {
	req := &gmail.WatchRequest{
		TopicName: topicName,
	}

	_, err := c.Srv.Users.Watch("me", req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("gmail watch: %w", err)
	}

	return nil
}

func (c *Client) ListMessagesFromInbox(ctx context.Context, maxResults int64) ([]string, error) {
	resp, err := c.Srv.Users.Messages.List("me").
		LabelIds("INBOX").
		MaxResults(maxResults).
		Context(ctx).
		Do()

	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	var ids []string
	for _, msg := range resp.Messages {
		ids = append(ids, msg.Id)
	}

	return ids, nil
}

func extractHeader(msg *gmail.Message, name string) string {
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

func isDraft(msg *gmail.Message) bool {
	for _, labelID := range msg.LabelIds {
		if labelID == "DRAFT" {
			return true
		}
	}
	return false
}
