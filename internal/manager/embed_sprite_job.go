package manager

import (
	"context"
	"strconv"
	"sync"

	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/models"
)

var embedSpriteJobs sync.Map

// EmbedSpritesReady reports whether both sprite image and VTT exist for a remote scene.
func EmbedSpritesReady(scene *models.Scene) bool {
	if !IsEmbedSpriteScene(scene) {
		return false
	}

	hash := ResolveRemoteSceneSpriteHash(scene.ID)
	imageExists, _ := fsutil.FileExists(instance.Paths.Scene.GetSpriteImageFilePath(hash))
	vttExists, _ := fsutil.FileExists(instance.Paths.Scene.GetSpriteVttFilePath(hash))
	return imageExists && vttExists
}

// EnsureEmbedSpriteJob queues sprite generation when sprites are missing.
// Returns true when a new job was started.
func EnsureEmbedSpriteJob(ctx context.Context, scene *models.Scene) (bool, error) {
	if !IsEmbedSpriteScene(scene) || EmbedSpritesReady(scene) {
		return false, nil
	}

	if _, loaded := embedSpriteJobs.LoadOrStore(scene.ID, true); loaded {
		return false, nil
	}

	jobID, err := StartEmbedSpriteJob(ctx, scene)
	if err != nil {
		embedSpriteJobs.Delete(scene.ID)
		return false, err
	}

	if jobID == "0" {
		embedSpriteJobs.Delete(scene.ID)
		return false, nil
	}

	return true, nil
}

func clearEmbedSpriteJob(sceneID int) {
	embedSpriteJobs.Delete(sceneID)
}

// StartEmbedSpriteJob queues sprite generation for a remote embed scene.
func StartEmbedSpriteJob(ctx context.Context, scene *models.Scene) (string, error) {
	if !IsEmbedSpriteScene(scene) {
		return "0", nil
	}

	mgr := GetInstance()
	task := &GenerateEmbedSpriteTask{
		Scene:     *scene,
		Overwrite: false,
	}

	jobExec := job.MakeJobExec(func(ctx context.Context, progress *job.Progress) error {
		return task.Generate(ctx)
	})
	jobID := mgr.JobManager.Start(ctx, task.GetDescription(), jobExec)

	return strconv.Itoa(jobID), nil
}
