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

// osFileOpener implements file.Opener for OS files
type osFileOpener struct {
	path string
}

func (o *osFileOpener) Open() (io.ReadCloser, error) {
	return os.Open(o.path)
}

type ConvertToMP4Task struct {
	Scene                 models.Scene
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

func (t *ConvertToMP4Task) GetDescription() string {
	return fmt.Sprintf("Converting %s to MP4", t.Scene.Path)
}

func (t *ConvertToMP4Task) Execute(ctx context.Context, progress *job.Progress) error {
	f := t.Scene.Files.Primary()
	if f == nil {
		return fmt.Errorf("scene has no primary file")
	}

	if t.needsConversion(f) {
		logger.Infof("[convert] converting scene %d to MP4", t.Scene.ID)

		progress.SetTotal(3)
		progress.SetProcessed(0)

		var conversionErr error
		// Get original file size for display
		originalFileInfo, err := os.Stat(f.Path)
		if err == nil {
			logger.Infof("[convert] original file size: %d bytes (%.2f MB)", originalFileInfo.Size(), float64(originalFileInfo.Size())/1024/1024)
		}

		// Start file size monitoring
		originalSize := int64(0)
		if originalFileInfo != nil {
			originalSize = originalFileInfo.Size()
		}

		// Start monitoring file size in a goroutine
		done := make(chan bool)
		tempFile := filepath.Join(t.Config.GetGeneratedPath(), fmt.Sprintf("convert_%d_%s.mp4", t.Scene.ID, t.Scene.GetHash(t.FileNamingAlgorithm)))
		go t.monitorFileSize(tempFile, originalSize, progress, done)

		// Start monitoring in a goroutine
		go t.monitorFileSizeWithStatusUpdate(tempFile, originalSize, progress, done)

		// Create a task queue for dynamic status updates
		taskQueue := job.NewTaskQueue(ctx, progress, 100, 1)
		go t.monitorFileSizeWithQueue(tempFile, originalSize, taskQueue, done)

		// Wrap conversion in transaction
		conversionErr = t.Repository.WithTxn(ctx, func(ctx context.Context) error {
			return t.convertToMP4(ctx, f, progress, done)
		})
		if conversionErr != nil {
			logger.Errorf("[convert] error converting scene %d: %v", t.Scene.ID, conversionErr)
			return conversionErr
		}
		progress.SetProcessed(1)

		// Close task queue
		taskQueue.Close()

		// Stop monitoring
		close(done)

		progress.ExecuteTask("Updating scene metadata", func() {
			if conversionErr == nil {
				// Update scene metadata here
				progress.SetProcessed(2)
			}
		})

		progress.ExecuteTask("Finalizing conversion", func() {
			if conversionErr == nil {
				progress.SetProcessed(3)
			}
		})

		if conversionErr == nil {
			logger.Infof("[convert] successfully converted scene %d to MP4", t.Scene.ID)
		} else {
			return conversionErr
		}
	} else {
		logger.Infof("[convert] scene %d does not need conversion", t.Scene.ID)
		progress.SetTotal(1)
		progress.SetProcessed(1)
	}

	return nil
}

// For backward compatibility
func (t *ConvertToMP4Task) Start(ctx context.Context) {
	progress := &job.Progress{}
	t.Execute(ctx, progress)
}

func (t *ConvertToMP4Task) needsConversion(f *models.VideoFile) bool {
	// If scene is broken, always allow conversion regardless of format
	if t.Scene.IsBroken {
		logger.Infof("[convert] scene is broken, allowing MP4 conversion regardless of current format")
		return true
	}

	// Always convert non-MP4 files to MP4 for better performance
	// This includes: avi, flv, mkv, mov, wmv, webm, etc.
	if f.Format != "mp4" {
		logger.Infof("[convert] file format %s needs conversion to MP4", f.Format)
		return true
	}

	// For MP4 files, check if video codec needs conversion
	if f.VideoCodec != "h264" {
		logger.Infof("[convert] MP4 file with codec %s needs conversion to H.264", f.VideoCodec)
		return true
	}

	// If it's already MP4 with H.264, no conversion needed
	logger.Infof("[convert] file is already MP4 with H.264, no conversion needed")
	return false
}

func (t *ConvertToMP4Task) convertToMP4(ctx context.Context, f *models.VideoFile, progress *job.Progress, done chan bool) error {
	tempDir := t.Config.GetGeneratedPath()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("convert_%d_%s.mp4", t.Scene.ID, t.Scene.GetHash(t.FileNamingAlgorithm)))

	// Create independent backup copy in temp directory
	backupTempDir := t.Config.GetTempPath()
	logger.Infof("[convert] Creating backup temp directory: %s", backupTempDir)
	if err := os.MkdirAll(backupTempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp backup directory %s: %w", backupTempDir, err)
	}
	// Use original filename for backup in temp
	originalFilename := filepath.Base(f.Path)
	backupTempFile := filepath.Join(backupTempDir, originalFilename)
	logger.Infof("[convert] Backup temp file path: %s", backupTempFile)

	// Create backup copy of ORIGINAL file in temp directory BEFORE conversion
	logger.Infof("[convert] Creating backup copy of original file from %s to %s", f.Path, backupTempFile)
	if err := t.copyFileContent(f.Path, backupTempFile); err != nil {
		return fmt.Errorf("failed to create backup copy of original file in temp: %w", err)
	}
	logger.Infof("[convert] Successfully created backup copy of original file in temp: %s", backupTempFile)

	// Get original file size for progress tracking
	originalFileInfo, err := os.Stat(f.Path)
	if err != nil {
		logger.Warnf("[convert] failed to get original file size: %v", err)
	} else {
		logger.Infof("[convert] original file size: %d bytes (%.2f MB)", originalFileInfo.Size(), float64(originalFileInfo.Size())/1024/1024)
	}

	// Start file size monitoring
	originalSize := int64(0)
	if originalFileInfo != nil {
		originalSize = originalFileInfo.Size()
	}

	// Start monitoring file size in a goroutine (using shared done channel)
	go t.monitorFileSize(tempFile, originalSize, progress, done)

	// Track if conversion was successful
	conversionSuccessful := false

	// Clean up temp files at the end
	defer func() {
		// Don't close done channel here - it's already closed in Execute method

		// Clean up main temp file only on failure
		if !conversionSuccessful {
			if _, err := os.Stat(tempFile); err == nil {
				if err := os.Remove(tempFile); err != nil {
					logger.Warnf("[convert] failed to remove temp file %s: %v", tempFile, err)
				} else {
					logger.Infof("[convert] cleaned up temp file: %s", tempFile)
				}
			}
		}
	}()

	if err := t.performConversionWithProgress(ctx, f.Path, tempFile, progress); err != nil {
		logger.Errorf("[convert] conversion failed: %v", err)
		return fmt.Errorf("conversion failed: %w", err)
	}

	if err := t.validateConvertedFile(tempFile); err != nil {
		return fmt.Errorf("converted file validation failed: %w", err)
	}

	// Backup copy of original file was already created before conversion

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
		logger.Infof("[convert] checking if temp file needs to be copied to existing file: %s", finalPath)

		// Only copy if paths are different (avoid copying file to itself)
		if tempFile != finalPath {
			logger.Infof("[convert] copying temp file content to existing file: %s -> %s", tempFile, finalPath)
			if err := t.copyFileContent(tempFile, finalPath); err != nil {
				return fmt.Errorf("failed to copy temp file content to existing file: %w", err)
			}
		} else {
			logger.Infof("[convert] temp file and final path are the same, no copy needed: %s", finalPath)
		}

		// Validate the updated file
		if err := t.validateConvertedFile(finalPath); err != nil {
			logger.Errorf("[convert] updated file validation failed: %v", err)
			return fmt.Errorf("updated file validation failed: %w", err)
		}

		logger.Infof("[convert] successfully updated existing file: %s", finalPath)
	} else {
		// New file was created, move temp file to final location
		finalPath := t.getFinalPath(newFile)
		logger.Infof("[convert] moving file from %s to %s", tempFile, finalPath)

		// Check if temp file exists
		if _, err := os.Stat(tempFile); err != nil {
			return fmt.Errorf("temp file does not exist: %w", err)
		}

		// Copy temp file to final location (works across different filesystems)
		logger.Infof("[convert] copying temp file to final location: %s -> %s", tempFile, finalPath)
		if err := t.copyFileContent(tempFile, finalPath); err != nil {
			return fmt.Errorf("failed to copy converted file to final location: %w", err)
		}

		// Remove temp file after successful copy
		if err := os.Remove(tempFile); err != nil {
			logger.Warnf("[convert] failed to remove temp file %s: %v", tempFile, err)
		} else {
			logger.Infof("[convert] removed temp file: %s", tempFile)
		}

		// Verify the file was moved successfully
		if _, err := os.Stat(finalPath); err != nil {
			return fmt.Errorf("final file does not exist after move: %w", err)
		}

		logger.Infof("[convert] successfully moved file to %s", finalPath)

		if err := t.updateFilePath(ctx, newFile, finalPath); err != nil {
			return fmt.Errorf("failed to update file path: %w", err)
		}

		// Validate the converted file before removing the original
		if err := t.validateConvertedFile(finalPath); err != nil {
			logger.Errorf("[convert] converted file validation failed, keeping original: %v", err)
			return fmt.Errorf("converted file validation failed: %w", err)
		}

		// Remove the original file only after successful validation
		originalPath := f.Path
		if err := os.Remove(originalPath); err != nil {
			logger.Warnf("[convert] failed to remove original file %s: %v", originalPath, err)
		} else {
			logger.Infof("[convert] removed original file: %s", originalPath)
		}

		// Delete the old file record from database
		if err := t.deleteOldFileRecord(ctx, f); err != nil {
			logger.Warnf("[convert] failed to delete old file record: %v", err)
		} else {
			logger.Infof("[convert] deleted old file record from database")
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
		logger.Warnf("[convert] failed to recalculate file hashes: %v", err)
	} else {
		logger.Infof("[convert] recalculated file hashes")
	}

	// Regenerate sprites with new hash after conversion
	logger.Infof("[convert] regenerating sprites for converted file")
	if err := t.regenerateSprites(ctx); err != nil {
		logger.Warnf("[convert] failed to regenerate sprites: %v", err)
		// Don't fail the conversion if sprite generation fails
	}

	// Generate VTT file for the new video if it doesn't exist
	if err := t.generateVTTFile(ctx, newFile, finalPath); err != nil {
		logger.Warnf("[convert] failed to generate VTT file: %v", err)
	} else {
		logger.Infof("[convert] generated VTT file")
	}

	// Clean up backup temp file only after all operations are successful
	if _, err := os.Stat(backupTempFile); err == nil {
		if err := os.Remove(backupTempFile); err != nil {
			logger.Warnf("[convert] failed to remove backup temp file %s: %v", backupTempFile, err)
		} else {
			logger.Infof("[convert] cleaned up backup temp file: %s", backupTempFile)
		}
	}

	// Mark conversion as successful - temp file will be moved, not deleted
	conversionSuccessful = true
	return nil
}

