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

// SiteNewgrounds is a Newgrounds portal movie page.
const SiteNewgrounds Site = "newgrounds"

var (
	newgroundsPageRe            = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*newgrounds\.com/portal/view/(\d+)`)
	newgroundsEmbedControllerRe = regexp.MustCompile(`(?i)embedController\(\[{"url"\s*:\s*("[^"]+"),`)
	newgroundsDurationRe        = regexp.MustCompile(`(?i)"duration"\s*:\s*["']?(\d+)["']?`)
)

type newgroundsVideoResponse struct {
	Sources map[string][]newgroundsSource `json:"sources"`
}

type newgroundsSource struct {
	Type string `json:"type"`
	Src  string `json:"src"`
}

// IsNewgroundsSite reports whether the site is a Newgrounds portal page.
func IsNewgroundsSite(site Site) bool {
	return site == SiteNewgrounds
}

func isNewgroundsHost(host string) bool {
	return host == "newgrounds.com" || strings.HasSuffix(host, ".newgrounds.com")
}

func parseNewgroundsPage(raw string) (string, Site, error) {
	if !newgroundsPageRe.MatchString(raw) {
		return "", SiteNewgrounds, ErrInvalidURL
	}
	return "", SiteNewgrounds, nil
}

func newgroundsMediaID(rawURL string) string {
	if m := newgroundsPageRe.FindStringSubmatch(rawURL); len(m) == 2 {
		return m[1]
	}
	return ""
}

func extractNewgroundsStreamVariants(rawURL string) ([]HLSVariant, error) {
	mediaID := newgroundsMediaID(rawURL)
	if mediaID == "" {
		return nil, ErrInvalidURL
	}

	variants, err := extractNewgroundsStreamVariantsFromAPI(rawURL, mediaID)
	if err == nil && len(variants) > 0 {
		return variants, nil
	}

	pageVariants, pageErr := extractNewgroundsStreamVariantsFromPage(rawURL, mediaID)
	if pageErr == nil && len(pageVariants) > 0 {
		return pageVariants, nil
	}
	if err != nil {
		return nil, err
	}
	if pageErr != nil {
		return nil, pageErr
	}
	return nil, ErrStreamNotFound
}

func extractNewgroundsStreamVariantsFromAPI(pageURL, mediaID string) ([]HLSVariant, error) {
	body, err := fetchNewgroundsVideoJSON(pageURL, mediaID)
	if err != nil {
		return nil, err
	}

	var payload newgroundsVideoResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse newgrounds video json: %w", err)
	}

	var variants []HLSVariant
	seen := make(map[string]struct{})
	for formatID, sources := range payload.Sources {
		height := parseNewgroundsFormatHeight(formatID)
		for _, source := range sources {
			fileURL := strings.TrimSpace(source.Src)
			if fileURL == "" {
				continue
			}
			if strings.HasPrefix(strings.ToLower(source.Type), "audio/") {
				continue
			}
			if _, ok := seen[fileURL]; ok {
				continue
			}
			seen[fileURL] = struct{}{}
			if height == 0 {
				height = inferHeightFromURL(fileURL)
			}
			variants = append(variants, HLSVariant{
				URL:    fileURL,
				Height: height,
			})
		}
	}

	if len(variants) == 0 {
		return nil, ErrStreamNotFound
	}
	return prepareVariants(variants), nil
}

func extractNewgroundsStreamVariantsFromPage(pageURL, mediaID string) ([]HLSVariant, error) {
	body, err := fetchNewgroundsPage(pageURL)
	if err != nil {
		return nil, err
	}

	html := string(body)
	match := newgroundsEmbedControllerRe.FindStringSubmatch(html)
	if len(match) != 2 {
		return nil, ErrStreamNotFound
	}

	var fileURL string
	if err := json.Unmarshal([]byte(match[1]), &fileURL); err != nil {
		return nil, fmt.Errorf("parse newgrounds embed url: %w", err)
	}
	fileURL = strings.TrimSpace(fileURL)
	if fileURL == "" {
		return nil, ErrStreamNotFound
	}

	return []HLSVariant{{
		URL:    fileURL,
		Height: inferHeightFromURL(fileURL),
	}}, nil
}

func extractNewgroundsDuration(rawURL string) (float64, error) {
	body, err := fetchNewgroundsPage(rawURL)
	if err != nil {
		return 0, err
	}

	if match := newgroundsDurationRe.FindStringSubmatch(string(body)); len(match) == 2 {
		seconds, err := strconv.ParseFloat(match[1], 64)
		if err == nil && seconds > 0 {
			return seconds, nil
		}
	}

	return 0, ErrStreamNotFound
}

func fetchNewgroundsVideoJSON(pageURL, mediaID string) ([]byte, error) {
	apiURL := "https://www.newgrounds.com/portal/video/" + mediaID
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", streamUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", pageURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := streamHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("newgrounds video API returned status %d", resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
}

func fetchNewgroundsPage(pageURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}
	applySiteHeaders(req, SiteNewgrounds)

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

func parseNewgroundsFormatHeight(formatID string) int {
	formatID = strings.TrimSpace(strings.ToLower(formatID))
	formatID = strings.TrimSuffix(formatID, "p")
	if height, err := strconv.Atoi(formatID); err == nil && height > 0 {
		return height
	}
	return 0
}
