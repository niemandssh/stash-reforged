package embed

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseNewgroundsPage(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{
			name: "portal view",
			url:  "https://www.newgrounds.com/portal/view/1039229",
		},
		{
			name: "portal view without www",
			url:  "https://newgrounds.com/portal/view/872806",
		},
		{
			name:    "invalid path",
			url:     "https://www.newgrounds.com/audio/listen/549479",
			wantErr: ErrInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, site, err := Parse(tt.url)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("Parse() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if site != SiteNewgrounds {
				t.Fatalf("site = %q, want %q", site, SiteNewgrounds)
			}
			if src != "" {
				t.Fatalf("iframe src = %q, want empty", src)
			}
		})
	}
}

func TestParseNewgroundsFormatHeight(t *testing.T) {
	if got := parseNewgroundsFormatHeight("1080p"); got != 1080 {
		t.Fatalf("height = %d, want 1080", got)
	}
	if got := parseNewgroundsFormatHeight("source"); got != 0 {
		t.Fatalf("height = %d, want 0", got)
	}
}

func TestExtractNewgroundsStreamVariantsFromAPIPayload(t *testing.T) {
	payload := `{
		"sources": {
			"1080p": [{"type":"video/mp4","src":"https://uploads.ungrounded.net/alternate/1/1.1080p.mp4?1"}],
			"720p": [{"type":"video/mp4","src":"https://uploads.ungrounded.net/alternate/1/1.720p.mp4?2"}]
		}
	}`

	var response newgroundsVideoResponse
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var variants []HLSVariant
	for formatID, sources := range response.Sources {
		height := parseNewgroundsFormatHeight(formatID)
		for _, source := range sources {
			variants = append(variants, HLSVariant{URL: source.Src, Height: height})
		}
	}

	if len(variants) != 2 {
		t.Fatalf("variants = %#v", variants)
	}
}

func TestSupportsNativeStreamingNewgrounds(t *testing.T) {
	if !SupportsNativeStreaming(SiteNewgrounds) {
		t.Fatal("expected newgrounds native streaming support")
	}
}

func TestIsAllowedProxyURLNewgrounds(t *testing.T) {
	if !IsAllowedProxyURL("https://uploads.ungrounded.net/alternate/7870000/7870047_alternate_366343.720p.mp4?1782611121") {
		t.Fatal("expected ungrounded CDN to be allowed")
	}
}

func TestLiveNewgroundsExtract(t *testing.T) {
	if testing.Short() {
		t.Skip("skip live network test")
	}

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "example portal view",
			url:  "https://www.newgrounds.com/portal/view/1039229",
		},
		{
			name: "sfw portal view",
			url:  "https://www.newgrounds.com/portal/view/872806",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variants, site, err := ExtractStreamVariants(tt.url)
			if err != nil {
				t.Fatalf("ExtractStreamVariants() error = %v", err)
			}
			if site != SiteNewgrounds {
				t.Fatalf("site = %q", site)
			}
			if len(variants) == 0 {
				t.Fatal("expected variants")
			}
			if !strings.Contains(variants[0].URL, "ungrounded.net") && !strings.Contains(variants[0].URL, "ngfiles.com") {
				t.Fatalf("unexpected media URL: %q", variants[0].URL)
			}
		})
	}
}
