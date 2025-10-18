package manager

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// hlsFileOpener implements file.Opener for OS files
type hlsFileOpener struct {
	path string
}

func (o *hlsFileOpener) Open() (io.ReadCloser, error) {
	return os.Open(o.path)
}

type ConvertHLSToMP4Task struct {
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

func (t *ConvertHLSToMP4Task) GetDescription() string {
	return fmt.Sprintf("Converting HLS video %s to MP4", t.Scene.Path)
}

func (t *ConvertHLSToMP4Task) Execute(ctx context.Context, progress *job.Progress) error {
	scene := &t.Scene
	pf := scene.Files.Primary()
	if pf == nil {
		return fmt.Errorf("scene has no primary file")
	}

	if t.needsConversion(pf) {
		logger.Infof("[convert] converting HLS scene %d to MP4", scene.ID)

		progress.SetTotal(4)
		progress.SetProcessed(0)

		var conversionErr error

		// Get original file size for display
		originalFileInfo, err := os.Stat(pf.Path)
		if err == nil {
			logger.Infof("[convert] original HLS file size: %d bytes (%.2f MB)", originalFileInfo.Size(), float64(originalFileInfo.Size())/1024/1024)
		}

		// Start file size monitoring
		originalSize := int64(0)
		if originalFileInfo != nil {
			originalSize = originalFileInfo.Size()
		}

		// Start monitoring file size in a goroutine
		done := make(chan bool)
		tempFile := filepath.Join(t.Config.GetGeneratedPath(), fmt.Sprintf("convert_hls_%d_%s.mp4", scene.ID, scene.GetHash(t.FileNamingAlgorithm)))
		go t.monitorFileSize(tempFile, originalSize, progress, done)

		// Start monitoring in a goroutine
		go t.monitorFileSizeWithStatusUpdate(tempFile, originalSize, progress, done)

		// Create a task queue for dynamic status updates
		taskQueue := job.NewTaskQueue(ctx, progress, 100, 1)
		go t.monitorFileSizeWithQueue(tempFile, originalSize, taskQueue, done)

		// Wrap conversion in transaction
		conversionErr = t.Repository.WithTxn(ctx, func(ctx context.Context) error {
			return t.convertToMP4(ctx, pf, progress)
		})
		if conversionErr != nil {
			logger.Errorf("[convert] error converting HLS scene %d: %v", scene.ID, conversionErr)
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

		progress.ExecuteTask("Finalizing conversion", func() {
			if conversionErr == nil {
				progress.SetProcessed(3)
			}
		})

		progress.ExecuteTask("Cleaning up temporary files", func() {
			if conversionErr == nil {
				progress.SetProcessed(4)
			}
		})

		if conversionErr == nil {
			logger.Infof("[convert] successfully converted HLS scene %d to MP4", scene.ID)
		} else {
			return conversionErr
		}
	} else {
		logger.Infof("[convert] scene %d does not need HLS conversion", scene.ID)
		progress.SetTotal(1)
		progress.SetProcessed(1)
	}

	return nil
}

// For backward compatibility
func (t *ConvertHLSToMP4Task) Start(ctx context.Context) {
	progress := &job.Progress{}
	t.Execute(ctx, progress)
}

func (t *ConvertHLSToMP4Task) needsConversion(f *models.VideoFile) bool {
	// If scene is broken, always allow HLS conversion regardless of format
	if t.Scene.IsBroken {
		logger.Infof("[convert] scene is broken, allowing HLS conversion regardless of current format")
		return true
	}

	// Check if it's actually an HLS video
	audioCodec := ffmpeg.MissingUnsupported
	if f.AudioCodec != "" {
		audioCodec = ffmpeg.ProbeAudioCodec(f.AudioCodec)
	}

	container, err := GetVideoFileContainer(f)
	if err != nil {
		logger.Warnf("[convert] error getting container for scene %d: %v", t.Scene.ID, err)
		return false
	}

	var videoCodec string
	if f.VideoCodec != "" {
		videoCodec = f.VideoCodec
	}

	if !ffmpeg.IsHLSVideo(videoCodec, audioCodec, container, f.Duration) {
		logger.Infof("[convert] scene %d is not detected as HLS video", t.Scene.ID)
		return false
	}

	logger.Infof("[convert] HLS video detected for scene %d, needs conversion", t.Scene.ID)
	return true
}

func (t *ConvertHLSToMP4Task) convertToMP4(ctx context.Context, f *models.VideoFile, progress *job.Progress) error {
	tempDir := t.Config.GetGeneratedPath()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("convert_hls_%d_%s.mp4", t.Scene.ID, t.Scene.GetHash(t.FileNamingAlgorithm)))

	// Create independent backup copy in temp directory
	backupTempDir := t.Config.GetTempPath()
	logger.Infof("[convert] Creating HLS backup temp directory: %s", backupTempDir)
	if err := os.MkdirAll(backupTempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp backup directory %s: %w", backupTempDir, err)
	}
	// Use original filename for backup in temp
	originalFilename := filepath.Base(f.Path)
	backupTempFile := filepath.Join(backupTempDir, originalFilename)
	logger.Infof("[convert] HLS backup temp file path: %s", backupTempFile)

	// Create backup copy of ORIGINAL HLS file in temp directory BEFORE conversion
	logger.Infof("[convert] Creating backup copy of original HLS file from %s to %s", f.Path, backupTempFile)
	if err := t.copyFileContent(f.Path, backupTempFile); err != nil {
		return fmt.Errorf("failed to create backup copy of original HLS file in temp: %w", err)
	}
	logger.Infof("[convert] Successfully created backup copy of original HLS file in temp: %s", backupTempFile)

	// Get original file size for progress tracking
	originalFileInfo, err := os.Stat(f.Path)
	if err != nil {
		logger.Warnf("[convert] failed to get original HLS file size: %v", err)
	} else {
		logger.Infof("[convert] original HLS file size: %d bytes (%.2f MB)", originalFileInfo.Size(), float64(originalFileInfo.Size())/1024/1024)
	}

	// Start file size monitoring
	originalSize := int64(0)
	if originalFileInfo != nil {
		originalSize = originalFileInfo.Size()
	}

	// Start monitoring file size in a goroutine
	done := make(chan bool)
	go t.monitorFileSize(tempFile, originalSize, progress, done)

	// Track if conversion was successful
	conversionSuccessful := false

	// Always clean up backup temp file at the end
	defer func() {
		// Don't close done channel here - it's already closed in Execute method

		// Clean up backup temp file regardless of success/failure
		if _, err := os.Stat(backupTempFile); err == nil {
			if err := os.Remove(backupTempFile); err != nil {
				logger.Warnf("[convert] failed to remove backup temp HLS file %s: %v", backupTempFile, err)
			} else {
				logger.Infof("[convert] cleaned up backup temp HLS file: %s", backupTempFile)
			}
		}

		// Clean up main temp file only on failure
		if !conversionSuccessful {
			if _, err := os.Stat(tempFile); err == nil {
				if err := os.Remove(tempFile); err != nil {
					logger.Warnf("[convert] failed to remove temp HLS file %s: %v", tempFile, err)
				} else {
					logger.Infof("[convert] cleaned up temp HLS file: %s", tempFile)
				}
			}
		}
	}()

	if err := t.performConversionWithProgress(ctx, f.Path, tempFile, progress); err != nil {
		logger.Errorf("[convert] HLS conversion failed: %v", err)
		return fmt.Errorf("HLS conversion failed: %w", err)
	}

	if err := t.validateConvertedFile(tempFile); err != nil {
		return fmt.Errorf("converted HLS file validation failed: %w", err)
	}

	// Backup copy of original HLS file was already created before conversion

	newFile, err := t.createNewVideoFile(ctx, tempFile)
	if err != nil {
		return fmt.Errorf("failed to create new video file: %w", err)
	}

	if err := t.updateSceneWithNewFile(ctx, newFile); err != nil {
		return fmt.Errorf("failed to update scene with new file: %w", err)
	}

	// Move the converted file to replace the original HLS file
	originalPath := f.Path
	logger.Infof("[convert] moving converted HLS file from %s to %s", tempFile, originalPath)

	// Check if temp file exists
	if _, err := os.Stat(tempFile); err != nil {
		return fmt.Errorf("temp HLS file does not exist: %w", err)
	}

	// Remove the original HLS file first
	if err := os.Remove(originalPath); err != nil {
		logger.Warnf("[convert] failed to remove original HLS file %s: %v", originalPath, err)
	}

	// Move the converted file to the original location
	if err := os.Rename(tempFile, originalPath); err != nil {
		return fmt.Errorf("failed to move converted HLS file to original location: %w", err)
	}

	// Verify the file was moved successfully
	if _, err := os.Stat(originalPath); err != nil {
		return fmt.Errorf("converted HLS file does not exist after move: %w", err)
	}

	logger.Infof("[convert] successfully replaced HLS file with MP4 at %s", originalPath)

	// Validate the converted file
	if err := t.validateConvertedFile(originalPath); err != nil {
		logger.Errorf("[convert] converted HLS file validation failed: %v", err)
		return fmt.Errorf("converted HLS file validation failed: %w", err)
	}

	// Recalculate hashes for the updated file
	if err := t.recalculateFileHashes(ctx, newFile, originalPath); err != nil {
		logger.Warnf("[convert] failed to recalculate HLS file hashes: %v", err)
	} else {
		logger.Infof("[convert] recalculated HLS file hashes")
	}

	// Regenerate sprites with new hash after HLS conversion
	logger.Infof("[convert] regenerating sprites for converted HLS file")
	if err := t.regenerateSprites(ctx); err != nil {
		logger.Warnf("[convert] failed to regenerate HLS sprites: %v", err)
		// Don't fail the conversion if sprite generation fails
	}

	// Generate VTT file for the updated video if it doesn't exist
	if err := t.generateVTTFile(ctx, newFile, originalPath); err != nil {
		logger.Warnf("[convert] failed to generate VTT file for HLS: %v", err)
	} else {
		logger.Infof("[convert] generated VTT file for HLS")
	}

	// Mark conversion as successful - temp file will be moved, not deleted
	conversionSuccessful = true
	return nil
}

func (t *ConvertHLSToMP4Task) getHardwareCodecForConversion() *ffmpeg.VideoCodec {
	codecs := []ffmpeg.VideoCodec{
		ffmpeg.VideoCodecN264,
		ffmpeg.VideoCodecI264,
		ffmpeg.VideoCodecV264,
		ffmpeg.VideoCodecA264,
	}

	for _, codec := range codecs {
		logger.Infof("[convert] testing hardware codec for HLS: %s (%s)", codec.Name, codec.CodeName)
		if t.testHardwareCodec(codec) {
			logger.Infof("[convert] ✓ hardware codec %s is available for HLS", codec.Name)
			return &codec
		}
	}

	logger.Infof("[convert] no hardware codec available for HLS")
	return nil
}

func (t *ConvertHLSToMP4Task) testHardwareCodec(codec ffmpeg.VideoCodec) bool {
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

func (t *ConvertHLSToMP4Task) getVideoArgsForCodec(codec ffmpeg.VideoCodec, w, h int) ffmpeg.Args {
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

func (t *ConvertHLSToMP4Task) performConversionWithProgress(ctx context.Context, inputPath, outputPath string, progress *job.Progress) error {
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(inputPath)
	if err != nil {
		return fmt.Errorf("error reading HLS video file: %w", err)
	}

	w, h := videoFile.Width, videoFile.Height
	transcodeSize := t.Config.GetMaxTranscodeSize()

	if transcodeSize.GetMaxResolution() > 0 {
		w, h = videoFile.TranscodeScale(transcodeSize.GetMaxResolution())
	}

	audioArgs := ffmpeg.Args{
		"-c:a", "aac",
		"-ac", "2",
		"-ar", "44100",
		"-ab", "96k",
		"-strict", "-2",
		"-async", "1",
		"-af", "aresample=async=1",
		"-fflags", "+genpts+igndts",
		"-avoid_negative_ts", "make_zero",
	}

	extraInputArgs := append(t.Config.GetTranscodeInputArgs(),
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",
	)

	extraOutputArgs := append(t.Config.GetTranscodeOutputArgs(),
		"-movflags", "+faststart",
	)

	hwCodec := t.getHardwareCodecForConversion()

	if hwCodec != nil {
		logger.Infof("[convert] attempting hardware acceleration for HLS with codec: %s", hwCodec.Name)

		videoArgs := t.getVideoArgsForCodec(*hwCodec, w, h)

		args := transcoder.Transcode(inputPath, transcoder.TranscodeOptions{
			OutputPath:      outputPath,
			VideoCodec:      *hwCodec,
			VideoArgs:       videoArgs,
			AudioCodec:      ffmpeg.AudioCodecAAC,
			AudioArgs:       audioArgs,
			Format:          ffmpeg.FormatMP4,
			ExtraInputArgs:  extraInputArgs,
			ExtraOutputArgs: extraOutputArgs,
		})

		logger.Infof("[convert] running hardware-accelerated ffmpeg command for HLS: %v", args)
		logger.Infof("[convert] HLS video duration: %.2f seconds", videoFile.FileDuration)

		err := t.FFMpeg.GenerateWithProgress(ctx, args, progress, videoFile.FileDuration)
		if err == nil {
			logger.Infof("[convert] hardware acceleration successful for HLS")
			return nil
		}

		logger.Warnf("[convert] hardware acceleration failed for HLS: %v, falling back to software encoding", err)

		if _, removeErr := os.Stat(outputPath); removeErr == nil {
			os.Remove(outputPath)
		}
	} else {
		logger.Infof("[convert] no hardware acceleration available for HLS, using software encoding")
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

	args := transcoder.Transcode(inputPath,
		transcoder.TranscodeOptions{
			VideoCodec:      ffmpeg.VideoCodecLibX264,
			AudioCodec:      ffmpeg.AudioCodecAAC,
			VideoArgs:       videoArgs,
			AudioArgs:       audioArgs,
			ExtraInputArgs:  extraInputArgs,
			ExtraOutputArgs: extraOutputArgs,
			Format:          "mp4",
			OutputPath:      outputPath,
		},
	)

	logger.Infof("[convert] running software ffmpeg command for HLS: %v", args)
	logger.Infof("[convert] HLS video duration: %.2f seconds", videoFile.FileDuration)
	return t.FFMpeg.GenerateWithProgress(ctx, args, progress, videoFile.FileDuration)
}

func (t *ConvertHLSToMP4Task) validateConvertedFile(filePath string) error {
	// Check if file exists and is readable
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("converted HLS file does not exist or is not accessible: %w", err)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("converted HLS file is empty")
	}

	logger.Infof("[convert] validating converted HLS file: %s (size: %d bytes)", filePath, fileInfo.Size())

	// Probe the file with FFProbe
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to probe converted HLS file: %w", err)
	}

	// Validate duration
	if videoFile.FileDuration <= 0 {
		return fmt.Errorf("converted HLS file has invalid duration: %f", videoFile.FileDuration)
	}

	logger.Infof("[convert] converted HLS file duration: %.2f seconds", videoFile.FileDuration)

	// Validate video codec
	if videoFile.VideoCodec == "" {
		return fmt.Errorf("converted HLS file has no video stream")
	}

	if videoFile.VideoCodec != "h264" {
		return fmt.Errorf("converted HLS file has wrong video codec: %s (expected h264)", videoFile.VideoCodec)
	}

	logger.Infof("[convert] converted HLS file video codec: %s", videoFile.VideoCodec)

	// Validate audio codec (should be aac or empty)
	if videoFile.AudioCodec != "" && videoFile.AudioCodec != "aac" {
		logger.Warnf("[convert] converted HLS file has unexpected audio codec: %s", videoFile.AudioCodec)
	}

	// Format validation is handled by file extension (.mp4)

	// Validate resolution
	if videoFile.Width <= 0 || videoFile.Height <= 0 {
		return fmt.Errorf("converted HLS file has invalid resolution: %dx%d", videoFile.Width, videoFile.Height)
	}

	logger.Infof("[convert] converted HLS file resolution: %dx%d", videoFile.Width, videoFile.Height)

	logger.Infof("[convert] converted HLS file validation successful")
	return nil
}

func (t *ConvertHLSToMP4Task) createNewVideoFile(ctx context.Context, filePath string) (*models.VideoFile, error) {
	ffprobe := t.FFProbe
	videoFile, err := ffprobe.NewVideoFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to probe converted file: %w", err)
	}

	// Get the original file to update it instead of creating new one
	originalFile, err := t.Repository.File.FindByPath(ctx, t.Scene.Files.Primary().Path)
	if err != nil {
		return nil, fmt.Errorf("failed to find original HLS file: %w", err)
	}

	// Cast to VideoFile to access video-specific fields
	originalVideoFile, ok := originalFile.(*models.VideoFile)
	if !ok {
		return nil, fmt.Errorf("original file is not a video file")
	}

	// Update the existing file with new metadata
	originalVideoFile.Base().Path = filePath
	originalVideoFile.Base().Size = videoFile.Size
	originalVideoFile.Base().ModTime = time.Now() // Use current time since videoFile doesn't have ModTime
	originalVideoFile.Base().UpdatedAt = time.Now()

	// Update video-specific metadata
	originalVideoFile.Duration = videoFile.FileDuration
	originalVideoFile.VideoCodec = videoFile.VideoCodec
	originalVideoFile.AudioCodec = videoFile.AudioCodec
	originalVideoFile.Width = videoFile.Width
	originalVideoFile.Height = videoFile.Height
	originalVideoFile.FrameRate = videoFile.FrameRate
	originalVideoFile.BitRate = videoFile.Bitrate
	originalVideoFile.Format = "mp4"

	// Update the file in database
	err = t.Repository.File.Update(ctx, originalVideoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to update HLS video file in database: %w", err)
	}

	logger.Infof("[convert] updated existing HLS file %d with MP4 metadata", originalVideoFile.ID)
	return originalVideoFile, nil
}

func (t *ConvertHLSToMP4Task) updateSceneWithNewFile(ctx context.Context, newFile *models.VideoFile) error {
	// Ensure the file is associated with the scene
	fileIDs := []models.FileID{newFile.ID}
	if err := t.Repository.Scene.AssignFiles(ctx, t.Scene.ID, fileIDs); err != nil {
		return fmt.Errorf("failed to associate HLS file with scene: %w", err)
	}

	// Update scene to remove broken status and ensure primary file is set
	scenePartial := models.NewScenePartial()
	scenePartial.IsBroken = models.NewOptionalBool(false) // Remove broken status
	scenePartial.PrimaryFileID = &newFile.ID              // Set primary file ID

	// Update scene in database
	_, err := t.Repository.Scene.UpdatePartial(ctx, t.Scene.ID, scenePartial)
	if err != nil {
		return fmt.Errorf("failed to update HLS scene metadata: %w", err)
	}

	logger.Infof("[convert] updated HLS scene %d metadata and removed broken status", t.Scene.ID)
	return nil
}

func (t *ConvertHLSToMP4Task) recalculateFileHashes(ctx context.Context, file *models.VideoFile, filePath string) error {
	// Recalculate file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get HLS file info: %w", err)
	}
	file.Base().Size = fileInfo.Size()
	file.Base().ModTime = fileInfo.ModTime()

	// Create a file opener for the new file
	opener := &hlsFileOpener{path: filePath}

	// Recalculate fingerprints using the fingerprint calculator
	fingerprints, err := t.FingerprintCalculator.CalculateFingerprints(file.Base(), opener, false)
	if err != nil {
		return fmt.Errorf("failed to calculate HLS fingerprints: %w", err)
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
			logger.Warnf("[convert] failed to calculate HLS phash: %v", err)
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
		return fmt.Errorf("failed to update HLS file with new hashes: %w", err)
	}

	// Log the calculated hashes
	checksum := file.Base().Fingerprints.Get(models.FingerprintTypeMD5)
	oshash := file.Base().Fingerprints.Get(models.FingerprintTypeOshash)
	logger.Infof("[convert] recalculated HLS hashes - checksum: %v, oshash: %v", checksum, oshash)
	return nil
}

