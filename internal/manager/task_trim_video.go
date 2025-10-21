package manager

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/hash/md5"
	"github.com/stashapp/stash/pkg/hash/oshash"
	"github.com/stashapp/stash/pkg/hash/videophash"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/paths"
	"github.com/stashapp/stash/pkg/scene/generate"
)

// trimFileOpener implements file.Opener for OS files
type trimFileOpener struct {
	path string
}

func (o *trimFileOpener) Open() (io.ReadCloser, error) {
	return os.Open(o.path)
}

type TrimVideoTask struct {
	Scene                 models.Scene
	FileID                models.FileID // Конкретный файл для обрезки
	StartTime             *float64      // Время начала обрезки в секундах (nil = не установлено)
	EndTime               *float64      // Время окончания обрезки в секундах (nil = не установлено)
	FileNamingAlgorithm   models.HashAlgorithm
	G                     *generate.Generator
	FFMpeg                *ffmpeg.FFMpeg
	FFProbe               *ffmpeg.FFProbe
	Config                *config.Config
	Paths                 *paths.Paths
	Repository            models.Repository
	FingerprintCalculator interface {
		CalculateFingerprints(f *models.BaseFile, o file.Opener, useExisting bool) ([]models.Fingerprint, error)
	}
}

func (t *TrimVideoTask) GetDescription() string {
	startStr := "beginning"
	if t.StartTime != nil {
		startStr = fmt.Sprintf("%.2fs", *t.StartTime)
	}
	endStr := "end"
	if t.EndTime != nil {
		endStr = fmt.Sprintf("%.2fs", *t.EndTime)
	}
	return fmt.Sprintf("Trimming video %s from %s to %s", t.Scene.Path, startStr, endStr)
}

func (t *TrimVideoTask) Execute(ctx context.Context, progress *job.Progress) error {
	// Find specific file
	var targetFile *models.VideoFile
	for _, vf := range t.Scene.Files.List() {
		if vf.ID == t.FileID {
			targetFile = vf
			break
		}
	}

	if targetFile == nil {
		return fmt.Errorf("file with ID %d not found in scene", t.FileID)
	}

	// Validate trim times
	if t.StartTime != nil && *t.StartTime < 0 {
		return fmt.Errorf("start time cannot be negative: %.2f", *t.StartTime)
	}
	if t.EndTime != nil && t.StartTime != nil && *t.EndTime <= *t.StartTime {
		return fmt.Errorf("end time %.2f must be greater than start time %.2f", *t.EndTime, *t.StartTime)
	}
	if t.EndTime != nil && *t.EndTime > targetFile.Duration {
		return fmt.Errorf("end time %.2f cannot be greater than video duration %.2f", *t.EndTime, targetFile.Duration)
	}

	startStr := "beginning"
	if t.StartTime != nil {
		startStr = fmt.Sprintf("%.2fs", *t.StartTime)
	}
	endStr := "end"
	if t.EndTime != nil {
		endStr = fmt.Sprintf("%.2fs", *t.EndTime)
	}
	logger.Infof("[trim-video] trimming video of scene %d from %s to %s (duration: %.2fs)",
		t.Scene.ID, startStr, endStr, targetFile.Duration)

	progress.SetTotal(3)
	progress.SetProcessed(0)

	var conversionErr error

	// Get original file size for display
	originalFileInfo, err := os.Stat(targetFile.Path)
	if err == nil {
		logger.Infof("[trim-video] original file size: %d bytes (%.2f MB)", originalFileInfo.Size(), float64(originalFileInfo.Size())/1024/1024)
	}

	// Start file size monitoring
	originalSize := int64(0)
	if originalFileInfo != nil {
		originalSize = originalFileInfo.Size()
	}

	// Start monitoring file size in a goroutine
	done := make(chan bool)
	startVal := 0.0
	if t.StartTime != nil {
		startVal = *t.StartTime
	}
	endVal := 0.0
	if t.EndTime != nil {
		endVal = *t.EndTime
	}
	tempFile := filepath.Join(t.Config.GetGeneratedPath(), fmt.Sprintf("trim_video_%d_%s_%.2f_%.2f.mp4",
		t.Scene.ID, t.Scene.GetHash(t.FileNamingAlgorithm), startVal, endVal))
	go t.monitorFileSize(tempFile, originalSize, progress, done)

	// Create a task queue for dynamic status updates
	taskQueue := job.NewTaskQueue(ctx, progress, 100, 1)
	go t.monitorFileSizeWithQueue(tempFile, originalSize, taskQueue, done)

	// Perform conversion without transaction to avoid blocking
	conversionErr = t.trimVideo(ctx, targetFile, progress)
	if conversionErr != nil {
		logger.Errorf("[trim-video] error trimming video of scene %d: %v", t.Scene.ID, conversionErr)
		// Close task queue on error
		taskQueue.Close()
		close(done)
		return conversionErr
	}
	progress.SetProcessed(1)

	// Close task queue
	taskQueue.Close()

	// Stop monitoring
	close(done)

	progress.ExecuteTask("Updating scene metadata", func() {
		if conversionErr == nil {
			progress.SetProcessed(2)
		}
	})

	progress.ExecuteTask("Finalizing trim", func() {
		if conversionErr == nil {
			progress.SetProcessed(3)
		}
	})

	if conversionErr == nil {
		logger.Infof("[trim-video] successfully trimmed video of scene %d", t.Scene.ID)
	} else {
		return conversionErr
	}

	return nil
}

// For backward compatibility
func (t *TrimVideoTask) Start(ctx context.Context) {
	progress := &job.Progress{}
	t.Execute(ctx, progress)
}

