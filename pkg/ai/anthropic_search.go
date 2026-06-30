package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	anthropicBaseURL     = "https://api.deepseek.com/anthropic"
	maxWebSearchRounds   = 4
	webSearchResearchSys = `You are a web research assistant for video scene metadata.

Use the web_search tool to look up scene identity metadata on the internet.
Search using filename, scene code, title, performers, studio, and any provided URLs.

Find and report ONLY:
- Original scene title
- Performers
- Studio / site
- Release or shoot dates
- Source URLs

Do NOT guess visual content tags (body parts, acts, ratings). Visual tagging comes from video preview analysis, not web search.

Return a concise factual summary. Plain text only. No JSON, XML, DSML, or tool call markup.`
)

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Tools       []anthropicWebTool `json:"tools,omitempty"`
	Temperature float64            `json:"temperature"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicWebTool struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	MaxUses int    `json:"max_uses,omitempty"`
}

type anthropicResponse struct {
	Role       string                  `json:"role"`
	StopReason string                  `json:"stop_reason"`
	Content    []anthropicContentBlock `json:"content"`
	Error      *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type anthropicContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ChatJSONWithWebSearch runs web research first, then requests structured JSON via the text API.
func (c *Client) ChatJSONWithWebSearch(ctx context.Context, systemPrompt, userPrompt string, dest any) error {
	research, err := c.webSearchResearch(ctx, userPrompt)
	if err != nil {
		return c.ChatJSON(ctx, systemPrompt, userPrompt, dest)
	}

	combinedUser := userPrompt
	if research != "" {
		combinedUser += "\n\n## Web Research Results\n" + research
	}

	jsonSystem := systemPrompt + `

Web research results are included in the user message when available.
Use them together with preview description, filename, and existing metadata.
Output ONLY a valid JSON object. No markdown, commentary, or tool markup.`

	return c.ChatJSON(ctx, jsonSystem, combinedUser, dest)
}

func (c *Client) webSearchResearch(ctx context.Context, userPrompt string) (string, error) {
	messages := []anthropicMessage{
		{Role: "user", Content: userPrompt},
	}

	var lastText string
	for round := 0; round < maxWebSearchRounds; round++ {
		resp, err := c.postAnthropic(ctx, webSearchResearchSys, messages, true)
		if err != nil {
			return "", err
		}

		if resp.Error != nil {
			return "", fmt.Errorf("DeepSeek web search API error: %s", resp.Error.Message)
		}

		text := extractAnthropicResearch(resp.Content)
		if isUsefulResearch(text) {
			lastText = text
		}

		switch resp.StopReason {
		case "end_turn", "stop_sequence", "max_tokens":
			if isUsefulResearch(lastText) {
				return lastText, nil
			}
			if isUsefulResearch(text) {
				return text, nil
			}
			return "", fmt.Errorf("web search returned no usable research text")
		case "tool_use", "pause_turn":
			messages = append(messages,
				anthropicMessage{Role: "assistant", Content: resp.Content},
				anthropicMessage{Role: "user", Content: "Continue your web research and provide the factual summary."},
			)
			continue
		default:
			if isUsefulResearch(lastText) {
				return lastText, nil
			}
			if isUsefulResearch(text) {
				return text, nil
			}
			if round < maxWebSearchRounds-1 {
				messages = append(messages,
					anthropicMessage{Role: "assistant", Content: resp.Content},
					anthropicMessage{Role: "user", Content: "Continue your web research and provide the factual summary."},
				)
				continue
			}
			return "", fmt.Errorf("web search stopped with reason %q", resp.StopReason)
		}
	}

	if isUsefulResearch(lastText) {
		return lastText, nil
	}

	return "", fmt.Errorf("web search exceeded maximum rounds")
}

func (c *Client) postAnthropic(ctx context.Context, systemPrompt string, messages []anthropicMessage, withWebSearch bool) (*anthropicResponse, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is not configured")
	}

	model := c.Model
	if model == "" {
		model = defaultModel
	}

	switch model {
	case "deepseek-chat", "deepseek-reasoner":
		model = "deepseek-v4-flash"
	}

	reqBody := anthropicRequest{
		Model:       model,
		MaxTokens:   8192,
		System:      systemPrompt,
		Messages:    messages,
		Temperature: 0.1,
	}

	if withWebSearch {
		reqBody.Tools = []anthropicWebTool{
			{
				Type:    "web_search_20250305",
				Name:    "web_search",
				MaxUses: 5,
			},
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(anthropicBaseURL, "/") + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("DeepSeek web search API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse DeepSeek web search response: %w", err)
	}

	return &anthropicResp, nil
}

func extractAnthropicResearch(blocks []anthropicContentBlock) string {
	var parts []string

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if text := strings.TrimSpace(block.Text); text != "" && !looksLikeToolMarkup(text) {
				parts = append(parts, text)
			}
		case "web_search_tool_result":
			if text := strings.TrimSpace(block.Text); text != "" {
				parts = append(parts, text)
			}
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func looksLikeToolMarkup(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "dsml") ||
		strings.Contains(lower, "tool_calls") ||
		strings.Contains(lower, "<｜｜") ||
		strings.Contains(lower, "web_search") && strings.Contains(lower, "invoke")
}

func isUsefulResearch(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if looksLikeToolMarkup(text) {
		return false
	}
	return len(text) >= 20
}
