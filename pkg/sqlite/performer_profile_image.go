package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jmoiron/sqlx"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
	"gopkg.in/guregu/null.v4/zero"
)

type performerProfileImageRow struct {
	ID          int         `db:"id" goqu:"skipinsert"`
	PerformerID int         `db:"performer_id"`
	ImageBlob   zero.String `db:"image_blob"`
	IsPrimary   bool        `db:"is_primary"`
	Position    int         `db:"position"`
	CreatedAt   Timestamp   `db:"created_at"`
	UpdatedAt   Timestamp   `db:"updated_at"`
}

func (r *performerProfileImageRow) fromPerformerProfileImage(o models.PerformerProfileImage) {
	r.ID = o.ID
	r.PerformerID = o.PerformerID
	r.IsPrimary = o.IsPrimary
	r.Position = o.Position
	r.CreatedAt = Timestamp{Timestamp: o.CreatedAt}
	r.UpdatedAt = Timestamp{Timestamp: o.UpdatedAt}
}

func (r *performerProfileImageRow) resolve() *models.PerformerProfileImage {
	ret := &models.PerformerProfileImage{
		ID:          r.ID,
		PerformerID: r.PerformerID,
		IsPrimary:   r.IsPrimary,
		Position:    r.Position,
		CreatedAt:   r.CreatedAt.Timestamp,
		UpdatedAt:   r.UpdatedAt.Timestamp,
	}

	return ret
}

type performerProfileImageRowRecord struct {
	updateRecord
}

func (r *performerProfileImageRowRecord) fromPartial(o models.PerformerProfileImagePartial) {
	r.setInt("performer_id", o.PerformerID)
	r.setBool("is_primary", o.IsPrimary)
	r.setInt("position", o.Position)
	r.setTimestamp("created_at", o.CreatedAt)
	r.setTimestamp("updated_at", o.UpdatedAt)
}

var (
	performerProfileImageTableMgr = &table{
		table:    goqu.T(performerProfileImagesTable),
		idColumn: goqu.T(performerProfileImagesTable).Col(performerProfileImageIDColumn),
	}
)

type PerformerProfileImageStore struct {
	blobJoinQueryBuilder

	tableMgr *table
}

func NewPerformerProfileImageStore(blobStore *BlobStore) *PerformerProfileImageStore {
	return &PerformerProfileImageStore{
		blobJoinQueryBuilder: blobJoinQueryBuilder{
			blobStore: blobStore,
			joinTable: performerProfileImagesTable,
		},
		tableMgr: performerProfileImageTableMgr,
	}
}

func (qb *PerformerProfileImageStore) table() exp.IdentifierExpression {
	return qb.tableMgr.table
}

func (qb *PerformerProfileImageStore) selectDataset() *goqu.SelectDataset {
	return dialect.From(qb.table()).Select(qb.table().All())
}

func (qb *PerformerProfileImageStore) Create(ctx context.Context, newObject *models.CreatePerformerProfileImageInput) (*models.PerformerProfileImage, error) {
	var r performerProfileImageRow
	obj := models.NewPerformerProfileImage()
	obj.PerformerID = newObject.PerformerID
	if newObject.IsPrimary != nil {
		obj.IsPrimary = *newObject.IsPrimary
	}
	if newObject.Position != nil {
		obj.Position = *newObject.Position
	}

	r.fromPerformerProfileImage(obj)

	id, err := qb.tableMgr.insertID(ctx, r)
	if err != nil {
		return nil, err
	}

	// Update image blob if provided
	if newObject.Image != "" {
		imageData, err := utils.ProcessImageInput(ctx, newObject.Image)
		if err != nil {
			return nil, fmt.Errorf("processing image: %w", err)
		}

		if err := qb.UpdateImage(ctx, id, imageData); err != nil {
			return nil, fmt.Errorf("setting image: %w", err)
		}
	}

	// If this is set as primary, ensure no other images for this performer are primary
	if obj.IsPrimary {
		if err := qb.clearOtherPrimaryImages(ctx, obj.PerformerID, id); err != nil {
			return nil, fmt.Errorf("clearing other primary images: %w", err)
		}
	}

	return qb.find(ctx, id)
}

