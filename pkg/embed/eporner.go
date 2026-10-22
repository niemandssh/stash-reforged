package embed

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const SiteEporner Site = "eporner"

var (
	epornerPageRe = regexp.MustCompile(`(?i)^(?:https?://)?(?:[\w-]+\.)*eporner\.com/(?:video-|embed/|hd-porn/)([a-z0-9]+)`)
	epornerVidRe  = regexp.MustCompile(`vid\s*=\s*'([a-zA-Z0-9]+)'`)
	epornerHashRe = regexp.MustCompile(`hash\s*=\s*'([0-9a-f]{32})'`)
)

type epornerSource struct {
	LabelShort string `json:"labelShort"`
	Src        string `json:"src"`
}

type epornerResponse struct {
	Duration float64 `json:"duration"`
	Sources  struct {
		MP4 map[string]epornerSource `json:"mp4"`
	} `json:"sources"`
}

func isEpornerHost(host string) bool {
	return host == "eporner.com" || strings.HasSuffix(host, ".eporner.com")
}

func parseEpornerPage(raw string) (string, Site, error) {
	if !epornerPageRe.MatchString(raw) {
		return "", SiteEporner, ErrInvalidURL
	}
	// No iframe -- playback resolves the stream via the xhr API at request time.
	return "", SiteEporner, nil
}

// epornerHashParam converts the page's 32-char hex hash into the base36 chunks the xhr API expects.
func epornerHashParam(hash string) string {
	var sb strings.Builder
	for i := 0; i+8 <= len(hash); i += 8 {
		n, _ := strconv.ParseUint(hash[i:i+8], 16, 64)
		sb.WriteString(strconv.FormatUint(n, 36))
	}
	return sb.String()
}

func fetchEpornerVideo(rawURL string) (*epornerResponse, error) {
	m := epornerPageRe.FindStringSubmatch(rawURL)
	if len(m) != 2 {
		return nil, ErrInvalidURL
	}

	page, err := fetchSitePage("https://www.eporner.com/video-"+m[1]+"/", SiteEporner)
	if err != nil {
		return nil, err
	}
	vid := epornerVidRe.FindSubmatch(page)
	hash := epornerHashRe.FindSubmatch(page)
	if vid == nil || hash == nil {
		return nil, ErrStreamNotFound
	}

	q := url.Values{}
	q.Set("hash", epornerHashParam(string(hash[1])))
	q.Set("domain", "www.eporner.com")
	q.Set("fallback", "false")
	q.Set("embed", "false")
	q.Set("supportedFormats", "dash,mp4")
	body, err := fetchSitePage("https://www.eporner.com/xhr/video/"+string(vid[1])+"?"+q.Encode(), SiteEporner)
	if err != nil {
		return nil, err
	}

	var resp epornerResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse eporner response: %w", err)
	}
	return &resp, nil
}

func extractEpornerStreamVariants(rawURL string) ([]HLSVariant, error) {
	resp, err := fetchEpornerVideo(rawURL)
	if err != nil {
		return nil, err
	}

	var variants []HLSVariant
	for _, src := range resp.Sources.MP4 {
		if src.Src == "" {
			continue
		}
		height := inferHeightFromURL(src.Src)
		if height == 0 {
			height, _ = strconv.Atoi(strings.TrimSuffix(src.LabelShort, "p"))
		}
		variants = append(variants, HLSVariant{URL: src.Src, Height: height})
	}
	if len(variants) == 0 {
		return nil, ErrStreamNotFound
	}
	return prepareVariants(variants), nil
}

func extractEpornerDuration(rawURL string) (float64, error) {
	resp, err := fetchEpornerVideo(rawURL)
	if err != nil {
		return 0, err
	}
	if resp.Duration > 0 {
		return resp.Duration, nil
	}
	return 0, ErrStreamNotFound
}
