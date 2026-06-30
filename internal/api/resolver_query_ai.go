package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/stashapp/stash/pkg/ai"
)

func (r *queryResolver) FillSceneWithAi(ctx context.Context, sceneID string) (*AIFilledSceneResult, error) {
	id, err := strconv.Atoi(sceneID)
	if err != nil {
		return nil, fmt.Errorf("%w: scene_id is not an integer: '%s'", ErrInput, sceneID)
	}

	result, err := ai.FillScene(ctx, r.repository, id)
	if err != nil {
		return nil, err
	}

	return marshalAIFilledSceneResult(result), nil
}

func marshalAIFilledSceneResult(result *ai.FilledSceneResult) *AIFilledSceneResult {
	performerTags := make([]*AIPerformerTags, 0, len(result.PerformerTags))
	for _, pt := range result.PerformerTags {
		performerTags = append(performerTags, &AIPerformerTags{
			PerformerName: pt.PerformerName,
			TagNames:      pt.Tags,
		})
	}

	return &AIFilledSceneResult{
		Scene:         result.Scene,
		ShootDate:     result.ShootDate,
		PerformerTags: performerTags,
	}
}
