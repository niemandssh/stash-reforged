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

const (
	// CRF/CQ 18 is widely considered visually lossless for H.265.
	compressCRF = "18"
	// CPU fallback only — hardware encoders are tried first.
	compressX265Preset = "medium"
	compressX265Tune   = "ssim"
	compressX265Params = "log-level=error:aq-mode=3:ref=4:rc-lookahead=30:pools=+"
)

type CompressVideoTask struct {
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

func (t *CompressVideoTask) GetDescription() string {
	return fmt.Sprintf("Compressing %s to H.265 (max quality, CRF %s)", t.Scene.Path, compressCRF)
}

func (t *CompressVideoTask) Execute(ctx context.Context, progress *job.Progress) error {
	f := t.Scene.Files.Primary()
	if f == nil {
		return fmt.Errorf("scene has no primary file")
	}

	if !t.needsCompression(f) {
		logger.Infof("[compress] scene %d does not need compression", t.Scene.ID)
		progress.SetTotal(1)
		progress.SetProcessed(1)
		return nil
	}

	logger.Infof("[compress] compressing scene %d to H.265 at %dx%d", t.Scene.ID, f.Width, f.Height)

	progress.SetTotal(3)
	progress.SetProcessed(0)

	var conversionErr error

	originalFileInfo, err := os.Stat(f.Path)
	originalSize := int64(0)
	if err == nil {
		originalSize = originalFileInfo.Size()
		logger.Infof("[compress] original file size: %d bytes (%.2f MB)", originalSize, float64(originalSize)/1024/1024)
	}

	done := make(chan bool)
	oldHash := t.Scene.GetHash(t.FileNamingAlgorithm)
	tempFile := filepath.Join(t.Config.GetGeneratedPath(), fmt.Sprintf("compress_%d_%s.mp4", t.Scene.ID, oldHash))
	go t.monitorFileSize(tempFile, originalSize, progress, done)

	taskQueue := job.NewTaskQueue(ctx, progress, 100, 1)
	go t.monitorFileSizeWithQueue(tempFile, originalSize, taskQueue, done)

	conversionErr = t.compressVideo(ctx, f, tempFile, originalSize, progress, done)
	if conversionErr != nil {
		logger.Errorf("[compress] error compressing scene %d: %v", t.Scene.ID, conversionErr)
		close(done)
		taskQueue.Close()
		return conversionErr
	}
	progress.SetProcessed(1)

	taskQueue.Close()
	close(done)

	progress.ExecuteTask("Updating scene metadata", func() {
		progress.SetProcessed(2)
	})

	progress.ExecuteTask("Finalizing compression", func() {
		progress.SetProcessed(3)
	})

	logger.Infof("[compress] successfully compressed scene %d", t.Scene.ID)
	return nil
}

func (t *CompressVideoTask) needsCompression(f *models.VideoFile) bool {
	codec := strings.ToLower(f.VideoCodec)
	if codec == "hevc" || codec == "h265" {
		logger.Infof("[compress] file is already HEVC/H.265, skipping compression")
		return false
	}
	return true
}

func (t *CompressVideoTask) compressVideo(ctx context.Context, f *models.VideoFile, tempFile string, originalSize int64, progress *job.Progress, done chan bool) error {
	oldHash := t.Scene.GetHash(t.FileNamingAlgorithm)

	backupTempDir := t.Config.GetTempPath()
	if err := os.MkdirAll(backupTempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp backup directory: %w", err)
	}
	backupTempFile := filepath.Join(backupTempDir, filepath.Base(f.Path))
	if err := t.copyFileContent(f.Path, backupTempFile); err != nil {
		return fmt.Errorf("failed to create backup copy: %w", err)
	}
	defer func() {
		if _, err := os.Stat(backupTempFile); err == nil {
			_ = os.Remove(backupTempFile)
		}
	}()

	conversionSuccessful := false
	defer func() {
		if !conversionSuccessful {
			if _, err := os.Stat(tempFile); err == nil {
				_ = os.Remove(tempFile)
			}
		}
	}()

	if err := t.performCompressionWithProgress(ctx, f, tempFile, progress); err != nil {
		return fmt.Errorf("compression failed: %w", err)
	}

	compressedInfo, err := os.Stat(tempFile)
	if err != nil {
		return fmt.Errorf("compressed file not found: %w", err)
	}

	if originalSize > 0 && compressedInfo.Size() >= originalSize {
		savings := float64(originalSize-compressedInfo.Size()) / float64(originalSize) * 100
		logger.Infof("[compress] compressed file (%.2f MB) is not smaller than original (%.2f MB, savings %.1f%%), keeping original",
			float64(compressedInfo.Size())/1024/1024,
			float64(originalSize)/1024/1024,
			savings,
		)
		return nil
	}

	if originalSize > 0 {
		savings := float64(originalSize-compressedInfo.Size()) / float64(originalSize) * 100
		logger.Infof("[compress] file size reduced from %.2f MB to %.2f MB (%.1f%% savings)",
			float64(originalSize)/1024/1024,
			float64(compressedInfo.Size())/1024/1024,
			savings,
		)
	}

	if err := t.validateCompressedFile(tempFile, f.Width, f.Height); err != nil {
		return fmt.Errorf("compressed file validation failed: %w", err)
	}

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

	var finalPath string
	if isUpdated {
		finalPath = newFile.Base().Path
		if tempFile != finalPath {
			if err := t.copyFileContent(tempFile, finalPath); err != nil {
				return fmt.Errorf("failed to copy compressed file: %w", err)
			}
		}
		if err := t.validateCompressedFile(finalPath, f.Width, f.Height); err != nil {
			return fmt.Errorf("updated file validation failed: %w", err)
		}
	} else {
		finalPath = t.getFinalPath(newFile)
		if err := t.copyFileContent(tempFile, finalPath); err != nil {
			return fmt.Errorf("failed to copy compressed file to final location: %w", err)
		}
		_ = os.Remove(tempFile)

		if err := t.updateFilePath(ctx, newFile, finalPath); err != nil {
			return fmt.Errorf("failed to update file path: %w", err)
		}

		if err := t.validateCompressedFile(finalPath, f.Width, f.Height); err != nil {
			return fmt.Errorf("compressed file validation failed: %w", err)
		}

		if err := os.Remove(f.Path); err != nil {
			logger.Warnf("[compress] failed to remove original file %s: %v", f.Path, err)
		}

		if err := t.Repository.WithTxn(ctx, func(ctx context.Context) error {
			return t.deleteOldFileRecord(ctx, f)
		}); err != nil {
			logger.Warnf("[compress] failed to delete old file record: %v", err)
		}
	}

	if err := t.recalculateFileHashes(ctx, newFile, finalPath); err != nil {
		logger.Warnf("[compress] failed to recalculate file hashes: %v", err)
	}

	if err := t.regenerateSprites(ctx, oldHash); err != nil {
		logger.Warnf("[compress] failed to regenerate sprites: %v", err)
	}

	if err := t.generateVTTFile(ctx, newFile, finalPath); err != nil {
		logger.Warnf("[compress] failed to generate VTT file: %v", err)
	}

	conversionSuccessful = true
	if _, err := os.Stat(tempFile); err == nil {
		_ = os.Remove(tempFile)
	}

	return nil
}

func (t *CompressVideoTask) getHardwareHEVCCodec() *ffmpeg.VideoCodec {
	codecs := []ffmpeg.VideoCodec{
		{Name: "HEVC NVENC", CodeName: "hevc_nvenc"},
		{Name: "HEVC QSV", CodeName: "hevc_qsv"},
		{Name: "HEVC AMF", CodeName: "hevc_amf"},
		{Name: "HEVC VAAPI", CodeName: "hevc_vaapi"},
		{Name: "HEVC VideoToolbox", CodeName: "hevc_videotoolbox"},
	}

	for _, codec := range codecs {
		if t.testHardwareCodec(codec) {
			logger.Infof("[compress] using hardware codec %s", codec.CodeName)
			return &codec
		}
	}
	return nil
}

func (t *CompressVideoTask) testHardwareCodec(codec ffmpeg.VideoCodec) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var args ffmpeg.Args
	args = append(args, "-hide_banner", "-loglevel", "error")
	args = args.Format("lavfi")
	// Match typical scene resolution so unsupported encoders fail here, not on real files.
	args = args.Input("color=c=black:s=1920x1080")
	args = append(args, "-t", "0.1")
	args = args.VideoCodec(codec)
	args = append(args, t.getHEVCVideoArgs(codec)...)

	args = args.Format("null")
	args = args.Output("-")

	return t.FFMpeg.Command(ctx, args).Run() == nil
}