func (t *ConvertToMP4Task) monitorFileSize(tempFile string, originalSize int64, progress *job.Progress, done chan bool) {
	ticker := time.NewTicker(2 * time.Second) // Check every 2 seconds
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

					// Update progress bar based on file size
					progress.SetPercent(percent)

					logger.Infof("[convert] file size progress: %d/%d bytes (%.1f%%) - %.2f/%.2f MB",
						currentSize, originalSize, percent*100,
						float64(currentSize)/1024/1024, float64(originalSize)/1024/1024)
				} else {
					logger.Infof("[convert] current file size: %d bytes (%.2f MB)",
						currentSize, float64(currentSize)/1024/1024)
				}
			}
		}
	}
}

func (t *ConvertToMP4Task) monitorFileSizeWithStatusUpdate(tempFile string, originalSize int64, progress *job.Progress, done chan bool) {
	ticker := time.NewTicker(2 * time.Second) // Check every 2 seconds
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

					// Update progress bar based on file size
					progress.SetPercent(percent)

					logger.Infof("[convert] file size progress: %d/%d bytes (%.1f%%) - %.2f/%.2f MB",
						currentSize, originalSize, percent*100,
						float64(currentSize)/1024/1024, float64(originalSize)/1024/1024)
				} else {
					logger.Infof("[convert] current file size: %d bytes (%.2f MB)",
						currentSize, float64(currentSize)/1024/1024)
				}
			}
		}
	}
}

