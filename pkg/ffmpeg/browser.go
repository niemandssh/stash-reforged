package ffmpeg

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// only support H264 by default, since Safari does not support VP8/VP9
var defaultSupportedCodecs = []string{H264, H265, Hevc}

var validForH264Mkv = []Container{Mp4, Matroska, Avi, Mov, Wmv}
var validForH264 = []Container{Mp4, Avi, Mov, Wmv}
var validForH265Mkv = []Container{Mp4, Matroska, Avi, Mov, Wmv}
var validForH265 = []Container{Mp4, Avi, Mov, Wmv}
var validForVp8 = []Container{Webm}
var validForVp9Mkv = []Container{Webm, Matroska}
var validForVp9 = []Container{Webm}
var validForHevcMkv = []Container{Mp4, Matroska, Avi, Mov, Wmv}
var validForHevc = []Container{Mp4, Avi, Mov, Wmv}

var validAudioForMkv = []ProbeAudioCodec{Aac, Mp3, Vorbis, Opus}
var validAudioForWebm = []ProbeAudioCodec{Vorbis, Opus}
var validAudioForMp4 = []ProbeAudioCodec{Aac, Mp3, Opus}
var validAudioForAvi = []ProbeAudioCodec{Aac, Mp3, Vorbis, Opus}
var validAudioForMov = []ProbeAudioCodec{Aac, Mp3, Opus}
var validAudioForWmv = []ProbeAudioCodec{Aac, Mp3}

var (
	// ErrUnsupportedVideoCodecForBrowser is returned when the video codec is not supported for browser streaming.
	ErrUnsupportedVideoCodecForBrowser = errors.New("unsupported video codec for browser")

	// ErrUnsupportedVideoCodecContainer is returned when the video codec/container combination is not supported for browser streaming.
	ErrUnsupportedVideoCodecContainer = errors.New("video codec/container combination is unsupported for browser streaming")

	// ErrUnsupportedAudioCodecContainer is returned when the audio codec/container combination is not supported for browser streaming.
	ErrUnsupportedAudioCodecContainer = errors.New("audio codec/container combination is unsupported for browser streaming")
)

// IsStreamable returns nil if the file is streamable, or an error if it is not.
func IsStreamable(videoCodec string, audioCodec ProbeAudioCodec, container Container) error {
	supportedVideoCodecs := defaultSupportedCodecs

	// check if the video codec matches the supported codecs
	if !isValidCodec(videoCodec, supportedVideoCodecs) {
		return fmt.Errorf("%w: %s", ErrUnsupportedVideoCodecForBrowser, videoCodec)
	}

	if !isValidCombo(videoCodec, container, supportedVideoCodecs) {
		return fmt.Errorf("%w: %s/%s", ErrUnsupportedVideoCodecContainer, videoCodec, container)
	}

	if !IsValidAudioForContainer(audioCodec, container) {
		return fmt.Errorf("%w: %s/%s", ErrUnsupportedAudioCodecContainer, audioCodec, container)
	}

	return nil
}

// IsHLSVideo detects if a video file is likely from HLS based on comprehensive characteristics
func IsHLSVideo(videoCodec string, audioCodec ProbeAudioCodec, container Container, duration float64) bool {
	// Basic codec and container check first
	if container != Mp4 || videoCodec != H264 || audioCodec != Aac {
		return false
	}

	// Duration-based heuristic with more conservative criteria for longer videos
	if duration > 0 {
		segmentLength := 2.0
		remainder := math.Mod(duration, segmentLength)

		// Very short durations (< 30 seconds) are highly likely HLS segments
		isVeryShort := duration < 30.0
		// Exact multiples of segment length are suspicious regardless of duration
		isExactMultiple := math.Abs(remainder) < 0.01 || math.Abs(remainder-segmentLength) < 0.01

		// For short videos, use more lenient tolerance
		if isVeryShort {
			isDurationSuspicious := remainder < 0.03 || remainder > (segmentLength-0.03)
			if isDurationSuspicious || isExactMultiple {
				return true
			}
		} else if isExactMultiple {
			// For longer videos, only flag if timing is very exact (likely genuine HLS artifacts)
			return true
		}
	}

	return false
}

