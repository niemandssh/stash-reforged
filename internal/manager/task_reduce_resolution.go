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
	"github.com/stashapp/stash/pkg/ffmpeg/transcoder"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/hash/videophash"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/paths"
	"github.com/stashapp/stash/pkg/scene/generate"
)

// resolutionFileOpener implements file.Opener for OS files
type resolutionFileOpener struct {
	path string
}

func (o *resolutionFileOpener) Open() (io.ReadCloser, error) {
	return os.Open(o.path)
}

type ReduceResolutionTask struct {
	Scene                 models.Scene
	FileID                models.FileID // Конкретный файл для уменьшения разрешения
	TargetWidth           int
	TargetHeight          int
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

func (t *ReduceResolutionTask) GetDescription() string {
	return fmt.Sprintf("Reducing resolution of %s to %dx%d", t.Scene.Path, t.TargetWidth, t.TargetHeight)
}

func (t *ReduceResolutionTask) Execute(ctx context.Context, progress *job.Progress) error {
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

	// Проверка, что текущее разрешение больше целевого
	if targetFile.Width <= t.TargetWidth && targetFile.Height <= t.TargetHeight {
		return fmt.Errorf("current resolution %dx%d is already smaller or equal to target %dx%d",
			targetFile.Width, targetFile.Height, t.TargetWidth, t.TargetHeight)
	}

	logger.Infof("[reduce-res] reducing resolution of scene %d from %dx%d to %dx%d",
		t.Scene.ID, targetFile.Width, targetFile.Height, t.TargetWidth, t.TargetHeight)

	progress.SetTotal(3)
	progress.SetProcessed(0)

	var conversionErr error

	// Get original file size for display
	originalFileInfo, err := os.Stat(targetFile.Path)
	if err == nil {
		logger.Infof("[reduce-res] original file size: %d bytes (%.2f MB)", originalFileInfo.Size(), float64(originalFileInfo.Size())/1024/1024)
	}

	// Start file size monitoring
	originalSize := int64(0)
	if originalFileInfo != nil {
		originalSize = originalFileInfo.Size()
	}

	// Start monitoring file size in a goroutine
	done := make(chan bool)
	tempFile := filepath.Join(t.Config.GetGeneratedPath(), fmt.Sprintf("reduce_res_%d_%s_%dx%d.mp4",
		t.Scene.ID, t.Scene.GetHash(t.FileNamingAlgorithm), t.TargetWidth, t.TargetHeight))
	go t.monitorFileSize(tempFile, originalSize, progress, done)

	// Start monitoring in a goroutine
	go t.monitorFileSizeWithStatusUpdate(tempFile, originalSize, progress, done)

	// Create a task queue for dynamic status updates
	taskQueue := job.NewTaskQueue(ctx, progress, 100, 1)
	go t.monitorFileSizeWithQueue(tempFile, originalSize, taskQueue, done)

	// Wrap conversion in transaction
	conversionErr = t.Repository.WithTxn(ctx, func(ctx context.Context) error {
		return t.reduceResolution(ctx, targetFile, progress, done)
	})
	if conversionErr != nil {
		logger.Errorf("[reduce-res] error reducing resolution of scene %d: %v", t.Scene.ID, conversionErr)
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

	progress.ExecuteTask("Finalizing reduction", func() {
		if conversionErr == nil {
			progress.SetProcessed(3)
		}
	})

	if conversionErr == nil {
		logger.Infof("[reduce-res] successfully reduced resolution of scene %d", t.Scene.ID)
	} else {
		return conversionErr
	}

	return nil
}

// For backward compatibility
func (t *ReduceResolutionTask) Start(ctx context.Context) {
	progress := &job.Progress{}
	t.Execute(ctx, progress)
}

func (t *ReduceResolutionTask) reduceResolution(ctx context.Context, f *models.VideoFile, progress *job.Progress, done chan bool) error {
	// Save old hash BEFORE conversion for sprite migration
	oldHash := t.Scene.GetHash(t.FileNamingAlgorithm)
	logger.Infof("[reduce-res] old scene hash before reduction: %s", oldHash)

	tempDir := t.Config.GetGeneratedPath()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("reduce_res_%d_%s_%dx%d.mp4",
		t.Scene.ID, oldHash, t.TargetWidth, t.TargetHeight))

	// Create independent backup copy in temp directory
	backupTempDir := t.Config.GetTempPath()
	logger.Infof("[reduce-res] Creating backup temp directory: %s", backupTempDir)
	if err := os.MkdirAll(backupTempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp backup directory %s: %w", backupTempDir, err)
	}
	// Use original filename for backup in temp
	originalFilename := filepath.Base(f.Path)
	backupTempFile := filepath.Join(backupTempDir, originalFilename)
	logger.Infof("[reduce-res] Backup temp file path: %s", backupTempFile)

	// Create backup copy of ORIGINAL file in temp directory BEFORE conversion
	logger.Infof("[reduce-res] Creating backup copy of original file from %s to %s", f.Path, backupTempFile)
	if err := t.copyFileContent(f.Path, backupTempFile); err != nil {
		return fmt.Errorf("failed to create backup copy of original file in temp: %w", err)
	}
	logger.Infof("[reduce-res] Successfully created backup copy of original file in temp: %s", backupTempFile)

	// Get original file size for progress tracking
	originalFileInfo, err := os.Stat(f.Path)
	if err != nil {
		logger.Warnf("[reduce-res] failed to get original file size: %v", err)
	} else {
		logger.Infof("[reduce-res] original file size: %d bytes (%.2f MB)", originalFileInfo.Size(), float64(originalFileInfo.Size())/1024/1024)
	}

	// Track if conversion was successful
	conversionSuccessful := false

	// Clean up temp files at the end
	defer func() {
		// Clean up main temp file only on failure
		if !conversionSuccessful {
			if _, err := os.Stat(tempFile); err == nil {
				if err := os.Remove(tempFile); err != nil {
					logger.Warnf("[reduce-res] failed to remove temp file %s: %v", tempFile, err)
				} else {
					logger.Infof("[reduce-res] cleaned up temp file: %s", tempFile)
				}
			}
		}
	}()

	if err := t.performReductionWithProgress(ctx, f.Path, tempFile, progress); err != nil {
		logger.Errorf("[reduce-res] reduction failed: %v", err)
		return fmt.Errorf("reduction failed: %w", err)
	}

	if err := t.validateReducedFile(tempFile); err != nil {
		return fmt.Errorf("reduced file validation failed: %w", err)
	}

	newFile, isUpdated, err := t.createNewVideoFile(ctx, tempFile)
	if err != nil {
		return fmt.Errorf("failed to create new video file: %w", err)
	}

	if err := t.updateSceneWithNewFile(ctx, newFile); err != nil {
		return fmt.Errorf("failed to update scene with new file: %w", err)
	}

	if isUpdated {
		// File was updated, check if we need to copy temp file to existing file
		finalPath := newFile.Base().Path
		logger.Infof("[reduce-res] checking if temp file needs to be copied to existing file: %s", finalPath)

		// Only copy if paths are different (avoid copying file to itself)
		if tempFile != finalPath {
			logger.Infof("[reduce-res] copying temp file content to existing file: %s -> %s", tempFile, finalPath)
			if err := t.copyFileContent(tempFile, finalPath); err != nil {
				return fmt.Errorf("failed to copy temp file content to existing file: %w", err)
			}
		} else {
			logger.Infof("[reduce-res] temp file and final path are the same, no copy needed: %s", finalPath)
		}

		// Validate the updated file
		if err := t.validateReducedFile(finalPath); err != nil {
			logger.Errorf("[reduce-res] updated file validation failed: %v", err)
			return fmt.Errorf("updated file validation failed: %w", err)
		}

		logger.Infof("[reduce-res] successfully updated existing file: %s", finalPath)
	} else {
		// New file was created, move temp file to final location
		finalPath := t.getFinalPath(newFile)
		logger.Infof("[reduce-res] moving file from %s to %s", tempFile, finalPath)

		// Check if temp file exists
		if _, err := os.Stat(tempFile); err != nil {
			return fmt.Errorf("temp file does not exist: %w", err)
		}

		// Copy temp file to final location (works across different filesystems)
		logger.Infof("[reduce-res] copying temp file to final location: %s -> %s", tempFile, finalPath)
		if err := t.copyFileContent(tempFile, finalPath); err != nil {
			return fmt.Errorf("failed to copy reduced file to final location: %w", err)
		}

		// Remove temp file after successful copy
		if err := os.Remove(tempFile); err != nil {
			logger.Warnf("[reduce-res] failed to remove temp file %s: %v", tempFile, err)
		} else {
			logger.Infof("[reduce-res] removed temp file: %s", tempFile)
		}

		// Verify the file was moved successfully
		if _, err := os.Stat(finalPath); err != nil {
			return fmt.Errorf("final file does not exist after move: %w", err)
		}

		logger.Infof("[reduce-res] successfully moved file to %s", finalPath)

		if err := t.updateFilePath(ctx, newFile, finalPath); err != nil {
			return fmt.Errorf("failed to update file path: %w", err)
		}

		// Validate the reduced file before removing the original
		if err := t.validateReducedFile(finalPath); err != nil {
			logger.Errorf("[reduce-res] reduced file validation failed, keeping original: %v", err)
			return fmt.Errorf("reduced file validation failed: %w", err)
		}

		// Remove the original file only after successful validation
		originalPath := f.Path
		if err := os.Remove(originalPath); err != nil {
			logger.Warnf("[reduce-res] failed to remove original file %s: %v", originalPath, err)
		} else {
			logger.Infof("[reduce-res] removed original file: %s", originalPath)
		}

		// Delete the old file record from database
		if err := t.deleteOldFileRecord(ctx, f); err != nil {
			logger.Warnf("[reduce-res] failed to delete old file record: %v", err)
		} else {
			logger.Infof("[reduce-res] deleted old file record from database")
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
		logger.Warnf("[reduce-res] failed to recalculate file hashes: %v", err)
	} else {
		logger.Infof("[reduce-res] recalculated file hashes")
	}

	// Regenerate sprites with new hash after reduction (oldHash saved at start of function)
	logger.Infof("[reduce-res] regenerating sprites for reduced file")
	if err := t.regenerateSprites(ctx, oldHash); err != nil {
		logger.Warnf("[reduce-res] failed to regenerate sprites: %v", err)
		// Don't fail the conversion if sprite generation fails
	}

	// Generate VTT file for the new video if it doesn't exist
	if err := t.generateVTTFile(ctx, newFile, finalPath); err != nil {
		logger.Warnf("[reduce-res] failed to generate VTT file: %v", err)
	} else {
		logger.Infof("[reduce-res] generated VTT file")
	}

	// Clean up backup temp file only after all operations are successful
	if _, err := os.Stat(backupTempFile); err == nil {
		if err := os.Remove(backupTempFile); err != nil {
			logger.Warnf("[reduce-res] failed to remove backup temp file %s: %v", backupTempFile, err)
		} else {
			logger.Infof("[reduce-res] cleaned up backup temp file: %s", backupTempFile)
		}
	}

	// Mark conversion as successful
	conversionSuccessful = true
	return nil
}

func (t *ReduceResolutionTask) monitorFileSize(tempFile string, originalSize int64, progress *job.Progress, done chan bool) {
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

					logger.Infof("[reduce-res] file size progress: %d/%d bytes (%.1f%%) - %.2f/%.2f MB",
						currentSize, originalSize, percent*100,
						float64(currentSize)/1024/1024, float64(originalSize)/1024/1024)
				} else {
					logger.Infof("[reduce-res] current file size: %d bytes (%.2f MB)",
						currentSize, float64(currentSize)/1024/1024)
				}
			}
		}
	}
}

