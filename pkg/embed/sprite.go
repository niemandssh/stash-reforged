package embed

import "fmt"

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
		site = SitePornHub
	}
	return StreamHTTPHeadersForSite(site)
}

// StreamHTTPHeadersForSite returns ffmpeg/ffprobe HTTP headers for a supported site.
func StreamHTTPHeadersForSite(site Site) string {
	referer, origin := siteReferer(site)
	return fmt.Sprintf(
		"Referer: %s\r\nOrigin: %s\r\nUser-Agent: %s",
		referer,
		origin,
		streamUserAgent,
	)
}