func (t *TrimVideoTask) trimVideo(ctx context.Context, f *models.VideoFile, progress *job.Progress) error {
	// Save old hash BEFORE conversion for sprite migration
	oldHash := t.Scene.GetHash(t.FileNamingAlgorithm)
	logger.Infof("[trim-video] old scene hash before trim: %s", oldHash)

	tempDir := t.Config.GetGeneratedPath()
	startVal := 0.0
	if t.StartTime != nil {
		startVal = *t.StartTime
	}
	endVal := 0.0
	if t.EndTime != nil {
		endVal = *t.EndTime
	}
	tempFile := filepath.Join(tempDir, fmt.Sprintf("trim_video_%d_%s_%.2f_%.2f.mp4",
		t.Scene.ID, oldHash, startVal, endVal))

	// Create independent backup copy in temp directory
	backupTempDir := t.Config.GetTempPath()
	logger.Infof("[trim-video] Creating backup temp directory: %s", backupTempDir)
	if err := os.MkdirAll(backupTempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp backup directory %s: %w", backupTempDir, err)
	}
	// Use original filename for backup in temp
	originalFilename := filepath.Base(f.Path)
	backupTempFile := filepath.Join(backupTempDir, originalFilename)
	logger.Infof("[trim-video] Backup temp file path: %s", backupTempFile)

	// Create backup copy of ORIGINAL file in temp directory BEFORE conversion
	logger.Infof("[trim-video] Creating backup copy of original file from %s to %s", f.Path, backupTempFile)
	if err := t.copyFileContent(f.Path, backupTempFile); err != nil {
		return fmt.Errorf("failed to create backup copy of original file in temp: %w", err)
	}
	logger.Infof("[trim-video] Successfully created backup copy of original file in temp: %s", backupTempFile)

	// Get original file size for progress tracking
	originalFileInfo, err := os.Stat(f.Path)
	if err != nil {
		logger.Warnf("[trim-video] failed to get original file size: %v", err)
	} else {
		logger.Infof("[trim-video] original file size: %d bytes (%.2f MB)", originalFileInfo.Size(), float64(originalFileInfo.Size())/1024/1024)
	}

	// Track if conversion was successful
	conversionSuccessful := false

	// Clean up temp files at the end
	defer func() {
		// Clean up main temp file only on failure
		if !conversionSuccessful {
			if _, err := os.Stat(tempFile); err == nil {
				if err := os.Remove(tempFile); err != nil {
					logger.Warnf("[trim-video] failed to remove temp file %s: %v", tempFile, err)
				} else {
					logger.Infof("[trim-video] cleaned up temp file: %s", tempFile)
				}
			}
		}
	}()

	if err := t.performTrimWithProgress(ctx, f.Path, tempFile, progress); err != nil {
		logger.Errorf("[trim-video] trim failed: %v", err)
		return fmt.Errorf("trim failed: %w", err)
	}

	if err := t.validateTrimmedFile(tempFile); err != nil {
		return fmt.Errorf("trimmed file validation failed: %w", err)
	}

	// Create new video file in separate transaction
	var newFile *models.VideoFile
	var isUpdated bool
	if err := t.Repository.WithTxn(ctx, func(ctx context.Context) error {
		var err error
		newFile, isUpdated, err = t.createNewVideoFile(ctx, tempFile)
		return err
	}); err != nil {
		return fmt.Errorf("failed to create new video file: %w", err)
	}

	if err := t.updateSceneWithNewFile(ctx, newFile); err != nil {
		return fmt.Errorf("failed to update scene with new file: %w", err)
	}

	if isUpdated {
		// File was updated, check if we need to copy temp file to existing file
		finalPath := newFile.Base().Path
		logger.Infof("[trim-video] checking if temp file needs to be copied to existing file: %s", finalPath)

		// Only copy if paths are different (avoid copying file to itself)
		if tempFile != finalPath {
			logger.Infof("[trim-video] copying temp file content to existing file: %s -> %s", tempFile, finalPath)
			if err := t.copyFileContent(tempFile, finalPath); err != nil {
				return fmt.Errorf("failed to copy temp file content to existing file: %w", err)
			}
		} else {
			logger.Infof("[trim-video] temp file and final path are the same, no copy needed: %s", finalPath)
		}

		// Validate the updated file
		if err := t.validateTrimmedFile(finalPath); err != nil {
			logger.Errorf("[trim-video] updated file validation failed: %v", err)
			return fmt.Errorf("updated file validation failed: %w", err)
		}

		logger.Infof("[trim-video] successfully updated existing file: %s", finalPath)
	} else {
		// New file was created, move temp file to final location
		finalPath := t.getFinalPath(newFile)
		logger.Infof("[trim-video] moving file from %s to %s", tempFile, finalPath)

		// Check if temp file exists
		if _, err := os.Stat(tempFile); err != nil {
			return fmt.Errorf("temp file does not exist: %w", err)
		}

		// Copy temp file to final location (works across different filesystems)
		logger.Infof("[trim-video] copying temp file to final location: %s -> %s", tempFile, finalPath)
		if err := t.copyFileContent(tempFile, finalPath); err != nil {
			return fmt.Errorf("failed to copy trimmed file to final location: %w", err)
		}

		// Remove temp file after successful copy
		if err := os.Remove(tempFile); err != nil {
			logger.Warnf("[trim-video] failed to remove temp file %s: %v", tempFile, err)
		} else {
			logger.Infof("[trim-video] removed temp file: %s", tempFile)
		}

		// Verify the file was moved successfully
		if _, err := os.Stat(finalPath); err != nil {
			return fmt.Errorf("final file does not exist after move: %w", err)
		}

		logger.Infof("[trim-video] successfully moved file to %s", finalPath)

		if err := t.updateFilePath(ctx, newFile, finalPath); err != nil {
			return fmt.Errorf("failed to update file path: %w", err)
		}

		// Validate the trimmed file before removing the original
		if err := t.validateTrimmedFile(finalPath); err != nil {
			logger.Errorf("[trim-video] trimmed file validation failed, keeping original: %v", err)
			return fmt.Errorf("trimmed file validation failed: %w", err)
		}

		// Remove the original file only after successful validation
		originalPath := f.Path
		if err := os.Remove(originalPath); err != nil {
			logger.Warnf("[trim-video] failed to remove original file %s: %v", originalPath, err)
		} else {
			logger.Infof("[trim-video] removed original file: %s", originalPath)
		}

		// Delete the old file record from database
		if err := t.Repository.WithTxn(ctx, func(ctx context.Context) error {
			return t.deleteOldFileRecord(ctx, f)
		}); err != nil {
			logger.Warnf("[trim-video] failed to delete old file record: %v", err)
		} else {
			logger.Infof("[trim-video] deleted old file record from database")
		}
	}

	// Recalculate hashes for the new file
	var finalPath string
	if isUpdated {
		finalPath = newFile.Base().Path
	} else {
		finalPath = t.getFinalPath(newFile)
	}

	if err := t.recalculateFileHashes(ctx, newFile, finalPath); err != nil {
		logger.Warnf("[trim-video] failed to recalculate file hashes: %v", err)
	} else {
		logger.Infof("[trim-video] recalculated file hashes")
	}

	// Force recalculation of file hashes after trim (content has changed)
	logger.Infof("[trim-video] forcing recalculation of file hashes after trim")
	if err := t.Repository.WithTxn(ctx, func(ctx context.Context) error {
		// Get the updated scene
		updatedScene, err := t.Repository.Scene.Find(ctx, t.Scene.ID)
		if err != nil {
			return fmt.Errorf("failed to find updated scene: %w", err)
		}

		if updatedScene != nil {
			// Load scene files first
			if err := updatedScene.LoadFiles(ctx, t.Repository.Scene); err != nil {
				logger.Warnf("[trim-video] failed to load scene files: %v", err)
			} else {
				// Force update of all video files to trigger hash recalculation
				for _, vf := range updatedScene.Files.List() {
					videoFile := vf
					// Clear fingerprints to force recalculation (content has changed)
					videoFile.Base().Fingerprints = nil
					if err := t.Repository.File.Update(ctx, videoFile); err != nil {
						logger.Warnf("[trim-video] failed to update file fingerprints for file %d: %v", videoFile.ID, err)
					}
				}
			}
		}
		return nil
	}); err != nil {
		logger.Warnf("[trim-video] failed to recalculate file hashes: %v", err)
	}

	// Force generation of OSHash and Checksum for trimmed video
	logger.Infof("[trim-video] forcing generation of OSHash and Checksum for trimmed video")
	if err := t.Repository.WithTxn(ctx, func(ctx context.Context) error {
		// Get the updated scene
		updatedScene, err := t.Repository.Scene.Find(ctx, t.Scene.ID)
		if err != nil {
			return fmt.Errorf("failed to find updated scene: %w", err)
		}

		if updatedScene != nil {
			// Load scene files
			if err := updatedScene.LoadFiles(ctx, t.Repository.Scene); err != nil {
				return fmt.Errorf("failed to load scene files: %w", err)
			}

			// Generate OSHash and Checksum for each video file
			for _, vf := range updatedScene.Files.List() {
				videoFile := vf
				filePath := videoFile.Base().Path

				logger.Infof("[trim-video] generating hashes for file %d: %s", videoFile.ID, filePath)

				// Generate OSHash
				if oshash, err := oshash.FromFilePath(filePath); err == nil {
					// Add OSHash fingerprint
					osHashFingerprint := models.Fingerprint{
						Type:        models.FingerprintTypeOshash,
						Fingerprint: oshash,
					}
					videoFile.Base().Fingerprints = append(videoFile.Base().Fingerprints, osHashFingerprint)
					logger.Infof("[trim-video] generated OSHash for file %d: %s", videoFile.ID, oshash)
				} else {
					logger.Warnf("[trim-video] failed to generate OSHash for file %d: %v", videoFile.ID, err)
				}

				// Generate MD5 Checksum
				if checksum, err := md5.FromFilePath(filePath); err == nil {
					// Add MD5 fingerprint
					md5Fingerprint := models.Fingerprint{
						Type:        models.FingerprintTypeMD5,
						Fingerprint: checksum,
					}
					videoFile.Base().Fingerprints = append(videoFile.Base().Fingerprints, md5Fingerprint)
					logger.Infof("[trim-video] generated Checksum for file %d: %s", videoFile.ID, checksum)
				} else {
					logger.Warnf("[trim-video] failed to generate Checksum for file %d: %v", videoFile.ID, err)
				}

				// Update the file in database
				if err := t.Repository.File.Update(ctx, videoFile); err != nil {
					logger.Warnf("[trim-video] failed to update file %d with new fingerprints: %v", videoFile.ID, err)
				} else {
					logger.Infof("[trim-video] updated file %d with new fingerprints", videoFile.ID)
				}
			}
		}
		return nil
	}); err != nil {
		logger.Warnf("[trim-video] failed to generate hashes for trimmed video: %v", err)
	}

	// Wait a moment for hash recalculation to complete
	logger.Infof("[trim-video] waiting for hash recalculation to complete")
	time.Sleep(2 * time.Second)

	// Regenerate sprites with new hash after trim (oldHash saved at start of function)
	logger.Infof("[trim-video] regenerating sprites for trimmed file")
	if err := t.Repository.WithTxn(ctx, func(ctx context.Context) error {
		return t.regenerateSprites(ctx, oldHash)
	}); err != nil {
		logger.Warnf("[trim-video] failed to regenerate sprites: %v", err)
		// Don't fail the conversion if sprite generation fails
	}

	// Generate VTT file for the new video if it doesn't exist
	if err := t.Repository.WithTxn(ctx, func(ctx context.Context) error {
		return t.generateVTTFile(ctx, newFile, finalPath)
	}); err != nil {
		logger.Warnf("[trim-video] failed to generate VTT file: %v", err)
	} else {
		logger.Infof("[trim-video] generated VTT file")
	}

	// Clear start_time and end_time from scene after successful trim
	if err := t.clearTrimTimes(ctx); err != nil {
		logger.Warnf("[trim-video] failed to clear trim times: %v", err)
	} else {
		logger.Infof("[trim-video] cleared start_time and end_time from scene")
	}

	// Clean up backup temp file only after all operations are successful
	if _, err := os.Stat(backupTempFile); err == nil {
		if err := os.Remove(backupTempFile); err != nil {
			logger.Warnf("[trim-video] failed to remove backup temp file %s: %v", backupTempFile, err)
		} else {
			logger.Infof("[trim-video] cleaned up backup temp file: %s", backupTempFile)
		}
	}

	// Mark conversion as successful
	conversionSuccessful = true

	// Force cleanup of temp file regardless of success/failure
	if _, err := os.Stat(tempFile); err == nil {
		if err := os.Remove(tempFile); err != nil {
			logger.Warnf("[trim-video] failed to remove temp file %s: %v", tempFile, err)
		} else {
			logger.Infof("[trim-video] force cleaned up temp file: %s", tempFile)
		}
	}

	return nil
}

