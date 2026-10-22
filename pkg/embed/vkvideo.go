package embed

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	vkPlayerParamsRe = regexp.MustCompile(`(?i)var\s+playerParams\s*=\s*({.+?});`)
	vkVideoDataRe    = regexp.MustCompile(`(?i)window\.vkVideoData\s*=\s*({.+?});`)
	vkDeletedRe      = regexp.MustCompile(`(?i)видеозапись недоступна|video has been deleted|Video not found|Это видео недоступно`)
	vkDurationRe     = regexp.MustCompile(`(?i)property="og:duration"\s+content="(\d+)"`)
	vkBrowserCheckRe = regexp.MustCompile(`(?i)(?:browser is out of date|браузер устарел)`)
	// VK sometimes embeds video data differently
	vkVideoDataAltRe = regexp.MustCompile(`(?i)window\.videoData\s*=\s*({.+?});`)
)

// vkPlayerParams is the JSON structure embedded in VK video pages.
type vkPlayerParams struct {
	Player   vkPlayer `json:"player"`
	Duration float64  `json:"duration"`
}

type vkPlayer struct {
	Videos []vkVideoTrack `json:"videos"`
}

type vkVideoTrack struct {
	URL     string `json:"url"`
	Type    string `json:"type"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Quality string `json:"quality"`
}

// extended player params - wider capture for different VK formats
type vkPlayerParamsRaw struct {
	Params   *vkPlayerParams `json:"params"`
	Player   *vkPlayer       `json:"player"`
	Duration interface{}     `json:"duration"`
	Videos   []vkVideoTrack  `json:"videos"`
}

func validateVKVideoURL(rawURL string) bool {
	// VK video URLs should be valid HTTP(S) URLs and look like video files
	rawURL = strings.ReplaceAll(rawURL, "\\/", "/")
	
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return false
	}
	
	lower := strings.ToLower(rawURL)
	// Must be a video URL - .mp4, .m3u8, or video-related path
	if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".m3u8") {
		return true
	}
	// Also accept URLs that contain typical VK video CDN patterns
	if strings.Contains(lower, "/video") || strings.Contains(lower, "/videos/") || strings.Contains(lower, ".m3u") {
		return true
	}
	
	return false
}

func extractVKVideoStreamVariants(pageURL, embedSrc string) ([]HLSVariant, error) {
	payload, err := fetchVKVideoData(pageURL, embedSrc)
	if err != nil {
		return nil, err
	}

	byHeight := make(map[int]HLSVariant)

	for _, track := range payload.Player.Videos {
		if track.URL == "" {
			continue
		}

		url := strings.ReplaceAll(track.URL, "\\/", "/")
		
		// Skip invalid URLs (prevents garbage from bot-check page being used as stream)
		if !validateVKVideoURL(url) {
			continue
		}

		height := track.Height
		if height == 0 {
			height = inferHeightFromURL(url)
		}
		if height == 0 && track.Width > 0 {
			// Approximate height from width (assuming 16:9)
			height = track.Width * 9 / 16
		}
		if height == 0 {
			continue
		}

		trackType := strings.ToLower(track.Type)
		if trackType == "" {
			if strings.Contains(strings.ToLower(url), ".m3u8") {
				trackType = "hls"
			} else {
				trackType = "mp4"
			}
		}

		variant := HLSVariant{
			Height: height,
			Width:  track.Width,
			URL:    url,
		}

		score := variantScore(trackType, url)
		if existing, ok := byHeight[height]; !ok || score > variantScore(inferFormatFromURL(existing.URL), existing.URL) {
			byHeight[height] = variant
		}
	}

	if len(byHeight) == 0 {
		return nil, ErrStreamNotFound
	}

	variants := make([]HLSVariant, 0, len(byHeight))
	for _, v := range byHeight {
		variants = append(variants, v)
	}

	return prepareVariants(variants), nil
}

func inferFormatFromURL(rawURL string) string {
	lower := strings.ToLower(rawURL)
	if strings.Contains(lower, ".m3u8") {
		return "hls"
	}
	if strings.HasSuffix(lower, ".mp4") {
		return "mp4"
	}
	return ""
}

func extractVKVideoDuration(pageURL, embedSrc string) (float64, error) {
	if duration, err := extractVKOGDuration(pageURL); err == nil && duration > 0 {
		return duration, nil
	}
	if duration, err := extractVKOGDuration(embedSrc); err == nil && duration > 0 {
		return duration, nil
	}

	payload, err := fetchVKVideoData(pageURL, embedSrc)
	if err != nil {
		return 0, err
	}

	if payload.Duration > 0 {
		return payload.Duration, nil
	}

	return 0, ErrStreamNotFound
}

func extractVKOGDuration(pageURL string) (float64, error) {
	body, err := fetchVKVideoPage(pageURL)
	if err != nil {
		return 0, err
	}

	if match := vkDurationRe.FindStringSubmatch(string(body)); len(match) == 2 {
		seconds, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return 0, err
		}
		return seconds, nil
	}

	return 0, ErrStreamNotFound
}

func fetchVKVideoData(pageURL, embedSrc string) (vkPlayerParams, error) {
	pageURLs := vkVideoDataPageURLs(pageURL, embedSrc)

	var lastErr error
	for _, u := range pageURLs {
		body, err := fetchVKVideoPage(u)
		if err != nil {
			lastErr = err
			continue
		}

		html := string(body)
		if vkDeletedRe.MatchString(html) {
			lastErr = fmt.Errorf("video has been deleted")
			continue
		}

		payload, err := parseVKPlayerParams(html)
		if err == nil && len(payload.Player.Videos) > 0 {
			return payload, nil
		}
	}

	if lastErr != nil {
		return vkPlayerParams{}, lastErr
	}
	return vkPlayerParams{}, ErrStreamNotFound
}

func vkVideoDataPageURLs(rawURL, embedSrc string) []string {
	var pageURLs []string
	addURL := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		for _, existing := range pageURLs {
			if existing == candidate {
				return
			}
		}
		pageURLs = append(pageURLs, candidate)
	}

	// embed URL should be the most reliable
	addURL(embedSrc)
	addURL(embedSrc + "&__ref=stash") // try with extra parameter

	if viewURL := VKVideoViewPageURL(embedSrc); viewURL != "" {
		addURL(viewURL)
		addURL(strings.Replace(viewURL, "vk.com", "m.vk.com", 1))
	}
	if viewURL := VKVideoViewPageURL(rawURL); viewURL != "" {
		addURL(viewURL)
		addURL(strings.Replace(viewURL, "vk.com", "m.vk.com", 1))
	}
	if rawURL != embedSrc {
		addURL(rawURL)
	}
	return pageURLs
}

var (
	vkClientOnce  sync.Once
	vkHTTPClient  *http.Client
	vkCookieJar   *cookiejar.Jar
)

func getVKClient() *http.Client {
	vkClientOnce.Do(func() {
		jar, _ := cookiejar.New(nil)
		vkCookieJar = jar
		vkHTTPClient = &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		}
		// pre-fetch cookies from VK
		initVKCookies()
	})
	return vkHTTPClient
}

func initVKCookies() {
	req, _ := http.NewRequest(http.MethodGet, "https://vk.com/", nil)
	setVKHeaders(req, false)
	// don't care about the response, just want cookies
	resp, err := vkHTTPClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func setVKHeaders(req *http.Request, isVideoPage bool) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Referer", "https://vk.com/")
	req.Header.Set("Origin", "https://vk.com")
	req.Header.Set("Sec-Ch-Ua", `"Not/A)Brand";v="99", "Google Chrome";v="127", "Chromium";v="127"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("DNT", "1")
	req.Header.Set("Priority", "u=0, i")
}

