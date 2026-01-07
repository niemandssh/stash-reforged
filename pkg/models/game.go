package models

import (
	"context"
	"time"
)

type Game struct {
	ID int `json:"id"`

	Title     string `json:"title"`
	Details   string `json:"details"`
	Date      *Date  `json:"date"`
	Rating    *int   `json:"rating100"`
	Organized bool   `json:"organized"`

	OCounter    int `json:"o_counter"`
	OmegCounter int `json:"omg_counter"`

	Image []byte `json:"image"`

	FolderPath     string `json:"folder_path"`
	ExecutablePath string `json:"executable_path"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	URLs   RelatedStrings `json:"urls"`
	TagIDs RelatedIDs     `json:"tag_ids"`
}

func NewGame() Game {
	currentTime := time.Now()
	return Game{
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}
}

type GamePartial struct {
	Title          OptionalString
	Details        OptionalString
	Date           OptionalDate
	Rating         OptionalInt
	Organized      OptionalBool
	OCounter       OptionalInt
	OmegCounter    OptionalInt
	Image          OptionalBytes
	FolderPath     OptionalString
	ExecutablePath OptionalString
	URLs           *UpdateStrings
	TagIDs         *UpdateIDs
	CreatedAt      OptionalTime
	UpdatedAt      OptionalTime
}

func NewGamePartial() GamePartial {
	currentTime := time.Now()
	return GamePartial{
		UpdatedAt: NewOptionalTime(currentTime),
	}
}

func (g *Game) LoadURLs(ctx context.Context, l URLLoader) error {
	return g.URLs.load(func() ([]string, error) {
		return l.GetURLs(ctx, g.ID)
	})
}

func (g *Game) LoadTagIDs(ctx context.Context, l TagIDLoader) error {
	return g.TagIDs.load(func() ([]int, error) {
		return l.GetTagIDs(ctx, g.ID)
	})
}

type GameFilterType struct {
	OperatorFilter[GameFilterType]

	ID          *IntCriterionInput       `json:"id"`
	Title       *StringCriterionInput    `json:"title"`
	Details     *StringCriterionInput    `json:"details"`
	URL         *StringCriterionInput    `json:"url"`
	Date        *DateCriterionInput      `json:"date"`
	Rating100   *IntCriterionInput       `json:"rating100"`
	Organized   *bool                    `json:"organized"`
	OCounter    *IntCriterionInput       `json:"o_counter"`
	OmegCounter *IntCriterionInput       `json:"omg_counter"`
	TagIDs      *MultiCriterionInput     `json:"tag_ids"`
	CreatedAt   *TimestampCriterionInput `json:"created_at"`
	UpdatedAt   *TimestampCriterionInput `json:"updated_at"`
}

type GameCreateInput struct {
	ClientMutationID *string  `json:"clientMutationId"`
	Title            string   `json:"title"`
	Details          *string  `json:"details"`
	Date             *string  `json:"date"`
	Rating100        *int     `json:"rating100"`
	Organized        *bool    `json:"organized"`
	FolderPath       *string  `json:"folder_path"`
	ExecutablePath   *string  `json:"executable_path"`
	Urls             []string `json:"urls"`
	TagIds           []string `json:"tag_ids"`
	Image            *string  `json:"image"`
}

type GameUpdateInput struct {
	ClientMutationID *string  `json:"clientMutationId"`
	ID               string   `json:"id"`
	Title            *string  `json:"title"`
	Details          *string  `json:"details"`
	Date             *string  `json:"date"`
	Rating100        *int     `json:"rating100"`
	Organized        *bool    `json:"organized"`
	FolderPath       *string  `json:"folder_path"`
	ExecutablePath   *string  `json:"executable_path"`
	Urls             []string `json:"urls"`
	TagIds           []string `json:"tag_ids"`
	Image            *string  `json:"image"`
}

type GameDestroyInput struct {
	Ids []string `json:"ids"`
}
