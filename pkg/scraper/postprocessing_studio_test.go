package scraper

import (
	"context"
	"strings"
	"testing"

	"github.com/stashapp/stash/pkg/match"
	"github.com/stashapp/stash/pkg/models"
)

func TestTitleCandidatesFromScene(t *testing.T) {
	scene := &models.Scene{
		Title: "Pregnant Boy Girl Creampie & Blowjob",
		Path:  "/media/[ManyVids.com] ariana simon - scene title.mp4",
	}

	got := titleCandidatesFromScene(scene)
	if len(got) != 2 {
		t.Fatalf("titleCandidatesFromScene() = %v, want 2 candidates", got)
	}
	if got[0] != scene.Title {
		t.Fatalf("first candidate = %q, want scene title", got[0])
	}
	if got[1] != "[ManyVids.com] ariana simon - scene title" {
		t.Fatalf("second candidate = %q, want filename without extension", got[1])
	}
}

func TestScrapedSceneTitleCandidatesUsesFallbacks(t *testing.T) {
	scrapedTitle := "Pregnant Boy Girl Creampie & Blowjob"
	scene := models.ScrapedScene{
		Title: &scrapedTitle,
	}

	candidates := scrapedSceneTitleCandidates(scene, postScrapeOptions{
		titleCandidates: []string{"[ManyVids.com] ariana simon - scene title"},
	})

	if len(candidates) != 2 {
		t.Fatalf("scrapedSceneTitleCandidates() = %v, want 2 candidates", candidates)
	}
	if candidates[1] != "[ManyVids.com] ariana simon - scene title" {
		t.Fatalf("fallback candidate = %q", candidates[1])
	}
}

func TestStudioFromFallbackTitleCandidate(t *testing.T) {
	qb := &studioTitleTestFinder{
		studios: []*models.Studio{
			{
				ID:   11,
				Name: "ManyVids",
				URL:  "https://manyvids.com",
			},
		},
	}

	for _, title := range []string{
		"Pregnant Boy Girl Creampie & Blowjob",
		"[ManyVids.com] ariana simon - Pregnant Boy Girl Creampie & Blowjob [1080p]",
	} {
		got, ok := match.StudioFromTitle(context.Background(), qb, title)
		if title == "Pregnant Boy Girl Creampie & Blowjob" {
			if ok {
				t.Fatalf("StudioFromTitle(%q) should not match", title)
			}
			continue
		}

		if !ok || got == nil || got.StoredID == nil || *got.StoredID != "11" {
			t.Fatalf("StudioFromTitle(%q) = %#v, want ManyVids with stored_id 11", title, got)
		}
	}
}

type studioTitleTestFinder struct {
	studios []*models.Studio
}

func (t *studioTitleTestFinder) Query(ctx context.Context, filter *models.StudioFilterType, findFilter *models.FindFilterType) ([]*models.Studio, int, error) {
	if filter == nil {
		return nil, 0, nil
	}

	var matches []*models.Studio
	for _, st := range t.studios {
		if filter.URL != nil && filter.URL.Modifier == models.CriterionModifierIncludes {
			if strings.Contains(strings.ToLower(st.URL), strings.ToLower(filter.URL.Value)) {
				matches = append(matches, st)
			}
		}
	}

	return matches, len(matches), nil
}

func (t *studioTitleTestFinder) QueryCount(ctx context.Context, filter *models.StudioFilterType, findFilter *models.FindFilterType) (int, error) {
	_, count, err := t.Query(ctx, filter, findFilter)
	return count, err
}

func (t *studioTitleTestFinder) FindByStashID(ctx context.Context, stashID models.StashID) ([]*models.Studio, error) {
	return nil, nil
}
