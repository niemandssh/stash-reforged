package manager

import (
	"fmt"
	"strings"

	"github.com/stashapp/stash/pkg/embed"
)

// ResolveEmbedDuration returns the duration of a remote embed video in seconds.
// Tube sites use page metadata; direct media URLs are probed with ffprobe.
func ResolveEmbedDuration(embedURL string) (float64, error) {
	embedURL = strings.TrimSpace(embedURL)
	if embedURL == "" {
		return 0, fmt.Errorf("empty embed URL")
	}

	duration, site, err := embed.ExtractVideoDuration(embedURL)
	if err == nil && duration > 0 {
		return duration, nil
	}

	if site == embed.SiteDirect || site == embed.SitePlayerJS || site == embed.SiteKVS || site == embed.SiteJWPlayer || site == embed.SiteNewgrounds {
		return probeDirectEmbedDuration(embedURL)
	}

	if err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("invalid embed video duration")
}

func probeDirectEmbedDuration(embedURL string) (float64, error) {
	streamURL, _, err := embed.ExtractStreamURL(embedURL)
	if err != nil {
		return 0, err
	}
	if instance.FFProbe == nil {
		return 0, fmt.Errorf("ffprobe not available")
	}

	vf, err := instance.FFProbe.NewVideoFileWithHeaders(streamURL, embed.StreamHTTPHeadersForMediaURL(embedURL, streamURL))
	if err != nil {
		return 0, err
	}

	duration := vf.FileDuration
	if vf.VideoStreamDuration > 0 {
		duration = vf.VideoStreamDuration
	}
	if duration <= 0 {
		return 0, fmt.Errorf("invalid probed duration")
	}
	return duration, nil
}
