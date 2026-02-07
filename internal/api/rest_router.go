package api

import (
	"github.com/go-chi/chi/v5"
)

// RESTRoutes returns the chi.Router with all REST API v1 routes mounted.
func (h *RESTHandler) RESTRoutes() chi.Router {
	r := chi.NewRouter()

	// SSE endpoint for real-time events (replaces GraphQL subscriptions)
	r.Get("/events", h.SSEHandler)

	// System & Configuration (Этап 1)
	r.Route("/config", h.configRoutes)
	r.Route("/system", h.systemRoutes)
	r.Route("/stats", h.statsRoutes)
	r.Get("/logs", h.getLogs)
	r.Route("/notes", h.notesRoutes)

	// Simple CRUD entities (Этап 2)
	r.Route("/tags", h.tagRoutes)
	r.Route("/studios", h.studioRoutes)
	r.Route("/color-presets", h.colorPresetRoutes)
	r.Route("/filters", h.savedFilterRoutes)

	// Main entities (Этап 3)
	r.Route("/scenes", h.sceneRoutes)
	r.Route("/performers", h.performerRoutes)
	r.Route("/galleries", h.galleryRoutes)
	r.Route("/images", h.imageRoutes)

	// Auxiliary entities (Этап 4)
	r.Route("/groups", h.groupRoutes)
	r.Route("/games", h.gameRoutes)
	r.Route("/scene-markers", h.sceneMarkerRoutes)
	r.Route("/files", h.fileRoutes)
	r.Route("/folders", h.folderRoutes)

	// Operations (Этап 5)
	r.Route("/metadata", h.metadataRoutes)
	r.Route("/scrapers", h.scraperRoutes)
	r.Route("/plugins", h.pluginRESTRoutes)
	r.Route("/packages", h.packageRoutes)
	r.Route("/jobs", h.jobRoutes)

	// Special operations (Этап 6)
	r.Route("/dlna", h.dlnaRoutes)
	r.Route("/sql", h.sqlRoutes)
	r.Route("/stash-box", h.stashBoxRoutes)
	r.Route("/view-history", h.viewHistoryRoutes)
	r.Route("/database", h.databaseRoutes)
	r.Route("/misc", h.miscRoutes)

	return r
}

// --- Route groups (stubs for stages 1-6, to be implemented incrementally) ---

func (h *RESTHandler) configRoutes(r chi.Router) {
	r.Get("/", h.getConfiguration)
	r.Put("/general", h.configureGeneral)
	r.Put("/interface", h.configureInterface)
	r.Put("/defaults", h.configureDefaults)
	r.Put("/ui", h.configureUI)
	r.Put("/ui/setting", h.configureUISetting)
	r.Put("/scraping", h.configureScraping)
	r.Put("/dlna", h.configureDLNA)
	r.Put("/plugin/{pluginId}", h.configurePlugin)
	r.Post("/api-key", h.generateAPIKey)
}

func (h *RESTHandler) systemRoutes(r chi.Router) {
	r.Get("/status", h.getSystemStatus)
	r.Get("/version", h.getVersion)
	r.Get("/latest-version", h.getLatestVersion)
	r.Get("/directory", h.getDirectory)
	r.Post("/setup", h.postSetup)
	r.Post("/migrate", h.postMigrate)
	r.Post("/download-ffmpeg", h.postDownloadFFmpeg)
	r.Post("/validate-stashbox", h.postValidateStashBox)
}

func (h *RESTHandler) statsRoutes(r chi.Router) {
	r.Get("/", h.getStats)
	r.Get("/o-count", h.getOCountStats)
}

func (h *RESTHandler) notesRoutes(r chi.Router) {
	r.Get("/", h.getNotes)
	r.Put("/", h.putNotes)
}

func (h *RESTHandler) tagRoutes(r chi.Router) {
	r.Get("/{id}", h.findTag)
	r.Post("/query", h.findTags)
	r.Get("/colors", h.findTagColors)
	r.Post("/", h.createTag)
	r.Put("/{id}", h.updateTag)
	r.Put("/bulk", h.bulkUpdateTags)
	r.Delete("/{id}", h.destroyTag)
	r.Delete("/", h.destroyTags)
	r.Post("/merge", h.mergeTags)
}

func (h *RESTHandler) studioRoutes(r chi.Router) {
	r.Get("/{id}", h.findStudio)
	r.Post("/query", h.findStudios)
	r.Post("/", h.createStudio)
	r.Put("/{id}", h.updateStudio)
	r.Delete("/{id}", h.destroyStudio)
	r.Delete("/", h.destroyStudios)
}

