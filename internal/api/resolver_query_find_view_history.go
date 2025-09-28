package api

import (
	"context"

	"github.com/stashapp/stash/pkg/models"
)

func (r *queryResolver) FindViewHistory(ctx context.Context, historyFilter *ViewHistoryFilter, filter *models.FindFilterType) (ret *ViewHistoryResult, err error) {
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

		// Get aggregated view history directly from database with single query
		aggregatedViews, err := r.repository.Scene.GetAggregatedViewHistory(ctx, page, perPage)
		if err != nil {
			return err
		}

		// Convert to ViewHistoryEntry format
		var entries []*ViewHistoryEntry

		for _, av := range aggregatedViews {
			// Get scene data
			scene, err := r.repository.Scene.Find(ctx, av.SceneID)
			if err != nil {
				return err
			}
			if scene == nil {
				continue
			}

			entry := &ViewHistoryEntry{
				Scene:     scene,
				ViewDate:  av.ViewDate,
				ODate:     av.ODate,
				ViewCount: &av.ViewCount,
			}
			entries = append(entries, entry)
		}

		// Get total count for pagination
		totalCount, err := r.repository.Scene.GetAggregatedViewHistoryCount(ctx)
		if err != nil {
			return err
		}

		ret = &ViewHistoryResult{
			Count: totalCount,
			Items: entries,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}