func (t *TrimVideoTask) monitorFileSize(tempFile string, originalSize int64, progress *job.Progress, done chan bool) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if fileInfo, err := os.Stat(tempFile); err == nil {
				currentSize := fileInfo.Size()
				if originalSize > 0 {
					percent := float64(currentSize) / float64(originalSize)
					if percent > 1.0 {
						percent = 1.0
					}

					progress.SetPercent(percent)

					logger.Infof("[trim-video] file size progress: %d/%d bytes (%.1f%%) - %.2f/%.2f MB",
						currentSize, originalSize, percent*100,
						float64(currentSize)/1024/1024, float64(originalSize)/1024/1024)
				} else {
					logger.Infof("[trim-video] current file size: %d bytes (%.2f MB)",
						currentSize, float64(currentSize)/1024/1024)
				}
			}
		}
	}
}

func (t *TrimVideoTask) monitorFileSizeWithQueue(tempFile string, originalSize int64, taskQueue *job.TaskQueue, done chan bool) {
	ticker := time.NewTicker(6 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if fileInfo, err := os.Stat(tempFile); err == nil {
				currentSize := fileInfo.Size()
				if originalSize > 0 {
					percent := float64(currentSize) / float64(originalSize)
					if percent > 1.0 {
						percent = 1.0
					}

					startVal := 0.0
					if t.StartTime != nil {
						startVal = *t.StartTime
					}
					endVal := 0.0
					if t.EndTime != nil {
						endVal = *t.EndTime
					}
					statusText := fmt.Sprintf("Trimming video from %.2fs to %.2fs - %.1f%% (%.2f/%.2f MB)",
						startVal, endVal, percent*100,
						float64(currentSize)/1024/1024,
						float64(originalSize)/1024/1024)

					select {
					case <-done:
						return
					default:
						taskQueue.Add(statusText, func(ctx context.Context) {
							time.Sleep(4 * time.Second)
						})
					}

					logger.Infof("[trim-video] file size progress: %d/%d bytes (%.1f%%) - %.2f/%.2f MB",
						currentSize, originalSize, percent*100,
						float64(currentSize)/1024/1024, float64(originalSize)/1024/1024)
				} else {
					startVal := 0.0
					if t.StartTime != nil {
						startVal = *t.StartTime
					}
					endVal := 0.0
					if t.EndTime != nil {
						endVal = *t.EndTime
					}
					statusText := fmt.Sprintf("Trimming video from %.2fs to %.2fs - %.2f MB",
						startVal, endVal,
						float64(currentSize)/1024/1024)

					select {
					case <-done:
						return
					default:
						taskQueue.Add(statusText, func(ctx context.Context) {
							time.Sleep(4 * time.Second)
						})
					}

					logger.Infof("[trim-video] current file size: %d bytes (%.2f MB)",
						currentSize, float64(currentSize)/1024/1024)
				}
			}
		}
	}
}

