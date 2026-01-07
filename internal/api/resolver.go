package api

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/stashapp/stash/internal/build"
	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/scraper"
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

func (r *Resolver) scraperCache() *scraper.Cache {
	return manager.GetInstance().ScraperCache
}

func (r *Resolver) Gallery() GalleryResolver {
	return &galleryResolver{r}
}
func (r *Resolver) Game() GameResolver {
	return &gameResolver{r}
}
func (r *Resolver) GameFilterType() GameFilterTypeResolver {
	return &gameFilterTypeResolver{r}
}
func (r *Resolver) GalleryChapter() GalleryChapterResolver {
	return &galleryChapterResolver{r}
}
func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Performer() PerformerResolver {
	return &performerResolver{r}
}
func (r *Resolver) PerformerProfileImage() PerformerProfileImageResolver {
	return &performerProfileImageResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}
func (r *Resolver) Scene() SceneResolver {
	return &sceneResolver{r}
}
func (r *Resolver) Image() ImageResolver {
	return &imageResolver{r}
}
func (r *Resolver) SceneMarker() SceneMarkerResolver {
	return &sceneMarkerResolver{r}
}
func (r *Resolver) Studio() StudioResolver {
	return &studioResolver{r}
}

func (r *Resolver) Group() GroupResolver {
	return &groupResolver{r}
}
func (r *Resolver) Movie() MovieResolver {
	return &movieResolver{&groupResolver{r}}
}

func (r *Resolver) Subscription() SubscriptionResolver {
	return &subscriptionResolver{r}
}
func (r *Resolver) Tag() TagResolver {
	return &tagResolver{r}
}
func (r *Resolver) GalleryFile() GalleryFileResolver {
	return &galleryFileResolver{r}
}
func (r *Resolver) VideoFile() VideoFileResolver {
	return &videoFileResolver{r}
}
func (r *Resolver) ImageFile() ImageFileResolver {
	return &imageFileResolver{r}
}
func (r *Resolver) BasicFile() BasicFileResolver {
	return &basicFileResolver{r}
}
func (r *Resolver) Folder() FolderResolver {
	return &folderResolver{r}
}
func (r *Resolver) SavedFilter() SavedFilterResolver {
	return &savedFilterResolver{r}
}
func (r *Resolver) Plugin() PluginResolver {
	return &pluginResolver{r}
}
func (r *Resolver) ConfigResult() ConfigResultResolver {
	return &configResultResolver{r}
}
func (r *Resolver) SceneUpdateInput() SceneUpdateInputResolver {
	return &sceneUpdateInputResolver{r}
}
func (r *Resolver) SceneCreateInput() SceneCreateInputResolver {
	return &sceneCreateInputResolver{r}
}

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }

type galleryResolver struct{ *Resolver }
type gameResolver struct{ *Resolver }
type galleryChapterResolver struct{ *Resolver }
type performerResolver struct{ *Resolver }
type performerProfileImageResolver struct{ *Resolver }
type sceneResolver struct{ *Resolver }
type sceneMarkerResolver struct{ *Resolver }
type imageResolver struct{ *Resolver }
type studioResolver struct{ *Resolver }

// movie is group under the hood
type groupResolver struct{ *Resolver }
type movieResolver struct{ *groupResolver }

type tagResolver struct{ *Resolver }
type galleryFileResolver struct{ *Resolver }
type videoFileResolver struct{ *Resolver }
type imageFileResolver struct{ *Resolver }
type basicFileResolver struct{ *Resolver }
type sceneUpdateInputResolver struct{ *Resolver }
type sceneCreateInputResolver struct{ *Resolver }
type folderResolver struct{ *Resolver }
type savedFilterResolver struct{ *Resolver }
type pluginResolver struct{ *Resolver }
type configResultResolver struct{ *Resolver }

func (r *Resolver) withTxn(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.repository.WithTxn(ctx, fn)
}

func (r *Resolver) withReadTxn(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.repository.WithReadTxn(ctx, fn)
}

