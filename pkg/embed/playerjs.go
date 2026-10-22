package embed

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// SitePlayerJS is a watch page that embeds PlayerJS with a (often temporary) media URL.
const SitePlayerJS Site = "playerjs"

var (
	playerjsMarkerRe = regexp.MustCompile(`(?i)playerjs`)
	playerjsCallRe   = regexp.MustCompile(`(?i)(?:new\s+)?Playerjs\s*\(`)
	// Matches file:"https://..." / "file":"https://..." inside PlayerJS config / playlists.
	playerjsFileRe = regexp.MustCompile(`(?i)(?:["']file["']|\bfile)\s*:\s*["'](https?://[^"']+)["']`)
	playerjsDurationRe = regexp.MustCompile(`(?i)\bduration\s*:\s*(\d+(?:\.\d+)?)`)
	// iframe / JSON embeds like playerjs.html?file=mp4/1000/1996.mp4
	playerjsHTMLEmbedRe = regexp.MustCompile(`(?i)((?:https?:)?//[^"'\\\s<>]*?/)?playerjs\.html\?([^"'\\\s<>]+)`)
	playerjsAPIURLRe    = regexp.MustCompile(`(?i)API_URL\s*=\s*['"](https?://[^'"]+)['"]`)
)

type playerJSHTMLEmbed struct {
	playerURL string
	file      string
}

// IsPlayerJSSite reports whether the site is a PlayerJS page.
func IsPlayerJSSite(site Site) bool {
	return site == SitePlayerJS
}

func parsePlayerJSPage(raw string) (string, Site, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", SitePlayerJS, ErrInvalidURL
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", SitePlayerJS, ErrInvalidURL
	}
	if parsed.Hostname() == "" {
		return "", SitePlayerJS, ErrInvalidURL
	}
	// No iframe — playback uses sceneStreams with request-time URL refresh.
	return "", SitePlayerJS, nil
}

func extractPlayerJSStreamVariants(rawURL string) ([]HLSVariant, Site, error) {
	body, err := fetchPlayerJSPage(rawURL)
	if err != nil {
		return nil, SitePlayerJS, err
	}

	html := string(body)

	// KVS (kt_player) pages are also accepted as generic remote URLs.
	if isKVSPage(html) {
		variants, err := extractKVSStreamVariantsFromHTML(rawURL, html)
		if err != nil {
			return nil, SiteKVS, err
		}
		return variants, SiteKVS, nil
	}

	if !playerjsMarkerRe.MatchString(html) {
		return nil, SitePlayerJS, fmt.Errorf("%w: playerjs not found on page", ErrStreamNotFound)
	}

	files := extractPlayerJSFiles(html)
	if len(files) == 0 {
		files, err = resolvePlayerJSSignedFiles(rawURL, html)
		if err != nil {
			return nil, SitePlayerJS, err
		}
	}
	if len(files) == 0 {
		return nil, SitePlayerJS, fmt.Errorf("%w: playerjs file URL not found", ErrStreamNotFound)
	}

	variants := make([]HLSVariant, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	for _, fileURL := range files {
		if _, ok := seen[fileURL]; ok {
			continue
		}
		seen[fileURL] = struct{}{}
		variants = append(variants, HLSVariant{
			URL:    fileURL,
			Height: inferHeightFromURL(fileURL),
		})
	}
	return variants, SitePlayerJS, nil
}

func extractPlayerJSDuration(rawURL string) (float64, error) {
	body, err := fetchPlayerJSPage(rawURL)
	if err != nil {
		return 0, err
	}

	html := string(body)
	if !playerjsMarkerRe.MatchString(html) {
		return 0, ErrStreamNotFound
	}

	if match := playerjsDurationRe.FindStringSubmatch(html); len(match) == 2 {
		seconds, err := strconv.ParseFloat(match[1], 64)
		if err == nil && seconds > 0 {
			return seconds, nil
		}
	}

	return 0, ErrStreamNotFound
}

func extractPlayerJSFiles(html string) []string {
	var files []string

	// Prefer file URLs that appear near a Playerjs(...) call.
	locs := playerjsCallRe.FindAllStringIndex(html, -1)
	for _, loc := range locs {
		start := loc[1]
		for start < len(html) && (html[start] == ' ' || html[start] == '\n' || html[start] == '\r' || html[start] == '\t') {
			start++
		}
		if start >= len(html) || html[start] != '{' {
			continue
		}
		obj, err := extractBalancedJSObject(html, start)
		if err != nil {
			continue
		}
		files = append(files, playerJSFilesFromConfig(obj)...)
	}

	if len(files) == 0 {
		files = append(files, playerJSFilesFromConfig(html)...)
	}

	return uniqueMediaURLs(files)
}

// resolvePlayerJSSignedFiles handles sites that load PlayerJS via
// playerjs.html?file=relative/path.mp4 and resolve a temporary URL through an API.
func resolvePlayerJSSignedFiles(pageURL, html string) ([]string, error) {
	embeds := findPlayerJSHTMLEmbeds(pageURL, html)
	if len(embeds) == 0 {
		return nil, nil
	}

	apiCache := make(map[string]string, len(embeds))
	var files []string
	var lastErr error

	for _, emb := range embeds {
		apiURL, ok := apiCache[emb.playerURL]
		if !ok {
			apiURL = discoverPlayerJSLinkAPI(emb.playerURL)
			apiCache[emb.playerURL] = apiURL
		}
		if apiURL == "" {
			lastErr = fmt.Errorf("playerjs link API not found")
			continue
		}

		signed, err := fetchPlayerJSSignedURL(apiURL, emb.file, pageURL)
		if err != nil {
			lastErr = err
			continue
		}
		files = append(files, signed)
	}

	files = uniqueMediaURLs(files)
	if len(files) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return files, nil
}

func findPlayerJSHTMLEmbeds(pageURL, html string) []playerJSHTMLEmbed {
	normalized := strings.ReplaceAll(html, `\/`, `/`)
	matches := playerjsHTMLEmbedRe.FindAllStringSubmatch(normalized, -1)
	if len(matches) == 0 {
		return nil
	}

	base, err := url.Parse(pageURL)
	if err != nil {
		return nil
	}

	var embeds []playerJSHTMLEmbed
	seen := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		query := strings.ReplaceAll(m[2], `&amp;`, `&`)
		values, err := url.ParseQuery(query)
		if err != nil {
			continue
		}
		file := strings.TrimSpace(values.Get("file"))
		if file == "" {
			continue
		}

		playerRef := "playerjs.html?" + query
		if prefix := strings.TrimSpace(m[1]); prefix != "" {
			playerRef = prefix + "playerjs.html?" + query
		}

		playerURL, err := base.Parse(playerRef)
		if err != nil {
			continue
		}
		abs := playerURL.String()
		key := abs + "\n" + file
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		embeds = append(embeds, playerJSHTMLEmbed{
			playerURL: abs,
			file:      file,
		})
	}
	return embeds
}

