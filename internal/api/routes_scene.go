package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/embed"
	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/file/video"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

type SceneFinder interface {
	models.SceneGetter

	FindByChecksum(ctx context.Context, checksum string) ([]*models.Scene, error)
	FindByOSHash(ctx context.Context, oshash string) ([]*models.Scene, error)
	GetCover(ctx context.Context, sceneID int) ([]byte, error)
}

type SceneMarkerFinder interface {
	models.SceneMarkerGetter
	FindBySceneID(ctx context.Context, sceneID int) ([]*models.SceneMarker, error)
}

type SceneMarkerTagFinder interface {
	models.TagGetter
	FindBySceneMarkerID(ctx context.Context, sceneMarkerID int) ([]*models.Tag, error)
}

type CaptionFinder interface {
	GetCaptions(ctx context.Context, fileID models.FileID) ([]*models.VideoCaption, error)
}

type sceneRoutes struct {
	routes
	sceneFinder       SceneFinder
	fileGetter        models.FileGetter
	captionFinder     CaptionFinder
	sceneMarkerFinder SceneMarkerFinder
	tagFinder         SceneMarkerTagFinder
}

func (rs sceneRoutes) Routes() chi.Router {
	r := chi.NewRouter()

	// Hash-based sprite routes must be registered before /{sceneId}, otherwise chi
	// treats e.g. "remote_4030_thumbs.vtt" as a scene ID and SceneCtx returns 400.
	r.Get("/{sceneHash}_thumbs.vtt", rs.VttThumbs)
	r.Get("/{sceneHash}_sprite.jpg", rs.VttSprite)

	r.Route("/{sceneId}", func(r chi.Router) {
		r.Use(rs.SceneCtx)

		// streaming endpoints
		r.Get("/stream", rs.StreamDirect)
		r.Get("/stream.mp4", rs.StreamMp4)
		r.Get("/stream.webm", rs.StreamWebM)
		r.Get("/stream.mkv", rs.StreamMKV)
		r.Get("/stream.m3u8", rs.StreamHLS)
		r.Get("/stream.m3u8/{segment}.ts", rs.StreamHLSSegment)
		r.Get("/stream.mpd", rs.StreamDASH)
		r.Get("/stream.mpd/{segment}_v.webm", rs.StreamDASHVideoSegment)
		r.Get("/stream.mpd/{segment}_a.webm", rs.StreamDASHAudioSegment)

		r.Get("/screenshot", rs.Screenshot)
		r.Get("/preview", rs.Preview)
		r.Get("/webp", rs.Webp)
		r.Get("/vtt/chapter", rs.VttChapter)
		r.Get("/vtt/thumbs", rs.VttThumbs)
		r.Get("/vtt/sprite", rs.VttSprite)
		r.Get("/vtt/{asset}", rs.VttSpriteAsset)
		r.Get("/funscript", rs.Funscript)
		r.Get("/interactive_csv", rs.InteractiveCSV)
		r.Get("/interactive_heatmap", rs.InteractiveHeatmap)
		r.Get("/caption", rs.CaptionLang)
		r.Get("/embed/stream.m3u8", rs.EmbedStreamManifest)
		r.Get("/embed/proxy", rs.EmbedStreamProxy)

		r.Get("/scene_marker/{sceneMarkerId}/stream", rs.SceneMarkerStream)
		r.Get("/scene_marker/{sceneMarkerId}/preview", rs.SceneMarkerPreview)
		r.Get("/scene_marker/{sceneMarkerId}/screenshot", rs.SceneMarkerScreenshot)
	})

	return r
}

func (rs sceneRoutes) StreamDirect(w http.ResponseWriter, r *http.Request) {
	scene := r.Context().Value(sceneKey).(*models.Scene)
	ss := manager.SceneServer{
		TxnManager:       rs.txnManager,
		SceneCoverGetter: rs.sceneFinder,
	}
	ss.StreamSceneDirect(scene, w, r)
}

func (rs sceneRoutes) StreamMp4(w http.ResponseWriter, r *http.Request) {
	rs.streamTranscode(w, r, ffmpeg.StreamTypeMP4)
}

