package api

import (
	"context"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

func (r *queryResolver) FindColorPreset(ctx context.Context, id string) (ret *models.ColorPreset, err error) {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.ColorPreset.Find(ctx, idInt)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) FindColorPresets(ctx context.Context, filter *models.FindFilterType, ids []string) (ret *FindColorPresetsResultType, err error) {
	idInts, err := stringslice.StringSliceToIntSlice(ids)
	if err != nil {
		return nil, err
	}

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var colorPresets []*models.ColorPreset
		var err error
		var total int

		if len(idInts) > 0 {
			// Find specific color presets by IDs
			colorPresets = make([]*models.ColorPreset, len(idInts))
			for i, id := range idInts {
				preset, err := r.repository.ColorPreset.Find(ctx, id)
				if err != nil {
					return err
				}
				if preset != nil {
					colorPresets[i] = preset
					total++
				}
			}
		} else {
			// Find all color presets
			colorPresets, err = r.repository.ColorPreset.FindAll(ctx)
			if err != nil {
				return err
			}
			total = len(colorPresets)
		}

		ret = &FindColorPresetsResultType{
			Count:        total,
			ColorPresets: colorPresets,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}
