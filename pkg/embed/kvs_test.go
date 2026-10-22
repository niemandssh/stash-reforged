package embed

import (
	"strings"
	"testing"
)

func TestKVSGetRealURL(t *testing.T) {
	obfuscated := "function/0/https://www.camsclips.net/get_file/1/7e0ac2caa748d9013f1843311a18f63dc71c4018bd/220000/220150/220150.mp4/?br=1346&embed=true"
	license := "$564813918634199"
	got := kvsGetRealURL(obfuscated, license)
	wantPrefix := "https://www.camsclips.net/get_file/1/3aed33f1c01a7da09f213168c8441a78"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("kvsGetRealURL() = %q, want prefix %q", got, wantPrefix)
	}
	if strings.HasPrefix(got, "function/0/") {
		t.Fatalf("url still obfuscated: %q", got)
	}
}

func TestExtractKVSStreamVariantsFromHTML(t *testing.T) {
	html := `
<script src="https://www.camsclips.net/player/kt_player.js?v=6.2.1"></script>
<script>
var flashvars = {
  video_id: '220150',
  license_code: '$564813918634199',
  video_url: 'function/0/https://www.camsclips.net/get_file/1/7e0ac2caa748d9013f1843311a18f63dc71c4018bd/220000/220150/220150.mp4/?br=1346&embed=true',
  video_url_text: '720p',
  video_url_hd: '1'
};
kt_player('kt_player', 'swf', '100%', '100%', flashvars);
</script>`

	if !isKVSPage(html) {
		t.Fatal("expected KVS page detection")
	}

	variants, err := extractKVSStreamVariantsFromHTML("https://www.camsclips.net/embed/220150", html)
	if err != nil {
		t.Fatalf("extractKVSStreamVariantsFromHTML() error = %v", err)
	}
	if len(variants) != 1 {
		t.Fatalf("len(variants) = %d", len(variants))
	}
	if variants[0].Height != 720 {
		t.Fatalf("height = %d, want 720", variants[0].Height)
	}
	if !strings.Contains(variants[0].URL, "/get_file/1/3aed33f1c01a7da09f213168c8441a78") {
		t.Fatalf("url = %q", variants[0].URL)
	}
}

func TestSupportsNativeStreamingKVS(t *testing.T) {
	if !SupportsNativeStreaming(SiteKVS) {
		t.Fatal("expected KVS native streaming")
	}
}

func TestLiveCamsClipsKVSExtract(t *testing.T) {
	if testing.Short() {
		t.Skip("skip live network test")
	}

	page := "https://www.camsclips.net/embed/220150"
	variants, site, err := ExtractStreamVariants(page)
	if err != nil {
		t.Fatalf("ExtractStreamVariants() error = %v", err)
	}
	if site != SiteKVS {
		t.Fatalf("site = %q, want %q", site, SiteKVS)
	}
	if len(variants) == 0 {
		t.Fatal("expected variants")
	}
	if !strings.Contains(variants[0].URL, "/get_file/") {
		t.Fatalf("url = %q", variants[0].URL)
	}
	t.Logf("media URL: %s", variants[0].URL)
}
