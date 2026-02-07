package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/utils"
)

// GET /api/v1/games/{id}
func (h *RESTHandler) findGame(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *models.Game
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		ret, err = h.repository.Game.Find(ctx, id)
		return err
	}) {
		return
	}

	if ret == nil {
		respondNotFound(w)
		return
	}

	respondOK(w, ret)
}

// POST /api/v1/games/query
func (h *RESTHandler) findGames(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Filter     *models.FindFilterType `json:"filter,omitempty"`
		GameFilter *models.GameFilterType `json:"game_filter,omitempty"`
		IDs        []string               `json:"ids,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var games []*models.Game
	var count int

	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		if len(input.IDs) > 0 {
			ids, err := stringslice.StringSliceToIntSlice(input.IDs)
			if err != nil {
				return fmt.Errorf("converting ids: %w", err)
			}
			games, err = h.repository.Game.FindMany(ctx, ids)
			if err != nil {
				return err
			}
			count = len(games)
		} else {
			var err error
			games, count, err = h.repository.Game.Query(ctx, input.GameFilter, input.Filter)
			if err != nil {
				return err
			}
		}
		return nil
	}) {
		return
	}

	respondList(w, http.StatusOK, games, count)
}

// POST /api/v1/games
func (h *RESTHandler) createGame(w http.ResponseWriter, r *http.Request) {
	var input models.GameCreateInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	if input.Title == "" {
		respondBadRequest(w, errors.New("title must not be empty"))
		return
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
			respondBadRequest(w, fmt.Errorf("parsing date: %w", err))
			return
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
			respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
			return
		}
		newGame.TagIDs = models.NewRelatedIDs(tagIDs)
	}

	if input.Image != nil && *input.Image != "" {
		imageData, err := utils.ProcessImageInput(r.Context(), *input.Image)
		if err != nil {
			respondBadRequest(w, fmt.Errorf("processing image: %w", err))
			return
		}
		newGame.Image = imageData
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.repository.Game.Create(ctx, &newGame)
	}) {
		return
	}

	var result *models.Game
	var err error
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		result, err = h.repository.Game.Find(ctx, newGame.ID)
		return err
	}) {
		return
	}

	respondCreated(w, result)
}

// PUT /api/v1/games/{id}
func (h *RESTHandler) updateGame(w http.ResponseWriter, r *http.Request) {
	gameID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var input models.GameUpdateInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}
	input.ID = strconv.Itoa(gameID)

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
				respondBadRequest(w, fmt.Errorf("parsing date: %w", err))
				return
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
			respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
			return
		}
		partial.TagIDs = &models.UpdateIDs{
			IDs:  tagIDs,
			Mode: models.RelationshipUpdateModeSet,
		}
	}

	if input.Image != nil {
		if *input.Image == "" {
			partial.Image = models.OptionalBytes{Null: true, Set: true}
		} else {
			imageData, err := utils.ProcessImageInput(r.Context(), *input.Image)
			if err != nil {
				respondBadRequest(w, fmt.Errorf("processing image: %w", err))
				return
			}
			partial.Image = models.NewOptionalBytes(imageData)
		}
	}

	var ret *models.Game
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		ret, err = h.repository.Game.UpdatePartial(ctx, gameID, partial)
		return err
	}) {
		return
	}

	respondOK(w, ret)
}

// DELETE /api/v1/games/{id}
func (h *RESTHandler) destroyGame(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.repository.Game.Destroy(ctx, id)
	}) {
		return
	}

	respondNoContent(w)
}
