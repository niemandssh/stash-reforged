package match

import (
	"context"
	"strings"
	"testing"

	"github.com/stashapp/stash/pkg/models"
)

type testStudioFinder struct {
	studios []*models.Studio
}

func (t *testStudioFinder) Query(ctx context.Context, filter *models.StudioFilterType, findFilter *models.FindFilterType) ([]*models.Studio, int, error) {
	if filter == nil {
		return nil, 0, nil
	}

	var matches []*models.Studio
	for _, st := range t.studios {
		if studioMatchesFilter(st, filter) {
			matches = append(matches, st)
		}
	}

	return matches, len(matches), nil
}

func (t *testStudioFinder) FindByStashID(ctx context.Context, stashID models.StashID) ([]*models.Studio, error) {
	return nil, nil
}

func (t *testStudioFinder) QueryCount(ctx context.Context, filter *models.StudioFilterType, findFilter *models.FindFilterType) (int, error) {
	ret, count, err := t.Query(ctx, filter, findFilter)
	if err != nil {
		return 0, err
	}
	_ = ret
	return count, nil
}

func studioMatchesFilter(st *models.Studio, filter *models.StudioFilterType) bool {
	if filter.Aliases != nil && filter.Aliases.Modifier == models.CriterionModifierEquals && st.Aliases.Loaded() {
		for _, alias := range st.Aliases.List() {
			if strings.EqualFold(alias, filter.Aliases.Value) {
				return true
			}
		}
	}

	if filter.URL != nil && filter.URL.Modifier == models.CriterionModifierIncludes {
		if strings.Contains(strings.ToLower(st.URL), strings.ToLower(filter.URL.Value)) {
			return true
		}
	}

	if filter.Name != nil {
		switch filter.Name.Modifier {
		case models.CriterionModifierEquals:
			if strings.EqualFold(st.Name, filter.Name.Value) {
				return true
			}
		case models.CriterionModifierIncludes:
			if strings.Contains(strings.ToLower(st.Name), strings.ToLower(filter.Name.Value)) {
				return true
			}
		}
	}

	return false
}

