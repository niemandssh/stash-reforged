package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/scene"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

// POST /api/v1/scene-markers/query
func (h *RESTHandler) findSceneMarkers(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Filter             *models.FindFilterType        `json:"filter,omitempty"`
		SceneMarkerFilter  *models.SceneMarkerFilterType `json:"scene_marker_filter,omitempty"`
		IDs                []string                      `json:"ids,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var markers []*models.SceneMarker
	var count int

	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		if len(input.IDs) > 0 {
			ids, err := stringslice.StringSliceToIntSlice(input.IDs)
			if err != nil {
				return fmt.Errorf("converting ids: %w", err)
			}
			markers, err = h.repository.SceneMarker.FindMany(ctx, ids)
			if err != nil {
				return err
			}
			count = len(markers)
		} else {
			var err error
			markers, count, err = h.repository.SceneMarker.Query(ctx, input.SceneMarkerFilter, input.Filter)
			if err != nil {
				return err
			}
		}
		return nil
	}) {
		return
	}

	respondList(w, http.StatusOK, markers, count)
}

// POST /api/v1/scene-markers
func (h *RESTHandler) createSceneMarker(w http.ResponseWriter, r *http.Request) {
	var input SceneMarkerCreateInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	sceneID, err := strconv.Atoi(input.SceneID)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting scene id: %w", err))
		return
	}

	primaryTagID, err := strconv.Atoi(input.PrimaryTagID)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting primary tag id: %w", err))
		return
	}

	newMarker := models.NewSceneMarker()
	newMarker.Title = input.Title
	newMarker.Seconds = input.Seconds
	newMarker.PrimaryTagID = primaryTagID
	newMarker.SceneID = sceneID

	if input.EndSeconds != nil {
		if *input.EndSeconds < input.Seconds {
			respondBadRequest(w, fmt.Errorf("end_seconds must be >= seconds"))
			return
		}
		newMarker.EndSeconds = input.EndSeconds
	}

	tagIDs, err := stringslice.StringSliceToIntSlice(input.TagIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}

	var ret *models.SceneMarker
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.SceneMarker
		if err := qb.Create(ctx, &newMarker); err != nil {
			return err
		}

		// Remove primary tag from tag list
		filteredTagIDs := make([]int, 0, len(tagIDs))
		for _, id := range tagIDs {
			if id != primaryTagID {
				filteredTagIDs = append(filteredTagIDs, id)
			}
		}

		if err := qb.UpdateTags(ctx, newMarker.ID, filteredTagIDs); err != nil {
			return err
		}

		ret = &newMarker
		return nil
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), ret.ID, hook.SceneMarkerCreatePost, input, nil)

	respondCreated(w, ret)
}

// PUT /api/v1/scene-markers/{id}
func (h *RESTHandler) updateSceneMarker(w http.ResponseWriter, r *http.Request) {
	markerID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var input SceneMarkerUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	input.ID = strconv.Itoa(markerID)
	translator := newRESTChangesetTranslator(inputMap)

	updatedMarker := models.SceneMarkerPartial{
		Title:        translator.optionalString(input.Title, "title"),
		Seconds:      translator.optionalFloat64(input.Seconds, "seconds"),
		EndSeconds:   translator.optionalFloat64(input.EndSeconds, "end_seconds"),
	}

	if input.SceneID != nil {
		sceneID, err := strconv.Atoi(*input.SceneID)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("converting scene id: %w", err))
			return
		}
		updatedMarker.SceneID = models.NewOptionalInt(sceneID)
	}

	if input.PrimaryTagID != nil {
		primaryTagID, err := strconv.Atoi(*input.PrimaryTagID)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("converting primary tag id: %w", err))
			return
		}
		updatedMarker.PrimaryTagID = models.NewOptionalInt(primaryTagID)
	}

	var ret *models.SceneMarker
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.SceneMarker

		ret, err = qb.UpdatePartial(ctx, markerID, updatedMarker)
		if err != nil {
			return err
		}

		// Update tags if included
		if translator.hasField("tag_ids") && len(input.TagIds) > 0 {
			tagIDs, err := stringslice.StringSliceToIntSlice(input.TagIds)
			if err != nil {
				return fmt.Errorf("converting tag ids: %w", err)
			}

			// Remove primary tag from tag list
			primaryTagID := ret.PrimaryTagID
			filteredTagIDs := make([]int, 0, len(tagIDs))
			for _, id := range tagIDs {
				if id != primaryTagID {
					filteredTagIDs = append(filteredTagIDs, id)
				}
			}

			if err := qb.UpdateTags(ctx, markerID, filteredTagIDs); err != nil {
				return err
			}
		}

		return nil
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), ret.ID, hook.SceneMarkerUpdatePost, input, translator.getFields())

	respondOK(w, ret)
}

// DELETE /api/v1/scene-markers/{id}
func (h *RESTHandler) destroySceneMarker(w http.ResponseWriter, r *http.Request) {
	markerID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	fileNamingAlgo := manager.GetInstance().Config.GetVideoFileNamingAlgorithm()

	fileDeleter := &scene.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: fileNamingAlgo,
		Paths:          manager.GetInstance().Paths,
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.SceneMarker

		marker, err := qb.Find(ctx, markerID)
		if err != nil {
			return err
		}
		if marker == nil {
			return fmt.Errorf("scene marker with id %d not found", markerID)
		}

		s, err := h.repository.Scene.Find(ctx, marker.SceneID)
		if err != nil {
			return err
		}

		return scene.DestroyMarker(ctx, s, marker, qb, fileDeleter)
	}) {
		fileDeleter.Rollback()
		return
	}

	fileDeleter.Commit()

	h.hookExecutor.ExecutePostHooks(r.Context(), markerID, hook.SceneMarkerDestroyPost, strconv.Itoa(markerID), nil)

	respondNoContent(w)
}

// DELETE /api/v1/scene-markers (body: {"ids": [...]})
func (h *RESTHandler) destroySceneMarkers(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := decodeBody(r, &body); err != nil {
		respondBadRequest(w, err)
		return
	}

	markerIDs, err := stringslice.StringSliceToIntSlice(body.IDs)
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

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.SceneMarker

		for _, id := range markerIDs {
			marker, err := qb.Find(ctx, id)
			if err != nil {
				return err
			}
			if marker == nil {
				return fmt.Errorf("scene marker with id %d not found", id)
			}

			s, err := h.repository.Scene.Find(ctx, marker.SceneID)
			if err != nil {
				return err
			}

			if err := scene.DestroyMarker(ctx, s, marker, qb, fileDeleter); err != nil {
				return err
			}
		}
		return nil
	}) {
		fileDeleter.Rollback()
		return
	}

	fileDeleter.Commit()

	for _, id := range markerIDs {
		h.hookExecutor.ExecutePostHooks(r.Context(), id, hook.SceneMarkerDestroyPost, body.IDs, nil)
	}

	respondNoContent(w)
}