func (t *ConvertToMP4Task) monitorFileSizeWithQueue(tempFile string, originalSize int64, taskQueue *job.TaskQueue, done chan bool) {
	ticker := time.NewTicker(6 * time.Second) // Check every 6 seconds
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

					// Create a task with dynamic description
					statusText := fmt.Sprintf("Converting video to MP4 - %.1f%% (%.2f/%.2f MB)",
						percent*100,
						float64(currentSize)/1024/1024,
						float64(originalSize)/1024/1024)

					taskQueue.Add(statusText, func(ctx context.Context) {
						// This task will update the description
						// Add a longer delay to make the status visible
						time.Sleep(4 * time.Second)
					})

					logger.Infof("[convert] file size progress: %d/%d bytes (%.1f%%) - %.2f/%.2f MB",
						currentSize, originalSize, percent*100,
						float64(currentSize)/1024/1024, float64(originalSize)/1024/1024)
				} else {
					statusText := fmt.Sprintf("Converting video to MP4 - %.2f MB",
						float64(currentSize)/1024/1024)

					taskQueue.Add(statusText, func(ctx context.Context) {
						// This task will update the description
						// Add a longer delay to make the status visible
						time.Sleep(4 * time.Second)
					})

					logger.Infof("[convert] current file size: %d bytes (%.2f MB)",
						currentSize, float64(currentSize)/1024/1024)
				}
			}
		}
	}
}