func (qb *PerformerProfileImageStore) UpdatePartial(ctx context.Context, id int, partial models.PerformerProfileImagePartial) (*models.PerformerProfileImage, error) {
	r := performerProfileImageRowRecord{
		updateRecord{
			Record: make(exp.Record),
		},
	}

	r.fromPartial(partial)

	if len(r.Record) > 0 {
		if err := qb.tableMgr.updateByID(ctx, id, r.Record); err != nil {
			return nil, err
		}
	}

	// If this is set as primary, ensure no other images for this performer are primary
	if partial.IsPrimary.Set && partial.IsPrimary.Value {
		existing, err := qb.find(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := qb.clearOtherPrimaryImages(ctx, existing.PerformerID, id); err != nil {
			return nil, fmt.Errorf("clearing other primary images: %w", err)
		}
	}

	return qb.find(ctx, id)
}

func (qb *PerformerProfileImageStore) Destroy(ctx context.Context, id int) error {
	// Remove image blob first
	if err := qb.destroyImage(ctx, id); err != nil {
		return err
	}

	return qb.tableMgr.destroy(ctx, []int{id})
}

func (qb *PerformerProfileImageStore) Count(ctx context.Context) (int, error) {
	q := dialect.Select(goqu.COUNT("*")).From(qb.table())
	return count(ctx, q)
}

// returns nil, nil if not found
func (qb *PerformerProfileImageStore) Find(ctx context.Context, id int) (*models.PerformerProfileImage, error) {
	ret, err := qb.find(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return ret, err
}

// returns nil, sql.ErrNoRows if not found
func (qb *PerformerProfileImageStore) find(ctx context.Context, id int) (*models.PerformerProfileImage, error) {
	q := qb.selectDataset().Where(qb.tableMgr.byID(id))

	ret, err := qb.get(ctx, q)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (qb *PerformerProfileImageStore) get(ctx context.Context, q *goqu.SelectDataset) (*models.PerformerProfileImage, error) {
	ret, err := qb.getMany(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, sql.ErrNoRows
	}

	return ret[0], nil
}

func (qb *PerformerProfileImageStore) getMany(ctx context.Context, q *goqu.SelectDataset) ([]*models.PerformerProfileImage, error) {
	const single = false
	var ret []*models.PerformerProfileImage
	if err := queryFunc(ctx, q, single, func(r *sqlx.Rows) error {
		var f performerProfileImageRow
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

func (qb *PerformerProfileImageStore) FindByPerformerID(ctx context.Context, performerID int) ([]*models.PerformerProfileImage, error) {
	q := qb.selectDataset().Where(
		qb.table().Col("performer_id").Eq(performerID),
	).Order(
		qb.table().Col("position").Asc(),
		qb.table().Col("id").Asc(),
	)

	return qb.getMany(ctx, q)
}

func (qb *PerformerProfileImageStore) GetImage(ctx context.Context, id int) ([]byte, error) {
	return qb.blobJoinQueryBuilder.GetImage(ctx, id, performerProfileImageBlobColumn)
}

func (qb *PerformerProfileImageStore) HasImage(ctx context.Context, id int) (bool, error) {
	return qb.blobJoinQueryBuilder.HasImage(ctx, id, performerProfileImageBlobColumn)
}

func (qb *PerformerProfileImageStore) UpdateImage(ctx context.Context, id int, image []byte) error {
	return qb.blobJoinQueryBuilder.UpdateImage(ctx, id, performerProfileImageBlobColumn, image)
}

func (qb *PerformerProfileImageStore) destroyImage(ctx context.Context, id int) error {
	return qb.blobJoinQueryBuilder.DestroyImage(ctx, id, performerProfileImageBlobColumn)
}

// clearOtherPrimaryImages ensures only the specified image is marked as primary for the given performer
func (qb *PerformerProfileImageStore) clearOtherPrimaryImages(ctx context.Context, performerID int, exceptImageID int) error {
	q := dialect.Update(qb.table()).Set(
		goqu.Record{"is_primary": false},
	).Where(
		qb.table().Col("performer_id").Eq(performerID),
		qb.table().Col("id").Neq(exceptImageID),
		qb.table().Col("is_primary").Eq(true),
	)

	_, err := exec(ctx, q)
	return err
}
