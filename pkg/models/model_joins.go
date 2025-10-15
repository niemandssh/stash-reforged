package models

import (
	"fmt"
	"strconv"
)

type GroupsScenes struct {
	GroupID int `json:"movie_id"`
	// SceneID    int  `json:"scene_id"`
	SceneIndex *int `json:"scene_index"`
}

func (s GroupsScenes) SceneMovieInput() SceneMovieInput {
	return SceneMovieInput{
		MovieID:    strconv.Itoa(s.GroupID),
		SceneIndex: s.SceneIndex,
	}
}

func (s GroupsScenes) Equal(o GroupsScenes) bool {
	return o.GroupID == s.GroupID && ((o.SceneIndex == nil && s.SceneIndex == nil) ||
		(o.SceneIndex != nil && s.SceneIndex != nil && *o.SceneIndex == *s.SceneIndex))
}

func GroupsScenesFromInput(input []SceneMovieInput) ([]GroupsScenes, error) {
	ret := make([]GroupsScenes, len(input))

	for i, v := range input {
		mID, err := strconv.Atoi(v.MovieID)
		if err != nil {
			return nil, fmt.Errorf("invalid movie ID: %s", v.MovieID)
		}

		ret[i] = GroupsScenes{
			GroupID:    mID,
			SceneIndex: v.SceneIndex,
		}
	}

	return ret, nil
}

type ScenesTagsPerformer struct {
	SceneID     int  `json:"scene_id"`
	TagID       int  `json:"tag_id"`
	PerformerID *int `json:"performer_id"`
}

type GroupIDDescription struct {
	GroupID     int    `json:"group_id"`
	Description string `json:"description"`
}
