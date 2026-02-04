package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type Client struct {
	client *openai.Client
	model  openai.ChatModel
	temp   float64
	maxTok int
}

type Draft struct {
	Title     string   `json:"title"`
	Slug      string   `json:"slug"`
	ContentMD string   `json:"content_md"`
	Tags      []string `json:"tags"`
}

func New(apiKey, model string, temperature float64, maxTokens int) *Client {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &Client{
		client: client,
		model:  openai.ChatModel(model),
		temp:   temperature,
		maxTok: maxTokens,
	}
}

func (c *Client) GenerateDraft(ctx context.Context, systemPrompt, userPrompt string) (*Draft, error) {
	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		Model: c.model,
	}
	if c.temp > 0 {
		params.Temperature = openai.Float(c.temp)
	}
	if c.maxTok > 0 {
		params.MaxTokens = openai.Int(int64(c.maxTok))
	}

	resp, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, errors.New("no choices returned")
	}
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	if content == "" {
		return nil, errors.New("empty content")
	}

	jsonText := extractJSON(content)
	var draft Draft
	if err := json.Unmarshal([]byte(jsonText), &draft); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	if draft.Title == "" || draft.Slug == "" || draft.ContentMD == "" {
		return nil, errors.New("draft missing required fields")
	}
	return &draft, nil
}

func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