func (t *CompressVideoTask) getHEVCVideoArgs(codec ffmpeg.VideoCodec) ffmpeg.Args {
	var videoArgs ffmpeg.Args
	videoArgs = append(videoArgs, "-tag:v", "hvc1", "-pix_fmt", "yuv420p")

	switch codec.CodeName {
	case "hevc_nvenc":
		videoArgs = append(videoArgs,
			"-rc", "vbr",
			"-cq", compressCRF,
			"-preset", "p4",
			"-tune", "hq",
			"-profile:v", "main",
			"-b:v", "0",
		)
	case "hevc_qsv":
		videoArgs = append(videoArgs,
			"-global_quality", compressCRF,
			"-preset", "medium",
			"-look_ahead", "1",
			"-look_ahead_depth", "20",
			"-profile:v", "main",
		)
	case "hevc_amf":
		videoArgs = append(videoArgs,
			"-quality", "quality",
			"-rc", "cqp",
			"-qp_i", compressCRF,
			"-qp_p", compressCRF,
			"-profile:v", "main",
		)
	case "hevc_vaapi":
		videoArgs = append(videoArgs,
			"-qp", compressCRF,
			"-profile:v", "main",
		)
	case "hevc_videotoolbox":
		videoArgs = append(videoArgs, "-q:v", "65", "-profile:v", "main")
	default:
		videoArgs = append(videoArgs,
			"-preset", compressX265Preset,
			"-tune", compressX265Tune,
			"-crf", compressCRF,
			"-x265-params", compressX265Params,
		)
	}

	return videoArgs
}

