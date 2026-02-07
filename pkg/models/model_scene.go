package models

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"time"
)

// Scene stores the metadata for a single video scene.
type Scene struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Code      string `json:"code"`
	Details   string `json:"details"`
	Director  string `json:"director"`
	Date      *Date  `json:"date"`       // Date of release
	ShootDate *Date  `json:"shoot_date"` // Date of filming/shooting
	// Rating expressed in 1-100 scale
	Rating                  *int    `json:"rating100"`
	Organized               bool    `json:"organized"`
	Pinned                  bool    `json:"pinned"`
	IsBroken                bool    `json:"is_broken"`
	IsNotBroken             bool    `json:"is_not_broken"`
	AudioOffsetMs           int     `json:"audio_offset_ms"`
	AudioPlaybackSpeed      float64 `json:"audio_playback_speed"`
	ForceHLS                bool    `json:"force_hls"`
	DisableNextSceneOverlay bool    `json:"disable_next_scene_overlay"`
	StudioID                *int    `json:"studio_id"`

	// transient - not persisted
	Files         RelatedVideoFiles
	PrimaryFileID *FileID
	// transient - path of primary file - empty if no files
	Path string
	// transient - oshash of primary file - empty if no files
	OSHash string
	// transient - checksum of primary file - empty if no files
	Checksum string

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ResumeTime   float64 `json:"resume_time"`
	PlayDuration float64 `json:"play_duration"`

	// Video trimming fields
	StartTime *float64 `json:"start_time"`
	EndTime   *float64 `json:"end_time"`

	// Video filters and transformations
	VideoFilters    *VideoFilters    `json:"video_filters"`
	VideoTransforms *VideoTransforms `json:"video_transforms"`

	URLs            RelatedStrings         `json:"urls"`
	GalleryIDs      RelatedIDs             `json:"gallery_ids"`
	TagIDs          RelatedIDs             `json:"tag_ids"`
	PerformerIDs    RelatedIDs             `json:"performer_ids"`
	ScenePerformers RelatedScenePerformers `json:"scene_performers"`
	PerformerTagIDs RelatedPerformerTags   `json:"performer_tag_ids"`
	Groups          RelatedGroups          `json:"groups"`
	StashIDs        RelatedStashIDs        `json:"stash_ids"`
}

func NewScene() Scene {
	currentTime := time.Now()
	return Scene{
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}
}

// ScenePartial represents part of a Scene object. It is used to update
// the database entry.
type ScenePartial struct {
	Title     OptionalString
	Code      OptionalString
	Details   OptionalString
	Director  OptionalString
	Date      OptionalDate
	ShootDate OptionalDate
	// Rating expressed in 1-100 scale
	Rating                  OptionalInt
	Organized               OptionalBool
	Pinned                  OptionalBool
	IsBroken                OptionalBool
	IsNotBroken             OptionalBool
	AudioOffsetMs           OptionalInt
	AudioPlaybackSpeed      OptionalFloat64
	ForceHLS                OptionalBool
	DisableNextSceneOverlay OptionalBool
	StudioID                OptionalInt
	CreatedAt               OptionalTime
	UpdatedAt               OptionalTime
	ResumeTime              OptionalFloat64
	PlayDuration            OptionalFloat64
	StartTime               OptionalFloat64
	EndTime                 OptionalFloat64

	VideoFilters    *VideoFilters
	VideoTransforms *VideoTransforms

	URLs            *UpdateStrings
	GalleryIDs      *UpdateIDs
	TagIDs          *UpdateIDs
	PerformerIDs    *UpdateIDs
	ScenePerformers *UpdateScenePerformers
	PerformerTagIDs *UpdatePerformerTags
	GroupIDs        *UpdateGroupIDs
	StashIDs        *UpdateStashIDs
	PrimaryFileID   *FileID
}

func NewScenePartial() ScenePartial {
	currentTime := time.Now()
	return ScenePartial{
		UpdatedAt: NewOptionalTime(currentTime),
	}
}