func (t *TrimVideoTask) performTrimWithProgress(ctx context.Context, inputPath, outputPath string, progress *job.Progress) error {
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(inputPath)
	if err != nil {
		return fmt.Errorf("error reading video file: %w", err)
	}

	// Build FFmpeg arguments based on what parameters are set
	args := ffmpeg.Args{"-i", inputPath}

	// Add start time if set
	if t.StartTime != nil {
		args = append(args, "-ss", fmt.Sprintf("%.2f", *t.StartTime))
	}

	// Add end time or duration if set
	if t.EndTime != nil {
		// If both start and end are set, use -to for end time
		if t.StartTime != nil {
			args = append(args, "-to", fmt.Sprintf("%.2f", *t.EndTime))
			logger.Infof("[trim-video] trimming from %.2fs to %.2fs", *t.StartTime, *t.EndTime)
		} else {
			// Only end time is set, trim from beginning to end time
			args = append(args, "-to", fmt.Sprintf("%.2f", *t.EndTime))
			logger.Infof("[trim-video] trimming from beginning to %.2fs", *t.EndTime)
		}
	} else if t.StartTime != nil {
		// Only start time is set, trim from start time to end
		logger.Infof("[trim-video] trimming from %.2fs to end", *t.StartTime)
	}

	// Add stream copy and other options
	args = append(args, "-c", "copy", "-avoid_negative_ts", "make_zero", outputPath)

	logger.Infof("[trim-video] running ffmpeg command: %v", args)
	logger.Infof("[trim-video] video duration: %.2f seconds", videoFile.FileDuration)

	// For stream copy, we can't track progress accurately, so we'll use a simple progress simulation
	progress.SetPercent(0)

	cmd := t.FFMpeg.Command(ctx, args)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg trim failed: %w", err)
	}

	progress.SetPercent(100)
	return nil
}

