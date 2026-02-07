package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

// ScenePaths contains computed URL paths for scene assets.
type ScenePaths struct {
	Screenshot        string `json:"screenshot"`
	Preview           string `json:"preview"`
	Stream            string `json:"stream"`
	Webp              string `json:"webp"`
	Vtt               string `json:"vtt"`
	Sprite            string `json:"sprite"`
	Funscript         string `json:"funscript"`
	InteractiveHeatmap string `json:"interactive_heatmap"`
	Caption           string `json:"caption"`
}

// ImagePaths contains computed URL paths for image assets.
type ImagePaths struct {
	Thumbnail string `json:"thumbnail"`
	Preview   string `json:"preview"`
	Image     string `json:"image"`
}

// GalleryPaths contains computed URL paths for gallery assets.
type GalleryPaths struct {
	Cover   string `json:"cover"`
	Preview string `json:"preview"`
}

// PerformerPaths contains computed URL paths for performer assets.
type PerformerPaths struct {
	Image string `json:"image"`
}

// StudioPaths contains computed URL paths for studio assets.
type StudioPaths struct {
	Image string `json:"image"`
}

// TagPaths contains computed URL paths for tag assets.
type TagPaths struct {
	Image string `json:"image"`
}

// GroupPaths contains computed URL paths for group assets.
type GroupPaths struct {
	FrontImage string `json:"front_image"`
	BackImage  string `json:"back_image"`
}

// MinimalTag is a minimal representation of a Tag (with image path and description for cards).
type MinimalTag struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color,omitempty"`
	Description string `json:"description,omitempty"`
	ImagePath   string `json:"image_path,omitempty"`
}

// MinimalPerformer is a minimal representation of a Performer for scene cards and lists.
type MinimalPerformer struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Disambiguation string  `json:"disambiguation,omitempty"`
	Gender         string  `json:"gender,omitempty"`
	Favorite       bool    `json:"favorite"`
	ImagePath      string  `json:"image_path,omitempty"`
	Birthdate      *string `json:"birthdate,omitempty"`
	DeathDate      *string `json:"death_date,omitempty"`
	Country        string  `json:"country,omitempty"`
	Rating100      *int    `json:"rating100,omitempty"`
	// Optional: primary_tag for card display (id + name)
	PrimaryTag *MinimalTag `json:"primary_tag,omitempty"`
}

// MinimalStudio is a minimal representation of a Studio.
type MinimalStudio struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ImagePath string `json:"image_path,omitempty"`
}

// MinimalGallery is a minimal representation of a Gallery.
type MinimalGallery struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// MinimalGroup is a minimal representation of a Group for scene.groups.
type MinimalGroup struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	FrontImage string `json:"front_image_path,omitempty"`
}

// SceneGroupEntry represents a group entry in scene response.
type SceneGroupEntry struct {
	Group      MinimalGroup `json:"group"`
	SceneIndex *int         `json:"scene_index,omitempty"`
}

// ScenePerformerEntry is the frontend-facing scene_performers item: nested performer + small_role + role_description.
type ScenePerformerEntry struct {
	Performer       MinimalPerformer `json:"performer"`
	SmallRole       bool             `json:"small_role"`
	RoleDescription *string          `json:"role_description,omitempty"`
}

// MinimalSceneMarker is a minimal representation of a SceneMarker.
type MinimalSceneMarker struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Seconds    float64    `json:"seconds"`
	EndSeconds *float64   `json:"end_seconds,omitempty"`
	PrimaryTag MinimalTag `json:"primary_tag"`
	Tags       []MinimalTag `json:"tags"`
	Stream     string     `json:"stream"`
	Preview    string     `json:"preview"`
	Screenshot string     `json:"screenshot"`
}

// EnrichedScene extends a Scene with resolved relations.
// Override all Related* fields from Scene to ensure proper JSON serialization
// (the embedded Scene's Related* fields serialize as objects, not arrays).
type EnrichedScene struct {
	*models.Scene

	// Resolved relations (full objects, not just IDs)
	Tags         []MinimalTag          `json:"tags"`
	Performers   []MinimalPerformer    `json:"performers"`
	Galleries    []MinimalGallery      `json:"galleries"`
	Groups       []SceneGroupEntry     `json:"groups"`
	SceneMarkers []MinimalSceneMarker  `json:"scene_markers"`
	Studio       *MinimalStudio        `json:"studio,omitempty"`
	Paths        *ScenePaths           `json:"paths"`
	FileObjs     interface{}           `json:"files"`
	StashIDArr   interface{}           `json:"stash_ids"`

	// Override Related* fields from embedded Scene to serialize as proper arrays
	URLs            []string              `json:"urls"`
	TagIDList       []int                 `json:"tag_ids"`
	PerformerIDList []int                 `json:"performer_ids"`
	GalleryIDList   []int                 `json:"gallery_ids"`
	ScenePerformersList []ScenePerformerEntry `json:"scene_performers"`
	PerformerTagIDList  interface{}       `json:"performer_tag_ids"`
}

