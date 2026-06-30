package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/stashapp/stash/pkg/fsutil"
)

func (c *VisionClient) DescribeSceneVision(ctx context.Context, panelPaths []string) (string, error) {
	panels, err := loadAIVisionPanels(panelPaths)
	if err != nil {
		return "", err
	}
	if len(panels) == 0 {
		return "", fmt.Errorf("no vision preview panels available")
	}

	return c.describeVisionPanels(ctx, panels)
}

func loadAIVisionPanels(panelPaths []string) ([][]byte, error) {
	if len(panelPaths) == 0 {
		return nil, fmt.Errorf("no AI vision panel paths provided")
	}

	var panels [][]byte
	for _, path := range panelPaths {
		exists, err := fsutil.FileExists(path)
		if err != nil || !exists {
			return nil, fmt.Errorf("AI vision panel not found: %s", path)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading AI vision panel %s: %w", path, err)
		}
		panels = append(panels, data)
	}

	return panels, nil
}
