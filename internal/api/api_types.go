// API types - extracted from generated GraphQL models for use with REST API.
package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/stashapp/stash/internal/identify"
	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scraper"
)

type AddTempDLNAIPInput struct {
	Address string `json:"address"`
	// Duration to enable, in minutes. 0 or null for indefinite.
	Duration *int `json:"duration,omitempty"`
}

type AnonymiseDatabaseInput struct {
	Download *bool `json:"download,omitempty"`
}

type AssignSceneFileInput struct {
	SceneID string `json:"scene_id"`
	FileID  string `json:"file_id"`
}

type BackupDatabaseInput struct {
	Download *bool `json:"download,omitempty"`
}

type BulkGalleryUpdateInput struct {
	ClientMutationID *string            `json:"clientMutationId,omitempty"`
	Ids              []string           `json:"ids,omitempty"`
	Code             *string            `json:"code,omitempty"`
	URL              *string            `json:"url,omitempty"`
	Urls             *BulkUpdateStrings `json:"urls,omitempty"`
	Date             *string            `json:"date,omitempty"`
	Details          *string            `json:"details,omitempty"`
	Photographer     *string            `json:"photographer,omitempty"`
	Rating100        *int               `json:"rating100,omitempty"`
	Organized        *bool              `json:"organized,omitempty"`
	DisplayMode      *int               `json:"display_mode,omitempty"`
	SceneIds         *BulkUpdateIds     `json:"scene_ids,omitempty"`
	StudioID         *string            `json:"studio_id,omitempty"`
	TagIds           *BulkUpdateIds     `json:"tag_ids,omitempty"`
	PerformerIds     *BulkUpdateIds     `json:"performer_ids,omitempty"`
}

type BulkGroupUpdateInput struct {
	ClientMutationID *string                           `json:"clientMutationId,omitempty"`
	Ids              []string                          `json:"ids,omitempty"`
	Rating100        *int                              `json:"rating100,omitempty"`
	StudioID         *string                           `json:"studio_id,omitempty"`
	Director         *string                           `json:"director,omitempty"`
	Urls             *BulkUpdateStrings                `json:"urls,omitempty"`
	TagIds           *BulkUpdateIds                    `json:"tag_ids,omitempty"`
	ContainingGroups *BulkUpdateGroupDescriptionsInput `json:"containing_groups,omitempty"`
	SubGroups        *BulkUpdateGroupDescriptionsInput `json:"sub_groups,omitempty"`
}

type BulkImageUpdateInput struct {
	ClientMutationID *string            `json:"clientMutationId,omitempty"`
	Ids              []string           `json:"ids,omitempty"`
	Title            *string            `json:"title,omitempty"`
	Code             *string            `json:"code,omitempty"`
	Rating100        *int               `json:"rating100,omitempty"`
	Organized        *bool              `json:"organized,omitempty"`
	URL              *string            `json:"url,omitempty"`
	Urls             *BulkUpdateStrings `json:"urls,omitempty"`
	Date             *string            `json:"date,omitempty"`
	Details          *string            `json:"details,omitempty"`
	Photographer     *string            `json:"photographer,omitempty"`
	StudioID         *string            `json:"studio_id,omitempty"`
	PerformerIds     *BulkUpdateIds     `json:"performer_ids,omitempty"`
	TagIds           *BulkUpdateIds     `json:"tag_ids,omitempty"`
	GalleryIds       *BulkUpdateIds     `json:"gallery_ids,omitempty"`
}

type BulkMovieUpdateInput struct {
	ClientMutationID *string            `json:"clientMutationId,omitempty"`
	Ids              []string           `json:"ids,omitempty"`
	Rating100        *int               `json:"rating100,omitempty"`
	StudioID         *string            `json:"studio_id,omitempty"`
	Director         *string            `json:"director,omitempty"`
	Urls             *BulkUpdateStrings `json:"urls,omitempty"`
	TagIds           *BulkUpdateIds     `json:"tag_ids,omitempty"`
}

type BulkPerformerUpdateInput struct {
	ClientMutationID *string                   `json:"clientMutationId,omitempty"`
	Ids              []string                  `json:"ids,omitempty"`
	Disambiguation   *string                   `json:"disambiguation,omitempty"`
	URL              *string                   `json:"url,omitempty"`
	Urls             *BulkUpdateStrings        `json:"urls,omitempty"`
	Gender           *models.GenderEnum        `json:"gender,omitempty"`
	Birthdate        *string                   `json:"birthdate,omitempty"`
	Ethnicity        *string                   `json:"ethnicity,omitempty"`
	Country          *string                   `json:"country,omitempty"`
	EyeColor         *string                   `json:"eye_color,omitempty"`
	HeightCm         *int                      `json:"height_cm,omitempty"`
	Measurements     *string                   `json:"measurements,omitempty"`
	FakeTits         *string                   `json:"fake_tits,omitempty"`
	PenisLength      *float64                  `json:"penis_length,omitempty"`
	Circumcised      *models.CircumisedEnum    `json:"circumcised,omitempty"`
	CareerLength     *string                   `json:"career_length,omitempty"`
	Tattoos          *string                   `json:"tattoos,omitempty"`
	Piercings        *string                   `json:"piercings,omitempty"`
	AliasList        *BulkUpdateStrings        `json:"alias_list,omitempty"`
	Twitter          *string                   `json:"twitter,omitempty"`
	Instagram        *string                   `json:"instagram,omitempty"`
	Favorite         *bool                     `json:"favorite,omitempty"`
	TagIds           *BulkUpdateIds            `json:"tag_ids,omitempty"`
	Rating100        *int                      `json:"rating100,omitempty"`
	Details          *string                   `json:"details,omitempty"`
	DeathDate        *string                   `json:"death_date,omitempty"`
	HairColor        *string                   `json:"hair_color,omitempty"`
	Weight           *int                      `json:"weight,omitempty"`
	IgnoreAutoTag    *bool                     `json:"ignore_auto_tag,omitempty"`
	CustomFields     *models.CustomFieldsInput `json:"custom_fields,omitempty"`
}

type BulkSceneUpdateInput struct {
	ClientMutationID *string            `json:"clientMutationId,omitempty"`
	Ids              []string           `json:"ids,omitempty"`
	Title            *string            `json:"title,omitempty"`
	Code             *string            `json:"code,omitempty"`
	Details          *string            `json:"details,omitempty"`
	Director         *string            `json:"director,omitempty"`
	URL              *string            `json:"url,omitempty"`
	Urls             *BulkUpdateStrings `json:"urls,omitempty"`
	Date             *string            `json:"date,omitempty"`
	ShootDate        *string            `json:"shoot_date,omitempty"`
	Rating100        *int               `json:"rating100,omitempty"`
	Organized        *bool              `json:"organized,omitempty"`
	IsBroken         *bool              `json:"is_broken,omitempty"`
	IsNotBroken      *bool              `json:"is_not_broken,omitempty"`
	StudioID         *string            `json:"studio_id,omitempty"`
	GalleryIds       *BulkUpdateIds     `json:"gallery_ids,omitempty"`
	PerformerIds     *BulkUpdateIds     `json:"performer_ids,omitempty"`
	TagIds           *BulkUpdateIds     `json:"tag_ids,omitempty"`
	GroupIds         *BulkUpdateIds     `json:"group_ids,omitempty"`
	MovieIds         *BulkUpdateIds     `json:"movie_ids,omitempty"`
}

type BulkTagUpdateInput struct {
	Ids               []string           `json:"ids,omitempty"`
	Description       *string            `json:"description,omitempty"`
	Aliases           *BulkUpdateStrings `json:"aliases,omitempty"`
	IgnoreAutoTag     *bool              `json:"ignore_auto_tag,omitempty"`
	IsPoseTag         *bool              `json:"is_pose_tag,omitempty"`
	IgnoreSuggestions *bool              `json:"ignore_suggestions,omitempty"`
	Favorite          *bool              `json:"favorite,omitempty"`
	Weight            *float64           `json:"weight,omitempty"`
	Color             *string            `json:"color,omitempty"`
	ParentIds         *BulkUpdateIds     `json:"parent_ids,omitempty"`
	ChildIds          *BulkUpdateIds     `json:"child_ids,omitempty"`
}

