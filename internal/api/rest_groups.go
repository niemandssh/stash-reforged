package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stashapp/stash/internal/static"
	"github.com/stashapp/stash/pkg/group"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/utils"
)

// GET /api/v1/groups/{id}
func (h *RESTHandler) findGroup(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *models.Group
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		ret, err = h.repository.Group.Find(ctx, id)
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

// POST /api/v1/groups/query
func (h *RESTHandler) findGroups(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Filter      *models.FindFilterType  `json:"filter,omitempty"`
		GroupFilter *models.GroupFilterType `json:"group_filter,omitempty"`
		IDs         []string                `json:"ids,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var groups []*models.Group
	var count int

	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		if len(input.IDs) > 0 {
			ids, err := stringslice.StringSliceToIntSlice(input.IDs)
			if err != nil {
				return fmt.Errorf("converting ids: %w", err)
			}
			groups, err = h.repository.Group.FindMany(ctx, ids)
			if err != nil {
				return err
			}
			count = len(groups)
		} else {
			var err error
			groups, count, err = h.repository.Group.Query(ctx, input.GroupFilter, input.Filter)
			if err != nil {
				return err
			}
		}
		return nil
	}) {
		return
	}

	respondList(w, http.StatusOK, groups, count)
}

// POST /api/v1/groups
func (h *RESTHandler) createGroup(w http.ResponseWriter, r *http.Request) {
	var input GroupCreateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	translator := newRESTChangesetTranslator(inputMap)

	newGroup := models.NewGroup()
	newGroup.Name = input.Name
	newGroup.Aliases = translator.string(input.Aliases)
	newGroup.Duration = input.Duration
	newGroup.Rating = input.Rating100
	newGroup.Director = translator.string(input.Director)
	newGroup.Synopsis = translator.string(input.Synopsis)

	newGroup.Date, err = translator.datePtr(input.Date)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting date: %w", err))
		return
	}
	newGroup.StudioID, err = translator.intPtrFromString(input.StudioID)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting studio id: %w", err))
		return
	}

	newGroup.TagIDs, err = translator.relatedIds(input.TagIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}

	newGroup.ContainingGroups, err = translator.groupIDDescriptions(input.ContainingGroups)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting containing group ids: %w", err))
		return
	}

	newGroup.SubGroups, err = translator.groupIDDescriptions(input.SubGroups)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting sub group ids: %w", err))
		return
	}

	if input.Urls != nil {
		newGroup.URLs = models.NewRelatedStrings(input.Urls)
	}

	var frontimageData []byte
	if input.FrontImage != nil {
		frontimageData, err = utils.ProcessImageInput(r.Context(), *input.FrontImage)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("processing front image: %w", err))
			return
		}
	}

	var backimageData []byte
	if input.BackImage != nil {
		backimageData, err = utils.ProcessImageInput(r.Context(), *input.BackImage)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("processing back image: %w", err))
			return
		}
	}

	if len(frontimageData) == 0 && len(backimageData) != 0 {
		frontimageData = static.ReadAll(static.DefaultGroupImage)
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.groupService.Create(ctx, &newGroup, frontimageData, backimageData)
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), newGroup.ID, hook.GroupCreatePost, input, nil)
	h.hookExecutor.ExecutePostHooks(r.Context(), newGroup.ID, hook.MovieCreatePost, input, nil)

	var result *models.Group
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		result, err = h.repository.Group.Find(ctx, newGroup.ID)
		return err
	}) {
		return
	}

	respondCreated(w, result)
}

// PUT /api/v1/groups/{id}
func (h *RESTHandler) updateGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var input GroupUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	input.ID = strconv.Itoa(groupID)
	translator := newRESTChangesetTranslator(inputMap)

	updatedGroup, err := groupPartialFromGroupUpdateInput(translator, input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var frontimageData []byte
	frontImageIncluded := translator.hasField("front_image")
	if input.FrontImage != nil {
		frontimageData, err = utils.ProcessImageInput(r.Context(), *input.FrontImage)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("processing front image: %w", err))
			return
		}
	}

	var backimageData []byte
	backImageIncluded := translator.hasField("back_image")
	if input.BackImage != nil {
		backimageData, err = utils.ProcessImageInput(r.Context(), *input.BackImage)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("processing back image: %w", err))
			return
		}
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		frontImage := group.ImageInput{Image: frontimageData, Set: frontImageIncluded}
		backImage := group.ImageInput{Image: backimageData, Set: backImageIncluded}

		_, err = h.groupService.UpdatePartial(ctx, groupID, updatedGroup, frontImage, backImage)
		return err
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), groupID, hook.GroupUpdatePost, input, translator.getFields())
	h.hookExecutor.ExecutePostHooks(r.Context(), groupID, hook.MovieUpdatePost, input, translator.getFields())

	var result *models.Group
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		result, err = h.repository.Group.Find(ctx, groupID)
		return err
	}) {
		return
	}

	respondOK(w, result)
}

// PUT /api/v1/groups/bulk
func (h *RESTHandler) bulkUpdateGroups(w http.ResponseWriter, r *http.Request) {
	var input BulkGroupUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	groupIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting ids: %w", err))
		return
	}

	translator := newRESTChangesetTranslator(inputMap)

	updatedGroup, err := groupPartialFromBulkGroupUpdateInput(translator, input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret []*models.Group
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		for _, groupID := range groupIDs {
			g, err := h.groupService.UpdatePartial(ctx, groupID, updatedGroup, group.ImageInput{}, group.ImageInput{})
			if err != nil {
				return err
			}
			ret = append(ret, g)
		}
		return nil
	}) {
		return
	}

	for _, g := range ret {
		h.hookExecutor.ExecutePostHooks(r.Context(), g.ID, hook.GroupUpdatePost, input, translator.getFields())
		h.hookExecutor.ExecutePostHooks(r.Context(), g.ID, hook.MovieUpdatePost, input, translator.getFields())
	}

	respondOK(w, ret)
}

// DELETE /api/v1/groups/{id}
func (h *RESTHandler) destroyGroup(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.repository.Group.Destroy(ctx, id)
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), id, hook.GroupDestroyPost, GroupDestroyInput{ID: strconv.Itoa(id)}, nil)
	h.hookExecutor.ExecutePostHooks(r.Context(), id, hook.MovieDestroyPost, GroupDestroyInput{ID: strconv.Itoa(id)}, nil)

	respondNoContent(w)
}

// DELETE /api/v1/groups (body: {"ids": [...]})
func (h *RESTHandler) destroyGroups(w http.ResponseWriter, r *http.Request) {
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
		qb := h.repository.Group
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
		h.hookExecutor.ExecutePostHooks(r.Context(), id, hook.GroupDestroyPost, body.IDs, nil)
		h.hookExecutor.ExecutePostHooks(r.Context(), id, hook.MovieDestroyPost, body.IDs, nil)
	}

	respondNoContent(w)
}

// POST /api/v1/groups/{id}/sub-groups
func (h *RESTHandler) addGroupSubGroups(w http.ResponseWriter, r *http.Request) {
	groupID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var body struct {
		SubGroups   []*GroupDescriptionInput `json:"sub_groups"`
		InsertIndex *int                     `json:"insert_index,omitempty"`
	}
	if err := decodeBody(r, &body); err != nil {
		respondBadRequest(w, err)
		return
	}

	subGroups, err := groupsDescriptionsFromGroupInput(body.SubGroups)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting sub group ids: %w", err))
		return
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.groupService.AddSubGroups(ctx, groupID, subGroups, body.InsertIndex)
	}) {
		return
	}

	respondOK(w, true)
}

// DELETE /api/v1/groups/{id}/sub-groups
func (h *RESTHandler) removeGroupSubGroups(w http.ResponseWriter, r *http.Request) {
	groupID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var body struct {
		SubGroupIDs []string `json:"sub_group_ids"`
	}
	if err := decodeBody(r, &body); err != nil {
		respondBadRequest(w, err)
		return
	}

	subGroupIDs, err := stringslice.StringSliceToIntSlice(body.SubGroupIDs)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting sub group ids: %w", err))
		return
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.groupService.RemoveSubGroups(ctx, groupID, subGroupIDs)
	}) {
		return
	}

	respondOK(w, true)
}

// POST /api/v1/groups/{id}/sub-groups/reorder
func (h *RESTHandler) reorderSubGroups(w http.ResponseWriter, r *http.Request) {
	groupID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var body struct {
		SubGroupIDs []string `json:"sub_group_ids"`
		InsertAtID  string   `json:"insert_at_id"`
		InsertAfter *bool    `json:"insert_after,omitempty"`
	}
	if err := decodeBody(r, &body); err != nil {
		respondBadRequest(w, err)
		return
	}

	subGroupIDs, err := stringslice.StringSliceToIntSlice(body.SubGroupIDs)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting sub group ids: %w", err))
		return
	}

	insertPointID, err := strconv.Atoi(body.InsertAtID)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting insert at id: %w", err))
		return
	}

	insertAfter := utils.IsTrue(body.InsertAfter)

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.groupService.ReorderSubGroups(ctx, groupID, subGroupIDs, insertPointID, insertAfter)
	}) {
		return
	}

	respondOK(w, true)
}
