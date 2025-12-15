package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"mailassist/internal/domain/email"
)

type Client struct {
	api   openai.Client
	model string
}

func NewClient() (*Client, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}

	modelName := os.Getenv("MODEL_NAME")
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &Client{
		api:   client,
		model: modelName,
	}, nil
}

type llmResponse struct {
	Category   string `json:"category"`
	Label      string `json:"label"`
	Reply      string `json:"reply"`
	SenderName string `json:"sender_name"`
}

func (c *Client) Classify(ctx context.Context, subject, body string) (*email.Classification, error) {
	prompt := fmt.Sprintf(`Analyze the following email and return ONLY pure JSON, without markdown and without backticks.

Categories: ["business","private","payments","action_needed","junk","newsletter"]

Labels:
- if email is Promotions then it's "newsletter"
- if email is private email then it's "private"
- if email is Bank/invoices then it's "payments"
- if email is business offer/linkedIn then it's "business"
- if email is Junk/spam then it's "junk"

If the email is from a real person and not spam/newsletter/ads/invoices, and requires a response or action, categorize it as "action_needed" and draft a short, polite reply in the language of origin.
Reply should be a short draft reply only for "action_needed" category with sender name included. For other categories, reply should be empty string.
Include sender_name in the output, extracted from the email body or subject if possible, otherwise use "there".

Format:
{"category":"...","label":"...","reply":"...", "sender_name":"..."}

Email:
Subject: %s

Body:
%s`, subject, body)

	resp, err := c.api.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: c.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai api error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty LLM response")
	}

	text := resp.Choices[0].Message.Content
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var llmResp llmResponse
	if err := json.Unmarshal([]byte(text), &llmResp); err != nil {
		log.Printf("LLM parse error: %v (raw=%s)", err, text)
		return nil, fmt.Errorf("cannot parse JSON: %w", err)
	}

	return email.NewClassification(
		email.Category(llmResp.Category),
		email.Label(llmResp.Label),
		llmResp.Reply,
		llmResp.SenderName,
	), nil
}