type BulkUpdateGroupDescriptionsInput struct {
	Groups []*GroupDescriptionInput      `json:"groups"`
	Mode   models.RelationshipUpdateMode `json:"mode"`
}

type BulkUpdateIds struct {
	Ids  []string                      `json:"ids,omitempty"`
	Mode models.RelationshipUpdateMode `json:"mode"`
}

type BulkUpdateStrings struct {
	Values []string                      `json:"values,omitempty"`
	Mode   models.RelationshipUpdateMode `json:"mode"`
}

type ColorPresetCreateInput struct {
	Name                       string  `json:"name"`
	Color                      string  `json:"color"`
	Sort                       *int    `json:"sort,omitempty"`
	TagRequirementsDescription *string `json:"tag_requirements_description,omitempty"`
	RequiredForRequirements    *bool   `json:"required_for_requirements,omitempty"`
}

type ColorPresetDestroyInput struct {
	ID string `json:"id"`
}

type ColorPresetUpdateInput struct {
	ID                         string  `json:"id"`
	Name                       *string `json:"name,omitempty"`
	Color                      *string `json:"color,omitempty"`
	Sort                       *int    `json:"sort,omitempty"`
	TagRequirementsDescription *string `json:"tag_requirements_description,omitempty"`
	RequiredForRequirements    *bool   `json:"required_for_requirements,omitempty"`
}

type ConfigDLNAInput struct {
	ServerName *string `json:"serverName,omitempty"`
	// True if DLNA service should be enabled by default
	Enabled *bool `json:"enabled,omitempty"`
	// Defaults to 1338
	Port *int `json:"port,omitempty"`
	// List of IPs whitelisted for DLNA service
	WhitelistedIPs []string `json:"whitelistedIPs,omitempty"`
	// List of interfaces to run DLNA on. Empty for all
	Interfaces []string `json:"interfaces,omitempty"`
	// Order to sort videos
	VideoSortOrder *string `json:"videoSortOrder,omitempty"`
}

type ConfigDLNAResult struct {
	ServerName string `json:"serverName"`
	// True if DLNA service should be enabled by default
	Enabled bool `json:"enabled"`
	// Defaults to 1338
	Port int `json:"port"`
	// List of IPs whitelisted for DLNA service
	WhitelistedIPs []string `json:"whitelistedIPs"`
	// List of interfaces to run DLNA on. Empty for all
	Interfaces []string `json:"interfaces"`
	// Order to sort videos
	VideoSortOrder string `json:"videoSortOrder"`
}

type ConfigDefaultSettingsInput struct {
	Scan     *manager.ScanMetadataInput     `json:"scan,omitempty"`
	Identify *identify.Options              `json:"identify,omitempty"`
	AutoTag  *manager.AutoTagMetadataInput  `json:"autoTag,omitempty"`
	Generate *manager.GenerateMetadataInput `json:"generate,omitempty"`
	// If true, delete file checkbox will be checked by default
	DeleteFile *bool `json:"deleteFile,omitempty"`
	// If true, delete generated files checkbox will be checked by default
	DeleteGenerated *bool `json:"deleteGenerated,omitempty"`
}

type ConfigDefaultSettingsResult struct {
	Scan     *config.ScanMetadataOptions     `json:"scan,omitempty"`
	Identify *identify.Options               `json:"identify,omitempty"`
	AutoTag  *config.AutoTagMetadataOptions  `json:"autoTag,omitempty"`
	Generate *models.GenerateMetadataOptions `json:"generate,omitempty"`
	// If true, delete file checkbox will be checked by default
	DeleteFile *bool `json:"deleteFile,omitempty"`
	// If true, delete generated supporting files checkbox will be checked by default
	DeleteGenerated *bool `json:"deleteGenerated,omitempty"`
}

type ConfigDisableDropdownCreateInput struct {
	Performer *bool `json:"performer,omitempty"`
	Tag       *bool `json:"tag,omitempty"`
	Studio    *bool `json:"studio,omitempty"`
	Movie     *bool `json:"movie,omitempty"`
}

type ConfigGeneralInput struct {
	// Array of file paths to content
	Stashes []*config.StashConfigInput `json:"stashes,omitempty"`
	// Path to the SQLite database
	DatabasePath *string `json:"databasePath,omitempty"`
	// Path to backup directory
	BackupDirectoryPath *string `json:"backupDirectoryPath,omitempty"`
	// Path to generated files
	GeneratedPath *string `json:"generatedPath,omitempty"`
	// Path to import/export files
	MetadataPath *string `json:"metadataPath,omitempty"`
	// Path to scrapers
	ScrapersPath *string `json:"scrapersPath,omitempty"`
	// Path to plugins
	PluginsPath *string `json:"pluginsPath,omitempty"`
	// Path to cache
	CachePath *string `json:"cachePath,omitempty"`
	// Path to blobs - required for filesystem blob storage
	BlobsPath *string `json:"blobsPath,omitempty"`
	// Where to store blobs
	BlobsStorage *config.BlobsStorageType `json:"blobsStorage,omitempty"`
	// Path to the ffmpeg binary. If empty, stash will attempt to find it in the path or config directory
	FfmpegPath *string `json:"ffmpegPath,omitempty"`
	// Path to the ffprobe binary. If empty, stash will attempt to find it in the path or config directory
	FfprobePath *string `json:"ffprobePath,omitempty"`
	// Whether to calculate MD5 checksums for scene video files
	CalculateMd5 *bool `json:"calculateMD5,omitempty"`
	// Hash algorithm to use for generated file naming
	VideoFileNamingAlgorithm *models.HashAlgorithm `json:"videoFileNamingAlgorithm,omitempty"`
	// Number of parallel tasks to start during scan/generate
	ParallelTasks *int `json:"parallelTasks,omitempty"`
	// Include audio stream in previews
	PreviewAudio *bool `json:"previewAudio,omitempty"`
	// Number of segments in a preview file
	PreviewSegments *int `json:"previewSegments,omitempty"`
	// Preview segment duration, in seconds
	PreviewSegmentDuration *float64 `json:"previewSegmentDuration,omitempty"`
	// Duration of start of video to exclude when generating previews
	PreviewExcludeStart *string `json:"previewExcludeStart,omitempty"`
	// Duration of end of video to exclude when generating previews
	PreviewExcludeEnd *string `json:"previewExcludeEnd,omitempty"`
	// Preset when generating preview
	PreviewPreset *models.PreviewPreset `json:"previewPreset,omitempty"`
	// Transcode Hardware Acceleration
	TranscodeHardwareAcceleration *bool `json:"transcodeHardwareAcceleration,omitempty"`
	// Max generated transcode size
	MaxTranscodeSize *models.StreamingResolutionEnum `json:"maxTranscodeSize,omitempty"`
	// Max streaming transcode size
	MaxStreamingTranscodeSize *models.StreamingResolutionEnum `json:"maxStreamingTranscodeSize,omitempty"`
	// ffmpeg transcode input args - injected before input file
	// These are applied to generated transcodes (previews and transcodes)
	TranscodeInputArgs []string `json:"transcodeInputArgs,omitempty"`
	// ffmpeg transcode output args - injected before output file
	// These are applied to generated transcodes (previews and transcodes)
	TranscodeOutputArgs []string `json:"transcodeOutputArgs,omitempty"`
	// ffmpeg stream input args - injected before input file
	// These are applied when live transcoding
	LiveTranscodeInputArgs []string `json:"liveTranscodeInputArgs,omitempty"`
	// ffmpeg stream output args - injected before output file
	// These are applied when live transcoding
	LiveTranscodeOutputArgs []string `json:"liveTranscodeOutputArgs,omitempty"`
	// whether to include range in generated funscript heatmaps
	DrawFunscriptHeatmapRange *bool `json:"drawFunscriptHeatmapRange,omitempty"`
	// Write image thumbnails to disk when generating on the fly
	WriteImageThumbnails *bool `json:"writeImageThumbnails,omitempty"`
	// Create Image Clips from Video extensions when Videos are disabled in Library
	CreateImageClipsFromVideos *bool `json:"createImageClipsFromVideos,omitempty"`
	// Username
	Username *string `json:"username,omitempty"`
	// Password
	Password *string `json:"password,omitempty"`
	// Maximum session cookie age
	MaxSessionAge *int `json:"maxSessionAge,omitempty"`
	// Name of the log file
	LogFile *string `json:"logFile,omitempty"`
	// Whether to also output to stderr
	LogOut *bool `json:"logOut,omitempty"`
	// Minimum log level
	LogLevel *string `json:"logLevel,omitempty"`
	// Whether to log http access
	LogAccess *bool `json:"logAccess,omitempty"`
	// True if galleries should be created from folders with images
	CreateGalleriesFromFolders *bool `json:"createGalleriesFromFolders,omitempty"`
	// Regex used to identify images as gallery covers
	GalleryCoverRegex *string `json:"galleryCoverRegex,omitempty"`
	// Array of video file extensions
	VideoExtensions []string `json:"videoExtensions,omitempty"`
	// Array of image file extensions
	ImageExtensions []string `json:"imageExtensions,omitempty"`
	// Array of gallery zip file extensions
	GalleryExtensions []string `json:"galleryExtensions,omitempty"`
	// Array of file regexp to exclude from Video Scans
	Excludes []string `json:"excludes,omitempty"`
	// Array of file regexp to exclude from Image Scans
	ImageExcludes []string `json:"imageExcludes,omitempty"`
	// Custom Performer Image Location
	CustomPerformerImageLocation *string `json:"customPerformerImageLocation,omitempty"`
	// Stash-box instances used for tagging
	StashBoxes []*config.StashBoxInput `json:"stashBoxes,omitempty"`
	// Python path - resolved using path if unset
	PythonPath *string `json:"pythonPath,omitempty"`
	// Source of scraper packages
	ScraperPackageSources []*PackageSourceInput `json:"scraperPackageSources,omitempty"`
	// Source of plugin packages
	PluginPackageSources []*PackageSourceInput `json:"pluginPackageSources,omitempty"`
}

