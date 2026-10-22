package embed

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// SiteVJav is a vjav.com (and mirror/reposting hosts such as videovjav.com) watch or
// embed page backed by the VJAV JW Player API.
const SiteVJav Site = "vjav"

var (
	vjavVideoIDRe  = regexp.MustCompile(`(?i)/(?:videos|embed)/(\d+)`)
	vjavDurationRe = regexp.MustCompile(`"duration"\s*:\s*"(\d+):(\d+)(?::(\d+))?"`)
)

// vjavBase164Replacer undoes the site's custom base64 alphabet, which swaps a handful
// of Latin letters for Cyrillic homoglyphs and uses "." "," "~" instead of "+" "/" "=".
var vjavBase164Replacer = strings.NewReplacer(
	"А", "A", "В", "B", "С", "C", "Е", "E", "М", "M",
	".", "+", ",", "/", "~", "=",
)

func isVJavHost(host string) bool {
	return strings.HasSuffix(host, "vjav.com") ||
		strings.HasSuffix(host, "vjav.tube") ||
		strings.HasSuffix(host, "vjav.us") ||
		strings.HasSuffix(host, "videovjav.com")
}

func vjavVideoID(rawURL string) string {
	m := vjavVideoIDRe.FindStringSubmatch(rawURL)
	if len(m) != 2 {
		return ""
	}
	return m[1]
}

func parseVJavPage(raw string) (string, Site, error) {
	if vjavVideoID(raw) == "" {
		return "", SiteVJav, ErrInvalidURL
	}
	// No iframe -- playback resolves the stream via the VJAV API at request time.
	return "", SiteVJav, nil
}

func vjavBase164Decode(encoded string) (string, error) {
	s := vjavBase164Replacer.Replace(strings.TrimSpace(encoded))
	if pad := len(s) % 4; pad != 0 {
		s += strings.Repeat("=", 4-pad)
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

type vjavFileEntry struct {
	Format   string `json:"format"`
	VideoURL string `json:"video_url"`
}

func extractVJavStreamVariants(rawURL string) ([]HLSVariant, error) {
	host, videoID, err := vjavHostAndID(rawURL)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://%s/api/videofile.php?video_id=%s&lifetime=8640000", host, videoID)
	body, err := fetchVJavResource(apiURL, rawURL)
	if err != nil {
		return nil, err
	}

	var entries []vjavFileEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parse vjav videofile response: %w", err)
	}

	var variants []HLSVariant
	seen := make(map[string]struct{})
	for _, entry := range entries {
		if entry.VideoURL == "" {
			continue
		}
		decodedPath, err := vjavBase164Decode(entry.VideoURL)
		if err != nil || !strings.HasPrefix(decodedPath, "/get_file/") {
			continue
		}

		finalURL, err := resolveVJavFinalURL("https://"+host+decodedPath, rawURL)
		if err != nil {
			continue
		}
		if _, ok := seen[finalURL]; ok {
			continue
		}
		seen[finalURL] = struct{}{}

		variants = append(variants, HLSVariant{
			URL:    finalURL,
			Height: vjavInferHeight(entry.Format, finalURL),
		})
	}

	if len(variants) == 0 {
		return nil, ErrStreamNotFound
	}
	return prepareVariants(variants), nil
}

func vjavInferHeight(format, mediaURL string) int {
	if height := inferHeightFromURL(mediaURL); height > 0 {
		return height
	}
	switch strings.ToLower(strings.TrimSuffix(format, ".mp4")) {
	case "_hd":
		return 1080
	case "_sd":
		return 480
	case "_lq":
		return 360
	default:
		return 720
	}
}

func resolveVJavFinalURL(getFileURL, refererURL string) (string, error) {
	req, err := http.NewRequest(http.MethodHead, getFileURL, nil)
	if err != nil {
		return "", err
	}
	applyVJavHeaders(req, refererURL)

	resp, err := streamHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("vjav get_file returned status %d", resp.StatusCode)
	}
	return resp.Request.URL.String(), nil
}

func extractVJavDuration(rawURL string) (float64, error) {
	host, videoID, err := vjavHostAndID(rawURL)
	if err != nil {
		return 0, err
	}

	id, err := strconv.Atoi(videoID)
	if err != nil {
		return 0, ErrInvalidURL
	}
	shardMillion := (id / 1_000_000) * 1_000_000
	shardThousand := (id / 1_000) * 1_000

	apiURL := fmt.Sprintf("https://%s/api/json/video/86400/%d/%d/%s.json", host, shardMillion, shardThousand, videoID)
	body, err := fetchVJavResource(apiURL, rawURL)
	if err != nil {
		return 0, err
	}

	m := vjavDurationRe.FindSubmatch(body)
	if len(m) < 3 {
		return 0, ErrStreamNotFound
	}
	return parseVJavDuration(m)
}

func parseVJavDuration(m [][]byte) (float64, error) {
	var parts []int
	for _, p := range m[1:] {
		if len(p) == 0 {
			continue
		}
		v, err := strconv.Atoi(string(p))
		if err != nil {
			return 0, err
		}
		parts = append(parts, v)
	}

	var seconds int
	switch len(parts) {
	case 2:
		seconds = parts[0]*60 + parts[1]
	case 3:
		seconds = parts[0]*3600 + parts[1]*60 + parts[2]
	default:
		return 0, ErrStreamNotFound
	}
	return float64(seconds), nil
}

func vjavHostAndID(rawURL string) (host, videoID string, err error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", "", ErrInvalidURL
	}
	host = normalizeHost(parsed.Hostname())
	videoID = vjavVideoID(rawURL)
	if videoID == "" {
		return "", "", ErrInvalidURL
	}
	return host, videoID, nil
}

func fetchVJavResource(apiURL, refererURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	applyVJavHeaders(req, refererURL)

	resp, err := streamHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vjav api returned status %d", resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
}

func applyVJavHeaders(req *http.Request, refererURL string) {
	req.Header.Set("User-Agent", streamUserAgent)
	if refererURL != "" {
		req.Header.Set("Referer", refererURL)
	}
}

func extractCachedVJavVariants(rawURL string) ([]HLSVariant, error) {
	cacheKey := "vjav:" + strings.TrimSpace(rawURL)
	if variants, ok := getCachedVariants(cacheKey); ok {
		return variants, nil
	}

	result, err, _ := remotePageVariantGroup.Do(cacheKey, func() (interface{}, error) {
		if variants, ok := getCachedVariants(cacheKey); ok {
			return variants, nil
		}

		variants, err := extractVJavStreamVariants(rawURL)
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
