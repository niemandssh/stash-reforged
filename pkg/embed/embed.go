package embed

import (
	"errors"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrUnsupportedSite = errors.New("unsupported embed site")
	ErrInvalidURL      = errors.New("invalid embed URL")
)

// Site identifies a supported video host.
type Site string

const (
	SitePornHub  Site = "pornhub"
	SiteXVideos  Site = "xvideos"
	SiteXHamster Site = "xhamster"
	SiteXnxx     Site = "xnxx"
)

var (
	pornhubEmbedRe = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*pornhub\.[a-z.]+/embed/([a-z0-9]+)`)

	xvideosPageDotRe = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*xvideos\.[a-z.]+/video\.([a-z0-9]+)`)
	xvideosPageNumRe = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*xvideos\.[a-z.]+/video(\d+)`)
	xvideosEmbedRe   = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*xvideos\.[a-z.]+/embedframe/([a-z0-9]+)`)

	xhamsterPageRe  = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*xhamster\.[a-z.]+/videos/[^/?#]+-(\d+)`)
	xhamsterEmbedRe = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*xhamster\.[a-z.]+/xembed\.php\?video=(\d+)`)

	xnxxPageRe  = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*xnxx\.[a-z.]+/video-([a-z0-9]+)`)
	xnxxEmbedRe = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*xnxx\.[a-z.]+/embedframe/([a-z0-9]+)`)
)

func normalizeHost(host string) string {
	host = strings.ToLower(host)
	for {
		trimmed := strings.TrimPrefix(host, "www.")
		trimmed = strings.TrimPrefix(trimmed, "m.")
		if trimmed == host {
			break
		}
		host = trimmed
	}
	return host
}

func isPornHubHost(host string) bool {
	return strings.HasPrefix(host, "pornhub.") || host == "pornhub.com" || host == "pornhub.org"
}

func isXVideosHost(host string) bool {
	return strings.HasPrefix(host, "xvideos.") || host == "xvideos.com"
}

func isXHamsterHost(host string) bool {
	return strings.Contains(host, "xhamster")
}

func isXnxxHost(host string) bool {
	return strings.HasPrefix(host, "xnxx.") || host == "xnxx.com"
}

// Parse converts a page or embed URL into an iframe src URL and site name.
func Parse(rawURL string) (iframeSrc string, site Site, err error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", "", ErrInvalidURL
	}

	normalized := rawURL
	if !strings.Contains(normalized, "://") {
		normalized = "https://" + normalized
	}

	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", "", ErrInvalidURL
	}

	host := normalizeHost(parsed.Hostname())

	switch {
	case isPornHubHost(host):
		return parsePornHub(normalized)
	case isXVideosHost(host):
		return parseXVideos(normalized)
	case isXHamsterHost(host):
		return parseXHamster(normalized)
	case isXnxxHost(host):
		return parseXnxx(normalized)
	default:
		return "", "", ErrUnsupportedSite
	}
}

// PornHubViewPageURL returns the watch page URL for a PornHub embed or view URL.
func PornHubViewPageURL(embedOrViewURL string) string {
	if m := pornhubEmbedRe.FindStringSubmatch(embedOrViewURL); len(m) == 2 {
		return "https://www.pornhub.com/view_video.php?viewkey=" + m[1]
	}

	parsed, err := url.Parse(embedOrViewURL)
	if err != nil {
		return ""
	}
	if !strings.Contains(strings.ToLower(parsed.Hostname()), "pornhub") {
		return ""
	}
	if viewKey := parsed.Query().Get("viewkey"); viewKey != "" {
		return "https://www.pornhub.com/view_video.php?viewkey=" + viewKey
	}

	return ""
}

func parsePornHub(raw string) (string, Site, error) {
	if m := pornhubEmbedRe.FindStringSubmatch(raw); len(m) == 2 {
		return "https://www.pornhub.com/embed/" + m[1], SitePornHub, nil
	}

	parsed, err := url.Parse(raw)
	if err == nil && strings.Contains(strings.ToLower(parsed.Host), "pornhub") {
		if viewKey := parsed.Query().Get("viewkey"); viewKey != "" {
			return "https://www.pornhub.com/embed/" + viewKey, SitePornHub, nil
		}
	}

	return "", SitePornHub, ErrInvalidURL
}

func parseXVideos(raw string) (string, Site, error) {
	if m := xvideosEmbedRe.FindStringSubmatch(raw); len(m) == 2 {
		return "https://www.xvideos.com/embedframe/" + m[1], SiteXVideos, nil
	}
	if m := xvideosPageDotRe.FindStringSubmatch(raw); len(m) == 2 {
		return "https://www.xvideos.com/embedframe/" + m[1], SiteXVideos, nil
	}
	if m := xvideosPageNumRe.FindStringSubmatch(raw); len(m) == 2 {
		return "https://www.xvideos.com/embedframe/" + m[1], SiteXVideos, nil
	}
	return "", SiteXVideos, ErrInvalidURL
}

func parseXHamster(raw string) (string, Site, error) {
	if m := xhamsterEmbedRe.FindStringSubmatch(raw); len(m) == 2 {
		return "https://xhamster.com/xembed.php?video=" + m[1], SiteXHamster, nil
	}
	if m := xhamsterPageRe.FindStringSubmatch(raw); len(m) == 2 {
		return "https://xhamster.com/xembed.php?video=" + m[1], SiteXHamster, nil
	}
	return "", SiteXHamster, ErrInvalidURL
}

func parseXnxx(raw string) (string, Site, error) {
	if m := xnxxEmbedRe.FindStringSubmatch(raw); len(m) == 2 {
		return "https://www.xnxx.com/embedframe/" + m[1], SiteXnxx, nil
	}
	if m := xnxxPageRe.FindStringSubmatch(raw); len(m) == 2 {
		return "https://www.xnxx.com/embedframe/" + m[1], SiteXnxx, nil
	}
	return "", SiteXnxx, ErrInvalidURL
}

// Supported returns true if the URL belongs to a supported host.
func Supported(rawURL string) bool {
	_, _, err := Parse(rawURL)
	return err == nil
}

// ApplyStartTime appends a playback start offset to an embed iframe URL when supported.
func ApplyStartTime(src string, site Site, startSeconds float64) string {
	if startSeconds <= 0 {
		return src
	}

	parsed, err := url.Parse(src)
	if err != nil {
		return src
	}

	start := int(math.Round(startSeconds))
	q := parsed.Query()

	switch site {
	case SitePornHub:
		// PornHub page URLs use ?t= (seconds, may be fractional). Embed iframes ignore it,
		// but keep the parameter for hosts that honor it.
		q.Set("t", formatPornHubStartTime(startSeconds))
	case SiteXVideos, SiteXHamster, SiteXnxx:
		q.Set("t", strconv.Itoa(start))
	}

	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func formatPornHubStartTime(startSeconds float64) string {
	if startSeconds == math.Trunc(startSeconds) {
		return strconv.Itoa(int(startSeconds))
	}
	return strconv.FormatFloat(startSeconds, 'f', -1, 64)
}
