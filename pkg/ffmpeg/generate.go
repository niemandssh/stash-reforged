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

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting command: %w", err)
	}

	// Читаем stderr для получения прогресса
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			logger.Debugf("[ffmpeg] stderr: %s", line) // Логируем все строки для отладки

			// Парсим прогресс из вывода ffmpeg
			// Ищем строки вида: frame=xxx fps=xx q=xx size=xx time=00:00:xx bitrate=xxx speed=xx
			if strings.Contains(line, "frame=") && strings.Contains(line, "time=") {
				logger.Infof("[ffmpeg] found progress line: %s", line)
				// Извлекаем время из строки
				if timeStr := extractTimeFromFFmpegOutput(line); timeStr != "" {
					logger.Infof("[ffmpeg] extracted time: %s", timeStr)
					if currentTime, err := parseFFmpegTime(timeStr); err == nil {
						// Вычисляем процент прогресса
						if duration > 0 {
							percent := (currentTime / duration) * 100
							if percent > 100 {
								percent = 100
							}
							progress.SetPercent(percent / 100) // SetPercent ожидает значение от 0 до 1

							logger.Infof("[ffmpeg] progress: %s (%.2f/%.2f seconds, %.1f%%)", timeStr, currentTime, duration, percent)
						} else {
							logger.Infof("[ffmpeg] progress: %s (%.2f seconds, duration unknown)", timeStr, currentTime)
						}
					} else {
						logger.Errorf("[ffmpeg] failed to parse time %s: %v", timeStr, err)
					}
				} else {
					logger.Warnf("[ffmpeg] could not extract time from line: %s", line)
				}
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Log the stderr for debugging
			logger.Errorf("[ffmpeg] stderr: %s", string(exitErr.Stderr))

			// Check if the error is related to audio processing
			if strings.Contains(string(exitErr.Stderr), "Error submitting packet to decoder") ||
				strings.Contains(string(exitErr.Stderr), "Invalid data found when processing input") ||
				strings.Contains(string(exitErr.Stderr), "audio processing error") {
				return fmt.Errorf("audio processing error: %w", exitErr)
			}
			return exitErr
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
