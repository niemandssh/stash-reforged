package api

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/stashapp/stash/internal/identify"
	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/internal/manager/task"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/pkg"
	"github.com/stashapp/stash/pkg/scraper"
)

// --- Metadata operations ---

// POST /api/v1/metadata/scan
func (h *RESTHandler) metadataScan(w http.ResponseWriter, r *http.Request) {
	var input manager.ScanMetadataInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	jobID, err := manager.GetInstance().Scan(r.Context(), input)
	if err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/metadata/generate
func (h *RESTHandler) metadataGenerate(w http.ResponseWriter, r *http.Request) {
	var input manager.GenerateMetadataInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	jobID, err := manager.GetInstance().Generate(r.Context(), input)
	if err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/metadata/auto-tag
func (h *RESTHandler) metadataAutoTag(w http.ResponseWriter, r *http.Request) {
	var input manager.AutoTagMetadataInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	jobID := manager.GetInstance().AutoTag(r.Context(), input)

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/metadata/clean
func (h *RESTHandler) metadataClean(w http.ResponseWriter, r *http.Request) {
	var input manager.CleanMetadataInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	jobID := manager.GetInstance().Clean(r.Context(), input)

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/metadata/clean-generated
func (h *RESTHandler) metadataCleanGenerated(w http.ResponseWriter, r *http.Request) {
	var input task.CleanGeneratedOptions
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	mgr := manager.GetInstance()
	t := &task.CleanGeneratedJob{
		Options:                  input,
		Paths:                    mgr.Paths,
		BlobsStorageType:         mgr.Config.GetBlobsStorage(),
		VideoFileNamingAlgorithm: mgr.Config.GetVideoFileNamingAlgorithm(),
		Repository:               mgr.Repository,
		BlobCleaner:              mgr.Repository.Blob,
	}
	jobID := mgr.JobManager.Add(r.Context(), "Cleaning generated files...", t)

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/metadata/identify
func (h *RESTHandler) metadataIdentify(w http.ResponseWriter, r *http.Request) {
	var input identify.Options
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	t := manager.CreateIdentifyJob(input)
	jobID := manager.GetInstance().JobManager.Add(r.Context(), "Identifying...", t)

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/metadata/export
func (h *RESTHandler) metadataExport(w http.ResponseWriter, r *http.Request) {
	jobID, err := manager.GetInstance().Export(r.Context())
	if err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/metadata/import
func (h *RESTHandler) metadataImport(w http.ResponseWriter, r *http.Request) {
	jobID, err := manager.GetInstance().Import(r.Context())
	if err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/metadata/export-objects
func (h *RESTHandler) exportObjects(w http.ResponseWriter, r *http.Request) {
	var input manager.ExportObjectsInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	t := manager.CreateExportTask(config.GetInstance().GetVideoFileNamingAlgorithm(), input)

	var wg sync.WaitGroup
	wg.Add(1)
	t.Start(r.Context(), &wg)

	if t.DownloadHash != "" {
		baseURL, _ := r.Context().Value(BaseURLCtxKey).(string)
		suffix := time.Now().Format("20060102-150405")
		ret := baseURL + "/downloads/" + t.DownloadHash + "/export" + suffix + ".zip"
		respondOK(w, map[string]string{"download_url": ret})
	} else {
		respondOK(w, nil)
	}
}

// POST /api/v1/metadata/import-objects
func (h *RESTHandler) importObjects(w http.ResponseWriter, r *http.Request) {
	var input manager.ImportObjectsInput
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	t, err := manager.CreateImportTask(config.GetInstance().GetVideoFileNamingAlgorithm(), input)
	if err != nil {
		respondInternalError(w, err)
		return
	}

	jobID := manager.GetInstance().RunSingleTask(r.Context(), t)

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// --- Scrapers ---

// GET /api/v1/scrapers
func (h *RESTHandler) listScrapers(w http.ResponseWriter, r *http.Request) {
	// Get scraper types from query params
	var types []scraper.ScrapeContentType
	for _, t := range r.URL.Query()["type"] {
		types = append(types, scraper.ScrapeContentType(t))
	}

	scrapers := manager.GetInstance().ScraperCache.ListScrapers(types)

	respondOK(w, scrapers)
}

// POST /api/v1/scrapers/reload
func (h *RESTHandler) reloadScrapers(w http.ResponseWriter, r *http.Request) {
	manager.GetInstance().RefreshScraperCache()
	respondOK(w, true)
}

// POST /api/v1/scrapers/scrape-scene
func (h *RESTHandler) scrapeSingleScene(w http.ResponseWriter, r *http.Request) {
	notImplementedHandler(w, r)
}

// POST /api/v1/scrapers/scrape-scenes
func (h *RESTHandler) scrapeMultiScenes(w http.ResponseWriter, r *http.Request) {
	notImplementedHandler(w, r)
}

// POST /api/v1/scrapers/scrape-performer
func (h *RESTHandler) scrapeSinglePerformer(w http.ResponseWriter, r *http.Request) {
	notImplementedHandler(w, r)
}

// POST /api/v1/scrapers/scrape-performers
func (h *RESTHandler) scrapeMultiPerformers(w http.ResponseWriter, r *http.Request) {
	notImplementedHandler(w, r)
}

// POST /api/v1/scrapers/scrape-gallery
func (h *RESTHandler) scrapeSingleGallery(w http.ResponseWriter, r *http.Request) {
	notImplementedHandler(w, r)
}

// POST /api/v1/scrapers/scrape-group
func (h *RESTHandler) scrapeSingleGroup(w http.ResponseWriter, r *http.Request) {
	notImplementedHandler(w, r)
}

// POST /api/v1/scrapers/scrape-image
func (h *RESTHandler) scrapeSingleImage(w http.ResponseWriter, r *http.Request) {
	notImplementedHandler(w, r)
}

// POST /api/v1/scrapers/scrape-url
func (h *RESTHandler) scrapeURL(w http.ResponseWriter, r *http.Request) {
	notImplementedHandler(w, r)
}

// --- Plugins ---

// GET /api/v1/plugins
func (h *RESTHandler) getPlugins(w http.ResponseWriter, r *http.Request) {
	plugins := manager.GetInstance().PluginCache.ListPlugins()
	respondOK(w, plugins)
}

// GET /api/v1/plugins/tasks
func (h *RESTHandler) getPluginTasks(w http.ResponseWriter, r *http.Request) {
	tasks := manager.GetInstance().PluginCache.ListPluginTasks()
	respondOK(w, tasks)
}

// POST /api/v1/plugins/reload
func (h *RESTHandler) reloadPlugins(w http.ResponseWriter, r *http.Request) {
	manager.GetInstance().RefreshPluginCache()
	respondOK(w, true)
}

// POST /api/v1/plugins/enabled
func (h *RESTHandler) setPluginsEnabled(w http.ResponseWriter, r *http.Request) {
	var input map[string]bool
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	cfg := config.GetInstance()
	existingDisabled := cfg.GetDisabledPlugins()

	// Create a set of disabled plugins
	disabledMap := make(map[string]bool)
	for _, p := range existingDisabled {
		disabledMap[p] = true
	}

	for pluginID, enabled := range input {
		if enabled {
			delete(disabledMap, pluginID)
		} else {
			disabledMap[pluginID] = true
		}
	}

	var newDisabled []string
	for p := range disabledMap {
		newDisabled = append(newDisabled, p)
	}

	cfg.SetInterface(config.DisabledPlugins, newDisabled)
	if err := cfg.Write(); err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, true)
}

// POST /api/v1/plugins/run-task
func (h *RESTHandler) runPluginTask(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PluginID    string                 `json:"plugin_id"`
		TaskName    *string                `json:"task_name,omitempty"`
		Description *string                `json:"description,omitempty"`
		Args        map[string]interface{} `json:"args,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	jobID := manager.GetInstance().RunPluginTask(r.Context(), input.PluginID, input.TaskName, input.Description, input.Args)
	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/plugins/run-operation
func (h *RESTHandler) runPluginOperation(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PluginID string                 `json:"plugin_id"`
		Args     map[string]interface{} `json:"args,omitempty"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	result, err := manager.GetInstance().PluginCache.RunPlugin(r.Context(), input.PluginID, input.Args)
	if err != nil {
		respondInternalError(w, err)
		return
	}

	respondOK(w, result)
}

// --- Packages ---

// GET /api/v1/packages/installed?type=...&upgrades=true
func (h *RESTHandler) getInstalledPackages(w http.ResponseWriter, r *http.Request) {
	typeStr := queryParam(r, "type")
	withUpgrades := queryParam(r, "upgrades") == "true"

	pm, err := getPackageManager(PackageType(typeStr))
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	ctx := r.Context()

	if withUpgrades {
		installed, err := pm.ListInstalled(ctx)
		if err != nil {
			respondInternalError(w, err)
			return
		}

		allRemoteList, err := pm.ListInstalledRemotes(ctx, installed)
		if err != nil {
			respondInternalError(w, err)
			return
		}

		packageStatusIndex := pkg.MakePackageStatusIndex(installed, allRemoteList)

		ret := make([]*Package, len(packageStatusIndex))
		i := 0
		for _, k := range sortedPackageSpecKeys(packageStatusIndex) {
			v := packageStatusIndex[k]
			p := manifestToPackage(*v.Local)
			if v.Remote != nil {
				pp := remotePackageToPackage(*v.Remote, allRemoteList)
				p.SourcePackage = pp
			}
			ret[i] = p
			i++
		}

		respondOK(w, ret)
		return
	}

	installed, err := pm.ListInstalled(ctx)
	if err != nil {
		respondInternalError(w, err)
		return
	}

	ret := make([]*Package, len(installed))
	i := 0
	for _, k := range sortedPackageSpecKeys(installed) {
		ret[i] = manifestToPackage(installed[k])
		i++
	}

	respondOK(w, ret)
}

// GET /api/v1/packages/available?type=...&source=...
func (h *RESTHandler) getAvailablePackages(w http.ResponseWriter, r *http.Request) {
	typeStr := queryParam(r, "type")
	source := queryParam(r, "source")

	pm, err := getPackageManager(PackageType(typeStr))
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	available, err := pm.ListRemote(r.Context(), source)
	if err != nil {
		respondInternalError(w, err)
		return
	}

	ret := make([]*Package, len(available))
	i := 0
	for _, k := range sortedPackageSpecKeys(available) {
		p := available[k]
		ret[i] = remotePackageToPackage(p, available)
		i++
	}

	respondOK(w, ret)
}

// POST /api/v1/packages/install
func (h *RESTHandler) installPackages(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Type     PackageType                `json:"type"`
		Packages []*models.PackageSpecInput `json:"packages"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	pm, err := getPackageManager(input.Type)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	mgr := manager.GetInstance()
	t := &task.InstallPackagesJob{
		PackagesJob: task.PackagesJob{
			PackageManager: pm,
			OnComplete:     func() { refreshPackageType(input.Type) },
		},
		Packages: input.Packages,
	}
	jobID := mgr.JobManager.Add(r.Context(), "Installing packages...", t)

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/packages/update
func (h *RESTHandler) updatePackages(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Type     PackageType                `json:"type"`
		Packages []*models.PackageSpecInput `json:"packages"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	pm, err := getPackageManager(input.Type)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	mgr := manager.GetInstance()
	t := &task.UpdatePackagesJob{
		PackagesJob: task.PackagesJob{
			PackageManager: pm,
			OnComplete:     func() { refreshPackageType(input.Type) },
		},
		Packages: input.Packages,
	}
	jobID := mgr.JobManager.Add(r.Context(), "Updating packages...", t)

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// POST /api/v1/packages/uninstall
func (h *RESTHandler) uninstallPackages(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Type     PackageType                `json:"type"`
		Packages []*models.PackageSpecInput `json:"packages"`
	}
	if err := decodeBody(r, &input); err != nil {
		respondBadRequest(w, err)
		return
	}

	pm, err := getPackageManager(input.Type)
	if err != nil {
		respondBadRequest(w, err)
		return
	}

	mgr := manager.GetInstance()
	t := &task.UninstallPackagesJob{
		PackagesJob: task.PackagesJob{
			PackageManager: pm,
			OnComplete:     func() { refreshPackageType(input.Type) },
		},
		Packages: input.Packages,
	}
	jobID := mgr.JobManager.Add(r.Context(), "Uninstalling packages...", t)

	respondOK(w, map[string]string{"job_id": strconv.Itoa(jobID)})
}

// --- Jobs ---

// GET /api/v1/jobs
func (h *RESTHandler) getJobQueue(w http.ResponseWriter, r *http.Request) {
	queue := manager.GetInstance().JobManager.GetQueue()

	var jobs []*Job
	for _, j := range queue {
		jobs = append(jobs, jobToJobModel(j))
	}

	respondOK(w, jobs)
}

// GET /api/v1/jobs/{id}
func (h *RESTHandler) findJob(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("invalid job id: %w", err))
		return
	}

	j := manager.GetInstance().JobManager.GetJob(id)
	if j == nil {
		respondNotFound(w)
		return
	}

	ret := jobToJobModel(*j)
	respondOK(w, ret)
}

// DELETE /api/v1/jobs/{id}
func (h *RESTHandler) stopJob(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		respondBadRequest(w, fmt.Errorf("invalid job id: %w", err))
		return
	}

	manager.GetInstance().JobManager.CancelJob(id)

	respondOK(w, true)
}

// DELETE /api/v1/jobs
func (h *RESTHandler) stopAllJobs(w http.ResponseWriter, r *http.Request) {
	manager.GetInstance().JobManager.CancelAll()
	respondOK(w, true)
}
