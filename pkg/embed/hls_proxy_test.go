package embed

import (
	"strings"
	"testing"
)

func TestRewriteM3U8(t *testing.T) {
	manifest := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-KEY:METHOD=AES-128,URI="key.bin"
index-v1-a1.m3u8?validfrom=1&validto=2
`

	baseURL := "https://ee-h.phncdn.com/hls/videos/test/master.m3u8?token=abc"
	proxyPath := "/scene/42/embed/proxy"

	got, err := RewriteM3U8([]byte(manifest), baseURL, proxyPath)
	if err != nil {
		t.Fatalf("RewriteM3U8() error = %v", err)
	}

	output := string(got)
	if !strings.Contains(output, proxyPath+"?u=") {
		t.Fatalf("RewriteM3U8() = %q, expected proxied URLs", output)
	}
	if strings.Contains(output, "index-v1-a1.m3u8?validfrom=1") {
		t.Fatalf("RewriteM3U8() should rewrite media playlist URI: %q", output)
	}
}

func TestIsAllowedProxyURL(t *testing.T) {
	tests := []struct {
		url   string
		allow bool
	}{
		{"https://ee-h.phncdn.com/hls/test.m3u8", true},
		{"https://www.pornhub.com/video/get_media", true},
		{"https://hls-cdn77.xnxx-cdn.com/abc/0/hls.m3u8", true},
		{"https://mp4-cdn77.xnxx-cdn.com/abc/0/video_360p.mp4", true},
		{"https://cdn77-vid.xvideos-cdn.com/hls.m3u8", true},
		{"https://video7.xhcdn.com/key=abc/media=hls4/test.m3u8", true},
		{"https://cdn77-pr.pvvstream.pro/videos/480/-204376816_456239053.mp4?secure=abc", true},
		{"https://uploads.ungrounded.net/alternate/7870000/7870047_alternate_366343.720p.mp4?1782611121", true},
		{"https://www.xnxx.com/embedframe/abc", true},
		{"https://evil.example.com/video.ts", false},
		{"http://ee-h.phncdn.com/video.ts", false},
	}

	for _, tt := range tests {
		if got := IsAllowedProxyURL(tt.url); got != tt.allow {
			t.Fatalf("IsAllowedProxyURL(%q) = %v, want %v", tt.url, got, tt.allow)
		}
	}
}

func TestParseMasterVariants(t *testing.T) {
	manifest := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=2048000,RESOLUTION=1280x720
720p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=4096000,RESOLUTION=1920x1080
1080p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=819200,RESOLUTION=640x360
360p.m3u8
`
	baseURL := "https://ee-h.phncdn.com/hls/videos/test/master.m3u8"

	variants, err := ParseMasterVariants([]byte(manifest), baseURL)
	if err != nil {
		t.Fatalf("ParseMasterVariants() error = %v", err)
	}
	if len(variants) != 3 {
		t.Fatalf("ParseMasterVariants() len = %d, want 3", len(variants))
	}

	got1080 := false
	for _, v := range variants {
		if v.Height == 1080 {
			got1080 = true
			if !strings.HasSuffix(v.URL, "1080p.m3u8") {
				t.Fatalf("1080p URL = %q", v.URL)
			}
		}
	}
	if !got1080 {
		t.Fatal("expected 1080p variant")
	}
}

func TestSelectVariantDefault1080p(t *testing.T) {
	variants := []HLSVariant{
		{Height: 720, Bandwidth: 2048000, URL: "https://example.com/720.m3u8"},
		{Height: 480, Bandwidth: 1024000, URL: "https://example.com/480.m3u8"},
	}

	selected, err := SelectVariant(variants, DefaultPreferredHeight)
	if err != nil {
		t.Fatalf("SelectVariant() error = %v", err)
	}
	if selected.Height != 720 {
		t.Fatalf("SelectVariant() height = %d, want 720 (best below 1080)", selected.Height)
	}

	variants = append(variants, HLSVariant{Height: 1080, Bandwidth: 4096000, URL: "https://example.com/1080.m3u8"})
	selected, err = SelectVariant(variants, DefaultPreferredHeight)
	if err != nil {
		t.Fatalf("SelectVariant() error = %v", err)
	}
	if selected.Height != 1080 {
		t.Fatalf("SelectVariant() height = %d, want 1080", selected.Height)
	}
}

func TestSortVariantsDefaultFirst(t *testing.T) {
	variants := []HLSVariant{
		{Height: 720, Bandwidth: 2048000},
		{Height: 1080, Bandwidth: 4096000},
		{Height: 480, Bandwidth: 1024000},
	}

	sorted := SortVariantsDefaultFirst(variants, DefaultPreferredHeight)
	if sorted[0].Height != 1080 {
		t.Fatalf("first variant height = %d, want 1080", sorted[0].Height)
	}
}

func TestHeightsFromVariants(t *testing.T) {
	heights := HeightsFromVariants([]HLSVariant{
		{Height: 480},
		{Height: 1080},
		{Height: 720},
		{Height: 480},
	})
	if len(heights) != 3 {
		t.Fatalf("HeightsFromVariants() len = %d, want 3", len(heights))
	}
	if heights[0] != 1080 || heights[1] != 720 || heights[2] != 480 {
		t.Fatalf("HeightsFromVariants() = %v", heights)
	}
}

func TestSelectVariantByBandwidthTier(t *testing.T) {
	variants := []HLSVariant{
		{Bandwidth: 800000, URL: "https://cdn.example.com/240p.m3u8"},
		{Bandwidth: 1600000, URL: "https://cdn.example.com/480p.m3u8"},
		{Bandwidth: 3200000, URL: "https://cdn.example.com/720p.m3u8"},
		{Bandwidth: 6400000, URL: "https://cdn.example.com/1080p.m3u8"},
	}

	low, err := SelectVariant(variants, 240)
	if err != nil {
		t.Fatalf("SelectVariant(240) error = %v", err)
	}
	if !strings.Contains(low.URL, "240p") {
		t.Fatalf("SelectVariant(240) URL = %q", low.URL)
	}

	high, err := SelectVariant(variants, 1080)
	if err != nil {
		t.Fatalf("SelectVariant(1080) error = %v", err)
	}
	if !strings.Contains(high.URL, "1080p") {
		t.Fatalf("SelectVariant(1080) URL = %q", high.URL)
	}
	if low.URL == high.URL {
		t.Fatal("expected different variants for 240p and 1080p")
	}
}

func TestInferHeightFromURL(t *testing.T) {
	if got := inferHeightFromURL("https://cdn.example.com/hls/1080P_2000K_224p.m3u8"); got != 1080 {
		t.Fatalf("inferHeightFromURL() = %d, want 1080", got)
	}
}