type ConfigGeneralResult struct {
	// Array of file paths to content
	Stashes []*config.StashConfig `json:"stashes"`
	// Path to the SQLite database
	DatabasePath string `json:"databasePath"`
	// Path to backup directory
	BackupDirectoryPath string `json:"backupDirectoryPath"`
	// Path to generated files
	GeneratedPath string `json:"generatedPath"`
	// Path to import/export files
	MetadataPath string `json:"metadataPath"`
	// Path to the config file used
	ConfigFilePath string `json:"configFilePath"`
	// Path to scrapers
	ScrapersPath string `json:"scrapersPath"`
	// Path to plugins
	PluginsPath string `json:"pluginsPath"`
	// Path to cache
	CachePath string `json:"cachePath"`
	// Path to blobs - required for filesystem blob storage
	BlobsPath string `json:"blobsPath"`
	// Where to store blobs
	BlobsStorage config.BlobsStorageType `json:"blobsStorage"`
	// Path to the ffmpeg binary. If empty, stash will attempt to find it in the path or config directory
	FfmpegPath string `json:"ffmpegPath"`
	// Path to the ffprobe binary. If empty, stash will attempt to find it in the path or config directory
	FfprobePath string `json:"ffprobePath"`
	// Whether to calculate MD5 checksums for scene video files
	CalculateMd5 bool `json:"calculateMD5"`
	// Hash algorithm to use for generated file naming
	VideoFileNamingAlgorithm models.HashAlgorithm `json:"videoFileNamingAlgorithm"`
	// Number of parallel tasks to start during scan/generate
	ParallelTasks int `json:"parallelTasks"`
	// Include audio stream in previews
	PreviewAudio bool `json:"previewAudio"`
	// Number of segments in a preview file
	PreviewSegments int `json:"previewSegments"`
	// Preview segment duration, in seconds
	PreviewSegmentDuration float64 `json:"previewSegmentDuration"`
	// Duration of start of video to exclude when generating previews
	PreviewExcludeStart string `json:"previewExcludeStart"`
	// Duration of end of video to exclude when generating previews
	PreviewExcludeEnd string `json:"previewExcludeEnd"`
	// Preset when generating preview
	PreviewPreset models.PreviewPreset `json:"previewPreset"`
	// Transcode Hardware Acceleration
	TranscodeHardwareAcceleration bool `json:"transcodeHardwareAcceleration"`
	// Max generated transcode size
	MaxTranscodeSize *models.StreamingResolutionEnum `json:"maxTranscodeSize,omitempty"`
	// Max streaming transcode size
	MaxStreamingTranscodeSize *models.StreamingResolutionEnum `json:"maxStreamingTranscodeSize,omitempty"`
	// ffmpeg transcode input args - injected before input file
	// These are applied to generated transcodes (previews and transcodes)
	TranscodeInputArgs []string `json:"transcodeInputArgs"`
	// ffmpeg transcode output args - injected before output file
	// These are applied to generated transcodes (previews and transcodes)
	TranscodeOutputArgs []string `json:"transcodeOutputArgs"`
	// ffmpeg stream input args - injected before input file
	// These are applied when live transcoding
	LiveTranscodeInputArgs []string `json:"liveTranscodeInputArgs"`
	// ffmpeg stream output args - injected before output file
	// These are applied when live transcoding
	LiveTranscodeOutputArgs []string `json:"liveTranscodeOutputArgs"`
	// whether to include range in generated funscript heatmaps
	DrawFunscriptHeatmapRange bool `json:"drawFunscriptHeatmapRange"`
	// Write image thumbnails to disk when generating on the fly
	WriteImageThumbnails bool `json:"writeImageThumbnails"`
	// Create Image Clips from Video extensions when Videos are disabled in Library
	CreateImageClipsFromVideos bool `json:"createImageClipsFromVideos"`
	// API Key
	APIKey string `json:"apiKey"`
	// Username
	Username string `json:"username"`
	// Password
	Password string `json:"password"`
	// Maximum session cookie age
	MaxSessionAge int `json:"maxSessionAge"`
	// Name of the log file
	LogFile *string `json:"logFile,omitempty"`
	// Whether to also output to stderr
	LogOut bool `json:"logOut"`
	// Minimum log level
	LogLevel string `json:"logLevel"`
	// Whether to log http access
	LogAccess bool `json:"logAccess"`
	// Array of video file extensions
	VideoExtensions []string `json:"videoExtensions"`
	// Array of image file extensions
	ImageExtensions []string `json:"imageExtensions"`
	// Array of gallery zip file extensions
	GalleryExtensions []string `json:"galleryExtensions"`
	// True if galleries should be created from folders with images
	CreateGalleriesFromFolders bool `json:"createGalleriesFromFolders"`
	// Regex used to identify images as gallery covers
	GalleryCoverRegex string `json:"galleryCoverRegex"`
	// Array of file regexp to exclude from Video Scans
	Excludes []string `json:"excludes"`
	// Array of file regexp to exclude from Image Scans
	ImageExcludes []string `json:"imageExcludes"`
	// Custom Performer Image Location
	CustomPerformerImageLocation *string `json:"customPerformerImageLocation,omitempty"`
	// Stash-box instances used for tagging
	StashBoxes []*models.StashBox `json:"stashBoxes"`
	// Python path - resolved using path if unset
	PythonPath string `json:"pythonPath"`
	// Source of scraper packages
	ScraperPackageSources []*models.PackageSource `json:"scraperPackageSources"`
	// Source of plugin packages
	PluginPackageSources []*models.PackageSource `json:"pluginPackageSources"`
}

type ConfigImageLightboxInput struct {
	SlideshowDelay             *int                             `json:"slideshowDelay,omitempty"`
	DisplayMode                *config.ImageLightboxDisplayMode `json:"displayMode,omitempty"`
	ScaleUp                    *bool                            `json:"scaleUp,omitempty"`
	ResetZoomOnNav             *bool                            `json:"resetZoomOnNav,omitempty"`
	ScrollMode                 *config.ImageLightboxScrollMode  `json:"scrollMode,omitempty"`
	ScrollAttemptsBeforeChange *int                             `json:"scrollAttemptsBeforeChange,omitempty"`
}