func (t *ReduceResolutionTask) monitorFileSizeWithStatusUpdate(tempFile string, originalSize int64, progress *job.Progress, done chan bool) {
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

					logger.Infof("[reduce-res] file size progress: %d/%d bytes (%.1f%%) - %.2f/%.2f MB",
						currentSize, originalSize, percent*100,
						float64(currentSize)/1024/1024, float64(originalSize)/1024/1024)
				} else {
					logger.Infof("[reduce-res] current file size: %d bytes (%.2f MB)",
						currentSize, float64(currentSize)/1024/1024)
				}
			}
		}
	}
}

func (t *ReduceResolutionTask) monitorFileSizeWithQueue(tempFile string, originalSize int64, taskQueue *job.TaskQueue, done chan bool) {
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

					statusText := fmt.Sprintf("Reducing video resolution to %dx%d - %.1f%% (%.2f/%.2f MB)",
						t.TargetWidth, t.TargetHeight, percent*100,
						float64(currentSize)/1024/1024,
						float64(originalSize)/1024/1024)

					taskQueue.Add(statusText, func(ctx context.Context) {
						time.Sleep(4 * time.Second)
					})

					logger.Infof("[reduce-res] file size progress: %d/%d bytes (%.1f%%) - %.2f/%.2f MB",
						currentSize, originalSize, percent*100,
						float64(currentSize)/1024/1024, float64(originalSize)/1024/1024)
				} else {
					statusText := fmt.Sprintf("Reducing video resolution to %dx%d - %.2f MB",
						t.TargetWidth, t.TargetHeight,
						float64(currentSize)/1024/1024)

					taskQueue.Add(statusText, func(ctx context.Context) {
						time.Sleep(4 * time.Second)
					})

					logger.Infof("[reduce-res] current file size: %d bytes (%.2f MB)",
						currentSize, float64(currentSize)/1024/1024)
				}
			}
		}
	}
}

