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

		// Get combined aggregated view history for scenes and galleries
		combinedViews, err := r.repository.Scene.GetCombinedAggregatedViewHistory(ctx, page, perPage)
		if err != nil {
			return err
		}

		// Convert to ViewHistoryEntry format
		var entries []*ViewHistoryEntry

		for _, cv := range combinedViews {
			if cv.ContentType == "scene" {
				// Get scene data
				scene, err := r.repository.Scene.Find(ctx, cv.ContentID)
				if err != nil {
					return err
				}
				if scene == nil {
					continue
				}

				entry := &ViewHistoryEntry{
					Scene:     scene,
					ViewDate:  cv.ViewDate,
					ODate:     cv.ODate,
					ViewCount: &cv.ViewCount,
				}
				entries = append(entries, entry)
			} else if cv.ContentType == "gallery" {
				// Get gallery data
				gallery, err := r.repository.Gallery.Find(ctx, cv.ContentID)
				if err != nil {
					return err
				}
				if gallery == nil {
					continue
				}

				entry := &ViewHistoryEntry{
					Gallery:   gallery,
					ViewDate:  cv.ViewDate,
					ODate:     cv.ODate,
					ViewCount: &cv.ViewCount,
				}
				entries = append(entries, entry)
			}
		}

		// Get total count for pagination (scenes view history count)
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
