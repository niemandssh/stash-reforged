package embed

import (
	"strings"
	"testing"
)

func TestParseDirectMP4(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{
			name: "https mp4",
			url:  "https://cdn.example.com/videos/clip.mp4",
		},
		{
			name: "https mp4 with query",
			url:  "https://cdn.example.com/videos/clip.mp4?token=abc",
		},
		{
			name: "webm",
			url:  "https://cdn.example.com/a/b/video.webm",
		},
		{
			name: "html page becomes playerjs candidate",
			url:  "https://cdn.example.com/videos/clip.html",
		},
		{
			name: "no extension becomes playerjs candidate",
			url:  "https://cdn.example.com/videos/clip",
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
			wantSite := SiteDirect
			if strings.HasSuffix(tt.name, "playerjs candidate") {
				wantSite = SitePlayerJS
			}
			if site != wantSite {
				t.Fatalf("site = %q, want %q", site, wantSite)
			}
			if src != "" {
				t.Fatalf("iframe src = %q, want empty", src)
			}
		})
	}
}

func TestIsProgressiveMediaURLTrailingSlash(t *testing.T) {
	u := "https://www.camsclips.net/get_file/1/abc/220000/220150/220150.mp4/?br=1346&embed=true"
	if !IsProgressiveMediaURL(u) {
		t.Fatalf("expected progressive media for trailing-slash mp4: %q", u)
	}
}

func TestExtractDirectStreamVariants(t *testing.T) {
	url := "https://cdn.example.com/path/1080p_movie.mp4"
	variants, site, err := ExtractStreamVariants(url)
	if err != nil {
		t.Fatalf("ExtractStreamVariants() error = %v", err)
	}
	if site != SiteDirect {
		t.Fatalf("site = %q", site)
	}
	if len(variants) != 1 {
		t.Fatalf("len(variants) = %d", len(variants))
	}
	if variants[0].URL != url {
		t.Fatalf("url = %q", variants[0].URL)
	}
	if variants[0].Height != 1080 {
		t.Fatalf("height = %d, want 1080", variants[0].Height)
	}
}

func TestIsAllowedProxyURLForSceneDirect(t *testing.T) {
	embedURL := "https://cdn.example.com/videos/clip.mp4"
	if !IsAllowedProxyURLForScene(embedURL, embedURL) {
		t.Fatal("expected scene-bound direct URL to be allowed")
	}
	if IsAllowedProxyURLForScene("https://evil.example.com/other.mp4", embedURL) {
		t.Fatal("did not expect unrelated URL")
	}
	if IsAllowedProxyURL(embedURL) {
		t.Fatal("direct arbitrary host should not be globally allowed")
	}
}

func TestSupportsNativeStreamingDirect(t *testing.T) {
	if !SupportsNativeStreaming(SiteDirect) {
		t.Fatal("expected direct media native streaming support")
	}
}
