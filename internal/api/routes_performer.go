package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

type PerformerFinder interface {
	models.PerformerGetter
	GetImage(ctx context.Context, performerID int) ([]byte, error)
	GetProfileImage(ctx context.Context, performerID int, imageID int) ([]byte, error)
	FindProfileImage(ctx context.Context, performerID int, imageID int) (*models.PerformerProfileImage, error)
}

type performerRoutes struct {
	routes
	performerFinder PerformerFinder
}

func (rs performerRoutes) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/{performerId}", func(r chi.Router) {
		r.Use(rs.PerformerCtx)
		r.Get("/image", rs.Image)
		r.Route("/profile_image/{imageId}", func(r chi.Router) {
			r.Use(rs.ProfileImageCtx)
			r.Get("/", rs.ProfileImage)
		})
	})

	return r
}

func (rs performerRoutes) Image(w http.ResponseWriter, r *http.Request) {
	performer := r.Context().Value(performerKey).(*models.Performer)
	defaultParam := r.URL.Query().Get("default")

	var image []byte
	if defaultParam != "true" {
		readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
			var err error
			image, err = rs.performerFinder.GetImage(ctx, performer.ID)
			return err
		})
		if errors.Is(readTxnErr, context.Canceled) {
			return
		}
		if readTxnErr != nil {
			logger.Warnf("read transaction error on fetch performer image: %v", readTxnErr)
		}
	}

	if len(image) == 0 {
		image = getDefaultPerformerImage(performer.Name, performer.Gender)
	}

	utils.ServeImage(w, r, image)
}

func (rs performerRoutes) ProfileImage(w http.ResponseWriter, r *http.Request) {
	// Debug: check what's in the context
	performerValue := r.Context().Value(performerKey)
	profileImageValue := r.Context().Value(profileImageKey)

	logger.Debugf("ProfileImage: performerValue type: %T, profileImageValue type: %T", performerValue, profileImageValue)

	performer, ok := performerValue.(*models.Performer)
	if !ok {
		logger.Errorf("ProfileImage: performerKey not found in context or wrong type")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	profileImage, ok := profileImageValue.(*models.PerformerProfileImage)
	if !ok {
		logger.Errorf("ProfileImage: profileImageKey not found in context or wrong type")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	var image []byte
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		image, err = rs.performerFinder.GetProfileImage(ctx, performer.ID, profileImage.ID)
		return err
	})
	if errors.Is(readTxnErr, context.Canceled) {
		return
	}
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch performer profile image: %v", readTxnErr)
	}

	if len(image) == 0 {
		image = getDefaultPerformerImage(performer.Name, performer.Gender)
	}

	utils.ServeImage(w, r, image)
}

func (rs performerRoutes) ProfileImageCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		imageID, err := strconv.Atoi(chi.URLParam(r, "imageId"))
		if err != nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}

		// Debug: check what's in the context before we modify it
		performerValue := r.Context().Value(performerKey)
		logger.Infof("ProfileImageCtx: performerValue type: %T, value: %v", performerValue, performerValue)

		// Debug: check all context keys
		requestCtx := r.Context()
		logger.Infof("ProfileImageCtx: context keys available:")
		// This is a bit hacky but helps debug
		if requestCtx.Value(performerKey) != nil {
			logger.Infof("  - performerKey: %T", requestCtx.Value(performerKey))
		}
		if requestCtx.Value(profileImageKey) != nil {
			logger.Infof("  - profileImageKey: %T", requestCtx.Value(profileImageKey))
		}

		performer, ok := performerValue.(*models.Performer)
		if !ok {
			logger.Errorf("ProfileImageCtx: performerKey not found in context or wrong type")
			http.Error(w, http.StatusText(500), 500)
			return
		}

		var profileImage *models.PerformerProfileImage
		_ = rs.withReadTxn(r, func(ctx context.Context) error {
			var err error
			profileImage, err = rs.performerFinder.FindProfileImage(ctx, performer.ID, imageID)
			logger.Infof("ProfileImageCtx: FindProfileImage result: %v, error: %v", profileImage, err)
			return err
		})
		if profileImage == nil {
			logger.Errorf("ProfileImageCtx: profileImage is nil for performerID %d, imageID %d", performer.ID, imageID)
			http.Error(w, http.StatusText(404), 404)
			return
		}

		// Preserve the existing context and add profileImageKey
		logger.Infof("ProfileImageCtx: setting profileImageKey to: %T %v", profileImage, profileImage)
		ctx := context.WithValue(r.Context(), profileImageKey, profileImage)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (rs performerRoutes) PerformerCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		performerID, err := strconv.Atoi(chi.URLParam(r, "performerId"))
		if err != nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}

		logger.Infof("PerformerCtx: processing performerID %d", performerID)

		var performer *models.Performer
		_ = rs.withReadTxn(r, func(ctx context.Context) error {
			var err error
			performer, err = rs.performerFinder.Find(ctx, performerID)
			return err
		})
		if performer == nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}

		ctx := context.WithValue(r.Context(), performerKey, performer)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
