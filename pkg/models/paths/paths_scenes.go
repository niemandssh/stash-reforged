package paths

import (
	"fmt"
	"path/filepath"

	"github.com/stashapp/stash/pkg/fsutil"
)

const AIVisionPanelCount = 5

type scenePaths struct {
	generatedPaths
}

func newScenePaths(p Paths) *scenePaths {
	sp := scenePaths{
		generatedPaths: *p.Generated,
	}
	return &sp
}

func (sp *scenePaths) GetLegacyScreenshotPath(checksum string) string {
	return filepath.Join(sp.Screenshots, checksum+".jpg")
}

func (sp *scenePaths) GetTranscodePath(checksum string) string {
	return filepath.Join(sp.Transcodes, checksum+".mp4")
}

func (sp *scenePaths) GetStreamPath(scenePath string, checksum string) string {
	transcodePath := sp.GetTranscodePath(checksum)
	transcodeExists, _ := fsutil.FileExists(transcodePath)
	if transcodeExists {
		return transcodePath
	}
	return scenePath
}

func (sp *scenePaths) GetVideoPreviewPath(checksum string) string {
	return filepath.Join(sp.Screenshots, checksum+".mp4")
}

func (sp *scenePaths) GetWebpPreviewPath(checksum string) string {
	return filepath.Join(sp.Screenshots, checksum+".webp")
}

func (sp *scenePaths) GetSpriteImageFilePath(checksum string) string {
	return filepath.Join(sp.Vtt, checksum+"_sprite.jpg")
}

func (sp *scenePaths) GetSpriteVttFilePath(checksum string) string {
	return filepath.Join(sp.Vtt, checksum+"_thumbs.vtt")
}

func (sp *scenePaths) GetAIVisionPanelFilePath(checksum string, index int) string {
	return filepath.Join(sp.Vtt, fmt.Sprintf("%s_ai_vision_%d.jpg", checksum, index))
}

func (sp *scenePaths) GetAIVisionPanelFilePaths(checksum string) []string {
	paths := make([]string, AIVisionPanelCount)
	for i := 0; i < AIVisionPanelCount; i++ {
		paths[i] = sp.GetAIVisionPanelFilePath(checksum, i)
	}
	return paths
}

func (sp *scenePaths) AIVisionPanelsExist(checksum string) bool {
	if checksum == "" {
		return false
	}

	for i := 0; i < AIVisionPanelCount; i++ {
		exists, _ := fsutil.FileExists(sp.GetAIVisionPanelFilePath(checksum, i))
		if !exists {
			return false
		}
	}

	return true
}

func (sp *scenePaths) GetInteractiveHeatmapPath(checksum string) string {
	return filepath.Join(sp.InteractiveHeatmap, checksum+".png")
}