func (rs sceneRoutes) StreamWebM(w http.ResponseWriter, r *http.Request) {
	rs.streamTranscode(w, r, ffmpeg.StreamTypeWEBM)
}

func (rs sceneRoutes) StreamMKV(w http.ResponseWriter, r *http.Request) {
	// only allow mkv streaming if the scene container is an mkv already
	scene := r.Context().Value(sceneKey).(*models.Scene)

	pf := scene.Files.Primary()
	if pf == nil {
		return
	}

	container, err := manager.GetVideoFileContainer(pf)
	if err != nil {
		logger.Errorf("[transcode] error getting container: %v", err)
	}

	if container != ffmpeg.Matroska {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("not an mkv file")); err != nil {
			logger.Warnf("[stream] error writing to stream: %v", err)
		}
		return
	}

	rs.streamTranscode(w, r, ffmpeg.StreamTypeMKV)
}

func (rs sceneRoutes) streamTranscode(w http.ResponseWriter, r *http.Request, streamType ffmpeg.StreamFormat) {
	scene := r.Context().Value(sceneKey).(*models.Scene)

	streamManager := manager.GetInstance().StreamManager
	if streamManager == nil {
		http.Error(w, "Live transcoding disabled", http.StatusServiceUnavailable)
		return
	}

	f := scene.Files.Primary()
	if f == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		logger.Warnf("[transcode] error parsing query form: %v", err)
	}

	startTime := r.Form.Get("start")
	ss, _ := strconv.ParseFloat(startTime, 64)
	resolution := r.Form.Get("resolution")

	options := ffmpeg.TranscodeOptions{
		StreamType: streamType,
		VideoFile:  f,
		Resolution: resolution,
		StartTime:  ss,
	}

	logger.Debugf("[transcode] streaming scene %d as %s", scene.ID, streamType.MimeType)
	streamManager.ServeTranscode(w, r, options)
}

func (rs sceneRoutes) StreamHLS(w http.ResponseWriter, r *http.Request) {
	rs.streamManifest(w, r, ffmpeg.StreamTypeHLS, "HLS")
}

func (rs sceneRoutes) StreamDASH(w http.ResponseWriter, r *http.Request) {
	rs.streamManifest(w, r, ffmpeg.StreamTypeDASHVideo, "DASH")
}

func (rs sceneRoutes) streamManifest(w http.ResponseWriter, r *http.Request, streamType *ffmpeg.StreamType, logName string) {
	scene := r.Context().Value(sceneKey).(*models.Scene)

	streamManager := manager.GetInstance().StreamManager
	if streamManager == nil {
		http.Error(w, "Live transcoding disabled", http.StatusServiceUnavailable)
		return
	}

	f := scene.Files.Primary()
	if f == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		logger.Warnf("[transcode] error parsing query form: %v", err)
	}

	resolution := r.Form.Get("resolution")

	logger.Debugf("[transcode] returning %s manifest for scene %d", logName, scene.ID)
	streamManager.ServeManifest(w, r, streamType, f, resolution)
}

func (rs sceneRoutes) StreamHLSSegment(w http.ResponseWriter, r *http.Request) {
	rs.streamSegment(w, r, ffmpeg.StreamTypeHLS)
}

func (rs sceneRoutes) StreamDASHVideoSegment(w http.ResponseWriter, r *http.Request) {
	rs.streamSegment(w, r, ffmpeg.StreamTypeDASHVideo)
}

func (rs sceneRoutes) StreamDASHAudioSegment(w http.ResponseWriter, r *http.Request) {
	rs.streamSegment(w, r, ffmpeg.StreamTypeDASHAudio)
}

func (rs sceneRoutes) streamSegment(w http.ResponseWriter, r *http.Request, streamType *ffmpeg.StreamType) {
	scene := r.Context().Value(sceneKey).(*models.Scene)

	streamManager := manager.GetInstance().StreamManager
	if streamManager == nil {
		http.Error(w, "Live transcoding disabled", http.StatusServiceUnavailable)
		return
	}

	f := scene.Files.Primary()
	if f == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		logger.Warnf("[transcode] error parsing query form: %v", err)
	}

	sceneHash := scene.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())

	segment := chi.URLParam(r, "segment")
	resolution := r.Form.Get("resolution")

	options := ffmpeg.StreamOptions{
		StreamType: streamType,
		VideoFile:  f,
		Resolution: resolution,
		Hash:       sceneHash,
		Segment:    segment,
	}

	streamManager.ServeSegment(w, r, options)
}

