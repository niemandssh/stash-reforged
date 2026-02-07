package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/image"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/utils"
)

// GET /api/v1/images/{id}
func (h *RESTHandler) findImage(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var ret *models.Image
	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		var err error
		ret, err = h.repository.Image.Find(ctx, id)
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

// POST /api/v1/images/query
func (h *RESTHandler) findImages(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Filter      *models.FindFilterType  `json:"filter,omitempty"`
		ImageFilter *models.ImageFilterType `json:"image_filter,omitempty"`
		IDs         []string                `json:"ids,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	var images []*models.Image
	var count int

	if !h.withReadTxnRest(w, r, func(ctx context.Context) error {
		if len(input.IDs) > 0 {
			ids, err := stringslice.StringSliceToIntSlice(input.IDs)
			if err != nil {
				return fmt.Errorf("converting ids: %w", err)
			}
			images, err = h.repository.Image.FindMany(ctx, ids)
			if err != nil {
				return err
			}
			count = len(images)
		} else {
			qr, err := h.repository.Image.Query(ctx, models.ImageQueryOptions{
				QueryOptions: models.QueryOptions{
					FindFilter: input.Filter,
					Count:      true,
				},
				ImageFilter: input.ImageFilter,
			})
			if err != nil {
				return err
			}
			images, err = qr.Resolve(ctx)
			if err != nil {
				return err
			}
			count = qr.Count
		}
		return nil
	}) {
		return
	}

	respondList(w, http.StatusOK, images, count)
}

// PUT /api/v1/images/{id}
func (h *RESTHandler) updateImage(w http.ResponseWriter, r *http.Request) {
	imageID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var input models.ImageUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	input.ID = strconv.Itoa(imageID)
	translator := newRESTChangesetTranslator(inputMap)

	updatedImage := models.NewImagePartial()

	updatedImage.Title = translator.optionalString(input.Title, "title")
	updatedImage.Code = translator.optionalString(input.Code, "code")
	updatedImage.Details = translator.optionalString(input.Details, "details")
	updatedImage.Photographer = translator.optionalString(input.Photographer, "photographer")
	updatedImage.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedImage.Organized = translator.optionalBool(input.Organized, "organized")

	updatedImage.Date, err = translator.optionalDate(input.Date, "date")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting date: %w", err))
		return
	}
	updatedImage.StudioID, err = translator.optionalIntFromString(input.StudioID, "studio_id")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting studio id: %w", err))
		return
	}

	updatedImage.URLs = translator.optionalURLs(input.Urls, input.URL)

	updatedImage.PrimaryFileID, err = translator.fileIDPtrFromString(input.PrimaryFileID)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting primary file id: %w", err))
		return
	}

	updatedImage.PerformerIDs, err = translator.updateIds(input.PerformerIds, "performer_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting performer ids: %w", err))
		return
	}
	updatedImage.TagIDs, err = translator.updateIds(input.TagIds, "tag_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}
	updatedImage.GalleryIDs, err = translator.updateIds(input.GalleryIds, "gallery_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting gallery ids: %w", err))
		return
	}

	var ret *models.Image
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Image

		// Validate primary file
		if updatedImage.PrimaryFileID != nil {
			i, err := qb.Find(ctx, imageID)
			if err != nil {
				return err
			}
			if i == nil {
				return fmt.Errorf("image with id %d not found", imageID)
			}

			if err := i.LoadFiles(ctx, qb); err != nil {
				return err
			}

			primaryFileID := *updatedImage.PrimaryFileID
			var found bool
			for _, ff := range i.Files.List() {
				if ff.Base().ID == primaryFileID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("file with id %d not associated with image", primaryFileID)
			}
		}

		// Validate gallery changes
		var updatedGalleryIDs []int
		if updatedImage.GalleryIDs != nil {
			i, err := qb.Find(ctx, imageID)
			if err != nil {
				return err
			}
			if i == nil {
				return fmt.Errorf("image with id %d not found", imageID)
			}

			if err := i.LoadGalleryIDs(ctx, qb); err != nil {
				return err
			}

			if err := h.galleryService.ValidateImageGalleryChange(ctx, i, *updatedImage.GalleryIDs); err != nil {
				return err
			}

			updatedGalleryIDs = updatedImage.GalleryIDs.ImpactedIDs(i.GalleryIDs.List())
		}

		ret, err = qb.UpdatePartial(ctx, imageID, updatedImage)
		if err != nil {
			return err
		}

		// Update impacted galleries
		for _, galleryID := range updatedGalleryIDs {
			if err := h.galleryService.Updated(ctx, galleryID); err != nil {
				return fmt.Errorf("updating gallery %d: %w", galleryID, err)
			}
		}

		return nil
	}) {
		return
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), ret.ID, hook.ImageUpdatePost, input, translator.getFields())

	respondOK(w, ret)
}

// PUT /api/v1/images/bulk
func (h *RESTHandler) bulkUpdateImages(w http.ResponseWriter, r *http.Request) {
	var input BulkImageUpdateInput
	inputMap, err := decodeBodyWithMap(r, &input)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	imageIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting ids: %w", err))
		return
	}

	translator := newRESTChangesetTranslator(inputMap)

	updatedImage := models.NewImagePartial()
	updatedImage.Title = translator.optionalString(input.Title, "title")
	updatedImage.Code = translator.optionalString(input.Code, "code")
	updatedImage.Details = translator.optionalString(input.Details, "details")
	updatedImage.Photographer = translator.optionalString(input.Photographer, "photographer")
	updatedImage.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedImage.Organized = translator.optionalBool(input.Organized, "organized")

	updatedImage.Date, err = translator.optionalDate(input.Date, "date")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting date: %w", err))
		return
	}
	updatedImage.StudioID, err = translator.optionalIntFromString(input.StudioID, "studio_id")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting studio id: %w", err))
		return
	}

	updatedImage.URLs = translator.optionalURLsBulk(input.Urls, input.URL)

	updatedImage.GalleryIDs, err = translator.updateIdsBulk(input.GalleryIds, "gallery_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting gallery ids: %w", err))
		return
	}
	updatedImage.PerformerIDs, err = translator.updateIdsBulk(input.PerformerIds, "performer_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting performer ids: %w", err))
		return
	}
	updatedImage.TagIDs, err = translator.updateIdsBulk(input.TagIds, "tag_ids")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting tag ids: %w", err))
		return
	}

	var ret []*models.Image
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		var updatedGalleryIDs []int
		qb := h.repository.Image

		for _, imageID := range imageIDs {
			i, err := qb.Find(ctx, imageID)
			if err != nil {
				return err
			}
			if i == nil {
				return fmt.Errorf("image with id %d not found", imageID)
			}

			if updatedImage.GalleryIDs != nil {
				if err := i.LoadGalleryIDs(ctx, qb); err != nil {
					return err
				}
				if err := h.galleryService.ValidateImageGalleryChange(ctx, i, *updatedImage.GalleryIDs); err != nil {
					return err
				}
				thisUpdatedGalleryIDs := updatedImage.GalleryIDs.ImpactedIDs(i.GalleryIDs.List())
				updatedGalleryIDs = sliceutil.AppendUniques(updatedGalleryIDs, thisUpdatedGalleryIDs)
			}

			img, err := qb.UpdatePartial(ctx, imageID, updatedImage)
			if err != nil {
				return err
			}
			ret = append(ret, img)
		}

		for _, galleryID := range updatedGalleryIDs {
			if err := h.galleryService.Updated(ctx, galleryID); err != nil {
				return fmt.Errorf("updating gallery %d: %w", galleryID, err)
			}
		}

		return nil
	}) {
		return
	}

	for _, img := range ret {
		h.hookExecutor.ExecutePostHooks(r.Context(), img.ID, hook.ImageUpdatePost, input, translator.getFields())
	}

	respondOK(w, ret)
}

// DELETE /api/v1/images/{id}
func (h *RESTHandler) destroyImage(w http.ResponseWriter, r *http.Request) {
	imageID, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	var body struct {
		DeleteGenerated *bool `json:"delete_generated,omitempty"`
		DeleteFile      *bool `json:"delete_file,omitempty"`
	}
	_ = decodeBody(r, &body)

	fileDeleter := &image.FileDeleter{
		Deleter: file.NewDeleter(),
		Paths:   manager.GetInstance().Paths,
	}

	var i *models.Image
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		i, err = h.repository.Image.Find(ctx, imageID)
		if err != nil {
			return err
		}
		if i == nil {
			return fmt.Errorf("image with id %d not found", imageID)
		}
		return h.imageService.Destroy(ctx, i, fileDeleter, utils.IsTrue(body.DeleteGenerated), utils.IsTrue(body.DeleteFile))
	}) {
		fileDeleter.Rollback()
		return
	}

	fileDeleter.Commit()

	input := models.ImageDestroyInput{
		ID:              strconv.Itoa(imageID),
		DeleteFile:      body.DeleteFile,
		DeleteGenerated: body.DeleteGenerated,
	}

	h.hookExecutor.ExecutePostHooks(r.Context(), i.ID, hook.ImageDestroyPost, plugin.ImageDestroyInput{
		ImageDestroyInput: input,
		Checksum:          i.Checksum,
		Path:              i.Path,
	}, nil)

	respondNoContent(w)
}

// DELETE /api/v1/images (body: {"ids": [...], ...})
func (h *RESTHandler) destroyImages(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs             []string `json:"ids"`
		DeleteGenerated *bool    `json:"delete_generated,omitempty"`
		DeleteFile      *bool    `json:"delete_file,omitempty"`
	}
	if err := decodeBody(r, &body); err != nil {
		respondBadRequest(w, err)
		return
	}

	imageIDs, err := stringslice.StringSliceToIntSlice(body.IDs)
	if err != nil {
		respondBadRequest(w, fmt.Errorf("converting ids: %w", err))
		return
	}

	fileDeleter := &image.FileDeleter{
		Deleter: file.NewDeleter(),
		Paths:   manager.GetInstance().Paths,
	}

	var images []*models.Image
	if !h.withTxnRest(w, r, func(ctx context.Context) error {
		qb := h.repository.Image
		for _, imageID := range imageIDs {
			i, err := qb.Find(ctx, imageID)
			if err != nil {
				return err
			}
			if i == nil {
				return fmt.Errorf("image with id %d not found", imageID)
			}

			images = append(images, i)

			if err := h.imageService.Destroy(ctx, i, fileDeleter, utils.IsTrue(body.DeleteGenerated), utils.IsTrue(body.DeleteFile)); err != nil {
				return err
			}
		}
		return nil
	}) {
		fileDeleter.Rollback()
		return
	}

	fileDeleter.Commit()

	destroyInput := models.ImagesDestroyInput{
		Ids:             body.IDs,
		DeleteFile:      body.DeleteFile,
		DeleteGenerated: body.DeleteGenerated,
	}

	for _, img := range images {
		h.hookExecutor.ExecutePostHooks(r.Context(), img.ID, hook.ImageDestroyPost, plugin.ImagesDestroyInput{
			ImagesDestroyInput: destroyInput,
			Checksum:           img.Checksum,
			Path:               img.Path,
		}, nil)
	}

	respondNoContent(w)
}
