package generate

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"sort"

	"github.com/disintegration/imaging"
	"github.com/stashapp/stash/pkg/logger"
)

const (
	AIVisionSampleCount = 180
	AIVisionPanelCount  = 5
	AIVisionPanelCols   = 6
	AIVisionPanelRows   = 6

	// Oversample the ending so climax/finish actions are more likely to appear in panels.
	aiVisionEndingSampleFraction = 0.45
	aiVisionEndingPortion        = 0.22

	// Stay away from EOF — reported duration is often slightly longer than decodable media.
	aiVisionEndSafetyMargin = 0.5
)

// GenerateAIVisionPanels extracts frames from a video and writes montage panels to disk.
func (g Generator) GenerateAIVisionPanels(ctx context.Context, videoPath string, duration float64, outputPaths []string) error {
	if len(outputPaths) != AIVisionPanelCount {
		return fmt.Errorf("expected %d output paths, got %d", AIVisionPanelCount, len(outputPaths))
	}

	panels, err := g.buildAIVisionPanelBytes(ctx, videoPath, duration)
	if err != nil {
		return err
	}

	for i, data := range panels {
		if err := os.WriteFile(outputPaths[i], data, 0644); err != nil {
			return fmt.Errorf("writing AI vision panel %d: %w", i, err)
		}
	}

	return nil
}

// BuildAIVisionPanelBytes extracts frames from a video and returns montage panel JPEG data.
func (g Generator) BuildAIVisionPanelBytes(ctx context.Context, videoPath string, duration float64) ([][]byte, error) {
	return g.buildAIVisionPanelBytes(ctx, videoPath, duration)
}

func (g Generator) buildAIVisionPanelBytes(ctx context.Context, videoPath string, duration float64) ([][]byte, error) {
	framesPerPanel := AIVisionPanelCols * AIVisionPanelRows
	totalFrames := AIVisionPanelCount * framesPerPanel
	if totalFrames > AIVisionSampleCount {
		totalFrames = AIVisionSampleCount
	}

	times := clampAIVisionSampleTimes(buildAIVisionSampleTimes(duration, totalFrames), duration)
	boostLastPanelSampleTimes(times, duration, framesPerPanel)

	var panels [][]byte
	var lastGood image.Image

	for p := 0; p < AIVisionPanelCount; p++ {
		start := p * framesPerPanel
		if start >= len(times) {
			break
		}

		end := start + framesPerPanel
		if end > len(times) {
			end = len(times)
		}

		panelTimes := times[start:end]
		images := make([]image.Image, 0, framesPerPanel)

		for _, t := range panelTimes {
			img, err := g.extractAIVisionFrame(ctx, videoPath, t, duration, lastGood)
			if err != nil {
				logger.Warnf("[ai-vision] frame at %.2fs failed, using fallback: %v", t, err)
				if lastGood != nil {
					images = append(images, lastGood)
				} else {
					images = append(images, blankAIVisionFrame())
				}
				continue
			}

			lastGood = img
			images = append(images, img)
		}

		for len(images) < framesPerPanel {
			if lastGood != nil {
				images = append(images, lastGood)
			} else {
				images = append(images, blankAIVisionFrame())
			}
		}

		montage := combineAIVisionMontage(images, AIVisionPanelCols, AIVisionPanelRows)
		data, err := encodeAIVisionJPEG(montage)
		if err != nil {
			return nil, err
		}
		panels = append(panels, data)
	}

	if len(panels) == 0 {
		return nil, fmt.Errorf("no AI vision panels could be generated")
	}

	return panels, nil
}

func (g Generator) extractAIVisionFrame(ctx context.Context, videoPath string, t float64, duration float64, fallback image.Image) (image.Image, error) {
	attempts := []struct {
		time   float64
		hybrid bool
	}{
		{clampAIVisionTime(t, duration), false},
		{clampAIVisionTime(t, duration), true},
	}

	if t > 1 {
		attempts = append(attempts,
			struct {
				time   float64
				hybrid bool
			}{clampAIVisionTime(t-1, duration), true},
			struct {
				time   float64
				hybrid bool
			}{clampAIVisionTime(duration*0.99, duration), true},
			struct {
				time   float64
				hybrid bool
			}{clampAIVisionTime(duration*0.95, duration), true},
		)
	}

	var lastErr error
	for _, attempt := range attempts {
		var img image.Image
		var err error
		if attempt.hybrid {
			img, err = g.SpriteScreenshotHybrid(ctx, videoPath, attempt.time)
		} else {
			img, err = g.SpriteScreenshot(ctx, videoPath, attempt.time)
		}
		if err == nil {
			return img, nil
		}
		lastErr = err
	}

	if fallback != nil {
		return fallback, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to extract frame at %.2fs: %w", t, lastErr)
	}

	return nil, fmt.Errorf("failed to extract frame at %.2fs", t)
}