func (t *ConvertToMP4Task) performConversion(ctx context.Context, inputPath, outputPath string) error {
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(inputPath)
	if err != nil {
		return fmt.Errorf("error reading video file: %w", err)
	}

	w, h := videoFile.Width, videoFile.Height
	transcodeSize := t.Config.GetMaxTranscodeSize()

	if transcodeSize.GetMaxResolution() > 0 {
		w, h = videoFile.TranscodeScale(transcodeSize.GetMaxResolution())
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
		"-crf", "18",
	)

	audioArgs := ffmpeg.Args{
		"-ac", "2", // Explicitly specify stereo output
		"-ar", "44100",
		"-ab", "128k",
		"-strict", "-2",
		// More aggressive parameters for problematic audio codecs
		"-ignore_errors", "1",
		"-fflags", "+genpts+igndts",
		"-avoid_negative_ts", "make_zero",
		"-async", "1",
		"-err_detect", "ignore_err", // Ignore minor errors in data
	}

	// Add extra input args for problematic video files (AVI, FLV, etc.)
	extraInputArgs := append(t.Config.GetTranscodeInputArgs(),
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",
	)

	// For non-MP4 files (AVI, FLV, etc.), always convert audio to AAC instead of copying
	// These formats often have codecs that aren't compatible with MP4 container
	audioCodec := ffmpeg.AudioCodecAAC
	audioArgs = ffmpeg.Args{
		"-ac", "2",
		"-ar", "44100",
		"-ab", "128k",
		"-strict", "-2",
	}

	args := transcoder.Transcode(inputPath, transcoder.TranscodeOptions{
		OutputPath:      outputPath,
		VideoCodec:      ffmpeg.VideoCodecLibX264,
		VideoArgs:       videoArgs,
		AudioCodec:      audioCodec,
		AudioArgs:       audioArgs,
		Format:          ffmpeg.FormatMP4,
		ExtraInputArgs:  extraInputArgs,
		ExtraOutputArgs: t.Config.GetTranscodeOutputArgs(),
	})

	return t.FFMpeg.Generate(ctx, args)
}