func (t *CompressVideoTask) getAudioSettings(sourceAudioCodec string) (ffmpeg.AudioCodec, ffmpeg.Args) {
	switch strings.ToLower(sourceAudioCodec) {
	case "aac", "mp3", "ac3", "eac3", "alac", "flac":
		return ffmpeg.AudioCodecCopy, nil
	}
	return ffmpeg.AudioCodecAAC, ffmpeg.Args{
		"-ac", "2",
		"-ar", "48000",
		"-ab", "256k",
		"-strict", "-2",
	}
}

func (t *CompressVideoTask) performCompressionWithProgress(ctx context.Context, f *models.VideoFile, outputPath string, progress *job.Progress) error {
	videoFile, err := t.FFProbe.NewVideoFile(f.Path)
	if err != nil {
		return fmt.Errorf("error reading video file: %w", err)
	}

	audioCodec, audioArgs := t.getAudioSettings(f.AudioCodec)

	extraInputArgs := append(t.Config.GetTranscodeInputArgs(),
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",
	)
	extraOutputArgs := append(t.Config.GetTranscodeOutputArgs(),
		"-movflags", "+faststart",
	)

	hwCodec := t.getHardwareHEVCCodec()
	if hwCodec != nil {
		logger.Infof("[compress] attempting hardware HEVC encoder: %s", hwCodec.Name)

		if err := t.runCompressionEncode(ctx, f.Path, outputPath, progress, videoFile.FileDuration, *hwCodec, extraInputArgs, extraOutputArgs, audioCodec, audioArgs); err == nil {
			logger.Infof("[compress] hardware HEVC encoding successful")
			return nil
		}

		logger.Warnf("[compress] hardware HEVC encoding failed, falling back to libx265 (CPU)")
		if _, removeErr := os.Stat(outputPath); removeErr == nil {
			_ = os.Remove(outputPath)
		}
	} else {
		logger.Infof("[compress] no hardware HEVC encoder available, using libx265 (CPU)")
	}

	videoArgs := t.getHEVCVideoArgs(ffmpeg.VideoCodecLibX265)
	args := transcoder.Transcode(f.Path, transcoder.TranscodeOptions{
		OutputPath:      outputPath,
		VideoCodec:      ffmpeg.VideoCodecLibX265,
		VideoArgs:       videoArgs,
		AudioCodec:      audioCodec,
		AudioArgs:       audioArgs,
		Format:          ffmpeg.FormatMP4,
		ExtraInputArgs:  extraInputArgs,
		ExtraOutputArgs: extraOutputArgs,
	})

	logger.Infof("[compress] running libx265 (CPU) ffmpeg command: %v", args)
	return t.FFMpeg.GenerateWithProgress(ctx, args, progress, videoFile.FileDuration)
}

