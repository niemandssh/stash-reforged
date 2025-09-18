package api

import (
	"context"

	"github.com/stashapp/stash/pkg/models"
)

func (r *sceneUpdateInputResolver) VideoFilters(ctx context.Context, obj *models.SceneUpdateInput, data *VideoFiltersInput) error {
	if data == nil {
		obj.VideoFilters = nil
		return nil
	}

	obj.VideoFilters = &models.VideoFilters{
		Contrast:     data.Contrast,
		Brightness:   data.Brightness,
		Gamma:        data.Gamma,
		Saturate:     data.Saturate,
		HueRotate:    data.HueRotate,
		WhiteBalance: data.WhiteBalance,
		Red:          data.Red,
		Green:        data.Green,
		Blue:         data.Blue,
		Blur:         data.Blur,
	}

	return nil
}

func (r *sceneUpdateInputResolver) VideoTransforms(ctx context.Context, obj *models.SceneUpdateInput, data *VideoTransformsInput) error {
	if data == nil {
		obj.VideoTransforms = nil
		return nil
	}

	obj.VideoTransforms = &models.VideoTransforms{
		Rotate:      data.Rotate,
		Scale:       data.Scale,
		AspectRatio: data.AspectRatio,
	}

	return nil
}
