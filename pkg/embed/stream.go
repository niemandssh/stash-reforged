package embed

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

var (
	ErrStreamNotSupported = errors.New("stream extraction not supported for this site")
	ErrStreamNotFound     = errors.New("stream URL not found")
)

type mediaDefinition struct {
	Format   string          `json:"format"`
	VideoURL string          `json:"videoUrl"`
	Quality  json.RawMessage `json:"quality"`
	Height   int             `json:"height"`
}

var urlsetVideoIDRe = regexp.MustCompile(`_(\d+)\.mp4\.urlset`)
var flashvarsAssignRe = regexp.MustCompile(`var\s+flashvars(?:_\d+)?\s*=\s*\{`)

const flashvarsVariantCacheTTL = 45 * time.Second

type cachedVariantsEntry struct {
	variants []HLSVariant
	expiry   time.Time
}

var (
	flashvarsVariantCacheMu sync.RWMutex
	flashvarsVariantCache   = map[string]cachedVariantsEntry{}
	flashvarsVariantGroup   singleflight.Group
	remotePageVariantGroup  singleflight.Group
)

type flashvarsPayload struct {
	MediaDefinitions []mediaDefinition `json:"mediaDefinitions"`
	VideoDuration    float64           `json:"video_duration"`
}

// ExtractStreamURL resolves the default-quality HLS media playlist URL for supported remote scenes.
func ExtractStreamURL(rawURL string) (string, Site, error) {
	variants, site, err := ExtractStreamVariants(rawURL)
	if err != nil {
		return "", site, err
	}

	selected, err := SelectVariant(variants, DefaultPreferredHeight)
	if err != nil {
		return "", site, err
	}

	return selected.URL, site, nil
}

// ExtractStreamVariants returns available HLS quality variants for supported remote scenes.
func ExtractStreamVariants(rawURL string) ([]HLSVariant, Site, error) {
	embedSrc, site, err := Parse(rawURL)
	if err != nil {
		return nil, site, err
	}

	switch site {
	case SitePornHub:
		variants, err := extractPornHubStreamVariants(rawURL, embedSrc)
		if err != nil {
			return nil, SitePornHub, err
		}
		return variants, SitePornHub, nil
	case SiteXVideos, SiteXnxx:
		variants, err := extractHTML5PlayerStreamVariants(embedSrc, site)
		if err != nil {
			return nil, site, err
		}
		return variants, site, nil
	case SiteXHamster:
		variants, err := extractXHamsterStreamVariants(rawURL, embedSrc)
		if err != nil {
			return nil, SiteXHamster, err
		}
		return variants, SiteXHamster, nil
	case SiteDirect:
		mediaURL, err := normalizeDirectMediaURL(rawURL)
		if err != nil {
			return nil, SiteDirect, err
		}
		return []HLSVariant{{
			URL:    mediaURL,
			Height: inferHeightFromURL(mediaURL),
		}}, SiteDirect, nil
	case SitePlayerJS:
		variants, site, err := extractCachedRemotePageVariants(rawURL)
		if err != nil {
			return nil, site, err
		}
		return variants, site, nil
	case SiteKVS:
		variants, site, err := extractCachedRemotePageVariants(rawURL)
		if err != nil {
			return nil, site, err
		}
		return variants, site, nil
	case SiteVKVideo:
		variants, err := extractVKVideoStreamVariants(rawURL, embedSrc)
		if err != nil {
			return nil, SiteVKVideo, err
		}
		return variants, SiteVKVideo, nil
	case SiteJWPlayer:
		variants, err := extractCachedJWPlayerVariants(rawURL)
		if err != nil {
			return nil, SiteJWPlayer, err
		}
		return variants, SiteJWPlayer, nil
	case SiteNewgrounds:
		variants, err := extractNewgroundsStreamVariants(rawURL)
		if err != nil {
			return nil, SiteNewgrounds, err
		}
		return variants, SiteNewgrounds, nil
	case SiteVJav:
		variants, err := extractCachedVJavVariants(rawURL)
		if err != nil {
			return nil, SiteVJav, err
		}
		return variants, SiteVJav, nil
	case SiteEporner:
		variants, err := extractEpornerStreamVariants(rawURL)
		if err != nil {
			return nil, SiteEporner, err
		}
		return variants, SiteEporner, nil
	case SiteTNAFlix:
		variants, err := extractTNAFlixStreamVariants(rawURL)
		if err != nil {
			return nil, SiteTNAFlix, err
		}
		return variants, SiteTNAFlix, nil
	default:
		return nil, site, ErrStreamNotSupported
	}
}

