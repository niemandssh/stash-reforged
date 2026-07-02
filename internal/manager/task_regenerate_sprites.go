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
	if IsEmbedSpriteScene(&t.Scene) {
		return fmt.Sprintf("Regenerating embed sprites for scene %d", t.Scene.ID)
	}
	return fmt.Sprintf("Regenerating sprites for %s", t.Scene.Path)
}

func (t *RegenerateSpritesTask) Start(ctx context.Context) {
	progress := &job.Progress{}
	t.Execute(ctx, progress)
}

func (t *RegenerateSpritesTask) Execute(ctx context.Context, progress *job.Progress) error {
	logger.Infof("[regenerate-sprites] starting sprite regeneration for scene %d", t.Scene.ID)

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
		return reloadedScene.LoadFiles(ctx, t.Repository.Scene)
	}); err != nil {
		return err
	}

	t.Scene = *reloadedScene

	if IsEmbedSpriteScene(&t.Scene) {
		return t.executeEmbed(ctx, progress)
	}

	return t.executeLocal(ctx, progress)
}

func (t *RegenerateSpritesTask) executeEmbed(ctx context.Context, progress *job.Progress) error {
	sceneHash := ResolveRemoteSceneSpriteHash(t.Scene.ID)
	spriteImagePath := t.Paths.Scene.GetSpriteImageFilePath(sceneHash)
	spriteVttPath := t.Paths.Scene.GetSpriteVttFilePath(sceneHash)

	progress.ExecuteTask("Deleting existing sprites", func() {
		if err := t.deleteExistingSprites(spriteImagePath, spriteVttPath); err != nil {
			logger.Warnf("[regenerate-sprites] failed to delete existing embed sprites: %v", err)
		}
	})

	progress.ExecuteTask("Generating new sprites", func() {
		task := GenerateEmbedSpriteTask{
			Scene:     t.Scene,
			Overwrite: true,
		}
		if err := task.Generate(ctx); err != nil {
			logger.Errorf("[regenerate-sprites] failed to generate embed sprites: %v", err)
		}
	})

	logger.Infof("[regenerate-sprites] completed embed sprite regeneration for scene %d", t.Scene.ID)
	return nil
}

func (t *RegenerateSpritesTask) executeLocal(ctx context.Context, progress *job.Progress) error {
	sceneHash := t.Scene.GetHash(t.FileNamingAlgorithm)
	logger.Infof("[regenerate-sprites] scene hash: %s", sceneHash)

	if sceneHash == "" {
		if err := t.Scene.LoadFiles(ctx, t.Repository.Scene); err != nil {
			logger.Warnf("[regenerate-sprites] failed to load scene files: %v", err)
		} else {
			for _, vf := range t.Scene.Files.List() {
				if phash := vf.Base().Fingerprints.Get(models.FingerprintTypePhash); phash != nil {
					sceneHash = phash.(string)
					break
				}
			}
		}
	}

	if sceneHash == "" {
		return fmt.Errorf("no hash available for sprite generation")
	}

	spriteImagePath := t.Paths.Scene.GetSpriteImageFilePath(sceneHash)
	spriteVttPath := t.Paths.Scene.GetSpriteVttFilePath(sceneHash)

	progress.ExecuteTask("Deleting existing sprites", func() {
		if err := t.deleteExistingSprites(spriteImagePath, spriteVttPath); err != nil {
			logger.Warnf("[regenerate-sprites] failed to delete existing sprites: %v", err)
		}
	})

	progress.ExecuteTask("Generating new sprites", func() {
		spriteTask := GenerateSpriteTask{
			Scene:               t.Scene,
			Overwrite:           true,
			fileNamingAlgorithm: t.FileNamingAlgorithm,
		}
		spriteTask.Start(ctx)
	})

	logger.Infof("[regenerate-sprites] completed sprite regeneration for scene %d", t.Scene.ID)
	return nil
}

func (t *RegenerateSpritesTask) deleteExistingSprites(spriteImagePath, spriteVttPath string) error {
	if _, err := os.Stat(spriteImagePath); err == nil {
		if err := os.Remove(spriteImagePath); err != nil {
			return fmt.Errorf("failed to delete sprite image: %w", err)
		}
	}

	if _, err := os.Stat(spriteVttPath); err == nil {
		if err := os.Remove(spriteVttPath); err != nil {
			return fmt.Errorf("failed to delete sprite VTT: %w", err)
		}
	}

	return nil
}
