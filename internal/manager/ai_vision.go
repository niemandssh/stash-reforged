package manager

import (
	"context"
	"fmt"
	"strconv"

	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/models"
)

// EnsureSceneAIVisionPanels generates AI vision panels for a scene if they do not exist yet.
// If generation is already in progress, it waits for that job to finish.
func EnsureSceneAIVisionPanels(ctx context.Context, repo models.Repository, sceneID int) error {
	mgr := GetInstance()
	fileNamingAlgorithm := mgr.Config.GetVideoFileNamingAlgorithm()

	var scene *models.Scene
	if err := repo.WithReadTxn(ctx, func(ctx context.Context) error {
		var err error
		scene, err = repo.Scene.Find(ctx, sceneID)
		if err != nil {
			return err
		}
		if scene == nil {
			return fmt.Errorf("scene not found")
		}

		if err := scene.LoadFiles(ctx, repo.Scene); err != nil {
			return err
		}

		if scene.Path == "" {
			if pf := scene.Files.Primary(); pf != nil {
				scene.Path = pf.Path
			} else if files := scene.Files.List(); len(files) > 0 {
				scene.Path = files[0].Path
			}
		}

		return nil
	}); err != nil {
		return err
	}

	sceneHash := scene.GetHash(fileNamingAlgorithm)
	if sceneHash != "" && mgr.Paths.Scene.AIVisionPanelsExist(sceneHash) {
		return nil
	}

	if running := findRunningAIVisionJob(mgr.JobManager, sceneID); running != nil {
		if err := mgr.JobManager.Wait(ctx, running.ID); err != nil {
			return fmt.Errorf("waiting for AI vision panel generation: %w", err)
		}
	} else {
		task := &GenerateAIVisionTask{
			Scene:               *scene,
			Overwrite:           false,
			fileNamingAlgorithm: fileNamingAlgorithm,
		}

		jobExec := job.MakeJobExec(task.Execute)
		jobID := mgr.JobManager.Start(ctx, task.GetDescription(), jobExec)
		if err := mgr.JobManager.Wait(ctx, jobID); err != nil {
			return fmt.Errorf("AI vision panel generation failed: %w", err)
		}
	}

	if sceneHash == "" {
		sceneHash = scene.GetHash(fileNamingAlgorithm)
	}
	if sceneHash == "" || !mgr.Paths.Scene.AIVisionPanelsExist(sceneHash) {
		return fmt.Errorf("AI vision panels are not available for this scene")
	}

	return nil
}

// StartSceneAIVisionPanelsJob starts AI vision panel generation when panels are missing.
// Returns job ID as string, or "0" if panels already exist.
func StartSceneAIVisionPanelsJob(ctx context.Context, scene *models.Scene) (string, error) {
	mgr := GetInstance()
	fileNamingAlgorithm := mgr.Config.GetVideoFileNamingAlgorithm()

	sceneHash := scene.GetHash(fileNamingAlgorithm)
	if sceneHash != "" && mgr.Paths.Scene.AIVisionPanelsExist(sceneHash) {
		return "0", nil
	}

	if running := findRunningAIVisionJob(mgr.JobManager, scene.ID); running != nil {
		return strconv.Itoa(running.ID), nil
	}

	task := &GenerateAIVisionTask{
		Scene:               *scene,
		Overwrite:           false,
		fileNamingAlgorithm: fileNamingAlgorithm,
	}

	jobExec := job.MakeJobExec(task.Execute)
	jobID := mgr.JobManager.Start(ctx, task.GetDescription(), jobExec)

	return strconv.Itoa(jobID), nil
}

func findRunningAIVisionJob(jobManager *job.Manager, sceneID int) *job.Job {
	expectedDesc := AIVisionJobDescription(sceneID)

	for _, j := range jobManager.GetQueue() {
		if j.Description == expectedDesc && (j.Status == job.StatusReady || j.Status == job.StatusRunning || j.Status == job.StatusStopping) {
			jCopy := j
			return &jCopy
		}
	}

	return nil
}
