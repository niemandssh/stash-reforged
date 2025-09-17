package api

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
)

func (r *queryResolver) FindTagColors(ctx context.Context) ([]string, error) {
	var colors []string

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		// Get all tags and extract unique colors, excluding preset colors
		perPage := -1
		tags, _, err := r.repository.Tag.Query(ctx, nil, &models.FindFilterType{
			PerPage: &perPage, // Get all tags
		})
		if err != nil {
			return fmt.Errorf("finding all tags: %w", err)
		}

		// Get preset colors to exclude
		presets, err := r.repository.ColorPreset.FindAll(ctx)
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
	}); err != nil {
		return nil, err
	}

	return colors, nil
}
