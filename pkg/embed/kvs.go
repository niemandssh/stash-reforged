package embed

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// SiteKVS is a Kernel Video Sharing (kt_player) page or embed.
const SiteKVS Site = "kvs"

var (
	kvsPlayerMarkerRe = regexp.MustCompile(`(?i)kt_player|flashvars\s*=`)
	kvsLicenseCodeRe  = regexp.MustCompile(`(?i)license_code\s*:\s*['"]([^'"]+)['"]`)
	kvsVideoURLRe     = regexp.MustCompile(`(?i)(video_url|video_alt_url\d*)\s*:\s*['"]([^'"]+)['"]`)
	kvsVideoURLTextRe = regexp.MustCompile(`(?i)(video_url|video_alt_url\d*)_text\s*:\s*['"]([^'"]+)['"]`)
	kvsHeightInTextRe = regexp.MustCompile(`(?i)(\d{3,4})p`)
	kvsDurationRe     = regexp.MustCompile(`(?i)video_duration\s*:\s*['"]?(\d+(?:\.\d+)?)['"]?`)
	kvsSourceTagRe    = regexp.MustCompile(`(?i)<source\b[^>]*\bsrc=["']([^"']*/get_file/[^"']+)["'][^>]*>`)
	kvsTitleAttrRe    = regexp.MustCompile(`(?i)\btitle=["']([^"']*)["']`)
)

// IsKVSSite reports whether the site is a KVS (kt_player) page.
func IsKVSSite(site Site) bool {
	return site == SiteKVS
}

func isKVSPage(html string) bool {
	if !kvsPlayerMarkerRe.MatchString(html) {
		return false
	}
	return (strings.Contains(html, "video_url") && strings.Contains(html, "license_code")) ||
		kvsSourceTagRe.MatchString(html)
}

// extractKVSSourceTagVariants reads plain <source> tags that some KVS themes (Fluid Player) render instead of flashvars.
func extractKVSSourceTagVariants(base *url.URL, html string) []HLSVariant {
	var variants []HLSVariant
	seen := make(map[string]struct{})
	for _, m := range kvsSourceTagRe.FindAllStringSubmatch(html, -1) {
		abs, err := base.Parse(strings.ReplaceAll(m[1], "&amp;", "&"))
		if err != nil {
			continue
		}
		mediaURL := abs.String()
		if _, ok := seen[mediaURL]; ok {
			continue
		}
		seen[mediaURL] = struct{}{}

		height := inferHeightFromURL(mediaURL)
		if height == 0 {
			if tm := kvsTitleAttrRe.FindStringSubmatch(m[0]); len(tm) == 2 {
				if hm := kvsHeightInTextRe.FindStringSubmatch(tm[1]); len(hm) == 2 {
					height, _ = strconv.Atoi(hm[1])
				}
			}
		}
		variants = append(variants, HLSVariant{URL: mediaURL, Height: height})
	}
	return variants
}

func extractKVSStreamVariantsFromHTML(pageURL, html string) ([]HLSVariant, error) {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, err
	}

	license := ""
	if m := kvsLicenseCodeRe.FindStringSubmatch(html); len(m) == 2 {
		license = strings.TrimSpace(m[1])
	}
	if license == "" {
		if variants := extractKVSSourceTagVariants(base, html); len(variants) > 0 {
			return variants, nil
		}
		return nil, fmt.Errorf("%w: kvs license_code not found", ErrStreamNotFound)
	}

	texts := map[string]string{}
	for _, m := range kvsVideoURLTextRe.FindAllStringSubmatch(html, -1) {
		if len(m) == 3 {
			texts[strings.ToLower(m[1])] = strings.TrimSpace(m[2])
		}
	}

	var variants []HLSVariant
	seen := make(map[string]struct{})
	for _, m := range kvsVideoURLRe.FindAllStringSubmatch(html, -1) {
		if len(m) != 3 {
			continue
		}
		key := strings.ToLower(m[1])
		raw := strings.TrimSpace(m[2])
		if raw == "" || !strings.Contains(raw, "/get_file/") {
			continue
		}

		decoded := kvsGetRealURL(raw, license)
		abs, err := base.Parse(decoded)
		if err != nil {
			continue
		}
		mediaURL := abs.String()
		if _, ok := seen[mediaURL]; ok {
			continue
		}
		seen[mediaURL] = struct{}{}

		height := inferHeightFromURL(mediaURL)
		if height == 0 {
			if text := texts[key]; text != "" {
				if hm := kvsHeightInTextRe.FindStringSubmatch(text); len(hm) == 2 {
					height, _ = strconv.Atoi(hm[1])
				}
			}
		}
		variants = append(variants, HLSVariant{
			URL:    mediaURL,
			Height: height,
		})
	}

	if len(variants) == 0 {
		return nil, fmt.Errorf("%w: kvs video_url not found", ErrStreamNotFound)
	}
	return variants, nil
}

func extractKVSDurationFromHTML(html string) (float64, error) {
	if m := kvsDurationRe.FindStringSubmatch(html); len(m) == 2 {
		seconds, err := strconv.ParseFloat(m[1], 64)
		if err == nil && seconds > 0 {
			return seconds, nil
		}
	}
	return 0, ErrStreamNotFound
}

// kvsGetRealURL deobfuscates KVS function/0/ video URLs using license_code.
// Algorithm matches yt-dlp's GenericIE._kvs_get_real_url.
func kvsGetRealURL(videoURL, licenseCode string) string {
	if !strings.HasPrefix(videoURL, "function/0/") {
		return videoURL
	}

	parsed, err := url.Parse(videoURL[len("function/0/"):])
	if err != nil {
		return videoURL
	}

	token := kvsLicenseToken(licenseCode)
	parts := strings.Split(parsed.Path, "/")
	if len(parts) < 4 {
		return videoURL
	}

	const hashLen = 32
	if len(parts[3]) < hashLen {
		return videoURL
	}

	hash := parts[3][:hashLen]
	indices := make([]int, hashLen)
	for i := range indices {
		indices[i] = i
	}

	accum := 0
	for src := hashLen - 1; src >= 0; src-- {
		accum += token[src]
		dest := (src + accum) % hashLen
		indices[src], indices[dest] = indices[dest], indices[src]
	}

	var rebuilt strings.Builder
	rebuilt.Grow(hashLen)
	for _, idx := range indices {
		rebuilt.WriteByte(hash[idx])
	}
	parts[3] = rebuilt.String() + parts[3][hashLen:]
	parsed.Path = strings.Join(parts, "/")
	return parsed.String()
}

func kvsLicenseToken(licenseCode string) []int {
	licenseCode = strings.ReplaceAll(licenseCode, "$", "")
	licenseValues := make([]int, len(licenseCode))
	for i, ch := range licenseCode {
		licenseValues[i] = int(ch - '0')
	}

	mod := strings.ReplaceAll(licenseCode, "0", "1")
	center := len(mod) / 2
	front, _ := strconv.Atoi(mod[:center+1])
	back, _ := strconv.Atoi(mod[center:])
	mod = strconv.Itoa(4 * absInt(front-back))
	if len(mod) > center+1 {
		mod = mod[:center+1]
	}

	token := make([]int, 0, len(mod)*4)
	for index, current := range mod {
		cur := int(current - '0')
		for offset := 0; offset < 4; offset++ {
			token = append(token, (licenseValues[index+offset]+cur)%10)
		}
	}
	return token
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
