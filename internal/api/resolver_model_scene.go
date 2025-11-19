package api

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/stashapp/stash/internal/api/loaders"
	"github.com/stashapp/stash/internal/api/urlbuilders"
	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

func convertVideoFile(f models.File) (*models.VideoFile, error) {
	vf, ok := f.(*models.VideoFile)
	if !ok {
		return nil, fmt.Errorf("file %T is not a video file", f)
	}
	return vf, nil
}

func (r *sceneResolver) getPrimaryFile(ctx context.Context, obj *models.Scene) (*models.VideoFile, error) {
	if obj.PrimaryFileID != nil {
		f, err := loaders.From(ctx).FileByID.Load(*obj.PrimaryFileID)
		if err != nil {
			return nil, err
		}

		ret, err := convertVideoFile(f)
		if err != nil {
			return nil, err
		}

		obj.Files.SetPrimary(ret)

		return ret, nil
	} else {
		_ = obj.LoadPrimaryFile(ctx, r.repository.File)
	}

	return nil, nil
}

func (r *sceneResolver) getFiles(ctx context.Context, obj *models.Scene) ([]*models.VideoFile, error) {
	fileIDs, err := loaders.From(ctx).SceneFiles.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	files, errs := loaders.From(ctx).FileByID.LoadAll(fileIDs)
	err = firstError(errs)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.VideoFile, len(files))
	for i, f := range files {
		ret[i], err = convertVideoFile(f)
		if err != nil {
			return nil, err
		}
	}

	obj.Files.Set(ret)

	return ret, nil
}

func (r *sceneResolver) Date(ctx context.Context, obj *models.Scene) (*string, error) {
	if obj.Date != nil {
		result := obj.Date.String()
		return &result, nil
	}
	return nil, nil
}

func (r *sceneResolver) ShootDate(ctx context.Context, obj *models.Scene) (*string, error) {
	if obj.ShootDate != nil {
		result := obj.ShootDate.String()
		return &result, nil
	}
	return nil, nil
}

func (r *sceneResolver) Files(ctx context.Context, obj *models.Scene) ([]*VideoFile, error) {
	files, err := r.getFiles(ctx, obj)
	if err != nil {
		return nil, err
	}

	ret := make([]*VideoFile, len(files))

	for i, f := range files {
		ret[i] = &VideoFile{
			VideoFile: f,
		}
	}

	return ret, nil
}

func (r *sceneResolver) Rating(ctx context.Context, obj *models.Scene) (*int, error) {
	if obj.Rating != nil {
		rating := models.Rating100To5(*obj.Rating)
		return &rating, nil
	}
	return nil, nil
}

func (r *sceneResolver) Rating100(ctx context.Context, obj *models.Scene) (*int, error) {
	return obj.Rating, nil
}

func (r *sceneResolver) IsBroken(ctx context.Context, obj *models.Scene) (bool, error) {
	return obj.IsBroken, nil
}

func (r *sceneResolver) IsNotBroken(ctx context.Context, obj *models.Scene) (bool, error) {
	return obj.IsNotBroken, nil
}

func (r *sceneResolver) IsProbablyBroken(ctx context.Context, obj *models.Scene) (bool, error) {
	// Load primary file if not already loaded
	_, err := r.getPrimaryFile(ctx, obj)
	if err != nil {
		logger.Infof("[DEBUG] IsProbablyBroken: failed to load primary file for scene %d: %v", obj.ID, err)
		return false, err
	}

	// Use the manager function to determine if the scene is probably broken
	result := manager.IsProbablyBroken(obj)
	return result, nil
}

func (r *sceneResolver) Paths(ctx context.Context, obj *models.Scene) (*ScenePathsType, error) {
	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	config := manager.GetInstance().Config
	builder := urlbuilders.NewSceneURLBuilder(baseURL, obj)
	screenshotPath := builder.GetScreenshotURL()
	previewPath := builder.GetStreamPreviewURL()
	streamPath := builder.GetStreamURL(config.GetAPIKey()).String()
	webpPath := builder.GetStreamPreviewImageURL()
	objHash := obj.GetHash(config.GetVideoFileNamingAlgorithm())
	vttPath := builder.GetSpriteVTTURL(objHash)
	spritePath := builder.GetSpriteURL(objHash)
	funscriptPath := builder.GetFunscriptURL()
	captionBasePath := builder.GetCaptionURL()
	interactiveHeatmap := builder.GetInteractiveHeatmapURL()

	return &ScenePathsType{
		Screenshot:         &screenshotPath,
		Preview:            &previewPath,
		Stream:             &streamPath,
		Webp:               &webpPath,
		Vtt:                &vttPath,
		Sprite:             &spritePath,
		Funscript:          &funscriptPath,
		InteractiveHeatmap: &interactiveHeatmap,
		Caption:            &captionBasePath,
	}, nil
}