func (t *ReduceResolutionTask) getHardwareCodecForReduction() *ffmpeg.VideoCodec {
	codecs := []ffmpeg.VideoCodec{
		ffmpeg.VideoCodecN264,
		ffmpeg.VideoCodecI264,
		ffmpeg.VideoCodecV264,
		ffmpeg.VideoCodecA264,
	}

	for _, codec := range codecs {
		logger.Infof("[reduce-res] testing hardware codec: %s (%s)", codec.Name, codec.CodeName)
		if t.testHardwareCodec(codec) {
			logger.Infof("[reduce-res] ✓ hardware codec %s is available", codec.Name)
			return &codec
		}
	}

	logger.Infof("[reduce-res] no hardware codec available")
	return nil
}

func (t *ReduceResolutionTask) testHardwareCodec(codec ffmpeg.VideoCodec) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var args ffmpeg.Args
	args = append(args, "-hide_banner", "-loglevel", "error")
	args = args.Format("lavfi")
	args = args.Input("color=c=black:s=320x240")
	args = append(args, "-t", "0.1")
	args = args.VideoCodec(codec)

	switch codec {
	case ffmpeg.VideoCodecN264:
		args = append(args, "-preset", "fast", "-b:v", "1M")
	case ffmpeg.VideoCodecI264:
		args = append(args, "-preset", "fast", "-global_quality", "20")
	case ffmpeg.VideoCodecV264:
		args = append(args, "-qp", "20")
	case ffmpeg.VideoCodecA264:
		args = append(args, "-quality", "balanced")
	}

	args = args.Format("null")
	args = args.Output("-")

	cmd := t.FFMpeg.Command(ctx, args)
	err := cmd.Run()

	return err == nil
}

