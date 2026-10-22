package embed

import (
	"encoding/json"
	"fmt"
	"math/bits"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	xhamsterXPlayerSettingsRe = regexp.MustCompile(`"xplayerSettings"\s*:\s*\{`)
	xhamsterVideoPagePathRe   = regexp.MustCompile(`(?i)/videos/[^"'?\s]+-(?:xh[a-z0-9]+|\d+)`)
	xhamsterQualityHeightRe   = regexp.MustCompile(`(?i)^(\d+)p$`)
)

type xhamsterPlayerSettings struct {
	VideoID  int                    `json:"videoId"`
	Duration float64                `json:"duration"`
	Sources  xhamsterPlayerSources  `json:"sources"`
}

type xhamsterPlayerSources struct {
	HLS      map[string]xhamsterURLEntry   `json:"hls"`
	Standard  map[string][]xhamsterURLEntry `json:"standard"`
}

type xhamsterURLEntry struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
	Label   string `json:"label"`
}

func extractXHamsterStreamVariants(rawURL, embedSrc string) ([]HLSVariant, error) {
	pageURL, err := resolveXHamsterViewPageURL(rawURL, embedSrc)
	if err != nil {
		return nil, err
	}

	body, err := fetchSitePage(pageURL, SiteXHamster)
	if err != nil {
		return nil, err
	}

	settings, err := extractXHamsterPlayerSettings(string(body))
	if err != nil {
		return nil, err
	}

	if variants := xhamsterHLSVariants(settings); len(variants) > 0 {
		return variants, nil
	}

	if variants := xhamsterMP4Variants(settings); len(variants) > 0 {
		return variants, nil
	}

	return nil, ErrStreamNotFound
}

func extractXHamsterDuration(rawURL, embedSrc string) (float64, error) {
	pageURL, err := resolveXHamsterViewPageURL(rawURL, embedSrc)
	if err != nil {
		return 0, err
	}

	body, err := fetchSitePage(pageURL, SiteXHamster)
	if err != nil {
		return 0, err
	}

	settings, err := extractXHamsterPlayerSettings(string(body))
	if err != nil {
		return 0, err
	}
	if settings.Duration <= 0 {
		return 0, ErrStreamNotFound
	}
	return settings.Duration, nil
}

func resolveXHamsterViewPageURL(rawURL, embedSrc string) (string, error) {
	if page := normalizeXHamsterPageURL(rawURL); page != "" {
		return page, nil
	}
	if page := normalizeXHamsterPageURL(embedSrc); page != "" {
		return page, nil
	}

	// Embed pages link back to the watch URL.
	body, err := fetchSitePage(embedSrc, SiteXHamster)
	if err != nil {
		return "", err
	}

	html := string(body)
	if m := regexp.MustCompile(`(?i)https://(?:[\w-]+\.)*xhamster\.[a-z.]+/videos/[^"'?\s]+`).FindString(html); m != "" {
		if page := normalizeXHamsterPageURL(m); page != "" {
			return page, nil
		}
	}
	if m := xhamsterVideoPagePathRe.FindString(html); m != "" {
		return "https://xhamster.com" + m, nil
	}

	return "", ErrStreamNotFound
}

func normalizeXHamsterPageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if !isXHamsterHost(normalizeHost(parsed.Hostname())) {
		return ""
	}
	if !xhamsterVideoPagePathRe.MatchString(parsed.Path) {
		return ""
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func extractXHamsterPlayerSettings(html string) (xhamsterPlayerSettings, error) {
	var best xhamsterPlayerSettings
	bestScore := -1

	locs := xhamsterXPlayerSettingsRe.FindAllStringIndex(html, -1)
	for _, loc := range locs {
		start := loc[1] - 1
		if start < 0 || start >= len(html) || html[start] != '{' {
			continue
		}

		jsonStr, err := extractBalancedJSONObject(html, start)
		if err != nil {
			continue
		}

		var settings xhamsterPlayerSettings
		if err := json.Unmarshal([]byte(jsonStr), &settings); err != nil {
			continue
		}

		score := len(settings.Sources.HLS)
		for _, list := range settings.Sources.Standard {
			score += len(list)
		}
		if settings.Duration > 0 {
			score++
		}
		if score > bestScore {
			best = settings
			bestScore = score
		}
	}

	if bestScore < 0 || (len(best.Sources.HLS) == 0 && len(best.Sources.Standard) == 0) {
		return xhamsterPlayerSettings{}, ErrStreamNotFound
	}
	return best, nil
}

func extractBalancedJSONObject(html string, start int) (string, error) {
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

func xhamsterHLSVariants(settings xhamsterPlayerSettings) []HLSVariant {
	entry, ok := settings.Sources.HLS["h264"]
	if !ok || strings.TrimSpace(entry.URL) == "" {
		// Prefer h264, but fall back to any available codec.
		for _, e := range settings.Sources.HLS {
			if strings.TrimSpace(e.URL) != "" {
				entry = e
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil
	}

	masterURL := decodeXHamsterURL(entry.URL)
	if masterURL == "" || !strings.Contains(strings.ToLower(masterURL), ".m3u8") {
		return nil
	}

	masterBody, _, err := fetchStreamResourceForSite(masterURL, SiteXHamster)
	if err != nil {
		return nil
	}

	variants, err := ParseMasterVariants(masterBody, masterURL)
	if err != nil || len(variants) == 0 {
		// Master URL itself can be playable / used as a single variant.
		height := inferHeightFromURL(masterURL)
		return prepareVariants([]HLSVariant{{
			Height: height,
			URL:    masterURL,
		}})
	}

	return prepareVariants(variants)
}

func xhamsterMP4Variants(settings xhamsterPlayerSettings) []HLSVariant {
	entries := settings.Sources.Standard["h264"]
	if len(entries) == 0 {
		for _, list := range settings.Sources.Standard {
			if len(list) > 0 {
				entries = list
				break
			}
		}
	}

	byHeight := make(map[int]HLSVariant)
	for _, entry := range entries {
		decoded := decodeXHamsterURL(entry.URL)
		if decoded == "" {
			continue
		}

		height := xhamsterQualityToHeight(entry.Quality)
		if height == 0 {
			height = xhamsterQualityToHeight(entry.Label)
		}
		if height == 0 {
			height = inferHeightFromURL(decoded)
		}
		if height == 0 && strings.EqualFold(entry.Quality, "auto") {
			continue
		}

		byHeight[height] = HLSVariant{
			Height: height,
			URL:    decoded,
		}
	}

	if len(byHeight) == 0 {
		return nil
	}

	variants := make([]HLSVariant, 0, len(byHeight))
	for _, v := range byHeight {
		variants = append(variants, v)
	}
	return prepareVariants(variants)
}

func xhamsterQualityToHeight(q string) int {
	q = strings.TrimSpace(strings.ToLower(q))
	if m := xhamsterQualityHeightRe.FindStringSubmatch(q); len(m) == 2 {
		h, _ := strconv.Atoi(m[1])
		return h
	}
	return 0
}

// decodeXHamsterURL decrypts xHamster's hex-encoded media URLs.
// Format: [algo:1byte][seed:4bytes LE][xor-encrypted payload...]
func decodeXHamsterURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if decoded := decodeXHamsterHexURL(raw); decoded != "" {
		return decoded
	}
	return ""
}

func decodeXHamsterHexURL(hexStr string) string {
	if len(hexStr) < 12 || len(hexStr)%2 != 0 {
		return ""
	}
	for _, r := range hexStr {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return ""
		}
	}

	data := make([]byte, len(hexStr)/2)
	for i := 0; i < len(data); i++ {
		v, err := strconv.ParseUint(hexStr[i*2:i*2+2], 16, 8)
		if err != nil {
			return ""
		}
		data[i] = byte(v)
	}
	if len(data) < 5 {
		return ""
	}

	algo := int(data[0])
	if algo < 1 || algo > 7 {
		return ""
	}

	seed := int32(uint32(data[1]) | uint32(data[2])<<8 | uint32(data[3])<<16 | uint32(data[4])<<24)
	prng, err := newXHamsterPRNG(algo, seed)
	if err != nil {
		return ""
	}

	out := make([]byte, len(data)-5)
	for i := 5; i < len(data); i++ {
		out[i-5] = data[i] ^ prng()
	}

	decoded, err := url.QueryUnescape(toPercentEncoded(out))
	if err != nil {
		decoded = string(out)
	}
	if decoded == "" {
		return ""
	}
	for _, r := range decoded {
		if r < 0x20 || r > 0x7e {
			return ""
		}
	}
	return decoded
}

func toPercentEncoded(b []byte) string {
	var sb strings.Builder
	sb.Grow(3 * len(b))
	for _, c := range b {
		sb.WriteByte('%')
		sb.WriteByte("0123456789abcdef"[c>>4])
		sb.WriteByte("0123456789abcdef"[c&0x0f])
	}
	return sb.String()
}

func newXHamsterPRNG(algo int, seed int32) (func() byte, error) {
	i := seed

	const (
		mul85ebca77 uint32 = 0x85ebca77
		mulC2b2ae3d uint32 = 0xc2b2ae3d
		mul27d4eb2d uint32 = 0x27d4eb2d
		mul2c9277b5 uint32 = 0x2c9277b5
		mul7feb352d uint32 = 0x7feb352d
		mul846ca68b uint32 = 0x846ca68b
	)

	switch algo {
	case 1:
		return func() byte {
			i = int32(uint32(jsImul(i, 1664525)) + 0x3c6ef35f)
			return byte(i)
		}, nil
	case 2:
		return func() byte {
			i ^= i << 13
			i ^= int32(uint32(i) >> 17)
			i ^= i << 5
			return byte(i)
		}, nil
	case 3:
		return func() byte {
			i = int32(uint32(i) + 0x9e3779b9)
			e := i
			e ^= int32(uint32(e) >> 16)
			e = jsImul(e, u32AsI32(mul85ebca77))
			e ^= int32(uint32(e) >> 13)
			e = jsImul(e, u32AsI32(mulC2b2ae3d))
			e ^= int32(uint32(e) >> 16)
			return byte(e)
		}, nil
	case 4:
		return func() byte {
			i = int32(uint32(i) + 0x6d2b79f5)
			e := bits.RotateLeft32(uint32(i), 7)
			e += 0x9e3779b9
			e ^= e >> 11
			return byte(jsImul(int32(e), u32AsI32(mul27d4eb2d)))
		}, nil
	case 5:
		return func() byte {
			i ^= i << 7
			i ^= int32(uint32(i) >> 9)
			i ^= i << 8
			i = int32(uint32(i) + 0xa5a5a5a5)
			return byte(i)
		}, nil
	case 6:
		return func() byte {
			i = int32(uint32(jsImul(i, u32AsI32(mul2c9277b5))) + 0xac564b05)
			u := uint32(i)
			return byte((u ^ (u >> 18)) >> ((u >> 27) & 31))
		}, nil
	case 7:
		return func() byte {
			i = int32(uint32(i) + 0x9e3779b9)
			e := i ^ (i << 5)
			e = jsImul(e, u32AsI32(mul7feb352d))
			e ^= int32(uint32(e) >> 15)
			e = jsImul(e, u32AsI32(mul846ca68b))
			return byte(e)
		}, nil
	default:
		return nil, fmt.Errorf("unknown xhamster algo id: %d", algo)
	}
}

// jsImul mirrors JavaScript Math.imul / 32-bit signed multiply.
func jsImul(a, b int32) int32 {
	return int32(uint32(a) * uint32(b))
}

func u32AsI32(v uint32) int32 {
	return int32(v)
}
