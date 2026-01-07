package urlbuilders

import (
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

type GameURLBuilder struct {
	BaseURL   string
	GameID    string
	UpdatedAt string
}

func NewGameURLBuilder(baseURL string, game *models.Game) GameURLBuilder {
	return GameURLBuilder{
		BaseURL:   baseURL,
		GameID:    strconv.Itoa(game.ID),
		UpdatedAt: strconv.FormatInt(game.UpdatedAt.Unix(), 10),
	}
}

func (b GameURLBuilder) GetGameImageURL(hasImage bool) string {
	url := b.BaseURL + "/game/" + b.GameID + "/image?t=" + b.UpdatedAt
	if !hasImage {
		url += "&default=true"
	}
	return url
}