func (t *ReduceResolutionTask) getVideoArgsForCodec(codec ffmpeg.VideoCodec, w, h int) ffmpeg.Args {
	var videoArgs ffmpeg.Args

	if w != 0 && h != 0 {
		var videoFilter ffmpeg.VideoFilter
		videoFilter = videoFilter.ScaleDimensions(w, h)
		videoArgs = videoArgs.VideoFilter(videoFilter)
	}

	switch codec {
	case ffmpeg.VideoCodecN264, ffmpeg.VideoCodecN264H:
		videoArgs = append(videoArgs,
			"-rc", "vbr",
			"-cq", "23",
			"-preset", "p4",
			"-tune", "hq",
			"-profile:v", "high",
			"-level", "4.2",
			"-b:v", "0",
		)
	case ffmpeg.VideoCodecI264, ffmpeg.VideoCodecI264C:
		videoArgs = append(videoArgs,
			"-global_quality", "23",
			"-preset", "medium",
			"-profile:v", "high",
			"-level", "4.2",
			"-look_ahead", "1",
		)
	case ffmpeg.VideoCodecV264:
		videoArgs = append(videoArgs,
			"-qp", "23",
			"-profile:v", "high",
			"-level", "4.2",
			"-quality", "1",
		)
	case ffmpeg.VideoCodecM264:
		videoArgs = append(videoArgs,
			"-b:v", "0",
			"-q:v", "70",
			"-profile:v", "high",
			"-level", "4.2",
		)
	case ffmpeg.VideoCodecA264:
		videoArgs = append(videoArgs,
			"-quality", "balanced",
			"-rc", "vbr_latency",
			"-qp_i", "23",
			"-qp_p", "23",
			"-profile:v", "high",
			"-level", "4.2",
		)
	default:
		videoArgs = append(videoArgs,
			"-pix_fmt", "yuv420p",
			"-profile:v", "high",
			"-level", "4.2",
			"-preset", "medium",
			"-crf", "23",
		)
	}

	return videoArgs
}