func (t *CompressVideoTask) runCompressionEncode(
	ctx context.Context,
	inputPath, outputPath string,
	progress *job.Progress,
	duration float64,
	videoCodec ffmpeg.VideoCodec,
	extraInputArgs, extraOutputArgs ffmpeg.Args,
	audioCodec ffmpeg.AudioCodec,
	audioArgs ffmpeg.Args,
) error {
	videoArgs := t.getHEVCVideoArgs(videoCodec)
	args := transcoder.Transcode(inputPath, transcoder.TranscodeOptions{
		OutputPath:      outputPath,
		VideoCodec:      videoCodec,
		VideoArgs:       videoArgs,
		AudioCodec:      audioCodec,
		AudioArgs:       audioArgs,
		Format:          ffmpeg.FormatMP4,
		ExtraInputArgs:  extraInputArgs,
		ExtraOutputArgs: extraOutputArgs,
	})

	logger.Infof("[compress] running %s ffmpeg command: %v", videoCodec.CodeName, args)
	return t.FFMpeg.GenerateWithProgress(ctx, args, progress, duration)
}

func (t *CompressVideoTask) validateCompressedFile(filePath string, expectedWidth, expectedHeight int) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("compressed file does not exist: %w", err)
	}
	if fileInfo.Size() == 0 {
		return fmt.Errorf("compressed file is empty")
	}

	videoFile, err := t.FFProbe.NewVideoFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to probe compressed file: %w", err)
	}

	if videoFile.FileDuration <= 0 {
		return fmt.Errorf("compressed file has invalid duration")
	}

	codec := strings.ToLower(videoFile.VideoCodec)
	if codec != "hevc" && codec != "h265" {
		return fmt.Errorf("compressed file has wrong video codec: %s (expected hevc)", videoFile.VideoCodec)
	}

	if videoFile.Width != expectedWidth || videoFile.Height != expectedHeight {
		return fmt.Errorf("compressed file resolution %dx%d does not match original %dx%d",
			videoFile.Width, videoFile.Height, expectedWidth, expectedHeight)
	}

	return nil
}

func (t *CompressVideoTask) monitorFileSize(tempFile string, originalSize int64, progress *job.Progress, done chan bool) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if fileInfo, err := os.Stat(tempFile); err == nil && originalSize > 0 {
				percent := float64(fileInfo.Size()) / float64(originalSize)
				if percent > 1.0 {
					percent = 1.0
				}
				progress.SetPercent(percent)
			}
		}
	}
}

func (t *CompressVideoTask) monitorFileSizeWithQueue(tempFile string, originalSize int64, taskQueue *job.TaskQueue, done chan bool) {
	ticker := time.NewTicker(6 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if fileInfo, err := os.Stat(tempFile); err == nil {
				statusText := fmt.Sprintf("Compressing to H.265 - %.2f MB", float64(fileInfo.Size())/1024/1024)
				if originalSize > 0 {
					percent := float64(fileInfo.Size()) / float64(originalSize) * 100
					if percent > 100 {
						percent = 100
					}
					statusText = fmt.Sprintf("Compressing to H.265 - %.1f%% (%.2f/%.2f MB)",
						percent,
						float64(fileInfo.Size())/1024/1024,
						float64(originalSize)/1024/1024,
					)
				}
				taskQueue.Add(statusText, func(ctx context.Context) {
					time.Sleep(4 * time.Second)
				})
			}
		}
	}
}

