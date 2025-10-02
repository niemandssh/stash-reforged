package api

import (
	"context"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

func (r *mutationResolver) ColorPresetCreate(ctx context.Context, input ColorPresetCreateInput) (*models.ColorPreset, error) {
	// Populate color preset from the input
	newColorPreset := models.NewColorPreset()
	newColorPreset.Name = input.Name
	newColorPreset.Color = input.Color
	if input.Sort != nil {
		newColorPreset.Sort = *input.Sort
	}

	// Start the transaction and save the color preset
	var colorPreset *models.ColorPreset
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		var err error
		colorPreset, err = r.repository.ColorPreset.Create(ctx, newColorPreset)
		return err
	}); err != nil {
		return nil, err
	}

	return colorPreset, nil
}

func (r *mutationResolver) ColorPresetUpdate(ctx context.Context, input ColorPresetUpdateInput) (*models.ColorPreset, error) {
	colorPresetID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, err
	}

	// Populate color preset from the input
	updatedColorPreset := models.NewColorPresetPartial()

	// Set fields directly if they are provided
	if input.Name != nil {
		updatedColorPreset.Name = models.NewOptionalString(*input.Name)
	}
	if input.Color != nil {
		updatedColorPreset.Color = models.NewOptionalString(*input.Color)
	}
	if input.Sort != nil {
		updatedColorPreset.Sort = models.NewOptionalInt(*input.Sort)
	}

	// Start the transaction and save the color preset
	var colorPreset *models.ColorPreset
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		var err error
		colorPreset, err = r.repository.ColorPreset.Update(ctx, colorPresetID, updatedColorPreset)
		return err
	}); err != nil {
		return nil, err
	}

	return colorPreset, nil
}

func (r *mutationResolver) ColorPresetDestroy(ctx context.Context, input ColorPresetDestroyInput) (bool, error) {
	colorPresetID, err := strconv.Atoi(input.ID)
	if err != nil {
		return false, err
	}

	// Start the transaction and delete the color preset
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		return r.repository.ColorPreset.Destroy(ctx, colorPresetID)
	}); err != nil {
		return false, err
	}

	return true, nil
}