func extractPornHubStreamVariants(rawURL, embedSrc string) ([]HLSVariant, error) {
	cacheKey := embedSrc

	if variants, ok := getCachedVariants(cacheKey); ok {
		return variants, nil
	}

	result, err, _ := flashvarsVariantGroup.Do(cacheKey, func() (interface{}, error) {
		if variants, ok := getCachedVariants(cacheKey); ok {
			return variants, nil
		}

		variants, err := extractPornHubStreamVariantsUncached(rawURL, embedSrc)
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

func extractPornHubStreamVariantsUncached(rawURL, embedSrc string) ([]HLSVariant, error) {
	payload, err := fetchPornHubFlashvars(rawURL, embedSrc)
	if err != nil {
		return nil, err
	}

	byHeight := make(map[int]HLSVariant)
	scores := make(map[int]int)

	addVariant := func(v HLSVariant, format, rawURL string) {
		if rawURL == "" {
			return
		}
		if v.Height == 0 {
			v.Height = inferHeightFromURL(rawURL)
		}
		if v.Height == 0 {
			return
		}
		v.URL = rawURL
		score := variantScore(format, rawURL)
		if existingScore, ok := scores[v.Height]; !ok || score > existingScore {
			byHeight[v.Height] = v
			scores[v.Height] = score
		}
	}

	for _, def := range payload.MediaDefinitions {
		if def.VideoURL == "" {
			continue
		}

		definitions, err := resolveMediaDefinitionURL(def)
		if err != nil {
			continue
		}

		for _, resolved := range definitions {
			if resolved.VideoURL == "" {
				continue
			}

			height := parseDefinitionQuality(resolved.Quality)
			if height == 0 {
				height = parseDefinitionQuality(def.Quality)
			}
			if height == 0 && resolved.Height > 0 {
				height = resolved.Height
			}
			if height == 0 && def.Height > 0 {
				height = def.Height
			}
			if height == 0 {
				height = inferHeightFromURL(resolved.VideoURL)
			}

			format := resolved.Format
			if format == "" {
				format = def.Format
			}

			if strings.Contains(strings.ToLower(resolved.VideoURL), ".urlset") {
				expanded, err := expandURLSetVariants(resolved.VideoURL)
				if err == nil {
					for _, v := range expanded {
						addVariant(v, "hls", v.URL)
					}
					continue
				}
			}

			originalFormat := format
			streamURL := resolved.VideoURL
			if strings.EqualFold(format, "mp4") || isMP4MediaURL(streamURL) {
				if hlsURL := resolveHLSFromMediaURL(streamURL); hlsURL != "" {
					streamURL = hlsURL
				}
			}

			addVariant(HLSVariant{Height: height}, originalFormat, streamURL)
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

func resolveMediaDefinitionURL(def mediaDefinition) ([]mediaDefinition, error) {
	if !strings.Contains(def.VideoURL, "/video/get_media") {
		return []mediaDefinition{def}, nil
	}

	body, _, err := FetchStreamResource(def.VideoURL)
	if err != nil {
		return nil, err
	}

	var mediaList []mediaDefinition
	if err := json.Unmarshal(body, &mediaList); err != nil {
		return nil, err
	}

	return mediaList, nil
}

func parseDefinitionQuality(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}

	var asInt int
	if err := json.Unmarshal(raw, &asInt); err == nil && asInt > 0 {
		return asInt
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		if parsed, err := strconv.Atoi(strings.TrimSpace(asString)); err == nil && parsed > 0 {
			return parsed
		}
	}

	return 0
}

func variantScore(format, rawURL string) int {
	score := 0
	switch strings.ToLower(format) {
	case "hls":
		score += 100
	case "mp4":
		score += 40
	}

	lower := strings.ToLower(rawURL)
	switch {
	case strings.Contains(lower, "index-v1"), strings.Contains(lower, "index-f1"):
		score += 50
	case strings.Contains(lower, "master.m3u8"):
		score += 20
	case strings.Contains(lower, ".urlset"):
		score += 10
	}

	if m := regexp.MustCompile(`(?i)(\d+)p_\d+k`).FindStringSubmatch(lower); len(m) == 2 {
		if bw, err := strconv.Atoi(m[1]); err == nil {
			score += bw / 100
		}
	}

	return score
}

func isMP4MediaURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	path := strings.ToLower(parsed.Path)
	return strings.HasSuffix(path, ".mp4")
}

// resolveHLSFromMediaURL maps PornHub media URLs to an HLS manifest URL.
func resolveHLSFromMediaURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	lowerPath := strings.ToLower(parsed.Path)
	if strings.Contains(lowerPath, ".m3u8") {
		return rawURL
	}
	if !strings.HasSuffix(lowerPath, ".mp4") {
		return ""
	}

	host := strings.ToLower(parsed.Hostname())
	if !strings.HasSuffix(host, "phncdn.com") {
		return ""
	}

	manifest := *parsed
	switch {
	case strings.Contains(lowerPath, "/hls/"):
		manifest.Path = parsed.Path + "/master.m3u8"
	case strings.Contains(lowerPath, "/videos/"):
		if !strings.Contains(host, "-h.") {
			dot := strings.Index(host, ".")
			if dot > 0 {
				manifest.Host = host[:dot] + "-h" + host[dot:]
				if parsed.Port() != "" {
					manifest.Host += ":" + parsed.Port()
				}
			}
		}
		manifest.Path = "/hls" + parsed.Path + "/master.m3u8"
	default:
		return ""
	}

	return manifest.String()
}

// expandURLSetVariants splits PornHub urlset master URLs into per-quality media playlists.
func expandURLSetVariants(urlsetURL string) ([]HLSVariant, error) {
	parsed, err := url.Parse(urlsetURL)
	if err != nil {
		return nil, err
	}

	pathLower := strings.ToLower(parsed.Path)
	idx := strings.Index(pathLower, ".mp4.urlset")
	if idx < 0 {
		return nil, nil
	}

	path := parsed.Path[:idx]
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash < 0 {
		return nil, nil
	}

	prefix := parsed.Path[:lastSlash+1]
	qualitiesPart := parsed.Path[lastSlash+1:]

	idMatch := urlsetVideoIDRe.FindStringSubmatch(parsed.Path)
	if len(idMatch) != 2 {
		return nil, nil
	}
	videoID := idMatch[1]

	qualitiesPart = strings.TrimPrefix(qualitiesPart, ",")
	parts := strings.Split(qualitiesPart, ",")

	var variants []HLSVariant
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || strings.HasPrefix(part, "_") {
			continue
		}

		height := inferHeightFromURL(part)
		if height == 0 {
			continue
		}

		mediaPath := fmt.Sprintf("%s%s_%s.mp4/index-v1-a1.m3u8", prefix, part, videoID)
		mediaURL := *parsed
		mediaURL.Path = mediaPath
		variants = append(variants, HLSVariant{
			Height: height,
			URL:    mediaURL.String(),
		})
	}

	if len(variants) == 0 {
		return nil, nil
	}

	return variants, nil
}

// SelectVariantForRequest picks a variant using height and optional bandwidth.
func SelectVariantForRequest(variants []HLSVariant, height int, bandwidth int) (*HLSVariant, error) {
	if bandwidth > 0 {
		for i := range variants {
			if variants[i].Bandwidth == bandwidth {
				return &variants[i], nil
			}
		}
	}
	return SelectVariant(variants, height)
}

// ResolveMediaPlaylist fetches the media playlist for a requested quality level.
func ResolveMediaPlaylist(embedURL string, requestedHeight, requestedBandwidth int) ([]byte, string, error) {
	embedSrc, _, err := Parse(embedURL)
	if err != nil {
		return nil, "", err
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			invalidateCachedVariants(embedSrc)
			time.Sleep(time.Duration(attempt) * 150 * time.Millisecond)
		}

		body, manifestURL, err := resolveMediaPlaylistOnce(embedURL, requestedHeight, requestedBandwidth)
		if err == nil {
			return body, manifestURL, nil
		}

		lastErr = err
		if !isExpiredUpstreamError(err) {
			return nil, "", err
		}
	}

	return nil, "", lastErr
}

func resolveMediaPlaylistOnce(embedURL string, requestedHeight, requestedBandwidth int) ([]byte, string, error) {
	variants, _, err := ExtractStreamVariants(embedURL)
	if err != nil {
		return nil, "", err
	}

	selected, err := SelectVariantForRequest(variants, requestedHeight, requestedBandwidth)
	if err != nil {
		return nil, "", err
	}

	return fetchMediaPlaylistBody(selected.URL, requestedHeight, requestedBandwidth)
}

func isExpiredUpstreamError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 410") ||
		strings.Contains(msg, "status 403") ||
		strings.Contains(msg, "status 429") ||
		strings.Contains(msg, "status 471")
}

