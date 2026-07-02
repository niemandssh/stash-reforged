package embed

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DefaultPreferredHeight is the target playback height for remote HLS streams.
const DefaultPreferredHeight = 1080

// StandardStreamHeights lists typical remote HLS quality options for the player UI.
// Actual availability is resolved when the manifest is requested.
var StandardStreamHeights = []int{1080, 720, 480, 240}

var extXKeyURIRe = regexp.MustCompile(`URI="([^"]+)"`)
var resolutionRe = regexp.MustCompile(`(?i)RESOLUTION="?(\d+)x(\d+)"?`)
var bandwidthRe = regexp.MustCompile(`(?i)BANDWIDTH=(\d+)`)

// HLSVariant represents a single quality level from an HLS master playlist.
type HLSVariant struct {
	Height    int
	Width     int
	Bandwidth int
	URL       string
}

type streamInf struct {
	height    int
	width     int
	bandwidth int
}

var streamHTTPClient = &http.Client{Timeout: 30 * time.Second}

const streamUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// IsAllowedProxyURL restricts embed stream proxying to known video CDNs.
func IsAllowedProxyURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	return strings.HasSuffix(host, "phncdn.com") ||
		strings.Contains(host, "pornhub.") ||
		strings.HasSuffix(host, "xvideos-cdn.com") ||
		strings.Contains(host, "xhamster.") ||
		strings.Contains(host, "xnxx.")
}

// FetchStreamResource downloads a remote stream resource using browser-like headers.
func FetchStreamResource(rawURL string) ([]byte, string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", err
	}

	return fetchStreamResourceForSite(rawURL, siteForProxyHost(parsed.Hostname()))
}

func IsM3U8Content(contentType, rawURL string) bool {
	lower := strings.ToLower(contentType)
	if strings.Contains(lower, "mpegurl") || strings.Contains(lower, "m3u8") {
		return true
	}

	path := strings.ToLower(rawURL)
	return strings.Contains(path, ".m3u8")
}

// RewriteM3U8 rewrites playlist URIs to go through a same-origin proxy path.
func RewriteM3U8(body []byte, manifestURL, proxyPath string) ([]byte, error) {
	base, err := url.Parse(manifestURL)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		rewritten, err := rewriteM3U8Line(line, base, proxyPath)
		if err != nil {
			return nil, err
		}
		out.WriteString(rewritten)
		out.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// ParseMasterVariants extracts quality variants from an HLS master playlist.
// Returns nil when the playlist is a media playlist (no variants).
func ParseMasterVariants(body []byte, manifestURL string) ([]HLSVariant, error) {
	base, err := url.Parse(manifestURL)
	if err != nil {
		return nil, err
	}

	var variants []HLSVariant
	var pending *streamInf

	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			inf := &streamInf{}
			if m := resolutionRe.FindStringSubmatch(line); len(m) == 3 {
				inf.width, _ = strconv.Atoi(m[1])
				inf.height, _ = strconv.Atoi(m[2])
			}
			if m := bandwidthRe.FindStringSubmatch(line); len(m) == 2 {
				inf.bandwidth, _ = strconv.Atoi(m[1])
			}
			pending = inf
			continue
		}

		if pending != nil && !strings.HasPrefix(line, "#") {
			refURL, err := url.Parse(line)
			if err != nil {
				return nil, err
			}
			absolute := base.ResolveReference(refURL).String()
			variants = append(variants, HLSVariant{
				Height:    pending.height,
				Width:     pending.width,
				Bandwidth: pending.bandwidth,
				URL:       absolute,
			})
			pending = nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return variants, nil
}

// IsMasterPlaylist reports whether the playlist is an HLS master (variant) playlist.
func IsMasterPlaylist(body []byte) bool {
	return bytes.Contains(bytes.ToUpper(body), []byte("#EXT-X-STREAM-INF"))
}

// IsMediaPlaylist reports whether the playlist is an HLS media playlist.
func IsMediaPlaylist(body []byte) bool {
	return bytes.Contains(body, []byte("#EXTINF"))
}

func inferHeightFromURL(rawURL string) int {
	path := strings.ToLower(rawURL)
	for _, h := range []int{2160, 1440, 1080, 720, 480, 360, 240, 144} {
		patterns := []string{
			fmt.Sprintf("%dp", h),
			fmt.Sprintf("%dp_", h),
			fmt.Sprintf("/%d/", h),
			fmt.Sprintf("_%d_", h),
			fmt.Sprintf("-%d-", h),
		}
		for _, p := range patterns {
			if strings.Contains(path, p) {
				return h
			}
		}
	}
	return 0
}

func prepareVariants(variants []HLSVariant) []HLSVariant {
	if len(variants) == 0 {
		return variants
	}

	prepared := append([]HLSVariant(nil), variants...)
	for i := range prepared {
		if prepared[i].Height == 0 {
			prepared[i].Height = inferHeightFromURL(prepared[i].URL)
		}
	}
	return prepared
}

func bandwidthTierIndex(height int, count int) int {
	if count <= 1 {
		return 0
	}

	var fraction float64
	switch {
	case height <= 240:
		fraction = 0
	case height <= 480:
		fraction = 0.33
	case height <= 720:
		fraction = 0.66
	default:
		fraction = 1
	}

	idx := int(fraction*float64(count-1) + 0.5)
	if idx < 0 {
		return 0
	}
	if idx >= count {
		return count - 1
	}
	return idx
}

func selectByBandwidthTier(variants []HLSVariant, height int) *HLSVariant {
	sorted := append([]HLSVariant(nil), variants...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Bandwidth != sorted[j].Bandwidth {
			return sorted[i].Bandwidth < sorted[j].Bandwidth
		}
		return sorted[i].Height < sorted[j].Height
	})

	return &sorted[bandwidthTierIndex(height, len(sorted))]
}

