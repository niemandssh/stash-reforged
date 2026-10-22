package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jmoiron/sqlx"

	"github.com/stashapp/stash/pkg/models"
)

type watchLaterTable struct {
	table
	dateColumn exp.IdentifierExpression
}

func (t *watchLaterTable) isInWatchLater(ctx context.Context, id int) (bool, error) {
	table := t.table.table
	q := dialect.Select(goqu.COUNT("*")).From(table).Where(t.idColumn.Eq(id))

	var count int
	if err := querySimple(ctx, q, &count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func (t *watchLaterTable) getManyInWatchLater(ctx context.Context, ids []int) ([]bool, error) {
	if len(ids) == 0 {
		return []bool{}, nil
	}

	table := t.table.table
	q := dialect.Select(t.idColumn).From(table).Where(t.idColumn.In(ids))

	ret := make([]bool, len(ids))
	idToIndex := idToIndexMap(ids)

	const single = false
	if err := queryFunc(ctx, q, single, func(rows *sqlx.Rows) error {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ret[idToIndex[id]] = true
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (t *watchLaterTable) getAddedAt(ctx context.Context, id int) (*time.Time, error) {
	table := t.table.table
	q := dialect.Select(t.dateColumn).From(table).Where(t.idColumn.Eq(id))

	var date NullTimestamp
	if err := querySimple(ctx, q, &date); err != nil {
		return nil, err
	}

	return date.TimePtr(), nil
}

func (t *watchLaterTable) getManyAddedAt(ctx context.Context, ids []int) ([]*time.Time, error) {
	if len(ids) == 0 {
		return []*time.Time{}, nil
	}

	table := t.table.table
	q := dialect.Select(t.idColumn, t.dateColumn).From(table).Where(t.idColumn.In(ids))

	ret := make([]*time.Time, len(ids))
	idToIndex := idToIndexMap(ids)

	const single = false
	if err := queryFunc(ctx, q, single, func(rows *sqlx.Rows) error {
		var id int
		var date Timestamp
		if err := rows.Scan(&id, &date); err != nil {
			return err
		}
		t := date.Timestamp
		ret[idToIndex[id]] = &t
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (t *watchLaterTable) add(ctx context.Context, id int, at time.Time) (*time.Time, error) {
	table := t.table.table

	q := dialect.Insert(table).Cols(t.idColumn.GetCol(), t.dateColumn.GetCol()).Vals(
		goqu.Vals{id, UTCTimestamp{Timestamp{at}}},
	).OnConflict(goqu.DoUpdate("scene_id", goqu.Record{
		"added_at": UTCTimestamp{Timestamp{at}},
	}))

	if _, err := exec(ctx, q); err != nil {
		return nil, fmt.Errorf("inserting into %s: %w", table.GetTable(), err)
	}

	return t.getAddedAt(ctx, id)
}

func (t *watchLaterTable) remove(ctx context.Context, id int) error {
	table := t.table.table
	q := dialect.Delete(table).Where(t.idColumn.Eq(id))

	if _, err := exec(ctx, q); err != nil {
		return fmt.Errorf("deleting from %s: %w", table.GetTable(), err)
	}

	return nil
}

func (t *watchLaterTable) findEntries(ctx context.Context, page, perPage int) ([]models.WatchLaterEntry, error) {
	table := t.table.table

	offset := (page - 1) * perPage
	q := dialect.Select(t.idColumn, t.dateColumn).From(table).Order(t.dateColumn.Desc()).Limit(uint(perPage)).Offset(uint(offset))

	var ret []models.WatchLaterEntry
	const single = false
	if err := queryFunc(ctx, q, single, func(rows *sqlx.Rows) error {
		var sceneID int
		var date Timestamp
		if err := rows.Scan(&sceneID, &date); err != nil {
			return err
		}
		ret = append(ret, models.WatchLaterEntry{
			SceneID: sceneID,
			AddedAt: date.Timestamp,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (t *watchLaterTable) count(ctx context.Context) (int, error) {
	table := t.table.table
	q := dialect.Select(goqu.COUNT("*")).From(table)

	var ret int
	if err := querySimple(ctx, q, &ret); err != nil {
		return 0, err
	}

	return ret, nil
}

type watchLaterManager struct {
	tableMgr *watchLaterTable
}

func (m *watchLaterManager) IsInWatchLater(ctx context.Context, id int) (bool, error) {
	return m.tableMgr.isInWatchLater(ctx, id)
}

func (m *watchLaterManager) GetManyInWatchLater(ctx context.Context, ids []int) ([]bool, error) {
	return m.tableMgr.getManyInWatchLater(ctx, ids)
}

func (m *watchLaterManager) GetWatchLaterAt(ctx context.Context, id int) (*time.Time, error) {
	return m.tableMgr.getAddedAt(ctx, id)
}

func (m *watchLaterManager) GetManyWatchLaterAt(ctx context.Context, ids []int) ([]*time.Time, error) {
	return m.tableMgr.getManyAddedAt(ctx, ids)
}

func (m *watchLaterManager) AddWatchLater(ctx context.Context, id int) (*time.Time, error) {
	return m.tableMgr.add(ctx, id, time.Now())
}

func (m *watchLaterManager) RemoveWatchLater(ctx context.Context, id int) error {
	return m.tableMgr.remove(ctx, id)
}

func (m *watchLaterManager) FindWatchLater(ctx context.Context, page, perPage int) ([]models.WatchLaterEntry, error) {
	return m.tableMgr.findEntries(ctx, page, perPage)
}

func (m *watchLaterManager) CountWatchLater(ctx context.Context) (int, error) {
	return m.tableMgr.count(ctx)
}
