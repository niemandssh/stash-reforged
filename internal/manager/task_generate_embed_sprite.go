package manager

import (
	"context"
	"fmt"
	"image"
	"strings"

	"github.com/disintegration/imaging"

	"github.com/stashapp/stash/pkg/embed"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scene/generate"
)

const embedSpriteRows = 9
const embedSpriteCols = 9

type GenerateEmbedSpriteTask struct {
	Scene     models.Scene
	Overwrite bool
}

func (t *GenerateEmbedSpriteTask) GetDescription() string {
	return fmt.Sprintf("Generating embed sprites for scene %d", t.Scene.ID)
}

func (t *GenerateEmbedSpriteTask) Start(ctx context.Context) {
	if err := t.Generate(ctx); err != nil {
		logger.Errorf("error generating embed sprites for scene %d: %v", t.Scene.ID, err)
		logErrorOutput(err)
	}
}

func (t *GenerateEmbedSpriteTask) Generate(ctx context.Context) error {
	defer clearEmbedSpriteJob(t.Scene.ID)

	if t.Scene.EmbedURL == nil || strings.TrimSpace(*t.Scene.EmbedURL) == "" {
		return fmt.Errorf("scene has no embed URL")
	}

	if !embed.SupportsSpriteGeneration(*t.Scene.EmbedURL) {
		return embed.ErrStreamNotSupported
	}

	embedURL := strings.TrimSpace(*t.Scene.EmbedURL)

	duration, err := ResolveEmbedDuration(embedURL)
	if err != nil {
		return err
	}
	if duration <= 0 {
		return fmt.Errorf("invalid embed video duration")
	}

	sceneHash := ResolveRemoteSceneSpriteHash(t.Scene.ID)
	imagePath := instance.Paths.Scene.GetSpriteImageFilePath(sceneHash)
	vttPath := instance.Paths.Scene.GetSpriteVttFilePath(sceneHash)

	if !t.Overwrite {
		imageExists, _ := fsutil.FileExists(imagePath)
		vttExists, _ := fsutil.FileExists(vttPath)
		if imageExists && vttExists {
			return nil
		}
	}

	// Resolve a fresh media URL after duration probing. Temporary PlayerJS links are
	// downloaded once so auth tokens cannot expire across many ffmpeg seeks.
	input, headers, cleanup, err := prepareEmbedSpriteSource(ctx, embedURL)
	if err != nil {
		return err
	}
	defer cleanup()

	chunkCount := embedSpriteRows * embedSpriteCols
	stepSize := duration / float64(chunkCount)

	generator := &generate.Generator{
		Encoder:      instance.FFMpeg,
		FFMpegConfig: instance.Config,
		LockManager:  instance.ReadLockManager,
		ScenePaths:   instance.Paths.Scene,
	}

	var images []image.Image
	for i := 0; i < chunkCount; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		time := float64(i) * stepSize
		var img image.Image
		if headers != "" {
			img, err = generator.SpriteScreenshotHybridHTTP(ctx, input, time, headers)
		} else {
			img, err = generator.SpriteScreenshotHybrid(ctx, input, time)
		}
		if err != nil {
			return fmt.Errorf("embed sprite screenshot at %.2fs: %w", time, err)
		}
		images = append(images, img)
	}

	if len(images) == 0 {
		return fmt.Errorf("no embed sprite images generated")
	}

	if err := imaging.Save(generator.CombineSpriteImages(images), imagePath); err != nil {
		return fmt.Errorf("saving embed sprite image: %w", err)
	}

	if err := generator.SpriteVTT(ctx, vttPath, imagePath, stepSize, "sprite"); err != nil {
		return fmt.Errorf("generating embed sprite vtt: %w", err)
	}

	logger.Infof("generated embed sprites for scene %d", t.Scene.ID)
	return nil
}

func IsEmbedSpriteScene(scene *models.Scene) bool {
	if scene == nil {
		return false
	}
	if scene.Path != "" {
		return false
	}
	if scene.PrimaryFileID != nil {
		return false
	}
	if scene.Files.PrimaryLoaded() {
		if scene.Files.Primary() != nil {
			return false
		}
	} else if scene.Files.Loaded() && len(scene.Files.List()) > 0 {
		return false
	}
	if scene.EmbedURL == nil || strings.TrimSpace(*scene.EmbedURL) == "" {
		return false
	}
	return embed.SupportsSpriteGeneration(*scene.EmbedURL)
}

func SceneSpriteHash(scene *models.Scene, fileNamingAlgorithm models.HashAlgorithm) string {
	if hash := scene.GetHash(fileNamingAlgorithm); hash != "" {
		return hash
	}
	if IsEmbedSpriteScene(scene) {
		return ResolveRemoteSceneSpriteHash(scene.ID)
	}
	return ""
}

// ResolveRemoteSceneSpriteHash returns the on-disk sprite key for a remote scene,
// preferring the current naming scheme and falling back to legacy files.
func ResolveRemoteSceneSpriteHash(sceneID int) string {
	current := models.RemoteSceneSpriteHash(sceneID)
	if remoteSpriteFilesExist(current) {
		return current
	}

	legacy := models.LegacyRemoteSceneSpriteHash(sceneID)
	if remoteSpriteFilesExist(legacy) {
		return legacy
	}

	return current
}

func remoteSpriteFilesExist(hash string) bool {
	if hash == "" {
		return false
	}

	imageExists, _ := fsutil.FileExists(instance.Paths.Scene.GetSpriteImageFilePath(hash))
	vttExists, _ := fsutil.FileExists(instance.Paths.Scene.GetSpriteVttFilePath(hash))
	return imageExists || vttExists
}
