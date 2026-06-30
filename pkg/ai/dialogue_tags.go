package ai

import "strings"

type dialogueTagGroup struct {
	id       int
	roots    []string
	textKeys []string
}

var dialogueTagGroups = []dialogueTagGroup{
	{
		id:    0,
		roots: []string{"impregnate", "impregnation", "impregnating", "impregnated"},
		textKeys: []string{
			"impregnate", "impregnated", "impregnating", "impregnation",
		},
	},
	{
		id:    1,
		roots: []string{"birth control", "contraception", "contraceptive"},
		textKeys: []string{
			"birth control", "contraception", "contraceptive", "on the pill", "the pill",
		},
	},
}

func dialogueTagGroupForName(tagName string) (dialogueTagGroup, bool) {
	lower := strings.ToLower(strings.TrimSpace(tagName))
	for _, group := range dialogueTagGroups {
		for _, root := range group.roots {
			if lower == root || strings.Contains(lower, root) {
				return group, true
			}
		}
	}
	return dialogueTagGroup{}, false
}

func textMentionsDialogueGroup(texts []string, group dialogueTagGroup) bool {
	combined := strings.ToLower(strings.Join(texts, "\n"))
	for _, key := range group.textKeys {
		if strings.Contains(key, " ") {
			if strings.Contains(combined, key) {
				return true
			}
			continue
		}
		if containsWord(combined, key) {
			return true
		}
	}
	return false
}

func filterDialogueOnlyTagList(tags []string, texts []string) []string {
	filtered := make([]string, 0, len(tags))
	for _, tag := range tags {
		group, isDialogue := dialogueTagGroupForName(tag)
		if !isDialogue {
			filtered = append(filtered, tag)
			continue
		}
		if textMentionsDialogueGroup(texts, group) {
			filtered = append(filtered, tag)
		}
	}
	return resolveDialogueTagConflicts(filtered, texts)
}

func resolveDialogueTagConflicts(tags []string, texts []string) []string {
	var hasImpregnate, hasBirthControl bool
	for _, tag := range tags {
		group, ok := dialogueTagGroupForName(tag)
		if !ok {
			continue
		}
		switch group.id {
		case 0:
			hasImpregnate = true
		case 1:
			hasBirthControl = true
		}
	}
	if !hasImpregnate || !hasBirthControl {
		return tags
	}

	impregnateInText := textMentionsDialogueGroup(texts, dialogueTagGroups[0])
	birthControlInText := textMentionsDialogueGroup(texts, dialogueTagGroups[1])

	var dropGroup int
	switch {
	case birthControlInText:
		dropGroup = 0
	case impregnateInText:
		dropGroup = 1
	default:
		return filterTags(tags, func(tag string) bool {
			_, isDialogue := dialogueTagGroupForName(tag)
			return !isDialogue
		})
	}

	filtered := make([]string, 0, len(tags))
	for _, tag := range tags {
		group, ok := dialogueTagGroupForName(tag)
		if ok && group.id == dropGroup {
			continue
		}
		filtered = append(filtered, tag)
	}
	return filtered
}

func applyDialogueOnlyTagRules(output *SceneFillOutput, title, details, filename string) {
	texts := []string{title, details, filename}
	output.Tags = filterDialogueOnlyTagList(output.Tags, texts)
	for i := range output.PerformerTags {
		output.PerformerTags[i].Tags = filterDialogueOnlyTagList(output.PerformerTags[i].Tags, texts)
	}
}
