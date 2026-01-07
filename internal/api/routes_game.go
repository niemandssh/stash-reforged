package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/stashapp/stash/internal/static"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

type GameFinder interface {
	models.GameFinder
}

type gameRoutes struct {
	routes
	gameFinder GameFinder
}

func (rs gameRoutes) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/{gameId}", func(r chi.Router) {
		r.Use(rs.GameCtx)
		r.Get("/image", rs.Image)
	})

	return r
}

func (rs gameRoutes) Image(w http.ResponseWriter, r *http.Request) {
	game := r.Context().Value(gameKey).(*models.Game)
	defaultParam := r.URL.Query().Get("default")

	var image []byte
	if defaultParam != "true" {
		readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
			var err error
			image, err = rs.gameFinder.GetImage(ctx, game.ID)
			return err
		})
		if errors.Is(readTxnErr, context.Canceled) {
			return
		}
		if readTxnErr != nil {
			logger.Warnf("read transaction error on fetch game image: %v", readTxnErr)
		}
	}

	// fallback to default image
	if len(image) == 0 {
		image = static.ReadAll(static.DefaultStudioImage) // Using default studio image as placeholder
	}

	utils.ServeImage(w, r, image)
}

func (rs gameRoutes) GameCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gameID, err := strconv.Atoi(chi.URLParam(r, "gameId"))
		if err != nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}

		var game *models.Game
		_ = rs.withReadTxn(r, func(ctx context.Context) error {
			var err error
			game, err = rs.gameFinder.Find(ctx, gameID)
			return err
		})
		if game == nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}

		ctx := context.WithValue(r.Context(), gameKey, game)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
