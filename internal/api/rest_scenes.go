package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/scene"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/utils"
)

// GET /api/v1/scenes/{id}
func (h *RESTHandler) findScene(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *models.Scene
	var enriched []EnrichedScene
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		ret, err = h.repository.Scene.Find(ctx, id)
		if err != nil {
			return err
		}
		if ret != nil {
			if err := ret.LoadRelationships(ctx, h.repository.Scene); err != nil {
				return err
			}
			enriched, err = h.enrichScenes(ctx, []*models.Scene{ret})
			if err != nil {
				return err
			}
		}
		return nil
	}) {
		return
	}

	if ret == nil {
		respondNotFound(w)
		return
	}

	respondOK(w, enriched[0])
}

// POST /api/v1/scenes/query
func (h *RESTHandler) findScenes(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Filter      *models.FindFilterType  `json:"filter,omitempty"`
		SceneFilter *models.SceneFilterType `json:"scene_filter,omitempty"`
		IDs         []string                `json:"ids,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var scenes []*models.Scene
	var enriched []EnrichedScene
	var count int

	var sceneIDs []int
	if len(input.IDs) > 0 {
		var err error
		sceneIDs, err = stringslice.StringSliceToIntSlice(input.IDs)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("converting ids: %w", err))
			return
		}
	}

	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		if len(sceneIDs) > 0 {
			var err error
			scenes, err = h.repository.Scene.FindMany(ctx, sceneIDs)
			if err != nil {
				return err
			}
			count = len(scenes)
		} else {
			qr, err := h.repository.Scene.Query(ctx, models.SceneQueryOptions{
				QueryOptions: models.QueryOptions{
					FindFilter: input.Filter,
					Count:      true,
				},
				SceneFilter: input.SceneFilter,
			})
			if err != nil {
				return err
			}
			scenes, err = qr.Resolve(ctx)
			if err != nil {
				return err
			}
			count = qr.Count
		}

		// Load relationships for all scenes
		for _, s := range scenes {
			if err := s.LoadRelationships(ctx, h.repository.Scene); err != nil {
				return fmt.Errorf("loading scene relationships: %w", err)
			}
		}

		// Enrich scenes with resolved related objects
		var err error
		enriched, err = h.enrichScenes(ctx, scenes)
		if err != nil {
			return fmt.Errorf("enriching scenes: %w", err)
		}

		return nil
	}) {
		return
	}

	respondList(w, http.StatusOK, enriched, count)
}

// POST /api/v1/scenes
func (h *RESTHandler) createScene(w http.ResponseWriter, r *http.Request) {
	var input models.SceneCreateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	translator := newRESTChangesetTranslator(inputMap)

	fileIDs, err := translator.fileIDSliceFromStringSlice(input.FileIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting file ids: %w", err))
		return
	}

	newScene := models.NewScene()

	newScene.Title = translator.string(input.Title)
	newScene.Code = translator.string(input.Code)
	newScene.Details = translator.string(input.Details)
	newScene.Director = translator.string(input.Director)
	newScene.Rating = input.Rating100
	newScene.Organized = translator.bool(input.Organized)
	newScene.IsBroken = translator.bool(input.IsBroken)
	newScene.StashIDs = models.NewRelatedStashIDs(models.StashIDInputs(input.StashIds).ToStashIDs())

	newScene.Date, err = translator.datePtr(input.Date)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting date: %w", err))
		return
	}
	newScene.ShootDate, err = translator.datePtr(input.ShootDate)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting shoot_date: %w", err))
		return
	}
	newScene.StudioID, err = translator.intPtrFromString(input.StudioID)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting studio id: %w", err))
		return
	}

	if input.Urls != nil {
		newScene.URLs = models.NewRelatedStrings(input.Urls)
	} else if input.URL != nil {
		newScene.URLs = models.NewRelatedStrings([]string{*input.URL})
	}

	newScene.PerformerIDs, err = translator.relatedIds(input.PerformerIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting performer ids: %w", err))
		return
	}
	newScene.TagIDs, err = translator.relatedIds(input.TagIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}
	newScene.GalleryIDs, err = translator.relatedIds(input.GalleryIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting gallery ids: %w", err))
		return
	}

	if len(input.Groups) > 0 {
		newScene.Groups, err = translator.relatedGroups(input.Groups)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("converting groups: %w", err))
			return
		}
	} else if len(input.Movies) > 0 {
		newScene.Groups, err = translator.relatedGroupsFromMovies(input.Movies)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("converting movies: %w", err))
			return
		}
	}

	if len(input.ScenePerformers) > 0 {
		newScene.ScenePerformers = models.NewRelatedScenePerformers(input.ScenePerformers)
	}

	var coverImageData []byte
	if input.CoverImage != nil {
		coverImageData, err = utils.ProcessImageInput(r.Context(), *input.CoverImage)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("processing cover image: %w", err))
			return
		}
	}

	var ret *models.Scene
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		ret, err = h.sceneService.Create(ctx, &newScene, fileIDs, coverImageData)
		return err
	}) {
		return
	}

	respondCreated(w, ret)
}

// PUT /api/v1/scenes/{id}
func (h *RESTHandler) updateScene(w http.ResponseWriter, r *http.Request) {
	sceneID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var input models.SceneUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	input.ID = models.FlexibleID(strconv.Itoa(sceneID))
	translator := newRESTChangesetTranslator(inputMap)

	updatedScene, err := scenePartialFromInput(input, translator)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *models.Scene
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Scene

		originalScene, err := qb.Find(ctx, sceneID)
		if err != nil {
			return err
		}
		if originalScene == nil {
			return fmt.Errorf("scene with id %d not found", sceneID)
		}

		ret, err = qb.UpdatePartial(ctx, sceneID, *updatedScene)
		if err != nil {
			return err
		}

		// Update cover image if provided
		if input.CoverImage != nil {
			imageData, err := utils.ProcessImageInput(ctx, *input.CoverImage)
			if err != nil {
				return fmt.Errorf("processing cover image: %w", err)
			}
			if err := qb.UpdateCover(ctx, sceneID, imageData); err != nil {
				return err
			}
		}

		return ret.LoadRelationships(ctx, qb)
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), ret.ID, hook.SceneUpdatePost, input, translator.getFields())

	respondOK(w, ret)
}

// PUT /api/v1/scenes/bulk
func (h *RESTHandler) bulkUpdateScenes(w http.ResponseWriter, r *http.Request) {
	var input BulkSceneUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	sceneIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting ids: %w", err))
		return
	}

	translator := newRESTChangesetTranslator(inputMap)

	// Build a partial to apply to all scenes
	updatedScene := models.NewScenePartial()
	updatedScene.Title = translator.optionalString(input.Title, "title")
	updatedScene.Code = translator.optionalString(input.Code, "code")
	updatedScene.Details = translator.optionalString(input.Details, "details")
	updatedScene.Director = translator.optionalString(input.Director, "director")
	updatedScene.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedScene.Organized = translator.optionalBool(input.Organized, "organized")

	updatedScene.Date, err = translator.optionalDate(input.Date, "date")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting date: %w", err))
		return
	}
	updatedScene.StudioID, err = translator.optionalIntFromString(input.StudioID, "studio_id")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting studio id: %w", err))
		return
	}

	updatedScene.URLs = translator.optionalURLsBulk(input.Urls, input.URL)

	updatedScene.PerformerIDs, err = translator.updateIdsBulk(input.PerformerIds, "performer_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting performer ids: %w", err))
		return
	}
	updatedScene.TagIDs, err = translator.updateIdsBulk(input.TagIds, "tag_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}
	updatedScene.GalleryIDs, err = translator.updateIdsBulk(input.GalleryIds, "gallery_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting gallery ids: %w", err))
		return
	}

	if translator.hasField("group_ids") {
		updatedScene.GroupIDs, err = translator.updateGroupIDsBulk(input.GroupIds, "group_ids")
		if err != nil {
			respondBadRequest(w, fmt.Errorf("converting group ids: %w", err))
			return
		}
	} else if translator.hasField("movie_ids") {
		updatedScene.GroupIDs, err = translator.updateGroupIDsBulk(input.MovieIds, "movie_ids")
		if err != nil {
			respondBadRequest(w, fmt.Errorf("converting movie ids: %w", err))
			return
		}
	}

	var ret []*models.Scene
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Scene
		for _, id := range sceneIDs {
			s, err := qb.UpdatePartial(ctx, id, updatedScene)
			if err != nil {
				return err
			}
			ret = append(ret, s)
		}
		return nil
	}) {
		return
	}

	for _, s := range ret {
		h.hookExecutor.ExecutePostHooks(r.Context(), s.ID, hook.SceneUpdatePost, input, translator.getFields())
	}

	respondOK(w, ret)
}

// DELETE /api/v1/scenes/{id}
func (h *RESTHandler) destroyScene(w http.ResponseWriter, r *http.Request) {
	sceneID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var body struct {
		DeleteGenerated *bool `json:"delete_generated,omitempty"`
		DeleteFile      *bool `json:"delete_file,omitempty"`
	}
	// Body is optional for DELETE
	_ = decodeBody(r, &body)

	fileNamingAlgo := manager.GetInstance().Config.GetVideoFileNamingAlgorithm()

	fileDeleter := &scene.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: fileNamingAlgo,
		Paths:          manager.GetInstance().Paths,
	}

	deleteGenerated := utils.IsTrue(body.DeleteGenerated)
	deleteFile := utils.IsTrue(body.DeleteFile)

	var s *models.Scene
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Scene
		s, err = qb.Find(ctx, sceneID)
		if err != nil {
			return err
		}
		if s == nil {
			return fmt.Errorf("scene with id %d not found", sceneID)
		}

		manager.KillRunningStreams(s, fileNamingAlgo)

		return h.sceneService.Destroy(ctx, s, fileDeleter, deleteGenerated, deleteFile)
	}) {
		fileDeleter.Rollback()
		return
	}

	fileDeleter.Commit()

	input := models.SceneDestroyInput{
		ID:              strconv.Itoa(sceneID),
		DeleteFile:      body.DeleteFile,
		DeleteGenerated: body.DeleteGenerated,
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), s.ID, hook.SceneDestroyPost, plugin.SceneDestroyInput{
		SceneDestroyInput: input,
		Checksum:          s.Checksum,
		OSHash:            s.OSHash,
		Path:              s.Path,
	}, nil)

	respondNoContent(w)
}

// DELETE /api/v1/scenes  (body: {"ids": [...], "delete_generated": bool, "delete_file": bool})
func (h *RESTHandler) destroyScenes(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs             []string `json:"ids"`
		DeleteGenerated *bool    `json:"delete_generated,omitempty"`
		DeleteFile      *bool    `json:"delete_file,omitempty"`
	}
	if err := decodeBody(r, &body); err != nil {
		respondBadRequest(w, err)
		return
	}

	sceneIDs, err := stringslice.StringSliceToIntSlice(body.IDs)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting ids: %w", err))
		return
	}

	fileNamingAlgo := manager.GetInstance().Config.GetVideoFileNamingAlgorithm()

	fileDeleter := &scene.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: fileNamingAlgo,
		Paths:          manager.GetInstance().Paths,
	}

	deleteGenerated := utils.IsTrue(body.DeleteGenerated)
	deleteFile := utils.IsTrue(body.DeleteFile)

	var scenes []*models.Scene
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Scene
		for _, id := range sceneIDs {
			s, err := qb.Find(ctx, id)
			if err != nil {
				return err
			}
			if s == nil {
				return fmt.Errorf("scene with id %d not found", id)
			}

			scenes = append(scenes, s)
			manager.KillRunningStreams(s, fileNamingAlgo)

			if err := h.sceneService.Destroy(ctx, s, fileDeleter, deleteGenerated, deleteFile); err != nil {
				return err
			}
		}
		return nil
	}) {
		fileDeleter.Rollback()
		return
	}

	fileDeleter.Commit()

	destroyInput := models.ScenesDestroyInput{
		Ids:             body.IDs,
		DeleteFile:      body.DeleteFile,
		DeleteGenerated: body.DeleteGenerated,
	}

	for _, s := range scenes {
		h.hookExecutor.ExecutePostHooks(r.Context(), s.ID, hook.SceneDestroyPost, plugin.ScenesDestroyInput{
			ScenesDestroyInput: destroyInput,
			Checksum:           s.Checksum,
			OSHash:             s.OSHash,
			Path:               s.Path,
		}, nil)
	}

	respondNoContent(w)
}

// GET /api/v1/scenes/{id}/streams
func (h *RESTHandler) getSceneStreams(w http.ResponseWriter, r *http.Request) {
	sceneID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var s *models.Scene
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		s, err = h.repository.Scene.Find(ctx, sceneID)
		if err != nil {
			return err
		}
		if s != nil {
			return s.LoadFiles(ctx, h.repository.Scene)
		}
		return nil
	}) {
		return
	}

	if s == nil {
		respondNotFound(w)
		return
	}

	// Build the base stream URL: /scene/{id}/stream
	baseURL := &url.URL{
		Scheme: "http",
		Host:   r.Host,
		Path:   fmt.Sprintf("/scene/%d/stream", sceneID),
	}

	maxTranscodeSize := config.GetInstance().GetMaxStreamingTranscodeSize()
	streams, err := manager.GetSceneStreamPaths(s, baseURL, maxTranscodeSize)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_ERROR")
		return
	}

	if streams == nil {
		streams = []*manager.SceneStreamEndpoint{}
	}

	respondOK(w, streams)
}

// GET /api/v1/scenes/{id}/similar
func (h *RESTHandler) findSimilarScenes(w http.ResponseWriter, r *http.Request) {
	sceneID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var similarities []*models.SceneSimilarity
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		similarities, err = h.repository.SceneSimilarity.FindSimilarScenes(ctx, sceneID, 25)
		return err
	}) {
		return
	}

	if similarities == nil || len(similarities) == 0 {
		respondOK(w, []interface{}{})
		return
	}

	// Load the actual scene objects for each similar scene
	type similarSceneResult struct {
		Scene               interface{} `json:"scene"`
		SimilarityScore     float64     `json:"similarity_score"`
		SimilarityScoreData interface{} `json:"similarity_score_data,omitempty"`
	}

	var results []similarSceneResult
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		for _, sim := range similarities {
			similarID := sim.SimilarSceneID
			if similarID == sceneID {
				similarID = sim.SceneID
			}
			s, err := h.repository.Scene.Find(ctx, similarID)
			if err != nil {
				return err
			}
			if s == nil {
				continue
			}
			if err := s.LoadRelationships(ctx, h.repository.Scene); err != nil {
				return err
			}
			enriched, err := h.enrichScenes(ctx, []*models.Scene{s})
			if err != nil {
				return err
			}
			if len(enriched) > 0 {
				results = append(results, similarSceneResult{
					Scene:               enriched[0],
					SimilarityScore:     sim.SimilarityScore,
					SimilarityScoreData: sim.SimilarityScoreData,
				})
			}
		}
		return nil
	}) {
		return
	}

	if results == nil {
		results = []similarSceneResult{}
	}

	respondOK(w, results)
}

// GET /api/v1/scene-markers/tags/{sceneId}
func (h *RESTHandler) sceneMarkerTags(w http.ResponseWriter, r *http.Request) {
	sceneID, err := urlParamInt(r, "sceneId")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	type markerTagResult struct {
		Tag          MinimalTag           `json:"tag"`
		SceneMarkers []MinimalSceneMarker `json:"scene_markers"`
	}

	var results []markerTagResult
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		markers, err := h.repository.SceneMarker.FindBySceneID(ctx, sceneID)
		if err != nil {
			return err
		}

		// Group markers by their primary tag
		tagMarkersMap := make(map[int][]MinimalSceneMarker)
		tagOrder := make([]int, 0)
		tagMap := make(map[int]MinimalTag)

		for _, m := range markers {
			mID := strconv.Itoa(m.ID)
			sID := strconv.Itoa(sceneID)

			// Load secondary tags for this marker
			secondaryTags, err := h.repository.Tag.FindBySceneMarkerID(ctx, m.ID)
			if err != nil {
				return fmt.Errorf("loading tags for marker %d: %w", m.ID, err)
			}
			markerTags := make([]MinimalTag, 0, len(secondaryTags))
			for _, t := range secondaryTags {
				idStr := strconv.Itoa(t.ID)
				markerTags = append(markerTags, MinimalTag{
					ID:          idStr,
					Name:        t.Name,
					Color:       t.Color,
					Description: t.Description,
					ImagePath:   "/tag/" + idStr + "/image",
				})
			}

			msm := MinimalSceneMarker{
				ID:         mID,
				Title:      m.Title,
				Seconds:    m.Seconds,
				EndSeconds: m.EndSeconds,
				Tags:       markerTags,
				Stream:     "/scene/" + sID + "/scene_marker/" + mID + "/stream",
				Preview:    "/scene/" + sID + "/scene_marker/" + mID + "/preview",
				Screenshot: "/scene/" + sID + "/scene_marker/" + mID + "/screenshot",
			}

			// Load primary tag if not already loaded
			if _, ok := tagMap[m.PrimaryTagID]; !ok {
				tag, err := h.repository.Tag.Find(ctx, m.PrimaryTagID)
				if err != nil {
					return fmt.Errorf("loading primary tag %d: %w", m.PrimaryTagID, err)
				}
				if tag != nil {
					idStr := strconv.Itoa(tag.ID)
					tagMap[m.PrimaryTagID] = MinimalTag{
						ID:          idStr,
						Name:        tag.Name,
						Color:       tag.Color,
						Description: tag.Description,
						ImagePath:   "/tag/" + idStr + "/image",
					}
					tagOrder = append(tagOrder, m.PrimaryTagID)
				}
			}

			msm.PrimaryTag = tagMap[m.PrimaryTagID]
			tagMarkersMap[m.PrimaryTagID] = append(tagMarkersMap[m.PrimaryTagID], msm)
		}

		// Build result preserving tag order
		results = make([]markerTagResult, 0, len(tagOrder))
		for _, tagID := range tagOrder {
			results = append(results, markerTagResult{
				Tag:          tagMap[tagID],
				SceneMarkers: tagMarkersMap[tagID],
			})
		}

		return nil
	}) {
		return
	}

	if results == nil {
		results = []markerTagResult{}
	}

	respondOK(w, results)
}
