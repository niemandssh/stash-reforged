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
}