func getCachedVariants(cacheKey string) ([]HLSVariant, bool) {
	flashvarsVariantCacheMu.RLock()
	defer flashvarsVariantCacheMu.RUnlock()

	entry, ok := flashvarsVariantCache[cacheKey]
	if !ok || time.Now().After(entry.expiry) {
		return nil, false
	}

	return append([]HLSVariant(nil), entry.variants...), true
}

func setCachedVariants(cacheKey string, variants []HLSVariant) {
	flashvarsVariantCacheMu.Lock()
	defer flashvarsVariantCacheMu.Unlock()

	flashvarsVariantCache[cacheKey] = cachedVariantsEntry{
		variants: append([]HLSVariant(nil), variants...),
		expiry:   time.Now().Add(flashvarsVariantCacheTTL),
	}
}

type remotePageVariantsResult struct {
	variants []HLSVariant
	site     Site
}

// extractCachedRemotePageVariants caches PlayerJS/KVS resolutions briefly so Range
// seeks against /embed/media do not re-fetch and re-decrypt the page on every request.
func extractCachedRemotePageVariants(rawURL string) ([]HLSVariant, Site, error) {
	cacheKey := "page:" + strings.TrimSpace(rawURL)
	if variants, ok := getCachedVariants(cacheKey); ok {
		return variants, siteFromRemoteVariants(variants), nil
	}

	result, err, _ := remotePageVariantGroup.Do(cacheKey, func() (interface{}, error) {
		if variants, ok := getCachedVariants(cacheKey); ok {
			return remotePageVariantsResult{variants: variants, site: siteFromRemoteVariants(variants)}, nil
		}

		variants, site, err := extractPlayerJSStreamVariants(rawURL)
		if err != nil {
			return nil, err
		}
		setCachedVariants(cacheKey, variants)
		return remotePageVariantsResult{variants: variants, site: site}, nil
	})
	if err != nil {
		return nil, SitePlayerJS, err
	}

	out := result.(remotePageVariantsResult)
	return out.variants, out.site, nil
}

