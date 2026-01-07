package models

import (
	"context"
	"time"
)

// GameGetter provides methods to get games by ID.
type GameGetter interface {
	Find(ctx context.Context, id int) (*Game, error)
	FindMany(ctx context.Context, ids []int) ([]*Game, error)
}

// GameFinder provides methods to get games by ID.
type GameFinder interface {
	GameGetter
	GetImage(ctx context.Context, gameID int) ([]byte, error)
	HasImage(ctx context.Context, gameID int) (bool, error)
}

// GameQueryer provides methods to query games.
type GameQueryer interface {
	Query(ctx context.Context, gameFilter *GameFilterType, findFilter *FindFilterType) ([]*Game, int, error)
	QueryCount(ctx context.Context, gameFilter *GameFilterType, findFilter *FindFilterType) (int, error)
}

// GameCreator provides methods to create games.
type GameCreator interface {
	Create(ctx context.Context, game *Game) error
}

// GameUpdater provides methods to update games.
type GameUpdater interface {
	Update(ctx context.Context, game *Game) error
	UpdatePartial(ctx context.Context, id int, partial GamePartial) (*Game, error)
}

// GameDestroyer provides methods to delete games.
type GameDestroyer interface {
	Destroy(ctx context.Context, id int) error
}

// GameReader exposes read operations for games.
type GameReader interface {
	GameFinder
	GameQueryer
	URLLoader
	TagIDLoader

	GetManyOCount(ctx context.Context, ids []int) ([]int, error)
	GetManyODates(ctx context.Context, ids []int) ([][]time.Time, error)
	GetODates(ctx context.Context, id int) ([]time.Time, error)

	GetManyOMGCount(ctx context.Context, ids []int) ([]int, error)
	GetManyOMGDates(ctx context.Context, ids []int) ([][]time.Time, error)
	GetOMGDates(ctx context.Context, id int) ([]time.Time, error)

	GetManyViewCount(ctx context.Context, ids []int) ([]int, error)
	GetManyViewDates(ctx context.Context, ids []int) ([][]time.Time, error)
	GetViewDates(ctx context.Context, id int) ([]time.Time, error)
	CountViews(ctx context.Context, id int) (int, error)

	All(ctx context.Context) ([]*Game, error)
}

// GameWriter exposes write operations for games.
type GameWriter interface {
	GameCreator
	GameUpdater
	GameDestroyer
	OHistoryWriter
	OMGHistoryWriter
	ViewHistoryWriter

	IncrementOCounter(ctx context.Context, id int) (int, error)
	DecrementOCounter(ctx context.Context, id int) (int, error)
	ResetOCounter(ctx context.Context, id int) (int, error)

	IncrementOMGCounter(ctx context.Context, id int) (int, error)
	DecrementOMGCounter(ctx context.Context, id int) (int, error)
	ResetOMGCounter(ctx context.Context, id int) (int, error)
}

// GameReaderWriter aggregates all game methods.
type GameReaderWriter interface {
	GameReader
	GameWriter
}
