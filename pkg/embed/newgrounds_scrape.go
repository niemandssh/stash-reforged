package embed

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	newgroundsTagsBlockRe  = regexp.MustCompile(`(?is)<dt[^>]*>\s*Tags\s*</dt>\s*<dd[^>]*>(.*?)</dd>`)
	newgroundsLegacyTagsRe = regexp.MustCompile(`(?is)<dd class="tags">(.*?)</dd>`)
	newgroundsTagLinkRe    = regexp.MustCompile(`(?is)<a[^>]*>([^<]+)</a>`)
	newgroundsUploadDateRe = regexp.MustCompile(`(?is)itemprop="uploadDate"\s+content="([^"]+)"`)
	newgroundsOGImageRe    = regexp.MustCompile(`(?is)property="og:image"\s+content="([^"]+)"`)
	newgroundsOGDescRe     = regexp.MustCompile(`(?is)property="og:description"\s+content="([^"]+)"`)
)

// SceneMetadata is scraped Newgrounds portal metadata.
type SceneMetadata struct {
	Title   string
	URL     string
	Date    string
	Image   string
	Details string
	Studio  string
	Tags    []string
}

type newgroundsVideoMetadataResponse struct {
	ID     json.Number `json:"id"`
	Title  string      `json:"title"`
	Author string      `json:"author"`
}

// ScrapeNewgroundsScene collects scene metadata from the JSON video API and,
// when available, the portal HTML page for tags and other fields.
func ScrapeNewgroundsScene(pageURL string, pageHTML ...string) (*SceneMetadata, error) {
	mediaID := newgroundsMediaID(pageURL)
	if mediaID == "" {
		return nil, ErrInvalidURL
	}

	meta := &SceneMetadata{
		URL: pageURL,
	}

	body, err := fetchNewgroundsVideoJSON(pageURL, mediaID)
	if err != nil {
		return nil, err
	}

	var payload newgroundsVideoMetadataResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse newgrounds video json: %w", err)
	}

	meta.Title = strings.TrimSpace(payload.Title)
	meta.Studio = strings.TrimSpace(payload.Author)

	if len(pageHTML) > 0 && pageHTML[0] != "" {
		enrichNewgroundsSceneFromHTML(meta, pageHTML[0])
	} else if pageBody, pageErr := fetchNewgroundsPage(pageURL); pageErr == nil {
		enrichNewgroundsSceneFromHTML(meta, string(pageBody))
	}

	if meta.Title == "" {
		return nil, fmt.Errorf("newgrounds scene metadata missing title")
	}

	return meta, nil
}

func enrichNewgroundsSceneFromHTML(meta *SceneMetadata, pageHTML string) {
	if strings.Contains(pageHTML, "Content Filtered") {
		return
	}

	if meta.Date == "" {
		if match := newgroundsUploadDateRe.FindStringSubmatch(pageHTML); len(match) == 2 {
			meta.Date = strings.SplitN(match[1], "T", 2)[0]
		}
	}

	if meta.Image == "" {
		if match := newgroundsOGImageRe.FindStringSubmatch(pageHTML); len(match) == 2 {
			meta.Image = html.UnescapeString(match[1])
		}
	}

	if meta.Details == "" {
		if match := newgroundsOGDescRe.FindStringSubmatch(pageHTML); len(match) == 2 {
			meta.Details = html.UnescapeString(match[1])
		}
	}

	if len(meta.Tags) == 0 {
		meta.Tags = extractNewgroundsTagsFromHTML(pageHTML)
	}
}

func extractNewgroundsTagsFromHTML(pageHTML string) []string {
	var blocks []string
	if match := newgroundsTagsBlockRe.FindStringSubmatch(pageHTML); len(match) == 2 {
		blocks = append(blocks, match[1])
	}
	if match := newgroundsLegacyTagsRe.FindStringSubmatch(pageHTML); len(match) == 2 {
		blocks = append(blocks, match[1])
	}

	seen := make(map[string]struct{})
	var tags []string
	for _, block := range blocks {
		for _, match := range newgroundsTagLinkRe.FindAllStringSubmatch(block, -1) {
			if len(match) != 2 {
				continue
			}

			tag := strings.TrimSpace(html.UnescapeString(match[1]))
			if tag == "" || strings.Contains(tag, "(function(") {
				continue
			}

			key := strings.ToLower(tag)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			tags = append(tags, tag)
		}
	}

	return tags
}