func (t *CompressVideoTask) createNewVideoFile(ctx context.Context, filePath string) (*models.VideoFile, bool, error) {
	videoFile, err := t.FFProbe.NewVideoFile(filePath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to probe file: %w", err)
	}

	originalFile, err := t.Repository.File.FindByPath(ctx, t.Scene.Files.Primary().Path)
	if err != nil {
		return nil, false, fmt.Errorf("failed to find original file: %w", err)
	}

	originalBasename := originalFile.Base().Basename
	ext := filepath.Ext(originalBasename)
	properBasename := strings.TrimSuffix(originalBasename, ext) + ".mp4"

	existingFile, err := t.Repository.File.FindByBasenameAndParentFolderID(ctx, properBasename, originalFile.Base().ParentFolderID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check for existing file: %w", err)
	}

	if existingFile != nil {
		existingVideoFile, ok := existingFile.(*models.VideoFile)
		if !ok {
			return nil, false, fmt.Errorf("existing file is not a video file")
		}

		isAssociated, err := t.isFileAssociatedWithScene(ctx, existingVideoFile.ID)
		if err != nil {
			return nil, false, err
		}

		finalPath := t.getFinalPath(existingVideoFile)
		existingVideoFile.Base().Path = finalPath
		existingVideoFile.Base().Size = videoFile.Size
		existingVideoFile.Base().ModTime = time.Now()
		existingVideoFile.Base().UpdatedAt = time.Now()
		existingVideoFile.Duration = videoFile.FileDuration
		existingVideoFile.VideoCodec = videoFile.VideoCodec
		existingVideoFile.AudioCodec = videoFile.AudioCodec
		existingVideoFile.Width = videoFile.Width
		existingVideoFile.Height = videoFile.Height
		existingVideoFile.FrameRate = videoFile.FrameRate
		existingVideoFile.BitRate = videoFile.Bitrate
		existingVideoFile.Format = "mp4"

		if err := t.Repository.File.Update(ctx, existingVideoFile); err != nil {
			return nil, false, err
		}

		if !isAssociated {
			if err := t.Repository.Scene.AssignFiles(ctx, t.Scene.ID, []models.FileID{existingVideoFile.ID}); err != nil {
				return nil, false, err
			}
		}

		return existingVideoFile, true, nil
	}

	newFile := &models.VideoFile{
		BaseFile: &models.BaseFile{
			Path:           filePath,
			Basename:       properBasename,
			Size:           videoFile.Size,
			ParentFolderID: originalFile.Base().ParentFolderID,
			CreatedAt:      originalFile.Base().CreatedAt,
			UpdatedAt:      originalFile.Base().UpdatedAt,
			DirEntry:       models.DirEntry{ModTime: originalFile.Base().ModTime},
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

	if err := t.Repository.File.Create(ctx, newFile); err != nil {
		return nil, false, err
	}

	return newFile, false, nil
}

func (t *CompressVideoTask) updateSceneWithNewFile(ctx context.Context, newFile *models.VideoFile) error {
	return t.Repository.WithTxn(ctx, func(ctx context.Context) error {
		if err := t.Repository.Scene.AssignFiles(ctx, t.Scene.ID, []models.FileID{newFile.ID}); err != nil {
			return err
		}
		scenePartial := models.NewScenePartial()
		scenePartial.PrimaryFileID = &newFile.ID
		scenePartial.IsBroken = models.NewOptionalBool(false)
		_, err := t.Repository.Scene.UpdatePartial(ctx, t.Scene.ID, scenePartial)
		return err
	})
}

func (t *CompressVideoTask) getFinalPath(file *models.VideoFile) string {
	originalFile := t.Scene.Files.Primary()
	originalDir := filepath.Dir(originalFile.Path)
	ext := filepath.Ext(originalFile.Base().Basename)
	newBasename := strings.TrimSuffix(originalFile.Base().Basename, ext) + ".mp4"
	_ = os.MkdirAll(originalDir, 0755)
	return filepath.Join(originalDir, newBasename)
}

func (t *CompressVideoTask) updateFilePath(ctx context.Context, file *models.VideoFile, newPath string) error {
	file.Base().Path = newPath
	file.Base().Basename = filepath.Base(newPath)
	return t.Repository.File.Update(ctx, file)
}

func (t *CompressVideoTask) deleteOldFileRecord(ctx context.Context, oldFile *models.VideoFile) error {
	return t.Repository.File.Destroy(ctx, oldFile.ID)
}

func (t *CompressVideoTask) recalculateFileHashes(ctx context.Context, file *models.VideoFile, filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	file.Base().Size = fileInfo.Size()
	file.Base().ModTime = fileInfo.ModTime()
	file.Base().Fingerprints = nil

	opener := &osFileOpener{path: filePath}
	fingerprints, err := t.FingerprintCalculator.CalculateFingerprints(file.Base(), opener, false)
	if err != nil {
		return err
	}
	for _, fp := range fingerprints {
		file.Base().Fingerprints = file.Base().Fingerprints.AppendUnique(fp)
	}

	if file.Duration > 0 {
		phash, err := videophash.Generate(t.FFMpeg, file)
		if err == nil {
			phashInt := int64(*phash)
			file.Base().Fingerprints = file.Base().Fingerprints.AppendUnique(models.Fingerprint{
				Type:        models.FingerprintTypePhash,
				Fingerprint: phashInt,
			})
		}
	}

	return t.Repository.File.Update(ctx, file)
}

func (t *CompressVideoTask) regenerateSprites(ctx context.Context, oldHash string) error {
	updatedScene, err := t.Repository.Scene.Find(ctx, t.Scene.ID)
	if err != nil {
		return err
	}
	if err := updatedScene.LoadFiles(ctx, t.Repository.Scene); err != nil {
		return err
	}

	newHash := updatedScene.GetHash(t.FileNamingAlgorithm)
	oldSpriteImagePath := t.Paths.Scene.GetSpriteImageFilePath(oldHash)
	oldSpriteVttPath := t.Paths.Scene.GetSpriteVttFilePath(oldHash)
	newSpriteImagePath := t.Paths.Scene.GetSpriteImageFilePath(newHash)
	newSpriteVttPath := t.Paths.Scene.GetSpriteVttFilePath(newHash)

	if _, err := os.Stat(oldSpriteImagePath); err == nil {
		if _, err := os.Stat(oldSpriteVttPath); err == nil {
			_ = t.updateVttFileHash(oldSpriteVttPath, oldHash, newHash)
			_ = os.Rename(oldSpriteImagePath, newSpriteImagePath)
			_ = os.Rename(oldSpriteVttPath, newSpriteVttPath)
			return nil
		}
	}

	spriteTask := GenerateSpriteTask{
		Scene:               *updatedScene,
		Overwrite:           true,
		fileNamingAlgorithm: t.FileNamingAlgorithm,
	}
	spriteTask.Start(ctx)
	return nil
}

func (t *CompressVideoTask) updateVttFileHash(vttPath, oldHash, newHash string) error {
	content, err := os.ReadFile(vttPath)
	if err != nil {
		return err
	}
	return os.WriteFile(vttPath, []byte(strings.ReplaceAll(string(content), oldHash, newHash)), 0644)
}

func (t *CompressVideoTask) generateVTTFile(ctx context.Context, file *models.VideoFile, filePath string) error {
	updatedScene, err := t.Repository.Scene.Find(ctx, t.Scene.ID)
	if err != nil {
		return err
	}
	if err := updatedScene.LoadFiles(ctx, t.Repository.Scene); err != nil {
		return err
	}

	sceneHash := updatedScene.GetHash(t.FileNamingAlgorithm)
	vttPath := t.Paths.Scene.GetSpriteVttFilePath(sceneHash)
	if _, err := os.Stat(vttPath); err == nil {
		return nil
	}

	spritePath := t.Paths.Scene.GetSpriteImageFilePath(sceneHash)
	if _, err := os.Stat(spritePath); err != nil {
		return nil
	}

	generator := &generate.Generator{
		Encoder:      t.FFMpeg,
		FFMpegConfig: t.Config,
		LockManager:  t.G.LockManager,
		ScenePaths:   t.Paths.Scene,
	}

	stepSize := 10.0
	if file.Duration > 0 {
		stepSize = file.Duration / 100.0
	}

	return generator.SpriteVTT(ctx, vttPath, spritePath, stepSize)
}

func (t *CompressVideoTask) isFileAssociatedWithScene(ctx context.Context, fileID models.FileID) (bool, error) {
	sceneFiles, err := t.Repository.Scene.GetFiles(ctx, t.Scene.ID)
	if err != nil {
		return false, err
	}
	for _, sceneFile := range sceneFiles {
		if sceneFile.ID == fileID {
			return true, nil
		}
	}
	return false, nil
}

func (t *CompressVideoTask) copyFileContent(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return dstFile.Sync()
}
