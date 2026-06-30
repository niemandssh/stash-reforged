package ai

import "testing"

func TestFilterDialogueOnlyTagsRequiresTextSource(t *testing.T) {
	texts := []string{"Hot Scene", "", "scene.mp4"}
	tags := []string{"missionary", "impregnate", "birth control"}

	got := filterDialogueOnlyTagList(tags, texts)
	if len(got) != 1 || got[0] != "missionary" {
		t.Fatalf("expected only missionary, got %v", got)
	}
}

func TestFilterDialogueOnlyTagsFromTitle(t *testing.T) {
	texts := []string{"Breeding Impregnation Fantasy", "", ""}
	tags := []string{"impregnate", "missionary"}

	got := filterDialogueOnlyTagList(tags, texts)
	if len(got) != 2 {
		t.Fatalf("expected impregnate + missionary, got %v", got)
	}
}

func TestResolveImpregnateBirthControlConflict(t *testing.T) {
	texts := []string{"She mentions birth control", "", ""}
	tags := []string{"birth control", "impregnate"}

	got := filterDialogueOnlyTagList(tags, texts)
	if len(got) != 1 || got[0] != "birth control" {
		t.Fatalf("expected only birth control, got %v", got)
	}
}

func TestResolveImpregnateBirthControlConflictKeepsImpregnate(t *testing.T) {
	texts := []string{"Impregnate Me", "", ""}
	tags := []string{"birth control", "impregnate"}

	got := filterDialogueOnlyTagList(tags, texts)
	if len(got) != 1 || got[0] != "impregnate" {
		t.Fatalf("expected only impregnate, got %v", got)
	}
}
