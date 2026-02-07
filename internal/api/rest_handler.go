package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/txn"
)

// RESTHandler is the base handler for all REST API endpoints.
// It holds the same dependencies as the GraphQL Resolver.
type RESTHandler struct {
	repository     models.Repository
	sceneService   manager.SceneService
	imageService   manager.ImageService
	galleryService manager.GalleryService
	groupService   manager.GroupService
	hookExecutor   hookExecutor
	sseBroker      *SSEBroker
}

// NewRESTHandler creates a new REST handler with the given dependencies.
func NewRESTHandler(
	repo models.Repository,
	sceneService manager.SceneService,
	imageService manager.ImageService,
	galleryService manager.GalleryService,
	groupService manager.GroupService,
	hookExec hookExecutor,
	sseBroker *SSEBroker,
) *RESTHandler {
	return &RESTHandler{
		repository:     repo,
		sceneService:   sceneService,
		imageService:   imageService,
		galleryService: galleryService,
		groupService:   groupService,
		hookExecutor:   hookExec,
		sseBroker:      sseBroker,
	}
}

// --- Response helpers ---

// respondJSON writes a JSON response with the given status code and data.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			logger.Errorf("REST: error encoding response: %v", err)
		}
	}
}

// respondData wraps data in a standard envelope and writes JSON response.
func respondData(w http.ResponseWriter, status int, data interface{}) {
	respondJSON(w, status, map[string]interface{}{
		"data": data,
	})
}

// respondList writes a standard list response with data and count.
func respondList(w http.ResponseWriter, status int, data interface{}, count int) {
	respondJSON(w, status, map[string]interface{}{
		"data":  data,
		"count": count,
	})
}

// respondNoContent writes a 204 No Content response.
func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// respondOK writes a 200 OK with data wrapped in envelope.
func respondOK(w http.ResponseWriter, data interface{}) {
	respondData(w, http.StatusOK, data)
}

// respondCreated writes a 201 Created with data wrapped in envelope.
func respondCreated(w http.ResponseWriter, data interface{}) {
	respondData(w, http.StatusCreated, data)
}

// --- Error helpers ---

type restError struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func respondError(w http.ResponseWriter, status int, message string, code string) {
	respondJSON(w, status, restError{
		Error: message,
		Code:  code,
	})
}

func respondBadRequest(w http.ResponseWriter, err error) {
	respondError(w, http.StatusBadRequest, err.Error(), "BAD_REQUEST")
}

func respondNotFound(w http.ResponseWriter) {
	respondError(w, http.StatusNotFound, "not found", "NOT_FOUND")
}

func respondInternalError(w http.ResponseWriter, err error) {
	logger.Errorf("REST: internal error: %v", err)
	respondError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
}

// handleError determines the appropriate HTTP error response for a given error.
func handleError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	// Default to internal server error
	respondInternalError(w, err)
}

// --- Request helpers ---

// decodeBody decodes the JSON request body into the given target.
func decodeBody(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(target)
}

// decodeBodyWithMap decodes the JSON request body into both a typed struct
// and a raw map. The map is used for changesetTranslator to determine
// which fields were actually provided in the request.
func decodeBodyWithMap(r *http.Request, target interface{}) (map[string]interface{}, error) {
	if r.Body == nil {
		return nil, fmt.Errorf("request body is empty")
	}

	// Read body bytes
	var buf []byte
	buf, err := readBody(r)
	if err != nil {
		return nil, err
	}

	// Decode into target struct
	if err := json.Unmarshal(buf, target); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Decode into raw map for field presence detection
	var rawMap map[string]interface{}
	if err := json.Unmarshal(buf, &rawMap); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return rawMap, nil
}

// readBody reads the full request body as bytes.
func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	var buf []byte
	buf = make([]byte, 0, 1024)
	for {
		tmp := make([]byte, 1024)
		n, err := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
	}
	return buf, nil
}

// urlParamInt extracts an integer URL parameter.
func urlParamInt(r *http.Request, name string) (int, error) {
	param := chi.URLParam(r, name)
	if param == "" {
		return 0, fmt.Errorf("missing URL parameter: %s", name)
	}
	return strconv.Atoi(param)
}

// urlParamString extracts a string URL parameter.
func urlParamString(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

// queryParam extracts a query string parameter.
func queryParam(r *http.Request, name string) string {
	return r.URL.Query().Get(name)
}

// queryParamInt extracts an integer query string parameter, returns 0 if not set.
func queryParamInt(r *http.Request, name string) (int, bool) {
	s := r.URL.Query().Get(name)
	if s == "" {
		return 0, false
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return v, true
}

// --- Transaction helpers ---

// withReadTxn wraps a read operation in a read transaction.
func (h *RESTHandler) withReadTxn(ctx context.Context, fn func(ctx context.Context) error) error {
	return h.repository.WithReadTxn(ctx, fn)
}

// withTxn wraps a write operation in a read-write transaction.
func (h *RESTHandler) withTxn(ctx context.Context, fn func(ctx context.Context) error) error {
	return h.repository.WithTxn(ctx, fn)
}

// withReadTxnRest wraps a read transaction with HTTP error handling.
func (h *RESTHandler) withReadTxnRest(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context) error) bool {
	if err := txn.WithReadTxn(r.Context(), h.repository.TxnManager, fn); err != nil {
		handleError(w, err)
		return false
	}
	return true
}

// withTxnRest wraps a write transaction with HTTP error handling.
func (h *RESTHandler) withTxnRest(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context) error) bool {
	if err := txn.WithTxn(r.Context(), h.repository.TxnManager, fn); err != nil {
		handleError(w, err)
		return false
	}
	return true
}

// --- changeset translator for REST ---

// restChangesetTranslator is a changesetTranslator that uses a raw JSON map
// instead of GraphQL context. This allows reusing the same partial update
// logic from the GraphQL resolvers.
func newRESTChangesetTranslator(inputMap map[string]interface{}) changesetTranslator {
	return changesetTranslator{
		inputMap: inputMap,
	}
}

// --- Hook executor helper ---

func (h *RESTHandler) executePostHooks(ctx context.Context, id int, hookType hook.TriggerEnum, input interface{}, fields []string) {
	h.hookExecutor.ExecutePostHooks(ctx, id, hookType, input, fields)
}
