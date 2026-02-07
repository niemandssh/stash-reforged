package api

import (
	"net/http"

	"github.com/stashapp/stash/internal/build"
	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

// --- Configuration endpoints ---

// GET /api/v1/config
func (h *RESTHandler) getConfiguration(w http.ResponseWriter, r *http.Request) {
	respondOK(w, makeConfigResult())
}

// PUT /api/v1/config/general
func (h *RESTHandler) configureGeneral(w http.ResponseWriter, r *http.Request) {
	var input ConfigGeneralInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	// Reuse the resolver logic
	resolver := &mutationResolver{&Resolver{
		repository:     h.repository,
		sceneService:   h.sceneService,
		imageService:   h.imageService,
		galleryService: h.galleryService,
		groupService:   h.groupService,
		hookExecutor:   h.hookExecutor,
	}}

	result, err := resolver.ConfigureGeneral(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, result)
}

// PUT /api/v1/config/interface
func (h *RESTHandler) configureInterface(w http.ResponseWriter, r *http.Request) {
	var input ConfigInterfaceInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	resolver := &mutationResolver{&Resolver{
		repository:     h.repository,
		sceneService:   h.sceneService,
		imageService:   h.imageService,
		galleryService: h.galleryService,
		groupService:   h.groupService,
		hookExecutor:   h.hookExecutor,
	}}

	result, err := resolver.ConfigureInterface(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, result)
}

// PUT /api/v1/config/defaults
func (h *RESTHandler) configureDefaults(w http.ResponseWriter, r *http.Request) {
	var input ConfigDefaultSettingsInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	resolver := &mutationResolver{&Resolver{
		repository:     h.repository,
		sceneService:   h.sceneService,
		imageService:   h.imageService,
		galleryService: h.galleryService,
		groupService:   h.groupService,
		hookExecutor:   h.hookExecutor,
	}}

	result, err := resolver.ConfigureDefaults(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, result)
}

// PUT /api/v1/config/ui
func (h *RESTHandler) configureUI(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Input   map[string]interface{} `json:"input,omitempty"`
		Partial map[string]interface{} `json:"partial,omitempty"`
	}
	if err := decodeBody(r, &body); err != nil {
		respondBadRequest(w, err)
		return
	}

	resolver := &mutationResolver{&Resolver{
		repository:     h.repository,
		sceneService:   h.sceneService,
		imageService:   h.imageService,
		galleryService: h.galleryService,
		groupService:   h.groupService,
		hookExecutor:   h.hookExecutor,
	}}

	result, err := resolver.ConfigureUI(r.Context(), body.Input, body.Partial)
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, result)
}

// PUT /api/v1/config/ui/setting
func (h *RESTHandler) configureUISetting(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}
	if err := decodeBody(r, &body); err != nil {
		respondBadRequest(w, err)
		return
	}

	resolver := &mutationResolver{&Resolver{
		repository:     h.repository,
		sceneService:   h.sceneService,
		imageService:   h.imageService,
		galleryService: h.galleryService,
		groupService:   h.groupService,
		hookExecutor:   h.hookExecutor,
	}}

	result, err := resolver.ConfigureUISetting(r.Context(), body.Key, body.Value)
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, result)
}

// PUT /api/v1/config/scraping
func (h *RESTHandler) configureScraping(w http.ResponseWriter, r *http.Request) {
	var input ConfigScrapingInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	resolver := &mutationResolver{&Resolver{
		repository:     h.repository,
		sceneService:   h.sceneService,
		imageService:   h.imageService,
		galleryService: h.galleryService,
		groupService:   h.groupService,
		hookExecutor:   h.hookExecutor,
	}}

	result, err := resolver.ConfigureScraping(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, result)
}

// PUT /api/v1/config/dlna
func (h *RESTHandler) configureDLNA(w http.ResponseWriter, r *http.Request) {
	var input ConfigDLNAInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	resolver := &mutationResolver{&Resolver{
		repository:     h.repository,
		sceneService:   h.sceneService,
		imageService:   h.imageService,
		galleryService: h.galleryService,
		groupService:   h.groupService,
		hookExecutor:   h.hookExecutor,
	}}

	result, err := resolver.ConfigureDlna(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, result)
}

// PUT /api/v1/config/plugin/{pluginId}
func (h *RESTHandler) configurePlugin(w http.ResponseWriter, r *http.Request) {
	pluginID := urlParamString(r, "pluginId")
	var input map[string]interface{}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	resolver := &mutationResolver{&Resolver{
		repository:     h.repository,
		sceneService:   h.sceneService,
		imageService:   h.imageService,
		galleryService: h.galleryService,
		groupService:   h.groupService,
		hookExecutor:   h.hookExecutor,
	}}

	result, err := resolver.ConfigurePlugin(r.Context(), pluginID, input)
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, result)
}

// POST /api/v1/config/api-key
func (h *RESTHandler) generateAPIKey(w http.ResponseWriter, r *http.Request) {
	var input GenerateAPIKeyInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	resolver := &mutationResolver{&Resolver{
		repository:     h.repository,
		sceneService:   h.sceneService,
		imageService:   h.imageService,
		galleryService: h.galleryService,
		groupService:   h.groupService,
		hookExecutor:   h.hookExecutor,
	}}

	result, err := resolver.GenerateAPIKey(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, result)
}

