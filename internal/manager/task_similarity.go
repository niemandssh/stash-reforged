package manager

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/similarity"
)

type SimilarityJob struct {
	repository models.Repository
	sceneID    *int // If nil, recalculate all similarities
}

func (j *SimilarityJob) Execute(ctx context.Context, progress *job.Progress) error {
	// Use global repository instance like other tasks
	repo := instance.Repository

	// Create similarity calculator
	weights := similarity.DefaultSimilarityWeights()
	calculator := similarity.NewSceneSimilarityCalculator(
		repo.SceneSimilarity,
		repo.Scene,
		repo.Tag,
		weights,
	)

	// Get transaction manager from repository
	txnManager := repo.TxnManager

	if j.sceneID != nil {
		// Recalculate similarities for a specific scene
		logger.Infof("Recalculating similarities for scene %d", *j.sceneID)
		progress.SetTotal(1)
		progress.SetProcessed(0)

		// Get scenes with database context
		dbCtx, err := instance.Database.WithDatabase(context.Background())
		if err != nil {
			return fmt.Errorf("creating database context: %w", err)
		}

		// Check if the scene still exists before proceeding
		scene, err := repo.Scene.Find(dbCtx, *j.sceneID)
		if err != nil {
			return fmt.Errorf("finding scene %d: %w", *j.sceneID, err)
		}
		if scene == nil {
			logger.Warnf("Scene %d not found, skipping similarity recalculation", *j.sceneID)
			progress.SetProcessed(1)
			return nil
		}

		scenes, err := repo.Scene.All(dbCtx)
		if err != nil {
			return fmt.Errorf("finding all scenes: %w", err)
		}

		// Calculate similarities using database context (individual transactions for each operation)
		err = calculator.RecalculateSceneSimilarities(dbCtx, *j.sceneID, scenes, txnManager)
		if err != nil {
			return fmt.Errorf("recalculating similarities for scene %d: %w", *j.sceneID, err)
		}

		progress.SetProcessed(1)
		logger.Infof("Completed similarity recalculation for scene %d", *j.sceneID)
	} else {
		// Recalculate all similarities
		logger.Info("Recalculating all scene similarities")

		// Get scenes with database context and all relationships
		dbCtx, err := instance.Database.WithDatabase(context.Background())
		if err != nil {
			return fmt.Errorf("creating database context: %w", err)
		}

		// For global recalculation, load scenes with all relationships
		scenes, err := repo.Scene.AllWithRelationships(dbCtx)
		if err != nil {
			return fmt.Errorf("finding all scenes with relationships: %w", err)
		}

		progress.SetTotal(len(scenes))
		progress.SetProcessed(0)

		// Calculate similarities using individual transactions for each operation
		err = calculator.RecalculateAllSimilarities(dbCtx, scenes, txnManager)
		if err != nil {
			return fmt.Errorf("recalculating all similarities: %w", err)
		}

		progress.SetProcessed(progress.GetTotal())
		logger.Info("Completed similarity recalculation for all scenes")
	}

	return nil
}

func (j *SimilarityJob) GetDescription() string {
	if j.sceneID != nil {
		return fmt.Sprintf("Recalculating similarities for scene %d", *j.sceneID)
	}
	return "Recalculating all scene similarities"
}

// NewSimilarityJob creates a new similarity calculation job
func NewSimilarityJob(repository models.Repository, sceneID *int) *SimilarityJob {
	return &SimilarityJob{
		repository: repository,
		sceneID:    sceneID,
	}
}

func CreateSimilarityJob(repository models.Repository, sceneID *int) *SimilarityJob {
	return NewSimilarityJob(repository, sceneID)
}