func fetchVKVideoPage(pageURL string) ([]byte, error) {
	client := getVKClient()

	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}

	setVKHeaders(req, true)

	// Add a cache-busting parameter to avoid getting cached bot-check pages
	q := req.URL.Query()
	q.Set("_t", fmt.Sprintf("%d", time.Now().UnixMilli()))
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vk video page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, err
	}

	html := string(body)

	if vkBrowserCheckRe.MatchString(html) {
		return nil, fmt.Errorf("vk returned browser check page")
	}

	return []byte(html), nil
}

func parseVKPlayerParams(html string) (vkPlayerParams, error) {
	// Try various JSON patterns that VK uses to embed video data

	extractors := []func(string) (vkPlayerParams, bool){
		// playerParams = {player: {videos: [...]}, duration: N}
		tryParsePlayerParams,
		// window.vkVideoData = {player: {videos: [...]}, duration: N}
		tryParseVKVideoData,
		// window.videoData = {...}
		tryParseVideoData,
	}

	for _, extractor := range extractors {
		if params, ok := extractor(html); ok {
			return params, nil
		}
	}

	return vkPlayerParams{}, ErrStreamNotFound
}

func tryParsePlayerParams(html string) (vkPlayerParams, bool) {
	if m := vkPlayerParamsRe.FindStringSubmatch(html); len(m) == 2 {
		jsonStr := cleanVKJSON(m[1])
		var params vkPlayerParams
		if err := json.Unmarshal([]byte(jsonStr), &params); err == nil && len(params.Player.Videos) > 0 {
			return params, true
		}
		// try raw format
		var raw vkPlayerParamsRaw
		if err := json.Unmarshal([]byte(jsonStr), &raw); err == nil {
			if raw.Player != nil && len(raw.Player.Videos) > 0 {
				return vkPlayerParams{Player: *raw.Player, Duration: parseDuration(raw.Duration)}, true
			}
			if raw.Params != nil && len(raw.Params.Player.Videos) > 0 {
				return *raw.Params, true
			}
		}
	}
	return vkPlayerParams{}, false
}

