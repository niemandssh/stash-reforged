// api_helpers.go - Helper functions extracted from resolver files for use by REST handlers.
package api

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/pkg"
)

// --- Job helpers (from resolver_query_job.go) ---

func jobToJobModel(j job.Job) *Job {
	ret := &Job{
		ID:          strconv.Itoa(j.ID),
		Status:      JobStatus(j.Status),
		Description: j.Description,
		SubTasks:    j.Details,
		StartTime:   j.StartTime,
		EndTime:     j.EndTime,
		AddTime:     j.AddTime,
		Error:       j.Error,
	}

	if j.Progress != -1 {
		ret.Progress = &j.Progress
	}

	return ret
}

// --- Package helpers (from resolver_query_package.go) ---

var ErrInvalidPackageType = errors.New("invalid package type")

func getPackageManager(typeArg PackageType) (*pkg.Manager, error) {
	var pm *pkg.Manager
	switch typeArg {
	case PackageTypeScraper:
		pm = manager.GetInstance().ScraperPackageManager
	case PackageTypePlugin:
		pm = manager.GetInstance().PluginPackageManager
	default:
		return nil, ErrInvalidPackageType
	}

	if pm == nil {
		return nil, fmt.Errorf("%s package manager not initialized", typeArg)
	}

	return pm, nil
}

func manifestToPackage(p pkg.Manifest) *Package {
	ret := &Package{
		PackageID: p.ID,
		Name:      p.Name,
		SourceURL: p.RepositoryURL,
	}

	if len(p.Version) > 0 {
		ret.Version = &p.Version
	}
	if !p.Date.IsZero() {
		ret.Date = &p.Date.Time
	}

	ret.Metadata = p.Metadata
	if ret.Metadata == nil {
		ret.Metadata = make(map[string]interface{})
	}

	return ret
}

func remotePackageToPackage(p pkg.RemotePackage, index pkg.RemotePackageIndex) *Package {
	ret := &Package{
		PackageID: p.ID,
		Name:      p.Name,
	}

	if len(p.Version) > 0 {
		ret.Version = &p.Version
	}
	if !p.Date.IsZero() {
		ret.Date = &p.Date.Time
	}

	ret.Metadata = p.Metadata
	if ret.Metadata == nil {
		ret.Metadata = make(map[string]interface{})
	}

	ret.SourceURL = p.Repository.Path()

	for _, r := range p.Requires {
		// required packages must come from the same source
		spec := models.PackageSpecInput{
			ID:        r,
			SourceURL: p.Repository.Path(),
		}

		req, found := index[spec]
		if !found {
			// shouldn't happen, but we'll ignore it
			continue
		}

		ret.Requires = append(ret.Requires, remotePackageToPackage(req, index))
	}

	return ret
}

func sortedPackageSpecKeys[V any](m map[models.PackageSpecInput]V) []models.PackageSpecInput {
	// sort keys
	var keys []models.PackageSpecInput
	for k := range m {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		a := keys[i]
		b := keys[j]

		aID := a.ID
		bID := b.ID

		if aID == bID {
			return a.SourceURL < b.SourceURL
		}

		aIDL := strings.ToLower(aID)
		bIDL := strings.ToLower(bID)

		if aIDL == bIDL {
			return aID < bID
		}

		return aIDL < bIDL
	})

	return keys
}

// --- DLNA helpers (from resolver_mutation_dlna.go) ---

func parseMinutes(minutes *int) *time.Duration {
	var ret *time.Duration
	if minutes != nil {
		d := time.Duration(*minutes) * time.Minute
		ret = &d
	}

	return ret
}

// --- Gallery helpers (from resolver_mutation_gallery.go) ---

func isStashPath(path string) bool {
	stashConfigs := manager.GetInstance().Config.GetStashPaths()
	for _, config := range stashConfigs {
		if path == config.Path {
			return true
		}
	}

	return false
}

// --- Group helpers (from resolver_mutation_group.go) ---

