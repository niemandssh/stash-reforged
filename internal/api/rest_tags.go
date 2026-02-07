package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/tag"
	"github.com/stashapp/stash/pkg/utils"
)

// --- Tags CRUD handlers ---

// GET /api/v1/tags/{id}
func (h *RESTHandler) findTag(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *models.Tag
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		ret, err = h.repository.Tag.Find(ctx, id)
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

// POST /api/v1/tags/query
func (h *RESTHandler) findTags(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Filter    *models.FindFilterType `json:"filter,omitempty"`
		TagFilter *models.TagFilterType  `json:"tag_filter,omitempty"`
		IDs       []string               `json:"ids,omitempty"`
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

	var result *FindTagsResultType
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var tags []*models.Tag
		var err error
		var total int

		if len(idInts) > 0 {
			tags, err = h.repository.Tag.FindMany(ctx, idInts)
			total = len(tags)
		} else {
			tags, total, err = h.repository.Tag.Query(ctx, input.TagFilter, input.Filter)
		}

		if err != nil {
			return err
		}

		result = &FindTagsResultType{
			Count: total,
			Tags:  tags,
		}
		return nil
	}) {
		return
	}

	respondList(w, http.StatusOK, result.Tags, result.Count)
}

// GET /api/v1/tags/colors
func (h *RESTHandler) findTagColors(w http.ResponseWriter, r *http.Request) {
	var colors []string
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		// Get all tags and extract unique colors, excluding preset colors
		perPage := -1
		tags, _, err := h.repository.Tag.Query(ctx, nil, &models.FindFilterType{
			PerPage: &perPage,
		})
		if err != nil {
			return fmt.Errorf("finding all tags: %w", err)
		}

		// Get preset colors to exclude
		presets, err := h.repository.ColorPreset.FindAll(ctx)
		if err != nil {
			return fmt.Errorf("finding color presets: %w", err)
		}

		// Create a set of preset colors
		presetColors := make(map[string]bool)
		for _, preset := range presets {
			if preset.Color != "" {
				presetColors[preset.Color] = true
			}
		}

		// Extract unique colors from tags, excluding preset colors
		colorSet := make(map[string]bool)
		for _, tag := range tags {
			if tag.Color != "" && !presetColors[tag.Color] {
				colorSet[tag.Color] = true
			}
		}

		// Convert set to slice
		for color := range colorSet {
			colors = append(colors, color)
		}

		return nil
	}) {
		return
	}

	respondOK(w, colors)
}

// POST /api/v1/tags
func (h *RESTHandler) createTag(w http.ResponseWriter, r *http.Request) {
	var input TagCreateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	translator := newRESTChangesetTranslator(inputMap)

	// Build the new tag
	newTag := models.NewTag()
	newTag.Name = input.Name
	newTag.SortName = translator.string(input.SortName)
	newTag.Aliases = models.NewRelatedStrings(input.Aliases)
	newTag.Favorite = translator.bool(input.Favorite)
	newTag.Description = translator.string(input.Description)
	newTag.IgnoreAutoTag = translator.bool(input.IgnoreAutoTag)
	newTag.IsPoseTag = translator.bool(input.IsPoseTag)
	newTag.IgnoreSuggestions = translator.bool(input.IgnoreSuggestions)
	newTag.Color = translator.string(input.Color)

	if input.Weight != nil {
		newTag.Weight = *input.Weight
	}

	newTag.ParentIDs, err = translator.relatedIds(input.ParentIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting parent tag ids: %w", err))
		return
	}

	newTag.ChildIDs, err = translator.relatedIds(input.ChildIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting child tag ids: %w", err))
		return
	}

	// Process image
	var imageData []byte
	if input.Image != nil {
		imageData, err = utils.ProcessImageInput(r.Context(), *input.Image)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("processing image: %w", err))
			return
		}
	}

	// Save in transaction
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Tag

		if err := tag.ValidateCreate(ctx, newTag, qb); err != nil {
			return err
		}

		if err := qb.Create(ctx, &newTag); err != nil {
			return err
		}

		if len(imageData) > 0 {
			if err := qb.UpdateImage(ctx, newTag.ID, imageData); err != nil {
				return err
			}
		}

		return nil
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), newTag.ID, hook.TagCreatePost, input, nil)

	// Fetch the created tag
	var created *models.Tag
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		created, err = h.repository.Tag.Find(ctx, newTag.ID)
		return err
	}) {
		return
	}

	respondCreated(w, created)
}

