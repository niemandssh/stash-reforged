package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/utils"
)

func (r *mutationResolver) getGame(ctx context.Context, id int) (ret *models.Game, err error) {
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Game.Find(ctx, id)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) GameCreate(ctx context.Context, input models.GameCreateInput) (*models.Game, error) {
	if input.Title == "" {
		return nil, errors.New("title must not be empty")
	}

	newGame := models.NewGame()
	newGame.Title = input.Title

	if input.Details != nil {
		newGame.Details = *input.Details
	}
	if input.Organized != nil {
		newGame.Organized = *input.Organized
	}
	if input.Date != nil && *input.Date != "" {
		date, err := models.ParseDate(*input.Date)
		if err != nil {
			return nil, fmt.Errorf("parsing date: %w", err)
		}
		newGame.Date = &date
	}
	if input.Rating100 != nil {
		newGame.Rating = input.Rating100
	}
	if input.FolderPath != nil {
		newGame.FolderPath = *input.FolderPath
	}
	if input.ExecutablePath != nil {
		newGame.ExecutablePath = *input.ExecutablePath
	}
	if input.Urls != nil {
		newGame.URLs = models.NewRelatedStrings(input.Urls)
	}
	if input.TagIds != nil {
		tagIDs, err := stringslice.StringSliceToIntSlice(input.TagIds)
		if err != nil {
			return nil, fmt.Errorf("converting tag ids: %w", err)
		}
		newGame.TagIDs = models.NewRelatedIDs(tagIDs)
	}

	// Process the base 64 encoded image string
	if input.Image != nil && *input.Image != "" {
		imageData, err := utils.ProcessImageInput(ctx, *input.Image)
		if err != nil {
			return nil, fmt.Errorf("processing image: %w", err)
		}
		newGame.Image = imageData
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		return r.repository.Game.Create(ctx, &newGame)
	}); err != nil {
		return nil, err
	}

	return r.getGame(ctx, newGame.ID)
}

func (r *mutationResolver) GameUpdate(ctx context.Context, input models.GameUpdateInput) (ret *models.Game, err error) {
	gameID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	partial := models.NewGamePartial()

	if input.Title != nil {
		partial.Title = models.NewOptionalString(*input.Title)
	}
	if input.Details != nil {
		partial.Details = models.NewOptionalString(*input.Details)
	}
	if input.Date != nil {
		if *input.Date == "" {
			partial.Date = models.OptionalDate{Set: true, Null: true}
		} else {
			date, err := models.ParseDate(*input.Date)
			if err != nil {
				return nil, fmt.Errorf("parsing date: %w", err)
			}
			partial.Date = models.NewOptionalDate(date)
		}
	}
	if input.Rating100 != nil {
		partial.Rating = models.NewOptionalInt(*input.Rating100)
	}
	if input.Organized != nil {
		partial.Organized = models.NewOptionalBool(*input.Organized)
	}
	if input.FolderPath != nil {
		partial.FolderPath = models.NewOptionalString(*input.FolderPath)
	}
	if input.ExecutablePath != nil {
		partial.ExecutablePath = models.NewOptionalString(*input.ExecutablePath)
	}
	if input.Urls != nil {
		partial.URLs = &models.UpdateStrings{
			Values: input.Urls,
			Mode:   models.RelationshipUpdateModeSet,
		}
	}
	if input.TagIds != nil {
		tagIDs, err := stringslice.StringSliceToIntSlice(input.TagIds)
		if err != nil {
			return nil, fmt.Errorf("converting tag ids: %w", err)
		}
		partial.TagIDs = &models.UpdateIDs{
			IDs:  tagIDs,
			Mode: models.RelationshipUpdateModeSet,
		}
	}

	// Process the base 64 encoded image string
	if input.Image != nil {
		if *input.Image == "" {
			// If image is explicitly set to empty string, clear it
			partial.Image = models.OptionalBytes{Null: true, Set: true}
		} else {
			imageData, err := utils.ProcessImageInput(ctx, *input.Image)
			if err != nil {
				return nil, fmt.Errorf("processing image: %w", err)
			}
			partial.Image = models.NewOptionalBytes(imageData)
		}
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Game.UpdatePartial(ctx, gameID, partial)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) GameDestroy(ctx context.Context, input models.GameDestroyInput) (bool, error) {
	if len(input.Ids) == 0 {
		return true, nil
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		for _, id := range input.Ids {
			gameID, err := strconv.Atoi(id)
			if err != nil {
				return fmt.Errorf("converting id: %w", err)
			}

			if err := r.repository.Game.Destroy(ctx, gameID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return false, err
	}

	return true, nil
}

func (r *mutationResolver) GameAddO(ctx context.Context, id string, times []*time.Time) (*HistoryMutationResult, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var converted []time.Time
	for _, t := range times {
		if t != nil {
			converted = append(converted, t.Local())
		}
	}

	var updated []time.Time
	var count int

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Game

		updated, err = qb.AddO(ctx, gameID, converted)
		if err != nil {
			return err
		}

		count, err = qb.IncrementOCounter(ctx, gameID)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   count,
		History: sliceutil.ValuesToPtrs(updated),
	}, nil
}

func (r *mutationResolver) GameDeleteO(ctx context.Context, id string, times []*time.Time) (*HistoryMutationResult, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var converted []time.Time
	for _, t := range times {
		if t != nil {
			converted = append(converted, t.Local())
		}
	}

	var updated []time.Time
	var count int

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Game

		updated, err = qb.DeleteO(ctx, gameID, converted)
		if err != nil {
			return err
		}

		count, err = qb.DecrementOCounter(ctx, gameID)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   count,
		History: sliceutil.ValuesToPtrs(updated),
	}, nil
}

func (r *mutationResolver) GameIncrementO(ctx context.Context, id string) (int, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}

	var ret int
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Game.IncrementOCounter(ctx, gameID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *mutationResolver) GameDecrementO(ctx context.Context, id string) (int, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}

	var ret int
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Game.DecrementOCounter(ctx, gameID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *mutationResolver) GameResetO(ctx context.Context, id string) (int, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}

	var ret int
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Game.ResetOCounter(ctx, gameID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *mutationResolver) GameAddOmg(ctx context.Context, id string, times []*time.Time) (*HistoryMutationResult, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var converted []time.Time
	for _, t := range times {
		if t != nil {
			converted = append(converted, t.Local())
		}
	}

	var updated []time.Time
	var count int

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Game

		updated, err = qb.AddOMG(ctx, gameID, converted)
		if err != nil {
			return err
		}

		count, err = qb.IncrementOMGCounter(ctx, gameID)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   count,
		History: sliceutil.ValuesToPtrs(updated),
	}, nil
}