func siteFromRemoteVariants(variants []HLSVariant) Site {
	for _, v := range variants {
		if strings.Contains(strings.ToLower(v.URL), "/get_file/") {
			return SiteKVS
		}
	}
	return SitePlayerJS
}

func invalidateCachedVariants(cacheKey string) {
	flashvarsVariantCacheMu.Lock()
	delete(flashvarsVariantCache, cacheKey)
	flashvarsVariantCacheMu.Unlock()
}

func fetchMediaPlaylistBody(startURL string, requestedHeight, requestedBandwidth int) ([]byte, string, error) {
	body, _, err := FetchStreamResource(startURL)
	if err != nil {
		if alt := alternateMediaPlaylistURL(startURL); alt != "" {
			return fetchMediaPlaylistBody(alt, requestedHeight, requestedBandwidth)
		}
		return nil, "", err
	}

	if !IsMasterPlaylist(body) {
		if !IsMediaPlaylist(body) && alternateMediaPlaylistURL(startURL) != "" {
			return fetchMediaPlaylistBody(alternateMediaPlaylistURL(startURL), requestedHeight, requestedBandwidth)
		}
		return body, startURL, nil
	}

	subVariants, err := ParseMasterVariants(body, startURL)
	if err != nil {
		return nil, "", err
	}
	if len(subVariants) == 0 {
		if alt := alternateMediaPlaylistURL(startURL); alt != "" && alt != startURL {
			return fetchMediaPlaylistBody(alt, requestedHeight, requestedBandwidth)
		}
		return body, startURL, nil
	}

	selected, err := SelectVariantForRequest(subVariants, requestedHeight, requestedBandwidth)
	if err != nil {
		return nil, "", err
	}
	if selected.URL == startURL {
		return body, startURL, nil
	}

	return fetchMediaPlaylistBody(selected.URL, requestedHeight, requestedBandwidth)
}