func (r *sceneResolver) SceneMarkers(ctx context.Context, obj *models.Scene) (ret []*models.SceneMarker, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.SceneMarker.FindBySceneID(ctx, obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *sceneResolver) Captions(ctx context.Context, obj *models.Scene) (ret []*models.VideoCaption, err error) {
	primaryFile, err := r.getPrimaryFile(ctx, obj)
	if err != nil {
		return nil, err
	}
	if primaryFile == nil {
		return nil, nil
	}

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.File.GetCaptions(ctx, primaryFile.Base().ID)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, err
}

func (r *sceneResolver) Galleries(ctx context.Context, obj *models.Scene) (ret []*models.Gallery, err error) {
	if !obj.GalleryIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadGalleryIDs(ctx, r.repository.Scene)
		}); err != nil {
			return nil, err
		}
	}

	var errs []error
	ret, errs = loaders.From(ctx).GalleryByID.LoadAll(obj.GalleryIDs.List())
	return ret, firstError(errs)
}

func (r *sceneResolver) Studio(ctx context.Context, obj *models.Scene) (ret *models.Studio, err error) {
	if obj.StudioID == nil {
		return nil, nil
	}

	return loaders.From(ctx).StudioByID.Load(*obj.StudioID)
}

func (r *sceneResolver) Movies(ctx context.Context, obj *models.Scene) (ret []*SceneMovie, err error) {
	if !obj.Groups.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			qb := r.repository.Scene

			return obj.LoadGroups(ctx, qb)
		}); err != nil {
			return nil, err
		}
	}

	loader := loaders.From(ctx).GroupByID

	for _, sm := range obj.Groups.List() {
		movie, err := loader.Load(sm.GroupID)
		if err != nil {
			return nil, err
		}

		sceneIdx := sm.SceneIndex
		sceneMovie := &SceneMovie{
			Movie:      movie,
			SceneIndex: sceneIdx,
		}

		ret = append(ret, sceneMovie)
	}

	return ret, nil
}

func (r *sceneResolver) Groups(ctx context.Context, obj *models.Scene) (ret []*SceneGroup, err error) {
	if !obj.Groups.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			qb := r.repository.Scene

			return obj.LoadGroups(ctx, qb)
		}); err != nil {
			return nil, err
		}
	}

	loader := loaders.From(ctx).GroupByID

	for _, sm := range obj.Groups.List() {
		group, err := loader.Load(sm.GroupID)
		if err != nil {
			return nil, err
		}

		sceneIdx := sm.SceneIndex
		sceneGroup := &SceneGroup{
			Group:      group,
			SceneIndex: sceneIdx,
		}

		ret = append(ret, sceneGroup)
	}

	return ret, nil
}

func (r *sceneResolver) ScenePerformers(ctx context.Context, obj *models.Scene) (ret []*ScenePerformer, err error) {
	if !obj.ScenePerformers.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			qb := r.repository.Scene
			// Call GetScenePerformers directly instead of using interface
			scenePerformers, err := qb.GetScenePerformers(ctx, obj.ID)
			if err != nil {
				return err
			}
			obj.ScenePerformers = models.NewRelatedScenePerformers(scenePerformers)
			return nil
		}); err != nil {
			return nil, err
		}
	}

	// If ScenePerformers is not loaded or empty, return empty array
	if !obj.ScenePerformers.Loaded() {
		return []*ScenePerformer{}, nil
	}

	loader := loaders.From(ctx).PerformerByID

	for _, sp := range obj.ScenePerformers.List() {
		performer, err := loader.Load(sp.PerformerID)
		if err != nil {
			return nil, err
		}

		scenePerformer := &ScenePerformer{
			Performer:       performer,
			SmallRole:       sp.SmallRole,
			RoleDescription: sp.RoleDescription,
		}

		ret = append(ret, scenePerformer)
	}

	return ret, nil
}

func (r *sceneResolver) Tags(ctx context.Context, obj *models.Scene) (ret []*models.Tag, err error) {
	if !obj.TagIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadTagIDs(ctx, r.repository.Scene)
		}); err != nil {
			return nil, err
		}
	}

	// Get regular scene tags
	tagIDs := obj.TagIDs.List()

	// Also include performer tags in the scene tags for GraphQL
	if !obj.PerformerTagIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadPerformerTagIDs(ctx, r.repository.Scene)
		}); err != nil {
			return nil, err
		}
	}

	// Add performer tag IDs to the list
	for _, pt := range obj.PerformerTagIDs.List() {
		tagIDs = append(tagIDs, pt.TagID)
	}

	// Remove duplicates
	seen := make(map[int]bool)
	var uniqueTagIDs []int
	for _, id := range tagIDs {
		if !seen[id] {
			seen[id] = true
			uniqueTagIDs = append(uniqueTagIDs, id)
		}
	}

	var errs []error
	ret, errs = loaders.From(ctx).TagByID.LoadAll(uniqueTagIDs)
	return ret, firstError(errs)
}

