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
		Performers: 0.4, // 30% weight
		Groups:     0.2, // 20% weight
		Tags:       0.5, // 50% weight (highest)
		Studio:     0.2, // 10% weight (lowest)
		/////////////////////////////////////////////////////////////
		MinScore: 0.1, // Only store similarities with score >= 0.1
	}
}

// SceneSimilarityCalculator calculates similarity between scenes
type SceneSimilarityCalculator struct {
	repository models.SceneSimilarityReaderWriter
	sceneRepo  models.SceneReader
	tagRepo    models.TagReader
	weights    SimilarityWeights
}

// NewSceneSimilarityCalculator creates a new similarity calculator
func NewSceneSimilarityCalculator(repository models.SceneSimilarityReaderWriter, sceneRepo models.SceneReader, tagRepo models.TagReader, weights SimilarityWeights) *SceneSimilarityCalculator {
	return &SceneSimilarityCalculator{
		repository: repository,
		sceneRepo:  sceneRepo,
		tagRepo:    tagRepo,
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
	tagScore, err := c.calculateTagSimilarity(ctx, tags1, tags2)
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

	// Apply broken status penalty
	brokenPenalty := 1.0
	if scene1.IsBroken || scene2.IsBroken {
		brokenPenalty = 0.3 // Strong penalty for broken scenes
		if debugScene1 {
			fmt.Printf("DEBUG: Broken penalty applied: %.3f\n", brokenPenalty)
		}
	}

	finalScore := math.Min(totalScore, 1.0) * brokenPenalty
	if debugScene1 {
		fmt.Printf("DEBUG: Total weighted score: %.3f (capped at %.3f, broken penalty: %.3f)\n", totalScore, math.Min(totalScore, 1.0), brokenPenalty)
		fmt.Printf("DEBUG: Final score: %.3f\n", finalScore)
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

// calculateTagSimilarity calculates similarity based on shared tags with weights
// Weight system: 0.0 = least important, 1.0 = most important
// Higher weight tags contribute more to similarity calculation
func (c *SceneSimilarityCalculator) calculateTagSimilarity(ctx context.Context, tags1, tags2 []int) (float64, error) {
	// Check if this is for scene 1 (we need to pass this info somehow)
	// For now, let's always show debug for tag similarity
	fmt.Printf("DEBUG: calculateTagSimilarity: tags1=%v, tags2=%v\n", tags1, tags2)

	if len(tags1) == 0 && len(tags2) == 0 {
		fmt.Printf("DEBUG: Both tag lists empty, returning 0.0\n")
		return 0.0, nil
	}

	// Get all unique tags from both scenes
	allTags := make(map[int]bool)
	for _, tagID := range tags1 {
		allTags[tagID] = true
	}
	for _, tagID := range tags2 {
		allTags[tagID] = true
	}

	// Load tag weights from database
	tagWeights := make(map[int]float64)
	for tagID := range allTags {
		tag, err := c.tagRepo.Find(ctx, tagID)
		if err != nil {
			fmt.Printf("DEBUG: Error loading tag %d: %v\n", tagID, err)
			continue
		}
		if tag != nil {
			tagWeights[tagID] = tag.Weight
		} else {
			tagWeights[tagID] = 0.5 // Default weight if tag not found
		}
	}

	// Calculate weighted similarity
	var weightedShared float64
	var weightedTotal float64

	// Calculate weighted shared tags
	sharedTags := make(map[int]bool)
	for _, tagID := range tags1 {
		if c.contains(tags2, tagID) {
			sharedTags[tagID] = true
			weight := tagWeights[tagID]
			weightedShared += weight
		}
	}

	// Calculate weighted total for both tag lists
	for _, tagID := range tags1 {
		weight := tagWeights[tagID]
		weightedTotal += weight
	}
	for _, tagID := range tags2 {
		weight := tagWeights[tagID]
		weightedTotal += weight
	}

	// Subtract shared weights to avoid double counting
	weightedTotal -= weightedShared

	fmt.Printf("DEBUG: Weighted shared: %.3f, Weighted total: %.3f\n", weightedShared, weightedTotal)

	// If no shared tags, return 0
	if weightedShared == 0 {
		fmt.Printf("DEBUG: No shared tags, returning 0.0\n")
		return 0.0, nil
	}

	if weightedTotal == 0 {
		fmt.Printf("DEBUG: Weighted total is 0, returning 0.0\n")
		return 0.0, nil
	}

	// Check for 100% match: all tags from first scene are in second scene
	allTags1InTags2 := true
	for _, tagID := range tags1 {
		if !c.contains(tags2, tagID) {
			allTags1InTags2 = false
			break
		}
	}

	if allTags1InTags2 {
		fmt.Printf("DEBUG: 100%% match - all tags from first scene are in second scene\n")
		// Calculate maximum multiplier for 100% match based on tag weights
		var maxMultiplier float64
		for _, tagID := range tags1 {
			weight := tagWeights[tagID]
			if weight >= 1.0 {
				maxMultiplier += 0.5
			} else if weight > 0.7 {
				maxMultiplier += 0.4
			} else if weight > 0.5 {
				maxMultiplier += 0.3
			} else {
				maxMultiplier += 0.2
			}
		}
		// Ensure minimum multiplier of 1.0
		if maxMultiplier < 1.0 {
			maxMultiplier = 1.0
		}
		fmt.Printf("DEBUG: 100%% match multiplier: %.3f\n", maxMultiplier)
		return maxMultiplier, nil
	}

	// Calculate base similarity using weighted tags
	var weightedTags1, weightedTags2 float64
	for _, tagID := range tags1 {
		weightedTags1 += tagWeights[tagID]
	}
	for _, tagID := range tags2 {
		weightedTags2 += tagWeights[tagID]
	}

	// Calculate weighted similarity: shared weight / total weight of first scene
	// Start from 0.5 base and add weighted contribution
	baseSimilarity := 0.5 + (weightedShared/weightedTags1)*0.5
	fmt.Printf("DEBUG: Base similarity: 0.5 + (%.3f / %.3f) * 0.5 = %.3f\n", weightedShared, weightedTags1, baseSimilarity)

	// Calculate weight-based multiplier
	var highWeightTags, mediumWeightTags, lowWeightTags int
	var totalWeight float64

	for tagID := range sharedTags {
		weight := tagWeights[tagID]
		totalWeight += weight

		if weight >= 1.0 {
			highWeightTags++
		} else if weight > 0.7 {
			highWeightTags++
		} else if weight > 0.5 {
			mediumWeightTags++
		} else {
			lowWeightTags++
		}
	}

	// Calculate multiplier based on weight distribution
	var multiplier float64
	if highWeightTags > 0 {
		// Significant multiplier for high weight tags
		multiplier = 1.0 + float64(highWeightTags)*0.3
	} else if mediumWeightTags > 0 {
		// Small multiplier for medium weight tags
		multiplier = 1.0 + float64(mediumWeightTags)*0.1
	} else {
		// Penalty for low weight tags
		multiplier = 0.5 + float64(lowWeightTags)*0.1
	}

	// Additional bonus for high average weight
	avgWeight := totalWeight / float64(len(sharedTags))
	if avgWeight > 0.8 {
		multiplier *= 1.2
	} else if avgWeight < 0.3 {
		multiplier *= 0.8
	}

	fmt.Printf("DEBUG: Weight distribution: high=%d, medium=%d, low=%d, avg=%.3f\n", highWeightTags, mediumWeightTags, lowWeightTags, avgWeight)
	fmt.Printf("DEBUG: Multiplier: %.3f\n", multiplier)

	finalScore := baseSimilarity * multiplier
	fmt.Printf("DEBUG: Final weighted tag similarity: %.3f * %.3f = %.3f\n", baseSimilarity, multiplier, finalScore)

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

// contains checks if a slice contains a specific element
func (c *SceneSimilarityCalculator) contains(slice []int, element int) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
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