func alternateMediaPlaylistURL(manifestURL string) string {
	lower := strings.ToLower(manifestURL)
	const masterSuffix = "master.m3u8"
	if !strings.HasSuffix(lower, masterSuffix) {
		return ""
	}

	parsed, err := url.Parse(manifestURL)
	if err != nil {
		return ""
	}

	parsed.Path = parsed.Path[:len(parsed.Path)-len(masterSuffix)] + "index-v1-a1.m3u8"
	return parsed.String()
}

// ExtractVideoDuration returns the remote video duration in seconds when available.
func ExtractVideoDuration(rawURL string) (float64, Site, error) {
	embedSrc, site, err := Parse(rawURL)
	if err != nil {
		return 0, site, err
	}

	switch site {
	case SitePornHub:
		payload, err := fetchPornHubFlashvars(rawURL, embedSrc)
		if err != nil {
			return 0, SitePornHub, err
		}
		if payload.VideoDuration <= 0 {
			return 0, SitePornHub, ErrStreamNotFound
		}
		return payload.VideoDuration, SitePornHub, nil
	case SiteXVideos, SiteXnxx:
		duration, err := extractHTML5PlayerDuration(embedSrc, rawURL, site)
		if err != nil {
			return 0, site, err
		}
		return duration, site, nil
	case SiteXHamster:
		duration, err := extractXHamsterDuration(rawURL, embedSrc)
		if err != nil {
			return 0, site, err
		}
		return duration, SiteXHamster, nil
	case SiteDirect:
		return 0, SiteDirect, ErrStreamNotFound
	case SitePlayerJS:
		duration, err := extractPlayerJSDuration(rawURL)
		if err == nil && duration > 0 {
			return duration, SitePlayerJS, nil
		}
		// Page may actually be KVS; duration probing falls back to ffprobe.
		return 0, SitePlayerJS, ErrStreamNotFound
	case SiteKVS:
		body, err := fetchPlayerJSPage(rawURL)
		if err != nil {
			return 0, SiteKVS, err
		}
		duration, err := extractKVSDurationFromHTML(string(body))
		if err != nil {
			return 0, SiteKVS, err
		}
		return duration, SiteKVS, nil
	case SiteVKVideo:
		duration, err := extractVKVideoDuration(rawURL, embedSrc)
		if err != nil {
			return 0, SiteVKVideo, err
		}
		return duration, SiteVKVideo, nil
	case SiteJWPlayer:
		duration, err := extractJWPlayerDuration(rawURL)
		if err != nil {
			return 0, SiteJWPlayer, err
		}
		return duration, SiteJWPlayer, nil
	case SiteNewgrounds:
		duration, err := extractNewgroundsDuration(rawURL)
		if err == nil && duration > 0 {
			return duration, SiteNewgrounds, nil
		}
		return 0, SiteNewgrounds, ErrStreamNotFound
	case SiteVJav:
		duration, err := extractVJavDuration(rawURL)
		if err != nil {
			return 0, SiteVJav, err
		}
		return duration, SiteVJav, nil
	case SiteEporner:
		duration, err := extractEpornerDuration(rawURL)
		if err != nil {
			return 0, SiteEporner, err
		}
		return duration, SiteEporner, nil
	case SiteTNAFlix:
		duration, err := extractTNAFlixDuration(rawURL)
		if err != nil {
			return 0, SiteTNAFlix, err
		}
		return duration, SiteTNAFlix, nil
	default:
		return 0, site, ErrStreamNotSupported
	}
}

