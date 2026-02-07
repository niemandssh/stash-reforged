package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

// --- Files ---

// GET /api/v1/files/{id}
func (h *RESTHandler) findFile(w http.ResponseWriter, r *http.Request) {
	idStr := urlParamString(r, "id")

	idInt, err := strconv.Atoi(idStr)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("invalid file id: %w", err))
		return
	}

	var ret models.File
	if err := h.withReadTxn(r.Context(), func(ctx context.Context) error {
		qb := h.repository.File

		files, err := qb.Find(ctx, models.FileID(idInt))
		if err != nil {
			return err
		}
		if len(files) > 0 {
			ret = files[0]
		}
		return nil
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	if ret == nil {
		respondNotFound(w)
		return
	}

	respondOK(w, convertBaseFile(ret))
}

// POST /api/v1/files/query
func (h *RESTHandler) findFiles(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FileFilter *models.FileFilterType `json:"file_filter,omitempty"`
		Filter     *models.FindFilterType `json:"filter,omitempty"`
		IDs        []string               `json:"ids,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *FindFilesResultType
	if err := h.withReadTxn(r.Context(), func(ctx context.Context) error {
		var files []models.File
		var err error

		result := &models.FileQueryResult{}

		if len(input.IDs) > 0 {
			fileIDsInt, err := stringslice.StringSliceToIntSlice(input.IDs)
			if err != nil {
				return err
			}
			fileIDs := models.FileIDsFromInts(fileIDsInt)

			files, err = h.repository.File.Find(ctx, fileIDs...)
			if err != nil {
				return err
			}

			result.Count = len(files)
			for _, f := range files {
				if asVideo, ok := f.(*models.VideoFile); ok {
					result.TotalDuration += asVideo.Duration
				}
				if asImage, ok := f.(*models.ImageFile); ok {
					result.Megapixels += asImage.Megapixels()
				}
				result.TotalSize += f.Base().Size
			}
		} else {
			result, err = h.repository.File.Query(ctx, models.FileQueryOptions{
				QueryOptions: models.QueryOptions{
					FindFilter: input.Filter,
					Count:      true,
				},
				FileFilter:    input.FileFilter,
				TotalDuration: true,
				Megapixels:    true,
				TotalSize:     true,
			})
			if err != nil {
				return err
			}
			files, err = result.Resolve(ctx)
			if err != nil {
				return err
			}
		}

		ret = &FindFilesResultType{
			Count:      result.Count,
			Files:      convertBaseFiles(files),
			Duration:   result.TotalDuration,
			Megapixels: result.Megapixels,
			Size:       int(result.TotalSize),
		}
		return nil
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, ret)
}

// POST /api/v1/files/move
func (h *RESTHandler) moveFiles(w http.ResponseWriter, r *http.Request) {
	var input MoveFilesInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	if err := h.withTxn(r.Context(), func(ctx context.Context) error {
		fileStore := h.repository.File
		folderStore := h.repository.Folder
		mover := file.NewMover(fileStore, folderStore)
		mover.RegisterHooks(ctx)

		var (
			folder   *models.Folder
			basename string
		)

		fileIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
		if err != nil {
			return fmt.Errorf("converting ids: %w", err)
		}

		switch {
		case input.DestinationFolderID != nil:
			folderID, err := strconv.Atoi(*input.DestinationFolderID)
			if err != nil {
				return fmt.Errorf("converting destination folder id: %w", err)
			}

			folder, err = folderStore.Find(ctx, models.FolderID(folderID))
			if err != nil {
				return fmt.Errorf("finding destination folder: %w", err)
			}

			if folder == nil {
				return fmt.Errorf("folder with id %d not found", folderID)
			}

			if folder.ZipFileID != nil {
				return fmt.Errorf("cannot move to %s, is in a zip file", folder.Path)
			}
		case input.DestinationFolder != nil:
			folderPath := *input.DestinationFolder

			// ensure folder path is within the library
			paths := manager.GetInstance().Config.GetStashPaths()
			if l := paths.GetStashFromDirPath(folderPath); l == nil {
				return fmt.Errorf("folder path %s must be within a stash library path", folderPath)
			}

			// get or create folder hierarchy
			folder, err = file.GetOrCreateFolderHierarchy(ctx, folderStore, folderPath)
			if err != nil {
				return fmt.Errorf("getting or creating folder hierarchy: %w", err)
			}
		default:
			return fmt.Errorf("must specify destination folder or path")
		}

		if input.DestinationBasename != nil {
			if len(input.Ids) != 1 {
				return fmt.Errorf("must specify one file when providing destination basename")
			}
			basename = *input.DestinationBasename
		}

		// create folder hierarchy in the filesystem
		if err := mover.CreateFolderHierarchy(folder.Path); err != nil {
			return fmt.Errorf("creating folder hierarchy %s in filesystem: %w", folder.Path, err)
		}

		for _, fileIDInt := range fileIDs {
			fileID := models.FileID(fileIDInt)
			f, err := fileStore.Find(ctx, fileID)
			if err != nil {
				return fmt.Errorf("finding file %d: %w", fileID, err)
			}

			if basename != "" {
				if err := validateFileExtension(f[0].Base().Basename, basename); err != nil {
					return err
				}
			}

			if err := mover.Move(ctx, f[0], folder, basename); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, true)
}

func validateFileExtension(oldBasename, newBasename string) error {
	c := manager.GetInstance().Config

	extsLists := [][]string{
		c.GetVideoExtensions(),
		c.GetImageExtensions(),
		c.GetGalleryExtensions(),
	}

	for _, exts := range extsLists {
		if fsutil.MatchExtension(oldBasename, exts) && !fsutil.MatchExtension(newBasename, exts) {
			return fmt.Errorf("file extension for %s is inconsistent with old filename %s", newBasename, oldBasename)
		}
	}

	return nil
}

// DELETE /api/v1/files
func (h *RESTHandler) deleteFiles(w http.ResponseWriter, r *http.Request) {
	var input struct {
		IDs []string `json:"ids"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	fileIDs, err := stringslice.StringSliceToIntSlice(input.IDs)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	fileDeleter := file.NewDeleter()
	destroyer := &file.ZipDestroyer{
		FileDestroyer:   h.repository.File,
		FolderDestroyer: h.repository.Folder,
	}

	if err := h.withTxn(r.Context(), func(ctx context.Context) error {
		qb := h.repository.File

		for _, fileIDInt := range fileIDs {
			fileID := models.FileID(fileIDInt)
			f, err := qb.Find(ctx, fileID)
			if err != nil {
				return err
			}

			path := f[0].Base().Path

			// ensure not a primary file
			isPrimary, err := qb.IsPrimary(ctx, fileID)
			if err != nil {
				return fmt.Errorf("checking if file %s is primary: %w", path, err)
			}

			if isPrimary {
				return fmt.Errorf("cannot delete primary file %s", path)
			}

			// destroy files in zip file
			inZip, err := qb.FindByZipFileID(ctx, fileID)
			if err != nil {
				return fmt.Errorf("finding zip file contents for %s: %w", path, err)
			}

			for _, ff := range inZip {
				const deleteFileInZip = false
				if err := file.Destroy(ctx, qb, ff, fileDeleter, deleteFileInZip); err != nil {
					return fmt.Errorf("destroying file %s: %w", ff.Base().Path, err)
				}
			}

			const deleteFile = true
			if err := destroyer.DestroyZip(ctx, f[0], fileDeleter, deleteFile); err != nil {
				return fmt.Errorf("deleting file %s: %w", path, err)
			}
		}

		return nil
	}); err != nil {
		fileDeleter.Rollback()
		respondInternalError(w, err)
		return
	}

	fileDeleter.Commit()
	respondOK(w, true)
}

// POST /api/v1/files/fingerprints
func (h *RESTHandler) setFileFingerprints(w http.ResponseWriter, r *http.Request) {
	var input FileSetFingerprintsInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	fileIDInt, err := strconv.Atoi(input.ID)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting id: %w", err))
		return
	}

	fileID := models.FileID(fileIDInt)

	var (
		fingerprints []models.Fingerprint
		toDelete     []string
	)

	for _, i := range input.Fingerprints {
		if i.Type == models.FingerprintTypeMD5 || i.Type == models.FingerprintTypeOshash {
			respondBadRequest(w, fmt.Errorf("cannot modify %s fingerprint", i.Type))
			return
		}

		if i.Value == nil {
			toDelete = append(toDelete, i.Type)
		} else {
			var v interface{}
			v = *i.Value

			if i.Type == models.FingerprintTypePhash {
				vInt, err := strconv.ParseUint(*i.Value, 16, 64)
				if err != nil {
					respondBadRequest(w, fmt.Errorf("converting phash %s: %w", *i.Value, err))
					return
				}
				v = vInt
			}

			fingerprints = append(fingerprints, models.Fingerprint{
				Type:        i.Type,
				Fingerprint: v,
			})
		}
	}

	if err := h.withTxn(r.Context(), func(ctx context.Context) error {
		qb := h.repository.File

		if len(fingerprints) > 0 {
			if err := qb.ModifyFingerprints(ctx, fileID, fingerprints); err != nil {
				return fmt.Errorf("modifying fingerprints: %w", err)
			}
		}

		if len(toDelete) > 0 {
			if err := qb.DestroyFingerprints(ctx, fileID, toDelete); err != nil {
				return fmt.Errorf("destroying fingerprints: %w", err)
			}
		}

		return nil
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, true)
}

// POST /api/v1/files/scan-threats
func (h *RESTHandler) scanVideoFileThreats(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FileID string `json:"file_id"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	jobID, err := manager.GetInstance().ScanVideoFileThreats(r.Context(), input.FileID)
	if err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/files/scan-all-threats
func (h *RESTHandler) scanAllScenesForThreats(w http.ResponseWriter, r *http.Request) {
	jobID, err := manager.GetInstance().ScanAllScenesForThreats(r.Context())
	if err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// --- Folders ---

// GET /api/v1/folders/{id}
func (h *RESTHandler) findFolder(w http.ResponseWriter, r *http.Request) {
	idStr := urlParamString(r, "id")

	var ret *models.Folder
	idInt, err := strconv.Atoi(idStr)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("invalid folder id: %w", err))
		return
	}

	if err := h.withReadTxn(r.Context(), func(ctx context.Context) error {
		var err error
		ret, err = h.repository.Folder.Find(ctx, models.FolderID(idInt))
		return err
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	if ret == nil {
		respondNotFound(w)
		return
	}

	respondOK(w, ret)
}

// POST /api/v1/folders/query
func (h *RESTHandler) findFolders(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FolderFilter *models.FolderFilterType `json:"folder_filter,omitempty"`
		Filter       *models.FindFilterType   `json:"filter,omitempty"`
		IDs          []string                 `json:"ids,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *FindFoldersResultType
	if err := h.withReadTxn(r.Context(), func(ctx context.Context) error {
		var folders []*models.Folder
		var err error

		result := &models.FolderQueryResult{}

		if len(input.IDs) > 0 {
			folderIDsInt, err := stringslice.StringSliceToIntSlice(input.IDs)
			if err != nil {
				return err
			}
			folderIDs := models.FolderIDsFromInts(folderIDsInt)

			folders, err = h.repository.Folder.FindMany(ctx, folderIDs)
			if err != nil {
				return err
			}
			result.Count = len(folders)
		} else {
			result, err = h.repository.Folder.Query(ctx, models.FolderQueryOptions{
				QueryOptions: models.QueryOptions{
					FindFilter: input.Filter,
					Count:      true,
				},
				FolderFilter: input.FolderFilter,
			})
			if err != nil {
				return err
			}
			folders, err = result.Resolve(ctx)
			if err != nil {
				return err
			}
		}

		ret = &FindFoldersResultType{
			Count:   result.Count,
			Folders: folders,
		}
		return nil
	}); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, ret)
}