type ConfigInterfaceInput struct {
	// Ordered list of items that should be shown in the menu
	MenuItems []string `json:"menuItems,omitempty"`
	// Enable sound on mouseover previews
	SoundOnPreview *bool `json:"soundOnPreview,omitempty"`
	// Show title and tags in wall view
	WallShowTitle *bool `json:"wallShowTitle,omitempty"`
	// Wall playback type
	WallPlayback *string `json:"wallPlayback,omitempty"`
	// Show scene scrubber by default
	ShowScrubber *bool `json:"showScrubber,omitempty"`
	// Maximum duration (in seconds) in which a scene video will loop in the scene player
	MaximumLoopDuration *int `json:"maximumLoopDuration,omitempty"`
	// If true, video will autostart on load in the scene player
	AutostartVideo *bool `json:"autostartVideo,omitempty"`
	// If true, video will autostart when loading from play random or play selected
	AutostartVideoOnPlaySelected *bool `json:"autostartVideoOnPlaySelected,omitempty"`
	// If true, next scene in playlist will be played at video end by default
	ContinuePlaylistDefault *bool `json:"continuePlaylistDefault,omitempty"`
	// If true, studio overlays will be shown as text instead of logo images
	ShowStudioAsText *bool `json:"showStudioAsText,omitempty"`
	// Custom CSS
	CSS        *string `json:"css,omitempty"`
	CSSEnabled *bool   `json:"cssEnabled,omitempty"`
	// Custom Javascript
	Javascript        *string `json:"javascript,omitempty"`
	JavascriptEnabled *bool   `json:"javascriptEnabled,omitempty"`
	// Custom Locales
	CustomLocales        *string `json:"customLocales,omitempty"`
	CustomLocalesEnabled *bool   `json:"customLocalesEnabled,omitempty"`
	// Interface language
	Language      *string                   `json:"language,omitempty"`
	ImageLightbox *ConfigImageLightboxInput `json:"imageLightbox,omitempty"`
	// Set to true to disable creating new objects via the dropdown menus
	DisableDropdownCreate *ConfigDisableDropdownCreateInput `json:"disableDropdownCreate,omitempty"`
	// Timer for autoplay next video (in seconds)
	AutoplayNextVideoTimer *int `json:"autoplayNextVideoTimer,omitempty"`
	// Minimum rating threshold for Random button (0-100)
	RandomRatingThreshold *int `json:"randomRatingThreshold,omitempty"`
	// Minimum rating threshold for Random Best button (0-100)
	RandomBestRatingThreshold *int `json:"randomBestRatingThreshold,omitempty"`
	// Handy Connection Key
	HandyKey *string `json:"handyKey,omitempty"`
	// Funscript Time Offset
	FunscriptOffset *int `json:"funscriptOffset,omitempty"`
	// Whether to use Stash Hosted Funscript
	UseStashHostedFunscript *bool `json:"useStashHostedFunscript,omitempty"`
	// True if we should not auto-open a browser window on startup
	NoBrowser *bool `json:"noBrowser,omitempty"`
	// True if we should send notifications to the desktop
	NotificationsEnabled *bool `json:"notificationsEnabled,omitempty"`
	// Show percent of scene similarity in similar scenes
	ShowSimilarityPercent *bool `json:"showSimilarityPercent,omitempty"`
	// External video player command
	ExternalVideoPlayer *string `json:"externalVideoPlayer,omitempty"`
	// Redirect home page to scenes page
	RedirectHomeToScenes *bool `json:"redirectHomeToScenes,omitempty"`
}

type ConfigInterfaceResult struct {
	// Ordered list of items that should be shown in the menu
	MenuItems []string `json:"menuItems,omitempty"`
	// Enable sound on mouseover previews
	SoundOnPreview *bool `json:"soundOnPreview,omitempty"`
	// Show title and tags in wall view
	WallShowTitle *bool `json:"wallShowTitle,omitempty"`
	// Wall playback type
	WallPlayback *string `json:"wallPlayback,omitempty"`
	// Show scene scrubber by default
	ShowScrubber *bool `json:"showScrubber,omitempty"`
	// Maximum duration (in seconds) in which a scene video will loop in the scene player
	MaximumLoopDuration *int `json:"maximumLoopDuration,omitempty"`
	// True if we should not auto-open a browser window on startup
	NoBrowser *bool `json:"noBrowser,omitempty"`
	// True if we should send desktop notifications
	NotificationsEnabled *bool `json:"notificationsEnabled,omitempty"`
	// If true, video will autostart on load in the scene player
	AutostartVideo *bool `json:"autostartVideo,omitempty"`
	// If true, video will autostart when loading from play random or play selected
	AutostartVideoOnPlaySelected *bool `json:"autostartVideoOnPlaySelected,omitempty"`
	// If true, next scene in playlist will be played at video end by default
	ContinuePlaylistDefault *bool `json:"continuePlaylistDefault,omitempty"`
	// If true, studio overlays will be shown as text instead of logo images
	ShowStudioAsText *bool `json:"showStudioAsText,omitempty"`
	// Custom CSS
	CSS        *string `json:"css,omitempty"`
	CSSEnabled *bool   `json:"cssEnabled,omitempty"`
	// Custom Javascript
	Javascript        *string `json:"javascript,omitempty"`
	JavascriptEnabled *bool   `json:"javascriptEnabled,omitempty"`
	// Custom Locales
	CustomLocales        *string `json:"customLocales,omitempty"`
	CustomLocalesEnabled *bool   `json:"customLocalesEnabled,omitempty"`
	// Interface language
	Language      *string                           `json:"language,omitempty"`
	ImageLightbox *config.ConfigImageLightboxResult `json:"imageLightbox"`
	// Fields are true if creating via dropdown menus are disabled
	DisableDropdownCreate *config.ConfigDisableDropdownCreate `json:"disableDropdownCreate"`
	// Timer for autoplay next video (in seconds)
	AutoplayNextVideoTimer *int `json:"autoplayNextVideoTimer,omitempty"`
	// Minimum rating threshold for Random button (0-100)
	RandomRatingThreshold *int `json:"randomRatingThreshold,omitempty"`
	// Minimum rating threshold for Random Best button (0-100)
	RandomBestRatingThreshold *int `json:"randomBestRatingThreshold,omitempty"`
	// Handy Connection Key
	HandyKey *string `json:"handyKey,omitempty"`
	// Funscript Time Offset
	FunscriptOffset *int `json:"funscriptOffset,omitempty"`
	// Whether to use Stash Hosted Funscript
	UseStashHostedFunscript *bool `json:"useStashHostedFunscript,omitempty"`
	// Show percent of scene similarity in similar scenes
	ShowSimilarityPercent *bool `json:"showSimilarityPercent,omitempty"`
	// External video player command
	ExternalVideoPlayer *string `json:"externalVideoPlayer,omitempty"`
	// Redirect home page to scenes page
	RedirectHomeToScenes *bool `json:"redirectHomeToScenes,omitempty"`
}

// All configuration settings
type ConfigResult struct {
	General   *ConfigGeneralResult              `json:"general"`
	Interface *ConfigInterfaceResult            `json:"interface"`
	Dlna      *ConfigDLNAResult                 `json:"dlna"`
	Scraping  *ConfigScrapingResult             `json:"scraping"`
	Defaults  *ConfigDefaultSettingsResult      `json:"defaults"`
	UI        map[string]any                    `json:"ui"`
	Plugins   map[string]map[string]interface{} `json:"plugins"`
}

type ConfigScrapingInput struct {
	// Scraper user agent string
	ScraperUserAgent *string `json:"scraperUserAgent,omitempty"`
	// Scraper CDP path. Path to chrome executable or remote address
	ScraperCDPPath *string `json:"scraperCDPPath,omitempty"`
	// Whether the scraper should check for invalid certificates
	ScraperCertCheck *bool `json:"scraperCertCheck,omitempty"`
	// Tags blacklist during scraping
	ExcludeTagPatterns []string `json:"excludeTagPatterns,omitempty"`
}

type ConfigScrapingResult struct {
	// Scraper user agent string
	ScraperUserAgent *string `json:"scraperUserAgent,omitempty"`
	// Scraper CDP path. Path to chrome executable or remote address
	ScraperCDPPath *string `json:"scraperCDPPath,omitempty"`
	// Whether the scraper should check for invalid certificates
	ScraperCertCheck bool `json:"scraperCertCheck"`
	// Tags blacklist during scraping
	ExcludeTagPatterns []string `json:"excludeTagPatterns"`
}

type DestroyFilterInput struct {
	ID string `json:"id"`
}

// Directory structure of a path
type Directory struct {
	Path        string   `json:"path"`
	Parent      *string  `json:"parent,omitempty"`
	Directories []string `json:"directories"`
	Files       []string `json:"files"`
}

