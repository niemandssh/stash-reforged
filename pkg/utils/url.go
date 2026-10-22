package utils

import (
	"regexp"
	"strings"
)

var imageURLPrefixRE = regexp.MustCompile(`(?i)^https?://[^\s'"\x00-\x1f<>]+`)

// SanitizeImageURL extracts a usable HTTP(S) URL from scraped image input.
// Scrapers occasionally return script fragments around the URL; Go's url parser
// rejects control characters such as tabs in those strings.
func SanitizeImageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "data:") {
		return raw
	}

	if m := imageURLPrefixRE.FindString(raw); m != "" {
		return m
	}

	return raw
}

// URLFromHandle adds the site URL to the input if the input is not already a URL
// siteURL must not end with a slash
func URLFromHandle(input string, siteURL string) string {
	// if the input is already a URL, return it
	re := regexp.MustCompile(`^https?://`)
	if re.MatchString(input) {
		return input
	}

	return siteURL + "/" + input
}
