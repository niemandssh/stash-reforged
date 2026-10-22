package embed

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// SiteJWPlayer is a watch page that embeds JW Player with signed CDN sources.
const SiteJWPlayer Site = "jwplayer"

var (
	jwPlayerMarkerRe   = regexp.MustCompile(`(?i)jwplayer`)
	jwPlaylistAssignRe = regexp.MustCompile(`(?i)window\.playlist\s*=\s*\{`)
	jwVideoDurationRe  = regexp.MustCompile(`(?i)<meta\s+property="video:duration"\s+content="(\d+)"`)
	pvvStreamWatchRe   = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*(?:noodlemagazine|ukdevilz|mat6tube)\.[a-z.]+/watch/(-?\d+_\d+)`)
	pornzogVideoRe     = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*pornzog\.[a-z.]+/video/(\d+)`)
)

type jwPlaylistSource struct {
	File  string `json:"file"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

type jwPlaylistPayload struct {
	Sources []jwPlaylistSource `json:"sources"`
}

// IsJWPlayerSite reports whether the site is a JW Player watch page.
func IsJWPlayerSite(site Site) bool {
	return site == SiteJWPlayer
}

func isPvvStreamHost(host string) bool {
	return strings.HasSuffix(host, "noodlemagazine.com") ||
		strings.HasSuffix(host, "ukdevilz.com") ||
		strings.HasSuffix(host, "mat6tube.com")
}

func isPornzogHost(host string) bool {
	return strings.HasSuffix(host, "pornzog.com")
}

func parseJWPlayerPage(raw string) (string, Site, error) {
	if !pvvStreamWatchRe.MatchString(raw) && !pornzogVideoRe.MatchString(raw) {
		return "", SiteJWPlayer, ErrInvalidURL
	}
	// No iframe — playback uses sceneStreams with request-time URL refresh.
	return "", SiteJWPlayer, nil
}

func extractJWPlayerStreamVariants(rawURL string) ([]HLSVariant, error) {
	body, err := fetchJWPlayerPage(rawURL)
	if err != nil {
		return nil, err
	}

	return extractJWPlayerStreamVariantsFromHTML(string(body))
}

func extractJWPlayerStreamVariantsFromHTML(html string) ([]HLSVariant, error) {
	if !jwPlayerMarkerRe.MatchString(html) {
		return nil, fmt.Errorf("%w: jwplayer not found on page", ErrStreamNotFound)
	}

	payload, err := extractJWPlaylistPayload(html)
	if err != nil {
		return nil, err
	}

	var variants []HLSVariant
	seen := make(map[string]struct{}, len(payload.Sources))
	for _, source := range payload.Sources {
		fileURL := strings.TrimSpace(source.File)
		if fileURL == "" {
			continue
		}
		if _, ok := seen[fileURL]; ok {
			continue
		}
		seen[fileURL] = struct{}{}

		height := parseJWSourceHeight(source)
		variants = append(variants, HLSVariant{
			URL:    fileURL,
			Height: height,
		})
	}

	if len(variants) == 0 {
		return nil, fmt.Errorf("%w: jwplayer sources not found", ErrStreamNotFound)
	}

	return prepareVariants(variants), nil
}

func extractJWPlayerDuration(rawURL string) (float64, error) {
	body, err := fetchJWPlayerPage(rawURL)
	if err != nil {
		return 0, err
	}

	html := string(body)
	if match := jwVideoDurationRe.FindStringSubmatch(html); len(match) == 2 {
		seconds, err := strconv.ParseFloat(match[1], 64)
		if err == nil && seconds > 0 {
			return seconds, nil
		}
	}

	return 0, ErrStreamNotFound
}

func extractJWPlaylistPayload(html string) (jwPlaylistPayload, error) {
	loc := jwPlaylistAssignRe.FindStringIndex(html)
	if loc == nil {
		return jwPlaylistPayload{}, fmt.Errorf("%w: jwplayer playlist not found", ErrStreamNotFound)
	}

	start := loc[1] - 1
	if start < 0 || start >= len(html) || html[start] != '{' {
		return jwPlaylistPayload{}, fmt.Errorf("%w: jwplayer playlist not found", ErrStreamNotFound)
	}

	playlistJSON, err := extractBalancedJSObject(html, start)
	if err != nil {
		return jwPlaylistPayload{}, err
	}

	var payload jwPlaylistPayload
	if err := json.Unmarshal([]byte(playlistJSON), &payload); err != nil {
		return jwPlaylistPayload{}, fmt.Errorf("parse jwplayer playlist: %w", err)
	}

	return payload, nil
}

func parseJWSourceHeight(source jwPlaylistSource) int {
	if label := strings.TrimSpace(source.Label); label != "" {
		if height, err := strconv.Atoi(label); err == nil && height > 0 {
			return height
		}
	}
	return inferHeightFromURL(source.File)
}

func fetchJWPlayerPage(pageURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}
	applyPageHeaders(req, pageURL)

	resp, err := streamHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("page fetch returned status %d", resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
}

func extractCachedJWPlayerVariants(rawURL string) ([]HLSVariant, error) {
	cacheKey := "jw:" + strings.TrimSpace(rawURL)
	if variants, ok := getCachedVariants(cacheKey); ok {
		return variants, nil
	}

	result, err, _ := remotePageVariantGroup.Do(cacheKey, func() (interface{}, error) {
		if variants, ok := getCachedVariants(cacheKey); ok {
			return variants, nil
		}

		variants, err := extractJWPlayerStreamVariants(rawURL)
		if err != nil {
			return nil, err
		}
		setCachedVariants(cacheKey, variants)
		return variants, nil
	})
	if err != nil {
		return nil, err
	}

	variants, ok := result.([]HLSVariant)
	if !ok {
		return nil, ErrStreamNotFound
	}

	return variants, nil
}