type DisableDLNAInput struct {
	// Duration to enable, in minutes. 0 or null for indefinite.
	Duration *int `json:"duration,omitempty"`
}

type EnableDLNAInput struct {
	// Duration to enable, in minutes. 0 or null for indefinite.
	Duration *int `json:"duration,omitempty"`
}

type FileSetFingerprintsInput struct {
	ID string `json:"id"`
	// only supplied fingerprint types will be modified
	Fingerprints []*SetFingerprintsInput `json:"fingerprints"`
}

type FindColorPresetsResultType struct {
	Count        int                   `json:"count"`
	ColorPresets []*models.ColorPreset `json:"color_presets"`
}

type FindFilesResultType struct {
	Count int `json:"count"`
	// Total megapixels of any image files
	Megapixels float64 `json:"megapixels"`
	// Total duration in seconds of any video files
	Duration float64 `json:"duration"`
	// Total file size in bytes
	Size  int        `json:"size"`
	Files []BaseFile `json:"files"`
}

type FindFoldersResultType struct {
	Count   int              `json:"count"`
	Folders []*models.Folder `json:"folders"`
}

type FindGalleriesResultType struct {
	Count     int               `json:"count"`
	Galleries []*models.Gallery `json:"galleries"`
}

type FindGalleryChaptersResultType struct {
	Count    int                      `json:"count"`
	Chapters []*models.GalleryChapter `json:"chapters"`
}

type FindGamesResultType struct {
	Count int            `json:"count"`
	Games []*models.Game `json:"games"`
}

type FindGroupsResultType struct {
	Count  int             `json:"count"`
	Groups []*models.Group `json:"groups"`
}

type FindImagesResultType struct {
	Count int `json:"count"`
	// Total megapixels of the images
	Megapixels float64 `json:"megapixels"`
	// Total file size in bytes
	Filesize float64         `json:"filesize"`
	Images   []*models.Image `json:"images"`
}

type FindJobInput struct {
	ID string `json:"id"`
}

type FindMoviesResultType struct {
	Count  int             `json:"count"`
	Movies []*models.Group `json:"movies"`
}

type FindPerformersResultType struct {
	Count      int                 `json:"count"`
	Performers []*models.Performer `json:"performers"`
}

type FindSceneMarkersResultType struct {
	Count        int                   `json:"count"`
	SceneMarkers []*models.SceneMarker `json:"scene_markers"`
}

type FindScenesResultType struct {
	Count int `json:"count"`
	// Total duration in seconds
	Duration float64 `json:"duration"`
	// Total file size in bytes
	Filesize float64         `json:"filesize"`
	Scenes   []*models.Scene `json:"scenes"`
}

type FindStudiosResultType struct {
	Count   int              `json:"count"`
	Studios []*models.Studio `json:"studios"`
}

type FindTagsResultType struct {
	Count int           `json:"count"`
	Tags  []*models.Tag `json:"tags"`
}

type GalleryAddInput struct {
	GalleryID string   `json:"gallery_id"`
	ImageIds  []string `json:"image_ids"`
}

type GalleryChapterCreateInput struct {
	GalleryID  string `json:"gallery_id"`
	Title      string `json:"title"`
	ImageIndex int    `json:"image_index"`
}

type GalleryChapterUpdateInput struct {
	ID         string  `json:"id"`
	GalleryID  *string `json:"gallery_id,omitempty"`
	Title      *string `json:"title,omitempty"`
	ImageIndex *int    `json:"image_index,omitempty"`
}

type GalleryCreateInput struct {
	Title        string   `json:"title"`
	Code         *string  `json:"code,omitempty"`
	URL          *string  `json:"url,omitempty"`
	Urls         []string `json:"urls,omitempty"`
	Date         *string  `json:"date,omitempty"`
	Details      *string  `json:"details,omitempty"`
	Photographer *string  `json:"photographer,omitempty"`
	Rating100    *int     `json:"rating100,omitempty"`
	Organized    *bool    `json:"organized,omitempty"`
	DisplayMode  *int     `json:"display_mode,omitempty"`
	SceneIds     []string `json:"scene_ids,omitempty"`
	StudioID     *string  `json:"studio_id,omitempty"`
	TagIds       []string `json:"tag_ids,omitempty"`
	PerformerIds []string `json:"performer_ids,omitempty"`
}

type GalleryPathsType struct {
	Cover   string `json:"cover"`
	Preview string `json:"preview"`
}

type GalleryRemoveInput struct {
	GalleryID string   `json:"gallery_id"`
	ImageIds  []string `json:"image_ids"`
}

type GalleryResetCoverInput struct {
	GalleryID string `json:"gallery_id"`
}

type GallerySetCoverInput struct {
	GalleryID    string `json:"gallery_id"`
	CoverImageID string `json:"cover_image_id"`
}

type GenerateAPIKeyInput struct {
	Clear *bool `json:"clear,omitempty"`
}

type GroupCreateInput struct {
	Name    string  `json:"name"`
	Aliases *string `json:"aliases,omitempty"`
	// Duration in seconds
	Duration         *int                     `json:"duration,omitempty"`
	Date             *string                  `json:"date,omitempty"`
	Rating100        *int                     `json:"rating100,omitempty"`
	StudioID         *string                  `json:"studio_id,omitempty"`
	Director         *string                  `json:"director,omitempty"`
	Synopsis         *string                  `json:"synopsis,omitempty"`
	Urls             []string                 `json:"urls,omitempty"`
	TagIds           []string                 `json:"tag_ids,omitempty"`
	ContainingGroups []*GroupDescriptionInput `json:"containing_groups,omitempty"`
	SubGroups        []*GroupDescriptionInput `json:"sub_groups,omitempty"`
	// This should be a URL or a base64 encoded data URL
	FrontImage *string `json:"front_image,omitempty"`
	// This should be a URL or a base64 encoded data URL
	BackImage *string `json:"back_image,omitempty"`
}

// GroupDescription represents a relationship to a group with a description of the relationship
type GroupDescription struct {
	Group       *models.Group `json:"group"`
	Description *string       `json:"description,omitempty"`
}

type GroupDescriptionInput struct {
	GroupID     string  `json:"group_id"`
	Description *string `json:"description,omitempty"`
}

type GroupDestroyInput struct {
	ID string `json:"id"`
}

type GroupSubGroupAddInput struct {
	ContainingGroupID string                   `json:"containing_group_id"`
	SubGroups         []*GroupDescriptionInput `json:"sub_groups"`
	// The index at which to insert the sub groups. If not provided, the sub groups will be appended to the end
	InsertIndex *int `json:"insert_index,omitempty"`
}

type GroupSubGroupRemoveInput struct {
	ContainingGroupID string   `json:"containing_group_id"`
	SubGroupIds       []string `json:"sub_group_ids"`
}

type GroupUpdateInput struct {
	ID               string                   `json:"id"`
	Name             *string                  `json:"name,omitempty"`
	Aliases          *string                  `json:"aliases,omitempty"`
	Duration         *int                     `json:"duration,omitempty"`
	Date             *string                  `json:"date,omitempty"`
	Rating100        *int                     `json:"rating100,omitempty"`
	StudioID         *string                  `json:"studio_id,omitempty"`
	Director         *string                  `json:"director,omitempty"`
	Synopsis         *string                  `json:"synopsis,omitempty"`
	Urls             []string                 `json:"urls,omitempty"`
	TagIds           []string                 `json:"tag_ids,omitempty"`
	ContainingGroups []*GroupDescriptionInput `json:"containing_groups,omitempty"`
	SubGroups        []*GroupDescriptionInput `json:"sub_groups,omitempty"`
	// This should be a URL or a base64 encoded data URL
	FrontImage *string `json:"front_image,omitempty"`
	// This should be a URL or a base64 encoded data URL
	BackImage *string `json:"back_image,omitempty"`
}

type HistoryMutationResult struct {
	Count   int          `json:"count"`
	History []*time.Time `json:"history"`
}

type ImageFileType struct {
	ModTime time.Time `json:"mod_time"`
	Size    int       `json:"size"`
	Width   int       `json:"width"`
	Height  int       `json:"height"`
}

type ImagePathsType struct {
	Thumbnail *string `json:"thumbnail,omitempty"`
	Preview   *string `json:"preview,omitempty"`
	Image     *string `json:"image,omitempty"`
}