func (t *ReduceResolutionTask) performReductionWithProgress(ctx context.Context, inputPath, outputPath string, progress *job.Progress) error {
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(inputPath)
	if err != nil {
		return fmt.Errorf("error reading video file: %w", err)
	}

	// Use target resolution
	w, h := t.TargetWidth, t.TargetHeight

	audioCodec := ffmpeg.AudioCodecAAC
	audioArgs := ffmpeg.Args{
		"-ac", "2",
		"-ar", "44100",
		"-ab", "96k",
		"-strict", "-2",
	}

	extraInputArgs := append(t.Config.GetTranscodeInputArgs(),
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",
	)

	extraOutputArgs := append(t.Config.GetTranscodeOutputArgs(),
		"-movflags", "+faststart",
	)

	hwCodec := t.getHardwareCodecForReduction()

	if hwCodec != nil {
		logger.Infof("[reduce-res] attempting hardware acceleration with codec: %s", hwCodec.Name)

		videoArgs := t.getVideoArgsForCodec(*hwCodec, w, h)

		args := transcoder.Transcode(inputPath, transcoder.TranscodeOptions{
			OutputPath:      outputPath,
			VideoCodec:      *hwCodec,
			VideoArgs:       videoArgs,
			AudioCodec:      audioCodec,
			AudioArgs:       audioArgs,
			Format:          ffmpeg.FormatMP4,
			ExtraInputArgs:  extraInputArgs,
			ExtraOutputArgs: extraOutputArgs,
		})

		logger.Infof("[reduce-res] running hardware-accelerated ffmpeg command: %v", args)
		logger.Infof("[reduce-res] video duration: %.2f seconds", videoFile.FileDuration)

		err := t.FFMpeg.GenerateWithProgress(ctx, args, progress, videoFile.FileDuration)
		if err == nil {
			logger.Infof("[reduce-res] hardware acceleration successful")
			return nil
		}

		logger.Warnf("[reduce-res] hardware acceleration failed: %v, falling back to software encoding", err)

		if _, removeErr := os.Stat(outputPath); removeErr == nil {
			os.Remove(outputPath)
		}
	} else {
		logger.Infof("[reduce-res] no hardware acceleration available, using software encoding")
	}

	var videoArgs ffmpeg.Args
	if w != 0 && h != 0 {
		var videoFilter ffmpeg.VideoFilter
		videoFilter = videoFilter.ScaleDimensions(w, h)
		videoArgs = videoArgs.VideoFilter(videoFilter)
	}

	videoArgs = append(videoArgs,
		"-pix_fmt", "yuv420p",
		"-profile:v", "high",
		"-level", "4.2",
		"-preset", "medium",
		"-crf", "23",
	)

	args := transcoder.Transcode(inputPath, transcoder.TranscodeOptions{
		OutputPath:      outputPath,
		VideoCodec:      ffmpeg.VideoCodecLibX264,
		VideoArgs:       videoArgs,
		AudioCodec:      audioCodec,
		AudioArgs:       audioArgs,
		Format:          ffmpeg.FormatMP4,
		ExtraInputArgs:  extraInputArgs,
		ExtraOutputArgs: extraOutputArgs,
	})

	logger.Infof("[reduce-res] running software ffmpeg command: %v", args)
	logger.Infof("[reduce-res] video duration: %.2f seconds", videoFile.FileDuration)
	return t.FFMpeg.GenerateWithProgress(ctx, args, progress, videoFile.FileDuration)
}

