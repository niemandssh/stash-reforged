package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/studio"
	"github.com/stashapp/stash/pkg/utils"
)

// GET /api/v1/studios/{id}
func (h *RESTHandler) findStudio(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *models.Studio
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		ret, err = h.repository.Studio.Find(ctx, id)
		return err
	}) {
		return
	}

	if ret == nil {
		respondNotFound(w)
		return
	}

	respondOK(w, ret)
}

// POST /api/v1/studios/query
func (h *RESTHandler) findStudios(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Filter       *models.FindFilterType   `json:"filter,omitempty"`
		StudioFilter *models.StudioFilterType `json:"studio_filter,omitempty"`
		IDs          []string                 `json:"ids,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	idInts, err := stringslice.StringSliceToIntSlice(input.IDs)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var result *FindStudiosResultType
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var studios []*models.Studio
		var err error
		var total int

		if len(idInts) > 0 {
			studios, err = h.repository.Studio.FindMany(ctx, idInts)
			total = len(studios)
		} else {
			studios, total, err = h.repository.Studio.Query(ctx, input.StudioFilter, input.Filter)
		}

		if err != nil {
			return err
		}

		result = &FindStudiosResultType{
			Count:   total,
			Studios: studios,
		}
		return nil
	}) {
		return
	}

	respondList(w, http.StatusOK, result.Studios, result.Count)
}

// POST /api/v1/studios
func (h *RESTHandler) createStudio(w http.ResponseWriter, r *http.Request) {
	var input models.StudioCreateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	translator := newRESTChangesetTranslator(inputMap)

	newStudio := models.NewStudio()
	newStudio.Name = input.Name
	newStudio.URL = translator.string(input.URL)
	newStudio.Rating = input.Rating100
	newStudio.Favorite = translator.bool(input.Favorite)
	newStudio.Details = translator.string(input.Details)
	newStudio.IgnoreAutoTag = translator.bool(input.IgnoreAutoTag)
	newStudio.Aliases = models.NewRelatedStrings(input.Aliases)
	newStudio.StashIDs = models.NewRelatedStashIDs(models.StashIDInputs(input.StashIds).ToStashIDs())

	newStudio.ParentID, err = translator.intPtrFromString(input.ParentID)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting parent id: %w", err))
		return
	}

	newStudio.TagIDs, err = translator.relatedIds(input.TagIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}

	var imageData []byte
	if input.Image != nil {
		imageData, err = utils.ProcessImageInput(r.Context(), *input.Image)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("processing image: %w", err))
			return
		}
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Studio

		if err := studio.ValidateCreate(ctx, newStudio, qb); err != nil {
			return err
		}

		err = qb.Create(ctx, &newStudio)
		if err != nil {
			return err
		}

		if len(imageData) > 0 {
			if err := qb.UpdateImage(ctx, newStudio.ID, imageData); err != nil {
				return err
			}
		}

		return nil
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), newStudio.ID, hook.StudioCreatePost, input, nil)

	var result *models.Studio
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		result, err = h.repository.Studio.Find(ctx, newStudio.ID)
		return err
	}) {
		return
	}

	respondCreated(w, result)
}

// PUT /api/v1/studios/{id}
func (h *RESTHandler) updateStudio(w http.ResponseWriter, r *http.Request) {
	studioID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var input models.StudioUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	input.ID = strconv.Itoa(studioID)
	translator := newRESTChangesetTranslator(inputMap)

	updatedStudio := models.NewStudioPartial()
	updatedStudio.ID = studioID
	updatedStudio.Name = translator.optionalString(input.Name, "name")
	updatedStudio.URL = translator.optionalString(input.URL, "url")
	updatedStudio.Details = translator.optionalString(input.Details, "details")
	updatedStudio.Favorite = translator.optionalBool(input.Favorite, "favorite")
	updatedStudio.IgnoreAutoTag = translator.optionalBool(input.IgnoreAutoTag, "ignore_auto_tag")
	updatedStudio.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedStudio.Aliases = translator.updateStrings(input.Aliases, "aliases")
	updatedStudio.ParentID, err = translator.optionalIntFromString(input.ParentID, "parent_id")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting parent id: %w", err))
		return
	}

	updatedStudio.StashIDs = translator.updateStashIDs(input.StashIds, "stash_ids")
	updatedStudio.TagIDs, err = translator.updateIds(input.TagIds, "tag_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}

	var imageData []byte
	imageIncluded := translator.hasField("image")
	if input.Image != nil {
		imageData, err = utils.ProcessImageInput(r.Context(), *input.Image)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("processing image: %w", err))
			return
		}
	}

	var s *models.Studio
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Studio

		if err := studio.ValidateModify(ctx, updatedStudio, qb); err != nil {
			return err
		}

		s, err = qb.UpdatePartial(ctx, updatedStudio)
		if err != nil {
			return err
		}

		if imageIncluded {
			if err := qb.UpdateImage(ctx, studioID, imageData); err != nil {
				return err
			}
		}

		return nil
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), s.ID, hook.StudioUpdatePost, input, translator.getFields())

	var result *models.Studio
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		result, err = h.repository.Studio.Find(ctx, s.ID)
		return err
	}) {
		return
	}

	respondOK(w, result)
}

// DELETE /api/v1/studios/{id}
func (h *RESTHandler) destroyStudio(w http.ResponseWriter, r *http.Request) {
	studioID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.repository.Studio.Destroy(ctx, studioID)
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), studioID, hook.StudioDestroyPost, StudioDestroyInput{ID: strconv.Itoa(studioID)}, nil)

	respondNoContent(w)
}

// DELETE /api/v1/studios  (body: {"ids": [...]})
func (h *RESTHandler) destroyStudios(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := decodeBody(r, &body); err != nil {
		respondBadRequest(w, err)
		return
	}

	ids, err := stringslice.StringSliceToIntSlice(body.IDs)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting ids: %w", err))
		return
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Studio
		for _, id := range ids {
			if err := qb.Destroy(ctx, id); err != nil {
				return err
			}
		}
		return nil
	}) {
		return
	}

	for _, id := range ids {
		h.hookExecutor.ExecutePostHooks(r.Context(), id, hook.StudioDestroyPost, body.IDs, nil)
	}

	respondNoContent(w)
}