func groupPartialFromGroupUpdateInput(translator changesetTranslator, input GroupUpdateInput) (ret models.GroupPartial, err error) {
	updatedGroup := models.NewGroupPartial()

	updatedGroup.Name = translator.optionalString(input.Name, "name")
	updatedGroup.Aliases = translator.optionalString(input.Aliases, "aliases")
	updatedGroup.Duration = translator.optionalInt(input.Duration, "duration")
	updatedGroup.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedGroup.Director = translator.optionalString(input.Director, "director")
	updatedGroup.Synopsis = translator.optionalString(input.Synopsis, "synopsis")

	updatedGroup.Date, err = translator.optionalDate(input.Date, "date")
	if err != nil {
		err = fmt.Errorf("converting date: %w", err)
		return
	}
	updatedGroup.StudioID, err = translator.optionalIntFromString(input.StudioID, "studio_id")
	if err != nil {
		err = fmt.Errorf("converting studio id: %w", err)
		return
	}

	updatedGroup.TagIDs, err = translator.updateIds(input.TagIds, "tag_ids")
	if err != nil {
		err = fmt.Errorf("converting tag ids: %w", err)
		return
	}

	updatedGroup.ContainingGroups, err = translator.updateGroupIDDescriptions(input.ContainingGroups, "containing_groups")
	if err != nil {
		err = fmt.Errorf("converting containing group ids: %w", err)
		return
	}

	updatedGroup.SubGroups, err = translator.updateGroupIDDescriptions(input.SubGroups, "sub_groups")
	if err != nil {
		err = fmt.Errorf("converting containing group ids: %w", err)
		return
	}

	updatedGroup.URLs = translator.updateStrings(input.Urls, "urls")

	return updatedGroup, nil
}

func groupPartialFromBulkGroupUpdateInput(translator changesetTranslator, input BulkGroupUpdateInput) (ret models.GroupPartial, err error) {
	updatedGroup := models.NewGroupPartial()

	updatedGroup.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedGroup.Director = translator.optionalString(input.Director, "director")

	updatedGroup.StudioID, err = translator.optionalIntFromString(input.StudioID, "studio_id")
	if err != nil {
		err = fmt.Errorf("converting studio id: %w", err)
		return
	}

	updatedGroup.TagIDs, err = translator.updateIdsBulk(input.TagIds, "tag_ids")
	if err != nil {
		err = fmt.Errorf("converting tag ids: %w", err)
		return
	}

	updatedGroup.ContainingGroups, err = translator.updateGroupIDDescriptionsBulk(input.ContainingGroups, "containing_groups")
	if err != nil {
		err = fmt.Errorf("converting containing group ids: %w", err)
		return
	}

	updatedGroup.SubGroups, err = translator.updateGroupIDDescriptionsBulk(input.SubGroups, "sub_groups")
	if err != nil {
		err = fmt.Errorf("converting containing group ids: %w", err)
		return
	}

	updatedGroup.URLs = translator.optionalURLsBulk(input.Urls, nil)

	return updatedGroup, nil
}

// --- Package helpers (from resolver_mutation_package.go) ---

func refreshPackageType(typeArg PackageType) {
	mgr := manager.GetInstance()

	if typeArg == PackageTypePlugin {
		mgr.RefreshPluginCache()
	} else if typeArg == PackageTypeScraper {
		mgr.RefreshScraperCache()
	}
}

// --- Performer helpers (from resolver_mutation_performer.go) ---

const (
	twitterURL   = "https://twitter.com"
	instagramURL = "https://instagram.com"
)

// --- Scene helpers (from resolver_mutation_scene.go) ---

