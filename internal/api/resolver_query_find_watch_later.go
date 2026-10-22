package api

import (
	"context"

	"github.com/stashapp/stash/pkg/models"
)

func (r *queryResolver) FindWatchLater(ctx context.Context, filter *models.FindFilterType) (ret *WatchLaterResult, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		page := 1
		perPage := 25
		if filter != nil {
			if filter.Page != nil {
				page = *filter.Page
			}
			if filter.PerPage != nil {
				perPage = *filter.PerPage
			}
		}

		entries, err := r.repository.Scene.FindWatchLater(ctx, page, perPage)
		if err != nil {
			return err
		}

		items := make([]*models.WatchLaterEntry, len(entries))
		for i, entry := range entries {
			items[i] = &models.WatchLaterEntry{
				SceneID: entry.SceneID,
				AddedAt: entry.AddedAt,
			}
		}

		totalCount, err := r.repository.Scene.CountWatchLater(ctx)
		if err != nil {
			return err
		}

		ret = &WatchLaterResult{
			Count: totalCount,
			Items: items,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

type watchLaterEntryResolver struct{ *Resolver }

func (r *Resolver) WatchLaterEntry() WatchLaterEntryResolver {
	return &watchLaterEntryResolver{r}
}

func (r *watchLaterEntryResolver) Scene(ctx context.Context, obj *models.WatchLaterEntry) (*models.Scene, error) {
	if obj == nil {
		return nil, nil
	}

	var scene *models.Scene
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		scene, err = r.repository.Scene.Find(ctx, obj.SceneID)
		return err
	}); err != nil {
		return nil, err
	}

	return scene, nil
}
