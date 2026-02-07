package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

// GET /api/v1/filters/{id}
func (h *RESTHandler) findSavedFilter(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *models.SavedFilter
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		ret, err = h.repository.SavedFilter.Find(ctx, id)
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

// GET /api/v1/filters?mode=...
func (h *RESTHandler) findSavedFilters(w http.ResponseWriter, r *http.Request) {
	modeStr := queryParam(r, "mode")

	var result []*models.SavedFilter
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		if modeStr != "" {
			mode := models.FilterMode(modeStr)
			result, err = h.repository.SavedFilter.FindByMode(ctx, mode)
		} else {
			result, err = h.repository.SavedFilter.All(ctx)
		}
		return err
	}) {
		return
	}

	respondOK(w, result)
}

// POST /api/v1/filters
func (h *RESTHandler) saveFilter(w http.ResponseWriter, r *http.Request) {
	var input SaveFilterInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var result *models.SavedFilter
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.SavedFilter

		filter := models.SavedFilter{
			Mode:         input.Mode,
			Name:         input.Name,
			FindFilter:   input.FindFilter,
			ObjectFilter: input.ObjectFilter,
			UIOptions:    input.UIOptions,
		}

		if input.ID != nil {
			filterID, err := strconv.Atoi(*input.ID)
			if err != nil {
				return fmt.Errorf("converting id: %w", err)
			}
			filter.ID = filterID

			err = qb.Update(ctx, &filter)
			if err != nil {
				return err
			}
			result = &filter
		} else {
			err := qb.Create(ctx, &filter)
			if err != nil {
				return err
			}
			result = &filter
		}

		return nil
	}) {
		return
	}

	if input.ID != nil {
		respondOK(w, result)
	} else {
		respondCreated(w, result)
	}
}

// DELETE /api/v1/filters/{id}
func (h *RESTHandler) destroySavedFilter(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.repository.SavedFilter.Destroy(ctx, id)
	}) {
		return
	}

	respondNoContent(w)
}

// GET /api/v1/filters/default?mode=...
func (h *RESTHandler) findDefaultFilter(w http.ResponseWriter, r *http.Request) {
	modeStr := queryParam(r, "mode")
	if modeStr == "" {
		respondBadRequest(w, fmt.Errorf("mode parameter is required"))
		return
	}

	mode := models.FilterMode(modeStr)

	// Read from the config (deprecated storage)
	cfg := config.GetInstance()
	uiConfig := cfg.GetUIConfiguration()
	if uiConfig == nil {
		respondOK(w, nil)
		return
	}

	m := utils.NestedMap(uiConfig)
	filterRaw, _ := m.Get("defaultFilters." + strings.ToLower(mode.String()))

	if filterRaw == nil {
		respondOK(w, nil)
		return
	}

	ret := &models.SavedFilter{}
	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:          "json",
		WeaklyTypedInput: true,
		Result:           ret,
	})
	if err != nil {
		respondInternalError(w, err)
		return
	}

	if err := d.Decode(filterRaw); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, ret)
}

// POST /api/v1/filters/default
func (h *RESTHandler) setDefaultFilter(w http.ResponseWriter, r *http.Request) {
	var input SetDefaultFilterInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	cfg := config.GetInstance()
	uiConfig := cfg.GetUIConfiguration()
	if uiConfig == nil {
		uiConfig = make(map[string]interface{})
	}

	m := utils.NestedMap(uiConfig)

	if input.FindFilter == nil && input.ObjectFilter == nil && input.UIOptions == nil {
		// Clearing default filter
		m.Delete("defaultFilters." + strings.ToLower(input.Mode.String()))
		cfg.SetUIConfiguration(m)

		if err := cfg.Write(); err != nil {
			respondInternalError(w, err)
			return
		}

		respondOK(w, true)
		return
	}

	subMap := make(map[string]interface{})
	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:          "json",
		WeaklyTypedInput: true,
		Result:           &subMap,
	})
	if err != nil {
		respondInternalError(w, err)
		return
	}

	if err := d.Decode(input); err != nil {
		respondInternalError(w, err)
		return
	}

	m.Set("defaultFilters."+strings.ToLower(input.Mode.String()), subMap)
	cfg.SetUIConfiguration(m)

	if err := cfg.Write(); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, true)
}
