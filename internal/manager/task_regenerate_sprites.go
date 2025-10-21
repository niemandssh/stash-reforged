package manager

import (
	"context"
	"fmt"
	"os"

	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/paths"
)

type RegenerateSpritesTask struct {
	Scene               models.Scene
	FileNamingAlgorithm models.HashAlgorithm
	Repository          models.Repository
	Paths               *paths.Paths
}

func (t *RegenerateSpritesTask) GetDescription() string {
	return fmt.Sprintf("Regenerating sprites for %s", t.Scene.Path)
}

func (t *RegenerateSpritesTask) Start(ctx context.Context) {
	progress := &job.Progress{}
	t.Execute(ctx, progress)
}

func (t *RegenerateSpritesTask) Execute(ctx context.Context, progress *job.Progress) error {
	logger.Infof("[regenerate-sprites] starting sprite regeneration for scene %d", t.Scene.ID)

	// Reload scene from database to get OSHash and Checksum from primary file
	logger.Infof("[regenerate-sprites] reloading scene from database to get hash")
	var reloadedScene *models.Scene
	if err := t.Repository.WithTxn(ctx, func(ctx context.Context) error {
		var err error
		reloadedScene, err = t.Repository.Scene.Find(ctx, t.Scene.ID)
		if err != nil {
			return fmt.Errorf("failed to reload scene from database: %w", err)
		}
		if reloadedScene == nil {
			return fmt.Errorf("scene %d not found", t.Scene.ID)
		}
		return nil
	}); err != nil {
		return err
	}

	// Update our scene with the reloaded data
	t.Scene = *reloadedScene

	// Get scene hash
	sceneHash := t.Scene.GetHash(t.FileNamingAlgorithm)
	logger.Infof("[regenerate-sprites] scene hash: %s", sceneHash)
	logger.Infof("[regenerate-sprites] scene OSHash: '%s', Checksum: '%s'", t.Scene.OSHash, t.Scene.Checksum)
	logger.Infof("[regenerate-sprites] file naming algorithm: %s", t.FileNamingAlgorithm)
	logger.Infof("[regenerate-sprites] scene PrimaryFileID: %v", t.Scene.PrimaryFileID)
	logger.Infof("[regenerate-sprites] scene Path: '%s'", t.Scene.Path)

	// If hash is empty, try to get phash from scene files
	if sceneHash == "" {
		logger.Infof("[regenerate-sprites] scene hash is empty, trying to get phash from files")

		// Load scene files to get phash
		if err := t.Scene.LoadFiles(ctx, t.Repository.Scene); err != nil {
			logger.Warnf("[regenerate-sprites] failed to load scene files: %v", err)
		} else {
			logger.Infof("[regenerate-sprites] loaded %d files for scene", len(t.Scene.Files.List()))
			
			// Look for phash in scene files
			for _, vf := range t.Scene.Files.List() {
				videoFile := vf
				logger.Infof("[regenerate-sprites] file %d: %s", videoFile.ID, videoFile.Base().Path)
				logger.Infof("[regenerate-sprites] file %d fingerprints: %v", videoFile.ID, videoFile.Base().Fingerprints)
				
				// Check all fingerprint types
				if oshash := videoFile.Base().Fingerprints.Get(models.FingerprintTypeOshash); oshash != nil {
					logger.Infof("[regenerate-sprites] file %d OSHash: %s", videoFile.ID, oshash)
				}
				if checksum := videoFile.Base().Fingerprints.Get(models.FingerprintTypeMD5); checksum != nil {
					logger.Infof("[regenerate-sprites] file %d Checksum: %s", videoFile.ID, checksum)
				}
				if phash := videoFile.Base().Fingerprints.Get(models.FingerprintTypePhash); phash != nil {
					sceneHash = phash.(string)
					logger.Infof("[regenerate-sprites] found phash in file %d: %s", videoFile.ID, sceneHash)
					break
				}
			}
		}
	}

	// Verify that we have some hash available
	if sceneHash == "" {
		logger.Errorf("[regenerate-sprites] no hash available (OSHash, Checksum, or phash), cannot generate sprites")
		return fmt.Errorf("no hash available for sprite generation")
	}

	// Get sprite file paths
	spriteImagePath := t.Paths.Scene.GetSpriteImageFilePath(sceneHash)
	spriteVttPath := t.Paths.Scene.GetSpriteVttFilePath(sceneHash)

	logger.Infof("[regenerate-sprites] sprite image path: %s", spriteImagePath)
	logger.Infof("[regenerate-sprites] sprite VTT path: %s", spriteVttPath)

	// Delete existing sprite files
	progress.ExecuteTask("Deleting existing sprites", func() {
		if err := t.deleteExistingSprites(spriteImagePath, spriteVttPath); err != nil {
			logger.Warnf("[regenerate-sprites] failed to delete existing sprites: %v", err)
		}
	})

	// Generate new sprites
	progress.ExecuteTask("Generating new sprites", func() {
		if err := t.generateNewSprites(ctx, progress); err != nil {
			logger.Errorf("[regenerate-sprites] failed to generate new sprites: %v", err)
		}
	})

	logger.Infof("[regenerate-sprites] completed sprite regeneration for scene %d", t.Scene.ID)
	return nil
}

func (t *RegenerateSpritesTask) deleteExistingSprites(spriteImagePath, spriteVttPath string) error {
	// Delete sprite image
	if _, err := os.Stat(spriteImagePath); err == nil {
		logger.Infof("[regenerate-sprites] deleting existing sprite image: %s", spriteImagePath)
		if err := os.Remove(spriteImagePath); err != nil {
			return fmt.Errorf("failed to delete sprite image: %w", err)
		}
		logger.Infof("[regenerate-sprites] deleted sprite image: %s", spriteImagePath)
	} else {
		logger.Infof("[regenerate-sprites] sprite image does not exist: %s", spriteImagePath)
	}

	// Delete sprite VTT
	if _, err := os.Stat(spriteVttPath); err == nil {
		logger.Infof("[regenerate-sprites] deleting existing sprite VTT: %s", spriteVttPath)
		if err := os.Remove(spriteVttPath); err != nil {
			return fmt.Errorf("failed to delete sprite VTT: %w", err)
		}
		logger.Infof("[regenerate-sprites] deleted sprite VTT: %s", spriteVttPath)
	} else {
		logger.Infof("[regenerate-sprites] sprite VTT does not exist: %s", spriteVttPath)
	}

	return nil
}

func (t *RegenerateSpritesTask) generateNewSprites(ctx context.Context, progress *job.Progress) error {
	// Create sprite generation task
	spriteTask := GenerateSpriteTask{
		Scene:               t.Scene,
		Overwrite:           true, // Force regeneration
		fileNamingAlgorithm: t.FileNamingAlgorithm,
	}

	// Run sprite generation
	logger.Infof("[regenerate-sprites] generating new sprites for scene %d", t.Scene.ID)
	spriteTask.Start(ctx)
	logger.Infof("[regenerate-sprites] completed sprite generation for scene %d", t.Scene.ID)

	return nil
}