func (h *RESTHandler) colorPresetRoutes(r chi.Router) {
	r.Get("/", h.findColorPresets)
	r.Get("/{id}", h.findColorPreset)
	r.Post("/query", h.findColorPresets)
	r.Post("/", h.createColorPreset)
	r.Put("/{id}", h.updateColorPreset)
	r.Delete("/{id}", h.destroyColorPreset)
}

func (h *RESTHandler) savedFilterRoutes(r chi.Router) {
	r.Get("/{id}", h.findSavedFilter)
	r.Get("/", h.findSavedFilters)
	r.Get("/default", h.findDefaultFilter)
	r.Post("/", h.saveFilter)
	r.Post("/default", h.setDefaultFilter)
	r.Delete("/{id}", h.destroySavedFilter)
}

func (h *RESTHandler) sceneRoutes(r chi.Router) {
	r.Get("/{id}", h.findScene)
	r.Post("/query", h.findScenes)
	r.Post("/", h.createScene)
	r.Put("/{id}", h.updateScene)
	r.Put("/bulk", h.bulkUpdateScenes)
	r.Put("/batch", h.batchUpdateScenes)
	r.Delete("/{id}", h.destroyScene)
	r.Delete("/", h.destroyScenes)

	// Special scene queries
	r.Get("/{id}/streams", h.getSceneStreams)
	r.Post("/by-hash", h.findSceneByHash)
	r.Post("/duplicates", h.findDuplicateScenes)
	r.Post("/parse-filenames", h.parseSceneFilenames)
	r.Get("/wall", h.sceneWall)

	// O-Count
	r.Post("/{id}/o", h.sceneAddO)
	r.Delete("/{id}/o", h.sceneDeleteO)
	r.Post("/{id}/o/reset", h.sceneResetO)

	// OMG-Count
	r.Post("/{id}/omg", h.sceneAddOmg)
	r.Delete("/{id}/omg", h.sceneDeleteOmg)
	r.Post("/{id}/omg/reset", h.sceneResetOmg)

	// Play count
	r.Post("/{id}/play", h.sceneAddPlay)
	r.Delete("/{id}/play", h.sceneDeletePlay)
	r.Post("/{id}/play/reset", h.sceneResetPlayCount)

	// Activity
	r.Put("/{id}/activity", h.sceneSaveActivity)
	r.Post("/{id}/activity/reset", h.sceneResetActivity)

	// Video operations
	r.Post("/{id}/convert/mp4", h.sceneConvertToMp4)
	r.Post("/{id}/convert/hls", h.sceneConvertHLSToMP4)
	r.Post("/{id}/reduce-resolution", h.sceneReduceResolution)
	r.Post("/{id}/trim", h.sceneTrimVideo)
	r.Post("/{id}/regenerate-sprites", h.sceneRegenerateSprites)
	r.Put("/{id}/broken", h.sceneSetBroken)
	r.Post("/{id}/screenshot", h.sceneGenerateScreenshot)
	r.Post("/{id}/filtered-screenshot", h.sceneSaveFilteredScreenshot)

	// Merge & files
	r.Post("/{id}/merge", h.sceneMerge)
	r.Put("/{id}/primary-file", h.sceneSetPrimaryFile)
	r.Put("/{id}/assign-file", h.sceneAssignFile)

	// Similarity
	r.Get("/{id}/similar", h.findSimilarScenes)
	r.Post("/{id}/recalculate-similarity", h.recalculateSceneSimilarities)
}

func (h *RESTHandler) performerRoutes(r chi.Router) {
	r.Get("/{id}", h.findPerformer)
	r.Post("/query", h.findPerformers)
	r.Post("/", h.createPerformer)
	r.Put("/{id}", h.updatePerformer)
	r.Put("/bulk", h.bulkUpdatePerformers)
	r.Delete("/{id}", h.destroyPerformer)
	r.Delete("/", h.destroyPerformers)

	// Profile images
	r.Post("/{id}/profile-images", h.createPerformerProfileImage)
	r.Put("/{id}/profile-images/{imageId}", h.updatePerformerProfileImage)
	r.Delete("/{id}/profile-images/{imageId}", h.destroyPerformerProfileImage)
}

