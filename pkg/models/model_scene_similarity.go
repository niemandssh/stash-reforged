package models

import (
	"encoding/json"
	"time"
)

// SimilarityScoreData represents the breakdown of similarity score by tag categories
type SimilarityScoreData struct {
	EnhancedTags float64 `json:"enhanced_tags"` // > 0.61
	NormalTags   float64 `json:"normal_tags"`   // 0.41 - 0.6
	ReducedTags  float64 `json:"reduced_tags"`  // < 0.4
	Tags         float64 `json:"tags"`          // Overall tag similarity contribution
	Performers   float64 `json:"performers"`
	Groups       float64 `json:"groups"`
	Studio       float64 `json:"studio"`
	Penalty      float64 `json:"penalty,omitempty"` // Broken status penalty
}

// SceneSimilarity represents the similarity relationship between two scenes
type SceneSimilarity struct {
	ID                  int                  `json:"id"`
	SceneID             int                  `json:"scene_id"`
	SimilarSceneID      int                  `json:"similar_scene_id"`
	SimilarityScore     float64              `json:"similarity_score"`
	SimilarityScoreData *SimilarityScoreData `json:"similarity_score_data,omitempty"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
}

// SceneSimilarityPartial represents part of a SceneSimilarity object for updates
type SceneSimilarityPartial struct {
	SceneID             OptionalInt
	SimilarSceneID      OptionalInt
	SimilarityScore     OptionalFloat64
	SimilarityScoreData OptionalString // JSON string
	CreatedAt           OptionalTime
	UpdatedAt           OptionalTime
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

// MarshalSimilarityScoreData converts SimilarityScoreData to JSON string
func (s *SimilarityScoreData) MarshalSimilarityScoreData() (string, error) {
	if s == nil {
		return "", nil
	}
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalSimilarityScoreData converts JSON string to SimilarityScoreData
func UnmarshalSimilarityScoreData(data string) (*SimilarityScoreData, error) {
	if data == "" {
		return nil, nil
	}
	var scoreData SimilarityScoreData
	err := json.Unmarshal([]byte(data), &scoreData)
	if err != nil {
		return nil, err
	}
	return &scoreData, nil
}
