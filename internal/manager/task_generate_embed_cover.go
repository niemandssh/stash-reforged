package manager

import (
	"context"
	"fmt"
	"strings"

	"github.com/stashapp/stash/pkg/embed"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scene/generate"
)

func (t *GenerateCoverTask) generateEmbedCover(ctx context.Context) {
	if t.Scene.EmbedURL == nil || strings.TrimSpace(*t.Scene.EmbedURL) == "" {
		return
	}

	embedURL := strings.TrimSpace(*t.Scene.EmbedURL)
	if !embed.SupportsSpriteGeneration(embedURL) {
		logger.Warnf("embed cover generation not supported for scene %d", t.Scene.ID)
		return
	}

	duration, err := ResolveEmbedDuration(embedURL)
	if err != nil {
		logger.Errorf("error extracting embed duration for cover: %v", err)
		logErrorOutput(err)
		return
	}
	if duration <= 0 {
		logger.Errorf("invalid embed duration for scene %d", t.Scene.ID)
		return
	}

	// Re-extract after duration probing so temporary CDN tokens are still fresh.
	streamURL, _, err := embed.ExtractStreamURL(embedURL)
	if err != nil {
		logger.Errorf("error extracting embed stream for cover: %v", err)
		logErrorOutput(err)
		return
	}

	at := duration * 0.2
	if t.ScreenshotAt != nil {
		at = *t.ScreenshotAt
	}

	logger.Debugf("Creating embed screenshot for scene %d at %.2fs", t.Scene.ID, at)

	g := generate.Generator{
		Encoder:      instance.FFMpeg,
		FFMpegConfig: instance.Config,
		LockManager:  instance.ReadLockManager,
		ScenePaths:   instance.Paths.Scene,
		Overwrite:    true,
	}

	coverImageData, err := g.ScreenshotHybridHTTP(
		context.TODO(),
		streamURL,
		duration,
		generate.ScreenshotOptions{At: &at},
		embed.StreamHTTPHeadersForMediaURL(embedURL, streamURL),
	)
	if err != nil {
		logger.Errorf("Error generating embed screenshot: %v", err)
		logErrorOutput(err)
		return
	}

	if err := t.repository.WithTxn(ctx, func(ctx context.Context) error {
		qb := t.repository.Scene
		scenePartial := models.NewScenePartial()

		if err := qb.UpdateCover(ctx, t.Scene.ID, coverImageData); err != nil {
			return fmt.Errorf("error setting embed screenshot: %v", err)
		}

		if _, err := qb.UpdatePartial(ctx, t.Scene.ID, scenePartial); err != nil {
			return fmt.Errorf("error updating scene: %v", err)
		}

		return nil
	}); err != nil && ctx.Err() == nil {
		logger.Error(err.Error())
	}
}