func fetchPornHubFlashvars(rawURL, embedSrc string) (flashvarsPayload, error) {
	pageURLs := pornHubFlashvarsPageURLs(rawURL, embedSrc)

	var best flashvarsPayload
	bestCount := 0
	var lastErr error

	for _, pageURL := range pageURLs {
		payload, err := fetchPornHubFlashvarsPage(pageURL)
		if err != nil {
			lastErr = err
			continue
		}

		count := countHLSMediaDefinitions(payload)
		if count > bestCount {
			best = payload
			bestCount = count
		}
	}

	if bestCount > 0 {
		return best, nil
	}

	if lastErr != nil {
		return flashvarsPayload{}, lastErr
	}
	return flashvarsPayload{}, ErrStreamNotFound
}

func countHLSMediaDefinitions(payload flashvarsPayload) int {
	count := 0
	for _, def := range payload.MediaDefinitions {
		if strings.EqualFold(def.Format, "hls") && def.VideoURL != "" {
			count++
		}
	}
	return count
}

func pornHubFlashvarsPageURLs(rawURL, embedSrc string) []string {
	var pageURLs []string
	addURL := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		for _, existing := range pageURLs {
			if existing == candidate {
				return
			}
		}
		pageURLs = append(pageURLs, candidate)
	}

	addURL(PornHubViewPageURL(rawURL))
	addURL(PornHubViewPageURL(embedSrc))
	addURL(embedSrc)
	if rawURL != embedSrc {
		addURL(rawURL)
	}

	return pageURLs
}

func fetchPornHubFlashvarsPage(pageURL string) (flashvarsPayload, error) {
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return flashvarsPayload{}, err
	}
	req.Header.Set("User-Agent", streamUserAgent)
	req.Header.Set("Referer", "https://www.pornhub.com/")
	req.Header.Set("Origin", "https://www.pornhub.com")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cookie", "accessAgeDisclaimerPH=1; age_verified=1")

	resp, err := streamHTTPClient.Do(req)
	if err != nil {
		return flashvarsPayload{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return flashvarsPayload{}, fmt.Errorf("pornhub page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return flashvarsPayload{}, err
	}

	html := string(body)
	if isPornHubBotChallenge(html) {
		return flashvarsPayload{}, ErrStreamNotFound
	}

	flashvarsJSON, err := extractFlashvarsJSON(html)
	if err != nil {
		return flashvarsPayload{}, err
	}

	var payload flashvarsPayload
	if err := json.Unmarshal([]byte(flashvarsJSON), &payload); err != nil {
		return flashvarsPayload{}, fmt.Errorf("parse pornhub flashvars: %w", err)
	}

	return payload, nil
}

func isPornHubBotChallenge(html string) bool {
	return strings.Contains(html, "function leastFactor(") ||
		strings.Contains(html, "rf - check failed")
}

func extractFlashvarsJSON(html string) (string, error) {
	loc := flashvarsAssignRe.FindStringIndex(html)
	if loc == nil {
		return "", ErrStreamNotFound
	}

	start := loc[1] - 1
	if start < 0 || start >= len(html) || html[start] != '{' {
		return "", ErrStreamNotFound
	}

	depth := 0
	inString := false
	escape := false

	for i := start; i < len(html); i++ {
		c := html[i]

		if escape {
			escape = false
			continue
		}

		if inString {
			if c == '\\' {
				escape = true
			} else if c == '"' {
				inString = false
			}
			continue
		}

		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return html[start : i+1], nil
			}
		}
	}

	return "", ErrStreamNotFound
}
