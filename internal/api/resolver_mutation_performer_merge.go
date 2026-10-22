package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/performer"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/utils"
)

func (r *mutationResolver) performerPartialFromUpdateInput(ctx context.Context, input models.PerformerUpdateInput, inputMap map[string]interface{}) (*models.PerformerPartial, []byte, bool, error) {
	translator := changesetTranslator{
		inputMap: inputMap,
	}

	updatedPerformer := models.NewPerformerPartial()

	var err error

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
		if err := r.validateNoLegacyURLs(translator); err != nil {
			return nil, nil, false, err
		}

		updatedPerformer.URLs = translator.updateStrings(input.Urls, "urls")
	}

	legacyURL := translator.optionalString(input.URL, "url")
	legacyTwitter := translator.optionalString(input.Twitter, "twitter")
	legacyInstagram := translator.optionalString(input.Instagram, "instagram")

	updatedPerformer.Birthdate, updatedPerformer.BirthdateDisplay, err = translator.optionalDateWithDisplay(input.Birthdate, "birthdate")
	if err != nil {
		return nil, nil, false, fmt.Errorf("converting birthdate: %w", err)
	}
	updatedPerformer.DeathDate, updatedPerformer.DeathDateDisplay, err = translator.optionalDateWithDisplay(input.DeathDate, "death_date")
	if err != nil {
		return nil, nil, false, fmt.Errorf("converting death date: %w", err)
	}

	if translator.hasField("height_cm") {
		updatedPerformer.Height = translator.optionalInt(input.HeightCm, "height_cm")
	}

	if translator.hasField("alias_list") {
		updatedPerformer.Aliases = translator.updateStrings(input.AliasList, "alias_list")
	}

	updatedPerformer.TagIDs, err = translator.updateIds(input.TagIds, "tag_ids")
	if err != nil {
		return nil, nil, false, fmt.Errorf("converting tag ids: %w", err)
	}

	if translator.hasField("primary_tag_id") {
		primaryTagID, err := translator.optionalIntFromString(input.PrimaryTagID, "primary_tag_id")
		if err != nil {
			return nil, nil, false, fmt.Errorf("converting primary tag id: %w", err)
		}
		updatedPerformer.PrimaryTagID = primaryTagID
	}

	updatedPerformer.CustomFields = input.CustomFields
	updatedPerformer.CustomFields.Full = convertMapJSONNumbers(updatedPerformer.CustomFields.Full)
	updatedPerformer.CustomFields.Partial = convertMapJSONNumbers(updatedPerformer.CustomFields.Partial)

	var imageData []byte
	imageIncluded := translator.hasField("image")
	if input.Image != nil {
		imageData, err = utils.ProcessImageInput(ctx, *input.Image)
		if err != nil {
			return nil, nil, false, fmt.Errorf("processing image: %w", err)
		}
	}

	performerID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("converting id: %w", err)
	}

	if legacyURL.Set || legacyTwitter.Set || legacyInstagram.Set {
		if err := r.handleLegacyURLs(ctx, performerID, legacyURL, legacyTwitter, legacyInstagram, &updatedPerformer); err != nil {
			return nil, nil, false, err
		}
	}

	return &updatedPerformer, imageData, imageIncluded, nil
}

func (r *mutationResolver) PerformersMerge(ctx context.Context, input PerformersMergeInput) (*models.Performer, error) {
	source, err := stringslice.StringSliceToIntSlice(input.Source)
	if err != nil {
		return nil, fmt.Errorf("converting source ids: %w", err)
	}

	destination, err := strconv.Atoi(input.Destination)
	if err != nil {
		return nil, fmt.Errorf("converting destination id: %w", err)
	}

	if len(source) == 0 {
		return nil, nil
	}

	var t *models.Performer
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Performer

		t, err = qb.Find(ctx, destination)
		if err != nil {
			return err
		}

		if t == nil {
			return fmt.Errorf("performer with id %d not found", destination)
		}

		sources, err := qb.FindMany(ctx, source)
		if err != nil {
			return err
		}

		if input.Values != nil {
			values := *input.Values
			values.ID = input.Destination

			partial, imageData, imageIncluded, err := r.performerPartialFromUpdateInput(ctx, values, getNamedUpdateInputMap(ctx, "input.values"))
			if err != nil {
				return err
			}

			if !imageIncluded {
				if err := copyPerformerImageIfMissing(ctx, qb, destination, sources); err != nil {
					return err
				}
			}

			if err := qb.Merge(ctx, source, destination); err != nil {
				return err
			}

			if err := performer.ValidateUpdate(ctx, destination, *partial, qb); err != nil {
				return err
			}

			if _, err = qb.UpdatePartial(ctx, destination, *partial); err != nil {
				return err
			}

			if imageIncluded {
				if err := qb.UpdateImage(ctx, destination, imageData); err != nil {
					return err
				}
			}

			return nil
		}

		autoPartial := performerAutoMergePartial(t, sources)
		if performerAutoMergePartialHasUpdates(autoPartial) {
			if err := performer.ValidateUpdate(ctx, destination, autoPartial, qb); err != nil {
				return err
			}

			if _, err = qb.UpdatePartial(ctx, destination, autoPartial); err != nil {
				return err
			}
		}

		if err := copyPerformerImageIfMissing(ctx, qb, destination, sources); err != nil {
			return err
		}

		return qb.Merge(ctx, source, destination)
	}); err != nil {
		return nil, err
	}

	r.hookExecutor.ExecutePostHooks(ctx, t.ID, hook.PerformerMergePost, input, nil)

	return r.getPerformer(ctx, t.ID)
}