func (t *ConvertHLSToMP4Task) generateVTTFile(ctx context.Context, file *models.VideoFile, filePath string) error {
	// Check if VTT file already exists
	sceneHash := t.Scene.GetHash(t.FileNamingAlgorithm)
	vttPath := t.Paths.Scene.GetSpriteVttFilePath(sceneHash)

	if _, err := os.Stat(vttPath); err == nil {
		logger.Infof("[convert] VTT file already exists for HLS: %s", vttPath)
		return nil
	}

	// Check if sprite image exists
	spritePath := t.Paths.Scene.GetSpriteImageFilePath(sceneHash)
	if _, err := os.Stat(spritePath); err != nil {
		logger.Infof("[convert] sprite image does not exist for HLS, skipping VTT generation: %s", spritePath)
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

	logger.Infof("[convert] generating VTT file for HLS: %s", vttPath)
	if err := generator.SpriteVTT(ctx, vttPath, spritePath, stepSize); err != nil {
		return fmt.Errorf("failed to generate VTT file for HLS: %w", err)
	}

	logger.Infof("[convert] successfully generated VTT file for HLS: %s", vttPath)
	return nil
}

func (t *ConvertHLSToMP4Task) monitorFileSize(filePath string, originalSize int64, progress *job.Progress, done chan bool) {
	ticker := time.NewTicker(2 * time.Second) // Check every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if fileInfo, err := os.Stat(filePath); err == nil {
				currentSize := fileInfo.Size()
				if originalSize > 0 {
					percent := float64(currentSize) / float64(originalSize)
					if percent > 1.0 {
						percent = 1.0
					}

					// Update progress bar based on file size
					progress.SetPercent(percent)

					logger.Infof("[convert] HLS file size progress: %d/%d bytes (%.1f%%) - %.2f/%.2f MB",
						currentSize, originalSize, percent*100,
						float64(currentSize)/1024/1024, float64(originalSize)/1024/1024)
				} else {
					logger.Infof("[convert] HLS current file size: %d bytes (%.2f MB)",
						currentSize, float64(currentSize)/1024/1024)
				}
			}
		}
	}
}