// SortVariantsDefaultFirst orders variants for playback: preferred height first,
// then descending by height, then by bandwidth.
func SortVariantsDefaultFirst(variants []HLSVariant, preferredHeight int) []HLSVariant {
	if len(variants) <= 1 {
		return variants
	}

	sorted := append([]HLSVariant(nil), variants...)
	sort.SliceStable(sorted, func(i, j int) bool {
		pi := variantSortPriority(sorted[i], preferredHeight)
		pj := variantSortPriority(sorted[j], preferredHeight)
		if pi != pj {
			return pi < pj
		}
		if sorted[i].Height != sorted[j].Height {
			return sorted[i].Height > sorted[j].Height
		}
		return sorted[i].Bandwidth > sorted[j].Bandwidth
	})

	return sorted
}

func variantSortPriority(v HLSVariant, preferredHeight int) int {
	if v.Height == preferredHeight {
		return 0
	}
	if v.Height > 0 && v.Height < preferredHeight {
		return 1 + (preferredHeight - v.Height)
	}
	if v.Height > preferredHeight {
		return 1000 + (v.Height - preferredHeight)
	}
	return 5000
}

// SelectVariant picks a variant for the requested height. height <= 0 uses preferredHeight.
func SelectVariant(variants []HLSVariant, height int) (*HLSVariant, error) {
	if len(variants) == 0 {
		return nil, ErrStreamNotFound
	}

	prepared := prepareVariants(variants)

	if height <= 0 {
		height = DefaultPreferredHeight
	}

	// Exact match
	for i := range prepared {
		if prepared[i].Height == height {
			return &prepared[i], nil
		}
	}

	// Closest height at or below requested
	var bestBelow *HLSVariant
	for i := range prepared {
		v := &prepared[i]
		if v.Height > 0 && v.Height <= height {
			if bestBelow == nil || v.Height > bestBelow.Height {
				bestBelow = v
			}
		}
	}
	if bestBelow != nil {
		return bestBelow, nil
	}

	// Closest above requested
	var bestAbove *HLSVariant
	for i := range prepared {
		v := &prepared[i]
		if v.Height > height {
			if bestAbove == nil || v.Height < bestAbove.Height {
				bestAbove = v
			}
		}
	}
	if bestAbove != nil {
		return bestAbove, nil
	}

	if len(prepared) > 1 {
		return selectByBandwidthTier(prepared, height), nil
	}

	return &prepared[0], nil
}

// HeightsFromVariants returns the sorted unique heights present in variants (descending).
func HeightsFromVariants(variants []HLSVariant) []int {
	if len(variants) == 0 {
		return nil
	}

	seen := make(map[int]struct{}, len(variants))
	for _, v := range variants {
		if v.Height > 0 {
			seen[v.Height] = struct{}{}
		}
	}

	heights := make([]int, 0, len(seen))
	for height := range seen {
		heights = append(heights, height)
	}

	sort.Slice(heights, func(i, j int) bool {
		return heights[i] > heights[j]
	})

	return heights
}

// VariantLabel returns a human-readable quality label.
func VariantLabel(v HLSVariant) string {
	if v.Height > 0 {
		return fmt.Sprintf("%dp", v.Height)
	}
	if v.Bandwidth > 0 {
		return fmt.Sprintf("%dk", v.Bandwidth/1000)
	}
	return "HLS"
}

func rewriteM3U8Line(line string, base *url.URL, proxyPath string) (string, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line, nil
	}

	if strings.HasPrefix(trimmed, "#") {
		if strings.Contains(trimmed, `URI="`) {
			return extXKeyURIRe.ReplaceAllStringFunc(trimmed, func(match string) string {
				parts := extXKeyURIRe.FindStringSubmatch(match)
				if len(parts) != 2 {
					return match
				}
				proxied, err := proxyURL(base, parts[1], proxyPath)
				if err != nil {
					return match
				}
				return `URI="` + proxied + `"`
			}), nil
		}
		return line, nil
	}

	return proxyURL(base, trimmed, proxyPath)
}

func proxyURL(base *url.URL, ref, proxyPath string) (string, error) {
	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}

	absolute := base.ResolveReference(refURL).String()
	if !IsAllowedProxyURL(absolute) {
		return "", fmt.Errorf("upstream URL not allowed: %s", absolute)
	}

	return proxyPath + "?u=" + url.QueryEscape(absolute), nil
}