func (t *ReduceResolutionTask) validateReducedFile(filePath string) error {
	// Check if file exists and is readable
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("reduced file does not exist or is not accessible: %w", err)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("reduced file is empty")
	}

	logger.Infof("[reduce-res] validating reduced file: %s (size: %d bytes)", filePath, fileInfo.Size())

	// Probe the file with FFProbe
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to probe reduced file: %w", err)
	}

	// Validate duration
	if videoFile.FileDuration <= 0 {
		return fmt.Errorf("reduced file has invalid duration: %f", videoFile.FileDuration)
	}

	logger.Infof("[reduce-res] reduced file duration: %.2f seconds", videoFile.FileDuration)

	// Validate video codec
	if videoFile.VideoCodec == "" {
		return fmt.Errorf("reduced file has no video stream")
	}

	if videoFile.VideoCodec != "h264" {
		return fmt.Errorf("reduced file has wrong video codec: %s (expected h264)", videoFile.VideoCodec)
	}

	logger.Infof("[reduce-res] reduced file video codec: %s", videoFile.VideoCodec)

	// Validate audio codec (should be aac or empty)
	if videoFile.AudioCodec != "" && videoFile.AudioCodec != "aac" {
		logger.Warnf("[reduce-res] reduced file has unexpected audio codec: %s", videoFile.AudioCodec)
	}

	// Validate resolution
	if videoFile.Width <= 0 || videoFile.Height <= 0 {
		return fmt.Errorf("reduced file has invalid resolution: %dx%d", videoFile.Width, videoFile.Height)
	}

	// Check if resolution matches target
	if videoFile.Width != t.TargetWidth || videoFile.Height != t.TargetHeight {
		logger.Warnf("[reduce-res] reduced file resolution %dx%d doesn't exactly match target %dx%d",
			videoFile.Width, videoFile.Height, t.TargetWidth, t.TargetHeight)
	}

	logger.Infof("[reduce-res] reduced file resolution: %dx%d", videoFile.Width, videoFile.Height)
	logger.Infof("[reduce-res] reduced file validation successful")
	return nil
}

func (t *ReduceResolutionTask) createNewVideoFile(ctx context.Context, filePath string) (*models.VideoFile, bool, error) {
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
		logger.Infof("[reduce-res] file %s already exists in folder %d, updating existing file", properBasename, originalFile.Base().ParentFolderID)

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

		// Update the file in database
		err = t.Repository.File.Update(ctx, existingVideoFile)
		if err != nil {
			return nil, false, fmt.Errorf("failed to update existing video file in database: %w", err)
		}

		// If file is not associated with this scene, associate it
		if !isAssociated {
			logger.Infof("[reduce-res] associating existing file %d with scene %d", existingVideoFile.ID, t.Scene.ID)
			fileIDs := []models.FileID{existingVideoFile.ID}
			if err := t.Repository.Scene.AssignFiles(ctx, t.Scene.ID, fileIDs); err != nil {
				return nil, false, fmt.Errorf("failed to associate existing file with scene: %w", err)
			}
		}

		logger.Infof("[reduce-res] updated existing file %d with new resolution metadata", existingVideoFile.ID)
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

	return newFile, false, nil
}

func (t *ReduceResolutionTask) updateSceneWithNewFile(ctx context.Context, newFile *models.VideoFile) error {
	// Associate the new file with the scene
	fileIDs := []models.FileID{newFile.ID}
	if err := t.Repository.Scene.AssignFiles(ctx, t.Scene.ID, fileIDs); err != nil {
		return fmt.Errorf("failed to associate file with scene: %w", err)
	}

	// Update scene to set new primary file
	scenePartial := models.NewScenePartial()
	scenePartial.PrimaryFileID = &newFile.ID

	// Update scene in database
	_, err := t.Repository.Scene.UpdatePartial(ctx, t.Scene.ID, scenePartial)
	if err != nil {
		return fmt.Errorf("failed to update scene metadata: %w", err)
	}

	logger.Infof("[reduce-res] updated scene %d metadata with new file", t.Scene.ID)
	return nil
}

