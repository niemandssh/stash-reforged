package models

import (
	"time"
)

type PerformerProfileImage struct {
	ID          int       `json:"id"`
	PerformerID int       `json:"performer_id"`
	IsPrimary   bool      `json:"is_primary"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PerformerProfileImagePartial struct {
	ID          OptionalInt
	PerformerID OptionalInt
	IsPrimary   OptionalBool
	Position    OptionalInt
	CreatedAt   OptionalTime
	UpdatedAt   OptionalTime
}

func NewPerformerProfileImage() PerformerProfileImage {
	currentTime := time.Now()
	return PerformerProfileImage{
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}
}

func NewPerformerProfileImagePartial() PerformerProfileImagePartial {
	currentTime := time.Now()
	return PerformerProfileImagePartial{
		UpdatedAt: NewOptionalTime(currentTime),
	}
}

type CreatePerformerProfileImageInput struct {
	PerformerID int    `json:"performer_id"`
	Image       string `json:"image"` // base64 encoded image data
	IsPrimary   *bool  `json:"is_primary"`
	Position    *int   `json:"position"`
}

type UpdatePerformerProfileImageInput struct {
	ID        int   `json:"id"`
	IsPrimary *bool `json:"is_primary"`
	Position  *int  `json:"position"`
}

type DeletePerformerProfileImageInput struct {
	ID int `json:"id"`
}