func (t *ConvertHLSToMP4Task) monitorFileSizeWithStatusUpdate(filePath string, originalSize int64, progress *job.Progress, done chan bool) {
	ticker := time.NewTicker(2 * time.Second) // Check every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if fileInfo, err := os.Stat(filePath); err == nil {
				currentSize := fileInfo.Size()
				if originalSize > 0 {
					percent := float64(currentSize) / float64(originalSize)
					if percent > 1.0 {
						percent = 1.0
					}

					// Update progress bar based on file size
					progress.SetPercent(percent)

					logger.Infof("[convert] HLS file size progress: %d/%d bytes (%.1f%%) - %.2f/%.2f MB",
						currentSize, originalSize, percent*100,
						float64(currentSize)/1024/1024, float64(originalSize)/1024/1024)
				} else {
					logger.Infof("[convert] HLS current file size: %d bytes (%.2f MB)",
						currentSize, float64(currentSize)/1024/1024)
				}
			}
		}
	}
}

func (t *ConvertHLSToMP4Task) monitorFileSizeWithQueue(filePath string, originalSize int64, taskQueue *job.TaskQueue, done chan bool) {
	ticker := time.NewTicker(6 * time.Second) // Check every 6 seconds
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if fileInfo, err := os.Stat(filePath); err == nil {
				currentSize := fileInfo.Size()
				if originalSize > 0 {
					percent := float64(currentSize) / float64(originalSize)
					if percent > 1.0 {
						percent = 1.0
					}

					// Create a task with dynamic description
					statusText := fmt.Sprintf("Converting HLS video to MP4 - %.1f%% (%.2f/%.2f MB)",
						percent*100,
						float64(currentSize)/1024/1024,
						float64(originalSize)/1024/1024)

					taskQueue.Add(statusText, func(ctx context.Context) {
						// This task will update the description
						// Add a longer delay to make the status visible
						time.Sleep(4 * time.Second)
					})

					logger.Infof("[convert] HLS file size progress: %d/%d bytes (%.1f%%) - %.2f/%.2f MB",
						currentSize, originalSize, percent*100,
						float64(currentSize)/1024/1024, float64(originalSize)/1024/1024)
				} else {
					statusText := fmt.Sprintf("Converting HLS video to MP4 - %.2f MB",
						float64(currentSize)/1024/1024)

					taskQueue.Add(statusText, func(ctx context.Context) {
						// This task will update the description
						// Add a longer delay to make the status visible
						time.Sleep(4 * time.Second)
					})

					logger.Infof("[convert] HLS current file size: %d bytes (%.2f MB)",
						currentSize, float64(currentSize)/1024/1024)
				}
			}
		}
	}
}

