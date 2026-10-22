package embed

import (
	"strings"
	"testing"
)

func TestParseJWPlayerPage(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{
			name: "noodlemagazine subdomain",
			url:  "https://hot.noodlemagazine.com/watch/-136203049_456239707",
		},
		{
			name: "ukdevilz subdomain",
			url:  "https://w0w.ukdevilz.com/watch/-204376816_456239053",
		},
		{
			name: "noodlemagazine apex",
			url:  "https://noodlemagazine.com/watch/-136203049_456239707",
		},
		{
			name: "mat6tube subdomain",
			url:  "https://wow.mat6tube.com/watch/-165330904_456245253",
		},
		{
			name: "mat6tube apex",
			url:  "https://mat6tube.com/watch/-165330904_456245253",
		},
		{
			name:    "invalid path",
			url:     "https://hot.noodlemagazine.com/video/123",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "wrong host falls through to playerjs",
			url:     "https://example.com/watch/-136203049_456239707",
			wantErr: nil,
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
			if tt.name == "wrong host falls through to playerjs" {
				if site != SitePlayerJS {
					t.Fatalf("site = %q, want %q", site, SitePlayerJS)
				}
				return
			}
			if site != SiteJWPlayer {
				t.Fatalf("site = %q, want %q", site, SiteJWPlayer)
			}
			if src != "" {
				t.Fatalf("iframe src = %q, want empty", src)
			}
		})
	}
}

func TestExtractJWPlayerStreamVariantsFromHTML(t *testing.T) {
	html := `
<link rel="preload" as="script" href="/static/players/8.34.5/jwplayer.js?v=1">
<script>
window.playlist = {"sources":[
  {"file":"https://cdn77-pr.pvvstream.pro/videos/480/-204376816_456239053.mp4?secure=abc","label":"480","type":"mp4","default":true},
  {"file":"https://cdn77-pr.pvvstream.pro/videos/360/-204376816_456239053.mp4?secure=def","label":"360","type":"mp4"}
]};
</script>`

	variants, err := extractJWPlayerStreamVariantsFromHTML(html)
	if err != nil {
		t.Fatalf("extractJWPlayerStreamVariantsFromHTML() error = %v", err)
	}
	if len(variants) != 2 {
		t.Fatalf("variants = %#v", variants)
	}

	byHeight := map[int]string{}
	for _, v := range variants {
		byHeight[v.Height] = v.URL
	}
	if byHeight[480] == "" || byHeight[360] == "" {
		t.Fatalf("variants = %#v", variants)
	}
}

func TestExtractJWPlayerDurationFromHTML(t *testing.T) {
	html := `<meta property="video:duration" content="6011"/>`
	match := jwVideoDurationRe.FindStringSubmatch(html)
	if len(match) != 2 || match[1] != "6011" {
		t.Fatalf("duration match = %#v", match)
	}
}

func TestSupportsNativeStreamingJWPlayer(t *testing.T) {
	if !SupportsNativeStreaming(SiteJWPlayer) {
		t.Fatal("expected jwplayer native streaming support")
	}
	if !SupportsSpriteGeneration("https://hot.noodlemagazine.com/watch/-136203049_456239707") {
		t.Fatal("expected sprite generation for jwplayer pages")
	}
}

func TestIsAllowedProxyURLPvvStream(t *testing.T) {
	if !IsAllowedProxyURL("https://cdn77-pr.pvvstream.pro/videos/480/-204376816_456239053.mp4?secure=abc") {
		t.Fatal("expected pvvstream CDN to be allowed")
	}
}

func TestLiveJWPlayerExtract(t *testing.T) {
	if testing.Short() {
		t.Skip("skip live network test")
	}

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "noodlemagazine",
			url:  "https://hot.noodlemagazine.com/watch/-136203049_456239707",
		},
		{
			name: "ukdevilz",
			url:  "https://w0w.ukdevilz.com/watch/-204376816_456239053",
		},
		{
			name: "mat6tube",
			url:  "https://wow.mat6tube.com/watch/-165330904_456245253",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variants, site, err := ExtractStreamVariants(tt.url)
			if err != nil {
				t.Fatalf("ExtractStreamVariants() error = %v", err)
			}
			if site != SiteJWPlayer {
				t.Fatalf("site = %q", site)
			}
			if len(variants) == 0 {
				t.Fatal("expected variants")
			}
			if !strings.Contains(variants[0].URL, "pvvstream.pro") {
				t.Fatalf("unexpected media URL: %q", variants[0].URL)
			}

			duration, site, err := ExtractVideoDuration(tt.url)
			if err != nil {
				t.Fatalf("ExtractVideoDuration() error = %v", err)
			}
			if site != SiteJWPlayer {
				t.Fatalf("duration site = %q", site)
			}
			if duration <= 0 {
				t.Fatalf("duration = %v", duration)
			}
		})
	}
}