// IsHLSVideoWithMetadata provides enhanced HLS detection using additional metadata
// This function should be used when VideoFile with full metadata is available
func IsHLSVideoWithMetadata(vf *VideoFile) bool {
	// Basic codec and container check
	// Container can be "mp4" or "mov,mp4,m4a,3gp,3g2,mj2" - check if it contains mp4
	isMP4Container := strings.Contains(strings.ToLower(vf.Container), "mp4")
	if !isMP4Container || vf.VideoCodec != H264 || vf.AudioCodec != string(Aac) {
		return false
	}

	// Check for HLS-specific metadata patterns in encoder
	hasHLSEncoder := false
	if vf.JSON.Format.Tags.Encoder != "" {
		encoder := strings.ToLower(vf.JSON.Format.Tags.Encoder)
		// HLS transcoding tools often leave specific encoder signatures
		hlsEncoders := []string{"hls", "m3u8", "segmenter", "apple", "darwin"}
		for _, hlsPattern := range hlsEncoders {
			if strings.Contains(encoder, hlsPattern) {
				hasHLSEncoder = true
				break
			}
		}
	}

	// If we have clear HLS indicators in encoder metadata, consider it HLS
	if hasHLSEncoder {
		return true
	}

	// Check major_brand and compatible_brands for unusual patterns
	majorBrand := strings.ToLower(vf.JSON.Format.Tags.MajorBrand)
	compatibleBrands := strings.ToLower(vf.JSON.Format.Tags.CompatibleBrands)

	// Missing brand info can indicate HLS segments or corrupted files
	if majorBrand == "" || compatibleBrands == "" {
		return IsHLSVideo(vf.VideoCodec, ProbeAudioCodec(vf.AudioCodec), Mp4, vf.FileDuration)
	}

	// Check for frame rate and timing irregularities (key indicator of HLS issues)
	if vf.FrameRate > 0 && vf.FrameCount > 0 {
		// Calculate expected frame count based on duration and frame rate
		expectedFrames := vf.FileDuration * vf.FrameRate
		actualFrames := float64(vf.FrameCount)

		frameCountDiff := math.Abs(expectedFrames - actualFrames)
		frameCountErrorRate := frameCountDiff / expectedFrames

		// More sensitive detection for timing issues (3% instead of 5%)
		// HLS videos often have sync issues that manifest as frame count mismatches
		if frameCountErrorRate > 0.03 {
			return true
		}

		// Check for unusual frame rates that might indicate HLS conversion artifacts
		// Most normal videos have standard frame rates (23.976, 24, 25, 29.97, 30, 50, 60)
		standardFrameRates := []float64{23.976, 24.0, 25.0, 29.97, 30.0, 50.0, 60.0}
		isStandardFrameRate := false
		for _, standardRate := range standardFrameRates {
			if math.Abs(vf.FrameRate-standardRate) < 0.1 {
				isStandardFrameRate = true
				break
			}
		}

		// Non-standard frame rates combined with suspicious duration might indicate HLS
		if !isStandardFrameRate {
			return IsHLSVideo(vf.VideoCodec, ProbeAudioCodec(vf.AudioCodec), Mp4, vf.FileDuration)
		}

		// Special case: if basic HLS detection is positive, trust it even with good technical specs
		// Some HLS videos may have been "fixed" but still have sync issues
		basicHLSDetection := IsHLSVideo(vf.VideoCodec, ProbeAudioCodec(vf.AudioCodec), Mp4, vf.FileDuration)
		if basicHLSDetection {
			return true
		}
	}

	// Check for suspicious timing patterns in video stream vs container duration
	if vf.VideoStreamDuration > 0 && vf.FileDuration > 0 {
		durationDiff := math.Abs(vf.VideoStreamDuration - vf.FileDuration)
		// Significant difference between container and stream duration can indicate HLS issues
		if durationDiff > 0.1 { // More than 100ms difference
			return true
		}
	}

	// Check for audio/video synchronization issues
	audioStream := vf.getAudioStream()
	if audioStream != nil && audioStream.Duration != "" {
		audioDuration, err := strconv.ParseFloat(audioStream.Duration, 64)
		if err == nil && audioDuration > 0 {
			// Compare audio duration with video duration
			audioDiff := math.Abs(vf.FileDuration - audioDuration)
			// Audio/video duration mismatch > 50ms often indicates sync issues from HLS conversion
			if audioDiff > 0.05 {
				return true
			}
		}
	}

	// Fallback to basic duration-based check with moderate sensitivity
	return IsHLSVideo(vf.VideoCodec, ProbeAudioCodec(vf.AudioCodec), Mp4, vf.FileDuration)
}

func isValidCodec(codecName string, supportedCodecs []string) bool {
	for _, c := range supportedCodecs {
		if c == codecName {
			return true
		}
	}
	return false
}

func isValidAudio(audio ProbeAudioCodec, validCodecs []ProbeAudioCodec) bool {
	// if audio codec is missing or unsupported by ffmpeg we can't do anything about it
	// report it as valid so that the file can at least be streamed directly if the video codec is supported
	if audio == MissingUnsupported {
		return true
	}

	for _, c := range validCodecs {
		if c == audio {
			return true
		}
	}

	return false
}

// IsValidAudioForContainer returns true if the audio codec is valid for the container.
func IsValidAudioForContainer(audio ProbeAudioCodec, format Container) bool {
	switch format {
	case Matroska:
		return isValidAudio(audio, validAudioForMkv)
	case Webm:
		return isValidAudio(audio, validAudioForWebm)
	case Mp4:
		return isValidAudio(audio, validAudioForMp4)
	case Avi:
		return isValidAudio(audio, validAudioForAvi)
	case Mov:
		return isValidAudio(audio, validAudioForMov)
	case Wmv:
		return isValidAudio(audio, validAudioForWmv)
	}
	return false
}

// isValidCombo checks if a codec/container combination is valid.
// Returns true on validity, false otherwise
func isValidCombo(codecName string, format Container, supportedVideoCodecs []string) bool {
	supportMKV := isValidCodec(Mkv, supportedVideoCodecs)
	supportHEVC := isValidCodec(Hevc, supportedVideoCodecs)

	switch codecName {
	case H264:
		if supportMKV {
			return isValidForContainer(format, validForH264Mkv)
		}
		return isValidForContainer(format, validForH264)
	case H265:
		if supportMKV {
			return isValidForContainer(format, validForH265Mkv)
		}
		return isValidForContainer(format, validForH265)
	case Vp8:
		return isValidForContainer(format, validForVp8)
	case Vp9:
		if supportMKV {
			return isValidForContainer(format, validForVp9Mkv)
		}
		return isValidForContainer(format, validForVp9)
	case Hevc:
		if supportHEVC {
			if supportMKV {
				return isValidForContainer(format, validForHevcMkv)
			}
			return isValidForContainer(format, validForHevc)
		}
	}
	return false
}

func isValidForContainer(format Container, validContainers []Container) bool {
	for _, fmt := range validContainers {
		if fmt == format {
			return true
		}
	}
	return false
}