func discoverPlayerJSLinkAPI(playerHTMLURL string) string {
	body, err := fetchPlayerJSPage(playerHTMLURL)
	if err == nil {
		if match := playerjsAPIURLRe.FindStringSubmatch(string(body)); len(match) == 2 {
			return strings.TrimSpace(match[1])
		}
	}

	parsed, err := url.Parse(playerHTMLURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host + "/api/get_video_link.php"
}

func fetchPlayerJSSignedURL(apiURL, file, refererURL string) (string, error) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}
	q := parsed.Query()
	q.Set("file", file)
	parsed.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", err
	}
	applyPageHeaders(req, refererURL)

	resp, err := streamHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("playerjs link API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}

	signed := strings.TrimSpace(string(body))
	signed = strings.Trim(signed, `"'`)
	if !looksLikePlayerJSMediaURL(signed) {
		return "", fmt.Errorf("%w: playerjs link API returned non-media URL", ErrStreamNotFound)
	}
	return signed, nil
}

func playerJSFilesFromConfig(config string) []string {
	matches := playerjsFileRe.FindAllStringSubmatch(config, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		fileURL := strings.TrimSpace(m[1])
		if fileURL == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(fileURL), "js:") {
			continue
		}
		if !looksLikePlayerJSMediaURL(fileURL) {
			continue
		}
		out = append(out, fileURL)
	}
	return out
}

func looksLikePlayerJSMediaURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	pathLower := strings.ToLower(parsed.Path)
	if isDirectMediaPath(parsed.Path) {
		return true
	}
	if strings.HasSuffix(pathLower, ".m3u8") || strings.Contains(pathLower, ".m3u8") {
		return true
	}
	// Signed CDN links sometimes omit a clean extension in path edge cases;
	// still accept common video hosts when query carries auth.
	host := strings.ToLower(parsed.Hostname())
	if strings.Contains(host, "backblazeb2.com") || strings.Contains(host, "b2.cloud") {
		return true
	}
	return false
}

func uniqueMediaURLs(urls []string) []string {
	seen := make(map[string]struct{}, len(urls))
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

func fetchPlayerJSPage(pageURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}
	applyPageHeaders(req, pageURL)

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

func applyPageHeaders(req *http.Request, pageURL string) {
	req.Header.Set("User-Agent", streamUserAgent)
	referer, origin := pageRefererOrigin(pageURL)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
}

func pageRefererOrigin(pageURL string) (referer, origin string) {
	parsed, err := url.Parse(pageURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", ""
	}
	origin = parsed.Scheme + "://" + parsed.Host
	return origin + "/", origin
}

// extractBalancedJSObject extracts a {...} object supporting single- and double-quoted strings.
func extractBalancedJSObject(html string, start int) (string, error) {
	depth := 0
	inString := false
	var quote byte
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
			} else if c == quote {
				inString = false
			}
			continue
		}

		switch c {
		case '"', '\'':
			inString = true
			quote = c
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return html[start : i+1], nil
			}
		}
	}

	return "", fmt.Errorf("unbalanced object")
}
