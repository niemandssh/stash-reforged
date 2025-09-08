package similarity

import (
	"context"
	"fmt"
	"math"

	"github.com/stashapp/stash/pkg/models"
)

// SimilarityWeights defines the weights for different similarity factors
type SimilarityWeights struct {
	Performers float64 // Weight for performer similarity
	Groups     float64 // Weight for group similarity
	Tags       float64 // Weight for tag similarity
	Studio     float64 // Weight for studio similarity
	MinScore   float64 // Minimum similarity score to store (default 0.3)
}

// DefaultSimilarityWeights returns the default weights for similarity calculation
func DefaultSimilarityWeights() SimilarityWeights {
	return SimilarityWeights{
		Performers: 0.3, // 30% weight
		Groups:     0.2, // 20% weight
		Tags:       0.4, // 40% weight (highest)
		Studio:     0.1, // 10% weight (lowest)
		/////////////////////////////////////////////////////////////
		MinScore: 0.1, // Only store similarities with score >= 0.1
	}
}

// SceneSimilarityCalculator calculates similarity between scenes
type SceneSimilarityCalculator struct {
	repository models.SceneSimilarityReaderWriter
	sceneRepo  models.SceneReader
	weights    SimilarityWeights
}

// NewSceneSimilarityCalculator creates a new similarity calculator
func NewSceneSimilarityCalculator(repository models.SceneSimilarityReaderWriter, sceneRepo models.SceneReader, weights SimilarityWeights) *SceneSimilarityCalculator {
	return &SceneSimilarityCalculator{
		repository: repository,
		sceneRepo:  sceneRepo,
		weights:    weights,
	}
}

// CalculateSimilarity calculates the similarity score between two scenes
func (c *SceneSimilarityCalculator) CalculateSimilarity(ctx context.Context, scene1, scene2 *models.Scene) (float64, error) {
	if scene1.ID == scene2.ID {
		return 1.0, nil // Same scene
	}

	// Debug for scene ID 1 - useful for testing
	debugScene1 := scene1.ID == 1 || scene2.ID == 1

	if debugScene1 {
		fmt.Printf("=== DEBUG FOR SCENE 1 ===\n")
		fmt.Printf("Comparing scenes %d and %d\n", scene1.ID, scene2.ID)
	}

	var totalScore float64

	// Calculate performer similarity
	performerScore, err := c.calculatePerformerSimilarity(scene1.PerformerIDs.List(), scene2.PerformerIDs.List())
	if err != nil {
		return 0, fmt.Errorf("calculating performer similarity: %w", err)
	}
	performerContribution := performerScore * c.weights.Performers
	totalScore += performerContribution
	if debugScene1 {
		fmt.Printf("DEBUG: Performer similarity: %.3f * %.3f = %.3f\n", performerScore, c.weights.Performers, performerContribution)
	}

	// Calculate group similarity
	groupScore, err := c.calculateGroupSimilarity(scene1.Groups.List(), scene2.Groups.List())
	if err != nil {
		return 0, fmt.Errorf("calculating group similarity: %w", err)
	}
	groupContribution := groupScore * c.weights.Groups
	totalScore += groupContribution
	if debugScene1 {
		fmt.Printf("DEBUG: Group similarity: %.3f * %.3f = %.3f\n", groupScore, c.weights.Groups, groupContribution)
	}

	// Calculate tag similarity
	tags1 := scene1.TagIDs.List()
	tags2 := scene2.TagIDs.List()
	if debugScene1 {
		fmt.Printf("DEBUG: Scene %d has %d tags: %v\n", scene1.ID, len(tags1), tags1)
		fmt.Printf("DEBUG: Scene %d has %d tags: %v\n", scene2.ID, len(tags2), tags2)
	}
	tagScore, err := c.calculateTagSimilarity(tags1, tags2)
	if err != nil {
		return 0, fmt.Errorf("calculating tag similarity: %w", err)
	}
	tagContribution := tagScore * c.weights.Tags
	totalScore += tagContribution
	if debugScene1 {
		fmt.Printf("DEBUG: Tag similarity: %.3f * %.3f = %.3f\n", tagScore, c.weights.Tags, tagContribution)
	}

	// Calculate studio similarity
	studioScore := c.calculateStudioSimilarity(scene1.StudioID, scene2.StudioID)
	studioContribution := studioScore * c.weights.Studio
	totalScore += studioContribution
	if debugScene1 {
		fmt.Printf("DEBUG: Studio similarity: %.3f * %.3f = %.3f\n", studioScore, c.weights.Studio, studioContribution)
	}

	finalScore := math.Min(totalScore, 1.0)
	if debugScene1 {
		fmt.Printf("DEBUG: Total weighted score: %.3f (capped at %.3f)\n", totalScore, finalScore)
		fmt.Printf("=== END DEBUG FOR SCENE 1 ===\n")
	}
	return finalScore, nil
}

