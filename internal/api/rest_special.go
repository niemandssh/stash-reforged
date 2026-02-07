package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/task"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scene"
	"github.com/stashapp/stash/pkg/stashbox"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/utils"
)

// --- DLNA ---

// GET /api/v1/dlna/status
func (h *RESTHandler) getDLNAStatus(w http.ResponseWriter, r *http.Request) {
	status := manager.GetInstance().DLNAService.Status()
	respondOK(w, status)
}

// POST /api/v1/dlna/enable
func (h *RESTHandler) enableDLNA(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Duration *int `json:"duration,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	if err := manager.GetInstance().DLNAService.Start(parseMinutes(input.Duration)); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, true)
}

// POST /api/v1/dlna/disable
func (h *RESTHandler) disableDLNA(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Duration *int `json:"duration,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	manager.GetInstance().DLNAService.Stop(parseMinutes(input.Duration))
	respondOK(w, true)
}

// POST /api/v1/dlna/ip/add
func (h *RESTHandler) addTempDLNAIP(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Address  string `json:"address"`
		Duration *int   `json:"duration,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	manager.GetInstance().DLNAService.AddTempDLNAIP(input.Address, parseMinutes(input.Duration))
	respondOK(w, true)
}

// POST /api/v1/dlna/ip/remove
func (h *RESTHandler) removeTempDLNAIP(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Address string `json:"address"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	ret := manager.GetInstance().DLNAService.RemoveTempDLNAIP(input.Address)
	respondOK(w, ret)
}

// --- SQL ---