func (h *RESTHandler) galleryRoutes(r chi.Router) {
	r.Get("/{id}", h.findGallery)
	r.Post("/query", h.findGalleries)
	r.Post("/", h.createGallery)
	r.Put("/{id}", h.updateGallery)
	r.Put("/bulk", h.bulkUpdateGalleries)
	r.Delete("/{id}", h.destroyGallery)

	// O/OMG/Play
	r.Post("/{id}/o", h.galleryAddO)
	r.Delete("/{id}/o", h.galleryDeleteO)
	r.Post("/{id}/o/reset", h.galleryResetO)
	r.Post("/{id}/omg", h.galleryAddOmg)
	r.Delete("/{id}/omg", h.galleryDeleteOmg)
	r.Post("/{id}/omg/reset", h.galleryResetOmg)
	r.Post("/{id}/play", h.galleryAddPlay)
	r.Delete("/{id}/play", h.galleryDeletePlay)
	r.Post("/{id}/play/reset", h.galleryResetPlayCount)

	// Images
	r.Post("/{id}/images", h.addGalleryImages)
	r.Delete("/{id}/images", h.removeGalleryImages)
	r.Put("/{id}/cover", h.setGalleryCover)
	r.Delete("/{id}/cover", h.resetGalleryCover)
	r.Put("/{id}/primary-file", h.gallerySetPrimaryFile)

	// Chapters
	r.Post("/{id}/chapters", h.createGalleryChapter)
	r.Put("/{id}/chapters/{chapterId}", h.updateGalleryChapter)
	r.Delete("/{id}/chapters/{chapterId}", h.destroyGalleryChapter)
}

func (h *RESTHandler) imageRoutes(r chi.Router) {
	r.Get("/{id}", h.findImage)
	r.Post("/query", h.findImages)
	r.Put("/{id}", h.updateImage)
	r.Put("/bulk", h.bulkUpdateImages)
	r.Delete("/{id}", h.destroyImage)
	r.Delete("/", h.destroyImages)

	// O/OMG
	r.Post("/{id}/o", h.imageAddO)
	r.Delete("/{id}/o", h.imageDeleteO)
	r.Post("/{id}/o/reset", h.imageResetO)
	r.Post("/{id}/omg", h.imageAddOmg)
	r.Delete("/{id}/omg", h.imageDeleteOmg)
	r.Post("/{id}/omg/reset", h.imageResetOmg)
	r.Put("/{id}/primary-file", h.imageSetPrimaryFile)
}

func (h *RESTHandler) groupRoutes(r chi.Router) {
	r.Get("/{id}", h.findGroup)
	r.Post("/query", h.findGroups)
	r.Post("/", h.createGroup)
	r.Put("/{id}", h.updateGroup)
	r.Put("/bulk", h.bulkUpdateGroups)
	r.Delete("/{id}", h.destroyGroup)
	r.Delete("/", h.destroyGroups)

	// Sub-groups
	r.Post("/{id}/sub-groups", h.addGroupSubGroups)
	r.Delete("/{id}/sub-groups", h.removeGroupSubGroups)
	r.Put("/{id}/sub-groups/reorder", h.reorderSubGroups)
}

func (h *RESTHandler) gameRoutes(r chi.Router) {
	r.Get("/{id}", h.findGame)
	r.Post("/query", h.findGames)
	r.Post("/", h.createGame)
	r.Put("/{id}", h.updateGame)
	r.Delete("/{id}", h.destroyGame)

	// O/OMG/Views
	r.Post("/{id}/o", h.gameAddO)
	r.Delete("/{id}/o", h.gameDeleteO)
	r.Post("/{id}/o/reset", h.gameResetO)
	r.Post("/{id}/omg", h.gameAddOmg)
	r.Delete("/{id}/omg", h.gameDeleteOmg)
	r.Post("/{id}/omg/reset", h.gameResetOmg)
	r.Post("/{id}/view", h.gameAddView)
	r.Delete("/{id}/view", h.gameDeleteView)
	r.Post("/{id}/view/reset", h.gameResetViews)
}

func (h *RESTHandler) sceneMarkerRoutes(r chi.Router) {
	r.Post("/query", h.findSceneMarkers)
	r.Get("/wall", h.markerWall)
	r.Get("/strings", h.markerStrings)
	r.Get("/tags/{sceneId}", h.sceneMarkerTags)
	r.Post("/", h.createSceneMarker)
	r.Put("/{id}", h.updateSceneMarker)
	r.Delete("/{id}", h.destroySceneMarker)
	r.Delete("/", h.destroySceneMarkers)
}