func (rs sceneRoutes) Screenshot(w http.ResponseWriter, r *http.Request) {
	scene := r.Context().Value(sceneKey).(*models.Scene)

	ss := manager.SceneServer{
		TxnManager:       rs.txnManager,
		SceneCoverGetter: rs.sceneFinder,
	}
	ss.ServeScreenshot(scene, w, r)
}

func (rs sceneRoutes) Preview(w http.ResponseWriter, r *http.Request) {
	scene := r.Context().Value(sceneKey).(*models.Scene)
	sceneHash := scene.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	filepath := manager.GetInstance().Paths.Scene.GetVideoPreviewPath(sceneHash)

	utils.ServeStaticFile(w, r, filepath)
}

func (rs sceneRoutes) Webp(w http.ResponseWriter, r *http.Request) {
	scene := r.Context().Value(sceneKey).(*models.Scene)
	sceneHash := scene.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	filepath := manager.GetInstance().Paths.Scene.GetWebpPreviewPath(sceneHash)

	utils.ServeStaticFile(w, r, filepath)
}

func (rs sceneRoutes) getChapterVttTitle(r *http.Request, marker *models.SceneMarker) (*string, error) {
	if marker.Title != "" {
		return &marker.Title, nil
	}

	var title string
	if err := rs.withReadTxn(r, func(ctx context.Context) error {
		qb := rs.tagFinder
		primaryTag, err := qb.Find(ctx, marker.PrimaryTagID)
		if err != nil {
			return err
		}

		title = primaryTag.Name

		tags, err := qb.FindBySceneMarkerID(ctx, marker.ID)
		if err != nil {
			return err
		}

		for _, t := range tags {
			title += ", " + t.Name
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &title, nil
}

func (rs sceneRoutes) VttChapter(w http.ResponseWriter, r *http.Request) {
	scene := r.Context().Value(sceneKey).(*models.Scene)
	var sceneMarkers []*models.SceneMarker
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		sceneMarkers, err = rs.sceneMarkerFinder.FindBySceneID(ctx, scene.ID)
		return err
	})
	if errors.Is(readTxnErr, context.Canceled) {
		return
	}
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch scene markers: %v", readTxnErr)
		http.Error(w, readTxnErr.Error(), http.StatusInternalServerError)
		return
	}

	vttLines := []string{"WEBVTT", ""}
	for i, marker := range sceneMarkers {
		vttLines = append(vttLines, strconv.Itoa(i+1))
		time := utils.GetVTTTime(marker.Seconds)
		vttLines = append(vttLines, time+" --> "+time)

		vttTitle, err := rs.getChapterVttTitle(r, marker)
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			logger.Warnf("read transaction error on fetch scene marker title: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		vttLines = append(vttLines, *vttTitle)
		vttLines = append(vttLines, "")
	}
	vtt := strings.Join(vttLines, "\n")

	w.Header().Set("Content-Type", "text/vtt")
	utils.ServeStaticContent(w, r, []byte(vtt))
}

func (rs sceneRoutes) VttThumbs(w http.ResponseWriter, r *http.Request) {
	scene, ok := r.Context().Value(sceneKey).(*models.Scene)
	var sceneHash string
	if ok && scene != nil {
		sceneHash = manager.SceneSpriteHash(scene, config.GetInstance().GetVideoFileNamingAlgorithm())
	} else {
		sceneHash = chi.URLParam(r, "sceneHash")
	}
	filepath := manager.GetInstance().Paths.Scene.GetSpriteVttFilePath(sceneHash)

	w.Header().Set("Content-Type", "text/vtt")

	if ok && scene != nil && manager.IsEmbedSpriteScene(scene) {
		if !manager.EmbedSpritesReady(scene) {
			if started, err := manager.EnsureEmbedSpriteJob(r.Context(), scene); err != nil {
				logger.Warnf("failed to start embed sprite generation for scene %d: %v", scene.ID, err)
			} else if started {
				logger.Infof("started embed sprite generation for scene %d", scene.ID)
			}
			w.Header().Set("Retry-After", "5")
			http.Error(w, "embed sprites are being generated", http.StatusServiceUnavailable)
			return
		}

		data, err := os.ReadFile(filepath)
		if err != nil {
			utils.ServeStaticFile(w, r, filepath)
			return
		}
		utils.ServeStaticContent(w, r, rewriteEmbedSpriteVTT(data))
		return
	}

	utils.ServeStaticFile(w, r, filepath)
}

func (rs sceneRoutes) VttSpriteAsset(w http.ResponseWriter, r *http.Request) {
	asset := chi.URLParam(r, "asset")
	if asset == "thumbs" {
		http.NotFound(w, r)
		return
	}
	if asset == "sprite" || strings.HasSuffix(asset, "_sprite.jpg") {
		rs.VttSprite(w, r)
		return
	}
	http.NotFound(w, r)
}

var embedSpriteVTTLineRe = regexp.MustCompile(`^(\S+)_sprite\.jpg(#xywh=)`)

func rewriteEmbedSpriteVTT(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if embedSpriteVTTLineRe.MatchString(line) {
			lines[i] = embedSpriteVTTLineRe.ReplaceAllString(line, "sprite$2")
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func (rs sceneRoutes) VttSprite(w http.ResponseWriter, r *http.Request) {
	scene, ok := r.Context().Value(sceneKey).(*models.Scene)
	var sceneHash string
	if ok && scene != nil {
		sceneHash = manager.SceneSpriteHash(scene, config.GetInstance().GetVideoFileNamingAlgorithm())
	} else {
		sceneHash = chi.URLParam(r, "sceneHash")
	}
	filepath := manager.GetInstance().Paths.Scene.GetSpriteImageFilePath(sceneHash)

	if ok && scene != nil && manager.IsEmbedSpriteScene(scene) {
		if !manager.EmbedSpritesReady(scene) {
			if started, err := manager.EnsureEmbedSpriteJob(r.Context(), scene); err != nil {
				logger.Warnf("failed to start embed sprite generation for scene %d: %v", scene.ID, err)
			} else if started {
				logger.Infof("started embed sprite generation for scene %d", scene.ID)
			}
			w.Header().Set("Retry-After", "5")
			http.Error(w, "embed sprites are being generated", http.StatusServiceUnavailable)
			return
		}
	}

	utils.ServeStaticFile(w, r, filepath)
}

func (rs sceneRoutes) Funscript(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(sceneKey).(*models.Scene)
	filepath := video.GetFunscriptPath(s.Path)

	utils.ServeStaticFile(w, r, filepath)
}

func (rs sceneRoutes) InteractiveCSV(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(sceneKey).(*models.Scene)
	filepath := video.GetFunscriptPath(s.Path)

	// TheHandy directly only accepts interactive CSVs
	csvBytes, err := manager.ConvertFunscriptToCSV(filepath)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	utils.ServeStaticContent(w, r, csvBytes)
}

func (rs sceneRoutes) InteractiveHeatmap(w http.ResponseWriter, r *http.Request) {
	scene := r.Context().Value(sceneKey).(*models.Scene)
	sceneHash := scene.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	filepath := manager.GetInstance().Paths.Scene.GetInteractiveHeatmapPath(sceneHash)

	utils.ServeStaticFile(w, r, filepath)
}

func (rs sceneRoutes) Caption(w http.ResponseWriter, r *http.Request, lang string, ext string) {
	s := r.Context().Value(sceneKey).(*models.Scene)

	var captions []*models.VideoCaption
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		primaryFile := s.Files.Primary()
		if primaryFile == nil {
			return nil
		}

		captions, err = rs.captionFinder.GetCaptions(ctx, primaryFile.Base().ID)

		return err
	})
	if errors.Is(readTxnErr, context.Canceled) {
		return
	}
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch scene captions: %v", readTxnErr)
		http.Error(w, readTxnErr.Error(), http.StatusInternalServerError)
		return
	}

	for _, caption := range captions {
		if lang != caption.LanguageCode || ext != caption.CaptionType {
			continue
		}

		sub, err := video.ReadSubs(caption.Path(s.Path))
		if err != nil {
			logger.Warnf("error while reading subs: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer

		err = sub.WriteToWebVTT(&buf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/vtt")
		utils.ServeStaticContent(w, r, buf.Bytes())
		return
	}
}

func (rs sceneRoutes) CaptionLang(w http.ResponseWriter, r *http.Request) {
	// serve caption based on lang query param, if provided
	if err := r.ParseForm(); err != nil {
		logger.Warnf("[caption] error parsing query form: %v", err)
	}

	l := r.Form.Get("lang")
	ext := r.Form.Get("type")
	rs.Caption(w, r, l, ext)
}

func (rs sceneRoutes) SceneMarkerStream(w http.ResponseWriter, r *http.Request) {
	scene := r.Context().Value(sceneKey).(*models.Scene)
	sceneHash := scene.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	sceneMarkerID, _ := strconv.Atoi(chi.URLParam(r, "sceneMarkerId"))
	var sceneMarker *models.SceneMarker
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		sceneMarker, err = rs.sceneMarkerFinder.Find(ctx, sceneMarkerID)
		return err
	})
	if errors.Is(readTxnErr, context.Canceled) {
		return
	}
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch scene marker: %v", readTxnErr)
		http.Error(w, readTxnErr.Error(), http.StatusInternalServerError)
		return
	}

	if sceneMarker == nil {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	filepath := manager.GetInstance().Paths.SceneMarkers.GetVideoPreviewPath(sceneHash, int(sceneMarker.Seconds))
	utils.ServeStaticFile(w, r, filepath)
}

func (rs sceneRoutes) SceneMarkerPreview(w http.ResponseWriter, r *http.Request) {
	scene := r.Context().Value(sceneKey).(*models.Scene)
	sceneHash := scene.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	sceneMarkerID, _ := strconv.Atoi(chi.URLParam(r, "sceneMarkerId"))
	var sceneMarker *models.SceneMarker
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		sceneMarker, err = rs.sceneMarkerFinder.Find(ctx, sceneMarkerID)
		return err
	})
	if errors.Is(readTxnErr, context.Canceled) {
		return
	}
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch scene marker preview: %v", readTxnErr)
		http.Error(w, readTxnErr.Error(), http.StatusInternalServerError)
		return
	}

	if sceneMarker == nil {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	filepath := manager.GetInstance().Paths.SceneMarkers.GetWebpPreviewPath(sceneHash, int(sceneMarker.Seconds))

	// If the image doesn't exist, send the placeholder
	exists, _ := fsutil.FileExists(filepath)
	if !exists {
		w.Header().Set("Content-Type", "image/png")
		utils.ServeStaticContent(w, r, utils.PendingGenerateResource)
	} else {
		utils.ServeStaticFile(w, r, filepath)
	}
}