// EnrichedImage extends an Image with resolved relations.
type EnrichedImage struct {
	*models.Image

	Tags        []MinimalTag       `json:"tags"`
	Performers  []MinimalPerformer `json:"performers"`
	Galleries   []MinimalGallery   `json:"galleries"`
	Studio      *MinimalStudio     `json:"studio,omitempty"`
	Paths       *ImagePaths        `json:"paths"`
	VisualFiles interface{}        `json:"visual_files"`
}

// EnrichedGallery extends a Gallery with resolved relations.
type EnrichedGallery struct {
	*models.Gallery

	Tags       []MinimalTag       `json:"tags"`
	Performers []MinimalPerformer `json:"performers"`
	Scenes     []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"scenes"`
	Studio *MinimalStudio `json:"studio,omitempty"`
	Paths  *GalleryPaths  `json:"paths"`
	Cover  interface{}    `json:"cover"`
}

func scenePaths(sceneID int) *ScenePaths {
	id := strconv.Itoa(sceneID)
	base := "/scene/" + id
	return &ScenePaths{
		Screenshot:        base + "/screenshot",
		Preview:           base + "/preview",
		Stream:            base + "/stream",
		Webp:              base + "/webp",
		Vtt:               base + "/vtt/thumbs",
		Sprite:            base + "/vtt/sprite",
		Funscript:         base + "/funscript",
		InteractiveHeatmap: base + "/interactive_heatmap",
		Caption:           base + "/caption",
	}
}

func imagePaths(imageID int) *ImagePaths {
	id := strconv.Itoa(imageID)
	base := "/image/" + id
	return &ImagePaths{
		Thumbnail: base + "/thumbnail",
		Preview:   base + "/preview",
		Image:     base + "/image",
	}
}

func galleryPaths(galleryID int) *GalleryPaths {
	id := strconv.Itoa(galleryID)
	base := "/gallery/" + id
	return &GalleryPaths{
		Cover:   base + "/cover",
		Preview: base + "/preview/0",
	}
}

// enrichScenes resolves tags, performers, groups, galleries, files, studio, and paths for a list of scenes.
func (h *RESTHandler) enrichScenes(ctx context.Context, scenes []*models.Scene) ([]EnrichedScene, error) {
	if len(scenes) == 0 {
		return []EnrichedScene{}, nil
	}

	// Collect all unique IDs needed
	tagIDSet := make(map[int]bool)
	performerIDSet := make(map[int]bool)
	galleryIDSet := make(map[int]bool)
	groupIDSet := make(map[int]bool)
	studioIDSet := make(map[int]bool)

	for _, s := range scenes {
		for _, id := range s.TagIDs.List() {
			tagIDSet[id] = true
		}
		for _, id := range s.PerformerIDs.List() {
			performerIDSet[id] = true
		}
		for _, id := range s.GalleryIDs.List() {
			galleryIDSet[id] = true
		}
		for _, g := range s.Groups.List() {
			groupIDSet[g.GroupID] = true
		}
		if s.StudioID != nil {
			studioIDSet[*s.StudioID] = true
		}
	}

	// Batch load tags
	tagMap := make(map[int]MinimalTag)
	if len(tagIDSet) > 0 {
		tagIDs := mapKeys(tagIDSet)
		tags, err := h.repository.Tag.FindMany(ctx, tagIDs)
		if err != nil {
			return nil, fmt.Errorf("loading tags: %w", err)
		}
		for _, t := range tags {
			idStr := strconv.Itoa(t.ID)
			tagMap[t.ID] = MinimalTag{
				ID:          idStr,
				Name:        t.Name,
				Color:       t.Color,
				Description: t.Description,
				ImagePath:   "/tag/" + idStr + "/image",
			}
		}
	}

	// Batch load performers
	performerMap := make(map[int]MinimalPerformer)
	if len(performerIDSet) > 0 {
		perfIDs := mapKeys(performerIDSet)
		performers, err := h.repository.Performer.FindMany(ctx, perfIDs)
		if err != nil {
			return nil, fmt.Errorf("loading performers: %w", err)
		}
		for _, p := range performers {
			perfID := strconv.Itoa(p.ID)
			mp := MinimalPerformer{
				ID:             perfID,
				Name:           p.Name,
				Disambiguation: p.Disambiguation,
				Favorite:       p.Favorite,
				ImagePath:      "/performer/" + perfID + "/image",
				Country:        p.Country,
				Rating100:      p.Rating,
			}
			if p.Gender != nil {
				mp.Gender = p.Gender.String()
			}
			if p.Birthdate != nil {
				s := p.Birthdate.String()
				mp.Birthdate = &s
			}
			if p.DeathDate != nil {
				s := p.DeathDate.String()
				mp.DeathDate = &s
			}
			performerMap[p.ID] = mp
		}
		// Load primary tags for performers that have one
		primaryTagIDSet := make(map[int]bool)
		for _, p := range performers {
			if p.PrimaryTagID != nil {
				primaryTagIDSet[*p.PrimaryTagID] = true
			}
		}
		if len(primaryTagIDSet) > 0 {
			ptIDs := mapKeys(primaryTagIDSet)
			primaryTags, err := h.repository.Tag.FindMany(ctx, ptIDs)
			if err != nil {
				return nil, fmt.Errorf("loading performer primary tags: %w", err)
			}
			primaryTagMap := make(map[int]MinimalTag)
			for _, t := range primaryTags {
				idStr := strconv.Itoa(t.ID)
				primaryTagMap[t.ID] = MinimalTag{
					ID: idStr, Name: t.Name, Color: t.Color,
					Description: t.Description,
					ImagePath:   "/tag/" + idStr + "/image",
				}
			}
			for _, p := range performers {
				if p.PrimaryTagID != nil {
					if pt, ok := primaryTagMap[*p.PrimaryTagID]; ok {
						mp := performerMap[p.ID]
						mp.PrimaryTag = &pt
						performerMap[p.ID] = mp
					}
				}
			}
		}
	}

	// Batch load galleries
	galleryMap := make(map[int]MinimalGallery)
	if len(galleryIDSet) > 0 {
		galIDs := mapKeys(galleryIDSet)
		galleries, err := h.repository.Gallery.FindMany(ctx, galIDs)
		if err != nil {
			return nil, fmt.Errorf("loading galleries: %w", err)
		}
		for _, g := range galleries {
			galleryMap[g.ID] = MinimalGallery{
				ID:    strconv.Itoa(g.ID),
				Title: g.Title,
			}
		}
	}

	// Batch load groups
	groupMap := make(map[int]MinimalGroup)
	if len(groupIDSet) > 0 {
		grpIDs := mapKeys(groupIDSet)
		groups, err := h.repository.Group.FindMany(ctx, grpIDs)
		if err != nil {
			return nil, fmt.Errorf("loading groups: %w", err)
		}
		for _, g := range groups {
			grpID := strconv.Itoa(g.ID)
			groupMap[g.ID] = MinimalGroup{
				ID:        grpID,
				Name:      g.Name,
				FrontImage: "/group/" + grpID + "/frontimage",
			}
		}
	}

	// Batch load studios
	studioMap := make(map[int]MinimalStudio)
	if len(studioIDSet) > 0 {
		studioIDs := mapKeys(studioIDSet)
		studios, err := h.repository.Studio.FindMany(ctx, studioIDs)
		if err != nil {
			return nil, fmt.Errorf("loading studios: %w", err)
		}
		for _, st := range studios {
			stID := strconv.Itoa(st.ID)
			studioMap[st.ID] = MinimalStudio{
				ID:        stID,
				Name:      st.Name,
				ImagePath: "/studio/" + stID + "/image",
			}
		}
	}

	// Load scene markers per scene and collect marker tag IDs
	sceneMarkersMap := make(map[int][]*models.SceneMarker)
	markerTagIDSet := make(map[int]bool)
	for _, s := range scenes {
		markers, err := h.repository.SceneMarker.FindBySceneID(ctx, s.ID)
		if err != nil {
			return nil, fmt.Errorf("loading scene markers for scene %d: %w", s.ID, err)
		}
		sceneMarkersMap[s.ID] = markers
		for _, m := range markers {
			markerTagIDSet[m.PrimaryTagID] = true
		}
	}

	// Batch load marker primary tags (merge into tagMap)
	if len(markerTagIDSet) > 0 {
		missingTagIDs := make([]int, 0)
		for id := range markerTagIDSet {
			if _, ok := tagMap[id]; !ok {
				missingTagIDs = append(missingTagIDs, id)
			}
		}
		if len(missingTagIDs) > 0 {
			extraTags, err := h.repository.Tag.FindMany(ctx, missingTagIDs)
			if err != nil {
				return nil, fmt.Errorf("loading marker tags: %w", err)
			}
			for _, t := range extraTags {
				idStr := strconv.Itoa(t.ID)
				tagMap[t.ID] = MinimalTag{
					ID:          idStr,
					Name:        t.Name,
					Color:       t.Color,
					Description: t.Description,
					ImagePath:   "/tag/" + idStr + "/image",
				}
			}
		}
	}

	// Build enriched scenes
	result := make([]EnrichedScene, len(scenes))
	for i, s := range scenes {
		es := EnrichedScene{
			Scene: s,
			Paths: scenePaths(s.ID),
		}

		// Override Related* fields with proper arrays
		if s.URLs.Loaded() {
			es.URLs = s.URLs.List()
			if es.URLs == nil {
				es.URLs = []string{}
			}
		} else {
			es.URLs = []string{}
		}

		es.TagIDList = s.TagIDs.List()
		if es.TagIDList == nil {
			es.TagIDList = []int{}
		}
		es.PerformerIDList = s.PerformerIDs.List()
		if es.PerformerIDList == nil {
			es.PerformerIDList = []int{}
		}
		es.GalleryIDList = s.GalleryIDs.List()
		if es.GalleryIDList == nil {
			es.GalleryIDList = []int{}
		}
		if s.ScenePerformers.Loaded() {
			es.ScenePerformersList = buildScenePerformersList(s.ScenePerformers.List(), performerMap)
		} else {
			es.ScenePerformersList = []ScenePerformerEntry{}
		}
		if s.PerformerTagIDs.Loaded() {
			es.PerformerTagIDList = s.PerformerTagIDs.List()
		} else {
			es.PerformerTagIDList = []interface{}{}
		}

		// Tags
		es.Tags = make([]MinimalTag, 0, len(s.TagIDs.List()))
		for _, id := range s.TagIDs.List() {
			if t, ok := tagMap[id]; ok {
				es.Tags = append(es.Tags, t)
			}
		}

		// Performers
		es.Performers = make([]MinimalPerformer, 0, len(s.PerformerIDs.List()))
		for _, id := range s.PerformerIDs.List() {
			if p, ok := performerMap[id]; ok {
				es.Performers = append(es.Performers, p)
			}
		}

		// Galleries
		es.Galleries = make([]MinimalGallery, 0, len(s.GalleryIDs.List()))
		for _, id := range s.GalleryIDs.List() {
			if g, ok := galleryMap[id]; ok {
				es.Galleries = append(es.Galleries, g)
			}
		}

		// Groups
		es.Groups = make([]SceneGroupEntry, 0, len(s.Groups.List()))
		for _, sg := range s.Groups.List() {
			if g, ok := groupMap[sg.GroupID]; ok {
				entry := SceneGroupEntry{
					Group:      g,
					SceneIndex: sg.SceneIndex,
				}
				es.Groups = append(es.Groups, entry)
			}
		}

		// Scene Markers
		markers := sceneMarkersMap[s.ID]
		es.SceneMarkers = make([]MinimalSceneMarker, 0, len(markers))
		for _, m := range markers {
			mID := strconv.Itoa(m.ID)
			sID := strconv.Itoa(s.ID)
			msm := MinimalSceneMarker{
				ID:         mID,
				Title:      m.Title,
				Seconds:    m.Seconds,
				EndSeconds: m.EndSeconds,
				PrimaryTag: tagMap[m.PrimaryTagID],
				Tags:       []MinimalTag{},
				Stream:     "/scene/" + sID + "/scene_marker/" + mID + "/stream",
				Preview:    "/scene/" + sID + "/scene_marker/" + mID + "/preview",
				Screenshot: "/scene/" + sID + "/scene_marker/" + mID + "/screenshot",
			}
			es.SceneMarkers = append(es.SceneMarkers, msm)
		}

		// Studio
		if s.StudioID != nil {
			if st, ok := studioMap[*s.StudioID]; ok {
				es.Studio = &st
			}
		}

		// Files - pass through the existing Files field
		if s.Files.Loaded() {
			es.FileObjs = s.Files.List()
		} else {
			es.FileObjs = []interface{}{}
		}

		// StashIDs
		if s.StashIDs.Loaded() {
			es.StashIDArr = s.StashIDs.List()
		} else {
			es.StashIDArr = []interface{}{}
		}

		result[i] = es
	}

	return result, nil
}

// buildScenePerformersList builds frontend-shaped scene_performers (performer object + small_role + role_description).
func buildScenePerformersList(list []models.PerformerScenes, performerMap map[int]MinimalPerformer) []ScenePerformerEntry {
	out := make([]ScenePerformerEntry, 0, len(list))
	for _, sp := range list {
		if p, ok := performerMap[sp.PerformerID]; ok {
			out = append(out, ScenePerformerEntry{
				Performer:       p,
				SmallRole:       sp.SmallRole,
				RoleDescription: sp.RoleDescription,
			})
		}
	}
	return out
}

// mapKeys extracts keys from a map[int]bool.
func mapKeys(m map[int]bool) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
