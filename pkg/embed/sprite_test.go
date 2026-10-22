package embed

import (
	"strings"
	"testing"
)

func TestSupportsSpriteGeneration(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://www.pornhub.com/view_video.php?viewkey=abc", true},
		{"https://www.pornhub.org/embed/abc", true},
		{"https://www.xvideos.com/video123/test", true},
		{"https://www.xnxx.com/video-abc123/test", true},
		{"https://xhamster.com/videos/title-xhCvkeh", true},
		{"https://cdn.example.com/videos/clip.mp4", true},
		{"https://cdn.example.com/videos/clip.webm", true},
		{"https://videos.example.com/watch/123", true},
		{"", false},
	}

	for _, tt := range tests {
		if got := SupportsSpriteGeneration(tt.url); got != tt.want {
			t.Errorf("SupportsSpriteGeneration(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestStreamHTTPHeaders(t *testing.T) {
	headers := StreamHTTPHeaders()
	if headers == "" {
		t.Fatal("expected non-empty headers")
	}
	if !strings.Contains(headers, "Referer:") || !strings.Contains(headers, "User-Agent:") {
		t.Fatalf("headers missing expected fields: %q", headers)
	}

	direct := StreamHTTPHeadersForURL("https://cdn.example.com/clip.mp4")
	if strings.Contains(direct, "Referer:") {
		t.Fatalf("direct headers should omit Referer: %q", direct)
	}
	if !strings.Contains(direct, "User-Agent:") {
		t.Fatalf("direct headers missing User-Agent: %q", direct)
	}

	media := StreamHTTPHeadersForMediaURL(
		"https://rusosky.com/watch/1",
		"https://f003.backblazeb2.com/file/bucket/a.mp4?Authorization=token",
	)
	if strings.Contains(media, "Referer:") {
		t.Fatalf("cross-host media headers should omit Referer: %q", media)
	}

	if !LooksLikeTemporaryMediaURL("https://f003.backblazeb2.com/file/x.mp4?Authorization=abc") {
		t.Fatal("expected temporary URL detection")
	}
}