// calculatePerformerSimilarity calculates similarity based on shared performers
func (c *SceneSimilarityCalculator) calculatePerformerSimilarity(performers1, performers2 []int) (float64, error) {
	if len(performers1) == 0 && len(performers2) == 0 {
		return 0.0, nil
	}

	shared := c.countSharedElements(performers1, performers2)
	total := len(performers1) + len(performers2) - shared

	if total == 0 {
		return 0.0, nil
	}

	// Jaccard similarity coefficient
	return float64(shared) / float64(total), nil
}

// calculateGroupSimilarity calculates similarity based on shared groups
func (c *SceneSimilarityCalculator) calculateGroupSimilarity(groups1, groups2 []models.GroupsScenes) (float64, error) {
	if len(groups1) == 0 && len(groups2) == 0 {
		return 0.0, nil
	}

	// Extract group IDs
	groupIDs1 := make([]int, len(groups1))
	for i, g := range groups1 {
		groupIDs1[i] = g.GroupID
	}

	groupIDs2 := make([]int, len(groups2))
	for i, g := range groups2 {
		groupIDs2[i] = g.GroupID
	}

	shared := c.countSharedElements(groupIDs1, groupIDs2)
	total := len(groupIDs1) + len(groupIDs2) - shared

	if total == 0 {
		return 0.0, nil
	}

	// Jaccard similarity coefficient
	return float64(shared) / float64(total), nil
}

// calculateTagSimilarity calculates similarity based on shared tags
// Dynamic weight: more shared tags = higher similarity score
func (c *SceneSimilarityCalculator) calculateTagSimilarity(tags1, tags2 []int) (float64, error) {
	// Check if this is for scene 1 (we need to pass this info somehow)
	// For now, let's always show debug for tag similarity
	fmt.Printf("DEBUG: calculateTagSimilarity: tags1=%v, tags2=%v\n", tags1, tags2)

	if len(tags1) == 0 && len(tags2) == 0 {
		fmt.Printf("DEBUG: Both tag lists empty, returning 0.0\n")
		return 0.0, nil
	}

	shared := c.countSharedElements(tags1, tags2)
	fmt.Printf("DEBUG: Shared tags: %d\n", shared)

	// If no shared tags, return 0
	if shared == 0 {
		fmt.Printf("DEBUG: No shared tags, returning 0.0\n")
		return 0.0, nil
	}

	total := len(tags1) + len(tags2) - shared
	fmt.Printf("DEBUG: Total unique tags: %d (len1=%d, len2=%d, shared=%d)\n", total, len(tags1), len(tags2), shared)

	if total == 0 {
		fmt.Printf("DEBUG: Total is 0, returning 0.0\n")
		return 0.0, nil
	}

	// Jaccard similarity coefficient
	jaccardScore := float64(shared) / float64(total)
	fmt.Printf("DEBUG: Jaccard score: %d / %d = %.3f\n", shared, total, jaccardScore)

	// Apply dynamic multiplier based on number of shared tags
	// More shared tags = higher multiplier (up to 2x for 5+ shared tags)
	multiplier := 1.0 + math.Min(float64(shared-1)*0.2, 1.0)
	fmt.Printf("DEBUG: Multiplier: 1.0 + min((%d-1)*0.2, 1.0) = %.3f\n", shared, multiplier)

	finalScore := jaccardScore * multiplier
	fmt.Printf("DEBUG: Final tag similarity: %.3f * %.3f = %.3f\n", jaccardScore, multiplier, finalScore)

	return finalScore, nil
}

// calculateStudioSimilarity calculates similarity based on shared studio
func (c *SceneSimilarityCalculator) calculateStudioSimilarity(studio1, studio2 *int) float64 {
	if studio1 == nil || studio2 == nil {
		return 0.0
	}

	if *studio1 == *studio2 {
		return 1.0
	}

	return 0.0
}

// countSharedElements counts the number of shared elements between two slices
func (c *SceneSimilarityCalculator) countSharedElements(slice1, slice2 []int) int {
	set1 := make(map[int]bool)
	for _, id := range slice1 {
		set1[id] = true
	}

	shared := 0
	for _, id := range slice2 {
		if set1[id] {
			shared++
		}
	}

	return shared
}

// CalculateAndStoreSimilarity calculates similarity between two scenes and stores it
func (c *SceneSimilarityCalculator) CalculateAndStoreSimilarity(ctx context.Context, scene1, scene2 *models.Scene) error {
	score, err := c.CalculateSimilarity(ctx, scene1, scene2)
	if err != nil {
		return err
	}

	// Special debug for scene ID 1
	debugScene1 := scene1.ID == 1 || scene2.ID == 1

	if debugScene1 {
		fmt.Printf("DEBUG: Total similarity score between scenes %d and %d: %.3f (threshold: %.3f)\n", scene1.ID, scene2.ID, score, c.weights.MinScore)
	}

	// Only store if similarity score is above threshold
	if score < c.weights.MinScore {
		if debugScene1 {
			fmt.Printf("DEBUG: Score %.3f below threshold %.3f, not storing\n", score, c.weights.MinScore)
		}
		return nil
	}

	if debugScene1 {
		fmt.Printf("DEBUG: Score %.3f above threshold %.3f, storing similarity\n", score, c.weights.MinScore)
	}

	similarity := models.SceneSimilarity{
		SceneID:         scene1.ID,
		SimilarSceneID:  scene2.ID,
		SimilarityScore: score,
	}

	// Create partial for timestamps
	partial := models.NewSceneSimilarityPartial()
	similarity.CreatedAt = partial.CreatedAt.Value
	similarity.UpdatedAt = partial.UpdatedAt.Value

	return c.repository.Upsert(ctx, similarity)
}