func (t *TrimVideoTask) validateTrimmedFile(filePath string) error {
	// Check if file exists and is readable
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("trimmed file does not exist or is not accessible: %w", err)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("trimmed file is empty")
	}

	logger.Infof("[trim-video] validating trimmed file: %s (size: %d bytes)", filePath, fileInfo.Size())

	// Probe the file with FFProbe
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to probe trimmed file: %w", err)
	}

	// Validate duration
	if videoFile.FileDuration <= 0 {
		return fmt.Errorf("trimmed file has invalid duration: %f", videoFile.FileDuration)
	}

	// Check if duration is approximately correct (within 1 second tolerance)
	var expectedDuration float64
	if t.StartTime != nil && t.EndTime != nil {
		expectedDuration = *t.EndTime - *t.StartTime
		if videoFile.FileDuration < expectedDuration-1.0 || videoFile.FileDuration > expectedDuration+1.0 {
			logger.Warnf("[trim-video] trimmed file duration %.2f doesn't match expected %.2f", videoFile.FileDuration, expectedDuration)
		}
		logger.Infof("[trim-video] trimmed file duration: %.2f seconds (expected: %.2f)", videoFile.FileDuration, expectedDuration)
	} else {
		logger.Infof("[trim-video] trimmed file duration: %.2f seconds", videoFile.FileDuration)
	}

	// Validate video codec
	if videoFile.VideoCodec == "" {
		return fmt.Errorf("trimmed file has no video stream")
	}

	if videoFile.VideoCodec != "h264" {
		return fmt.Errorf("trimmed file has wrong video codec: %s (expected h264)", videoFile.VideoCodec)
	}

	logger.Infof("[trim-video] trimmed file video codec: %s", videoFile.VideoCodec)

	// Validate audio codec (should be aac or empty)
	if videoFile.AudioCodec != "" && videoFile.AudioCodec != "aac" {
		logger.Warnf("[trim-video] trimmed file has unexpected audio codec: %s", videoFile.AudioCodec)
	}

	// Validate resolution
	if videoFile.Width <= 0 || videoFile.Height <= 0 {
		return fmt.Errorf("trimmed file has invalid resolution: %dx%d", videoFile.Width, videoFile.Height)
	}

	logger.Infof("[trim-video] trimmed file resolution: %dx%d", videoFile.Width, videoFile.Height)
	logger.Infof("[trim-video] trimmed file validation successful")
	return nil
}

func (t *TrimVideoTask) createNewVideoFile(ctx context.Context, filePath string) (*models.VideoFile, bool, error) {
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(filePath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to probe file: %w", err)
	}

	// Get the original file to copy its parent_folder_id
	originalFiles, err := t.Repository.File.Find(ctx, t.FileID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to find original file: %w", err)
	}

	if len(originalFiles) == 0 {
		return nil, false, fmt.Errorf("original file not found")
	}

	originalFile, ok := originalFiles[0].(*models.VideoFile)
	if !ok {
		return nil, false, fmt.Errorf("original file is not a video file")
	}

	// Create proper basename with .mp4 extension
	originalBasename := originalFile.Base().Basename
	ext := filepath.Ext(originalBasename)
	nameWithoutExt := strings.TrimSuffix(originalBasename, ext)
	properBasename := nameWithoutExt + ".mp4"

	// Check if a file with the same basename already exists in the same folder
	existingFile, err := t.Repository.File.FindByBasenameAndParentFolderID(ctx, properBasename, originalFile.Base().ParentFolderID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check for existing file: %w", err)
	}

	if existingFile != nil {
		// File with same name already exists, update it instead of creating new one
		logger.Infof("[trim-video] file %s already exists in folder %d, updating existing file", properBasename, originalFile.Base().ParentFolderID)

		// Cast to VideoFile to access video-specific fields
		existingVideoFile, ok := existingFile.(*models.VideoFile)
		if !ok {
			return nil, false, fmt.Errorf("existing file is not a video file")
		}

		// Check if the existing file is already associated with this scene
		isAssociated, err := t.isFileAssociatedWithScene(ctx, existingVideoFile.ID)
		if err != nil {
			return nil, false, fmt.Errorf("failed to check file association: %w", err)
		}

		// Update the existing file with new metadata
		finalPath := t.getFinalPath(existingVideoFile)
		existingVideoFile.Base().Path = finalPath
		existingVideoFile.Base().Size = videoFile.Size
		existingVideoFile.Base().ModTime = time.Now()
		existingVideoFile.Base().UpdatedAt = time.Now()

		// Update video-specific metadata
		existingVideoFile.Duration = videoFile.FileDuration
		existingVideoFile.VideoCodec = videoFile.VideoCodec
		existingVideoFile.AudioCodec = videoFile.AudioCodec
		existingVideoFile.Width = videoFile.Width
		existingVideoFile.Height = videoFile.Height
		existingVideoFile.FrameRate = videoFile.FrameRate
		existingVideoFile.BitRate = videoFile.Bitrate
		existingVideoFile.Format = "mp4"

		// Recalculate file hash as content has changed
		existingVideoFile.Base().Fingerprints = nil

		// Update the file in database
		err = t.Repository.File.Update(ctx, existingVideoFile)
		if err != nil {
			return nil, false, fmt.Errorf("failed to update existing video file in database: %w", err)
		}

		// If file is not associated with this scene, associate it
		if !isAssociated {
			logger.Infof("[trim-video] associating existing file %d with scene %d", existingVideoFile.ID, t.Scene.ID)
			fileIDs := []models.FileID{existingVideoFile.ID}
			if err := t.Repository.Scene.AssignFiles(ctx, t.Scene.ID, fileIDs); err != nil {
				return nil, false, fmt.Errorf("failed to associate existing file with scene: %w", err)
			}
		}

		logger.Infof("[trim-video] updated existing file %d with new trim metadata", existingVideoFile.ID)
		return existingVideoFile, true, nil
	}

	// No existing file found, create new one
	newFile := &models.VideoFile{
		BaseFile: &models.BaseFile{
			Path:           filePath,
			Basename:       properBasename,
			Size:           videoFile.Size,
			ParentFolderID: originalFile.Base().ParentFolderID,
			CreatedAt:      originalFile.Base().CreatedAt,
			UpdatedAt:      originalFile.Base().UpdatedAt,
			DirEntry: models.DirEntry{
				ModTime: originalFile.Base().ModTime,
			},
		},
		Duration:   videoFile.FileDuration,
		VideoCodec: videoFile.VideoCodec,
		AudioCodec: videoFile.AudioCodec,
		Width:      videoFile.Width,
		Height:     videoFile.Height,
		FrameRate:  videoFile.FrameRate,
		BitRate:    videoFile.Bitrate,
		Format:     "mp4",
	}

	// Create the file in database
	err = t.Repository.File.Create(ctx, newFile)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create video file in database: %w", err)
	}

	// Force recalculation of file fingerprints for new file
	newFile.Base().Fingerprints = nil
	err = t.Repository.File.Update(ctx, newFile)
	if err != nil {
		logger.Warnf("[trim-video] failed to update new file fingerprints: %v", err)
	}

	return newFile, false, nil
}

