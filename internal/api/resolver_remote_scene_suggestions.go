package api

import (
	"context"
	"strconv"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/models"
)

func (r *mutationResolver) StartRemoteSceneSuggestions(ctx context.Context, input manager.RemoteSceneSuggestionsInput) (string, error) {
	jobID := manager.GetInstance().StartRemoteSceneSuggestions(ctx, input)
	return strconv.Itoa(jobID), nil
}

func (r *queryResolver) RemoteSceneSuggestionResults(ctx context.Context, jobID string) ([]*models.ScrapedScene, error) {
	id, err := strconv.Atoi(jobID)
	if err != nil {
		return nil, err
	}

	scenes := manager.TakeRemoteSceneSuggestionResults(id)
	if scenes == nil {
		return []*models.ScrapedScene{}, nil
	}
	return scenes, nil
}
