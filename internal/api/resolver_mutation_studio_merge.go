package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/studio"
	"github.com/stashapp/stash/pkg/utils"
)

func (r *mutationResolver) studioPartialFromUpdateInput(ctx context.Context, input models.StudioUpdateInput, inputMap map[string]interface{}) (*models.StudioPartial, []byte, bool, error) {
	translator := changesetTranslator{
		inputMap: inputMap,
	}

	studioID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("converting id: %w", err)
	}

	updatedStudio := models.NewStudioPartial()
	updatedStudio.ID = studioID

	updatedStudio.Name = translator.optionalString(input.Name, "name")
	updatedStudio.URL = translator.optionalString(input.URL, "url")
	updatedStudio.Details = translator.optionalString(input.Details, "details")
	updatedStudio.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedStudio.Favorite = translator.optionalBool(input.Favorite, "favorite")
	updatedStudio.IgnoreAutoTag = translator.optionalBool(input.IgnoreAutoTag, "ignore_auto_tag")
	updatedStudio.Aliases = translator.updateStrings(input.Aliases, "aliases")
	updatedStudio.StashIDs = translator.updateStashIDs(input.StashIds, "stash_ids")

	updatedStudio.ParentID, err = translator.optionalIntFromString(input.ParentID, "parent_id")
	if err != nil {
		return nil, nil, false, fmt.Errorf("converting parent id: %w", err)
	}

	updatedStudio.TagIDs, err = translator.updateIds(input.TagIds, "tag_ids")
	if err != nil {
		return nil, nil, false, fmt.Errorf("converting tag ids: %w", err)
	}

	var imageData []byte
	imageIncluded := translator.hasField("image")
	if input.Image != nil {
		imageData, err = utils.ProcessImageInput(ctx, *input.Image)
		if err != nil {
			return nil, nil, false, fmt.Errorf("processing image: %w", err)
		}
	}

	return &updatedStudio, imageData, imageIncluded, nil
}

func (r *mutationResolver) StudiosMerge(ctx context.Context, input StudiosMergeInput) (*models.Studio, error) {
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

	var t *models.Studio
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Studio

		t, err = qb.Find(ctx, destination)
		if err != nil {
			return err
		}

		if t == nil {
			return fmt.Errorf("studio with id %d not found", destination)
		}

		sources, err := qb.FindMany(ctx, source)
		if err != nil {
			return err
		}

		if input.Values != nil {
			values := *input.Values
			values.ID = input.Destination

			partial, imageData, imageIncluded, err := r.studioPartialFromUpdateInput(ctx, values, getNamedUpdateInputMap(ctx, "input.values"))
			if err != nil {
				return err
			}

			if !imageIncluded {
				if err := copyStudioImageIfMissing(ctx, qb, destination, sources); err != nil {
					return err
				}
			}

			if err := qb.Merge(ctx, source, destination); err != nil {
				return err
			}

			if err := studio.ValidateModify(ctx, *partial, qb); err != nil {
				return err
			}

			if _, err = qb.UpdatePartial(ctx, *partial); err != nil {
				return err
			}

			if imageIncluded {
				if err := qb.UpdateImage(ctx, destination, imageData); err != nil {
					return err
				}
			}

			return nil
		}

		autoPartial := studioAutoMergePartial(t, sources)
		if studioAutoMergePartialHasUpdates(autoPartial) {
			if err := studio.ValidateModify(ctx, autoPartial, qb); err != nil {
				return err
			}

			if _, err = qb.UpdatePartial(ctx, autoPartial); err != nil {
				return err
			}
		}

		if err := copyStudioImageIfMissing(ctx, qb, destination, sources); err != nil {
			return err
		}

		return qb.Merge(ctx, source, destination)
	}); err != nil {
		return nil, err
	}

	r.hookExecutor.ExecutePostHooks(ctx, t.ID, hook.StudioMergePost, input, nil)

	return r.getStudio(ctx, t.ID)
}
