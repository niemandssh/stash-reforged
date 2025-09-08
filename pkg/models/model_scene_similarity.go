package models

import (
	"time"
)

// SceneSimilarity represents the similarity relationship between two scenes
type SceneSimilarity struct {
	ID              int       `json:"id"`
	SceneID         int       `json:"scene_id"`
	SimilarSceneID  int       `json:"similar_scene_id"`
	SimilarityScore float64   `json:"similarity_score"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SceneSimilarityPartial represents part of a SceneSimilarity object for updates
type SceneSimilarityPartial struct {
	SceneID         OptionalInt
	SimilarSceneID  OptionalInt
	SimilarityScore OptionalFloat64
	CreatedAt       OptionalTime
	UpdatedAt       OptionalTime
}

func NewSceneSimilarity() SceneSimilarity {
	currentTime := time.Now()
	return SceneSimilarity{
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}
}

func NewSceneSimilarityPartial() SceneSimilarityPartial {
	currentTime := time.Now()
	return SceneSimilarityPartial{
		UpdatedAt: NewOptionalTime(currentTime),
	}
}
