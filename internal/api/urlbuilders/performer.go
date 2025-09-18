package urlbuilders

import (
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

type PerformerURLBuilder struct {
	BaseURL     string
	PerformerID string
	UpdatedAt   string
}

func NewPerformerURLBuilder(baseURL string, performer *models.Performer) PerformerURLBuilder {
	return PerformerURLBuilder{
		BaseURL:     baseURL,
		PerformerID: strconv.Itoa(performer.ID),
		UpdatedAt:   strconv.FormatInt(performer.UpdatedAt.Unix(), 10),
	}
}

func (b PerformerURLBuilder) GetPerformerImageURL(hasImage bool) string {
	url := b.BaseURL + "/performer/" + b.PerformerID + "/image?t=" + b.UpdatedAt
	if !hasImage {
		url += "&default=true"
	}
	return url
}

func (b PerformerURLBuilder) GetPerformerProfileImageURL(imageID int, updatedAt int64) string {
	return b.BaseURL + "/performer/" + b.PerformerID + "/profile_image/" + strconv.Itoa(imageID) + "?t=" + strconv.FormatInt(updatedAt, 10)
}