func (r *mutationResolver) GameDeleteOmg(ctx context.Context, id string, times []*time.Time) (*HistoryMutationResult, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var converted []time.Time
	for _, t := range times {
		if t != nil {
			converted = append(converted, t.Local())
		}
	}

	var updated []time.Time
	var count int

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Game

		updated, err = qb.DeleteOMG(ctx, gameID, converted)
		if err != nil {
			return err
		}

		count, err = qb.DecrementOMGCounter(ctx, gameID)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   count,
		History: sliceutil.ValuesToPtrs(updated),
	}, nil
}

func (r *mutationResolver) GameIncrementOmg(ctx context.Context, id string) (int, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}

	var ret int
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Game.IncrementOMGCounter(ctx, gameID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *mutationResolver) GameDecrementOmg(ctx context.Context, id string) (int, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}

	var ret int
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Game.DecrementOMGCounter(ctx, gameID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *mutationResolver) GameResetOmg(ctx context.Context, id string) (int, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}

	var ret int
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Game.ResetOMGCounter(ctx, gameID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *mutationResolver) GameAddView(ctx context.Context, id string, times []*time.Time) (*HistoryMutationResult, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var converted []time.Time
	for _, t := range times {
		if t != nil {
			converted = append(converted, t.Local())
		}
	}

	var updated []time.Time
	var count int

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Game

		updated, err = qb.AddViews(ctx, gameID, converted)
		if err != nil {
			return err
		}

		count, err = qb.CountViews(ctx, gameID)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   count,
		History: sliceutil.ValuesToPtrs(updated),
	}, nil
}

func (r *mutationResolver) GameDeleteView(ctx context.Context, id string, times []*time.Time) (*HistoryMutationResult, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var converted []time.Time
	for _, t := range times {
		if t != nil {
			converted = append(converted, t.Local())
		}
	}

	var updated []time.Time
	var count int

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Game

		updated, err = qb.DeleteViews(ctx, gameID, converted)
		if err != nil {
			return err
		}

		count, err = qb.CountViews(ctx, gameID)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   count,
		History: sliceutil.ValuesToPtrs(updated),
	}, nil
}

func (r *mutationResolver) GameIncrementView(ctx context.Context, id string) (int, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("converting id: %w", err)
	}

	var ret int
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Game

		ret, err = qb.CountViews(ctx, gameID)
		if err != nil {
			return err
		}

		_, err = qb.AddViews(ctx, gameID, []time.Time{time.Now()})
		if err != nil {
			return err
		}

		ret++
		return nil
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *mutationResolver) GameResetViews(ctx context.Context, id string) (int, error) {
	gameID, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}

	var ret int
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Game
		ret, err = qb.DeleteAllViews(ctx, gameID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}
