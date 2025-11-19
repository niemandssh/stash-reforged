package manager

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

var dataURLRegexp = regexp.MustCompile(`^data:(.+?);base64,(.+)$`)

// SaveFilteredScreenshot persists a client-provided screenshot (with filters already applied)
// into the saved_screens directory and schedules a scan so that the new file appears under Images.
func (s *Manager) SaveFilteredScreenshot(ctx context.Context, sceneID string, imageData string, at *float64) error {
	id, err := strconv.Atoi(sceneID)
	if err != nil {
		return fmt.Errorf("converting scene id: %w", err)
	}

	var scene *models.Scene
	if err := s.Repository.WithTxn(ctx, func(ctx context.Context) error {
		var err error
		scene, err = s.Repository.Scene.Find(ctx, id)
		return err
	}); err != nil {
		return err
	}

	if scene == nil {
		return fmt.Errorf("scene with id %d not found", id)
	}

	outputPath, err := s.writeSavedScreenFile(id, imageData, at)
	if err != nil {
		return err
	}

	logger.Infof("Saved filtered screenshot for scene %d to %s", id, outputPath)

	if _, err := s.Scan(ctx, ScanMetadataInput{
		Paths: []string{s.Paths.Generated.SavedScreens},
	}); err != nil {
		return fmt.Errorf("scheduling scan for saved screenshots: %w", err)
	}

	return nil
}

func (s *Manager) writeSavedScreenFile(sceneID int, imageData string, at *float64) (string, error) {
	outputDir := s.Paths.Generated.SavedScreens
	if outputDir == "" {
		return "", fmt.Errorf("saved_screens path is not configured")
	}

	if err := fsutil.EnsureDirAll(outputDir); err != nil {
		return "", fmt.Errorf("ensuring saved_screens directory: %w", err)
	}

	if err := ensureForceGalleryFile(outputDir); err != nil {
		return "", fmt.Errorf("preparing saved_screens gallery marker: %w", err)
	}

	mimeType, rawData, err := decodeDataURLImage(imageData)
	if err != nil {
		return "", err
	}

	ext, err := extensionFromMime(mimeType)
	if err != nil {
		return "", err
	}

	timestamp := time.Now().UTC().Format("20060102_150405")
	var atSuffix string
	if at != nil {
		atSuffix = fmt.Sprintf("_%06dms", int(math.Round(*at*1000)))
	}

	fileName := fmt.Sprintf("scene_%d_%s%s.%s", sceneID, timestamp, atSuffix, ext)
	outputPath := filepath.Join(outputDir, fileName)

	if err := os.WriteFile(outputPath, rawData, 0o664); err != nil {
		return "", fmt.Errorf("writing screenshot file: %w", err)
	}

	return outputPath, nil
}

func decodeDataURLImage(image string) (string, []byte, error) {
	matches := dataURLRegexp.FindStringSubmatch(strings.TrimSpace(image))
	if len(matches) != 3 {
		return "", nil, fmt.Errorf("invalid image data provided")
	}

	mimeType := strings.ToLower(matches[1])
	data, err := utils.GetDataFromBase64String(matches[2])
	if err != nil {
		return "", nil, fmt.Errorf("decoding screenshot payload: %w", err)
	}

	return mimeType, data, nil
}

func extensionFromMime(mime string) (string, error) {
	switch mime {
	case "image/png":
		return "png", nil
	case "image/jpeg", "image/jpg":
		return "jpg", nil
	case "image/webp":
		return "webp", nil
	default:
		return "", fmt.Errorf("unsupported image mime type: %s", mime)
	}
}

func ensureForceGalleryFile(dir string) error {
	forceGalleryPath := filepath.Join(dir, ".forcegallery")
	exists, err := fsutil.FileExists(forceGalleryPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if exists {
		return nil
	}

	return os.WriteFile(forceGalleryPath, []byte{}, 0o664)
}