func (s *Scene) LoadURLs(ctx context.Context, l URLLoader) error {
	if s.URLs.Loaded() {
		return nil
	}
	return s.URLs.load(func() ([]string, error) {
		return l.GetURLs(ctx, s.ID)
	})
}

func (s *Scene) LoadFiles(ctx context.Context, l VideoFileLoader) error {
	if s.Files.Loaded() {
		return nil
	}
	return s.Files.load(func() ([]*VideoFile, error) {
		return l.GetFiles(ctx, s.ID)
	})
}

func (s *Scene) LoadPrimaryFile(ctx context.Context, l FileGetter) error {
	return s.Files.loadPrimary(func() (*VideoFile, error) {
		if s.PrimaryFileID == nil {
			return nil, nil
		}

		f, err := l.Find(ctx, *s.PrimaryFileID)
		if err != nil {
			return nil, err
		}

		var vf *VideoFile
		if len(f) > 0 {
			var ok bool
			vf, ok = f[0].(*VideoFile)
			if !ok {
				return nil, errors.New("not a video file")
			}
		}
		return vf, nil
	})
}

func (s *Scene) LoadGalleryIDs(ctx context.Context, l GalleryIDLoader) error {
	if s.GalleryIDs.Loaded() {
		return nil
	}
	return s.GalleryIDs.load(func() ([]int, error) {
		return l.GetGalleryIDs(ctx, s.ID)
	})
}

func (s *Scene) LoadPerformerIDs(ctx context.Context, l PerformerIDLoader) error {
	if s.PerformerIDs.Loaded() {
		return nil
	}
	return s.PerformerIDs.load(func() ([]int, error) {
		return l.GetPerformerIDs(ctx, s.ID)
	})
}

func (s *Scene) LoadScenePerformers(ctx context.Context, l ScenePerformerLoader) error {
	if s.ScenePerformers.Loaded() {
		return nil
	}
	return s.ScenePerformers.load(func() ([]PerformerScenes, error) {
		return l.GetScenePerformers(ctx, s.ID)
	})
}

func (s *Scene) LoadTagIDs(ctx context.Context, l TagIDLoader) error {
	if s.TagIDs.Loaded() {
		return nil
	}
	return s.TagIDs.load(func() ([]int, error) {
		ids, err := l.GetTagIDs(ctx, s.ID)
		return ids, err
	})
}

func (s *Scene) LoadGroups(ctx context.Context, l SceneGroupLoader) error {
	if s.Groups.Loaded() {
		return nil
	}
	return s.Groups.load(func() ([]GroupsScenes, error) {
		return l.GetGroups(ctx, s.ID)
	})
}

func (s *Scene) LoadStashIDs(ctx context.Context, l StashIDLoader) error {
	if s.StashIDs.Loaded() {
		return nil
	}
	return s.StashIDs.load(func() ([]StashID, error) {
		return l.GetStashIDs(ctx, s.ID)
	})
}

func (s *Scene) LoadPerformerTagIDs(ctx context.Context, l PerformerTagIDLoader) error {
	if s.PerformerTagIDs.Loaded() {
		return nil
	}
	return s.PerformerTagIDs.load(func() ([]ScenesTagsPerformer, error) {
		return l.GetPerformerTagIDs(ctx, s.ID)
	})
}

func (s *Scene) LoadRelationships(ctx context.Context, l SceneReader) error {
	if err := s.LoadURLs(ctx, l); err != nil {
		return err
	}

	if err := s.LoadGalleryIDs(ctx, l); err != nil {
		return err
	}

	if err := s.LoadPerformerIDs(ctx, l); err != nil {
		return err
	}

	if err := s.LoadTagIDs(ctx, l); err != nil {
		return err
	}

	if err := s.LoadGroups(ctx, l); err != nil {
		return err
	}

	if err := s.LoadStashIDs(ctx, l); err != nil {
		return err
	}

	if err := s.LoadPerformerTagIDs(ctx, l); err != nil {
		return err
	}

	if err := s.LoadScenePerformers(ctx, l); err != nil {
		return err
	}

	if err := s.LoadFiles(ctx, l); err != nil {
		return err
	}

	return nil
}