func (t *TrimVideoTask) updateSceneWithNewFile(ctx context.Context, newFile *models.VideoFile) error {
	// Use separate transaction for scene update to avoid blocking
	return t.Repository.WithTxn(ctx, func(ctx context.Context) error {
		// Associate the new file with the scene
		fileIDs := []models.FileID{newFile.ID}
		if err := t.Repository.Scene.AssignFiles(ctx, t.Scene.ID, fileIDs); err != nil {
			return fmt.Errorf("failed to associate file with scene: %w", err)
		}

		// Update scene to set new primary file and clear trim times
		scenePartial := models.NewScenePartial()
		scenePartial.PrimaryFileID = &newFile.ID
		// Clear start_time and end_time after trimming
		scenePartial.StartTime = models.OptionalFloat64{Null: true, Set: true}
		scenePartial.EndTime = models.OptionalFloat64{Null: true, Set: true}
		// Ensure scene is not marked as broken
		scenePartial.IsBroken = models.NewOptionalBool(false)

		// Update scene in database
		_, err := t.Repository.Scene.UpdatePartial(ctx, t.Scene.ID, scenePartial)
		if err != nil {
			return fmt.Errorf("failed to update scene metadata: %w", err)
		}

		logger.Infof("[trim-video] updated scene %d metadata with new file", t.Scene.ID)
		return nil
	})
}

func (t *TrimVideoTask) getFinalPath(file *models.VideoFile) string {
	// Find the original file from scene files
	var originalFile *models.VideoFile
	for _, vf := range t.Scene.Files.List() {
		if vf.ID == t.FileID {
			originalFile = vf
			break
		}
	}

	if originalFile == nil {
		logger.Warnf("[trim-video] original file not found, using scene primary file")
		originalFile = t.Scene.Files.Primary()
	}

	originalPath := originalFile.Path
	originalDir := filepath.Dir(originalPath)
	originalBasename := originalFile.Base().Basename

	// Create new filename with .mp4 extension
	ext := filepath.Ext(originalBasename)
	nameWithoutExt := strings.TrimSuffix(originalBasename, ext)
	newBasename := nameWithoutExt + ".mp4"

	// Ensure the original directory exists
	if err := os.MkdirAll(originalDir, 0755); err != nil {
		logger.Warnf("[trim-video] failed to ensure original directory exists %s: %v", originalDir, err)
	}

	logger.Infof("[trim-video] original path: %s", originalPath)
	logger.Infof("[trim-video] original basename: %s, new basename: %s", originalBasename, newBasename)
	logger.Infof("[trim-video] original directory: %s", originalDir)

	// Return the full path in the same directory as original file
	finalPath := filepath.Join(originalDir, newBasename)
	logger.Infof("[trim-video] final path: %s", finalPath)
	return finalPath
}

func (t *TrimVideoTask) updateFilePath(ctx context.Context, file *models.VideoFile, newPath string) error {
	// Update file path in database
	file.Base().Path = newPath
	file.Base().Basename = filepath.Base(newPath)

	err := t.Repository.File.Update(ctx, file)
	if err != nil {
		return fmt.Errorf("failed to update file path: %w", err)
	}

	logger.Infof("[trim-video] updated file path to %s", newPath)
	return nil
}

func (t *TrimVideoTask) deleteOldFileRecord(ctx context.Context, oldFile *models.VideoFile) error {
	// Delete the old file record from database
	if err := t.Repository.File.Destroy(ctx, oldFile.ID); err != nil {
		return fmt.Errorf("failed to delete old file record: %w", err)
	}

	logger.Infof("[trim-video] deleted old file record with ID %d", oldFile.ID)
	return nil
}

