package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

// GET /api/v1/color-presets/{id}
func (h *RESTHandler) findColorPreset(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *models.ColorPreset
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		ret, err = h.repository.ColorPreset.Find(ctx, id)
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

// POST /api/v1/color-presets/query
func (h *RESTHandler) findColorPresets(w http.ResponseWriter, r *http.Request) {
	var result []*models.ColorPreset
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		result, err = h.repository.ColorPreset.FindAll(ctx)
		return err
	}) {
		return
	}

	respondList(w, http.StatusOK, result, len(result))
}

// POST /api/v1/color-presets
func (h *RESTHandler) createColorPreset(w http.ResponseWriter, r *http.Request) {
	var input ColorPresetCreateInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	newPreset := models.NewColorPreset()
	newPreset.Name = input.Name
	newPreset.Color = input.Color

	if input.Sort != nil {
		newPreset.Sort = *input.Sort
	}
	if input.TagRequirementsDescription != nil {
		newPreset.TagRequirementsDescription = *input.TagRequirementsDescription
	}
	if input.RequiredForRequirements != nil {
		newPreset.RequiredForRequirements = *input.RequiredForRequirements
	}

	var created *models.ColorPreset
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		var err error
		created, err = h.repository.ColorPreset.Create(ctx, newPreset)
		return err
	}) {
		return
	}

	respondCreated(w, created)
}

// PUT /api/v1/color-presets/{id}
func (h *RESTHandler) updateColorPreset(w http.ResponseWriter, r *http.Request) {
	presetID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var input ColorPresetUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}
	input.ID = strconv.Itoa(presetID)

	// Build partial update
	partial := models.NewColorPresetPartial()
	if _, ok := inputMap["name"]; ok && input.Name != nil {
		partial.Name = models.NewOptionalString(*input.Name)
	}
	if _, ok := inputMap["color"]; ok && input.Color != nil {
		partial.Color = models.NewOptionalString(*input.Color)
	}
	if _, ok := inputMap["sort"]; ok && input.Sort != nil {
		partial.Sort = models.NewOptionalInt(*input.Sort)
	}
	if _, ok := inputMap["tag_requirements_description"]; ok && input.TagRequirementsDescription != nil {
		partial.TagRequirementsDescription = models.NewOptionalString(*input.TagRequirementsDescription)
	}
	if _, ok := inputMap["required_for_requirements"]; ok && input.RequiredForRequirements != nil {
		partial.RequiredForRequirements = models.NewOptionalBool(*input.RequiredForRequirements)
	}

	var result *models.ColorPreset
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		result, err = h.repository.ColorPreset.Update(ctx, presetID, partial)
		return err
	}) {
		return
	}

	respondOK(w, result)
}

// DELETE /api/v1/color-presets/{id}
func (h *RESTHandler) destroyColorPreset(w http.ResponseWriter, r *http.Request) {
	presetID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.repository.ColorPreset.Destroy(ctx, presetID)
	}) {
		return
	}

	respondNoContent(w)
}

// Helper: convert string slice to int slice for IDs, ignoring the error on empty
func toIntIDs(ids []string) ([]int, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return stringslice.StringSliceToIntSlice(ids)
}

// Helper: convert StashID pointer slice to value slice
func stashIDPtrSliceToSlice(ptrs []*models.StashID) []models.StashID {
	if ptrs == nil {
		return nil
	}
	result := make([]models.StashID, len(ptrs))
	for i, p := range ptrs {
		if p != nil {
			result[i] = *p
		}
	}
	return result
}

// Suppress unused imports
var _ = fmt.Sprintf