// UpdateInput constructs a SceneUpdateInput using the populated fields in the ScenePartial object.
func (s ScenePartial) UpdateInput(id int) SceneUpdateInput {
	var dateStr *string
	if s.Date.Set {
		d := s.Date.Value
		v := d.String()
		dateStr = &v
	}

	var stashIDs StashIDs
	if s.StashIDs != nil {
		stashIDs = StashIDs(s.StashIDs.StashIDs)
	}

	ret := SceneUpdateInput{
		ID:           FlexibleID(strconv.Itoa(id)),
		Title:        s.Title.Ptr(),
		Code:         s.Code.Ptr(),
		Details:      s.Details.Ptr(),
		Director:     s.Director.Ptr(),
		Urls:         s.URLs.Strings(),
		Date:         dateStr,
		Rating100:    s.Rating.Ptr(),
		Organized:    s.Organized.Ptr(),
		IsBroken:     s.IsBroken.Ptr(),
		StudioID:     s.StudioID.StringPtr(),
		GalleryIds:   s.GalleryIDs.IDStrings(),
		PerformerIds: s.PerformerIDs.IDStrings(),
		Movies:       s.GroupIDs.SceneMovieInputs(),
		TagIds:       s.TagIDs.IDStrings(),
		StashIds:     stashIDs.ToStashIDInputs(),
	}

	return ret
}

// GetTitle returns the title of the scene. If the Title field is empty,
// then the base filename is returned.
func (s Scene) GetTitle() string {
	if s.Title != "" {
		return s.Title
	}

	return filepath.Base(s.Path)
}

// DisplayName returns a display name for the scene for logging purposes.
// It returns Path if not empty, otherwise it returns the ID.
func (s Scene) DisplayName() string {
	if s.Path != "" {
		return s.Path
	}

	return strconv.Itoa(s.ID)
}

// GetHash returns the hash of the scene, based on the hash algorithm provided. If
// hash algorithm is MD5, then Checksum is returned. Otherwise, OSHash is returned.
func (s Scene) GetHash(hashAlgorithm HashAlgorithm) string {
	switch hashAlgorithm {
	case HashAlgorithmMd5:
		return s.Checksum
	case HashAlgorithmOshash:
		return s.OSHash
	}

	return ""
}

// SceneFileType represents the file metadata for a scene.
type SceneFileType struct {
	Size       *string  `json:"size"`
	Duration   *float64 `json:"duration"`
	VideoCodec *string  `json:"video_codec"`
	AudioCodec *string  `json:"audio_codec"`
	Width      *int     `json:"width"`
	Height     *int     `json:"height"`
	Framerate  *float64 `json:"framerate"`
	Bitrate    *int     `json:"bitrate"`
}

type VideoCaption struct {
	LanguageCode string `json:"language_code"`
	Filename     string `json:"filename"`
	CaptionType  string `json:"caption_type"`
}

func (c VideoCaption) Path(filePath string) string {
	return filepath.Join(filepath.Dir(filePath), c.Filename)
}

// SimilarScene represents a scene with its similarity score
type SimilarScene struct {
	Scene               *Scene               `json:"scene"`
	SimilarityScore     float64              `json:"similarity_score"`
	SimilarityScoreData *SimilarityScoreData `json:"similarity_score_data,omitempty"`
}

// VideoFilters represents video filter settings for a scene
type VideoFilters struct {
	Contrast     *int `json:"contrast"`
	Brightness   *int `json:"brightness"`
	Gamma        *int `json:"gamma"`
	Saturate     *int `json:"saturate"`
	HueRotate    *int `json:"hue_rotate"`
	WhiteBalance *int `json:"white_balance"`
	Red          *int `json:"red"`
	Green        *int `json:"green"`
	Blue         *int `json:"blue"`
	Blur         *int `json:"blur"`
}

// VideoTransforms represents video transformation settings for a scene
type VideoTransforms struct {
	Rotate      *int `json:"rotate"`
	Scale       *int `json:"scale"`
	AspectRatio *int `json:"aspect_ratio"`
}
