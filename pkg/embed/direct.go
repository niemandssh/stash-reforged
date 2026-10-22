package embed

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// SiteDirect is a direct HTTPS media file URL (e.g. .mp4) played in the Stash player.
const SiteDirect Site = "direct"

var directMediaExtensions = map[string]bool{
	".mp4":  true,
	".m4v":  true,
	".webm": true,
	".mov":  true,
}

var progressiveProxyHTTPClient = &http.Client{
	Timeout: 0,
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: 45 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	},
}

// IsDirectMediaSite reports whether the site is a direct media file URL.
func IsDirectMediaSite(site Site) bool {
	return site == SiteDirect
}

// IsProgressiveMediaURL reports whether the URL points at a progressive media file
// (as opposed to an HLS playlist).
func IsProgressiveMediaURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	pathLower := strings.ToLower(parsed.Path)
	// KVS get_file URLs often end with ".mp4/" before the query string.
	if strings.Contains(pathLower, "/get_file/") {
		return true
	}
	return isDirectMediaPath(parsed.Path)
}

func isDirectMediaURL(parsed *url.URL) bool {
	if parsed == nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" {
		return false
	}
	if parsed.Hostname() == "" {
		return false
	}
	return isDirectMediaPath(parsed.Path)
}

func isDirectMediaPath(rawPath string) bool {
	rawPath = strings.TrimRight(strings.TrimSpace(rawPath), "/")
	ext := strings.ToLower(path.Ext(rawPath))
	if i := strings.IndexAny(ext, "?;#"); i >= 0 {
		ext = ext[:i]
	}
	return directMediaExtensions[ext]
}

func parseDirectMedia(raw string) (string, Site, error) {
	if _, err := normalizeDirectMediaURL(raw); err != nil {
		return "", SiteDirect, err
	}
	// Direct media has no iframe embed; playback uses sceneStreams.
	return "", SiteDirect, nil
}

func normalizeDirectMediaURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || !isDirectMediaURL(parsed) {
		return "", ErrInvalidURL
	}

	parsed.Fragment = ""
	return parsed.String(), nil
}

// IsAllowedProxyURLForScene allows known CDNs, or the scene's own direct media URL.
func IsAllowedProxyURLForScene(upstream string, sceneEmbedURL string) bool {
	if IsAllowedProxyURL(upstream) {
		return true
	}
	return isSceneBoundDirectMediaProxy(upstream, sceneEmbedURL)
}

func isSceneBoundDirectMediaProxy(upstream, sceneEmbedURL string) bool {
	sceneEmbedURL = strings.TrimSpace(sceneEmbedURL)
	upstream = strings.TrimSpace(upstream)
	if sceneEmbedURL == "" || upstream == "" {
		return false
	}

	_, site, err := Parse(sceneEmbedURL)
	if err != nil || site != SiteDirect {
		return false
	}

	normalizedEmbed, err := normalizeDirectMediaURL(sceneEmbedURL)
	if err != nil {
		return false
	}

	normalizedUpstream, err := normalizeDirectMediaURL(upstream)
	if err != nil {
		return false
	}

	return mediaURLsEqual(normalizedEmbed, normalizedUpstream)
}

func mediaURLsEqual(a, b string) bool {
	pa, errA := url.Parse(a)
	pb, errB := url.Parse(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(a, b)
	}

	pa.Fragment = ""
	pb.Fragment = ""
	return strings.EqualFold(pa.String(), pb.String())
}

// ProxyProgressiveMedia streams a progressive media file to the client with Range support.
func ProxyProgressiveMedia(w http.ResponseWriter, r *http.Request, upstream string) error {
	return ProxyProgressiveMediaWithHeaders(w, r, upstream, nil)
}

// ProxyProgressiveMediaWithHeaders streams progressive media using optional request header setup.
func ProxyProgressiveMediaWithHeaders(w http.ResponseWriter, r *http.Request, upstream string, headerFn func(*http.Request)) error {
	method := r.Method
	if method != http.MethodGet && method != http.MethodHead {
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(r.Context(), method, upstream, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return err
	}

	if headerFn != nil {
		headerFn(req)
	} else {
		applySiteHeaders(req, SiteDirect)
	}
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}

	resp, err := progressiveProxyHTTPClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		err = fmt.Errorf("upstream returned status %d", resp.StatusCode)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Accept-Ranges", "bytes")

	if v := resp.Header.Get("Content-Length"); v != "" {
		w.Header().Set("Content-Length", v)
	}
	if v := resp.Header.Get("Content-Range"); v != "" {
		w.Header().Set("Content-Range", v)
	}

	w.WriteHeader(resp.StatusCode)
	if method == http.MethodHead {
		return nil
	}

	_, copyErr := io.Copy(w, resp.Body)
	return copyErr
}

// ProxyProgressiveMediaForEmbed proxies progressive media using headers derived from the embed page URL.
func ProxyProgressiveMediaForEmbed(w http.ResponseWriter, r *http.Request, upstream, embedURL string) error {
	return ProxyProgressiveMediaWithHeaders(w, r, upstream, func(req *http.Request) {
		_, site, err := Parse(embedURL)
		if err != nil {
			applySiteHeaders(req, SiteDirect)
			return
		}
		if site == SitePlayerJS || site == SiteJWPlayer {
			applyPageHeaders(req, embedURL)
			req.Header.Set("User-Agent", streamUserAgent)
			return
		}
		applySiteHeaders(req, site)
	})
}
