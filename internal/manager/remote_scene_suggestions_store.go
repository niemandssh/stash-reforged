package manager

import (
	"sync"
	"time"

	"github.com/stashapp/stash/pkg/models"
)

const remoteSceneSuggestionsResultTTL = 10 * time.Minute

type remoteSceneSuggestionsResult struct {
	Scenes    []*models.ScrapedScene
	CreatedAt time.Time
}

var remoteSceneSuggestionsResults sync.Map // map[int]remoteSceneSuggestionsResult

func storeRemoteSceneSuggestionResults(jobID int, scenes []*models.ScrapedScene) {
	if scenes == nil {
		scenes = []*models.ScrapedScene{}
	}
	remoteSceneSuggestionsResults.Store(jobID, remoteSceneSuggestionsResult{
		Scenes:    scenes,
		CreatedAt: time.Now(),
	})
	go cleanupExpiredRemoteSceneSuggestionResults()
}

// TakeRemoteSceneSuggestionResults returns and removes stored results for a job.
// Returns nil when results are not yet available.
func TakeRemoteSceneSuggestionResults(jobID int) []*models.ScrapedScene {
	cleanupExpiredRemoteSceneSuggestionResults()

	v, ok := remoteSceneSuggestionsResults.LoadAndDelete(jobID)
	if !ok {
		return nil
	}

	result, ok := v.(remoteSceneSuggestionsResult)
	if !ok {
		return nil
	}

	return result.Scenes
}

// PeekRemoteSceneSuggestionResults returns stored results without removing them.
func PeekRemoteSceneSuggestionResults(jobID int) []*models.ScrapedScene {
	cleanupExpiredRemoteSceneSuggestionResults()

	v, ok := remoteSceneSuggestionsResults.Load(jobID)
	if !ok {
		return nil
	}

	result, ok := v.(remoteSceneSuggestionsResult)
	if !ok {
		return nil
	}

	return result.Scenes
}

func cleanupExpiredRemoteSceneSuggestionResults() {
	now := time.Now()
	remoteSceneSuggestionsResults.Range(func(key, value any) bool {
		result, ok := value.(remoteSceneSuggestionsResult)
		if !ok || now.Sub(result.CreatedAt) > remoteSceneSuggestionsResultTTL {
			remoteSceneSuggestionsResults.Delete(key)
		}
		return true
	})
}