func (t *ConvertToMP4Task) performConversionWithProgress(ctx context.Context, inputPath, outputPath string, progress *job.Progress) error {
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(inputPath)
	if err != nil {
		return fmt.Errorf("error reading video file: %w", err)
	}

	w, h := videoFile.Width, videoFile.Height
	transcodeSize := t.Config.GetMaxTranscodeSize()

	if transcodeSize.GetMaxResolution() > 0 {
		w, h = videoFile.TranscodeScale(transcodeSize.GetMaxResolution())
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
		"-crf", "18",
	)

	audioArgs := ffmpeg.Args{
		"-ac", "2", // Explicitly specify stereo output
		"-ar", "44100",
		"-ab", "128k",
		"-strict", "-2",
		// More aggressive parameters for problematic audio codecs
		"-ignore_errors", "1",
		"-fflags", "+genpts+igndts",
		"-avoid_negative_ts", "make_zero",
		"-async", "1",
		"-err_detect", "ignore_err", // Ignore minor errors in data
	}

	// Add extra input args for problematic video files (AVI, FLV, etc.)
	extraInputArgs := append(t.Config.GetTranscodeInputArgs(),
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",
	)

	// For non-MP4 files (AVI, FLV, etc.), always convert audio to AAC instead of copying
	// These formats often have codecs that aren't compatible with MP4 container
	audioCodec := ffmpeg.AudioCodecAAC
	audioArgs = ffmpeg.Args{
		"-ac", "2",
		"-ar", "44100",
		"-ab", "128k",
		"-strict", "-2",
	}

	args := transcoder.Transcode(inputPath, transcoder.TranscodeOptions{
		OutputPath:      outputPath,
		VideoCodec:      ffmpeg.VideoCodecLibX264,
		VideoArgs:       videoArgs,
		AudioCodec:      audioCodec,
		AudioArgs:       audioArgs,
		Format:          ffmpeg.FormatMP4,
		ExtraInputArgs:  extraInputArgs,
		ExtraOutputArgs: t.Config.GetTranscodeOutputArgs(),
	})

	logger.Infof("[convert] running ffmpeg command: %v", args)
	logger.Infof("[convert] video duration: %.2f seconds", videoFile.FileDuration)
	return t.FFMpeg.GenerateWithProgress(ctx, args, progress, videoFile.FileDuration)
}

