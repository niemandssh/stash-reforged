package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/image"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/utils"
)

// GET /api/v1/galleries/{id}
func (h *RESTHandler) findGallery(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *models.Gallery
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		ret, err = h.repository.Gallery.Find(ctx, id)
		return err
	}) {
		return
	}

	if ret == nil {
		respondNotFound(w)
		return
	}

	respondOK(w, ret)
}

// POST /api/v1/galleries/query
func (h *RESTHandler) findGalleries(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Filter        *models.FindFilterType    `json:"filter,omitempty"`
		GalleryFilter *models.GalleryFilterType `json:"gallery_filter,omitempty"`
		IDs           []string                  `json:"ids,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var galleries []*models.Gallery
	var count int

	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		if len(input.IDs) > 0 {
			ids, err := stringslice.StringSliceToIntSlice(input.IDs)
			if err != nil {
				return fmt.Errorf("converting ids: %w", err)
			}
			galleries, err = h.repository.Gallery.FindMany(ctx, ids)
			if err != nil {
				return err
			}
			count = len(galleries)
		} else {
			var err error
			galleries, count, err = h.repository.Gallery.Query(ctx, input.GalleryFilter, input.Filter)
			if err != nil {
				return err
			}
		}
		return nil
	}) {
		return
	}

	respondList(w, http.StatusOK, galleries, count)
}

// POST /api/v1/galleries
func (h *RESTHandler) createGallery(w http.ResponseWriter, r *http.Request) {
	var input GalleryCreateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	if input.Title == "" {
		respondBadRequest(w, errors.New("title must not be empty"))
		return
	}

	translator := newRESTChangesetTranslator(inputMap)

	newGallery := models.NewGallery()
	newGallery.Title = input.Title
	newGallery.Code = translator.string(input.Code)
	newGallery.Details = translator.string(input.Details)
	newGallery.Photographer = translator.string(input.Photographer)
	newGallery.Rating = input.Rating100

	newGallery.Date, err = translator.datePtr(input.Date)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting date: %w", err))
		return
	}
	newGallery.StudioID, err = translator.intPtrFromString(input.StudioID)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting studio id: %w", err))
		return
	}

	newGallery.PerformerIDs, err = translator.relatedIds(input.PerformerIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting performer ids: %w", err))
		return
	}
	newGallery.TagIDs, err = translator.relatedIds(input.TagIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}
	newGallery.SceneIDs, err = translator.relatedIds(input.SceneIds)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting scene ids: %w", err))
		return
	}

	if input.Urls != nil {
		newGallery.URLs = models.NewRelatedStrings(input.Urls)
	} else if input.URL != nil {
		newGallery.URLs = models.NewRelatedStrings([]string{*input.URL})
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		return h.repository.Gallery.Create(ctx, &newGallery, nil)
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), newGallery.ID, hook.GalleryCreatePost, input, nil)

	var result *models.Gallery
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		result, err = h.repository.Gallery.Find(ctx, newGallery.ID)
		return err
	}) {
		return
	}

	respondCreated(w, result)
}

// PUT /api/v1/galleries/{id}
func (h *RESTHandler) updateGallery(w http.ResponseWriter, r *http.Request) {
	galleryID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var input models.GalleryUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	input.ID = strconv.Itoa(galleryID)
	translator := newRESTChangesetTranslator(inputMap)

	updatedGallery := models.NewGalleryPartial()

	if input.Title != nil {
		updatedGallery.Title = models.NewOptionalString(*input.Title)
	}

	updatedGallery.Code = translator.optionalString(input.Code, "code")
	updatedGallery.Details = translator.optionalString(input.Details, "details")
	updatedGallery.Photographer = translator.optionalString(input.Photographer, "photographer")
	updatedGallery.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedGallery.Organized = translator.optionalBool(input.Organized, "organized")
	updatedGallery.Pinned = translator.optionalBool(input.Pinned, "pinned")
	updatedGallery.DisplayMode = translator.optionalInt(input.DisplayMode, "display_mode")

	updatedGallery.Date, err = translator.optionalDate(input.Date, "date")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting date: %w", err))
		return
	}
	updatedGallery.StudioID, err = translator.optionalIntFromString(input.StudioID, "studio_id")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting studio id: %w", err))
		return
	}

	updatedGallery.URLs = translator.optionalURLs(input.Urls, input.URL)

	updatedGallery.PrimaryFileID, err = translator.fileIDPtrFromString(input.PrimaryFileID)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting primary file id: %w", err))
		return
	}

	updatedGallery.PerformerIDs, err = translator.updateIds(input.PerformerIds, "performer_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting performer ids: %w", err))
		return
	}
	updatedGallery.TagIDs, err = translator.updateIds(input.TagIds, "tag_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}
	updatedGallery.SceneIDs, err = translator.updateIds(input.SceneIds, "scene_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting scene ids: %w", err))
		return
	}

	var ret *models.Gallery
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Gallery

		// Validate title for user-created galleries
		if input.Title != nil && *input.Title == "" {
			originalGallery, err := qb.Find(ctx, galleryID)
			if err != nil {
				return err
			}
			if originalGallery != nil && originalGallery.IsUserCreated() {
				return errors.New("title must not be empty for user-created galleries")
			}
		}

		ret, err = qb.UpdatePartial(ctx, galleryID, updatedGallery)
		return err
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), ret.ID, hook.GalleryUpdatePost, input, translator.getFields())

	respondOK(w, ret)
}

// PUT /api/v1/galleries/bulk
func (h *RESTHandler) bulkUpdateGalleries(w http.ResponseWriter, r *http.Request) {
	var input BulkGalleryUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	galleryIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting ids: %w", err))
		return
	}

	translator := newRESTChangesetTranslator(inputMap)

	updatedGallery := models.NewGalleryPartial()
	updatedGallery.Code = translator.optionalString(input.Code, "code")
	updatedGallery.Details = translator.optionalString(input.Details, "details")
	updatedGallery.Photographer = translator.optionalString(input.Photographer, "photographer")
	updatedGallery.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedGallery.Organized = translator.optionalBool(input.Organized, "organized")
	updatedGallery.URLs = translator.optionalURLsBulk(input.Urls, input.URL)

	updatedGallery.Date, err = translator.optionalDate(input.Date, "date")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting date: %w", err))
		return
	}
	updatedGallery.StudioID, err = translator.optionalIntFromString(input.StudioID, "studio_id")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting studio id: %w", err))
		return
	}

	updatedGallery.PerformerIDs, err = translator.updateIdsBulk(input.PerformerIds, "performer_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting performer ids: %w", err))
		return
	}
	updatedGallery.TagIDs, err = translator.updateIdsBulk(input.TagIds, "tag_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}
	updatedGallery.SceneIDs, err = translator.updateIdsBulk(input.SceneIds, "scene_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting scene ids: %w", err))
		return
	}

	var ret []*models.Gallery
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Gallery
		for _, id := range galleryIDs {
			gallery, err := qb.UpdatePartial(ctx, id, updatedGallery)
			if err != nil {
				return err
			}
			ret = append(ret, gallery)
		}
		return nil
	}) {
		return
	}

	for _, gallery := range ret {
		h.hookExecutor.ExecutePostHooks(r.Context(), gallery.ID, hook.GalleryUpdatePost, input, translator.getFields())
	}

	respondOK(w, ret)
}

// DELETE /api/v1/galleries/{id}
func (h *RESTHandler) destroyGallery(w http.ResponseWriter, r *http.Request) {
	galleryID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var body struct {
		DeleteGenerated *bool `json:"delete_generated,omitempty"`
		DeleteFile      *bool `json:"delete_file,omitempty"`
	}
	_ = decodeBody(r, &body)

	input := models.GalleryDestroyInput{
		Ids:             []string{strconv.Itoa(galleryID)},
		DeleteFile:      body.DeleteFile,
		DeleteGenerated: body.DeleteGenerated,
	}

	fileDeleter := &image.FileDeleter{
		Deleter: file.NewDeleter(),
		Paths:   manager.GetInstance().Paths,
	}

	deleteGenerated := utils.IsTrue(body.DeleteGenerated)
	deleteFile := utils.IsTrue(body.DeleteFile)

	var gallery *models.Gallery
	var imgsDestroyed []*models.Image

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Gallery

		gallery, err = qb.Find(ctx, galleryID)
		if err != nil {
			return err
		}
		if gallery == nil {
			return fmt.Errorf("gallery with id %d not found", galleryID)
		}

		if err := gallery.LoadFiles(ctx, qb); err != nil {
			return fmt.Errorf("loading files for gallery %d", galleryID)
		}

		imgsDestroyed, err = h.galleryService.Destroy(ctx, gallery, fileDeleter, deleteGenerated, deleteFile)
		return err
	}) {
		fileDeleter.Rollback()
		return
	}

	fileDeleter.Commit()

	// Don't delete stash library paths
	if deleteFile && gallery.Path != "" && !isStashPath(gallery.Path) {
		_ = os.Remove(gallery.Path)
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), gallery.ID, hook.GalleryDestroyPost, plugin.GalleryDestroyInput{
		GalleryDestroyInput: input,
		Checksum:            gallery.PrimaryChecksum(),
		Path:                gallery.Path,
	}, nil)

	for _, img := range imgsDestroyed {
		h.hookExecutor.ExecutePostHooks(r.Context(), img.ID, hook.ImageDestroyPost, plugin.ImageDestroyInput{
			Checksum: img.Checksum,
			Path:     img.Path,
		}, nil)
	}

	respondNoContent(w)
}

// POST /api/v1/galleries/{id}/images
func (h *RESTHandler) addGalleryImages(w http.ResponseWriter, r *http.Request) {
	galleryID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var body struct {
		ImageIDs []string `json:"image_ids"`
	}
	if err := decodeBody(r, &body); err != nil {
		respondBadRequest(w, err)
		return
	}

	imageIDs, err := stringslice.StringSliceToIntSlice(body.ImageIDs)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting image ids: %w", err))
		return
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Gallery
		gallery, err := qb.Find(ctx, galleryID)
		if err != nil {
			return err
		}
		if gallery == nil {
			return fmt.Errorf("gallery with id %d not found", galleryID)
		}
		return h.galleryService.AddImages(ctx, gallery, imageIDs...)
	}) {
		return
	}

	respondOK(w, true)
}

// DELETE /api/v1/galleries/{id}/images
func (h *RESTHandler) removeGalleryImages(w http.ResponseWriter, r *http.Request) {
	galleryID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var body struct {
		ImageIDs []string `json:"image_ids"`
	}
	if err := decodeBody(r, &body); err != nil {
		respondBadRequest(w, err)
		return
	}

	imageIDs, err := stringslice.StringSliceToIntSlice(body.ImageIDs)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting image ids: %w", err))
		return
	}

	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Gallery
		gallery, err := qb.Find(ctx, galleryID)
		if err != nil {
			return err
		}
		if gallery == nil {
			return fmt.Errorf("gallery with id %d not found", galleryID)
		}
		return h.galleryService.RemoveImages(ctx, gallery, imageIDs...)
	}) {
		return
	}

	respondOK(w, true)
}
