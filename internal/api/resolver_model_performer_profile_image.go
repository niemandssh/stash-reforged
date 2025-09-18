package api

import (
	"context"

	"github.com/stashapp/stash/internal/api/urlbuilders"
	"github.com/stashapp/stash/pkg/models"
)

func (r *performerProfileImageResolver) ImagePath(ctx context.Context, obj *models.PerformerProfileImage) (*string, error) {
	// Check if the image exists
	var hasImage bool
	var performer *models.Performer
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		hasImage, err = r.repository.PerformerProfileImage.HasImage(ctx, obj.ID)
		if err != nil {
			return err
		}

		performer, err = r.repository.Performer.Find(ctx, obj.PerformerID)
		return err
	})
	if err != nil {
		return nil, err
	}

	if hasImage && performer != nil {
		baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
		urlBuilder := urlbuilders.NewPerformerURLBuilder(baseURL, performer)
		imagePath := urlBuilder.GetPerformerProfileImageURL(obj.ID, obj.UpdatedAt.Unix())
		return &imagePath, nil
	}

	return nil, nil
}
