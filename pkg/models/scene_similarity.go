package models

import "context"

type SceneSimilarityReader interface {
	Find(ctx context.Context, id int) (*SceneSimilarity, error)
	FindByScenePair(ctx context.Context, sceneID int, similarSceneID int) (*SceneSimilarity, error)
	FindSimilarScenes(ctx context.Context, sceneID int, limit int) ([]*SceneSimilarity, error)
	Count(ctx context.Context) (int, error)
}

type SceneSimilarityWriter interface {
	Create(ctx context.Context, newObject SceneSimilarity) (*SceneSimilarity, error)
	Update(ctx context.Context, updatedObject SceneSimilarity) (*SceneSimilarity, error)
	Destroy(ctx context.Context, id int) error
	DeleteByScene(ctx context.Context, sceneID int) error
	DeleteBySceneAsSource(ctx context.Context, sceneID int) error
	Upsert(ctx context.Context, similarity SceneSimilarity) error
}

type SceneSimilarityReaderWriter interface {
	SceneSimilarityReader
	SceneSimilarityWriter
}
