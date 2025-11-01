package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

func (r *sceneCreateInputResolver) ScenePerformers(ctx context.Context, obj *models.SceneCreateInput, data []*ScenePerformerInput) error {
	if data == nil {
		obj.ScenePerformers = nil
		return nil
	}

	// Convert ScenePerformerInput to PerformerScenes
	scenePerformers := make([]models.PerformerScenes, len(data))
	for i, sp := range data {
		performerID, err := strconv.Atoi(sp.PerformerID)
		if err != nil {
			return fmt.Errorf("invalid performer ID: %s", sp.PerformerID)
		}

		scenePerformers[i] = models.PerformerScenes{
			PerformerID:     performerID,
			SmallRole:       sp.SmallRole,
			RoleDescription: sp.RoleDescription,
		}
	}

	obj.ScenePerformers = scenePerformers
	return nil
}
