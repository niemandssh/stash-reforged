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
					OmgDate:   cv.OmgDate,
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
					OmgDate:   cv.OmgDate,
					ViewCount: &cv.ViewCount,
				}
				entries = append(entries, entry)
			}
		}

		// Get total count for pagination (combined scenes and galleries view history count)
		totalCount, err := r.repository.Scene.GetCombinedAggregatedViewHistoryCount(ctx)
		if err != nil {
			return err
		}

		// Get total O-Count for all scenes and galleries
		scenesTotalOCount, err := r.repository.Scene.GetAllOCount(ctx)
		if err != nil {
			return err
		}
		galleriesTotalOCount, err := r.repository.Gallery.GetAllOCount(ctx)
		if err != nil {
			return err
		}
		totalOCount := scenesTotalOCount + galleriesTotalOCount

		// Get total OMG-Count for all scenes and galleries
		scenesTotalOMGCount, err := r.repository.Scene.GetAllOMGCount(ctx)
		if err != nil {
			return err
		}
		galleriesTotalOMGCount, err := r.repository.Gallery.GetAllOMGCount(ctx)
		if err != nil {
			return err
		}
		totalOMGCount := scenesTotalOMGCount + galleriesTotalOMGCount

		ret = &ViewHistoryResult{
			Count:        totalCount,
			Items:        entries,
			TotalOCount:  totalOCount,
			TotalOMGCount: totalOMGCount,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}
