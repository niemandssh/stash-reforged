package api

import (
	"context"
	"time"

	"github.com/stashapp/stash/internal/api/loaders"
	"github.com/stashapp/stash/internal/api/urlbuilders"
	"github.com/stashapp/stash/pkg/models"
)

func (r *gameResolver) OCounter(ctx context.Context, obj *models.Game) (int, error) {
	ret, err := loaders.From(ctx).GameOCount.Load(obj.ID)
	if err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *gameResolver) OmgCounter(ctx context.Context, obj *models.Game) (int, error) {
	return obj.OmegCounter, nil
}

func (r *gameResolver) OHistory(ctx context.Context, obj *models.Game) ([]*time.Time, error) {
	history, err := loaders.From(ctx).GameOHistory.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return convertTimes(history), nil
}

func (r *gameResolver) OmgHistory(ctx context.Context, obj *models.Game) ([]*time.Time, error) {
	history, err := loaders.From(ctx).GameOMGHistory.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return convertTimes(history), nil
}

func (r *gameResolver) PlayCount(ctx context.Context, obj *models.Game) (int, error) {
	ret, err := loaders.From(ctx).GameViewCount.Load(obj.ID)
	if err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *gameResolver) ViewHistory(ctx context.Context, obj *models.Game) ([]*time.Time, error) {
	history, err := loaders.From(ctx).GameViewHistory.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return convertTimes(history), nil
}

func (r *gameResolver) ImagePath(ctx context.Context, obj *models.Game) (*string, error) {
	var hasImage bool
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		hasImage, err = r.repository.Game.HasImage(ctx, obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	imagePath := urlbuilders.NewGameURLBuilder(baseURL, obj).GetGameImageURL(hasImage)
	if imagePath == "" {
		return nil, nil
	}
	return &imagePath, nil
}

func (r *gameResolver) Date(ctx context.Context, obj *models.Game) (*string, error) {
	if obj.Date == nil {
		return nil, nil
	}

	result := obj.Date.String()
	return &result, nil
}

func (r *gameResolver) Rating100(ctx context.Context, obj *models.Game) (*int, error) {
	return obj.Rating, nil
}

func (r *gameResolver) Urls(ctx context.Context, obj *models.Game) ([]string, error) {
	if !obj.URLs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadURLs(ctx, r.repository.Game)
		}); err != nil {
			return nil, err
		}
	}

	return obj.URLs.List(), nil
}

func (r *gameResolver) Tags(ctx context.Context, obj *models.Game) (ret []*models.Tag, err error) {
	if !obj.TagIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadTagIDs(ctx, r.repository.Game)
		}); err != nil {
			return nil, err
		}
	}

	var errs []error
	ret, errs = loaders.From(ctx).TagByID.LoadAll(obj.TagIDs.List())
	return ret, firstError(errs)
}

func convertTimes(times []time.Time) []*time.Time {
	ptrs := make([]*time.Time, len(times))
	for i, t := range times {
		tt := t
		ptrs[i] = &tt
	}

	return ptrs
}

type gameFilterTypeResolver struct{ *Resolver }

func (r *gameFilterTypeResolver) OmgCounter(ctx context.Context, obj *models.GameFilterType, data *models.IntCriterionInput) error {
	obj.OmegCounter = data
	return nil
}
