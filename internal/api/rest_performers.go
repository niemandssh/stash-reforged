package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/performer"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/utils"
)

// GET /api/v1/performers/{id}
func (h *RESTHandler) findPerformer(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *models.Performer
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		ret, err = h.repository.Performer.Find(ctx, id)
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

// POST /api/v1/performers/query
func (h *RESTHandler) findPerformers(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Filter          *models.FindFilterType      `json:"filter,omitempty"`
		PerformerFilter *models.PerformerFilterType  `json:"performer_filter,omitempty"`
		IDs             []string                     `json:"ids,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var performers []*models.Performer
	var count int

	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		if len(input.IDs) > 0 {
			ids, err := stringslice.StringSliceToIntSlice(input.IDs)
			if err != nil {
				return fmt.Errorf("converting ids: %w", err)
			}
			performers, err = h.repository.Performer.FindMany(ctx, ids)
			if err != nil {
				return err
			}
			count = len(performers)
		} else {
			var err error
			performers, count, err = h.repository.Performer.Query(ctx, input.PerformerFilter, input.Filter)
			if err != nil {
				return err
			}
		}
		return nil
	}) {
		return
	}

	respondList(w, http.StatusOK, performers, count)
}

// POST /api/v1/performers
func (h *RESTHandler) createPerformer(w http.ResponseWriter, r *http.Request) {
	var input models.PerformerCreateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	translator := newRESTChangesetTranslator(inputMap)

	newPerformer := models.NewPerformer()

	newPerformer.Name = input.Name
	newPerformer.Disambiguation = translator.string(input.Disambiguation)
	newPerformer.Aliases = models.NewRelatedStrings(input.AliasList)
	newPerformer.Gender = input.Gender
	newPerformer.Ethnicity = translator.string(input.Ethnicity)
	newPerformer.Country = translator.string(input.Country)
	newPerformer.EyeColor = translator.string(input.EyeColor)
	newPerformer.Measurements = translator.string(input.Measurements)
	newPerformer.FakeTits = translator.string(input.FakeTits)
	newPerformer.PenisLength = input.PenisLength
	newPerformer.Circumcised = input.Circumcised
	newPerformer.CareerLength = translator.string(input.CareerLength)
	newPerformer.Tattoos = translator.string(input.Tattoos)
	newPerformer.Piercings = translator.string(input.Piercings)
	newPerformer.Favorite = translator.bool(input.Favorite)
	newPerformer.Rating = input.Rating100
	newPerformer.Details = translator.string(input.Details)
	newPerformer.HairColor = translator.string(input.HairColor)
	newPerformer.Height = input.HeightCm
	newPerformer.Weight = input.Weight
	newPerformer.IgnoreAutoTag = translator.bool(input.IgnoreAutoTag)
	newPerformer.SmallRole = translator.bool(input.SmallRole)
	newPerformer.StashIDs = models.NewRelatedStashIDs(models.StashIDInputs(input.StashIds).ToStashIDs())

	newPerformer.URLs = models.NewRelatedStrings([]string{})
	if input.URL != nil {
		newPerformer.URLs.Add(*input.URL)
	}
	if input.Twitter != nil {
		newPerformer.URLs.Add(utils.URLFromHandle(*input.Twitter, twitterURL))
	}
	if input.Instagram != nil {
		newPerformer.URLs.Add(utils.URLFromHandle(*input.Instagram, instagramURL))
	}
	if input.Urls != nil {
		newPerformer.URLs.Add(input.Urls...)
	}

	newPerformer.Birthdate, err = translator.datePtr(input.Birthdate)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting birthdate: %w", err))
		return
	}
	newPerformer.DeathDate, err = translator.datePtr(input.DeathDate)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting death date: %w", err))
		return
	}

	newPerformer.TagIDs, err = translator.relatedIds(input.TagIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}

	primaryTagID, err := translator.optionalIntFromString(input.PrimaryTagID, "primary_tag_id")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting primary tag id: %w", err))
		return
	}
	newPerformer.PrimaryTagID = primaryTagID.Ptr()

	var imageData []byte
	if input.Image != nil {
		imageData, err = utils.ProcessImageInput(r.Context(), *input.Image)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("processing image: %w", err))
			return
		}
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Performer

		if err := performer.ValidateCreate(ctx, newPerformer, qb); err != nil {
			return err
		}

		i := &models.CreatePerformerInput{
			Performer:    &newPerformer,
			CustomFields: convertMapJSONNumbers(input.CustomFields),
		}

		err = qb.Create(ctx, i)
		if err != nil {
			return err
		}

		if len(imageData) > 0 {
			if err := qb.UpdateImage(ctx, newPerformer.ID, imageData); err != nil {
				return err
			}
		}

		return nil
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), newPerformer.ID, hook.PerformerCreatePost, input, nil)

	var result *models.Performer
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		result, err = h.repository.Performer.Find(ctx, newPerformer.ID)
		return err
	}) {
		return
	}

	respondCreated(w, result)
}

// PUT /api/v1/performers/{id}
func (h *RESTHandler) updatePerformer(w http.ResponseWriter, r *http.Request) {
	performerID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var input models.PerformerUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	input.ID = strconv.Itoa(performerID)
	translator := newRESTChangesetTranslator(inputMap)

	updatedPerformer := models.NewPerformerPartial()

	updatedPerformer.Name = translator.optionalString(input.Name, "name")
	updatedPerformer.Disambiguation = translator.optionalString(input.Disambiguation, "disambiguation")
	updatedPerformer.Gender = translator.optionalString((*string)(input.Gender), "gender")
	updatedPerformer.Ethnicity = translator.optionalString(input.Ethnicity, "ethnicity")
	updatedPerformer.Country = translator.optionalString(input.Country, "country")
	updatedPerformer.EyeColor = translator.optionalString(input.EyeColor, "eye_color")
	updatedPerformer.Measurements = translator.optionalString(input.Measurements, "measurements")
	updatedPerformer.FakeTits = translator.optionalString(input.FakeTits, "fake_tits")
	updatedPerformer.PenisLength = translator.optionalFloat64(input.PenisLength, "penis_length")
	updatedPerformer.Circumcised = translator.optionalString((*string)(input.Circumcised), "circumcised")
	updatedPerformer.CareerLength = translator.optionalString(input.CareerLength, "career_length")
	updatedPerformer.Tattoos = translator.optionalString(input.Tattoos, "tattoos")
	updatedPerformer.Piercings = translator.optionalString(input.Piercings, "piercings")
	updatedPerformer.Favorite = translator.optionalBool(input.Favorite, "favorite")
	updatedPerformer.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedPerformer.Details = translator.optionalString(input.Details, "details")
	updatedPerformer.HairColor = translator.optionalString(input.HairColor, "hair_color")
	updatedPerformer.Weight = translator.optionalInt(input.Weight, "weight")
	updatedPerformer.IgnoreAutoTag = translator.optionalBool(input.IgnoreAutoTag, "ignore_auto_tag")
	updatedPerformer.SmallRole = translator.optionalBool(input.SmallRole, "small_role")
	updatedPerformer.StashIDs = translator.updateStashIDs(input.StashIds, "stash_ids")

	if translator.hasField("urls") {
		updatedPerformer.URLs = translator.updateStrings(input.Urls, "urls")
	}

	updatedPerformer.Birthdate, err = translator.optionalDate(input.Birthdate, "birthdate")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting birthdate: %w", err))
		return
	}
	updatedPerformer.DeathDate, err = translator.optionalDate(input.DeathDate, "death_date")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting death date: %w", err))
		return
	}

	if translator.hasField("height_cm") {
		updatedPerformer.Height = translator.optionalInt(input.HeightCm, "height_cm")
	}

	if translator.hasField("alias_list") {
		updatedPerformer.Aliases = translator.updateStrings(input.AliasList, "alias_list")
	}

	updatedPerformer.TagIDs, err = translator.updateIds(input.TagIds, "tag_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}

	if translator.hasField("primary_tag_id") {
		primaryTagID, err := translator.optionalIntFromString(input.PrimaryTagID, "primary_tag_id")
		if err != nil {
			respondBadRequest(w, fmt.Errorf("converting primary tag id: %w", err))
			return
		}
		updatedPerformer.PrimaryTagID = primaryTagID
	}

	updatedPerformer.CustomFields = input.CustomFields
	updatedPerformer.CustomFields.Full = convertMapJSONNumbers(updatedPerformer.CustomFields.Full)
	updatedPerformer.CustomFields.Partial = convertMapJSONNumbers(updatedPerformer.CustomFields.Partial)

	var imageData []byte
	imageIncluded := translator.hasField("image")
	if input.Image != nil {
		imageData, err = utils.ProcessImageInput(r.Context(), *input.Image)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("processing image: %w", err))
			return
		}
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Performer

		if err := performer.ValidateUpdate(ctx, performerID, updatedPerformer, qb); err != nil {
			return err
		}

		_, err = qb.UpdatePartial(ctx, performerID, updatedPerformer)
		if err != nil {
			return err
		}

		if imageIncluded {
			if err := qb.UpdateImage(ctx, performerID, imageData); err != nil {
				return err
			}
		}

		return nil
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), performerID, hook.PerformerUpdatePost, input, translator.getFields())

	var result *models.Performer
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		result, err = h.repository.Performer.Find(ctx, performerID)
		return err
	}) {
		return
	}

	respondOK(w, result)
}

// PUT /api/v1/performers/bulk
func (h *RESTHandler) bulkUpdatePerformers(w http.ResponseWriter, r *http.Request) {
	var input BulkPerformerUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	performerIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting ids: %w", err))
		return
	}

	translator := newRESTChangesetTranslator(inputMap)

	updatedPerformer := models.NewPerformerPartial()
	updatedPerformer.Disambiguation = translator.optionalString(input.Disambiguation, "disambiguation")
	updatedPerformer.Gender = translator.optionalString((*string)(input.Gender), "gender")
	updatedPerformer.Ethnicity = translator.optionalString(input.Ethnicity, "ethnicity")
	updatedPerformer.Country = translator.optionalString(input.Country, "country")
	updatedPerformer.EyeColor = translator.optionalString(input.EyeColor, "eye_color")
	updatedPerformer.Measurements = translator.optionalString(input.Measurements, "measurements")
	updatedPerformer.FakeTits = translator.optionalString(input.FakeTits, "fake_tits")
	updatedPerformer.PenisLength = translator.optionalFloat64(input.PenisLength, "penis_length")
	updatedPerformer.Circumcised = translator.optionalString((*string)(input.Circumcised), "circumcised")
	updatedPerformer.CareerLength = translator.optionalString(input.CareerLength, "career_length")
	updatedPerformer.Tattoos = translator.optionalString(input.Tattoos, "tattoos")
	updatedPerformer.Piercings = translator.optionalString(input.Piercings, "piercings")
	updatedPerformer.Favorite = translator.optionalBool(input.Favorite, "favorite")
	updatedPerformer.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedPerformer.Details = translator.optionalString(input.Details, "details")
	updatedPerformer.HairColor = translator.optionalString(input.HairColor, "hair_color")
	updatedPerformer.Weight = translator.optionalInt(input.Weight, "weight")
	updatedPerformer.IgnoreAutoTag = translator.optionalBool(input.IgnoreAutoTag, "ignore_auto_tag")

	if translator.hasField("urls") {
		updatedPerformer.URLs = translator.updateStringsBulk(input.Urls, "urls")
	}

	updatedPerformer.Birthdate, err = translator.optionalDate(input.Birthdate, "birthdate")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting birthdate: %w", err))
		return
	}
	updatedPerformer.DeathDate, err = translator.optionalDate(input.DeathDate, "death_date")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting death date: %w", err))
		return
	}

	if translator.hasField("height_cm") {
		updatedPerformer.Height = translator.optionalInt(input.HeightCm, "height_cm")
	}

	if translator.hasField("alias_list") {
		updatedPerformer.Aliases = translator.updateStringsBulk(input.AliasList, "alias_list")
	}

	updatedPerformer.TagIDs, err = translator.updateIdsBulk(input.TagIds, "tag_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}

	var ret []*models.Performer
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Performer

		for _, performerID := range performerIDs {
			if err := performer.ValidateUpdate(ctx, performerID, updatedPerformer, qb); err != nil {
				return err
			}

			p, err := qb.UpdatePartial(ctx, performerID, updatedPerformer)
			if err != nil {
				return err
			}

			ret = append(ret, p)
		}

		return nil
	}) {
		return
	}

	for _, p := range ret {
		h.hookExecutor.ExecutePostHooks(r.Context(), p.ID, hook.PerformerUpdatePost, input, translator.getFields())
	}

	respondOK(w, ret)
}

// DELETE /api/v1/performers/{id}
func (h *RESTHandler) destroyPerformer(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.repository.Performer.Destroy(ctx, id)
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), id, hook.PerformerDestroyPost, PerformerDestroyInput{ID: strconv.Itoa(id)}, nil)

	respondNoContent(w)
}

// DELETE /api/v1/performers (body: {"ids": [...]})
func (h *RESTHandler) destroyPerformers(w http.ResponseWriter, r *http.Request) {
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
		qb := h.repository.Performer
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
		h.hookExecutor.ExecutePostHooks(r.Context(), id, hook.PerformerDestroyPost, body.IDs, nil)
	}

	respondNoContent(w)
}
