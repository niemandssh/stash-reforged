package api

// This file contains stub handlers for REST endpoints that are not yet implemented.
// They will be replaced with real implementations as each migration stage is completed.
// Each stub returns 501 Not Implemented.

import "net/http"

func notImplementedHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not yet implemented", "NOT_IMPLEMENTED")
}

// --- Этап 3: Scenes (remaining stubs) ---
func (h *RESTHandler) batchUpdateScenes(w http.ResponseWriter, r *http.Request)         { notImplementedHandler(w, r) }
// getSceneStreams moved to rest_scenes.go
func (h *RESTHandler) findSceneByHash(w http.ResponseWriter, r *http.Request)           { notImplementedHandler(w, r) }
func (h *RESTHandler) findDuplicateScenes(w http.ResponseWriter, r *http.Request)       { notImplementedHandler(w, r) }
func (h *RESTHandler) parseSceneFilenames(w http.ResponseWriter, r *http.Request)       { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneWall(w http.ResponseWriter, r *http.Request)                  { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneAddO(w http.ResponseWriter, r *http.Request)                  { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneDeleteO(w http.ResponseWriter, r *http.Request)               { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneResetO(w http.ResponseWriter, r *http.Request)                { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneAddOmg(w http.ResponseWriter, r *http.Request)                { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneDeleteOmg(w http.ResponseWriter, r *http.Request)             { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneResetOmg(w http.ResponseWriter, r *http.Request)              { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneAddPlay(w http.ResponseWriter, r *http.Request)               { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneDeletePlay(w http.ResponseWriter, r *http.Request)            { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneResetPlayCount(w http.ResponseWriter, r *http.Request)        { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneSaveActivity(w http.ResponseWriter, r *http.Request)          { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneResetActivity(w http.ResponseWriter, r *http.Request)         { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneConvertToMp4(w http.ResponseWriter, r *http.Request)          { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneConvertHLSToMP4(w http.ResponseWriter, r *http.Request)       { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneReduceResolution(w http.ResponseWriter, r *http.Request)      { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneTrimVideo(w http.ResponseWriter, r *http.Request)             { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneRegenerateSprites(w http.ResponseWriter, r *http.Request)     { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneSetBroken(w http.ResponseWriter, r *http.Request)             { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneGenerateScreenshot(w http.ResponseWriter, r *http.Request)    { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneSaveFilteredScreenshot(w http.ResponseWriter, r *http.Request) { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneMerge(w http.ResponseWriter, r *http.Request)                 { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneSetPrimaryFile(w http.ResponseWriter, r *http.Request)        { notImplementedHandler(w, r) }
func (h *RESTHandler) sceneAssignFile(w http.ResponseWriter, r *http.Request)            { notImplementedHandler(w, r) }
func (h *RESTHandler) recalculateSceneSimilarities(w http.ResponseWriter, r *http.Request) { notImplementedHandler(w, r) }

// --- Этап 3: Performers (remaining stubs) ---
func (h *RESTHandler) createPerformerProfileImage(w http.ResponseWriter, r *http.Request) { notImplementedHandler(w, r) }
func (h *RESTHandler) updatePerformerProfileImage(w http.ResponseWriter, r *http.Request) { notImplementedHandler(w, r) }
func (h *RESTHandler) destroyPerformerProfileImage(w http.ResponseWriter, r *http.Request) { notImplementedHandler(w, r) }

// --- Этап 3: Galleries (remaining stubs) ---
func (h *RESTHandler) galleryAddO(w http.ResponseWriter, r *http.Request)                { notImplementedHandler(w, r) }
func (h *RESTHandler) galleryDeleteO(w http.ResponseWriter, r *http.Request)             { notImplementedHandler(w, r) }
func (h *RESTHandler) galleryResetO(w http.ResponseWriter, r *http.Request)              { notImplementedHandler(w, r) }
func (h *RESTHandler) galleryAddOmg(w http.ResponseWriter, r *http.Request)              { notImplementedHandler(w, r) }
func (h *RESTHandler) galleryDeleteOmg(w http.ResponseWriter, r *http.Request)           { notImplementedHandler(w, r) }
func (h *RESTHandler) galleryResetOmg(w http.ResponseWriter, r *http.Request)            { notImplementedHandler(w, r) }
func (h *RESTHandler) galleryAddPlay(w http.ResponseWriter, r *http.Request)             { notImplementedHandler(w, r) }
func (h *RESTHandler) galleryDeletePlay(w http.ResponseWriter, r *http.Request)          { notImplementedHandler(w, r) }
func (h *RESTHandler) galleryResetPlayCount(w http.ResponseWriter, r *http.Request)      { notImplementedHandler(w, r) }
func (h *RESTHandler) setGalleryCover(w http.ResponseWriter, r *http.Request)            { notImplementedHandler(w, r) }
func (h *RESTHandler) resetGalleryCover(w http.ResponseWriter, r *http.Request)          { notImplementedHandler(w, r) }
func (h *RESTHandler) gallerySetPrimaryFile(w http.ResponseWriter, r *http.Request)      { notImplementedHandler(w, r) }
func (h *RESTHandler) createGalleryChapter(w http.ResponseWriter, r *http.Request)       { notImplementedHandler(w, r) }
func (h *RESTHandler) updateGalleryChapter(w http.ResponseWriter, r *http.Request)       { notImplementedHandler(w, r) }
func (h *RESTHandler) destroyGalleryChapter(w http.ResponseWriter, r *http.Request)      { notImplementedHandler(w, r) }

// --- Этап 3: Images (remaining stubs) ---
func (h *RESTHandler) imageAddO(w http.ResponseWriter, r *http.Request)                  { notImplementedHandler(w, r) }
func (h *RESTHandler) imageDeleteO(w http.ResponseWriter, r *http.Request)               { notImplementedHandler(w, r) }
func (h *RESTHandler) imageResetO(w http.ResponseWriter, r *http.Request)                { notImplementedHandler(w, r) }
func (h *RESTHandler) imageAddOmg(w http.ResponseWriter, r *http.Request)                { notImplementedHandler(w, r) }
func (h *RESTHandler) imageDeleteOmg(w http.ResponseWriter, r *http.Request)             { notImplementedHandler(w, r) }
func (h *RESTHandler) imageResetOmg(w http.ResponseWriter, r *http.Request)              { notImplementedHandler(w, r) }
func (h *RESTHandler) imageSetPrimaryFile(w http.ResponseWriter, r *http.Request)        { notImplementedHandler(w, r) }

// --- Этап 4: Games (remaining stubs) ---
func (h *RESTHandler) gameAddO(w http.ResponseWriter, r *http.Request)                   { notImplementedHandler(w, r) }
func (h *RESTHandler) gameDeleteO(w http.ResponseWriter, r *http.Request)                { notImplementedHandler(w, r) }
func (h *RESTHandler) gameResetO(w http.ResponseWriter, r *http.Request)                 { notImplementedHandler(w, r) }
func (h *RESTHandler) gameAddOmg(w http.ResponseWriter, r *http.Request)                 { notImplementedHandler(w, r) }
func (h *RESTHandler) gameDeleteOmg(w http.ResponseWriter, r *http.Request)              { notImplementedHandler(w, r) }
func (h *RESTHandler) gameResetOmg(w http.ResponseWriter, r *http.Request)               { notImplementedHandler(w, r) }
func (h *RESTHandler) gameAddView(w http.ResponseWriter, r *http.Request)                { notImplementedHandler(w, r) }
func (h *RESTHandler) gameDeleteView(w http.ResponseWriter, r *http.Request)             { notImplementedHandler(w, r) }
func (h *RESTHandler) gameResetViews(w http.ResponseWriter, r *http.Request)             { notImplementedHandler(w, r) }

// --- Этап 4: Scene Markers (remaining stubs) ---
func (h *RESTHandler) markerWall(w http.ResponseWriter, r *http.Request)                 { notImplementedHandler(w, r) }
func (h *RESTHandler) markerStrings(w http.ResponseWriter, r *http.Request)              { notImplementedHandler(w, r) }
// sceneMarkerTags moved to rest_scenes.go

// (Этап 4: Files & Folders handlers moved to rest_files.go)
// (Этап 5 handlers moved to rest_operations.go)

// (Этап 6 handlers moved to rest_special.go)
