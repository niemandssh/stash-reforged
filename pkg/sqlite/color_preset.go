package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4"

	"github.com/stashapp/stash/pkg/models"
)

type colorPresetRow struct {
	ID                         int            `db:"id" goqu:"skipinsert"`
	Name                       null.String    `db:"name"`
	Color                      null.String    `db:"color"`
	Sort                       int            `db:"sort"`
	TagRequirementsDescription sql.NullString `db:"tag_requirements_description"`
	RequiredForRequirements    bool           `db:"required_for_requirements"`
	CreatedAt                  Timestamp      `db:"created_at"`
	UpdatedAt                  Timestamp      `db:"updated_at"`
}

func (r *colorPresetRow) fromColorPreset(o models.ColorPreset) {
	r.ID = o.ID
	r.Name = null.StringFrom(o.Name)
	r.Color = null.StringFrom(o.Color)
	r.Sort = o.Sort
	if o.TagRequirementsDescription != "" {
		r.TagRequirementsDescription = sql.NullString{String: o.TagRequirementsDescription, Valid: true}
	} else {
		r.TagRequirementsDescription = sql.NullString{Valid: false}
	}
	r.RequiredForRequirements = o.RequiredForRequirements
	r.CreatedAt = Timestamp{Timestamp: o.CreatedAt}
	r.UpdatedAt = Timestamp{Timestamp: o.UpdatedAt}
}

func (r *colorPresetRow) resolve() *models.ColorPreset {
	tagReqDesc := ""
	if r.TagRequirementsDescription.Valid {
		tagReqDesc = r.TagRequirementsDescription.String
	}

	ret := &models.ColorPreset{
		ID:                         r.ID,
		Name:                       r.Name.String,
		Color:                      r.Color.String,
		Sort:                       r.Sort,
		TagRequirementsDescription: tagReqDesc,
		RequiredForRequirements:    r.RequiredForRequirements,
		CreatedAt:                  r.CreatedAt.Timestamp,
		UpdatedAt:                  r.UpdatedAt.Timestamp,
	}

	return ret
}

type colorPresetRowRecord struct {
	updateRecord
	colorPresetRow
}

func (r *colorPresetRowRecord) fromPartial(o models.ColorPresetPartial) {
	// Only update fields that are set
	r.setNullString("name", o.Name)
	r.setNullString("color", o.Color)
	r.setNullInt("sort", o.Sort)
	r.setNullString("tag_requirements_description", o.TagRequirementsDescription)
	r.setBool("required_for_requirements", o.RequiredForRequirements)
}

type colorPresetRepository struct {
	repository
	tableMgr *table
}

func NewColorPresetRepository(db *sqlx.DB) *colorPresetRepository {
	return &colorPresetRepository{
		repository: repository{
			tableName: colorPresetTable,
			idColumn:  idColumn,
		},
		tableMgr: colorPresetTableMgr,
	}
}

func (qb *colorPresetRepository) table() exp.IdentifierExpression {
	return qb.tableMgr.table
}

func (qb *colorPresetRepository) selectDataset() *goqu.SelectDataset {
	return dialect.From(qb.table()).Select(qb.table().All())
}

func (qb *colorPresetRepository) Create(ctx context.Context, newColorPreset models.ColorPreset) (*models.ColorPreset, error) {
	var r colorPresetRow
	r.fromColorPreset(newColorPreset)

	id, err := qb.tableMgr.insertID(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("inserting color preset: %w", err)
	}

	updated, err := qb.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finding after create: %w", err)
	}

	return updated, nil
}

func (qb *colorPresetRepository) Update(ctx context.Context, id int, updatedColorPreset models.ColorPresetPartial) (*models.ColorPreset, error) {
	// Build update record using colorPresetRowRecord
	r := colorPresetRowRecord{
		updateRecord: updateRecord{
			Record: make(exp.Record),
		},
	}
	r.fromPartial(updatedColorPreset)

	// If no fields to update, return current record
	if len(r.Record) == 0 {
		return qb.Find(ctx, id)
	}

	// Build and execute update query
	if err := qb.tableMgr.updateByID(ctx, id, r.Record); err != nil {
		return nil, err
	}

	return qb.Find(ctx, id)
}

func (qb *colorPresetRepository) Destroy(ctx context.Context, id int) error {
	return qb.destroyExisting(ctx, []int{id})
}

func (qb *colorPresetRepository) Find(ctx context.Context, id int) (*models.ColorPreset, error) {
	q := qb.selectDataset().Where(qb.tableMgr.byID(id))

	ret, err := qb.get(ctx, q)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting color preset by id %d: %w", id, err)
	}

	return ret, nil
}

func (qb *colorPresetRepository) FindAll(ctx context.Context) ([]*models.ColorPreset, error) {
	table := qb.table()
	q := qb.selectDataset().Order(table.Col("sort").Asc(), table.Col("name").Asc())

	return qb.getMany(ctx, q)
}

func (qb *colorPresetRepository) get(ctx context.Context, q *goqu.SelectDataset) (*models.ColorPreset, error) {
	ret, err := qb.getMany(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, sql.ErrNoRows
	}

	return ret[0], nil
}

func (qb *colorPresetRepository) getMany(ctx context.Context, q *goqu.SelectDataset) ([]*models.ColorPreset, error) {
	const single = false
	var ret []*models.ColorPreset
	if err := queryFunc(ctx, q, single, func(r *sqlx.Rows) error {
		var f colorPresetRow
		if err := r.StructScan(&f); err != nil {
			return err
		}

		s := f.resolve()

		ret = append(ret, s)
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}