func (t *ReduceResolutionTask) getFinalPath(file *models.VideoFile) string {
	// Find the original file from scene files
	var originalFile *models.VideoFile
	for _, vf := range t.Scene.Files.List() {
		if vf.ID == t.FileID {
			originalFile = vf
			break
		}
	}

	if originalFile == nil {
		logger.Warnf("[reduce-res] original file not found, using scene primary file")
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
		logger.Warnf("[reduce-res] failed to ensure original directory exists %s: %v", originalDir, err)
	}

	logger.Infof("[reduce-res] original path: %s", originalPath)
	logger.Infof("[reduce-res] original basename: %s, new basename: %s", originalBasename, newBasename)
	logger.Infof("[reduce-res] original directory: %s", originalDir)

	// Return the full path in the same directory as original file
	finalPath := filepath.Join(originalDir, newBasename)
	logger.Infof("[reduce-res] final path: %s", finalPath)
	return finalPath
}

func (t *ReduceResolutionTask) updateFilePath(ctx context.Context, file *models.VideoFile, newPath string) error {
	// Update file path in database
	file.Base().Path = newPath
	file.Base().Basename = filepath.Base(newPath)

	err := t.Repository.File.Update(ctx, file)
	if err != nil {
		return fmt.Errorf("failed to update file path: %w", err)
	}

	logger.Infof("[reduce-res] updated file path to %s", newPath)
	return nil
}

func (t *ReduceResolutionTask) deleteOldFileRecord(ctx context.Context, oldFile *models.VideoFile) error {
	// Delete the old file record from database
	if err := t.Repository.File.Destroy(ctx, oldFile.ID); err != nil {
		return fmt.Errorf("failed to delete old file record: %w", err)
	}

	logger.Infof("[reduce-res] deleted old file record with ID %d", oldFile.ID)
	return nil
}

func (t *ReduceResolutionTask) recalculateFileHashes(ctx context.Context, file *models.VideoFile, filePath string) error {
	// Recalculate file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	file.Base().Size = fileInfo.Size()
	file.Base().ModTime = fileInfo.ModTime()

	// Create a file opener for the new file
	opener := &resolutionFileOpener{path: filePath}

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
			logger.Warnf("[reduce-res] failed to calculate phash: %v", err)
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
	logger.Infof("[reduce-res] recalculated hashes - checksum: %v, oshash: %v", checksum, oshash)
	return nil
}

func (t *ReduceResolutionTask) generateVTTFile(ctx context.Context, file *models.VideoFile, filePath string) error {
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
		logger.Infof("[reduce-res] VTT file already exists: %s", vttPath)
		return nil
	}

	// Check if sprite image exists
	spritePath := t.Paths.Scene.GetSpriteImageFilePath(sceneHash)
	if _, err := os.Stat(spritePath); err != nil {
		logger.Infof("[reduce-res] sprite image does not exist, skipping VTT generation: %s", spritePath)
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

	logger.Infof("[reduce-res] generating VTT file: %s", vttPath)
	if err := generator.SpriteVTT(ctx, vttPath, spritePath, stepSize); err != nil {
		return fmt.Errorf("failed to generate VTT file: %w", err)
	}

	logger.Infof("[reduce-res] successfully generated VTT file: %s", vttPath)
	return nil
}