// --- System endpoints ---

// GET /api/v1/system/status
func (h *RESTHandler) getSystemStatus(w http.ResponseWriter, r *http.Request) {
	mgr := manager.GetInstance()
	respondOK(w, mgr.GetSystemStatus())
}

// GET /api/v1/system/version
func (h *RESTHandler) getVersion(w http.ResponseWriter, r *http.Request) {
	version, hash, buildtime := build.Version()
	respondOK(w, &Version{
		Version:   &version,
		Hash:      hash,
		BuildTime: buildtime,
	})
}

// GET /api/v1/system/latest-version
func (h *RESTHandler) getLatestVersion(w http.ResponseWriter, r *http.Request) {
	latestRelease, err := GetLatestRelease(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	logger.Infof("Retrieved latest version: %s (%s)", latestRelease.Version, latestRelease.ShortHash)

	respondOK(w, &LatestVersion{
		Version:     latestRelease.Version,
		Shorthash:   latestRelease.ShortHash,
		ReleaseDate: latestRelease.Date,
		URL:         latestRelease.Url,
	})
}

// GET /api/v1/system/directory?path=...&locale=...
func (h *RESTHandler) getDirectory(w http.ResponseWriter, r *http.Request) {
	path := queryParam(r, "path")
	locale := queryParam(r, "locale")

	var pathPtr, localePtr *string
	if path != "" {
		pathPtr = &path
	}
	if locale != "" {
		localePtr = &locale
	}

	resolver := &queryResolver{&Resolver{repository: h.repository}}
	result, err := resolver.Directory(r.Context(), pathPtr, localePtr)
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, result)
}

// POST /api/v1/system/setup
func (h *RESTHandler) postSetup(w http.ResponseWriter, r *http.Request) {
	var input manager.SetupInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	err := manager.GetInstance().Setup(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, true)
}

// POST /api/v1/system/migrate
func (h *RESTHandler) postMigrate(w http.ResponseWriter, r *http.Request) {
	var input manager.MigrateInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	resolver := &mutationResolver{&Resolver{
		repository:   h.repository,
		hookExecutor: h.hookExecutor,
	}}

	result, err := resolver.Migrate(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, result)
}

// POST /api/v1/system/download-ffmpeg
func (h *RESTHandler) postDownloadFFmpeg(w http.ResponseWriter, r *http.Request) {
	resolver := &mutationResolver{&Resolver{
		repository:   h.repository,
		hookExecutor: h.hookExecutor,
	}}

	result, err := resolver.DownloadFFMpeg(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}

	respondOK(w, result)
}

// POST /api/v1/system/validate-stashbox
func (h *RESTHandler) postValidateStashBox(w http.ResponseWriter, r *http.Request) {
	var input config.StashBoxInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	box := models.StashBox{Endpoint: input.Endpoint, APIKey: input.APIKey}
	client := newStashBoxClient(box)

	user, err := client.GetUser(r.Context())

	valid := user != nil && user.Me != nil
	var status string
	if valid {
		status = "Successfully authenticated"
	} else if err != nil {
		status = err.Error()
	} else {
		status = "Unknown error"
	}

	respondOK(w, &StashBoxValidationResult{
		Valid:  valid,
		Status: status,
	})
}

// --- Stats endpoints ---

// GET /api/v1/stats
func (h *RESTHandler) getStats(w http.ResponseWriter, r *http.Request) {
	resolver := &queryResolver{&Resolver{repository: h.repository}}
	result, err := resolver.Stats(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	respondOK(w, result)
}

// GET /api/v1/stats/o-count
func (h *RESTHandler) getOCountStats(w http.ResponseWriter, r *http.Request) {
	resolver := &queryResolver{&Resolver{repository: h.repository}}
	result, err := resolver.OCountStats(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	respondOK(w, result)
}

// --- Logs endpoint ---

// GET /api/v1/logs
func (h *RESTHandler) getLogs(w http.ResponseWriter, r *http.Request) {
	resolver := &queryResolver{&Resolver{repository: h.repository}}
	result, err := resolver.Logs(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	respondOK(w, result)
}

// --- Notes endpoints ---

// GET /api/v1/notes
func (h *RESTHandler) getNotes(w http.ResponseWriter, r *http.Request) {
	resolver := &mutationResolver{&Resolver{repository: h.repository}}
	result, err := resolver.ReadNotesFile(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	respondOK(w, result)
}

// PUT /api/v1/notes
func (h *RESTHandler) putNotes(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if err := decodeBody(r, &body); err != nil {
		respondBadRequest(w, err)
		return
	}

	resolver := &mutationResolver{&Resolver{repository: h.repository}}
	result, err := resolver.WriteNotesFile(r.Context(), body.Content)
	if err != nil {
		handleError(w, err)
		return
	}
	respondOK(w, result)
}

// Helper to build a resolver for reusing existing logic during migration
func (h *RESTHandler) makeResolver() *Resolver {
	return &Resolver{
		repository:     h.repository,
		sceneService:   h.sceneService,
		imageService:   h.imageService,
		galleryService: h.galleryService,
		groupService:   h.groupService,
		hookExecutor:   h.hookExecutor,
	}
}