func (t *ConvertToMP4Task) performConversionWithStandardAAC(ctx context.Context, inputPath, outputPath string, progress *job.Progress) error {
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(inputPath)
	if err != nil {
		return fmt.Errorf("error reading video file: %w", err)
	}

	w, h := videoFile.Width, videoFile.Height
	transcodeSize := t.Config.GetMaxTranscodeSize()

	if transcodeSize.GetMaxResolution() > 0 {
		w, h = videoFile.TranscodeScale(transcodeSize.GetMaxResolution())
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
		"-crf", "18",
	)

	// Use standard AAC codec with robust parameters
	audioArgs := ffmpeg.Args{
		"-ac", "2", // Explicitly specify stereo output
		"-ar", "44100",
		"-ab", "128k",
		"-strict", "-2",
		"-err_detect", "ignore_err", // Ignore minor errors in data
		"-ignore_errors", "1",
		"-fflags", "+genpts+igndts",
		"-avoid_negative_ts", "make_zero",
		"-async", "1",
	}

	// Add extra input args for problematic video files (AVI, FLV, etc.)
	extraInputArgs := append(t.Config.GetTranscodeInputArgs(),
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",
	)

	args := transcoder.Transcode(inputPath, transcoder.TranscodeOptions{
		OutputPath:      outputPath,
		VideoCodec:      ffmpeg.VideoCodecLibX264,
		VideoArgs:       videoArgs,
		AudioCodec:      ffmpeg.AudioCodecAAC, // Use standard AAC
		AudioArgs:       audioArgs,
		Format:          ffmpeg.FormatMP4,
		ExtraInputArgs:  extraInputArgs,
		ExtraOutputArgs: t.Config.GetTranscodeOutputArgs(),
	})

	logger.Infof("[convert] running ffmpeg command (AAC): %v", args)
	logger.Infof("[convert] video duration: %.2f seconds", videoFile.FileDuration)
	return t.FFMpeg.GenerateWithProgress(ctx, args, progress, videoFile.FileDuration)
}

func (t *ConvertToMP4Task) performConversionWithoutAudio(ctx context.Context, inputPath, outputPath string, progress *job.Progress) error {
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(inputPath)
	if err != nil {
		return fmt.Errorf("error reading video file: %w", err)
	}

	w, h := videoFile.Width, videoFile.Height
	transcodeSize := t.Config.GetMaxTranscodeSize()

	if transcodeSize.GetMaxResolution() > 0 {
		w, h = videoFile.TranscodeScale(transcodeSize.GetMaxResolution())
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
		"-crf", "18",
	)

	// Add extra input args for problematic video files (AVI, FLV, etc.)
	extraInputArgs := append(t.Config.GetTranscodeInputArgs(),
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",
	)

	args := transcoder.Transcode(inputPath, transcoder.TranscodeOptions{
		OutputPath:      outputPath,
		VideoCodec:      ffmpeg.VideoCodecLibX264,
		VideoArgs:       videoArgs,
		AudioCodec:      "", // No audio
		Format:          ffmpeg.FormatMP4,
		ExtraInputArgs:  extraInputArgs,
		ExtraOutputArgs: t.Config.GetTranscodeOutputArgs(),
	})

	logger.Infof("[convert] running ffmpeg command (no audio): %v", args)
	logger.Infof("[convert] video duration: %.2f seconds", videoFile.FileDuration)
	return t.FFMpeg.GenerateWithProgress(ctx, args, progress, videoFile.FileDuration)
}

func (t *ConvertToMP4Task) validateConvertedFile(filePath string) error {
	// Check if file exists and is readable
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("converted file does not exist or is not accessible: %w", err)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("converted file is empty")
	}

	logger.Infof("[convert] validating converted file: %s (size: %d bytes)", filePath, fileInfo.Size())

	// Probe the file with FFProbe
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to probe converted file: %w", err)
	}

	// Validate duration
	if videoFile.FileDuration <= 0 {
		return fmt.Errorf("converted file has invalid duration: %f", videoFile.FileDuration)
	}

	logger.Infof("[convert] converted file duration: %.2f seconds", videoFile.FileDuration)

	// Validate video codec
	if videoFile.VideoCodec == "" {
		return fmt.Errorf("converted file has no video stream")
	}

	if videoFile.VideoCodec != "h264" {
		return fmt.Errorf("converted file has wrong video codec: %s (expected h264)", videoFile.VideoCodec)
	}

	logger.Infof("[convert] converted file video codec: %s", videoFile.VideoCodec)

	// Validate audio codec (should be aac or empty)
	if videoFile.AudioCodec != "" && videoFile.AudioCodec != "aac" {
		logger.Warnf("[convert] converted file has unexpected audio codec: %s", videoFile.AudioCodec)
	}

	// Format validation is handled by file extension (.mp4)

	// Validate resolution
	if videoFile.Width <= 0 || videoFile.Height <= 0 {
		return fmt.Errorf("converted file has invalid resolution: %dx%d", videoFile.Width, videoFile.Height)
	}

	logger.Infof("[convert] converted file resolution: %dx%d", videoFile.Width, videoFile.Height)

	logger.Infof("[convert] converted file validation successful")
	return nil
}