func TestWebsiteHostCandidates(t *testing.T) {
	got := WebsiteHostCandidates("https://www.montrealbbc.com/models/foo")
	want := []string{"www.montrealbbc.com", "montrealbbc.com", "montrealbbc"}

	if len(got) != len(want) {
		t.Fatalf("WebsiteHostCandidates() = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("WebsiteHostCandidates()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSecondLevelDomainName(t *testing.T) {
	if got := SecondLevelDomainName("https://www.montrealbbc.com"); got != "montrealbbc" {
		t.Fatalf("SecondLevelDomainName() = %q, want montrealbbc", got)
	}
	if got := SecondLevelDomainName("https://LegalPorno.com"); got != "LegalPorno" {
		t.Fatalf("SecondLevelDomainName() = %q, want LegalPorno", got)
	}
	if got := SecondLevelDomainName("PornBox.com"); got != "PornBox" {
		t.Fatalf("SecondLevelDomainName() = %q, want PornBox", got)
	}
}

func TestIsStudioCandidateURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://www.montrealbbc.com", true},
		{"https://www.montrealbbc.com/", true},
		{"https://twitter.com/performer", false},
		{"https://www.montrealbbc.com/model/jane", false},
		{"https://onlyfans.com/performer", false},
		{"https://pornolab.net/forum/viewtopic.php", false},
		{"pornolab.net", false},
		{"https://LegalPorno.com / PornBox.com / Xandervision", true},
		{"PornBox.com", true},
		{"Taboo Diaries/clips4sale.com", true},
		{"https://Taboo Diaries/clips4sale.com", true},
		{"https://www.clips4sale.com", true},
		{"example.com/studio", false},
	}

	for _, tt := range tests {
		if got := IsStudioCandidateURL(tt.url); got != tt.want {
			t.Fatalf("IsStudioCandidateURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestSplitStudioReferences(t *testing.T) {
	got := splitStudioReferences("https://LegalPorno.com / PornBox.com / Xandervision")
	want := []string{"https://LegalPorno.com", "PornBox.com", "Xandervision"}

	if len(got) != len(want) {
		t.Fatalf("splitStudioReferences() = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitStudioReferences()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitStudioReferencesSlashSeparated(t *testing.T) {
	tests := []struct {
		value string
		want  []string
	}{
		{
			value: "Taboo Diaries/clips4sale.com",
			want:  []string{"Taboo Diaries", "clips4sale.com"},
		},
		{
			value: "https://Taboo Diaries/clips4sale.com",
			want:  []string{"Taboo Diaries", "clips4sale.com"},
		},
		{
			value: "LegalPorno/PornBox.com",
			want:  []string{"LegalPorno", "PornBox.com"},
		},
		{
			value: "Princess Leia /clips4sale.com",
			want:  []string{"Princess Leia", "clips4sale.com"},
		},
		{
			value: "https://www.clips4sale.com",
			want:  []string{"https://www.clips4sale.com"},
		},
		{
			value: "example.com/studio",
			want:  []string{"example.com/studio"},
		},
	}

	for _, tt := range tests {
		got := splitStudioReferences(tt.value)
		if len(got) != len(tt.want) {
			t.Fatalf("splitStudioReferences(%q) = %v, want %v", tt.value, got, tt.want)
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Fatalf("splitStudioReferences(%q)[%d] = %q, want %q", tt.value, i, got[i], tt.want[i])
			}
		}
	}
}

func TestStudioFromReferenceDoesNotReturnNilWithTrue(t *testing.T) {
	qb := &testStudioFinder{}

	// Single-label host: candidate URL but no usable second-level domain name.
	for _, reference := range []string{
		"https://onlyfans",
		"https://internal",
	} {
		got, ok := StudioFromReference(context.Background(), qb, reference)
		if ok && got == nil {
			t.Fatalf("StudioFromReference(%q) = nil, true; want nil, false or non-nil studio", reference)
		}
	}
}

func TestExtractStudioURLsSlashSeparated(t *testing.T) {
	qb := &testStudioFinder{
		studios: []*models.Studio{
			{
				ID:   7,
				Name: "Clips4Sale",
				URL:  "https://clips4sale.com",
			},
		},
	}

	remaining, studio := ExtractStudioURLs(context.Background(), qb, []string{"Taboo Diaries/clips4sale.com"})
	if len(remaining) != 0 {
		t.Fatalf("remaining URLs = %v, want empty", remaining)
	}
	if studio == nil {
		t.Fatal("studio = nil, want compound studio result")
	}
	if studio.Name != "Taboo Diaries" {
		t.Fatalf("primary studio name = %q, want Taboo Diaries", studio.Name)
	}
	if studio.Parent == nil || studio.Parent.StoredID == nil || *studio.Parent.StoredID != "7" {
		t.Fatalf("parent studio = %#v, want Clips4Sale with stored_id 7", studio.Parent)
	}
}

func TestScrapedStudioProposalFromReference(t *testing.T) {
	if got := scrapedStudioProposalFromReference("PornBox.com"); got == nil || got.Name != "PornBox" {
		t.Fatalf("proposal from domain = %#v, want name PornBox", got)
	}

	if got := scrapedStudioProposalFromReference("Xandervision"); got == nil || got.Name != "Xandervision" {
		t.Fatalf("proposal from name = %#v, want name Xandervision", got)
	}
}

func TestCombineCompoundStudioResults(t *testing.T) {
	teamskeetID := "42"
	teamskeet := &models.ScrapedStudio{
		StoredID: &teamskeetID,
		Name:     "TeamSkeet",
	}
	littleAsians := &models.ScrapedStudio{
		Name: "LittleAsians",
		URL:  stringPtrIfNotEmpty("https://LittleAsians.com"),
	}

	got := combineCompoundStudioResults([]*models.ScrapedStudio{teamskeet}, []*models.ScrapedStudio{littleAsians})
	if got == nil || got.Name != "LittleAsians" {
		t.Fatalf("primary studio = %#v, want LittleAsians proposal", got)
	}
	if got.Parent == nil || got.Parent.Name != "TeamSkeet" || got.Parent.StoredID == nil || *got.Parent.StoredID != "42" {
		t.Fatalf("parent studio = %#v, want matched TeamSkeet", got.Parent)
	}
}

func TestStudioFromValueCompoundMatchedAndProposal(t *testing.T) {
	qb := &testStudioFinder{
		studios: []*models.Studio{
			{
				ID:   42,
				Name: "TeamSkeet",
				URL:  "https://teamskeet.com",
			},
		},
	}

	got, ok := studioFromValue(context.Background(), qb, "LittleAsians.com / TeamSkeet.com")
	if !ok || got == nil {
		t.Fatal("studioFromValue() = nil, want compound studio result")
	}
	if got.StoredID != nil {
		t.Fatalf("primary studio stored_id = %v, want nil proposal", got.StoredID)
	}
	if got.Name != "LittleAsians" {
		t.Fatalf("primary studio name = %q, want LittleAsians", got.Name)
	}
	if got.Parent == nil || got.Parent.StoredID == nil || *got.Parent.StoredID != "42" {
		t.Fatalf("parent studio = %#v, want TeamSkeet with stored_id 42", got.Parent)
	}
}

func TestStudioFromValueDoesNotFuzzyMatchName(t *testing.T) {
	qb := &testStudioFinder{
		studios: []*models.Studio{
			{
				ID:   99,
				Name: "Princess Cum",
				URL:  "https://princesscum.com",
			},
			{
				ID:   7,
				Name: "Clips4Sale",
				URL:  "https://clips4sale.com",
			},
		},
	}

	got, ok := studioFromValue(context.Background(), qb, "Princess Leia /clips4sale.com")
	if !ok || got == nil {
		t.Fatal("studioFromValue() = nil, want compound studio result")
	}
	if got.StoredID != nil {
		t.Fatalf("primary studio stored_id = %v, want Princess Leia proposal", got.StoredID)
	}
	if got.Name != "Princess Leia" {
		t.Fatalf("primary studio name = %q, want Princess Leia", got.Name)
	}
	if got.Parent == nil || got.Parent.StoredID == nil || *got.Parent.StoredID != "7" {
		t.Fatalf("parent studio = %#v, want Clips4Sale with stored_id 7", got.Parent)
	}
}

func TestPornolabIsNotStudio(t *testing.T) {
	qb := &testStudioFinder{
		studios: []*models.Studio{
			{
				ID:   1,
				Name: "Pornolab",
				URL:  "https://pornolab.net",
			},
		},
	}

	tests := []struct {
		reference string
	}{
		{reference: "pornolab"},
		{reference: "Pornolab"},
		{reference: "pornolab.net"},
		{reference: "https://pornolab.net/forum/viewtopic.php"},
		{reference: "https://www.pornolab.net"},
	}

	for _, tt := range tests {
		if got, ok := StudioFromReference(context.Background(), qb, tt.reference); ok {
			t.Fatalf("StudioFromReference(%q) = %#v, want no studio", tt.reference, got)
		}
	}

	remaining, studio := ExtractStudioURLs(context.Background(), qb, []string{"https://pornolab.net/forum/viewtopic.php?t=1"})
	if studio != nil {
		t.Fatalf("ExtractStudioURLs() studio = %#v, want nil", studio)
	}
	if len(remaining) != 1 || remaining[0] != "https://pornolab.net/forum/viewtopic.php?t=1" {
		t.Fatalf("ExtractStudioURLs() remaining = %v, want original URL preserved", remaining)
	}

	if got, ok := StudioFromTitle(context.Background(), qb, "[pornolab.net] scene title"); ok {
		t.Fatalf("StudioFromTitle() = %#v, want no studio", got)
	}

	if got := scrapedStudioProposalFromReference("pornolab.net"); got != nil {
		t.Fatalf("scrapedStudioProposalFromReference(pornolab.net) = %#v, want nil", got)
	}
}

func TestStudioReferenceFromTitle(t *testing.T) {
	got, ok := studioReferenceFromTitle("[ManyVids.com] ariana simon - Pregnant Boy Girl Creampie & Blowjob [13.03.2018 г.]")
	if !ok || got != "ManyVids.com" {
		t.Fatalf("studioReferenceFromTitle() = %q, %v, want ManyVids.com, true", got, ok)
	}

	if _, ok := studioReferenceFromTitle("ariana simon - title"); ok {
		t.Fatal("studioReferenceFromTitle() without prefix should return false")
	}
}

func TestStudioFromTitleManyVids(t *testing.T) {
	qb := &testStudioFinder{
		studios: []*models.Studio{
			{
				ID:   11,
				Name: "ManyVids",
				URL:  "https://manyvids.com",
			},
		},
	}

	title := "[ManyVids.com] ariana simon - Pregnant Boy Girl Creampie & Blowjob [13.03.2018 г., 1080p]"
	got, ok := StudioFromTitle(context.Background(), qb, title)
	if !ok || got == nil {
		t.Fatal("StudioFromTitle() = nil, want ManyVids studio")
	}
	if got.StoredID == nil || *got.StoredID != "11" {
		t.Fatalf("studio stored_id = %v, want 11", got.StoredID)
	}
	if got.Name != "ManyVids" {
		t.Fatalf("studio name = %q, want ManyVids", got.Name)
	}
}

func TestStudioFromTitleProposal(t *testing.T) {
	qb := &testStudioFinder{}

	title := "[NewStudio.com] performer - scene title"
	got, ok := StudioFromTitle(context.Background(), qb, title)
	if !ok || got == nil {
		t.Fatal("StudioFromTitle() = nil, want studio proposal")
	}
	if got.StoredID != nil {
		t.Fatalf("studio stored_id = %v, want nil proposal", got.StoredID)
	}
	if got.Name != "NewStudio" {
		t.Fatalf("studio name = %q, want NewStudio", got.Name)
	}
}