func scenePartialFromInput(input models.SceneUpdateInput, translator changesetTranslator) (*models.ScenePartial, error) {
	updatedScene := models.NewScenePartial()

	updatedScene.Title = translator.optionalString(input.Title, "title")
	updatedScene.Code = translator.optionalString(input.Code, "code")
	updatedScene.Details = translator.optionalString(input.Details, "details")
	updatedScene.Director = translator.optionalString(input.Director, "director")
	updatedScene.Rating = translator.optionalInt(input.Rating100, "rating100")

	updatedScene.PlayDuration = translator.optionalFloat64(input.PlayDuration, "play_duration")
	updatedScene.StartTime = translator.optionalFloat64(input.StartTime, "start_time")
	updatedScene.EndTime = translator.optionalFloat64(input.EndTime, "end_time")
	updatedScene.Organized = translator.optionalBool(input.Organized, "organized")
	updatedScene.Pinned = translator.optionalBool(input.Pinned, "pinned")

	updatedScene.VideoFilters = input.VideoFilters
	updatedScene.VideoTransforms = input.VideoTransforms
	updatedScene.IsBroken = translator.optionalBool(input.IsBroken, "is_broken")
	updatedScene.IsNotBroken = translator.optionalBool(input.IsNotBroken, "is_not_broken")
	updatedScene.AudioOffsetMs = translator.optionalInt(input.AudioOffsetMs, "audio_offset_ms")
	updatedScene.AudioPlaybackSpeed = translator.optionalFloat64(input.AudioPlaybackSpeed, "audio_playback_speed")
	updatedScene.ForceHLS = translator.optionalBool(input.ForceHLS, "force_hls")
	updatedScene.DisableNextSceneOverlay = translator.optionalBool(input.DisableNextSceneOverlay, "disable_next_scene_overlay")

	if updatedScene.IsNotBroken.Set && updatedScene.IsNotBroken.Value {
		updatedScene.IsBroken = models.NewOptionalBool(false)
	}

	updatedScene.StashIDs = translator.updateStashIDs(input.StashIds, "stash_ids")

	var err error

	updatedScene.Date, err = translator.optionalDate(input.Date, "date")
	if err != nil {
		return nil, fmt.Errorf("converting date: %w", err)
	}
	updatedScene.ShootDate, err = translator.optionalDate(input.ShootDate, "shoot_date")
	if err != nil {
		return nil, fmt.Errorf("converting shoot_date: %w", err)
	}
	updatedScene.StudioID, err = translator.optionalIntFromString(input.StudioID, "studio_id")
	if err != nil {
		return nil, fmt.Errorf("converting studio id: %w", err)
	}

	updatedScene.URLs = translator.optionalURLs(input.Urls, input.URL)

	updatedScene.PrimaryFileID, err = translator.fileIDPtrFromString(input.PrimaryFileID)
	if err != nil {
		return nil, fmt.Errorf("converting primary file id: %w", err)
	}

	updatedScene.PerformerIDs, err = translator.updateIds(input.PerformerIds, "performer_ids")
	if err != nil {
		return nil, fmt.Errorf("converting performer ids: %w", err)
	}
	updatedScene.TagIDs, err = translator.updateIds(input.TagIds, "tag_ids")
	if err != nil {
		return nil, fmt.Errorf("converting tag ids: %w", err)
	}
	updatedScene.PerformerTagIDs, err = translator.updatePerformerTags(input.PerformerTagIds, "performer_tag_ids")
	if err != nil {
		return nil, fmt.Errorf("converting performer tag ids: %w", err)
	}
	updatedScene.GalleryIDs, err = translator.updateIds(input.GalleryIds, "gallery_ids")
	if err != nil {
		return nil, fmt.Errorf("converting gallery ids: %w", err)
	}

	if translator.hasField("groups") {
		updatedScene.GroupIDs, err = translator.updateGroupIDs(input.Groups, "groups")
		if err != nil {
			return nil, fmt.Errorf("converting groups: %w", err)
		}
	} else if translator.hasField("movies") {
		updatedScene.GroupIDs, err = translator.updateGroupIDsFromMovies(input.Movies, "movies")
		if err != nil {
			return nil, fmt.Errorf("converting movies: %w", err)
		}
	}

	if translator.hasField("scene_performers") {
		updatedScene.ScenePerformers = &models.UpdateScenePerformers{
			ScenePerformers: input.ScenePerformers,
			Mode:            models.RelationshipUpdateModeSet,
		}
	}

	return &updatedScene, nil
}

// --- Log helpers (from resolver_query_logs.go) ---

func getLogLevel(logType string) LogLevel {
	switch logType {
	case "progress":
		return LogLevelProgress
	case "trace":
		return LogLevelTrace
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn":
		return LogLevelWarning
	case "error":
		return LogLevelError
	default:
		return LogLevelDebug
	}
}