func (t *ConvertToMP4Task) createNewVideoFile(ctx context.Context, filePath string) (*models.VideoFile, bool, error) {
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(filePath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to probe file: %w", err)
	}

	// Get the original file to copy its parent_folder_id
	originalFile, err := t.Repository.File.FindByPath(ctx, t.Scene.Files.Primary().Path)
	if err != nil {
		return nil, false, fmt.Errorf("failed to find original file: %w", err)
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
		logger.Infof("[convert] file %s already exists in folder %d, updating existing file", properBasename, originalFile.Base().ParentFolderID)

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

		// Update the existing file with new metadata - use final path, not temp path
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
			logger.Infof("[convert] associating existing file %d with scene %d", existingVideoFile.ID, t.Scene.ID)
			fileIDs := []models.FileID{existingVideoFile.ID}
			if err := t.Repository.Scene.AssignFiles(ctx, t.Scene.ID, fileIDs); err != nil {
				return nil, false, fmt.Errorf("failed to associate existing file with scene: %w", err)
			}
		}

		logger.Infof("[convert] updated existing file %d with new MP4 metadata", existingVideoFile.ID)
		return existingVideoFile, true, nil
	}

	// No existing file found, create new one
	newFile := &models.VideoFile{
		BaseFile: &models.BaseFile{
			Path:           filePath, // This will be updated later in updateFilePath
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

func (t *ConvertToMP4Task) updateSceneWithNewFile(ctx context.Context, newFile *models.VideoFile) error {
	// Associate the new file with the scene
	fileIDs := []models.FileID{newFile.ID}
	if err := t.Repository.Scene.AssignFiles(ctx, t.Scene.ID, fileIDs); err != nil {
		return fmt.Errorf("failed to associate file with scene: %w", err)
	}

	// Update scene to remove broken status and set new primary file
	scenePartial := models.NewScenePartial()
	scenePartial.IsBroken = models.NewOptionalBool(false) // Remove broken status
	scenePartial.PrimaryFileID = &newFile.ID              // Set new primary file

	// Update scene in database
	_, err := t.Repository.Scene.UpdatePartial(ctx, t.Scene.ID, scenePartial)
	if err != nil {
		return fmt.Errorf("failed to update scene metadata: %w", err)
	}

	logger.Infof("[convert] updated scene %d metadata and removed broken status", t.Scene.ID)
	return nil
}

func (t *ConvertToMP4Task) getFinalPath(file *models.VideoFile) string {
	// Get the original file path and directory
	originalFile := t.Scene.Files.Primary()
	originalPath := originalFile.Path
	originalDir := filepath.Dir(originalPath)
	originalBasename := originalFile.Base().Basename

	// Create new filename with .mp4 extension
	ext := filepath.Ext(originalBasename)
	nameWithoutExt := strings.TrimSuffix(originalBasename, ext)
	newBasename := nameWithoutExt + ".mp4"

	// Ensure the original directory exists
	if err := os.MkdirAll(originalDir, 0755); err != nil {
		logger.Warnf("[convert] failed to ensure original directory exists %s: %v", originalDir, err)
	}

	logger.Infof("[convert] original path: %s", originalPath)
	logger.Infof("[convert] original basename: %s, new basename: %s", originalBasename, newBasename)
	logger.Infof("[convert] original directory: %s", originalDir)

	// Return the full path in the same directory as original file
	finalPath := filepath.Join(originalDir, newBasename)
	logger.Infof("[convert] final path: %s", finalPath)
	return finalPath
}

func (t *ConvertToMP4Task) updateFilePath(ctx context.Context, file *models.VideoFile, newPath string) error {
	// Update file path in database
	file.Base().Path = newPath
	file.Base().Basename = filepath.Base(newPath)

	err := t.Repository.File.Update(ctx, file)
	if err != nil {
		return fmt.Errorf("failed to update file path: %w", err)
	}

	logger.Infof("[convert] updated file path to %s", newPath)
	return nil
}

func (t *ConvertToMP4Task) deleteOldFileRecord(ctx context.Context, oldFile *models.VideoFile) error {
	// Delete the old file record from database
	if err := t.Repository.File.Destroy(ctx, oldFile.ID); err != nil {
		return fmt.Errorf("failed to delete old file record: %w", err)
	}

	logger.Infof("[convert] deleted old file record with ID %d", oldFile.ID)
	return nil
}

func (t *ConvertToMP4Task) recalculateFileHashes(ctx context.Context, file *models.VideoFile, filePath string) error {
	// Recalculate file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	file.Base().Size = fileInfo.Size()
	file.Base().ModTime = fileInfo.ModTime()

	// Create a file opener for the new file
	opener := &osFileOpener{path: filePath}

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
			logger.Warnf("[convert] failed to calculate phash: %v", err)
			// Don't fail the entire operation if phash calculation fails
		} else {
			phashInt := int64(*phash)
			// Add phash to fingerprints
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
	logger.Infof("[convert] recalculated hashes - checksum: %v, oshash: %v", checksum, oshash)
	return nil
}

func (t *ConvertToMP4Task) generateVTTFile(ctx context.Context, file *models.VideoFile, filePath string) error {
	// Get updated scene from database with new hash
	// Use the existing transaction context instead of creating a new one
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
		logger.Infof("[convert] VTT file already exists: %s", vttPath)
		return nil
	}

	// Check if sprite image exists
	spritePath := t.Paths.Scene.GetSpriteImageFilePath(sceneHash)
	if _, err := os.Stat(spritePath); err != nil {
		logger.Infof("[convert] sprite image does not exist, skipping VTT generation: %s", spritePath)
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
	// Use default values similar to sprite generation
	stepSize := 10.0 // Default step size in seconds
	if file.Duration > 0 {
		stepSize = file.Duration / 100.0 // Divide video into ~100 segments
	}

	logger.Infof("[convert] generating VTT file: %s", vttPath)
	if err := generator.SpriteVTT(ctx, vttPath, spritePath, stepSize); err != nil {
		return fmt.Errorf("failed to generate VTT file: %w", err)
	}

	logger.Infof("[convert] successfully generated VTT file: %s", vttPath)
	return nil
}

// isFileAssociatedWithScene checks if a file is already associated with the current scene
func (t *ConvertToMP4Task) isFileAssociatedWithScene(ctx context.Context, fileID models.FileID) (bool, error) {
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

// copyFileContent copies the content from source to destination file
func (t *ConvertToMP4Task) copyFileContent(src, dst string) error {
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

	logger.Infof("[convert] successfully copied file content from %s to %s", src, dst)
	return nil
}

// regenerateSprites regenerates sprites for the scene after conversion
// NOTE: This function expects to be called within an existing transaction context
func (t *ConvertToMP4Task) regenerateSprites(ctx context.Context) error {
	// Get updated scene from database with new hash
	// Use the existing transaction context instead of creating a new one
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

	spriteTask := GenerateSpriteTask{
		Scene:               *updatedScene, // Use updated scene with new hash
		Overwrite:           true,          // Force regeneration with new hash
		fileNamingAlgorithm: t.FileNamingAlgorithm,
	}

	// Run sprite generation
	spriteTask.Start(ctx)
	logger.Infof("[convert] regenerated sprites for scene %d with updated hash", t.Scene.ID)
	return nil
}
