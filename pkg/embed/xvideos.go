package embed

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var (
	xvideosSetVideoHLSRe     = regexp.MustCompile(`(?i)setVideoHLS\s*\(\s*'([^']+)'`)
	xvideosSetVideoURLPathRe = regexp.MustCompile(`setVideoURL\s*\(\s*'([^']+)'`)
	xvideosSetVideoUrlRe     = regexp.MustCompile(`(?i)setVideoUrl(High|Low)\s*\(\s*'([^']+)'`)
	xvideosOgDurationRe      = regexp.MustCompile(`property="og:duration"\s+content="(\d+)"`)
	xvideosDeletedRe         = regexp.MustCompile(`(?i)video has been deleted`)
)

// SupportsNativeStreaming reports whether a site supports native HLS playback in Stash.
func SupportsNativeStreaming(site Site) bool {
	switch site {
	case SitePornHub, SiteXVideos, SiteXnxx:
		return true
	default:
		return false
	}
}

// XVideosViewPageURL returns a watch page URL for an XVideos embed or page URL.
func XVideosViewPageURL(rawURL string) string {
	return html5PlayerViewPageURL(rawURL, SiteXVideos)
}

func html5PlayerViewPageURL(rawURL string, site Site) string {
	embedSrc, parsedSite, err := Parse(rawURL)
	if err != nil || parsedSite != site {
		return ""
	}

	switch site {
	case SiteXVideos:
		if m := xvideosEmbedRe.FindStringSubmatch(embedSrc); len(m) == 2 {
			return "https://www.xvideos.com/video." + m[1] + "/"
		}
		if strings.Contains(rawURL, "/video.") || strings.Contains(rawURL, "/video") {
			parsed := strings.TrimSpace(rawURL)
			if !strings.Contains(parsed, "://") {
				return "https://www.xvideos.com" + parsed
			}
			return parsed
		}
	case SiteXnxx:
		if m := xnxxEmbedRe.FindStringSubmatch(embedSrc); len(m) == 2 {
			return "https://www.xnxx.com/embedframe/" + m[1]
		}
	}

	return ""
}

func siteHomeURL(site Site) string {
	referer, _ := siteReferer(site)
	return strings.TrimRight(referer, "/")
}

func extractHTML5PlayerStreamVariants(embedSrc string, site Site) ([]HLSVariant, error) {
	body, err := fetchSitePage(embedSrc, site)
	if err != nil {
		return nil, err
	}

	html := string(body)
	if xvideosDeletedRe.MatchString(html) {
		return nil, fmt.Errorf("video has been deleted")
	}

	if hlsMatch := xvideosSetVideoHLSRe.FindStringSubmatch(html); len(hlsMatch) == 2 {
		masterURL := strings.ReplaceAll(hlsMatch[1], "\\/", "/")
		masterBody, _, err := fetchStreamResourceForSite(masterURL, site)
		if err != nil {
			return nil, err
		}

		variants, err := ParseMasterVariants(masterBody, masterURL)
		if err != nil {
			return nil, err
		}
		if len(variants) > 0 {
			return prepareVariants(variants), nil
		}
	}

	return extractHTML5PlayerMP4Variants(html)
}

func extractHTML5PlayerMP4Variants(html string) ([]HLSVariant, error) {
	byHeight := make(map[int]HLSVariant)

	for _, match := range xvideosSetVideoUrlRe.FindAllStringSubmatch(html, -1) {
		if len(match) != 3 {
			continue
		}

		mp4URL := strings.ReplaceAll(match[2], "\\/", "/")
		height := inferXVideosMP4Height(mp4URL)
		if height == 0 {
			if strings.EqualFold(match[1], "High") {
				height = 360
			} else {
				height = 240
			}
		}

		byHeight[height] = HLSVariant{
			Height: height,
			URL:    mp4URL,
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

func inferXVideosMP4Height(mp4URL string) int {
	lower := strings.ToLower(mp4URL)
	for _, h := range []int{2160, 1440, 1080, 720, 480, 360, 240} {
		if strings.Contains(lower, fmt.Sprintf("video_%dp", h)) ||
			strings.Contains(lower, fmt.Sprintf("_%dp", h)) {
			return h
		}
	}
	return inferHeightFromURL(mp4URL)
}

func extractHTML5PlayerDuration(embedSrc, rawURL string, site Site) (float64, error) {
	body, err := fetchSitePage(embedSrc, site)
	if err != nil {
		return 0, err
	}

	html := string(body)
	if pathMatch := xvideosSetVideoURLPathRe.FindStringSubmatch(html); len(pathMatch) == 2 {
		pageURL := pathMatch[1]
		if !strings.HasPrefix(pageURL, "http") {
			pageURL = siteHomeURL(site) + pathMatch[1]
		}
		if duration, err := fetchHTML5PlayerDurationFromPage(pageURL, site); err == nil && duration > 0 {
			return duration, nil
		}
	}

	if viewURL := html5PlayerViewPageURL(rawURL, site); viewURL != "" {
		if duration, err := fetchHTML5PlayerDurationFromPage(viewURL, site); err == nil && duration > 0 {
			return duration, nil
		}
	}

	return 0, ErrStreamNotFound
}

func fetchHTML5PlayerDurationFromPage(pageURL string, site Site) (float64, error) {
	body, err := fetchSitePage(pageURL, site)
	if err != nil {
		return 0, err
	}

	if match := xvideosOgDurationRe.FindStringSubmatch(string(body)); len(match) == 2 {
		seconds, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return 0, err
		}
		return seconds, nil
	}

	return 0, ErrStreamNotFound
}

func fetchSitePage(pageURL string, site Site) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}

	applySiteHeaders(req, site)

	resp, err := streamHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("page fetch returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, err
	}

	return body, nil
}

func fetchStreamResourceForSite(rawURL string, site Site) ([]byte, string, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}

	applySiteHeaders(req, site)

	resp, err := streamHTTPClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("upstream returned status %d", resp.StatusCode)
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

func siteReferer(site Site) (referer, origin string) {
	switch site {
	case SiteXVideos:
		return "https://www.xvideos.com/", "https://www.xvideos.com"
	case SiteXnxx:
		return "https://www.xnxx.com/", "https://www.xnxx.com"
	case SiteXHamster:
		return "https://xhamster.com/", "https://xhamster.com"
	default:
		return "https://www.pornhub.com/", "https://www.pornhub.com"
	}
}

func siteForProxyHost(host string) Site {
	host = strings.ToLower(host)
	switch {
	case strings.HasSuffix(host, "xvideos-cdn.com") || strings.Contains(host, "xvideos."):
		return SiteXVideos
	case strings.Contains(host, "xnxx."):
		return SiteXnxx
	case strings.Contains(host, "xhamster."):
		return SiteXHamster
	default:
		return SitePornHub
	}
}

func applySiteHeaders(req *http.Request, site Site) {
	referer, origin := siteReferer(site)
	req.Header.Set("User-Agent", streamUserAgent)
	req.Header.Set("Referer", referer)
	req.Header.Set("Origin", origin)
}