// PUT /api/v1/tags/{id}
func (h *RESTHandler) updateTag(w http.ResponseWriter, r *http.Request) {
	tagID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var input TagUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	// Override ID from URL
	input.ID = strconv.Itoa(tagID)

	translator := newRESTChangesetTranslator(inputMap)

	// Build partial update
	updatedTag := models.NewTagPartial()
	updatedTag.Name = translator.optionalString(input.Name, "name")
	updatedTag.SortName = translator.optionalString(input.SortName, "sort_name")
	updatedTag.Favorite = translator.optionalBool(input.Favorite, "favorite")
	updatedTag.IgnoreAutoTag = translator.optionalBool(input.IgnoreAutoTag, "ignore_auto_tag")
	updatedTag.IsPoseTag = translator.optionalBool(input.IsPoseTag, "is_pose_tag")
	updatedTag.IgnoreSuggestions = translator.optionalBool(input.IgnoreSuggestions, "ignore_suggestions")
	updatedTag.Description = translator.optionalString(input.Description, "description")
	updatedTag.Color = translator.optionalString(input.Color, "color")
	updatedTag.Weight = translator.optionalFloat64(input.Weight, "weight")
	updatedTag.Aliases = translator.updateStrings(input.Aliases, "aliases")

	updatedTag.ParentIDs, err = translator.updateIds(input.ParentIds, "parent_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting parent tag ids: %w", err))
		return
	}

	updatedTag.ChildIDs, err = translator.updateIds(input.ChildIds, "child_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting child tag ids: %w", err))
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

	var t *models.Tag
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Tag

		if err := tag.ValidateUpdate(ctx, tagID, updatedTag, qb); err != nil {
			return err
		}

		t, err = qb.UpdatePartial(ctx, tagID, updatedTag)
		if err != nil {
			return err
		}

		if imageIncluded {
			if err := qb.UpdateImage(ctx, tagID, imageData); err != nil {
				return err
			}
		}

		return nil
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), t.ID, hook.TagUpdatePost, input, translator.getFields())

	// Re-fetch the tag
	var result *models.Tag
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		result, err = h.repository.Tag.Find(ctx, t.ID)
		return err
	}) {
		return
	}

	respondOK(w, result)
}

// PUT /api/v1/tags/bulk
func (h *RESTHandler) bulkUpdateTags(w http.ResponseWriter, r *http.Request) {
	var input BulkTagUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	tagIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting ids: %w", err))
		return
	}

	translator := newRESTChangesetTranslator(inputMap)

	updatedTag := models.NewTagPartial()
	updatedTag.Description = translator.optionalString(input.Description, "description")
	updatedTag.Favorite = translator.optionalBool(input.Favorite, "favorite")
	updatedTag.IgnoreAutoTag = translator.optionalBool(input.IgnoreAutoTag, "ignore_auto_tag")
	updatedTag.IsPoseTag = translator.optionalBool(input.IsPoseTag, "is_pose_tag")
	updatedTag.IgnoreSuggestions = translator.optionalBool(input.IgnoreSuggestions, "ignore_suggestions")
	updatedTag.Color = translator.optionalString(input.Color, "color")
	updatedTag.Weight = translator.optionalFloat64(input.Weight, "weight")
	updatedTag.Aliases = translator.updateStringsBulk(input.Aliases, "aliases")

	updatedTag.ParentIDs, err = translator.updateIdsBulk(input.ParentIds, "parent_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting parent tag ids: %w", err))
		return
	}

	updatedTag.ChildIDs, err = translator.updateIdsBulk(input.ChildIds, "child_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting child tag ids: %w", err))
		return
	}

	ret := []*models.Tag{}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Tag

		for _, tagID := range tagIDs {
			if err := tag.ValidateUpdate(ctx, tagID, updatedTag, qb); err != nil {
				return err
			}

			t, err := qb.UpdatePartial(ctx, tagID, updatedTag)
			if err != nil {
				return err
			}

			ret = append(ret, t)
		}

		return nil
	}) {
		return
	}

	// Execute post hooks and re-fetch
	var newRet []*models.Tag
	for _, t := range ret {
		h.hookExecutor.ExecutePostHooks(r.Context(), t.ID, hook.TagUpdatePost, input, translator.getFields())

		var fetched *models.Tag
		if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
			var err error
			fetched, err = h.repository.Tag.Find(ctx, t.ID)
			return err
		}) {
			return
		}
		newRet = append(newRet, fetched)
	}

	respondOK(w, newRet)
}

// DELETE /api/v1/tags/{id}
func (h *RESTHandler) destroyTag(w http.ResponseWriter, r *http.Request) {
	tagID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.repository.Tag.Destroy(ctx, tagID)
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), tagID, hook.TagDestroyPost, TagDestroyInput{ID: strconv.Itoa(tagID)}, nil)

	respondNoContent(w)
}

// DELETE /api/v1/tags  (body: {"ids": ["1", "2", ...]})
func (h *RESTHandler) destroyTags(w http.ResponseWriter, r *http.Request) {
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
		qb := h.repository.Tag
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
		h.hookExecutor.ExecutePostHooks(r.Context(), id, hook.TagDestroyPost, body.IDs, nil)
	}

	respondNoContent(w)
}

// POST /api/v1/tags/merge
func (h *RESTHandler) mergeTags(w http.ResponseWriter, r *http.Request) {
	var input TagsMergeInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	source, err := stringslice.StringSliceToIntSlice(input.Source)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting source ids: %w", err))
		return
	}

	destination, err := strconv.Atoi(input.Destination)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting destination id: %w", err))
		return
	}

	if len(source) == 0 {
		respondBadRequest(w, fmt.Errorf("source ids cannot be empty"))
		return
	}

	var t *models.Tag
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Tag

		t, err = qb.Find(ctx, destination)
		if err != nil {
			return err
		}

		if t == nil {
			return fmt.Errorf("tag with id %d not found", destination)
		}

		parents, children, err := tag.MergeHierarchy(ctx, destination, source, qb)
		if err != nil {
			return err
		}

		if err = qb.Merge(ctx, source, destination); err != nil {
			return err
		}

		if err = qb.UpdateParentTags(ctx, destination, parents); err != nil {
			return err
		}
		if err = qb.UpdateChildTags(ctx, destination, children); err != nil {
			return err
		}

		return tag.ValidateHierarchyExisting(ctx, t, parents, children, qb)
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), t.ID, hook.TagMergePost, input, nil)

	respondOK(w, t)
}
