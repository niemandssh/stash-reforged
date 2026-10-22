package embed

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

const SiteTNAFlix Site = "tnaflix"

var (
	tnaflixPageRe     = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*tnaflix\.com/.*video(\d+)`)
	tnaflixSourceRe   = regexp.MustCompile(`<source\s+src="([^"]+)"[^>]*\bsize="(\d+)"`)
	tnaflixDurationRe = regexp.MustCompile(`"duration"\s*:\s*(\d+)`)
)

func isTNAFlixHost(host string) bool {
	return host == "tnaflix.com" || strings.HasSuffix(host, ".tnaflix.com")
}

func parseTNAFlixPage(raw string) (string, Site, error) {
	if !tnaflixPageRe.MatchString(raw) {
		return "", SiteTNAFlix, ErrInvalidURL
	}
	// No iframe -- playback resolves the stream from the watch page at request time.
	return "", SiteTNAFlix, nil
}

func fetchTNAFlixPage(rawURL string) (string, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	body, err := fetchSitePage(rawURL, SiteTNAFlix)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func extractTNAFlixStreamVariants(rawURL string) ([]HLSVariant, error) {
	page, err := fetchTNAFlixPage(rawURL)
	if err != nil {
		return nil, err
	}

	var variants []HLSVariant
	for _, m := range tnaflixSourceRe.FindAllStringSubmatch(page, -1) {
		height, _ := strconv.Atoi(m[2])
		variants = append(variants, HLSVariant{URL: html.UnescapeString(m[1]), Height: height})
	}
	if len(variants) == 0 {
		return nil, ErrStreamNotFound
	}
	return prepareVariants(variants), nil
}

func extractTNAFlixDuration(rawURL string) (float64, error) {
	page, err := fetchTNAFlixPage(rawURL)
	if err != nil {
		return 0, err
	}
	m := tnaflixDurationRe.FindStringSubmatch(page)
	if len(m) != 2 {
		return 0, ErrStreamNotFound
	}
	return strconv.ParseFloat(m[1], 64)
}