func (rs sceneRoutes) SceneMarkerScreenshot(w http.ResponseWriter, r *http.Request) {
	scene := r.Context().Value(sceneKey).(*models.Scene)
	sceneHash := scene.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	sceneMarkerID, _ := strconv.Atoi(chi.URLParam(r, "sceneMarkerId"))
	var sceneMarker *models.SceneMarker
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		sceneMarker, err = rs.sceneMarkerFinder.Find(ctx, sceneMarkerID)
		return err
	})
	if errors.Is(readTxnErr, context.Canceled) {
		return
	}
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch scene marker screenshot: %v", readTxnErr)
		http.Error(w, readTxnErr.Error(), http.StatusInternalServerError)
		return
	}

	if sceneMarker == nil {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	filepath := manager.GetInstance().Paths.SceneMarkers.GetScreenshotPath(sceneHash, int(sceneMarker.Seconds))

	// If the image doesn't exist, send the placeholder
	exists, _ := fsutil.FileExists(filepath)
	if !exists {
		w.Header().Set("Content-Type", "image/png")
		utils.ServeStaticContent(w, r, utils.PendingGenerateResource)
	} else {
		utils.ServeStaticFile(w, r, filepath)
	}
}

func (rs sceneRoutes) EmbedStreamManifest(w http.ResponseWriter, r *http.Request) {
	scene := r.Context().Value(sceneKey).(*models.Scene)
	if scene.EmbedURL == nil || strings.TrimSpace(*scene.EmbedURL) == "" {
		http.Error(w, "scene has no embed URL", http.StatusNotFound)
		return
	}

	requestedHeight := 0
	requestedBandwidth := 0
	if heightParam := r.URL.Query().Get("height"); heightParam != "" {
		parsed, err := strconv.Atoi(heightParam)
		if err != nil || parsed < 0 {
			http.Error(w, "invalid height parameter", http.StatusBadRequest)
			return
		}
		requestedHeight = parsed
	}
	if bandwidthParam := r.URL.Query().Get("bandwidth"); bandwidthParam != "" {
		parsed, err := strconv.Atoi(bandwidthParam)
		if err != nil || parsed < 0 {
			http.Error(w, "invalid bandwidth parameter", http.StatusBadRequest)
			return
		}
		requestedBandwidth = parsed
	}

	body, manifestURL, err := embed.ResolveMediaPlaylist(
		*scene.EmbedURL,
		requestedHeight,
		requestedBandwidth,
	)
	if err != nil {
		rs.writeEmbedStreamError(w, scene.ID, err)
		return
	}

	sceneID := chi.URLParam(r, "sceneId")
	proxyPath := "/scene/" + sceneID + "/embed/proxy"
	rewritten, err := embed.RewriteM3U8(body, manifestURL, proxyPath)
	if err != nil {
		logger.Warnf("embed stream manifest rewrite failed for scene %d: %v", scene.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(rewritten)
}

func (rs sceneRoutes) EmbedStreamProxy(w http.ResponseWriter, r *http.Request) {
	upstream := r.URL.Query().Get("u")
	if upstream == "" {
		http.Error(w, "missing upstream URL", http.StatusBadRequest)
		return
	}

	if !embed.IsAllowedProxyURL(upstream) {
		http.Error(w, "upstream URL not allowed", http.StatusForbidden)
		return
	}

	body, contentType, err := embed.FetchStreamResource(upstream)
	if err != nil {
		logger.Warnf("embed stream proxy fetch failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if embed.IsM3U8Content(contentType, upstream) {
		sceneID := chi.URLParam(r, "sceneId")
		proxyPath := "/scene/" + sceneID + "/embed/proxy"
		rewritten, err := embed.RewriteM3U8(body, upstream, proxyPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		body = rewritten
		contentType = "application/vnd.apple.mpegurl"
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(body)
}

func (rs sceneRoutes) writeEmbedStreamError(w http.ResponseWriter, sceneID int, err error) {
	if errors.Is(err, embed.ErrStreamNotSupported) || errors.Is(err, embed.ErrUnsupportedSite) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, embed.ErrStreamNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	logger.Warnf("embed stream extraction failed for scene %d: %v", sceneID, err)
	http.Error(w, err.Error(), http.StatusBadGateway)
}

func (rs sceneRoutes) SceneCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sceneID, err := strconv.Atoi(chi.URLParam(r, "sceneId"))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var scene *models.Scene
		_ = rs.withReadTxn(r, func(ctx context.Context) error {
			qb := rs.sceneFinder
			scene, _ = qb.Find(ctx, sceneID)

			if scene != nil {
				if err := scene.LoadPrimaryFile(ctx, rs.fileGetter); err != nil {
					if !errors.Is(err, context.Canceled) {
						logger.Errorf("error loading primary file for scene %d: %v", sceneID, err)
					}
					// set scene to nil so that it doesn't try to use the primary file
					scene = nil
				}
			}

			return nil
		})
		if scene == nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}

		ctx := context.WithValue(r.Context(), sceneKey, scene)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
