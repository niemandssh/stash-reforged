package scraper

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/stashapp/stash/pkg/embed"
	"github.com/stashapp/stash/pkg/models"
)

const NewgroundsScraperID = "Newgrounds"

var (
	newgroundsURLRe = regexp.MustCompile(`(?i)newgrounds\.com/`)
	newgroundsFilenameIDRe = regexp.MustCompile(`^([^\[]+\[)+|(\]\.).*`)
)

type newgroundsScraper struct {
	globalConfig GlobalConfig
	config       *config
}

func getNewgroundsScraper(globalConfig GlobalConfig) scraper {
	return newgroundsScraper{
		globalConfig: globalConfig,
		config:       loadNewgroundsScraperConfig(globalConfig),
	}
}

func loadNewgroundsScraperConfig(globalConfig GlobalConfig) *config {
	path := globalConfig.GetScrapersPath()
	if path == "" {
		return nil
	}

	var found *config
	_ = filepath.Walk(path, func(fp string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Base(fp) != "Newgrounds.yml" {
			return nil
		}

		conf, loadErr := loadConfigFromYAMLFile(fp)
		if loadErr == nil {
			found = conf
		}
		return nil
	})

	return found
}

func (s newgroundsScraper) spec() Scraper {
	return Scraper{
		ID:   NewgroundsScraperID,
		Name: "Newgrounds",
		Scene: &ScraperSpec{
			Urls:             []string{"newgrounds.com/"},
			SupportedScrapes: []ScrapeType{ScrapeTypeURL, ScrapeTypeFragment},
		},
	}
}

func (s newgroundsScraper) supports(ty ScrapeContentType) bool {
	return ty == ScrapeContentTypeScene
}

func (s newgroundsScraper) supportsURL(url string, ty ScrapeContentType) bool {
	return ty == ScrapeContentTypeScene && newgroundsURLRe.MatchString(url)
}

func (s newgroundsScraper) viaURL(ctx context.Context, client *http.Client, url string, ty ScrapeContentType) (ScrapedContent, error) {
	if ty != ScrapeContentTypeScene {
		return nil, ErrNotSupported
	}

	return s.scrapeSceneURL(ctx, client, url)
}

func (s newgroundsScraper) viaFragment(ctx context.Context, client *http.Client, input Input) (ScrapedContent, error) {
	if input.Scene == nil {
		return nil, ErrNotSupported
	}

	url := sceneURLFromInput(*input.Scene)
	if url == "" {
		return nil, ErrNotSupported
	}

	return s.scrapeSceneURL(ctx, client, url)
}

func (s newgroundsScraper) viaScene(ctx context.Context, client *http.Client, scene *models.Scene) (*models.ScrapedScene, error) {
	url := ""
	if len(scene.URLs.List()) > 0 {
		url = scene.URLs.List()[0]
	}
	if url == "" {
		url = newgroundsURLFromPath(scene.Path)
	}
	if url == "" {
		return nil, ErrNotSupported
	}

	return s.scrapeSceneURL(ctx, client, url)
}

func (s newgroundsScraper) scrapeSceneURL(ctx context.Context, client *http.Client, url string) (*models.ScrapedScene, error) {
	var pageHTML string
	if html, err := s.fetchPageHTML(ctx, client, url); err == nil {
		pageHTML = html
	}

	meta, err := embed.ScrapeNewgroundsScene(url, pageHTML)
	if err != nil {
		return nil, err
	}

	return newgroundsSceneFromMetadata(meta), nil
}

func (s newgroundsScraper) fetchPageHTML(ctx context.Context, client *http.Client, pageURL string) (string, error) {
	if s.config == nil {
		return "", ErrNotSupported
	}

	r, err := loadURL(ctx, pageURL, client, *s.config, s.globalConfig)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func newgroundsSceneFromMetadata(meta *embed.SceneMetadata) *models.ScrapedScene {
	scene := &models.ScrapedScene{}
	if meta.Title != "" {
		title := meta.Title
		scene.Title = &title
	}
	if meta.URL != "" {
		sceneURL := meta.URL
		scene.URL = &sceneURL
	}
	if meta.Date != "" {
		date := meta.Date
		scene.Date = &date
	}
	if meta.Image != "" {
		image := meta.Image
		scene.Image = &image
	}
	if meta.Details != "" {
		details := meta.Details
		scene.Details = &details
	}
	if meta.Studio != "" {
		scene.Studio = &models.ScrapedStudio{
			Name: meta.Studio,
		}
	}
	for _, tag := range meta.Tags {
		scene.Tags = append(scene.Tags, &models.ScrapedTag{
			Name: tag,
		})
	}

	return scene
}

func sceneURLFromInput(scene models.ScrapedSceneInput) string {
	if scene.URL != nil && *scene.URL != "" {
		return *scene.URL
	}
	if len(scene.URLs) > 0 && scene.URLs[0] != "" {
		return scene.URLs[0]
	}
	return ""
}

func newgroundsURLFromPath(path string) string {
	id := newgroundsIDFromFilename(path)
	if id == "" {
		return ""
	}
	return "https://www.newgrounds.com/portal/view/" + id
}

func newgroundsIDFromFilename(path string) string {
	filename := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	filename = newgroundsFilenameIDRe.ReplaceAllString(filename, "")
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return ""
	}
	return filename
}