func (r *sceneResolver) PerformerTagIds(ctx context.Context, obj *models.Scene) (ret []*models.PerformerTag, err error) {
	if !obj.PerformerTagIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadPerformerTagIDs(ctx, r.repository.Scene)
		}); err != nil {
			return nil, err
		}
	}

	// Group tags by performer
	performerTagsMap := make(map[string][]string)
	for _, pt := range obj.PerformerTagIDs.List() {
		var performerKey string
		if pt.PerformerID != nil {
			performerKey = strconv.Itoa(*pt.PerformerID)
		} else {
			performerKey = "null" // For general scene tags
		}

		tagIDStr := strconv.Itoa(pt.TagID)
		performerTagsMap[performerKey] = append(performerTagsMap[performerKey], tagIDStr)
	}

	// Convert map to PerformerTag slice
	ret = make([]*models.PerformerTag, 0, len(performerTagsMap))
	for performerKey, tagIDs := range performerTagsMap {
		var performerID *string
		if performerKey != "null" {
			performerID = &performerKey
		}

		ret = append(ret, &models.PerformerTag{
			PerformerID: performerID,
			TagIds:      tagIDs,
		})
	}

	return ret, nil
}

func (r *sceneResolver) Performers(ctx context.Context, obj *models.Scene) (ret []*models.Performer, err error) {
	if !obj.PerformerIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadPerformerIDs(ctx, r.repository.Scene)
		}); err != nil {
			return nil, err
		}
	}

	var errs []error
	ret, errs = loaders.From(ctx).PerformerByID.LoadAll(obj.PerformerIDs.List())
	return ret, firstError(errs)
}

func (r *sceneResolver) StashIds(ctx context.Context, obj *models.Scene) (ret []*models.StashID, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		return obj.LoadStashIDs(ctx, r.repository.Scene)
	}); err != nil {
		return nil, err
	}

	return stashIDsSliceToPtrSlice(obj.StashIDs.List()), nil
}

func (r *sceneResolver) SceneStreams(ctx context.Context, obj *models.Scene) ([]*manager.SceneStreamEndpoint, error) {
	// load the primary file into the scene
	_, err := r.getPrimaryFile(ctx, obj)
	if err != nil {
		return nil, err
	}

	config := manager.GetInstance().Config

	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	builder := urlbuilders.NewSceneURLBuilder(baseURL, obj)
	apiKey := config.GetAPIKey()

	return manager.GetSceneStreamPaths(obj, builder.GetStreamURL(apiKey), config.GetMaxStreamingTranscodeSize())
}

func (r *sceneResolver) Interactive(ctx context.Context, obj *models.Scene) (bool, error) {
	primaryFile, err := r.getPrimaryFile(ctx, obj)
	if err != nil {
		return false, err
	}
	if primaryFile == nil {
		return false, nil
	}

	return primaryFile.Interactive, nil
}

func (r *sceneResolver) InteractiveSpeed(ctx context.Context, obj *models.Scene) (*int, error) {
	primaryFile, err := r.getPrimaryFile(ctx, obj)
	if err != nil {
		return nil, err
	}
	if primaryFile == nil {
		return nil, nil
	}

	return primaryFile.InteractiveSpeed, nil
}

func (r *sceneResolver) URL(ctx context.Context, obj *models.Scene) (*string, error) {
	if !obj.URLs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadURLs(ctx, r.repository.Scene)
		}); err != nil {
			return nil, err
		}
	}

	urls := obj.URLs.List()
	if len(urls) == 0 {
		return nil, nil
	}

	return &urls[0], nil
}

func (r *sceneResolver) Urls(ctx context.Context, obj *models.Scene) ([]string, error) {
	if !obj.URLs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadURLs(ctx, r.repository.Scene)
		}); err != nil {
			return nil, err
		}
	}

	return obj.URLs.List(), nil
}

