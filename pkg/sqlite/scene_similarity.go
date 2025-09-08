package sqlite

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
	"github.com/stashapp/stash/pkg/models"
)

type SceneSimilarityStore struct {
	*sceneSimilarityQueryBuilder
}

func NewSceneSimilarityStore() *SceneSimilarityStore {
	return &SceneSimilarityStore{
		sceneSimilarityQueryBuilder: NewSceneSimilarityQueryBuilder(),
	}
}

type sceneSimilarityQueryBuilder struct {
	repository
}

func NewSceneSimilarityQueryBuilder() *sceneSimilarityQueryBuilder {
	return &sceneSimilarityQueryBuilder{
		repository: repository{
			tableName: sceneSimilaritiesTable,
			idColumn:  idColumn,
		},
	}
}

func (qb *sceneSimilarityQueryBuilder) Create(ctx context.Context, newObject models.SceneSimilarity) (*models.SceneSimilarity, error) {
	query := fmt.Sprintf("INSERT INTO %s (scene_id, similar_scene_id, similarity_score, created_at, updated_at) VALUES (?, ?, ?, ?, ?)", qb.tableName)
	result, err := dbWrapper.Exec(ctx, query, newObject.SceneID, newObject.SimilarSceneID, newObject.SimilarityScore, newObject.CreatedAt, newObject.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting scene similarity: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting last insert id: %w", err)
	}

	return qb.Find(ctx, int(id))
}

func (qb *sceneSimilarityQueryBuilder) Update(ctx context.Context, updatedObject models.SceneSimilarity) (*models.SceneSimilarity, error) {
	query := fmt.Sprintf("UPDATE %s SET scene_id = ?, similar_scene_id = ?, similarity_score = ?, updated_at = ? WHERE %s = ?", qb.tableName, qb.idColumn)
	_, err := dbWrapper.Exec(ctx, query, updatedObject.SceneID, updatedObject.SimilarSceneID, updatedObject.SimilarityScore, updatedObject.UpdatedAt, updatedObject.ID)
	if err != nil {
		return nil, fmt.Errorf("updating scene similarity: %w", err)
	}

	return qb.Find(ctx, updatedObject.ID)
}

func (qb *sceneSimilarityQueryBuilder) Destroy(ctx context.Context, id int) error {
	return qb.destroyExisting(ctx, []int{id})
}

func (qb *sceneSimilarityQueryBuilder) Find(ctx context.Context, id int) (*models.SceneSimilarity, error) {
	var ret models.SceneSimilarity
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", qb.tableName, qb.idColumn)
	if err := qb.queryStruct(ctx, query, []interface{}{id}, &ret); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (qb *sceneSimilarityQueryBuilder) FindByScenePair(ctx context.Context, sceneID int, similarSceneID int) (*models.SceneSimilarity, error) {
	query := dialect.Select(sceneSimilaritiesTableMgr.table.Col("*")).From(sceneSimilaritiesTableMgr.table).Where(
		goqu.And(
			sceneSimilaritiesTableMgr.table.Col("scene_id").Eq(sceneID),
			sceneSimilaritiesTableMgr.table.Col("similar_scene_id").Eq(similarSceneID),
		),
	)

	var ret models.SceneSimilarity
	if err := querySimple(ctx, query, &ret); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (qb *sceneSimilarityQueryBuilder) FindSimilarScenes(ctx context.Context, sceneID int, limit int) ([]*models.SceneSimilarity, error) {
	// Search in both directions: where scene_id = ? OR similar_scene_id = ?
	// But we need to normalize the results so that sceneID is always the first column
	query := dialect.Select(
		sceneSimilaritiesTableMgr.table.Col("id"),
		goqu.Case().
			When(sceneSimilaritiesTableMgr.table.Col("scene_id").Eq(sceneID), sceneSimilaritiesTableMgr.table.Col("scene_id")).
			Else(sceneSimilaritiesTableMgr.table.Col("similar_scene_id")).As("scene_id"),
		goqu.Case().
			When(sceneSimilaritiesTableMgr.table.Col("scene_id").Eq(sceneID), sceneSimilaritiesTableMgr.table.Col("similar_scene_id")).
			Else(sceneSimilaritiesTableMgr.table.Col("scene_id")).As("similar_scene_id"),
		sceneSimilaritiesTableMgr.table.Col("similarity_score"),
		sceneSimilaritiesTableMgr.table.Col("created_at"),
		sceneSimilaritiesTableMgr.table.Col("updated_at"),
	).From(sceneSimilaritiesTableMgr.table).Where(
		goqu.Or(
			sceneSimilaritiesTableMgr.table.Col("scene_id").Eq(sceneID),
			sceneSimilaritiesTableMgr.table.Col("similar_scene_id").Eq(sceneID),
		),
	).Order(sceneSimilaritiesTableMgr.table.Col("similarity_score").Desc())

	if limit > 0 {
		query = query.Limit(uint(limit))
	}

	var ret []*models.SceneSimilarity
	if err := queryFunc(ctx, query, false, func(rows *sqlx.Rows) error {
		var similarity models.SceneSimilarity
		if err := rows.Scan(
			&similarity.ID,
			&similarity.SceneID,
			&similarity.SimilarSceneID,
			&similarity.SimilarityScore,
			&similarity.CreatedAt,
			&similarity.UpdatedAt,
		); err != nil {
			return err
		}
		ret = append(ret, &similarity)
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (qb *sceneSimilarityQueryBuilder) DeleteByScene(ctx context.Context, sceneID int) error {
	query := dialect.Delete(sceneSimilaritiesTableMgr.table).Where(
		goqu.Or(
			sceneSimilaritiesTableMgr.table.Col("scene_id").Eq(sceneID),
			sceneSimilaritiesTableMgr.table.Col("similar_scene_id").Eq(sceneID),
		),
	)

	if _, err := exec(ctx, query); err != nil {
		return fmt.Errorf("deleting scene similarities for scene %d: %w", sceneID, err)
	}

	return nil
}

func (qb *sceneSimilarityQueryBuilder) Upsert(ctx context.Context, similarity models.SceneSimilarity) error {
	query := dialect.Insert(sceneSimilaritiesTableMgr.table).
		Cols("scene_id", "similar_scene_id", "similarity_score", "created_at", "updated_at").
		Vals(goqu.Vals{
			similarity.SceneID,
			similarity.SimilarSceneID,
			similarity.SimilarityScore,
			similarity.CreatedAt,
			similarity.UpdatedAt,
		}).
		OnConflict(goqu.DoUpdate("scene_id, similar_scene_id", goqu.Record{
			"similarity_score": similarity.SimilarityScore,
			"updated_at":       similarity.UpdatedAt,
		}))

	if _, err := exec(ctx, query); err != nil {
		return fmt.Errorf("upserting scene similarity: %w", err)
	}

	return nil
}

func (qb *sceneSimilarityQueryBuilder) Count(ctx context.Context) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", qb.tableName)
	var count int
	if err := dbWrapper.Get(ctx, &count, query); err != nil {
		return 0, fmt.Errorf("counting scene similarities: %w", err)
	}
	return count, nil
}
