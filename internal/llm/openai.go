package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type Result struct {
	Category string `json:"category"`
	Label    string `json:"label"`
	Reply    string `json:"reply"`
}

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
		return nil, fmt.Errorf("MODEL_NAME is not set")
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &Client{
		api:   client,
		model: modelName,
	}, nil
}

func (c *Client) AnalyzeEmail(ctx context.Context, subject, body string) (*Result, error) {

	prompt := fmt.Sprintf(`Analyze the following email and return ONLY pure JSON, without markdown and without backticks.

							Categories: ["business","private","payments","action_needed","spam","newsletter"]
							
							Rules:
							- Promotions → newsletter
							- Request for reply / decision → action_needed
							- Bank / invoices → payments
							- B2B offers → business
							- Private conversations → private
							- Junk → spam
							
							Reply should be a short draft reply only for "action_needed" category. For other categories, reply should be empty string.

							Format:
							{"category":"...","label":"...","reply":"..."}
							
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
		return nil, err
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

	var out Result
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("cannot parse JSON: %v (raw=%s)", err, text)
	}

	return &out, nil
}