type Job struct {
	ID          string     `json:"id"`
	Status      JobStatus  `json:"status"`
	SubTasks    []string   `json:"subTasks,omitempty"`
	Description string     `json:"description"`
	Progress    *float64   `json:"progress,omitempty"`
	StartTime   *time.Time `json:"startTime,omitempty"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	AddTime     time.Time  `json:"addTime"`
	Error       *string    `json:"error,omitempty"`
}

type JobStatusUpdate struct {
	Type JobStatusUpdateType `json:"type"`
	Job  *Job                `json:"job"`
}

type LatestVersion struct {
	Version     string `json:"version"`
	Shorthash   string `json:"shorthash"`
	ReleaseDate string `json:"release_date"`
	URL         string `json:"url"`
}

type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   LogLevel  `json:"level"`
	Message string    `json:"message"`
}

type MigrateBlobsInput struct {
	DeleteOld *bool `json:"deleteOld,omitempty"`
}

type MigrateSceneScreenshotsInput struct {
	DeleteFiles       *bool `json:"deleteFiles,omitempty"`
	OverwriteExisting *bool `json:"overwriteExisting,omitempty"`
}

type MoveFilesInput struct {
	Ids []string `json:"ids"`
	// valid for single or multiple file ids
	DestinationFolder *string `json:"destination_folder,omitempty"`
	// valid for single or multiple file ids
	DestinationFolderID *string `json:"destination_folder_id,omitempty"`
	// valid only for single file id. If empty, existing basename is used
	DestinationBasename *string `json:"destination_basename,omitempty"`
}

type MovieCreateInput struct {
	Name    string  `json:"name"`
	Aliases *string `json:"aliases,omitempty"`
	// Duration in seconds
	Duration  *int     `json:"duration,omitempty"`
	Date      *string  `json:"date,omitempty"`
	Rating100 *int     `json:"rating100,omitempty"`
	StudioID  *string  `json:"studio_id,omitempty"`
	Director  *string  `json:"director,omitempty"`
	Synopsis  *string  `json:"synopsis,omitempty"`
	URL       *string  `json:"url,omitempty"`
	Urls      []string `json:"urls,omitempty"`
	TagIds    []string `json:"tag_ids,omitempty"`
	// This should be a URL or a base64 encoded data URL
	FrontImage *string `json:"front_image,omitempty"`
	// This should be a URL or a base64 encoded data URL
	BackImage *string `json:"back_image,omitempty"`
}

type MovieDestroyInput struct {
	ID string `json:"id"`
}

type MovieUpdateInput struct {
	ID        string   `json:"id"`
	Name      *string  `json:"name,omitempty"`
	Aliases   *string  `json:"aliases,omitempty"`
	Duration  *int     `json:"duration,omitempty"`
	Date      *string  `json:"date,omitempty"`
	Rating100 *int     `json:"rating100,omitempty"`
	StudioID  *string  `json:"studio_id,omitempty"`
	Director  *string  `json:"director,omitempty"`
	Synopsis  *string  `json:"synopsis,omitempty"`
	URL       *string  `json:"url,omitempty"`
	Urls      []string `json:"urls,omitempty"`
	TagIds    []string `json:"tag_ids,omitempty"`
	// This should be a URL or a base64 encoded data URL
	FrontImage *string `json:"front_image,omitempty"`
	// This should be a URL or a base64 encoded data URL
	BackImage *string `json:"back_image,omitempty"`
}

type Mutation struct {
}

type OCountDailyStatsType struct {
	Date        string `json:"date"`
	DateDisplay string `json:"date_display"`
	Count       int    `json:"count"`
}

type OCountStatsResultType struct {
	DailyStats []*OCountDailyStatsType `json:"daily_stats"`
}

type Package struct {
	PackageID string     `json:"package_id"`
	Name      string     `json:"name"`
	Version   *string    `json:"version,omitempty"`
	Date      *time.Time `json:"date,omitempty"`
	Requires  []*Package `json:"requires"`
	SourceURL string     `json:"sourceURL"`
	// The version of this package currently available from the remote source
	SourcePackage *Package       `json:"source_package,omitempty"`
	Metadata      map[string]any `json:"metadata"`
}

type PackageSourceInput struct {
	Name      *string `json:"name,omitempty"`
	URL       string  `json:"url"`
	LocalPath *string `json:"local_path,omitempty"`
}

type PerformerDestroyInput struct {
	ID string `json:"id"`
}

type PerformerProfileImageCreateInput struct {
	PerformerID string `json:"performer_id"`
	// This should be a URL or a base64 encoded data URL
	Image     string `json:"image"`
	IsPrimary *bool  `json:"is_primary,omitempty"`
	Position  *int   `json:"position,omitempty"`
}

type PerformerProfileImageDestroyInput struct {
	ID string `json:"id"`
}

type PerformerProfileImageUpdateInput struct {
	ID string `json:"id"`
	// This should be a URL or a base64 encoded data URL
	Image     *string `json:"image,omitempty"`
	IsPrimary *bool   `json:"is_primary,omitempty"`
	Position  *int    `json:"position,omitempty"`
}

type PluginPaths struct {
	Javascript []string `json:"javascript,omitempty"`
	CSS        []string `json:"css,omitempty"`
}

type PluginResult struct {
	Error  *string `json:"error,omitempty"`
	Result *string `json:"result,omitempty"`
}

// The query root for this schema
type Query struct {
}

type RemoveTempDLNAIPInput struct {
	Address string `json:"address"`
}

type ReorderSubGroupsInput struct {
	// ID of the group to reorder sub groups for
	GroupID string `json:"group_id"`
	// IDs of the sub groups to reorder. These must be a subset of the current sub groups.
	// Sub groups will be inserted in this order at the insert_index
	SubGroupIds []string `json:"sub_group_ids"`
	// The sub-group ID at which to insert the sub groups
	InsertAtID string `json:"insert_at_id"`
	// If true, the sub groups will be inserted after the insert_index, otherwise they will be inserted before
	InsertAfter *bool `json:"insert_after,omitempty"`
}

type SQLExecResult struct {
	// The number of rows affected by the query, usually an UPDATE, INSERT, or DELETE.
	// Not all queries or databases support this feature.
	RowsAffected *int64 `json:"rows_affected,omitempty"`
	// The integer generated by the database in response to a command.
	// Typically this will be from an "auto increment" column when inserting a new row.
	// Not all databases support this feature, and the syntax of such statements varies.
	LastInsertID *int64 `json:"last_insert_id,omitempty"`
}

type SQLQueryResult struct {
	// The column names, in the order they appear in the result set.
	Columns []string `json:"columns"`
	// The returned rows.
	Rows [][]any `json:"rows"`
}

type SaveFilterInput struct {
	// provide ID to overwrite existing filter
	ID           *string                `json:"id,omitempty"`
	Mode         models.FilterMode      `json:"mode"`
	Name         string                 `json:"name"`
	FindFilter   *models.FindFilterType `json:"find_filter,omitempty"`
	ObjectFilter map[string]any         `json:"object_filter,omitempty"`
	UIOptions    map[string]any         `json:"ui_options,omitempty"`
}

type SceneGroup struct {
	Group      *models.Group `json:"group"`
	SceneIndex *int          `json:"scene_index,omitempty"`
}

type SceneHashInput struct {
	Checksum *string `json:"checksum,omitempty"`
	Oshash   *string `json:"oshash,omitempty"`
}

type SceneMarkerCreateInput struct {
	Title string `json:"title"`
	// The required start time of the marker (in seconds). Supports decimals.
	Seconds float64 `json:"seconds"`
	// The optional end time of the marker (in seconds). Supports decimals.
	EndSeconds   *float64 `json:"end_seconds,omitempty"`
	SceneID      string   `json:"scene_id"`
	PrimaryTagID string   `json:"primary_tag_id"`
	TagIds       []string `json:"tag_ids,omitempty"`
}

type SceneMarkerTag struct {
	Tag          *models.Tag           `json:"tag"`
	SceneMarkers []*models.SceneMarker `json:"scene_markers"`
}

type SceneMarkerUpdateInput struct {
	ID    string  `json:"id"`
	Title *string `json:"title,omitempty"`
	// The start time of the marker (in seconds). Supports decimals.
	Seconds *float64 `json:"seconds,omitempty"`
	// The end time of the marker (in seconds). Supports decimals.
	EndSeconds   *float64 `json:"end_seconds,omitempty"`
	SceneID      *string  `json:"scene_id,omitempty"`
	PrimaryTagID *string  `json:"primary_tag_id,omitempty"`
	TagIds       []string `json:"tag_ids,omitempty"`
}

type SceneMergeInput struct {
	// If destination scene has no files, then the primary file of the
	// first source scene will be assigned as primary
	Source      []string                 `json:"source"`
	Destination string                   `json:"destination"`
	Values      *models.SceneUpdateInput `json:"values,omitempty"`
	PlayHistory *bool                    `json:"play_history,omitempty"`
	OHistory    *bool                    `json:"o_history,omitempty"`
}

type SceneMovie struct {
	Movie      *models.Group `json:"movie"`
	SceneIndex *int          `json:"scene_index,omitempty"`
}

type SceneParserResultType struct {
	Count   int                         `json:"count"`
	Results []*models.SceneParserResult `json:"results"`
}

type ScenePathsType struct {
	Screenshot         *string `json:"screenshot,omitempty"`
	Preview            *string `json:"preview,omitempty"`
	Stream             *string `json:"stream,omitempty"`
	Webp               *string `json:"webp,omitempty"`
	Vtt                *string `json:"vtt,omitempty"`
	Sprite             *string `json:"sprite,omitempty"`
	Funscript          *string `json:"funscript,omitempty"`
	InteractiveHeatmap *string `json:"interactive_heatmap,omitempty"`
	Caption            *string `json:"caption,omitempty"`
}

type ScenePerformer struct {
	Performer       *models.Performer `json:"performer"`
	SmallRole       bool              `json:"small_role"`
	RoleDescription *string           `json:"role_description,omitempty"`
}

type ScenePerformerInput struct {
	PerformerID     string  `json:"performer_id"`
	SmallRole       bool    `json:"small_role"`
	RoleDescription *string `json:"role_description,omitempty"`
}

type SceneSaveFilteredScreenshotInput struct {
	ID    string   `json:"id"`
	Image string   `json:"image"`
	At    *float64 `json:"at,omitempty"`
}

type ScrapeMultiPerformersInput struct {
	// Instructs to query by scene fingerprints
	PerformerIds []string `json:"performer_ids,omitempty"`
}

type ScrapeMultiScenesInput struct {
	// Instructs to query by scene fingerprints
	SceneIds []string `json:"scene_ids,omitempty"`
}

type ScrapeSingleGalleryInput struct {
	// Instructs to query by string
	Query *string `json:"query,omitempty"`
	// Instructs to query by gallery id
	GalleryID *string `json:"gallery_id,omitempty"`
	// Instructs to query by gallery fragment
	GalleryInput *models.ScrapedGalleryInput `json:"gallery_input,omitempty"`
}

type ScrapeSingleGroupInput struct {
	// Instructs to query by string
	Query *string `json:"query,omitempty"`
	// Instructs to query by group id
	GroupID *string `json:"group_id,omitempty"`
	// Instructs to query by group fragment
	GroupInput *ScrapedGroupInput `json:"group_input,omitempty"`
}

type ScrapeSingleImageInput struct {
	// Instructs to query by string
	Query *string `json:"query,omitempty"`
	// Instructs to query by image id
	ImageID *string `json:"image_id,omitempty"`
	// Instructs to query by image fragment
	ImageInput *models.ScrapedImageInput `json:"image_input,omitempty"`
}

type ScrapeSingleMovieInput struct {
	// Instructs to query by string
	Query *string `json:"query,omitempty"`
	// Instructs to query by movie id
	MovieID *string `json:"movie_id,omitempty"`
	// Instructs to query by movie fragment
	MovieInput *scraper.ScrapedMovieInput `json:"movie_input,omitempty"`
}

type ScrapeSinglePerformerInput struct {
	// Instructs to query by string
	Query *string `json:"query,omitempty"`
	// Instructs to query by performer id
	PerformerID *string `json:"performer_id,omitempty"`
	// Instructs to query by performer fragment
	PerformerInput *scraper.ScrapedPerformerInput `json:"performer_input,omitempty"`
}

type ScrapeSingleSceneInput struct {
	// Instructs to query by string
	Query *string `json:"query,omitempty"`
	// Instructs to query by scene fingerprints
	SceneID *string `json:"scene_id,omitempty"`
	// Instructs to query by scene fragment
	SceneInput *models.ScrapedSceneInput `json:"scene_input,omitempty"`
}

type ScrapeSingleStudioInput struct {
	// Query can be either a name or a Stash ID
	Query *string `json:"query,omitempty"`
}

type ScrapedGroupInput struct {
	Name     *string  `json:"name,omitempty"`
	Aliases  *string  `json:"aliases,omitempty"`
	Duration *string  `json:"duration,omitempty"`
	Date     *string  `json:"date,omitempty"`
	Rating   *string  `json:"rating,omitempty"`
	Director *string  `json:"director,omitempty"`
	Urls     []string `json:"urls,omitempty"`
	Synopsis *string  `json:"synopsis,omitempty"`
}

type SetDefaultFilterInput struct {
	Mode models.FilterMode `json:"mode"`
	// null to clear
	FindFilter   *models.FindFilterType `json:"find_filter,omitempty"`
	ObjectFilter map[string]any         `json:"object_filter,omitempty"`
	UIOptions    map[string]any         `json:"ui_options,omitempty"`
}

type SetFingerprintsInput struct {
	Type string `json:"type"`
	// an null value will remove the fingerprint
	Value *string `json:"value,omitempty"`
}

type StashBoxDraftSubmissionInput struct {
	ID               string  `json:"id"`
	StashBoxIndex    *int    `json:"stash_box_index,omitempty"`
	StashBoxEndpoint *string `json:"stash_box_endpoint,omitempty"`
}

type StashBoxFingerprintSubmissionInput struct {
	SceneIds         []string `json:"scene_ids"`
	StashBoxIndex    *int     `json:"stash_box_index,omitempty"`
	StashBoxEndpoint *string  `json:"stash_box_endpoint,omitempty"`
}

type StashBoxPerformerQueryInput struct {
	// Index of the configured stash-box instance to use
	StashBoxIndex *int `json:"stash_box_index,omitempty"`
	// Endpoint of the stash-box instance to use
	StashBoxEndpoint *string `json:"stash_box_endpoint,omitempty"`
	// Instructs query by scene fingerprints
	PerformerIds []string `json:"performer_ids,omitempty"`
	// Query by query string
	Q *string `json:"q,omitempty"`
}

type StashBoxPerformerQueryResult struct {
	Query   string                     `json:"query"`
	Results []*models.ScrapedPerformer `json:"results"`
}

type StashBoxSceneQueryInput struct {
	// Index of the configured stash-box instance to use
	StashBoxIndex *int `json:"stash_box_index,omitempty"`
	// Endpoint of the stash-box instance to use
	StashBoxEndpoint *string `json:"stash_box_endpoint,omitempty"`
	// Instructs query by scene fingerprints
	SceneIds []string `json:"scene_ids,omitempty"`
	// Query by query string
	Q *string `json:"q,omitempty"`
}

type StashBoxValidationResult struct {
	Valid  bool   `json:"valid"`
	Status string `json:"status"`
}

type StatsResultType struct {
	SceneCount        int     `json:"scene_count"`
	ScenesSize        float64 `json:"scenes_size"`
	ScenesDuration    float64 `json:"scenes_duration"`
	ImageCount        int     `json:"image_count"`
	ImagesSize        float64 `json:"images_size"`
	GalleryCount      int     `json:"gallery_count"`
	PerformerCount    int     `json:"performer_count"`
	StudioCount       int     `json:"studio_count"`
	GroupCount        int     `json:"group_count"`
	MovieCount        int     `json:"movie_count"`
	TagCount          int     `json:"tag_count"`
	TotalOCount       int     `json:"total_o_count"`
	TotalOmgCount     int     `json:"total_omg_count"`
	TotalPlayDuration float64 `json:"total_play_duration"`
	TotalPlayCount    int     `json:"total_play_count"`
	ScenesPlayed      int     `json:"scenes_played"`
}

type StudioDestroyInput struct {
	ID string `json:"id"`
}

type Subscription struct {
}

type TagCreateInput struct {
	Name string `json:"name"`
	// Value that does not appear in the UI but overrides name for sorting
	SortName          *string  `json:"sort_name,omitempty"`
	Description       *string  `json:"description,omitempty"`
	Aliases           []string `json:"aliases,omitempty"`
	IgnoreAutoTag     *bool    `json:"ignore_auto_tag,omitempty"`
	IsPoseTag         *bool    `json:"is_pose_tag,omitempty"`
	IgnoreSuggestions *bool    `json:"ignore_suggestions,omitempty"`
	Favorite          *bool    `json:"favorite,omitempty"`
	Weight            *float64 `json:"weight,omitempty"`
	Color             *string  `json:"color,omitempty"`
	// This should be a URL or a base64 encoded data URL
	Image     *string  `json:"image,omitempty"`
	ParentIds []string `json:"parent_ids,omitempty"`
	ChildIds  []string `json:"child_ids,omitempty"`
}

type TagDestroyInput struct {
	ID string `json:"id"`
}

type TagUpdateInput struct {
	ID   string  `json:"id"`
	Name *string `json:"name,omitempty"`
	// Value that does not appear in the UI but overrides name for sorting
	SortName          *string  `json:"sort_name,omitempty"`
	Description       *string  `json:"description,omitempty"`
	Aliases           []string `json:"aliases,omitempty"`
	IgnoreAutoTag     *bool    `json:"ignore_auto_tag,omitempty"`
	IsPoseTag         *bool    `json:"is_pose_tag,omitempty"`
	IgnoreSuggestions *bool    `json:"ignore_suggestions,omitempty"`
	Favorite          *bool    `json:"favorite,omitempty"`
	Weight            *float64 `json:"weight,omitempty"`
	Color             *string  `json:"color,omitempty"`
	// This should be a URL or a base64 encoded data URL
	Image     *string  `json:"image,omitempty"`
	ParentIds []string `json:"parent_ids,omitempty"`
	ChildIds  []string `json:"child_ids,omitempty"`
}

type TagsMergeInput struct {
	Source      []string `json:"source"`
	Destination string   `json:"destination"`
}

type Version struct {
	Version   *string `json:"version,omitempty"`
	Hash      string  `json:"hash"`
	BuildTime string  `json:"build_time"`
}

type VideoFiltersInput struct {
	Contrast     *int `json:"contrast,omitempty"`
	Brightness   *int `json:"brightness,omitempty"`
	Gamma        *int `json:"gamma,omitempty"`
	Saturate     *int `json:"saturate,omitempty"`
	HueRotate    *int `json:"hue_rotate,omitempty"`
	WhiteBalance *int `json:"white_balance,omitempty"`
	Red          *int `json:"red,omitempty"`
	Green        *int `json:"green,omitempty"`
	Blue         *int `json:"blue,omitempty"`
	Blur         *int `json:"blur,omitempty"`
}

type VideoTransformsInput struct {
	Rotate      *int `json:"rotate,omitempty"`
	Scale       *int `json:"scale,omitempty"`
	AspectRatio *int `json:"aspect_ratio,omitempty"`
}

type ViewHistoryEntry struct {
	Scene    *models.Scene   `json:"scene,omitempty"`
	Gallery  *models.Gallery `json:"gallery,omitempty"`
	ViewDate time.Time       `json:"viewDate"`
	// Время o-count если он был в близком промежутке времени (в пределах 5 минут после просмотра)
	ODate *time.Time `json:"oDate,omitempty"`
	// Время omg-count если он был в близком промежутке времени (в пределах 5 минут после просмотра)
	OmgDate *time.Time `json:"omgDate,omitempty"`
	// Количество просмотров в группе (для сгруппированных подряд идущих просмотров)
	ViewCount *int `json:"viewCount,omitempty"`
}

type ViewHistoryFilter struct {
	// Фильтр по дате просмотра
	ViewDate *models.TimestampCriterionInput `json:"viewDate,omitempty"`
	// Фильтр по исполнителям
	Performers *models.MultiCriterionInput `json:"performers,omitempty"`
	// Фильтр по тегам
	Tags *models.HierarchicalMultiCriterionInput `json:"tags,omitempty"`
	// Фильтр по студиям
	Studios *models.HierarchicalMultiCriterionInput `json:"studios,omitempty"`
}

type ViewHistoryResult struct {
	Count         int                 `json:"count"`
	Items         []*ViewHistoryEntry `json:"items"`
	TotalOCount   int                 `json:"totalOCount"`
	TotalOMGCount int                 `json:"totalOMGCount"`
}

type JobStatus string

const (
	JobStatusReady     JobStatus = "READY"
	JobStatusRunning   JobStatus = "RUNNING"
	JobStatusFinished  JobStatus = "FINISHED"
	JobStatusStopping  JobStatus = "STOPPING"
	JobStatusCancelled JobStatus = "CANCELLED"
	JobStatusFailed    JobStatus = "FAILED"
)

var AllJobStatus = []JobStatus{
	JobStatusReady,
	JobStatusRunning,
	JobStatusFinished,
	JobStatusStopping,
	JobStatusCancelled,
	JobStatusFailed,
}

func (e JobStatus) IsValid() bool {
	switch e {
	case JobStatusReady, JobStatusRunning, JobStatusFinished, JobStatusStopping, JobStatusCancelled, JobStatusFailed:
		return true
	}
	return false
}

func (e JobStatus) String() string {
	return string(e)
}

func (e *JobStatus) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	*e = JobStatus(s)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid JobStatus", s)
	}
	return nil
}

func (e JobStatus) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(string(e))), nil
}

type JobStatusUpdateType string

const (
	JobStatusUpdateTypeAdd    JobStatusUpdateType = "ADD"
	JobStatusUpdateTypeRemove JobStatusUpdateType = "REMOVE"
	JobStatusUpdateTypeUpdate JobStatusUpdateType = "UPDATE"
)

var AllJobStatusUpdateType = []JobStatusUpdateType{
	JobStatusUpdateTypeAdd,
	JobStatusUpdateTypeRemove,
	JobStatusUpdateTypeUpdate,
}

func (e JobStatusUpdateType) IsValid() bool {
	switch e {
	case JobStatusUpdateTypeAdd, JobStatusUpdateTypeRemove, JobStatusUpdateTypeUpdate:
		return true
	}
	return false
}

func (e JobStatusUpdateType) String() string {
	return string(e)
}

func (e *JobStatusUpdateType) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	*e = JobStatusUpdateType(s)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid JobStatusUpdateType", s)
	}
	return nil
}

func (e JobStatusUpdateType) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(string(e))), nil
}

type LogLevel string

const (
	LogLevelTrace    LogLevel = "Trace"
	LogLevelDebug    LogLevel = "Debug"
	LogLevelInfo     LogLevel = "Info"
	LogLevelProgress LogLevel = "Progress"
	LogLevelWarning  LogLevel = "Warning"
	LogLevelError    LogLevel = "Error"
)

var AllLogLevel = []LogLevel{
	LogLevelTrace,
	LogLevelDebug,
	LogLevelInfo,
	LogLevelProgress,
	LogLevelWarning,
	LogLevelError,
}

func (e LogLevel) IsValid() bool {
	switch e {
	case LogLevelTrace, LogLevelDebug, LogLevelInfo, LogLevelProgress, LogLevelWarning, LogLevelError:
		return true
	}
	return false
}

func (e LogLevel) String() string {
	return string(e)
}

func (e *LogLevel) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	*e = LogLevel(s)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid LogLevel", s)
	}
	return nil
}

func (e LogLevel) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(string(e))), nil
}

type PackageType string

const (
	PackageTypeScraper PackageType = "Scraper"
	PackageTypePlugin  PackageType = "Plugin"
)

var AllPackageType = []PackageType{
	PackageTypeScraper,
	PackageTypePlugin,
}

func (e PackageType) IsValid() bool {
	switch e {
	case PackageTypeScraper, PackageTypePlugin:
		return true
	}
	return false
}

func (e PackageType) String() string {
	return string(e)
}

func (e *PackageType) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	*e = PackageType(s)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid PackageType", s)
	}
	return nil
}

func (e PackageType) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(string(e))), nil
}
