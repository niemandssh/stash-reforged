package manager

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scene/generate"
)

type GenerateAIVisionTask struct {
	Scene               models.Scene
	Overwrite           bool
	fileNamingAlgorithm models.HashAlgorithm
}

func AIVisionJobDescription(sceneID int) string {
	return fmt.Sprintf("Generating AI vision panels for scene %d", sceneID)
}

func (t *GenerateAIVisionTask) GetDescription() string {
	return AIVisionJobDescription(t.Scene.ID)
}

func (t *GenerateAIVisionTask) Start(ctx context.Context) {
	if err := t.Execute(ctx, nil); err != nil {
		logger.Errorf("error generating AI vision panels: %s", err.Error())
		logErrorOutput(err)
	}
}

func (t *GenerateAIVisionTask) Execute(ctx context.Context, _ *job.Progress) error {
	if t.Scene.Path == "" {
		return fmt.Errorf("no video path for AI vision panel generation")
	}

	if !t.required() {
		return nil
	}

	if exists, err := fsutil.FileExists(t.Scene.Path); err != nil || !exists {
		return fmt.Errorf("video file no longer exists: %s", t.Scene.Path)
	}

	videoFile, err := instance.FFProbe.NewVideoFile(t.Scene.Path)
	if err != nil {
		return fmt.Errorf("reading video file: %w", err)
	}

	sceneHash := t.Scene.GetHash(t.fileNamingAlgorithm)
	if sceneHash == "" {
		return fmt.Errorf("no hash available for AI vision panel generation")
	}

	outputPaths := instance.Paths.Scene.GetAIVisionPanelFilePaths(sceneHash)
	gen := &generate.Generator{
		Encoder:      instance.FFMpeg,
		FFMpegConfig: instance.Config,
		LockManager:  instance.ReadLockManager,
		ScenePaths:   instance.Paths.Scene,
	}

	duration := videoFile.VideoStreamDuration
	if duration <= 0 || (videoFile.FileDuration > 0 && videoFile.FileDuration < duration) {
		duration = videoFile.FileDuration
	}
	if duration <= 0 {
		return fmt.Errorf("video duration is unknown")
	}

	if err := gen.GenerateAIVisionPanels(ctx, t.Scene.Path, duration, outputPaths); err != nil {
		return err
	}

	if !instance.Paths.Scene.AIVisionPanelsExist(sceneHash) {
		return fmt.Errorf("AI vision panels were not created")
	}

	return nil
}

func (t GenerateAIVisionTask) required() bool {
	if t.Scene.Path == "" {
		return false
	}

	if t.Overwrite {
		return true
	}

	sceneHash := t.Scene.GetHash(t.fileNamingAlgorithm)
	return !instance.Paths.Scene.AIVisionPanelsExist(sceneHash)
}
