package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.deepseek.com"
	defaultModel   = "deepseek-v4-flash"
)

// Client calls the DeepSeek OpenAI-compatible chat API (text-only).
type Client struct {
	APIKey  string
	BaseURL string
	Model   string
	HTTP    *http.Client
}

func NewClient(apiKey, model string) *Client {
	if model == "" {
		model = defaultModel
	}

	return &Client{
		APIKey:  apiKey,
		BaseURL: defaultBaseURL,
		Model:   model,
		HTTP: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Thinking       *thinkingConfig `json:"thinking,omitempty"`
	Temperature    float64         `json:"temperature"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type thinkingConfig struct {
	Type string `json:"type"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const defaultChatMaxTokens = 16384

// ChatJSON sends a text-only chat request and parses the assistant response as JSON into dest.
func (c *Client) ChatJSON(ctx context.Context, systemPrompt, userPrompt string, dest any) error {
	return c.chatJSON(ctx, systemPrompt, userPrompt, dest, false)
}

func (c *Client) chatJSON(ctx context.Context, systemPrompt, userPrompt string, dest any, compactRetry bool) error {
	if c.APIKey == "" {
		return fmt.Errorf("DeepSeek API key is not configured")
	}

	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	model := c.Model
	if model == "" {
		model = defaultModel
	}

	// Legacy model aliases do not support multimodal input; map to current models.
	switch model {
	case "deepseek-chat":
		model = "deepseek-v4-flash"
	case "deepseek-reasoner":
		model = "deepseek-v4-flash"
	}

	reqBody := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		Thinking:       &thinkingConfig{Type: "disabled"},
		Temperature:    0.1,
		MaxTokens:      defaultChatMaxTokens,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("DeepSeek API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return fmt.Errorf("failed to parse DeepSeek response: %w", err)
	}

	if chatResp.Error != nil {
		return fmt.Errorf("DeepSeek API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return fmt.Errorf("DeepSeek API returned no choices")
	}

	choice := chatResp.Choices[0]
	content := choice.Message.Content
	jsonContent := extractJSONFromText(content)
	if err := json.Unmarshal([]byte(jsonContent), dest); err != nil {
		if !compactRetry {
			retryPrompt := userPrompt + "\n\nIMPORTANT: Your previous response was invalid or truncated. Output a COMPLETE valid JSON object. Prefer tags from the provided lists; new tags only when clearly meaningful. Never enumerate similar variants. At most one \"cum on …\" and one \"cum in …\" tag on/in a body part."
			return c.chatJSON(ctx, systemPrompt, retryPrompt, dest, true)
		}
		return fmt.Errorf("failed to parse AI JSON output: %w (content: %s)", err, truncateForError(content))
	}

	return nil
}

var jsonFencePattern = regexp.MustCompile("(?s)```(?:json)?\\s*([\\s\\S]*?)```")

func extractJSONFromText(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "{") {
		return content
	}

	if match := jsonFencePattern.FindStringSubmatch(content); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		return content[start : end+1]
	}

	return content
}

func truncateForError(content string) string {
	const maxLen = 500
	content = strings.TrimSpace(content)
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "…"
}
