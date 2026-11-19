package embed

import (
	"fmt"
	"net/url"
	"strings"
)

// SupportsSpriteGeneration reports whether full sprite generation is available for the embed URL.
func SupportsSpriteGeneration(rawURL string) bool {
	_, site, err := Parse(rawURL)
	return err == nil && SupportsNativeStreaming(site)
}

// StreamHTTPHeaders returns ffmpeg/ffprobe HTTP headers for remote stream access.
func StreamHTTPHeaders() string {
	return StreamHTTPHeadersForSite(SitePornHub)
}

// StreamHTTPHeadersForURL returns ffmpeg/ffprobe HTTP headers for a remote embed URL.
func StreamHTTPHeadersForURL(rawURL string) string {
	_, site, err := Parse(rawURL)
	if err != nil {
		site = SiteDirect
	}
	if site == SitePlayerJS || site == SiteJWPlayer {
		return pageStreamHTTPHeaders(rawURL)
	}
	return StreamHTTPHeadersForSite(site)
}

// StreamHTTPHeadersForMediaURL returns headers for fetching a resolved media URL.
// When the media is hosted on a different domain than the embed page, only a
// User-Agent is sent so CDN auth tokens are not broken by a foreign Referer.
func StreamHTTPHeadersForMediaURL(embedURL, mediaURL string) string {
	embedURL = strings.TrimSpace(embedURL)
	mediaURL = strings.TrimSpace(mediaURL)
	if mediaURL == "" {
		return StreamHTTPHeadersForURL(embedURL)
	}

	if _, site, err := Parse(embedURL); err == nil {
		if referer, _ := siteReferer(site); referer != "" {
			return StreamHTTPHeadersForSite(site)
		}
	}

	embedHost := hostOf(embedURL)
	mediaHost := hostOf(mediaURL)
	if embedHost != "" && mediaHost != "" && !hostsEqualOrSubdomain(mediaHost, embedHost) {
		return fmt.Sprintf("User-Agent: %s", streamUserAgent)
	}
	return StreamHTTPHeadersForURL(embedURL)
}

// LooksLikeTemporaryMediaURL reports whether a media URL is likely short-lived.
func LooksLikeTemporaryMediaURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	q := strings.ToLower(parsed.RawQuery)
	if strings.Contains(q, "authorization=") ||
		strings.Contains(q, "x-amz-") ||
		strings.Contains(q, "signature=") ||
		strings.Contains(q, "expires=") ||
		strings.Contains(q, "secure=") ||
		strings.Contains(q, "token=") {
		return true
	}
	host := strings.ToLower(parsed.Hostname())
	if strings.Contains(host, "backblazeb2.com") || strings.Contains(host, "b2.cloud") {
		return true
	}
	pathLower := strings.ToLower(parsed.Path)
	return strings.Contains(pathLower, "/get_file/")
}

func hostOf(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

func hostsEqualOrSubdomain(host, parent string) bool {
	host = strings.TrimPrefix(host, "www.")
	parent = strings.TrimPrefix(parent, "www.")
	return host == parent || strings.HasSuffix(host, "."+parent)
}

func pageStreamHTTPHeaders(pageURL string) string {
	referer, origin := pageRefererOrigin(pageURL)
	var b strings.Builder
	if referer != "" {
		fmt.Fprintf(&b, "Referer: %s\r\n", referer)
	}
	if origin != "" {
		fmt.Fprintf(&b, "Origin: %s\r\n", origin)
	}
	fmt.Fprintf(&b, "User-Agent: %s", streamUserAgent)
	return b.String()
}

// StreamHTTPHeadersForSite returns ffmpeg/ffprobe HTTP headers for a supported site.
func StreamHTTPHeadersForSite(site Site) string {
	referer, origin := siteReferer(site)
	var b strings.Builder
	if referer != "" {
		fmt.Fprintf(&b, "Referer: %s\r\n", referer)
	}
	if origin != "" {
		fmt.Fprintf(&b, "Origin: %s\r\n", origin)
	}
	fmt.Fprintf(&b, "User-Agent: %s", streamUserAgent)
	return b.String()
}