func tryParseVKVideoData(html string) (vkPlayerParams, bool) {
	if m := vkVideoDataRe.FindStringSubmatch(html); len(m) == 2 {
		jsonStr := cleanVKJSON(m[1])
		var data vkPlayerParams
		if err := json.Unmarshal([]byte(jsonStr), &data); err == nil && len(data.Player.Videos) > 0 {
			return data, true
		}
		var raw vkPlayerParamsRaw
		if err := json.Unmarshal([]byte(jsonStr), &raw); err == nil {
			if raw.Player != nil && len(raw.Player.Videos) > 0 {
				return vkPlayerParams{Player: *raw.Player, Duration: parseDuration(raw.Duration)}, true
			}
		}
	}
	return vkPlayerParams{}, false
}

func tryParseVideoData(html string) (vkPlayerParams, bool) {
	if m := vkVideoDataAltRe.FindStringSubmatch(html); len(m) == 2 {
		jsonStr := cleanVKJSON(m[1])
		var raw vkPlayerParamsRaw
		if err := json.Unmarshal([]byte(jsonStr), &raw); err == nil {
			if raw.Player != nil && len(raw.Player.Videos) > 0 {
				return vkPlayerParams{Player: *raw.Player, Duration: parseDuration(raw.Duration)}, true
			}
			if len(raw.Videos) > 0 {
				return vkPlayerParams{Player: vkPlayer{Videos: raw.Videos}}, true
			}
		}
	}
	return vkPlayerParams{}, false
}

func cleanVKJSON(s string) string {
	// VK uses various escape patterns in embedded JSON
	s = strings.ReplaceAll(s, "\\/", "/")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

func parseDuration(d interface{}) float64 {
	if d == nil {
		return 0
	}
	switch v := d.(type) {
	case float64:
		return v
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

// fetchVKStreamResource downloads a stream resource from VK CDN using the VK-specific client with cookies.
func fetchVKStreamResource(rawURL string) ([]byte, string, error) {
	client := getVKClient()

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}

	setVKHeaders(req, true)

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("vk upstream returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
	if err != nil {
		return nil, "", err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}

	return body, contentType, nil
}
