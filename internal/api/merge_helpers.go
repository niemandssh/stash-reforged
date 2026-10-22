package api

import (
	"context"

	"github.com/stashapp/stash/pkg/models"
)

func studioAutoMergePartial(dest *models.Studio, sources []*models.Studio) models.StudioPartial {
	partial := models.NewStudioPartial()
	partial.ID = dest.ID

	for _, src := range sources {
		if !partial.URL.Set && dest.URL == "" && src.URL != "" {
			partial.URL = models.NewOptionalString(src.URL)
		}
		if !partial.Details.Set && dest.Details == "" && src.Details != "" {
			partial.Details = models.NewOptionalString(src.Details)
		}
		if !partial.Rating.Set && dest.Rating == nil && src.Rating != nil {
			partial.Rating = models.NewOptionalInt(*src.Rating)
		}
		if !partial.Favorite.Set && !dest.Favorite && src.Favorite {
			partial.Favorite = models.NewOptionalBool(true)
		}
	}

	return partial
}

func studioAutoMergePartialHasUpdates(partial models.StudioPartial) bool {
	return partial.URL.Set || partial.Details.Set || partial.Rating.Set || partial.Favorite.Set
}

func copyStudioImageIfMissing(ctx context.Context, qb models.StudioReaderWriter, destination int, sources []*models.Studio) error {
	hasImage, err := qb.HasImage(ctx, destination)
	if err != nil {
		return err
	}
	if hasImage {
		return nil
	}

	for _, src := range sources {
		srcHasImage, err := qb.HasImage(ctx, src.ID)
		if err != nil {
			return err
		}
		if !srcHasImage {
			continue
		}

		imageData, err := qb.GetImage(ctx, src.ID)
		if err != nil {
			return err
		}
		if len(imageData) == 0 {
			continue
		}

		return qb.UpdateImage(ctx, destination, imageData)
	}

	return nil
}

func performerAutoMergePartial(dest *models.Performer, sources []*models.Performer) models.PerformerPartial {
	partial := models.NewPerformerPartial()

	for _, src := range sources {
		if !partial.Disambiguation.Set && dest.Disambiguation == "" && src.Disambiguation != "" {
			partial.Disambiguation = models.NewOptionalString(src.Disambiguation)
		}
		if !partial.Details.Set && dest.Details == "" && src.Details != "" {
			partial.Details = models.NewOptionalString(src.Details)
		}
		if !partial.Rating.Set && dest.Rating == nil && src.Rating != nil {
			partial.Rating = models.NewOptionalInt(*src.Rating)
		}
		if !partial.Favorite.Set && !dest.Favorite && src.Favorite {
			partial.Favorite = models.NewOptionalBool(true)
		}
		if !partial.Gender.Set && dest.Gender == nil && src.Gender != nil {
			partial.Gender = models.NewOptionalString(src.Gender.String())
		}
		if !partial.Ethnicity.Set && dest.Ethnicity == "" && src.Ethnicity != "" {
			partial.Ethnicity = models.NewOptionalString(src.Ethnicity)
		}
		if !partial.Country.Set && dest.Country == "" && src.Country != "" {
			partial.Country = models.NewOptionalString(src.Country)
		}
	}

	return partial
}

func performerAutoMergePartialHasUpdates(partial models.PerformerPartial) bool {
	return partial.Disambiguation.Set || partial.Details.Set || partial.Rating.Set ||
		partial.Favorite.Set || partial.Gender.Set || partial.Ethnicity.Set || partial.Country.Set
}

func copyPerformerImageIfMissing(ctx context.Context, qb models.PerformerReaderWriter, destination int, sources []*models.Performer) error {
	hasImage, err := qb.HasImage(ctx, destination)
	if err != nil {
		return err
	}
	if hasImage {
		return nil
	}

	for _, src := range sources {
		srcHasImage, err := qb.HasImage(ctx, src.ID)
		if err != nil {
			return err
		}
		if !srcHasImage {
			continue
		}

		imageData, err := qb.GetImage(ctx, src.ID)
		if err != nil {
			return err
		}
		if len(imageData) == 0 {
			continue
		}

		return qb.UpdateImage(ctx, destination, imageData)
	}

	return nil
}
