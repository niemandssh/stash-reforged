package sqlite

import (
	"context"

	"github.com/stashapp/stash/pkg/models"
)

type gameFilterHandler struct {
	gameFilter *models.GameFilterType
}

func (h *gameFilterHandler) criterionHandler() criterionHandler {
	filter := h.gameFilter
	if filter == nil {
		return compoundHandler{}
	}
	return compoundHandler{
		intCriterionHandler(filter.ID, "games.id", nil),
		stringCriterionHandler(filter.Title, "games.title"),
		stringCriterionHandler(filter.Details, "games.details"),
		h.urlsCriterionHandler(filter.URL),
		&dateCriterionHandler{filter.Date, "games.date", nil},
		intCriterionHandler(filter.Rating100, "games.rating", nil),
		boolCriterionHandler(filter.Organized, "games.organized", nil),
		intCriterionHandler(filter.OCounter, "games.o_counter", nil),
		intCriterionHandler(filter.OmegCounter, "games.omg_counter", nil),
		h.tagsCriterionHandler(filter.TagIDs),
		&timestampCriterionHandler{filter.CreatedAt, "games.created_at", nil},
		&timestampCriterionHandler{filter.UpdatedAt, "games.updated_at", nil},
	}
}

func (h *gameFilterHandler) validate() error {
	if h.gameFilter == nil {
		return nil
	}
	return validateFilterCombination(h.gameFilter.OperatorFilter)
}

func (h *gameFilterHandler) handle(ctx context.Context, f *filterBuilder) {
	if h.gameFilter == nil {
		return
	}

	if err := h.validate(); err != nil {
		f.setError(err)
		return
	}

	if sub := h.gameFilter.SubFilter(); sub != nil {
		handleSubFilter(ctx, &gameFilterHandler{gameFilter: sub}, f, h.gameFilter.OperatorFilter)
	}

	f.handleCriterion(ctx, h.criterionHandler())
}

func (h *gameFilterHandler) urlsCriterionHandler(url *models.StringCriterionInput) criterionHandlerFunc {
	builder := stringListCriterionHandlerBuilder{
		primaryTable: gameTable,
		primaryFK:    gameIDColumn,
		joinTable:    gamesURLsTable,
		stringColumn: gameURLColumn,
		addJoinTable: func(f *filterBuilder) {
			gamesURLsTableMgr.join(f, "", "games.id")
		},
	}

	return builder.handler(url)
}

func (h *gameFilterHandler) tagsCriterionHandler(tags *models.MultiCriterionInput) criterionHandlerFunc {
	builder := multiCriterionHandlerBuilder{
		primaryTable: gameTable,
		foreignTable: tagTable,
		joinTable:    gamesTagsTable,
		primaryFK:    gameIDColumn,
		foreignFK:    tagIDColumn,
		addJoinsFunc: func(f *filterBuilder) {
			gamesTagsTableMgr.join(f, "", "games.id")
		},
	}

	return builder.handler(tags)
}