// RecalculateSceneSimilarities recalculates similarities for a specific scene
func (c *SceneSimilarityCalculator) RecalculateSceneSimilarities(ctx context.Context, sceneID int, allScenes []*models.Scene) error {
	// Get the scene
	scene, err := c.sceneRepo.Find(ctx, sceneID)
	if err != nil {
		return fmt.Errorf("finding scene %d: %w", sceneID, err)
	}

	// Load relationships for the main scene
	if err := scene.LoadRelationships(ctx, c.sceneRepo); err != nil {
		return fmt.Errorf("loading relationships for scene %d: %w", sceneID, err)
	}

	// Delete existing similarities for this scene
	if err := c.repository.DeleteByScene(ctx, sceneID); err != nil {
		return fmt.Errorf("deleting existing similarities for scene %d: %w", sceneID, err)
	}

	// Calculate similarities with all other scenes
	for _, otherScene := range allScenes {
		if otherScene.ID == sceneID {
			continue
		}

		// Load relationships for the other scene
		if err := otherScene.LoadRelationships(ctx, c.sceneRepo); err != nil {
			fmt.Printf("Error loading relationships for scene %d: %v\n", otherScene.ID, err)
			continue
		}

		// Calculate similarity without storing first
		score, err := c.CalculateSimilarity(ctx, scene, otherScene)
		if err != nil {
			fmt.Printf("Error calculating similarity between scenes %d and %d: %v\n", sceneID, otherScene.ID, err)
			continue
		}

		// Only store if similarity score is above threshold
		if score < c.weights.MinScore {
			continue
		}

		// Store the similarity
		similarity := models.SceneSimilarity{
			SceneID:         scene.ID,
			SimilarSceneID:  otherScene.ID,
			SimilarityScore: score,
		}

		// Create partial for timestamps
		partial := models.NewSceneSimilarityPartial()
		similarity.CreatedAt = partial.CreatedAt.Value
		similarity.UpdatedAt = partial.UpdatedAt.Value

		if err := c.repository.Upsert(ctx, similarity); err != nil {
			fmt.Printf("Error storing similarity between scenes %d and %d: %v\n", sceneID, otherScene.ID, err)
		}
	}

	return nil
}

// RecalculateAllSimilarities recalculates similarities for all scenes
func (c *SceneSimilarityCalculator) RecalculateAllSimilarities(ctx context.Context, scenes []*models.Scene) error {
	// Clear all existing similarities
	// Note: This is a simple approach. In production, you might want to do this more efficiently
	for _, scene := range scenes {
		if err := c.repository.DeleteByScene(ctx, scene.ID); err != nil {
			return fmt.Errorf("deleting similarities for scene %d: %w", scene.ID, err)
		}
	}

	// Calculate similarities for all pairs
	for i, scene1 := range scenes {
		for j := i + 1; j < len(scenes); j++ {
			scene2 := scenes[j]

			if err := c.CalculateAndStoreSimilarity(ctx, scene1, scene2); err != nil {
				// Log error but continue
				fmt.Printf("Error calculating similarity between scenes %d and %d: %v\n", scene1.ID, scene2.ID, err)
			}
		}
	}

	return nil
}

// RecalculateAllSimilaritiesWithTxn recalculates similarities for all scenes within a transaction
func (c *SceneSimilarityCalculator) RecalculateAllSimilaritiesWithTxn(ctx context.Context, repo models.Repository) error {
	var scenes []*models.Scene
	if err := repo.WithTxn(ctx, func(ctx context.Context) error {
		var err error
		scenes, err = repo.Scene.AllWithRelationships(ctx)
		return err
	}); err != nil {
		return fmt.Errorf("finding all scenes: %w", err)
	}

	// Clear all existing similarities and calculate new ones in batches
	batchSize := 100
	for i := 0; i < len(scenes); i += batchSize {
		end := i + batchSize
		if end > len(scenes) {
			end = len(scenes)
		}

		batch := scenes[i:end]

		if err := repo.WithTxn(ctx, func(ctx context.Context) error {
			// Clear existing similarities for this batch
			for _, scene := range batch {
				if err := c.repository.DeleteByScene(ctx, scene.ID); err != nil {
					return fmt.Errorf("deleting similarities for scene %d: %w", scene.ID, err)
				}
			}

			// Calculate similarities for all pairs in this batch
			for j, scene1 := range batch {
				for k := j + 1; k < len(batch); k++ {
					scene2 := batch[k]

					if err := c.CalculateAndStoreSimilarity(ctx, scene1, scene2); err != nil {
						// Log error but continue
						fmt.Printf("Error calculating similarity between scenes %d and %d: %v\n", scene1.ID, scene2.ID, err)
					}
				}
			}

			return nil
		}); err != nil {
			return fmt.Errorf("processing batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}
