package embed

import (
	"strings"
	"testing"
)

func TestDecodeXHamsterURL(t *testing.T) {
	// Encrypted HLS URL captured from a live xHamster page (algo 2).
	enc := "02db3b61b2fe17da9115c30df47dfb9f59084f78e6d220ad893ea4adc3911c2e39f12233fb3a5d7fa3acf8157ebb434106eb0d54c50ff0fadacc7e147715df68cc11f654aa875c7f8b726c9161e728d87b07179f08a7cfd44194c680e7199b27407a8bc1f88c3f97fa78f8bb2eaa680af4198c13003e48aeca83fdf9b00a628bb97e803cdf7eb62228799e42847ea75d85a419bd9d6b5db87b35190240b19f73167e417e606c109420f1c371b8376e6961719d791502aa85fe9579b695b11aab570738674d1cda691eeaf410d9751549c7c777d6a208dd44874438eb73e223cfe1e3bc7545788b76663f8786559b54"

	got := decodeXHamsterURL(enc)
	if !strings.HasPrefix(got, "https://") {
		t.Fatalf("decodeXHamsterURL() = %q, want https URL", got)
	}
	if !strings.Contains(got, ".m3u8") {
		t.Fatalf("decodeXHamsterURL() = %q, want m3u8", got)
	}
	if !strings.Contains(got, "xhcdn.com") {
		t.Fatalf("decodeXHamsterURL() = %q, want xhcdn.com host", got)
	}
}

func TestDecodeXHamsterURLPassthrough(t *testing.T) {
	raw := "https://video7.xhcdn.com/test.m3u8"
	if got := decodeXHamsterURL(raw); got != raw {
		t.Fatalf("passthrough = %q, want %q", got, raw)
	}
}

func TestExtractXHamsterPlayerSettings(t *testing.T) {
	html := `
<script>window.initials={"xplayerSettings":{"dimensions":{},"events":[]},"other":1};
window.initials2={"xplayerSettings":{"videoId":123,"duration":606,"sources":{"hls":{"h264":{"url":"https://video.xhcdn.com/a.m3u8"}},"standard":{"h264":[{"url":"https://video.xhcdn.com/480p.mp4","quality":"480p"}]}}};
</script>`

	settings, err := extractXHamsterPlayerSettings(html)
	if err != nil {
		t.Fatalf("extractXHamsterPlayerSettings() error = %v", err)
	}
	if settings.Duration != 606 {
		t.Fatalf("duration = %v, want 606", settings.Duration)
	}
	if settings.Sources.HLS["h264"].URL == "" {
		t.Fatal("expected hls h264 url")
	}
	if len(settings.Sources.Standard["h264"]) != 1 {
		t.Fatalf("standard h264 len = %d", len(settings.Sources.Standard["h264"]))
	}
}

func TestNormalizeXHamsterPageURL(t *testing.T) {
	got := normalizeXHamsterPageURL("https://xhamster.com/videos/19-years-old-trans-has-her-first-threesome-xhCvkeh?utm=1")
	want := "https://xhamster.com/videos/19-years-old-trans-has-her-first-threesome-xhCvkeh"
	if got != want {
		t.Fatalf("normalizeXHamsterPageURL() = %q, want %q", got, want)
	}
}

func TestSiteForProxyHostXHamsterCDN(t *testing.T) {
	if got := siteForProxyHost("video7.xhcdn.com"); got != SiteXHamster {
		t.Fatalf("siteForProxyHost(xhcdn) = %q, want %q", got, SiteXHamster)
	}
}
