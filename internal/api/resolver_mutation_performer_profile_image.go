package api

import (
	"context"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

func (r *mutationResolver) PerformerProfileImageCreate(ctx context.Context, input PerformerProfileImageCreateInput) (*models.PerformerProfileImage, error) {
	// Validate performer exists
	performerID, err := strconv.Atoi(input.PerformerID)
	if err != nil {
		return nil, err
	}

	var ret *models.PerformerProfileImage
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		// Check if performer exists
		_, err := r.repository.Performer.Find(ctx, performerID)
		if err != nil {
			return err
		}

		// Create the profile image
		createInput := &models.CreatePerformerProfileImageInput{
			PerformerID: performerID,
			Image:       input.Image,
			IsPrimary:   input.IsPrimary,
			Position:    input.Position,
		}

		ret, err = r.repository.PerformerProfileImage.Create(ctx, createInput)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) PerformerProfileImageUpdate(ctx context.Context, input PerformerProfileImageUpdateInput) (*models.PerformerProfileImage, error) {
	imageID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, err
	}

	var ret *models.PerformerProfileImage
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		// Create partial update
		partial := models.NewPerformerProfileImagePartial()

		if input.IsPrimary != nil {
			partial.IsPrimary = models.NewOptionalBool(*input.IsPrimary)
		}

		if input.Position != nil {
			partial.Position = models.NewOptionalInt(*input.Position)
		}

		ret, err = r.repository.PerformerProfileImage.UpdatePartial(ctx, imageID, partial)
		if err != nil {
			return err
		}

		// Update image if provided
		if input.Image != nil {
			// Convert base64 or URL to bytes
			imageData, err := utils.ProcessImageInput(ctx, *input.Image)
			if err != nil {
				return err
			}

			err = r.repository.PerformerProfileImage.UpdateImage(ctx, imageID, imageData)
			if err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) PerformerProfileImageDestroy(ctx context.Context, input PerformerProfileImageDestroyInput) (bool, error) {
	imageID, err := strconv.Atoi(input.ID)
	if err != nil {
		return false, err
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		return r.repository.PerformerProfileImage.Destroy(ctx, imageID)
	}); err != nil {
		return false, err
	}

	return true, nil
}
