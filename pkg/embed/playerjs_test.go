package embed

import (
	"strings"
	"testing"
)

func TestParsePlayerJSPage(t *testing.T) {
	src, site, err := Parse("https://videos.example.com/watch/123")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if site != SitePlayerJS {
		t.Fatalf("site = %q, want %q", site, SitePlayerJS)
	}
	if src != "" {
		t.Fatalf("iframe src = %q, want empty", src)
	}
}

func TestExtractPlayerJSFiles(t *testing.T) {
	html := `
<script src="/playerjs.js"></script>
<script>
var player = new Playerjs({
  id: "player",
  file: "https://f003.backblazeb2.com/file/bucket/video.mp4?Authorization=token",
  duration: 120
});
</script>`

	files := extractPlayerJSFiles(html)
	if len(files) != 1 {
		t.Fatalf("files = %#v", files)
	}
	if !strings.Contains(files[0], "backblazeb2.com") {
		t.Fatalf("file = %q", files[0])
	}
}

func TestExtractPlayerJSFilesPlaylist(t *testing.T) {
	html := `
Playerjs({
  id:"player",
  file:[
    {"title":"720p","file":"https://cdn.example.com/a/720.mp4"},
    {"title":"1080p","file":"https://cdn.example.com/a/1080.mp4"}
  ]
});`

	files := extractPlayerJSFiles(html)
	if len(files) != 2 {
		t.Fatalf("files = %#v", files)
	}
}

func TestExtractPlayerJSFilesSkipsJSDynamic(t *testing.T) {
	html := `new Playerjs({id:"player", file:"js:GetFile()"});`
	if files := extractPlayerJSFiles(html); len(files) != 0 {
		t.Fatalf("expected no files, got %#v", files)
	}
}

func TestFindPlayerJSHTMLEmbeds(t *testing.T) {
	html := `{"src":"<iframe src=\"https:\/\/rusosky.com\/playerjs\/playerjs.html?file=mp4\/1000\/1996.mp4\"><\/iframe>","quality":"720p"}`
	embeds := findPlayerJSHTMLEmbeds("https://rusosky.com/watch/1996", html)
	if len(embeds) != 1 {
		t.Fatalf("embeds = %#v", embeds)
	}
	if embeds[0].file != "mp4/1000/1996.mp4" {
		t.Fatalf("file = %q", embeds[0].file)
	}
	if !strings.Contains(embeds[0].playerURL, "playerjs.html?file=") {
		t.Fatalf("playerURL = %q", embeds[0].playerURL)
	}
}

func TestSupportsNativeStreamingPlayerJS(t *testing.T) {
	if !SupportsNativeStreaming(SitePlayerJS) {
		t.Fatal("expected playerjs native streaming support")
	}
	if !SupportsSpriteGeneration("https://videos.example.com/watch/1") {
		t.Fatal("expected sprite generation for playerjs pages")
	}
}

func TestLiveRusoSkyPlayerJSExtract(t *testing.T) {
	if testing.Short() {
		t.Skip("skip live network test")
	}

	url := "https://rusosky.com/watch/1996"
	variants, site, err := ExtractStreamVariants(url)
	if err != nil {
		t.Fatalf("ExtractStreamVariants() error = %v", err)
	}
	if site != SitePlayerJS {
		t.Fatalf("site = %q", site)
	}
	if len(variants) == 0 {
		t.Fatal("expected variants")
	}
	if !strings.Contains(variants[0].URL, "backblazeb2.com") && !strings.HasSuffix(strings.Split(variants[0].URL, "?")[0], ".mp4") {
		t.Fatalf("unexpected media URL: %q", variants[0].URL)
	}
	t.Logf("signed URL: %s", variants[0].URL)
}
