package scene

import (
	"os"
	"strings"

	"github.com/stashapp/stash/pkg/embed"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/paths"
)

// MarkSpriteFiles marks sprite image and VTT for deletion for the given content hash.
func (d *FileDeleter) MarkSpriteFiles(hash string) error {
	if hash == "" {
		return nil
	}

	var files []string

	spritePath := d.Paths.Scene.GetSpriteImageFilePath(hash)
	exists, _ := fsutil.FileExists(spritePath)
	if exists {
		files = append(files, spritePath)
	}

	vttPath := d.Paths.Scene.GetSpriteVttFilePath(hash)
	exists, _ = fsutil.FileExists(vttPath)
	if exists {
		files = append(files, vttPath)
	}

	if len(files) == 0 {
		return nil
	}

	return d.Files(files)
}

func spriteHashForScene(scene *models.Scene, algo models.HashAlgorithm) string {
	return scene.GetHash(algo)
}

func shouldPreserveSpritesForEmbed(scene *models.Scene) bool {
	if scene == nil || len(scene.Files.List()) != 1 {
		return false
	}
	if scene.EmbedURL == nil || strings.TrimSpace(*scene.EmbedURL) == "" {
		return false
	}
	return embed.SupportsSpriteGeneration(strings.TrimSpace(*scene.EmbedURL))
}

func preserveSpriteFiles(paths *paths.Paths, fromHash, toHash string) error {
	if fromHash == "" || toHash == "" || fromHash == toHash {
		return nil
	}

	dstImage := paths.Scene.GetSpriteImageFilePath(toHash)
	dstVtt := paths.Scene.GetSpriteVttFilePath(toHash)

	dstImageExists, _ := fsutil.FileExists(dstImage)
	dstVttExists, _ := fsutil.FileExists(dstVtt)
	if dstImageExists && dstVttExists {
		return nil
	}

	srcImage := paths.Scene.GetSpriteImageFilePath(fromHash)
	srcVtt := paths.Scene.GetSpriteVttFilePath(fromHash)

	srcImageExists, _ := fsutil.FileExists(srcImage)
	if !srcImageExists {
		return nil
	}

	if !dstImageExists {
		if err := copyFile(srcImage, dstImage); err != nil {
			return err
		}
	}

	srcVttExists, _ := fsutil.FileExists(srcVtt)
	if !srcVttExists || dstVttExists {
		return nil
	}

	vttData, err := os.ReadFile(srcVtt)
	if err != nil {
		return err
	}

	oldSpriteName := fromHash + "_sprite.jpg"
	newSpriteName := toHash + "_sprite.jpg"
	vttContent := strings.ReplaceAll(string(vttData), oldSpriteName, newSpriteName)

	return os.WriteFile(dstVtt, []byte(vttContent), 0o644)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
