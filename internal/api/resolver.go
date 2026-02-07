package api

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
)

var (
	// ErrNotImplemented is an error which means the given functionality isn't implemented by the API.
	ErrNotImplemented = errors.New("not implemented")

	// ErrNotSupported is returned whenever there's a test, which can be used to guard against the error,
	// but the given parameters aren't supported by the system.
	ErrNotSupported = errors.New("not supported")

	// ErrInput signifies errors where the input isn't valid for some reason. And no more specific error exists.
	ErrInput = errors.New("input error")
)

type hookExecutor interface {
	ExecutePostHooks(ctx context.Context, id int, hookType hook.TriggerEnum, input interface{}, inputFields []string)
}

type Resolver struct {
	repository     models.Repository
	sceneService   manager.SceneService
	imageService   manager.ImageService
	galleryService manager.GalleryService
	groupService   manager.GroupService

	hookExecutor hookExecutor
}

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

func (r *Resolver) withTxn(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.repository.WithTxn(ctx, fn)
}

func (r *Resolver) withReadTxn(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.repository.WithReadTxn(ctx, fn)
}

func (r *queryResolver) Stats(ctx context.Context) (*StatsResultType, error) {
	var ret StatsResultType
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		repo := r.repository
		sceneQB := repo.Scene
		imageQB := repo.Image
		galleryQB := repo.Gallery
		studioQB := repo.Studio
		performerQB := repo.Performer
		movieQB := repo.Group
		tagQB := repo.Tag

		scenesCount, err := sceneQB.Count(ctx)
		if err != nil {
			return err
		}

		scenesSize, err := sceneQB.Size(ctx)
		if err != nil {
			return err
		}

		scenesDuration, err := sceneQB.Duration(ctx)
		if err != nil {
			return err
		}

		imageCount, err := imageQB.Count(ctx)
		if err != nil {
			return err
		}

		imageSize, err := imageQB.Size(ctx)
		if err != nil {
			return err
		}

		galleryCount, err := galleryQB.Count(ctx)
		if err != nil {
			return err
		}

		performersCount, err := performerQB.Count(ctx)
		if err != nil {
			return err
		}

		studiosCount, err := studioQB.Count(ctx)
		if err != nil {
			return err
		}

		groupsCount, err := movieQB.Count(ctx)
		if err != nil {
			return err
		}

		tagsCount, err := tagQB.Count(ctx)
		if err != nil {
			return err
		}

		scenesTotalOCount, err := sceneQB.GetAllOCount(ctx)
		if err != nil {
			return err
		}
		imagesTotalOCount, err := imageQB.OCount(ctx)
		if err != nil {
			return err
		}
		galleriesTotalOCount, err := galleryQB.OCount(ctx)
		if err != nil {
			return err
		}
		totalOCount := scenesTotalOCount + imagesTotalOCount + galleriesTotalOCount

		scenesTotalOMGCount, err := sceneQB.GetAllOMGCount(ctx)
		if err != nil {
			return err
		}
		imagesTotalOMGCount, err := imageQB.GetAllOMGCount(ctx)
		if err != nil {
			return err
		}
		galleriesTotalOMGCount, err := galleryQB.GetAllOMGCount(ctx)
		if err != nil {
			return err
		}
		totalOMGCount := scenesTotalOMGCount + imagesTotalOMGCount + galleriesTotalOMGCount

		totalPlayDuration, err := sceneQB.PlayDuration(ctx)
		if err != nil {
			return err
		}

		totalPlayCount, err := sceneQB.CountAllViews(ctx)
		if err != nil {
			return err
		}

		uniqueScenePlayCount, err := sceneQB.CountUniqueViews(ctx)
		if err != nil {
			return err
		}

		ret = StatsResultType{
			SceneCount:        scenesCount,
			ScenesSize:        scenesSize,
			ScenesDuration:    scenesDuration,
			ImageCount:        imageCount,
			ImagesSize:        imageSize,
			GalleryCount:      galleryCount,
			PerformerCount:    performersCount,
			StudioCount:       studiosCount,
			GroupCount:        groupsCount,
			MovieCount:        groupsCount,
			TagCount:          tagsCount,
			TotalOCount:       totalOCount,
			TotalOmgCount:     totalOMGCount,
			TotalPlayDuration: totalPlayDuration,
			TotalPlayCount:    totalPlayCount,
			ScenesPlayed:      uniqueScenePlayCount,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (r *queryResolver) OCountStats(ctx context.Context) (*OCountStatsResultType, error) {
	var ret OCountStatsResultType
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		repo := r.repository
		sceneQB := repo.Scene
		imageQB := repo.Image
		galleryQB := repo.Gallery

		now := time.Now()
		oneYearAgo := now.AddDate(-1, 0, 0)

		sceneODates, err := sceneQB.GetODatesInRange(ctx, oneYearAgo, now)
		if err != nil {
			return err
		}

		imageODates, err := imageQB.GetODatesInRange(ctx, oneYearAgo, now)
		if err != nil {
			return err
		}

		galleryODates, err := galleryQB.GetODatesInRange(ctx, oneYearAgo, now)
		if err != nil {
			return err
		}

		sceneOMGDates, err := sceneQB.GetOMGDatesInRange(ctx, oneYearAgo, now)
		if err != nil {
			return err
		}

		imageOMGDates, err := imageQB.GetOMGDatesInRange(ctx, oneYearAgo, now)
		if err != nil {
			return err
		}

		galleryOMGDates, err := galleryQB.GetOMGDatesInRange(ctx, oneYearAgo, now)
		if err != nil {
			return err
		}

		allODates := sceneODates
		allODates = append(allODates, imageODates...)
		allODates = append(allODates, galleryODates...)
		allODates = append(allODates, sceneOMGDates...)
		allODates = append(allODates, imageOMGDates...)
		allODates = append(allODates, galleryOMGDates...)

		dailyCounts := make(map[string]int)
		for _, oDate := range allODates {
			dateStr := oDate.Format("2006-01-02")
			dailyCounts[dateStr]++
		}

		var dailyStats []*OCountDailyStatsType
		for date, count := range dailyCounts {
			parsedDate, err := time.Parse("2006-01-02", date)
			if err != nil {
				continue
			}

			dailyStats = append(dailyStats, &OCountDailyStatsType{
				Date:        date,
				DateDisplay: parsedDate.Format("2 Jan"),
				Count:       count,
			})
		}

		sort.Slice(dailyStats, func(i, j int) bool {
			return dailyStats[i].Date < dailyStats[j].Date
		})

		ret = OCountStatsResultType{
			DailyStats: dailyStats,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &ret, nil
}

func firstError(errs []error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}

	return nil
}
