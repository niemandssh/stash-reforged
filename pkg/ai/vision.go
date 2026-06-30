package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultVisionBaseURL = "https://api.groq.com/openai/v1"
	defaultVisionModel   = "meta-llama/llama-4-scout-17b-16e-instruct"
)

// VisionClient calls an OpenAI-compatible vision API to describe images.
type VisionClient struct {
	APIKey  string
	BaseURL string
	Model   string
	HTTP    *http.Client
}

func NewVisionClient(apiKey, baseURL, model string) *VisionClient {
	if baseURL == "" {
		baseURL = defaultVisionBaseURL
	}
	if model == "" {
		model = defaultVisionModel
	}

	return &VisionClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
		HTTP: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

type visionChatRequest struct {
	Model       string              `json:"model"`
	Messages    []visionChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
}

type visionChatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type visionTextPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type visionImagePart struct {
	Type     string `json:"type"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

type visionChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const visionPanelPrompt = `You are analyzing preview panels from an adult video scene. Each image is a grid of thumbnails sampled through the video, in chronological order (left-to-right, top-to-bottom within each panel; panels are sequential). The LAST panel(s) oversample the ending — inspect them carefully for climax/finish.

Describe ONLY what you can actually see. Do not invent information.

Be explicit about visibility of body parts and sex acts across ALL panels:
- For breasts/tits, ass, pussy/vagina, face, body overall: state whether each is visible, and how prominently (not visible / briefly / clearly / main focus).
- Note if the scene has distinct phases (e.g. clothed intro then nude) — describe each phase separately.
- Note sex acts and POSITIONS/POSES visible: missionary (woman on back, face-to-face), doggystyle (from behind), cowgirl (woman riding on top), reverse cowgirl, blowjob/oral, etc. List every pose seen in any panel.
- FINISH / CLIMAX: inspect the LAST panel in detail (bottom-right cells are latest). Describe where cum lands if visible — mouth/lips, face/facial, tits, body, internal/creampie, drip, swallow, pulling out, etc. Be specific. If unsure, say what is visible in the last cells.
- Note if content is live action, 3D/CGI, anime/hentai, or mixed.
- UNDERWEAR vs HOSIERY: underwear = bra and panties only. Stockings, thigh-highs, fishnets, and garter belts are NOT underwear — report them separately.
- CLOTHING STATUS: for the female performer, state whether she is fully nude (no bra, panties, or pants/shorts) in EVERY thumbnail, or if any panel shows clothed/partially clothed segments. Stockings without panties still means underwear_visible=no.
- Performers: count, appearance, hair color/style, eye color (when face is visible), skin tone, setting/location.
- ACCESSORIES: list ONLY items actually worn and clearly visible. Do NOT guess. A thin shadow on the neck is NOT a choker. If no choker necklace is visible, choker must be no.
- Any studio/watermark/title text readable on frames.

End with a short "Visibility ratings" section using this scale for each applicable category:
0 = not visible at all in any thumbnail
1-3 = briefly or partially visible
4-6 = clearly visible in several thumbnails
7-9 = prominent/main focus in many thumbnails

Required summary lines (always include all three):
Visibility ratings: tits=N, ass=N, pussy=N, face=N, vaginal=N, anal=N, oral=N, creampie=N, cumshot=N, cum_on_mouth=N (omit categories not applicable)
Clothing summary: fully_nude_all_panels=yes|no|partial, bra_visible=no|yes|partial, underwear_visible=no|yes|partial, pants_visible=no|yes|partial, stockings_visible=no|yes|partial
Accessories summary: list only visible items as name=yes(color) — e.g. earrings=yes, scrunchies=yes(green). Use choker=no unless a choker necklace is clearly on the neck. Omit absent items or set them no.
Poses summary: missionary=no|yes, doggystyle=no|yes, cowgirl=no|yes, reverse_cowgirl=no|yes, blowjob=no|yes, anal=no|yes, standing=no|yes (list every pose seen)
Finish summary: visible=no|yes, type=none|creampie|facial|cum_on_mouth|cum_in_mouth|cum_on_tits|cumshot|swallow|pullout|other

Be factual and concise.`

func (c *VisionClient) describeVisionPanels(ctx context.Context, panels [][]byte) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("vision API key is not configured")
	}
	if len(panels) == 0 {
		return "", fmt.Errorf("no vision panels to analyze")
	}

	prompt := fmt.Sprintf("%s\n\nYou will receive %d preview panel image(s).", visionPanelPrompt, len(panels))

	content := []any{
		visionTextPart{Type: "text", Text: prompt},
	}

	for _, panel := range panels {
		encoded := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(panel)
		content = append(content, visionImagePart{
			Type: "image_url",
			ImageURL: struct {
				URL string `json:"url"`
			}{URL: encoded},
		})
	}

	reqBody := visionChatRequest{
		Model: c.Model,
		Messages: []visionChatMessage{
			{Role: "user", Content: content},
		},
		Temperature: 0.1,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("vision API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp visionChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse vision API response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("vision API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("vision API returned no choices")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}
