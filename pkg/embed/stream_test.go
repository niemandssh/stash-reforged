package embed

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleEmbedHTML = `<html><body>
<script>
var flashvars = {"video_duration":522,"mediaDefinitions":[
  {"format":"mp4","videoUrl":"https://example.com/video.mp4"},
  {"format":"hls","videoUrl":"https://cdn.example.com/master.m3u8?token=abc"}
]},
utmSource = document.referrer.split('/')[2];
</script>
</body></html>`

func TestExtractFlashvarsJSON(t *testing.T) {
	got, err := extractFlashvarsJSON(sampleEmbedHTML)
	if err != nil {
		t.Fatalf("extractFlashvarsJSON() error = %v", err)
	}

	want := `{"video_duration":522,"mediaDefinitions":[
  {"format":"mp4","videoUrl":"https://example.com/video.mp4"},
  {"format":"hls","videoUrl":"https://cdn.example.com/master.m3u8?token=abc"}
]}`
	if got != want {
		t.Fatalf("extractFlashvarsJSON() = %q, want %q", got, want)
	}

	viewHTML := `<script>var flashvars_421235491 = {"video_duration":522,"mediaDefinitions":[
  {"format":"hls","quality":"1080","videoUrl":"https://cdn.example.com/1080.m3u8"}
]};</script>`
	got, err = extractFlashvarsJSON(viewHTML)
	if err != nil {
		t.Fatalf("extractFlashvarsJSON(view) error = %v", err)
	}
	if !strings.Contains(got, `"quality":"1080"`) {
		t.Fatalf("extractFlashvarsJSON(view) = %q", got)
	}
}

func TestExpandURLSetVariants(t *testing.T) {
	urlset := "https://ev-h.phncdn.com/hls/videos/202401/16/446618441/,1080P_4000K,720P_4000K,480P_2000K,240P_1000K,_446618441.mp4.urlset/index-f1-v1-a1.m3u8?validfrom=1"

	variants, err := expandURLSetVariants(urlset)
	if err != nil {
		t.Fatalf("expandURLSetVariants() error = %v", err)
	}
	if len(variants) != 4 {
		t.Fatalf("expandURLSetVariants() len = %d, want 4", len(variants))
	}

	heights := map[int]string{}
	for _, v := range variants {
		heights[v.Height] = v.URL
	}

	want1080 := "https://ev-h.phncdn.com/hls/videos/202401/16/446618441/1080P_4000K_446618441.mp4/index-v1-a1.m3u8?validfrom=1"
	if heights[1080] != want1080 {
		t.Fatalf("1080p URL = %q, want %q", heights[1080], want1080)
	}
	if heights[720] == "" || heights[480] == "" || heights[240] == "" {
		t.Fatalf("missing expected heights: %#v", heights)
	}
	if heights[1080] == heights[480] {
		t.Fatal("expected different URLs per quality")
	}
}

func TestResolveHLSFromMediaURL(t *testing.T) {
	progressive := "https://ee.phncdn.com/videos/202212/12/421235491/480P_2000K_421235491.mp4?token=abc"
	got := resolveHLSFromMediaURL(progressive)
	want := "https://ee-h.phncdn.com/hls/videos/202212/12/421235491/480P_2000K_421235491.mp4/master.m3u8?token=abc"
	if got != want {
		t.Fatalf("resolveHLSFromMediaURL(progressive) = %q, want %q", got, want)
	}

	hlsCDN := "https://ee-h.phncdn.com/hls/videos/202212/12/421235491/480P_2000K_421235491.mp4?token=abc"
	got = resolveHLSFromMediaURL(hlsCDN)
	want = "https://ee-h.phncdn.com/hls/videos/202212/12/421235491/480P_2000K_421235491.mp4/master.m3u8?token=abc"
	if got != want {
		t.Fatalf("resolveHLSFromMediaURL(hlsCDN) = %q, want %q", got, want)
	}

	existing := "https://ee-h.phncdn.com/hls/videos/test/480P_2000K_1.mp4/master.m3u8?token=abc"
	if resolveHLSFromMediaURL(existing) != existing {
		t.Fatalf("resolveHLSFromMediaURL() should preserve existing m3u8 URL")
	}
}

func TestExtractPornHubStreamVariantsFromFlashvars(t *testing.T) {
	html := `<html><body><script>
var flashvars = {"video_duration":522,"mediaDefinitions":[
  {"format":"mp4","quality":1080,"videoUrl":"https://ee.phncdn.com/videos/202401/16/1/1080P_4000K_1.mp4"},
  {"format":"mp4","quality":720,"videoUrl":"https://ee.phncdn.com/videos/202401/16/1/720P_2000K_1.mp4"},
  {"format":"hls","quality":480,"videoUrl":"https://ee-h.phncdn.com/hls/videos/202401/16/446618441/,480P_1000K,240P_500K,_1.mp4.urlset/master.m3u8"}
]},
utmSource = document.referrer.split('/')[2];
</script></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	variants, err := extractPornHubStreamVariants(server.URL, server.URL)
	if err != nil {
		t.Fatalf("extractPornHubStreamVariants() error = %v", err)
	}

	byHeight := map[int]string{}
	for _, v := range variants {
		byHeight[v.Height] = v.URL
	}

	if !strings.Contains(byHeight[1080], "1080P_4000K_1.mp4/index-v1-a1.m3u8") && !strings.Contains(byHeight[1080], "1080P_4000K_1.mp4/master.m3u8") {
		t.Fatalf("1080p variant = %q", byHeight[1080])
	}
	if !strings.Contains(byHeight[720], "720P_2000K_1.mp4/index-v1-a1.m3u8") && !strings.Contains(byHeight[720], "720P_2000K_1.mp4/master.m3u8") {
		t.Fatalf("720p variant = %q", byHeight[720])
	}
	if byHeight[1080] == byHeight[720] {
		t.Fatal("expected different URLs for 1080p and 720p")
	}
}

func TestExtractVideoDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleEmbedHTML))
	}))
	defer server.Close()

	got, err := fetchPornHubFlashvars(server.URL, server.URL)
	if err != nil {
		t.Fatalf("fetchPornHubFlashvars() error = %v", err)
	}
	if got.VideoDuration != 522 {
		t.Fatalf("VideoDuration = %v, want 522", got.VideoDuration)
	}
}

func TestExtractStreamURLUnsupportedSite(t *testing.T) {
	_, site, err := ExtractStreamURL("https://www.youtube.com/watch?v=abc")
	if site != SitePlayerJS {
		t.Fatalf("site = %q, want %q", site, SitePlayerJS)
	}
	if err == nil {
		t.Fatal("expected extraction error for non-playerjs page")
	}
}