func (r *sceneResolver) OCounter(ctx context.Context, obj *models.Scene) (*int, error) {
	ret, err := loaders.From(ctx).SceneOCount.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (r *sceneResolver) OmgCounter(ctx context.Context, obj *models.Scene) (*int, error) {
	var ret int
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		ret, err = r.repository.Scene.GetOMGCounter(ctx, obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (r *sceneResolver) LastPlayedAt(ctx context.Context, obj *models.Scene) (*time.Time, error) {
	ret, err := loaders.From(ctx).SceneLastPlayed.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *sceneResolver) PlayCount(ctx context.Context, obj *models.Scene) (*int, error) {
	ret, err := loaders.From(ctx).ScenePlayCount.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (r *sceneResolver) PlayHistory(ctx context.Context, obj *models.Scene) ([]*time.Time, error) {
	ret, err := loaders.From(ctx).ScenePlayHistory.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	// convert to pointer slice
	ptrRet := make([]*time.Time, len(ret))
	for i, t := range ret {
		tt := t
		ptrRet[i] = &tt
	}

	return ptrRet, nil
}

func (r *sceneResolver) OHistory(ctx context.Context, obj *models.Scene) ([]*time.Time, error) {
	ret, err := loaders.From(ctx).SceneOHistory.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	// convert to pointer slice
	ptrRet := make([]*time.Time, len(ret))
	for i, t := range ret {
		tt := t
		ptrRet[i] = &tt
	}

	return ptrRet, nil
}

func (r *sceneResolver) OmgHistory(ctx context.Context, obj *models.Scene) ([]*time.Time, error) {
	var ret []time.Time
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		ret, err = r.repository.Scene.GetOMGDates(ctx, obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	// convert to pointer slice
	ptrRet := make([]*time.Time, len(ret))
	for i, t := range ret {
		tt := t
		ptrRet[i] = &tt
	}

	return ptrRet, nil
}

func (r *sceneResolver) SimilarScenes(ctx context.Context, obj *models.Scene, limit *int) ([]*models.SimilarScene, error) {
	// Default limit to 10 if not specified
	defaultLimit := 10
	if limit != nil {
		defaultLimit = *limit
	}

	// Get similar scenes from the database within a transaction
	var similarities []*models.SceneSimilarity
	if err := r.repository.WithTxn(ctx, func(ctx context.Context) error {
		var err error
		similarities, err = r.repository.SceneSimilarity.FindSimilarScenes(ctx, obj.ID, defaultLimit)
		return err
	}); err != nil {
		return nil, fmt.Errorf("finding similar scenes: %w", err)
	}

	if len(similarities) == 0 {
		return []*models.SimilarScene{}, nil
	}

	// Extract scene IDs
	sceneIDs := make([]int, len(similarities))
	for i, sim := range similarities {
		sceneIDs[i] = sim.SimilarSceneID
	}

	// Load the actual scenes within a transaction
	// Use FindByIDs instead of FindMany to handle missing scenes gracefully
	var scenes []*models.Scene
	if err := r.repository.WithTxn(ctx, func(ctx context.Context) error {
		var err error
		scenes, err = r.repository.Scene.FindByIDs(ctx, sceneIDs)
		return err
	}); err != nil {
		return nil, fmt.Errorf("loading similar scenes: %w", err)
	}

	// Create SimilarScene objects with scores, filtering out missing scenes and duplicates
	similarScenes := make([]*models.SimilarScene, 0, len(similarities))
	seenSceneIDs := make(map[int]bool)

	for _, sim := range similarities {
		// Skip if we've already processed this scene (duplicate)
		if seenSceneIDs[sim.SimilarSceneID] {
			continue
		}

		// Find the corresponding scene
		var scene *models.Scene
		for _, s := range scenes {
			if s.ID == sim.SimilarSceneID {
				scene = s
				break
			}
		}

		if scene != nil {
			seenSceneIDs[sim.SimilarSceneID] = true
			similarScenes = append(similarScenes, &models.SimilarScene{
				Scene:               scene,
				SimilarityScore:     sim.SimilarityScore,
				SimilarityScoreData: sim.SimilarityScoreData,
			})
		}
		// Note: If scene is nil, it means the scene was deleted but similarity record still exists
		// This is handled gracefully by skipping the missing scene
	}

	return similarScenes, nil
}

func (r *sceneResolver) StartTime(ctx context.Context, obj *models.Scene) (*float64, error) {
	return obj.StartTime, nil
}

func (r *sceneResolver) EndTime(ctx context.Context, obj *models.Scene) (*float64, error) {
	return obj.EndTime, nil
}

func (r *sceneResolver) VideoFilters(ctx context.Context, obj *models.Scene) (*models.VideoFilters, error) {
	return obj.VideoFilters, nil
}

func (r *sceneResolver) VideoTransforms(ctx context.Context, obj *models.Scene) (*models.VideoTransforms, error) {
	return obj.VideoTransforms, nil
}
