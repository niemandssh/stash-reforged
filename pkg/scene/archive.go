package scene

import (
	"context"
	"fmt"
	"strings"

	"github.com/stashapp/stash/pkg/models"
)

// Archive marks a scene as archived, removes heavy generated video caches, and
// optionally deletes its video files from disk.
func (s *Service) Archive(ctx context.Context, scene *models.Scene, deleteFile bool, archiveReason *string, fileDeleter *FileDeleter) error {
	if err := fileDeleter.MarkVideoCacheFiles(scene); err != nil {
		return fmt.Errorf("deleting generated video cache: %w", err)
	}

	if deleteFile {
		if err := s.deleteSceneFilesFromDisk(ctx, scene, fileDeleter); err != nil {
			return fmt.Errorf("deleting scene files from disk: %w", err)
		}
	}

	partial := models.NewScenePartial()
	partial.IsArchived = models.NewOptionalBool(true)
	if archiveReason != nil && strings.TrimSpace(*archiveReason) != "" {
		partial.ArchiveReason = models.NewOptionalString(strings.TrimSpace(*archiveReason))
	}

	if _, err := s.Repository.UpdatePartial(ctx, scene.ID, partial); err != nil {
		return fmt.Errorf("updating scene: %w", err)
	}

	return nil
}