func (r *queryResolver) MarkerWall(ctx context.Context, q *string) (ret []*models.SceneMarker, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.SceneMarker.Wall(ctx, q)
		return err
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (r *queryResolver) SceneWall(ctx context.Context, q *string) (ret []*models.Scene, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Scene.Wall(ctx, q)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) MarkerStrings(ctx context.Context, q *string, sort *string) (ret []*models.MarkerStringsResultType, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.SceneMarker.GetMarkerStrings(ctx, q, sort)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
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

		// embrace the error

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

		// Get all o-count dates from the last year
		now := time.Now()
		oneYearAgo := now.AddDate(-1, 0, 0)

		// Get all o-count dates from scenes_o_dates table
		sceneODates, err := sceneQB.GetODatesInRange(ctx, oneYearAgo, now)
		if err != nil {
			return err
		}

		// Get all o-count dates from images_o_dates table
		imageODates, err := imageQB.GetODatesInRange(ctx, oneYearAgo, now)
		if err != nil {
			return err
		}

		// Get all o-count dates from galleries_o_dates table
		galleryODates, err := galleryQB.GetODatesInRange(ctx, oneYearAgo, now)
		if err != nil {
			return err
		}

		// Get all omg-count dates from scenes_omg_dates table
		sceneOMGDates, err := sceneQB.GetOMGDatesInRange(ctx, oneYearAgo, now)
		if err != nil {
			return err
		}

		// Get all omg-count dates from images_omg_dates table
		imageOMGDates, err := imageQB.GetOMGDatesInRange(ctx, oneYearAgo, now)
		if err != nil {
			return err
		}

		// Get all omg-count dates from galleries_omg_dates table
		galleryOMGDates, err := galleryQB.GetOMGDatesInRange(ctx, oneYearAgo, now)
		if err != nil {
			return err
		}

		// Combine all o-dates and omg-dates
		allODates := sceneODates
		allODates = append(allODates, imageODates...)
		allODates = append(allODates, galleryODates...)
		allODates = append(allODates, sceneOMGDates...)
		allODates = append(allODates, imageOMGDates...)
		allODates = append(allODates, galleryOMGDates...)

		// Group by date
		dailyCounts := make(map[string]int)
		for _, oDate := range allODates {
			dateStr := oDate.Format("2006-01-02")
			dailyCounts[dateStr]++
		}

		// Convert to slice and sort by date
		var dailyStats []*OCountDailyStatsType
		for date, count := range dailyCounts {
			// Parse date and format for display
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

		// Sort by date
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

func (r *queryResolver) Version(ctx context.Context) (*Version, error) {
	version, hash, buildtime := build.Version()

	return &Version{
		Version:   &version,
		Hash:      hash,
		BuildTime: buildtime,
	}, nil
}

func (r *queryResolver) Latestversion(ctx context.Context) (*LatestVersion, error) {
	latestRelease, err := GetLatestRelease(ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Errorf("Error while retrieving latest version: %v", err)
		}
		return nil, err
	}
	logger.Infof("Retrieved latest version: %s (%s)", latestRelease.Version, latestRelease.ShortHash)

	return &LatestVersion{
		Version:     latestRelease.Version,
		Shorthash:   latestRelease.ShortHash,
		ReleaseDate: latestRelease.Date,
		URL:         latestRelease.Url,
	}, nil
}

func (r *mutationResolver) ExecSQL(ctx context.Context, sql string, args []interface{}) (*SQLExecResult, error) {
	var rowsAffected *int64
	var lastInsertID *int64

	db := manager.GetInstance().Database
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		var err error
		rowsAffected, lastInsertID, err = db.ExecSQL(ctx, sql, args)
		return err
	}); err != nil {
		return nil, err
	}

	return &SQLExecResult{
		RowsAffected: rowsAffected,
		LastInsertID: lastInsertID,
	}, nil
}

func (r *mutationResolver) QuerySQL(ctx context.Context, sql string, args []interface{}) (*SQLQueryResult, error) {
	var cols []string
	var rows [][]interface{}

	db := manager.GetInstance().Database
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		var err error
		cols, rows, err = db.QuerySQL(ctx, sql, args)
		return err
	}); err != nil {
		return nil, err
	}

	return &SQLQueryResult{
		Columns: cols,
		Rows:    rows,
	}, nil
}

// Get scene marker tags which show up under the video.
func (r *queryResolver) SceneMarkerTags(ctx context.Context, scene_id string) ([]*SceneMarkerTag, error) {
	sceneID, err := strconv.Atoi(scene_id)
	if err != nil {
		return nil, err
	}

	var keys []int
	tags := make(map[int]*SceneMarkerTag)

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		sceneMarkers, err := r.repository.SceneMarker.FindBySceneID(ctx, sceneID)
		if err != nil {
			return err
		}

		tqb := r.repository.Tag
		for _, sceneMarker := range sceneMarkers {
			markerPrimaryTag, err := tqb.Find(ctx, sceneMarker.PrimaryTagID)
			if err != nil {
				return err
			}

			if markerPrimaryTag == nil {
				return fmt.Errorf("tag with id %d not found", sceneMarker.PrimaryTagID)
			}

			_, hasKey := tags[markerPrimaryTag.ID]
			if !hasKey {
				sceneMarkerTag := &SceneMarkerTag{Tag: markerPrimaryTag}
				tags[markerPrimaryTag.ID] = sceneMarkerTag
				keys = append(keys, markerPrimaryTag.ID)
			}
			tags[markerPrimaryTag.ID].SceneMarkers = append(tags[markerPrimaryTag.ID].SceneMarkers, sceneMarker)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// Sort so that primary tags that show up earlier in the video are first.
	sort.Slice(keys, func(i, j int) bool {
		a := tags[keys[i]]
		b := tags[keys[j]]
		return a.SceneMarkers[0].Seconds < b.SceneMarkers[0].Seconds
	})

	var result []*SceneMarkerTag
	for _, key := range keys {
		result = append(result, tags[key])
	}

	return result, nil
}

func firstError(errs []error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}

	return nil
}
