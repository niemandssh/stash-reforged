package models

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_scrapedToStudioInput(t *testing.T) {
	const name = "name"
	url := "url"
	emptyEndpoint := ""
	endpoint := "endpoint"
	remoteSiteID := "remoteSiteID"

	tests := []struct {
		name     string
		studio   *ScrapedStudio
		endpoint string
		want     *Studio
	}{
		{
			"set all",
			&ScrapedStudio{
				Name:         name,
				URL:          &url,
				RemoteSiteID: &remoteSiteID,
			},
			endpoint,
			&Studio{
				Name: name,
				URL:  url,
				StashIDs: NewRelatedStashIDs([]StashID{
					{
						Endpoint: endpoint,
						StashID:  remoteSiteID,
					},
				}),
			},
		},
		{
			"set none",
			&ScrapedStudio{
				Name: name,
			},
			emptyEndpoint,
			&Studio{
				Name: name,
			},
		},
		{
			"missing remoteSiteID",
			&ScrapedStudio{
				Name: name,
			},
			endpoint,
			&Studio{
				Name: name,
			},
		},
		{
			"set stashid",
			&ScrapedStudio{
				Name:         name,
				RemoteSiteID: &remoteSiteID,
			},
			endpoint,
			&Studio{
				Name: name,
				StashIDs: NewRelatedStashIDs([]StashID{
					{
						Endpoint: endpoint,
						StashID:  remoteSiteID,
					},
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.studio.ToStudio(tt.endpoint, nil)

			assert.NotEqual(t, time.Time{}, got.CreatedAt)
			assert.NotEqual(t, time.Time{}, got.UpdatedAt)

			got.CreatedAt = time.Time{}
			got.UpdatedAt = time.Time{}
			if got.StashIDs.Loaded() && len(got.StashIDs.List()) > 0 {
				for stid := range got.StashIDs.List() {
					got.StashIDs.List()[stid].UpdatedAt = time.Time{}
				}
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_scrapedToPerformerInput(t *testing.T) {
	name := "name"
	emptyEndpoint := ""
	endpoint := "endpoint"
	remoteSiteID := "remoteSiteID"

	var stringValues []string
	for i := 0; i < 20; i++ {
		stringValues = append(stringValues, strconv.Itoa(i))
	}

	upTo := 0
	nextVal := func() *string {
		ret := stringValues[upTo]
		upTo = (upTo + 1) % len(stringValues)
		return &ret
	}

	nextIntVal := func() *int {
		ret := upTo
		upTo = (upTo + 1) % len(stringValues)
		return &ret
	}

	dateFromInt := func(i int) *Date {
		t := time.Date(2001, 1, i, 0, 0, 0, 0, time.UTC)
		d := Date{Time: t}
		return &d
	}
	dateStrFromInt := func(i int) *string {
		s := dateFromInt(i).String()
		return &s
	}

	genderFromInt := func(i int) *GenderEnum {
		g := AllGenderEnum[i%len(AllGenderEnum)]
		return &g
	}
	genderStrFromInt := func(i int) *string {
		s := genderFromInt(i).String()
		return &s
	}

	tests := []struct {
		name      string
		performer *ScrapedPerformer
		endpoint  string
		want      *Performer
	}{
		{
			"set all",
			&ScrapedPerformer{
				Name:           &name,
				Disambiguation: nextVal(),
				Birthdate:      dateStrFromInt(*nextIntVal()),
				DeathDate:      dateStrFromInt(*nextIntVal()),
				Gender:         genderStrFromInt(*nextIntVal()),
				Ethnicity:      nextVal(),
				Country:        nextVal(),
				EyeColor:       nextVal(),
				HairColor:      nextVal(),
				Height:         nextVal(),
				Weight:         nextVal(),
				Measurements:   nextVal(),
				FakeTits:       nextVal(),
				CareerLength:   nextVal(),
				Tattoos:        nextVal(),
				Piercings:      nextVal(),
				Aliases:        nextVal(),
				URL:            nextVal(),
				Twitter:        nextVal(),
				Instagram:      nextVal(),
				Details:        nextVal(),
				RemoteSiteID:   &remoteSiteID,
			},
			endpoint,
			&Performer{
				Name:           name,
				Disambiguation: *nextVal(),
				Birthdate:      dateFromInt(*nextIntVal()),
				DeathDate:      dateFromInt(*nextIntVal()),
				Gender:         genderFromInt(*nextIntVal()),
				Ethnicity:      *nextVal(),
				Country:        *nextVal(),
				EyeColor:       *nextVal(),
				HairColor:      *nextVal(),
				Height:         nextIntVal(),
				Weight:         nextIntVal(),
				Measurements:   *nextVal(),
				FakeTits:       *nextVal(),
				CareerLength:   *nextVal(),
				Tattoos:        *nextVal(),
				Piercings:      *nextVal(),
				Aliases:        NewRelatedStrings([]string{*nextVal()}),
				URLs:           NewRelatedStrings([]string{*nextVal(), *nextVal(), *nextVal()}),
				Details:        *nextVal(),
				StashIDs: NewRelatedStashIDs([]StashID{
					{
						Endpoint: endpoint,
						StashID:  remoteSiteID,
					},
				}),
			},
		},
		{
			"set none",
			&ScrapedPerformer{
				Name: &name,
			},
			emptyEndpoint,
			&Performer{
				Name: name,
			},
		},
		{
			"missing remoteSiteID",
			&ScrapedPerformer{
				Name: &name,
			},
			endpoint,
			&Performer{
				Name: name,
			},
		},
		{
			"set stashid",
			&ScrapedPerformer{
				Name:         &name,
				RemoteSiteID: &remoteSiteID,
			},
			endpoint,
			&Performer{
				Name: name,
				StashIDs: NewRelatedStashIDs([]StashID{
					{
						Endpoint: endpoint,
						StashID:  remoteSiteID,
					},
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.performer.ToPerformer(tt.endpoint, nil)

			assert.NotEqual(t, time.Time{}, got.CreatedAt)
			assert.NotEqual(t, time.Time{}, got.UpdatedAt)

			got.CreatedAt = time.Time{}
			got.UpdatedAt = time.Time{}

			if got.StashIDs.Loaded() && len(got.StashIDs.List()) > 0 {
				for stid := range got.StashIDs.List() {
					got.StashIDs.List()[stid].UpdatedAt = time.Time{}
				}
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScrapedStudio_ToPartial(t *testing.T) {
	var (
		id                = 1000
		idStr             = strconv.Itoa(id)
		storedID          = "storedID"
		parentStoredID    = 2000
		parentStoredIDStr = strconv.Itoa(parentStoredID)
		name              = "name"
		url               = "url"
		remoteSiteID      = "remoteSiteID"
		endpoint          = "endpoint"
		image             = "image"
		images            = []string{image}

		existingEndpoint = "existingEndpoint"
		existingStashID  = StashID{"existingStashID", existingEndpoint, time.Time{}}
		existingStashIDs = []StashID{existingStashID}
	)

	fullStudio := ScrapedStudio{
		StoredID: &storedID,
		Name:     name,
		URL:      &url,
		Parent: &ScrapedStudio{
			StoredID: &parentStoredIDStr,
		},
		Image:        &image,
		Images:       images,
		RemoteSiteID: &remoteSiteID,
	}

	type args struct {
		id               string
		endpoint         string
		excluded         map[string]bool
		existingStashIDs []StashID
	}

	stdArgs := args{
		id:               idStr,
		endpoint:         endpoint,
		excluded:         map[string]bool{},
		existingStashIDs: existingStashIDs,
	}

	excludeAll := map[string]bool{
		"name":   true,
		"url":    true,
		"parent": true,
	}

	tests := []struct {
		name string
		o    ScrapedStudio
		args args
		want StudioPartial
	}{
		{
			"full no exclusions",
			fullStudio,
			stdArgs,
			StudioPartial{
				ID:       id,
				Name:     NewOptionalString(name),
				URL:      NewOptionalString(url),
				ParentID: NewOptionalInt(parentStoredID),
				StashIDs: &UpdateStashIDs{
					StashIDs: append(existingStashIDs, StashID{
						Endpoint: endpoint,
						StashID:  remoteSiteID,
					}),
					Mode: RelationshipUpdateModeSet,
				},
			},
		},
		{
			"exclude all",
			fullStudio,
			args{
				id:       idStr,
				excluded: excludeAll,
			},
			StudioPartial{
				ID: id,
			},
		},
		{
			"overwrite stash id",
			fullStudio,
			args{
				id:               idStr,
				excluded:         excludeAll,
				endpoint:         existingEndpoint,
				existingStashIDs: existingStashIDs,
			},
			StudioPartial{
				ID: id,
				StashIDs: &UpdateStashIDs{
					StashIDs: []StashID{{
						Endpoint: existingEndpoint,
						StashID:  remoteSiteID,
					}},
					Mode: RelationshipUpdateModeSet,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.o
			got := s.ToPartial(tt.args.id, tt.args.endpoint, tt.args.excluded, tt.args.existingStashIDs)

			// unset updatedAt - we don't need to compare it
			got.UpdatedAt = OptionalTime{}
			if got.StashIDs != nil && len(got.StashIDs.StashIDs) > 0 {
				for stid := range got.StashIDs.StashIDs {
					got.StashIDs.StashIDs[stid].UpdatedAt = time.Time{}
				}
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScrapedPerformerUnmarshalJSON_aliases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *string
		wantErr bool
	}{
		{
			name:  "alias string",
			input: `{"name": "test", "aliases": "alias1, alias2"}`,
			want:  strPtr("alias1, alias2"),
		},
		{
			name:  "alias array",
			input: `{"name": "test", "aliases": ["alias1", "alias2"]}`,
			want:  strPtr("alias1,alias2"),
		},
		{
			name:  "null aliases",
			input: `{"name": "test", "aliases": null}`,
			want:  nil,
		},
		{
			name:    "invalid aliases",
			input:   `{"name": "test", "aliases": 123}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ScrapedPerformer
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if tt.want == nil {
				assert.Nil(t, got.Aliases)
				return
			}

			assert.NotNil(t, got.Aliases)
			assert.Equal(t, *tt.want, *got.Aliases)
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestScrapedSceneUnmarshalJSON_performerAliasesArray(t *testing.T) {
	input := `{
		"title": "test scene",
		"performers": [
			{
				"name": "Performer One",
				"aliases": ["alias1", "alias2"]
			}
		]
	}`

	var got ScrapedScene
	err := json.Unmarshal([]byte(input), &got)
	assert.NoError(t, err)
	assert.Len(t, got.Performers, 1)
	assert.NotNil(t, got.Performers[0].Aliases)
	assert.Equal(t, "alias1,alias2", *got.Performers[0].Aliases)
}

func TestCleanScrapedPerformerAliases(t *testing.T) {
	aliases := "aka Montrealbbc, aka OtherName"
	performer := &ScrapedPerformer{
		Aliases: &aliases,
	}

	CleanScrapedPerformerAliases(performer)

	assert.NotNil(t, performer.Aliases)
	assert.Equal(t, "Montrealbbc,OtherName", *performer.Aliases)
}

func TestNormalizeScrapedPerformer_nameAkaAlias(t *testing.T) {
	name := "Marin Niko aka Marin Nico"
	performer := &ScrapedPerformer{
		Name: &name,
	}

	NormalizeScrapedPerformer(performer)

	assert.NotNil(t, performer.Name)
	assert.Equal(t, "Marin Niko", *performer.Name)
	assert.NotNil(t, performer.Aliases)
	assert.Equal(t, "Marin Nico", *performer.Aliases)
}

func TestScrapedPerformerUnmarshalJSON_nameAkaAlias(t *testing.T) {
	input := `{"name": "Marin Niko aka Marin Nico"}`

	var got ScrapedPerformer
	err := json.Unmarshal([]byte(input), &got)
	assert.NoError(t, err)
	assert.NotNil(t, got.Name)
	assert.Equal(t, "Marin Niko", *got.Name)
	assert.NotNil(t, got.Aliases)
	assert.Equal(t, "Marin Nico", *got.Aliases)
}

func TestNormalizeScrapedPerformer_nameAkaAliasAppendsExisting(t *testing.T) {
	name := "Marin Niko aka Marin Nico"
	aliases := "Existing Alias"
	performer := &ScrapedPerformer{
		Name:    &name,
		Aliases: &aliases,
	}

	NormalizeScrapedPerformer(performer)

	assert.Equal(t, "Marin Niko", *performer.Name)
	assert.Equal(t, "Existing Alias,Marin Nico", *performer.Aliases)
}

func TestScrapedPerformerUnmarshalJSON_stripsAkaPrefix(t *testing.T) {
	input := `{"name": "test", "aliases": ["aka Montrealbbc", "aka OtherName"]}`

	var got ScrapedPerformer
	err := json.Unmarshal([]byte(input), &got)
	assert.NoError(t, err)
	assert.NotNil(t, got.Aliases)
	assert.Equal(t, "Montrealbbc,OtherName", *got.Aliases)
}
