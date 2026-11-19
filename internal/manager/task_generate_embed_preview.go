package manager

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scene/generate"
)

var embedPreviewJobs sync.Map

type GenerateEmbedPreviewTask struct {
	Scene models.Scene
}

func (t *GenerateEmbedPreviewTask) GetDescription() string {
	return fmt.Sprintf("Generating embed preview for scene %d", t.Scene.ID)
}

func EmbedPreviewPath(scene *models.Scene) string {
	return instance.Paths.Scene.GetVideoPreviewPath(models.RemoteSceneSpriteHash(scene.ID))
}

// EnsureEmbedPreviewJob queues preview generation when the preview video is missing.
func EnsureEmbedPreviewJob(ctx context.Context, scene *models.Scene) {
	if !IsEmbedSpriteScene(scene) {
		return
	}
	if exists, _ := fsutil.FileExists(EmbedPreviewPath(scene)); exists {
		return
	}
	if _, loaded := embedPreviewJobs.LoadOrStore(scene.ID, true); loaded {
		return
	}

	task := &GenerateEmbedPreviewTask{Scene: *scene}
	jobExec := job.MakeJobExec(func(ctx context.Context, progress *job.Progress) error {
		return task.Generate(ctx)
	})
	jobID := GetInstance().JobManager.Start(ctx, task.GetDescription(), jobExec)
	logger.Infof("started embed preview generation for scene %d (job %d)", scene.ID, jobID)
}

func (t *GenerateEmbedPreviewTask) Generate(ctx context.Context) error {
	defer embedPreviewJobs.Delete(t.Scene.ID)

	embedURL := strings.TrimSpace(*t.Scene.EmbedURL)

	duration, err := ResolveEmbedDuration(embedURL)
	if err != nil {
		return err
	}
	if duration <= 0 {
		return fmt.Errorf("invalid embed video duration")
	}

	input, headers, cleanup, err := prepareEmbedSpriteSource(ctx, embedURL)
	if err != nil {
		return err
	}
	defer cleanup()

	options := getGeneratePreviewOptions(GeneratePreviewOptionsInput{})
	options.HTTPHeaders = headers

	g := generate.Generator{
		Encoder:      instance.FFMpeg,
		FFMpegConfig: instance.Config,
		LockManager:  instance.ReadLockManager,
		ScenePaths:   instance.Paths.Scene,
	}

	hash := models.RemoteSceneSpriteHash(t.Scene.ID)
	if err := g.PreviewVideo(ctx, input, duration, hash, options, false, false); err != nil {
		logger.Warnf("[generator] failed generating embed preview for scene %d, trying fallback", t.Scene.ID)
		if err := g.PreviewVideo(ctx, input, duration, hash, options, true, false); err != nil {
			return err
		}
	}

	logger.Infof("generated embed preview for scene %d", t.Scene.ID)
	return nil
}