// copyFileContent copies the content from source to destination file
func (t *ConvertHLSToMP4Task) copyFileContent(src, dst string) error {
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

	logger.Infof("[convert] successfully copied HLS file content from %s to %s", src, dst)
	return nil
}

// regenerateSprites regenerates sprites for the scene after HLS conversion
// NOTE: This function expects to be called within an existing transaction context
func (t *ConvertHLSToMP4Task) regenerateSprites(ctx context.Context) error {
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

	sceneHash := updatedScene.GetHash(t.FileNamingAlgorithm)
	spriteImagePath := t.Paths.Scene.GetSpriteImageFilePath(sceneHash)
	spriteVttPath := t.Paths.Scene.GetSpriteVttFilePath(sceneHash)

	if _, err := os.Stat(spriteImagePath); err == nil {
		if _, err := os.Stat(spriteVttPath); err == nil {
			logger.Infof("[convert] sprites already exist for HLS scene %d, skipping regeneration", t.Scene.ID)
			return nil
		}
	}

	spriteTask := GenerateSpriteTask{
		Scene:               *updatedScene, // Use updated scene with new hash
		Overwrite:           true,          // Force regeneration with new hash
		fileNamingAlgorithm: t.FileNamingAlgorithm,
	}

	// Run sprite generation
	spriteTask.Start(ctx)
	logger.Infof("[convert] regenerated sprites for HLS scene %d with updated hash", t.Scene.ID)
	return nil
}