func (h *RESTHandler) fileRoutes(r chi.Router) {
	r.Get("/{id}", h.findFile)
	r.Post("/query", h.findFiles)
	r.Post("/move", h.moveFiles)
	r.Delete("/", h.deleteFiles)
	r.Put("/fingerprints", h.setFileFingerprints)
	r.Post("/{id}/scan-threats", h.scanVideoFileThreats)
	r.Post("/scan-all-threats", h.scanAllScenesForThreats)
}

func (h *RESTHandler) folderRoutes(r chi.Router) {
	r.Get("/{id}", h.findFolder)
	r.Post("/query", h.findFolders)
}

func (h *RESTHandler) metadataRoutes(r chi.Router) {
	r.Post("/scan", h.metadataScan)
	r.Post("/generate", h.metadataGenerate)
	r.Post("/auto-tag", h.metadataAutoTag)
	r.Post("/clean", h.metadataClean)
	r.Post("/clean-generated", h.metadataCleanGenerated)
	r.Post("/identify", h.metadataIdentify)
	r.Post("/export", h.metadataExport)
	r.Post("/import", h.metadataImport)
	r.Post("/export-objects", h.exportObjects)
	r.Post("/import-objects", h.importObjects)
}

func (h *RESTHandler) scraperRoutes(r chi.Router) {
	r.Get("/", h.listScrapers)
	r.Post("/reload", h.reloadScrapers)
	r.Post("/scene", h.scrapeSingleScene)
	r.Post("/scenes", h.scrapeMultiScenes)
	r.Post("/performer", h.scrapeSinglePerformer)
	r.Post("/performers", h.scrapeMultiPerformers)
	r.Post("/gallery", h.scrapeSingleGallery)
	r.Post("/group", h.scrapeSingleGroup)
	r.Post("/image", h.scrapeSingleImage)
	r.Post("/url", h.scrapeURL)
}

func (h *RESTHandler) pluginRESTRoutes(r chi.Router) {
	r.Get("/", h.getPlugins)
	r.Get("/tasks", h.getPluginTasks)
	r.Post("/reload", h.reloadPlugins)
	r.Put("/enabled", h.setPluginsEnabled)
	r.Post("/{pluginId}/run", h.runPluginTask)
	r.Post("/{pluginId}/operation", h.runPluginOperation)
}

func (h *RESTHandler) packageRoutes(r chi.Router) {
	r.Get("/installed", h.getInstalledPackages)
	r.Get("/available", h.getAvailablePackages)
	r.Post("/install", h.installPackages)
	r.Post("/update", h.updatePackages)
	r.Post("/uninstall", h.uninstallPackages)
}

func (h *RESTHandler) jobRoutes(r chi.Router) {
	r.Get("/", h.getJobQueue)
	r.Get("/{id}", h.findJob)
	r.Post("/{id}/stop", h.stopJob)
	r.Post("/stop-all", h.stopAllJobs)
}

func (h *RESTHandler) dlnaRoutes(r chi.Router) {
	r.Get("/status", h.getDLNAStatus)
	r.Post("/enable", h.enableDLNA)
	r.Post("/disable", h.disableDLNA)
	r.Post("/temp-ip", h.addTempDLNAIP)
	r.Delete("/temp-ip", h.removeTempDLNAIP)
}

func (h *RESTHandler) sqlRoutes(r chi.Router) {
	r.Post("/query", h.querySQL)
	r.Post("/exec", h.execSQL)
}

func (h *RESTHandler) stashBoxRoutes(r chi.Router) {
	r.Post("/fingerprints", h.submitStashBoxFingerprints)
	r.Post("/scene-draft", h.submitStashBoxSceneDraft)
	r.Post("/performer-draft", h.submitStashBoxPerformerDraft)
	r.Post("/batch/performers", h.stashBoxBatchPerformerTag)
	r.Post("/batch/studios", h.stashBoxBatchStudioTag)
}

func (h *RESTHandler) viewHistoryRoutes(r chi.Router) {
	r.Post("/query", h.findViewHistory)
}

func (h *RESTHandler) databaseRoutes(r chi.Router) {
	r.Post("/backup", h.backupDatabase)
	r.Post("/anonymise", h.anonymiseDatabase)
	r.Post("/optimise", h.optimiseDatabase)
	r.Post("/migrate-hash-naming", h.migrateHashNaming)
	r.Post("/migrate-screenshots", h.migrateSceneScreenshots)
	r.Post("/migrate-blobs", h.migrateBlobs)
}

func (h *RESTHandler) miscRoutes(r chi.Router) {
	r.Post("/open-external-player/{id}", h.openInExternalPlayer)
}