// POST /api/v1/sql/query
func (h *RESTHandler) querySQL(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SQL  string        `json:"sql"`
		Args []interface{} `json:"args,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var cols []string
	var rows [][]interface{}

	db := manager.GetInstance().Database
	if err := h.withTxn(r.Context(), func(ctx context.Context) error {
		var err error
		cols, rows, err = db.QuerySQL(ctx, input.SQL, input.Args)
		return err
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, &SQLQueryResult{
		Columns: cols,
		Rows:    rows,
	})
}

// POST /api/v1/sql/exec
func (h *RESTHandler) execSQL(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SQL  string        `json:"sql"`
		Args []interface{} `json:"args,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var rowsAffected *int64
	var lastInsertID *int64

	db := manager.GetInstance().Database
	if err := h.withTxn(r.Context(), func(ctx context.Context) error {
		var err error
		rowsAffected, lastInsertID, err = db.ExecSQL(ctx, input.SQL, input.Args)
		return err
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, &SQLExecResult{
		RowsAffected: rowsAffected,
		LastInsertID: lastInsertID,
	})
}

// --- StashBox ---

// POST /api/v1/stashbox/fingerprints
func (h *RESTHandler) submitStashBoxFingerprints(w http.ResponseWriter, r *http.Request) {
	var input StashBoxFingerprintSubmissionInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	b, err := resolveStashBox(input.StashBoxIndex, input.StashBoxEndpoint)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	ids, err := stringslice.StringSliceToIntSlice(input.SceneIds)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	client := newStashBoxClient(*b)

	var scenes []*models.Scene
	if err := h.withReadTxn(r.Context(), func(ctx context.Context) error {
		var err error
		scenes, err = h.sceneService.FindByIDs(ctx, ids, scene.LoadStashIDs, scene.LoadFiles)
		return err
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	ok, err := client.SubmitFingerprints(r.Context(), scenes)
	if err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, ok)
}

// POST /api/v1/stashbox/scene-draft
func (h *RESTHandler) submitStashBoxSceneDraft(w http.ResponseWriter, r *http.Request) {
	var input StashBoxDraftSubmissionInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	b, err := resolveStashBox(input.StashBoxIndex, input.StashBoxEndpoint)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	client := newStashBoxClient(*b)

	id, err := strconv.Atoi(input.ID)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting id: %w", err))
		return
	}

	var res *string
	if err := h.withReadTxn(r.Context(), func(ctx context.Context) error {
		qb := h.repository.Scene
		s, err := qb.Find(ctx, id)
		if err != nil {
			return err
		}

		if s == nil {
			return fmt.Errorf("scene with id %d not found", id)
		}

		cover, err := qb.GetCover(ctx, id)
		if err != nil {
			logger.Errorf("Error getting scene cover: %v", err)
		}

		draft, err := h.makeSceneDraft(ctx, s, cover)
		if err != nil {
			return err
		}

		res, err = client.SubmitSceneDraft(ctx, *draft)
		return err
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, res)
}

func (h *RESTHandler) makeSceneDraft(ctx context.Context, s *models.Scene, cover []byte) (*stashbox.SceneDraft, error) {
	if err := s.LoadURLs(ctx, h.repository.Scene); err != nil {
		return nil, fmt.Errorf("loading scene URLs: %w", err)
	}

	if err := s.LoadStashIDs(ctx, h.repository.Scene); err != nil {
		return nil, err
	}

	draft := &stashbox.SceneDraft{
		Scene: s,
	}

	pqb := h.repository.Performer
	sqb := h.repository.Studio

	if s.StudioID != nil {
		var err error
		draft.Studio, err = sqb.Find(ctx, *s.StudioID)
		if err != nil {
			return nil, err
		}
		if draft.Studio == nil {
			return nil, fmt.Errorf("studio with id %d not found", *s.StudioID)
		}

		if err := draft.Studio.LoadStashIDs(ctx, h.repository.Studio); err != nil {
			return nil, err
		}
	}

	// submit all file fingerprints
	if err := s.LoadFiles(ctx, h.repository.Scene); err != nil {
		return nil, err
	}

	scenePerformers, err := pqb.FindBySceneID(ctx, s.ID)
	if err != nil {
		return nil, err
	}

	for _, p := range scenePerformers {
		if err := p.LoadStashIDs(ctx, pqb); err != nil {
			return nil, err
		}
	}
	draft.Performers = scenePerformers

	draft.Tags, err = h.repository.Tag.FindBySceneID(ctx, s.ID)
	if err != nil {
		return nil, err
	}

	draft.Cover = cover

	return draft, nil
}

// POST /api/v1/stashbox/performer-draft
func (h *RESTHandler) submitStashBoxPerformerDraft(w http.ResponseWriter, r *http.Request) {
	var input StashBoxDraftSubmissionInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	b, err := resolveStashBox(input.StashBoxIndex, input.StashBoxEndpoint)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	client := newStashBoxClient(*b)

	id, err := strconv.Atoi(input.ID)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting id: %w", err))
		return
	}

	var res *string
	if err := h.withReadTxn(r.Context(), func(ctx context.Context) error {
		qb := h.repository.Performer
		performer, err := qb.Find(ctx, id)
		if err != nil {
			return err
		}

		if performer == nil {
			return fmt.Errorf("performer with id %d not found", id)
		}

		if err := performer.LoadAliases(ctx, qb); err != nil {
			return err
		}

		if err := performer.LoadURLs(ctx, qb); err != nil {
			return err
		}

		if err := performer.LoadStashIDs(ctx, qb); err != nil {
			return err
		}

		img, _ := qb.GetImage(ctx, performer.ID)

		res, err = client.SubmitPerformerDraft(ctx, performer, img)
		return err
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, res)
}

// POST /api/v1/stashbox/batch-performer-tag
func (h *RESTHandler) stashBoxBatchPerformerTag(w http.ResponseWriter, r *http.Request) {
	var input manager.StashBoxBatchTagInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	b, err := resolveStashBoxBatchTagInput(input.Endpoint, input.StashBoxEndpoint)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	jobID := manager.GetInstance().StashBoxBatchPerformerTag(r.Context(), b, input)
	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/stashbox/batch-studio-tag
func (h *RESTHandler) stashBoxBatchStudioTag(w http.ResponseWriter, r *http.Request) {
	var input manager.StashBoxBatchTagInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	b, err := resolveStashBoxBatchTagInput(input.Endpoint, input.StashBoxEndpoint)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	jobID := manager.GetInstance().StashBoxBatchStudioTag(r.Context(), b, input)
	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// --- View History ---

// POST /api/v1/view-history/query
func (h *RESTHandler) findViewHistory(w http.ResponseWriter, r *http.Request) {
	var input struct {
		HistoryFilter *ViewHistoryFilter      `json:"history_filter,omitempty"`
		Filter        *models.FindFilterType  `json:"filter,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *ViewHistoryResult
	if err := h.withReadTxn(r.Context(), func(ctx context.Context) error {
		page := 1
		perPage := 25
		if input.Filter != nil {
			if input.Filter.Page != nil {
				page = *input.Filter.Page
			}
			if input.Filter.PerPage != nil {
				perPage = *input.Filter.PerPage
			}
		}

		combinedViews, err := h.repository.Scene.GetCombinedAggregatedViewHistory(ctx, page, perPage)
		if err != nil {
			return err
		}

		var entries []*ViewHistoryEntry

		for _, cv := range combinedViews {
			if cv.ContentType == "scene" {
				s, err := h.repository.Scene.Find(ctx, cv.ContentID)
				if err != nil {
					return err
				}
				if s == nil {
					continue
				}

				entry := &ViewHistoryEntry{
					Scene:     s,
					ViewDate:  cv.ViewDate,
					ODate:     cv.ODate,
					OmgDate:   cv.OmgDate,
					ViewCount: &cv.ViewCount,
				}
				entries = append(entries, entry)
			} else if cv.ContentType == "gallery" {
				g, err := h.repository.Gallery.Find(ctx, cv.ContentID)
				if err != nil {
					return err
				}
				if g == nil {
					continue
				}

				entry := &ViewHistoryEntry{
					Gallery:   g,
					ViewDate:  cv.ViewDate,
					ODate:     cv.ODate,
					OmgDate:   cv.OmgDate,
					ViewCount: &cv.ViewCount,
				}
				entries = append(entries, entry)
			}
		}

		totalCount, err := h.repository.Scene.GetCombinedAggregatedViewHistoryCount(ctx)
		if err != nil {
			return err
		}

		scenesTotalOCount, err := h.repository.Scene.GetAllOCount(ctx)
		if err != nil {
			return err
		}
		galleriesTotalOCount, err := h.repository.Gallery.GetAllOCount(ctx)
		if err != nil {
			return err
		}
		totalOCount := scenesTotalOCount + galleriesTotalOCount

		scenesTotalOMGCount, err := h.repository.Scene.GetAllOMGCount(ctx)
		if err != nil {
			return err
		}
		galleriesTotalOMGCount, err := h.repository.Gallery.GetAllOMGCount(ctx)
		if err != nil {
			return err
		}
		totalOMGCount := scenesTotalOMGCount + galleriesTotalOMGCount

		ret = &ViewHistoryResult{
			Count:         totalCount,
			Items:         entries,
			TotalOCount:   totalOCount,
			TotalOMGCount: totalOMGCount,
		}

		return nil
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, ret)
}

// --- Database Operations ---

// POST /api/v1/database/backup
func (h *RESTHandler) backupDatabase(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Download *bool `json:"download,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	download := input.Download != nil && *input.Download
	mgr := manager.GetInstance()

	backupPath, backupName, err := mgr.BackupDatabase(download)
	if err != nil {
		respondInternalError(w, err)
		return
	}

	if download {
		downloadHash, err := mgr.DownloadStore.RegisterFile(backupPath, "", false)
		if err != nil {
			respondInternalError(w, fmt.Errorf("error registering file for download: %w", err))
			return
		}

		baseURL, _ := r.Context().Value(BaseURLCtxKey).(string)
		ret := baseURL + "/downloads/" + downloadHash + "/" + backupName
		respondOK(w, map[string]string{"download_url": ret})
	} else {
		logger.Infof("Successfully backed up database to: %s", backupPath)
		respondOK(w, map[string]string{"path": backupPath})
	}
}

// POST /api/v1/database/anonymise
func (h *RESTHandler) anonymiseDatabase(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Download *bool `json:"download,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	download := input.Download != nil && *input.Download
	mgr := manager.GetInstance()

	outPath, outName, err := mgr.AnonymiseDatabase(download)
	if err != nil {
		respondInternalError(w, err)
		return
	}

	if download {
		downloadHash, err := mgr.DownloadStore.RegisterFile(outPath, "", false)
		if err != nil {
			respondInternalError(w, fmt.Errorf("error registering file for download: %w", err))
			return
		}

		baseURL, _ := r.Context().Value(BaseURLCtxKey).(string)
		ret := baseURL + "/downloads/" + downloadHash + "/" + outName
		respondOK(w, map[string]string{"download_url": ret})
	} else {
		logger.Infof("Successfully anonymised database to: %s", outPath)
		respondOK(w, map[string]string{"path": outPath})
	}
}

// POST /api/v1/database/optimise
func (h *RESTHandler) optimiseDatabase(w http.ResponseWriter, r *http.Request) {
	jobID := manager.GetInstance().OptimiseDatabase(r.Context())
	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/database/migrate-hash-naming
func (h *RESTHandler) migrateHashNaming(w http.ResponseWriter, r *http.Request) {
	jobID := manager.GetInstance().MigrateHash(r.Context())
	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/database/migrate-scene-screenshots
func (h *RESTHandler) migrateSceneScreenshots(w http.ResponseWriter, r *http.Request) {
	var input MigrateSceneScreenshotsInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	mgr := manager.GetInstance()
	t := &task.MigrateSceneScreenshotsJob{
		ScreenshotsPath: mgr.Paths.Generated.Screenshots,
		Input: scene.MigrateSceneScreenshotsInput{
			DeleteFiles:       utils.IsTrue(input.DeleteFiles),
			OverwriteExisting: utils.IsTrue(input.OverwriteExisting),
		},
		SceneRepo:  mgr.Repository.Scene,
		TxnManager: mgr.Repository.TxnManager,
	}
	jobID := mgr.JobManager.Add(r.Context(), "Migrating scene screenshots to blobs...", t)

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/database/migrate-blobs
func (h *RESTHandler) migrateBlobs(w http.ResponseWriter, r *http.Request) {
	var input MigrateBlobsInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	mgr := manager.GetInstance()
	t := &task.MigrateBlobsJob{
		TxnManager: mgr.Database,
		BlobStore:  mgr.Database.Blobs,
		Vacuumer:   mgr.Database,
		DeleteOld:  utils.IsTrue(input.DeleteOld),
	}
	jobID := mgr.JobManager.Add(r.Context(), "Migrating blobs...", t)

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// --- Misc ---

// POST /api/v1/scenes/{id}/open-external
func (h *RESTHandler) openInExternalPlayer(w http.ResponseWriter, r *http.Request) {
	sceneID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("invalid scene id: %w", err))
		return
	}

	var s *models.Scene
	if err := h.withReadTxn(r.Context(), func(ctx context.Context) error {
		var err error
		s, err = h.repository.Scene.Find(ctx, sceneID)
		return err
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	if s == nil {
		respondNotFound(w)
		return
	}

	cfg := manager.GetInstance().Config
	playerCommand := cfg.GetExternalVideoPlayer()

	if playerCommand == "" {
		respondBadRequest(w, fmt.Errorf("external video player not configured"))
		return
	}

	var videoFilePath string
	if err := h.withReadTxn(r.Context(), func(ctx context.Context) error {
		if err := s.LoadFiles(ctx, h.repository.Scene); err != nil {
			return err
		}

		pf := s.Files.Primary()
		if pf != nil {
			videoFilePath = pf.Path
		}
		return nil
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	if videoFilePath == "" {
		respondBadRequest(w, fmt.Errorf("no video file found for scene"))
		return
	}

	if _, err := os.Stat(videoFilePath); os.IsNotExist(err) {
		respondBadRequest(w, fmt.Errorf("video file does not exist: %s", videoFilePath))
		return
	}

	escapedPath := strings.ReplaceAll(videoFilePath, "'", "'\"'\"'")
	quotedPath := "'" + escapedPath + "'"

	var command string
	if strings.Contains(playerCommand, "{path}") {
		command = strings.ReplaceAll(playerCommand, "{path}", quotedPath)
	} else {
		command = playerCommand + " " + quotedPath
	}

	cmd := exec.Command("sh", "-c", command)
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		respondInternalError(w, fmt.Errorf("failed to start external player: %w", err))
		return
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			logger.Warnf("External player exited with error: %v", err)
		}
	}()

	respondOK(w, true)
}

// parseMinutes is already defined in resolver_mutation_dlna.go
// and is available within the same package
