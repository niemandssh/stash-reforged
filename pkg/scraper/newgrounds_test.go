package scraper

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/embed"
	"github.com/stashapp/stash/pkg/models"
)

func TestNewgroundsBuiltinScraperByURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skip live network test")
	}

	scraper := getNewgroundsScraper(mockGlobalConfig{})
	client := &http.Client{Timeout: 15 * time.Second}

	ctx := context.Background()
	content, err := scraper.(urlScraper).viaURL(ctx, client, "https://www.newgrounds.com/portal/view/1039229", ScrapeContentTypeScene)
	if err != nil {
		t.Fatalf("viaURL() error = %v", err)
	}
	if content == nil {
		t.Fatal("viaURL() returned nil content")
	}

	scene, ok := content.(*models.ScrapedScene)
	if !ok {
		t.Fatalf("content type = %T, want *models.ScrapedScene", content)
	}
	if scene.Title == nil || *scene.Title != "I know you love plushies" {
		t.Fatalf("title = %#v", scene.Title)
	}
	if scene.Studio == nil || scene.Studio.Name != "Star-Rifle" {
		t.Fatalf("studio = %#v", scene.Studio)
	}
}

func TestNewgroundsSceneFromMetadataTags(t *testing.T) {
	meta, err := embed.ScrapeNewgroundsScene("https://www.newgrounds.com/portal/view/1039229", `<dl class="sidestats"><dt>Tags</dt><dd><a>kneesocks-daemon</a><a>nsfw</a><a>parody</a></dd></dl>`)
	if err != nil {
		t.Fatalf("ScrapeNewgroundsScene() error = %v", err)
	}

	scene := newgroundsSceneFromMetadata(meta)
	if len(scene.Tags) != 3 {
		t.Fatalf("tags = %#v", scene.Tags)
	}
	if scene.Tags[0].Name != "kneesocks-daemon" {
		t.Fatalf("first tag = %q", scene.Tags[0].Name)
	}
}
