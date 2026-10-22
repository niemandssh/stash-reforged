package scraper

import (
	"context"
	"testing"

	"github.com/stashapp/stash/pkg/models"
)

type noopTagQueryer struct{}

func (noopTagQueryer) All(ctx context.Context) ([]*models.Tag, error) {
	return nil, nil
}

func (noopTagQueryer) Count(ctx context.Context) (int, error) {
	return 0, nil
}

func (noopTagQueryer) Create(ctx context.Context, tag *models.Tag) error {
	return nil
}

func (noopTagQueryer) Find(ctx context.Context, id int) (*models.Tag, error) {
	return nil, nil
}

func (noopTagQueryer) FindBySceneID(ctx context.Context, sceneID int) ([]*models.Tag, error) {
	return nil, nil
}

func (noopTagQueryer) FindMany(ctx context.Context, ids []int) ([]*models.Tag, error) {
	return nil, nil
}

func (noopTagQueryer) GetImage(ctx context.Context, tagID int) ([]byte, error) {
	return nil, nil
}

func (noopTagQueryer) Query(ctx context.Context, tagFilter *models.TagFilterType, findFilter *models.FindFilterType) ([]*models.Tag, int, error) {
	return nil, 0, nil
}

func (noopTagQueryer) QueryCount(ctx context.Context, tagFilter *models.TagFilterType, findFilter *models.FindFilterType) (int, error) {
	return 0, nil
}

func (noopTagQueryer) Update(ctx context.Context, tag *models.Tag) error {
	return nil
}

func (noopTagQueryer) UpdatePartial(ctx context.Context, id int, updated models.TagPartial) (*models.Tag, error) {
	return nil, nil
}

func TestPostProcessTagsSkipsNilEntries(t *testing.T) {
	tags, err := postProcessTags(context.Background(), noopTagQueryer{}, []*models.ScrapedTag{
		nil,
		{Name: "valid"},
		nil,
	})
	if err != nil {
		t.Fatalf("postProcessTags() error = %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "valid" {
		t.Fatalf("postProcessTags() = %#v, want single valid tag", tags)
	}
}

func TestFilterTagsSkipsNilEntries(t *testing.T) {
	tags, ignored := FilterTags(nil, []*models.ScrapedTag{
		nil,
		{Name: "kept"},
	})
	if len(ignored) != 0 {
		t.Fatalf("FilterTags() ignored = %v, want none", ignored)
	}
	if len(tags) != 1 || tags[0].Name != "kept" {
		t.Fatalf("FilterTags() = %#v, want single kept tag", tags)
	}
}
