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

type RegenerateAIVisionTask struct {
	Scene               models.Scene
	FileNamingAlgorithm models.HashAlgorithm
	Repository          models.Repository
	Paths               *paths.Paths
}

func (t *RegenerateAIVisionTask) GetDescription() string {
	return AIVisionJobDescription(t.Scene.ID)
}

func (t *RegenerateAIVisionTask) Start(ctx context.Context) {
	progress := &job.Progress{}
	t.Execute(ctx, progress)
}

func (t *RegenerateAIVisionTask) Execute(ctx context.Context, progress *job.Progress) error {
	logger.Infof("[regenerate-ai-vision] starting for scene %d", t.Scene.ID)

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

	t.Scene = *reloadedScene

	sceneHash := t.Scene.GetHash(t.FileNamingAlgorithm)
	if sceneHash == "" {
		if err := t.Scene.LoadFiles(ctx, t.Repository.Scene); err != nil {
			logger.Warnf("[regenerate-ai-vision] failed to load scene files: %v", err)
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
		return fmt.Errorf("no hash available for AI vision panel generation")
	}

	panelPaths := t.Paths.Scene.GetAIVisionPanelFilePaths(sceneHash)

	progress.ExecuteTask("Deleting existing AI vision panels", func() {
		for _, panelPath := range panelPaths {
			if _, err := os.Stat(panelPath); err == nil {
				if err := os.Remove(panelPath); err != nil {
					logger.Warnf("[regenerate-ai-vision] failed to delete %s: %v", panelPath, err)
				}
			}
		}
	})

	progress.ExecuteTask("Generating AI vision panels", func() {
		task := GenerateAIVisionTask{
			Scene:               t.Scene,
			Overwrite:           true,
			fileNamingAlgorithm: t.FileNamingAlgorithm,
		}
		task.Start(ctx)
	})

	logger.Infof("[regenerate-ai-vision] completed for scene %d", t.Scene.ID)
	return nil
}