func (t *TrimVideoTask) recalculateFileHashes(ctx context.Context, file *models.VideoFile, filePath string) error {
	// Recalculate file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	file.Base().Size = fileInfo.Size()
	file.Base().ModTime = fileInfo.ModTime()

	// Create a file opener for the new file
	opener := &trimFileOpener{path: filePath}

	// Recalculate fingerprints using the fingerprint calculator
	fingerprints, err := t.FingerprintCalculator.CalculateFingerprints(file.Base(), opener, false)
	if err != nil {
		return fmt.Errorf("failed to calculate fingerprints: %w", err)
	}

	// Update fingerprints in the file
	file.Base().Fingerprints = models.Fingerprints{}
	for _, fp := range fingerprints {
		file.Base().Fingerprints = file.Base().Fingerprints.AppendUnique(fp)
	}

	// Recalculate phash if it's a video file
	if file.Duration > 0 {
		phash, err := videophash.Generate(t.FFMpeg, file)
		if err != nil {
			logger.Warnf("[trim-video] failed to calculate phash: %v", err)
		} else {
			phashInt := int64(*phash)
			file.Base().Fingerprints = file.Base().Fingerprints.AppendUnique(models.Fingerprint{
				Type:        models.FingerprintTypePhash,
				Fingerprint: phashInt,
			})
		}
	}

	// Update the file record in database
	if err := t.Repository.File.Update(ctx, file); err != nil {
		return fmt.Errorf("failed to update file with new hashes: %w", err)
	}

	// Log the calculated hashes
	checksum := file.Base().Fingerprints.Get(models.FingerprintTypeMD5)
	oshash := file.Base().Fingerprints.Get(models.FingerprintTypeOshash)
	logger.Infof("[trim-video] recalculated hashes - checksum: %v, oshash: %v", checksum, oshash)
	return nil
}

func (t *TrimVideoTask) generateVTTFile(ctx context.Context, file *models.VideoFile, filePath string) error {
	// Get updated scene from database with new hash
	updatedScene, err := t.Repository.Scene.Find(ctx, t.Scene.ID)
	if err != nil {
		return fmt.Errorf("failed to load updated scene: %w", err)
	}

	if updatedScene != nil {
		if err := updatedScene.LoadFiles(ctx, t.Repository.Scene); err != nil {
			return fmt.Errorf("failed to load scene files: %w", err)
		}
	}

	if updatedScene == nil {
		return fmt.Errorf("updated scene not found")
	}

	// Check if VTT file already exists
	sceneHash := updatedScene.GetHash(t.FileNamingAlgorithm)
	vttPath := t.Paths.Scene.GetSpriteVttFilePath(sceneHash)

	if _, err := os.Stat(vttPath); err == nil {
		logger.Infof("[trim-video] VTT file already exists: %s", vttPath)
		return nil
	}

	// Check if sprite image exists
	spritePath := t.Paths.Scene.GetSpriteImageFilePath(sceneHash)
	if _, err := os.Stat(spritePath); err != nil {
		logger.Infof("[trim-video] sprite image does not exist, skipping VTT generation: %s", spritePath)
		return nil
	}

	// Generate VTT file using the Generator
	generator := &generate.Generator{
		Encoder:      t.FFMpeg,
		FFMpegConfig: t.Config,
		LockManager:  t.G.LockManager,
		ScenePaths:   t.Paths.Scene,
	}

	// Calculate step size for VTT generation
	stepSize := 10.0
	if file.Duration > 0 {
		stepSize = file.Duration / 100.0
	}

	logger.Infof("[trim-video] generating VTT file: %s", vttPath)
	if err := generator.SpriteVTT(ctx, vttPath, spritePath, stepSize); err != nil {
		return fmt.Errorf("failed to generate VTT file: %w", err)
	}

	logger.Infof("[trim-video] successfully generated VTT file: %s", vttPath)
	return nil
}

func (t *TrimVideoTask) isFileAssociatedWithScene(ctx context.Context, fileID models.FileID) (bool, error) {
	// Get all files associated with the scene
	sceneFiles, err := t.Repository.Scene.GetFiles(ctx, t.Scene.ID)
	if err != nil {
		return false, fmt.Errorf("failed to get scene files: %w", err)
	}

	// Check if the file ID is in the list
	for _, sceneFile := range sceneFiles {
		if sceneFile.ID == fileID {
			return true, nil
		}
	}

	return false, nil
}

func (t *TrimVideoTask) copyFileContent(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	// Copy content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content from %s to %s: %w", src, dst, err)
	}

	// Sync to ensure data is written to disk
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file %s: %w", dst, err)
	}

	logger.Infof("[trim-video] successfully copied file content from %s to %s", src, dst)
	return nil
}