func (t *ReduceResolutionTask) isFileAssociatedWithScene(ctx context.Context, fileID models.FileID) (bool, error) {
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

func (t *ReduceResolutionTask) copyFileContent(src, dst string) error {
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

	logger.Infof("[reduce-res] successfully copied file content from %s to %s", src, dst)
	return nil
}

func (t *ReduceResolutionTask) regenerateSprites(ctx context.Context, oldHash string) error {
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

	newHash := updatedScene.GetHash(t.FileNamingAlgorithm)
	logger.Infof("[reduce-res] sprite migration: old hash=%s, new hash=%s", oldHash, newHash)

	// Check if sprites exist for OLD hash
	oldSpriteImagePath := t.Paths.Scene.GetSpriteImageFilePath(oldHash)
	oldSpriteVttPath := t.Paths.Scene.GetSpriteVttFilePath(oldHash)

	// Paths for NEW hash
	newSpriteImagePath := t.Paths.Scene.GetSpriteImageFilePath(newHash)
	newSpriteVttPath := t.Paths.Scene.GetSpriteVttFilePath(newHash)

	logger.Infof("[reduce-res] checking old sprites:")
	logger.Infof("[reduce-res]   old image: %s", oldSpriteImagePath)
	logger.Infof("[reduce-res]   old vtt: %s", oldSpriteVttPath)
	logger.Infof("[reduce-res] new sprite paths:")
	logger.Infof("[reduce-res]   new image: %s", newSpriteImagePath)
	logger.Infof("[reduce-res]   new vtt: %s", newSpriteVttPath)

	oldSpriteImageExists := false
	oldSpriteVttExists := false

	if _, err := os.Stat(oldSpriteImagePath); err == nil {
		oldSpriteImageExists = true
		logger.Infof("[reduce-res] old sprite image exists")
	} else {
		logger.Infof("[reduce-res] old sprite image does not exist")
	}

	if _, err := os.Stat(oldSpriteVttPath); err == nil {
		oldSpriteVttExists = true
		logger.Infof("[reduce-res] old sprite vtt exists")
	} else {
		logger.Infof("[reduce-res] old sprite vtt does not exist")
	}

	// If both old sprites exist, rename them to new hash
	if oldSpriteImageExists && oldSpriteVttExists {
		logger.Infof("[reduce-res] migrating existing sprites from old hash to new hash")

		// First, update VTT file content to reference new hash
		if err := t.updateVttFileHash(oldSpriteVttPath, oldHash, newHash); err != nil {
			logger.Warnf("[reduce-res] failed to update VTT file hash: %v", err)
			// Continue with migration even if VTT update fails
		} else {
			logger.Infof("[reduce-res] updated VTT file to reference new hash")
		}

		// Rename sprite image
		if err := os.Rename(oldSpriteImagePath, newSpriteImagePath); err != nil {
			logger.Warnf("[reduce-res] failed to rename sprite image: %v", err)
		} else {
			logger.Infof("[reduce-res] renamed sprite image: %s -> %s", oldSpriteImagePath, newSpriteImagePath)
		}

		// Rename sprite vtt
		if err := os.Rename(oldSpriteVttPath, newSpriteVttPath); err != nil {
			logger.Warnf("[reduce-res] failed to rename sprite vtt: %v", err)
		} else {
			logger.Infof("[reduce-res] renamed sprite vtt: %s -> %s", oldSpriteVttPath, newSpriteVttPath)
		}

		logger.Infof("[reduce-res] sprite migration completed for scene %d", t.Scene.ID)
		return nil
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

	if newSpriteImageExists && newSpriteVttExists {
		logger.Infof("[reduce-res] sprites already exist for new hash, skipping regeneration")
		return nil
	}

	// Generate new sprites
	logger.Infof("[reduce-res] generating new sprites for scene %d", t.Scene.ID)
	spriteTask := GenerateSpriteTask{
		Scene:               *updatedScene,
		Overwrite:           true,
		fileNamingAlgorithm: t.FileNamingAlgorithm,
	}

	// Run sprite generation
	spriteTask.Start(ctx)
	logger.Infof("[reduce-res] generated new sprites for scene %d with hash %s", t.Scene.ID, newHash)
	return nil
}

// updateVttFileHash updates the VTT file to replace old hash with new hash in image references
func (t *ReduceResolutionTask) updateVttFileHash(vttPath, oldHash, newHash string) error {
	// Read VTT file content
	content, err := os.ReadFile(vttPath)
	if err != nil {
		return fmt.Errorf("failed to read VTT file: %w", err)
	}

	// Replace old hash with new hash in the content
	oldContent := string(content)
	newContent := strings.ReplaceAll(oldContent, oldHash, newHash)

	// Write updated content back to VTT file
	if err := os.WriteFile(vttPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated VTT file: %w", err)
	}

	logger.Infof("[reduce-res] updated VTT file: replaced %s with %s", oldHash, newHash)
	return nil
}
