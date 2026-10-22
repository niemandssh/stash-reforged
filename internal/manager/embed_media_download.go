package manager

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stashapp/stash/pkg/embed"
)

var embedMediaDownloadClient = &http.Client{
	Timeout: 0,
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: 60 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	},
}

// prepareEmbedSpriteSource resolves a local or remote path for sprite generation.
// Temporary PlayerJS/CDN URLs are downloaded once so short-lived auth tokens cannot
// expire across dozens of ffmpeg seeks.
func prepareEmbedSpriteSource(ctx context.Context, embedURL string) (input string, httpHeaders string, cleanup func(), err error) {
	cleanup = func() {}

	streamURL, site, err := embed.ExtractStreamURL(embedURL)
	if err != nil {
		return "", "", cleanup, err
	}

	headers := embed.StreamHTTPHeadersForMediaURL(embedURL, streamURL)
	if !shouldDownloadEmbedMedia(site, streamURL) {
		return streamURL, headers, cleanup, nil
	}

	tmpPath, err := downloadEmbedMediaTemp(ctx, streamURL, headers)
	if err != nil {
		return "", "", cleanup, err
	}

	cleanup = func() {
		_ = os.Remove(tmpPath)
	}
	return tmpPath, "", cleanup, nil
}

func shouldDownloadEmbedMedia(site embed.Site, streamURL string) bool {
	if embed.IsPlayerJSSite(site) || embed.IsKVSSite(site) || embed.IsJWPlayerSite(site) {
		return true
	}
	return embed.LooksLikeTemporaryMediaURL(streamURL)
}

func downloadEmbedMediaTemp(ctx context.Context, mediaURL, headerBlock string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return "", err
	}
	applyFFmpegStyleHeaders(req, headerBlock)
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	}

	resp, err := embedMediaDownloadClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download media returned status %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "stash-embed-sprite-*.mp4")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	return tmpPath, nil
}

func applyFFmpegStyleHeaders(req *http.Request, headerBlock string) {
	headerBlock = strings.TrimSpace(headerBlock)
	if headerBlock == "" {
		return
	}
	for _, line := range strings.Split(headerBlock, "\r\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		req.Header.Set(strings.TrimSpace(name), strings.TrimSpace(value))
	}
}