func clampAIVisionTime(t, duration float64) float64 {
	maxT := duration - aiVisionEndSafetyMargin
	if maxT < 0 {
		maxT = 0
	}
	if t < 0 {
		return 0
	}
	if t > maxT {
		return maxT
	}
	return t
}

func clampAIVisionSampleTimes(times []float64, duration float64) []float64 {
	clamped := make([]float64, len(times))
	for i, t := range times {
		clamped[i] = clampAIVisionTime(t, duration)
	}
	return clamped
}

func blankAIVisionFrame() image.Image {
	height := spriteScreenshotWidth * 9 / 16
	if height < 1 {
		height = 1
	}
	return imaging.New(spriteScreenshotWidth, height, color.NRGBA{R: 32, G: 32, B: 32, A: 255})
}

// buildAIVisionSampleTimes returns chronologically sorted timestamps with extra density near the end.
func buildAIVisionSampleTimes(duration float64, count int) []float64 {
	if count <= 0 || duration <= 0 {
		return nil
	}

	endingCount := int(math.Round(float64(count) * aiVisionEndingSampleFraction))
	if endingCount < 1 {
		endingCount = 1
	}
	if endingCount >= count {
		endingCount = count - 1
	}
	uniformCount := count - endingCount

	endingStart := duration * (1 - aiVisionEndingPortion)
	if endingStart < 0 {
		endingStart = 0
	}
	endTime := clampAIVisionTime(duration, duration)

	times := make([]float64, 0, count)

	for i := 0; i < uniformCount; i++ {
		var t float64
		if uniformCount == 1 {
			t = 0
		} else {
			t = endingStart * float64(i) / float64(uniformCount-1)
		}
		times = append(times, t)
	}

	for i := 0; i < endingCount; i++ {
		var t float64
		if endingCount == 1 {
			t = endTime
		} else {
			span := endTime - endingStart
			if span < 0 {
				span = 0
			}
			t = endingStart + span*float64(i)/float64(endingCount-1)
		}
		times = append(times, t)
	}

	sort.Float64s(times)

	// Collapse near-duplicate timestamps from rounding.
	unique := times[:0]
	const minGap = 0.05
	for _, t := range times {
		if len(unique) == 0 || t-unique[len(unique)-1] >= minGap {
			unique = append(unique, t)
		}
	}

	// If deduplication dropped samples, pad from the ending region.
	for len(unique) < count {
		unique = append(unique, endTime)
	}

	if len(unique) > count {
		unique = unique[:count]
	}

	return unique
}

// boostLastPanelSampleTimes resamples the final panel into the last 10% of the video for climax/finish detection.
func boostLastPanelSampleTimes(times []float64, duration float64, framesPerPanel int) {
	if len(times) < framesPerPanel || duration <= 0 {
		return
	}

	lastPanelStart := len(times) - framesPerPanel
	if lastPanelStart < 0 {
		lastPanelStart = 0
	}

	start := duration * 0.90
	end := duration - aiVisionEndSafetyMargin
	if end < start {
		end = start
	}
	span := end - start
	count := len(times) - lastPanelStart

	for i := lastPanelStart; i < len(times); i++ {
		idx := i - lastPanelStart
		var t float64
		if count <= 1 || span <= 0 {
			t = end
		} else {
			t = start + span*float64(idx)/float64(count-1)
		}
		times[i] = clampAIVisionTime(t, duration)
	}
}

func encodeAIVisionMontagePanels(images []image.Image, cols, rows int) ([][]byte, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("no images to encode")
	}

	framesPerPanel := cols * rows
	panelCount := int(math.Ceil(float64(len(images)) / float64(framesPerPanel)))
	if panelCount > AIVisionPanelCount {
		panelCount = AIVisionPanelCount
	}

	var panels [][]byte
	for p := 0; p < panelCount; p++ {
		start := p * framesPerPanel
		end := start + framesPerPanel
		if end > len(images) {
			end = len(images)
		}
		chunk := images[start:end]
		if len(chunk) == 0 {
			break
		}

		montage := combineAIVisionMontage(chunk, cols, rows)
		data, err := encodeAIVisionJPEG(montage)
		if err != nil {
			return nil, err
		}
		panels = append(panels, data)
	}

	return panels, nil
}

func combineAIVisionMontage(images []image.Image, cols, rows int) image.Image {
	width := images[0].Bounds().Dx()
	height := images[0].Bounds().Dy()
	canvasW := width * cols
	canvasH := height * rows
	montage := imaging.New(canvasW, canvasH, color.NRGBA{})

	for index, img := range images {
		if index >= cols*rows {
			break
		}
		x := width * (index % cols)
		y := height * (index / cols)
		montage = imaging.Paste(montage, img, image.Pt(x, y))
	}

	return montage
}

func encodeAIVisionJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(85)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
