package embed

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleXVideosEmbedHTML = `<html><body><script>
if (html5player) {
html5player.setVideoTitle('Test Video');
html5player.setVideoUrlLow('https://cdn.example.com/video_240p.mp4');
html5player.setVideoUrlHigh('https://cdn.example.com/video_360p.mp4');
html5player.setVideoHLS('https://cdn.example.com/hls.m3u8');
html5player.setVideoURL('/video.abc123/test_video');
}
</script></body></html>`

const sampleXVideosMasterM3U8 = `#EXTM3U
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=1327104,RESOLUTION=1280x720,NAME="720p"
hls-720p-test.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=763904,RESOLUTION=854x480,NAME="480p"
hls-480p-test.m3u8`

const sampleXVideosVideoPageHTML = `<html><head>
<meta property="og:duration" content="321" />
</head></html>`

func TestExtractHTML5PlayerStreamVariants(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/hls.m3u8"):
			_, _ = w.Write([]byte(sampleXVideosMasterM3U8))
		default:
			html := strings.Replace(sampleXVideosEmbedHTML, "https://cdn.example.com/hls.m3u8", server.URL+"/hls.m3u8", 1)
			_, _ = w.Write([]byte(html))
		}
	}))
	defer server.Close()

	variants, err := extractHTML5PlayerStreamVariants(server.URL, SiteXVideos)
	if err != nil {
		t.Fatalf("extractHTML5PlayerStreamVariants() error = %v", err)
	}

	byHeight := map[int]string{}
	for _, v := range variants {
		byHeight[v.Height] = v.URL
	}

	if byHeight[720] == "" || byHeight[480] == "" {
		t.Fatalf("missing expected heights: %#v", byHeight)
	}
	if !strings.Contains(byHeight[720], "hls-720p-test.m3u8") {
		t.Fatalf("720p URL = %q", byHeight[720])
	}
}

func TestExtractHTML5PlayerDuration(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/video.abc123"):
			_, _ = w.Write([]byte(sampleXVideosVideoPageHTML))
		default:
			html := strings.Replace(sampleXVideosEmbedHTML, "/video.abc123/test_video", server.URL+"/video.abc123/test_video", 1)
			_, _ = w.Write([]byte(html))
		}
	}))
	defer server.Close()

	duration, err := extractHTML5PlayerDuration(server.URL, server.URL, SiteXVideos)
	if err != nil {
		t.Fatalf("extractHTML5PlayerDuration() error = %v", err)
	}
	if duration != 321 {
		t.Fatalf("duration = %v, want 321", duration)
	}
}

func TestSupportsNativeStreaming(t *testing.T) {
	if !SupportsNativeStreaming(SiteXVideos) {
		t.Fatal("expected xvideos native streaming support")
	}
	if SupportsNativeStreaming(SiteXHamster) {
		t.Fatal("did not expect xhamster native streaming support")
	}
}

func TestStreamHTTPHeadersForURL(t *testing.T) {
	headers := StreamHTTPHeadersForURL("https://www.xvideos.com/video.abc/test")
	if !strings.Contains(headers, "Referer: https://www.xvideos.com/") {
		t.Fatalf("headers = %q", headers)
	}
}
