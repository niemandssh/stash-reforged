package ffmpeg

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
)

// Generate runs ffmpeg with the given args and waits for it to finish.
// Returns an error if the command fails. If the command fails, the return
// value will be of type *exec.ExitError.
func (f *FFMpeg) Generate(ctx context.Context, args Args) error {
	cmd := f.Command(ctx, args)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	logger.Infof("[ffmpeg] running command: %v", args)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting command: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitErr.Stderr = stderr.Bytes()
			logger.Errorf("[ffmpeg] stderr: %s", string(exitErr.Stderr))
			err = exitErr
		}
		return fmt.Errorf("error running ffmpeg command <%s>: %w", strings.Join(args, " "), err)
	}

	return nil
}

// GenerateOutput runs ffmpeg with the given args and returns it standard output.
func (f *FFMpeg) GenerateOutput(ctx context.Context, args []string, stdin io.Reader) ([]byte, error) {
	cmd := f.Command(ctx, args)
	cmd.Stdin = stdin

	ret, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running ffmpeg command <%s>: %w", strings.Join(args, " "), err)
	}

	return ret, nil
}

// GenerateWithProgress runs ffmpeg with the given args and reports progress.
// The progress parameter is used to report conversion progress.
// The duration parameter is the total duration of the video in seconds for progress calculation.
func (f *FFMpeg) GenerateWithProgress(ctx context.Context, args Args, progress *job.Progress, duration float64) error {
	cmd := f.Command(ctx, args)

	logger.Infof("[ffmpeg] running command with progress: %v", args)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting command: %w", err)
	}

	// Capture stderr for error reporting (StderrPipe doesn't populate exec.ExitError.Stderr)
	var stderrBuf bytes.Buffer
	stderr := io.TeeReader(stderrPipe, &stderrBuf)
	var stderrDone sync.WaitGroup
	stderrDone.Add(1)

	// Read stderr for progress and capture for error logging
	go func() {
		defer stderrDone.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			logger.Debugf("[ffmpeg] stderr: %s", line)

			// Parse progress from ffmpeg output
			if strings.Contains(line, "frame=") && strings.Contains(line, "time=") {
				if timeStr := extractTimeFromFFmpegOutput(line); timeStr != "" {
					if currentTime, err := parseFFmpegTime(timeStr); err == nil {
						if duration > 0 {
							percent := (currentTime / duration) * 100
							if percent > 100 {
								percent = 100
							}
							progress.SetPercent(percent / 100)
							logger.Infof("[ffmpeg] progress: %s (%.2f/%.2f seconds, %.1f%%)", timeStr, currentTime, duration, percent)
						} else {
							logger.Infof("[ffmpeg] progress: %s (%.2f seconds, duration unknown)", timeStr, currentTime)
						}
					}
				}
			}
		}
	}()

	waitErr := cmd.Wait()
	stderrDone.Wait() // Ensure stderr is fully captured before reading

	if waitErr != nil {
		err := waitErr
		stderrStr := stderrBuf.String()
		if stderrStr != "" {
			logger.Errorf("[ffmpeg] stderr: %s", stderrStr)
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Check for known error patterns
			if strings.Contains(stderrStr, "Error submitting packet to decoder") ||
				strings.Contains(stderrStr, "Invalid data found when processing input") ||
				strings.Contains(stderrStr, "audio processing error") {
				return fmt.Errorf("audio processing error (exit %d): %s: %w",
					exitErr.ExitCode(), strings.TrimSpace(stderrStr), exitErr)
			}
			if stderrStr != "" {
				return fmt.Errorf("ffmpeg failed (exit %d): %s: %w",
					exitErr.ExitCode(), strings.TrimSpace(stderrStr), exitErr)
			}
			return fmt.Errorf("ffmpeg failed with exit code %d: %w", exitErr.ExitCode(), exitErr)
		}
		return fmt.Errorf("error running ffmpeg command <%s>: %w", strings.Join(args, " "), err)
	}

	return nil
}

// extractTimeFromFFmpegOutput извлекает время из вывода ffmpeg
func extractTimeFromFFmpegOutput(line string) string {
	// Ищем паттерн time=HH:MM:SS.mmm или time=HH:MM:SS
	// Поддерживаем разные форматы времени
	patterns := []string{
		`time=(\d{2}:\d{2}:\d{2}\.\d{2})`,   // HH:MM:SS.mm
		`time=(\d{2}:\d{2}:\d{2})`,          // HH:MM:SS
		`time=(\d{1,2}:\d{2}:\d{2}\.\d{2})`, // H:MM:SS.mm
		`time=(\d{1,2}:\d{2}:\d{2})`,        // H:MM:SS
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// parseFFmpegTime парсит время в формате HH:MM:SS.mmm в секунды
func parseFFmpegTime(timeStr string) (float64, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format: %s", timeStr)
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, err
	}

	return float64(hours)*3600 + float64(minutes)*60 + seconds, nil
}