func (t *TrimVideoTask) regenerateSprites(ctx context.Context, oldHash string) error {
	// Get updated scene from database with new hash
	updatedScene, err := t.Repository.Scene.Find(ctx, t.Scene.ID)
	if err != nil {
		return fmt.Errorf("failed to load updated scene: %w", err)
	}

	if updatedScene == nil {
		return fmt.Errorf("updated scene not found")
	}

	newHash := updatedScene.GetHash(t.FileNamingAlgorithm)
	logger.Infof("[trim-video] sprite migration: old hash=%s, new hash=%s", oldHash, newHash)

	// If hash is empty, try to get phash from scene files
	if newHash == "" {
		logger.Infof("[trim-video] scene hash is empty, trying to get phash from files")

		// Load scene files to get phash
		if err := updatedScene.LoadFiles(ctx, t.Repository.Scene); err != nil {
			logger.Warnf("[trim-video] failed to load scene files: %v", err)
		} else {
			// Look for phash in scene files
			for _, vf := range updatedScene.Files.List() {
				videoFile := vf
				if phash := videoFile.Base().Fingerprints.Get(models.FingerprintTypePhash); phash != nil {
					newHash = phash.(string)
					logger.Infof("[trim-video] found phash in file %d: %s", videoFile.ID, newHash)
					break
				}
			}
		}
	}

	// Check if sprites exist for OLD hash
	oldSpriteImagePath := t.Paths.Scene.GetSpriteImageFilePath(oldHash)
	oldSpriteVttPath := t.Paths.Scene.GetSpriteVttFilePath(oldHash)

	// Paths for NEW hash
	newSpriteImagePath := t.Paths.Scene.GetSpriteImageFilePath(newHash)
	newSpriteVttPath := t.Paths.Scene.GetSpriteVttFilePath(newHash)

	logger.Infof("[trim-video] checking old sprites:")
	logger.Infof("[trim-video]   old image: %s", oldSpriteImagePath)
	logger.Infof("[trim-video]   old vtt: %s", oldSpriteVttPath)
	logger.Infof("[trim-video] new sprite paths:")
	logger.Infof("[trim-video]   new image: %s", newSpriteImagePath)
	logger.Infof("[trim-video]   new vtt: %s", newSpriteVttPath)

	// Verify that the new hash is not empty
	if newHash == "" {
		logger.Errorf("[trim-video] new hash is empty, cannot generate sprites")
		logger.Infof("[trim-video] scene OSHash: '%s', Checksum: '%s'", updatedScene.OSHash, updatedScene.Checksum)
		logger.Infof("[trim-video] file naming algorithm: %s", t.FileNamingAlgorithm)
		return fmt.Errorf("new hash is empty")
	}

	oldSpriteImageExists := false
	oldSpriteVttExists := false

	if _, err := os.Stat(oldSpriteImagePath); err == nil {
		oldSpriteImageExists = true
		logger.Infof("[trim-video] old sprite image exists")
	} else {
		logger.Infof("[trim-video] old sprite image does not exist")
	}

	if _, err := os.Stat(oldSpriteVttPath); err == nil {
		oldSpriteVttExists = true
		logger.Infof("[trim-video] old sprite vtt exists")
	} else {
		logger.Infof("[trim-video] old sprite vtt does not exist")
	}

	// For video trimming, we need to regenerate sprites as video content has changed
	// Delete old sprites first
	if oldSpriteImageExists {
		logger.Infof("[trim-video] deleting old sprite image: %s", oldSpriteImagePath)
		if err := os.Remove(oldSpriteImagePath); err != nil {
			logger.Warnf("[trim-video] failed to delete old sprite image: %v", err)
		}
	}

	if oldSpriteVttExists {
		logger.Infof("[trim-video] deleting old sprite VTT: %s", oldSpriteVttPath)
		if err := os.Remove(oldSpriteVttPath); err != nil {
			logger.Warnf("[trim-video] failed to delete old sprite VTT: %v", err)
		}
	}

	// If old sprites don't exist, check if new sprites already exist
	newSpriteImageExists := false
	newSpriteVttExists := false

	if _, err := os.Stat(newSpriteImagePath); err == nil {
		newSpriteImageExists = true
	}

	if _, err := os.Stat(newSpriteVttPath); err == nil {
		newSpriteVttExists = true
	}

	// Always generate new sprites for trimmed video as content has changed
	// Delete existing sprites for new hash if they exist
	if newSpriteImageExists {
		logger.Infof("[trim-video] deleting existing sprite image for new hash: %s", newSpriteImagePath)
		if err := os.Remove(newSpriteImagePath); err != nil {
			logger.Warnf("[trim-video] failed to delete existing sprite image: %v", err)
		}
	}

	if newSpriteVttExists {
		logger.Infof("[trim-video] deleting existing sprite VTT for new hash: %s", newSpriteVttPath)
		if err := os.Remove(newSpriteVttPath); err != nil {
			logger.Warnf("[trim-video] failed to delete existing sprite VTT: %v", err)
		}
	}

	logger.Infof("[trim-video] generating new sprites for trimmed video scene %d", t.Scene.ID)
	logger.Infof("[trim-video] using scene hash for sprite generation: %s", newHash)
	logger.Infof("[trim-video] scene path for sprite generation: %s", updatedScene.Path)

	spriteTask := GenerateSpriteTask{
		Scene:               *updatedScene,
		Overwrite:           true,
		fileNamingAlgorithm: t.FileNamingAlgorithm,
	}

	// Run sprite generation
	spriteTask.Start(ctx)
	logger.Infof("[trim-video] generated new sprites for scene %d with hash %s", t.Scene.ID, newHash)
	return nil
}

// clearTrimTimes removes start_time and end_time from the scene after successful trim
func (t *TrimVideoTask) clearTrimTimes(ctx context.Context) error {
	return t.Repository.WithTxn(ctx, func(ctx context.Context) error {
		// Create scene partial to clear start_time and end_time
		scenePartial := models.NewScenePartial()
		scenePartial.StartTime = models.OptionalFloat64{Null: true, Set: true} // Set to null to clear
		scenePartial.EndTime = models.OptionalFloat64{Null: true, Set: true}   // Set to null to clear

		// Update scene in database
		_, err := t.Repository.Scene.UpdatePartial(ctx, t.Scene.ID, scenePartial)
		if err != nil {
			return fmt.Errorf("failed to clear trim times from scene: %w", err)
		}

		logger.Infof("[trim-video] cleared start_time and end_time from scene %d", t.Scene.ID)
		return nil
	})
}
