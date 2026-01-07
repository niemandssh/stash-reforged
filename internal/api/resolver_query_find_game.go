package api

import (
	"context"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

func (r *queryResolver) FindGame(ctx context.Context, id string) (ret *models.Game, err error) {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Game.Find(ctx, idInt)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) FindGames(ctx context.Context, gameFilter *models.GameFilterType, filter *models.FindFilterType, ids []string) (ret *FindGamesResultType, err error) {
	idInts, err := stringslice.StringSliceToIntSlice(ids)
	if err != nil {
		return nil, err
	}

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var games []*models.Game
		var total int

		if len(idInts) > 0 {
			games, err = r.repository.Game.FindMany(ctx, idInts)
			total = len(games)
		} else {
			games, total, err = r.repository.Game.Query(ctx, gameFilter, filter)
		}

		if err != nil {
			return err
		}

		ret = &FindGamesResultType{
			Count: total,
			Games: games,
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}
