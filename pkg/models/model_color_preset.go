package models

import (
	"context"
	"time"
)

type ColorPreset struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewColorPreset() ColorPreset {
	currentTime := time.Now()
	return ColorPreset{
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
		Sort:      0,
	}
}

type ColorPresetPartial struct {
	Name  OptionalString
	Color OptionalString
	Sort  OptionalInt
}

func NewColorPresetPartial() ColorPresetPartial {
	return ColorPresetPartial{}
}

// ColorPresetReaderWriter интерфейс для работы с пресетами цветов
type ColorPresetReaderWriter interface {
	Create(ctx context.Context, newColorPreset ColorPreset) (*ColorPreset, error)
	Update(ctx context.Context, id int, updatedColorPreset ColorPresetPartial) (*ColorPreset, error)
	Destroy(ctx context.Context, id int) error
	Find(ctx context.Context, id int) (*ColorPreset, error)
	FindAll(ctx context.Context) ([]*ColorPreset, error)
}

// ColorPresetReader интерфейс только для чтения пресетов цветов
type ColorPresetReader interface {
	Find(ctx context.Context, id int) (*ColorPreset, error)
	FindAll(ctx context.Context) ([]*ColorPreset, error)
}
