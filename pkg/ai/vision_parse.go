package ai

import (
	"strconv"
	"strings"
)

func parseSummaryLine(desc, prefix string) map[string]string {
	lower := strings.ToLower(desc)
	prefixLower := strings.ToLower(prefix)
	idx := strings.Index(lower, prefixLower)
	if idx < 0 {
		return nil
	}

	line := desc[idx+len(prefix):]
	if nl := strings.Index(line, "\n"); nl >= 0 {
		line = line[:nl]
	}

	result := make(map[string]string)
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		result[strings.TrimSpace(strings.ToLower(k))] = strings.TrimSpace(strings.ToLower(v))
	}
	return result
}

func summaryValueYes(value string) bool {
	return strings.HasPrefix(value, "yes")
}

func visionAllowsClothingNegation(desc string) bool {
	fields := parseSummaryLine(desc, "Clothing summary:")
	if len(fields) > 0 {
		if fields["fully_nude_all_panels"] != "yes" {
			return false
		}
		return fields["bra_visible"] == "no" &&
			fields["underwear_visible"] == "no" &&
			fields["pants_visible"] == "no"
	}

	return visionIndicatesFullyNudeAllPanels(desc)
}

func removeInvalidClothingNegations(output *SceneFillOutput, visionDesc string) {
	fields := parseSummaryLine(visionDesc, "Clothing summary:")
	if len(fields) == 0 {
		return
	}

	removeIf := map[string]string{
		"no bra":       "bra_visible",
		"no underwear": "underwear_visible",
		"no pants":     "pants_visible",
	}

	output.Tags = filterTags(output.Tags, func(tag string) bool {
		lower := strings.ToLower(strings.TrimSpace(tag))
		field, ok := removeIf[lower]
		if !ok {
			return true
		}
		if summaryValueYes(fields[field]) {
			return false
		}
		if fields["fully_nude_all_panels"] == "partial" || fields["fully_nude_all_panels"] == "no" {
			return false
		}
		return true
	})
}

func filterTags(tags []string, keep func(string) bool) []string {
	filtered := make([]string, 0, len(tags))
	for _, tag := range tags {
		if keep(tag) {
			filtered = append(filtered, tag)
		}
	}
	return filtered
}

func parseVisibilityRatings(desc string) map[string]int {
	ratings := make(map[string]int)
	lower := strings.ToLower(desc)
	idx := strings.Index(lower, "visibility ratings:")
	if idx < 0 {
		return ratings
	}

	rest := desc[idx+len("visibility ratings:"):]
	if nl := strings.Index(rest, "\n"); nl >= 0 {
		rest = rest[:nl]
	}

	for _, part := range strings.Split(rest, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			continue
		}
		ratings[strings.TrimSpace(strings.ToLower(k))] = n
	}
	return ratings
}

var accessoryTagKeywords = map[string][]string{
	"choker":      {"choker", "neck choker"},
	"necklace":    {"necklace"},
	"ear rings":   {"earrings", "ear rings", "ear ring", "earring", "stud earring", "hoop earring"},
	"glasses":     {"glasses", "eyeglasses"},
	"piercing":    {"piercing", "piercings"},
	"piercings":   {"piercing", "piercings"},
	"hat":         {"hat", "cap"},
	"garter":      {"garter", "garters"},
	"garter belt": {"garter belt", "garter"},
	"stockings":   {"stockings", "hosiery"},
	"fishnets":    {"fishnets", "fishnet"},
	"heels":       {"heels", "high heels"},
	"scrunchie":   {"scrunchie", "scrunchies", "hair tie", "hair ties", "hair accessory", "hair accessories"},
	"hair tie":    {"hair tie", "hair ties", "scrunchie", "scrunchies"},
	"ribbon":      {"ribbon", "hair ribbon"},
	"bracelet":    {"bracelet"},
	"ring":        {"ring on finger", "finger ring"},
	"watch":       {"watch", "wristwatch"},
	"mask":        {"mask"},
	"gloves":      {"gloves"},
}

func isAccessoryCategory(c CriterionInfo) bool {
	text := strings.ToLower(c.Name + " " + c.Description)
	return strings.Contains(text, "accessor") ||
		strings.Contains(text, "jewel") ||
		strings.Contains(text, "choker") ||
		strings.Contains(text, "earring")
}

func collectAccessoryCandidateTags(criteria []CriterionInfo, ctx tagFilterContext) []string {
	seen := make(map[string]struct{})
	var tags []string

	add := func(name string) {
		canonical, ok := ctx.tagIndex[strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			return
		}
		key := strings.ToLower(canonical)
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		tags = append(tags, canonical)
	}

	for _, c := range criteria {
		if !isAccessoryCategory(c) {
			continue
		}
		for _, name := range c.ExistingTags {
			add(name)
		}
	}

	for name := range ctx.tagIndex {
		lower := strings.ToLower(name)
		if _, known := accessoryTagKeywords[lower]; known {
			add(name)
		}
	}

	return tags
}

func summaryValueNo(value string) bool {
	return value == "no" || strings.HasPrefix(value, "no(")
}

func accessorySummaryKeys(tagName string) []string {
	tagLower := strings.ToLower(strings.TrimSpace(tagName))
	keys := []string{tagLower, strings.ReplaceAll(tagLower, " ", "_")}
	if aliases, ok := accessoryTagKeywords[tagLower]; ok {
		for _, alias := range aliases {
			keys = append(keys, alias, strings.ReplaceAll(alias, " ", "_"))
		}
	}
	return keys
}

func visionProseWithoutSummaries(desc string) string {
	var kept []string
	for _, line := range strings.Split(desc, "\n") {
		trimmed := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(trimmed, "visibility ratings:") ||
			strings.HasPrefix(trimmed, "clothing summary:") ||
			strings.HasPrefix(trimmed, "accessories summary:") ||
			strings.HasPrefix(trimmed, "poses summary:") ||
			strings.HasPrefix(trimmed, "finish summary:") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.ToLower(strings.Join(kept, "\n"))
}

func containsWord(text, word string) bool {
	if word == "" {
		return false
	}
	idx := 0
	for {
		pos := strings.Index(text[idx:], word)
		if pos < 0 {
			return false
		}
		pos += idx
		beforeOK := pos == 0 || !isWordChar(rune(text[pos-1]))
		afterPos := pos + len(word)
		afterOK := afterPos >= len(text) || !isWordChar(rune(text[afterPos]))
		if beforeOK && afterOK {
			return true
		}
		idx = pos + len(word)
		if idx >= len(text) {
			return false
		}
	}
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
}

func visionMentionsAccessory(visionDesc, tagName string) bool {
	prose := visionProseWithoutSummaries(visionDesc)
	tagLower := strings.ToLower(strings.TrimSpace(tagName))
	if containsWord(prose, tagLower) {
		return true
	}

	if keys, ok := accessoryTagKeywords[tagLower]; ok {
		for _, key := range keys {
			if containsWord(prose, key) {
				return true
			}
		}
	}

	return false
}

func accessorySummaryMentions(visionDesc, tagName string) bool {
	fields := parseSummaryLine(visionDesc, "Accessories summary:")
	if len(fields) == 0 {
		return false
	}

	for _, key := range accessorySummaryKeys(tagName) {
		value, ok := fields[key]
		if !ok {
			continue
		}
		if summaryValueYes(value) {
			return true
		}
		if summaryValueNo(value) {
			return false
		}
	}

	return false
}

func accessoryExplicitlyDenied(visionDesc, tagName string) bool {
	fields := parseSummaryLine(visionDesc, "Accessories summary:")
	if len(fields) == 0 {
		return false
	}
	for _, key := range accessorySummaryKeys(tagName) {
		if summaryValueNo(fields[key]) {
			return true
		}
	}
	return false
}

func applyAccessoriesFromVision(output *SceneFillOutput, visionDesc string, criteria []CriterionInfo, ctx tagFilterContext) {
	if visionDesc == "" {
		return
	}

	seen := tagSet(output.Tags)

	for _, tag := range collectAccessoryCandidateTags(criteria, ctx) {
		key := strings.ToLower(tag)
		if _, exists := seen[key]; exists {
			continue
		}
		if accessoryExplicitlyDenied(visionDesc, tag) {
			continue
		}
		if !accessorySummaryMentions(visionDesc, tag) && !visionMentionsAccessory(visionDesc, tag) {
			continue
		}
		output.Tags = append(output.Tags, tag)
		seen[key] = struct{}{}
	}
}

var poseTagKeywords = map[string][]string{
	"missionary pose": {"missionary", "missionary position", "face to face", "lying on her back", "lying on back", "man on top"},
	"missionary":      {"missionary"},
	"doggystyle pose": {"doggystyle", "doggy style", "doggy", "from behind", "on all fours"},
	"doggystyle":      {"doggystyle", "doggy style", "from behind"},
	"cowgirl pose":    {"cowgirl", "riding", "woman on top", "straddling"},
	"cowgirl":         {"cowgirl", "riding"},
	"reverse cowgirl": {"reverse cowgirl"},
	"blowjob":         {"blowjob", "blow job", "oral sex", "fellatio", "sucking"},
	"anal":            {"anal sex", "anal penetration"},
}

func visionMentionsPose(visionLower, tagName string) bool {
	tagLower := strings.ToLower(strings.TrimSpace(tagName))
	if strings.Contains(visionLower, tagLower) {
		return true
	}

	if keys, ok := poseTagKeywords[tagLower]; ok {
		for _, key := range keys {
			if strings.Contains(visionLower, key) {
				return true
			}
		}
	}

	// "Missionary pose" -> also match keyword "missionary"
	base := strings.TrimSuffix(tagLower, " pose")
	if base != tagLower && strings.Contains(visionLower, base) {
		return true
	}

	return false
}

func poseSummaryMentions(visionLower, tagName string) bool {
	fields := parseSummaryLine(visionLower, "poses summary:")
	if len(fields) == 0 {
		return false
	}

	tagLower := strings.ToLower(strings.TrimSpace(tagName))
	base := strings.TrimSuffix(tagLower, " pose")

	for key, value := range fields {
		if !summaryValueYes(value) {
			continue
		}
		if key == tagLower || key == base || strings.Contains(tagLower, key) || strings.Contains(key, base) {
			return true
		}
	}
	return false
}

func applyPosesFromVision(output *SceneFillOutput, visionDesc string, ctx tagFilterContext) {
	if visionDesc == "" || len(ctx.poseTags) == 0 {
		return
	}

	visionLower := strings.ToLower(visionDesc)
	seen := tagSet(output.Tags)

	for _, tag := range ctx.poseTags {
		key := strings.ToLower(tag)
		if _, exists := seen[key]; exists {
			continue
		}
		if !visionMentionsPose(visionLower, tag) && !poseSummaryMentions(visionLower, tag) {
			continue
		}
		output.Tags = append(output.Tags, tag)
		seen[key] = struct{}{}
	}
}

func isFinishTagName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if strings.HasPrefix(lower, "cum on ") || strings.HasPrefix(lower, "cum in ") {
		return true
	}
	switch lower {
	case "creampie", "show creampie", "facial", "cumshot", "swallow", "cum swallow",
		"fake sperm", "real sperm", "cum on face", "cum in pussy", "cum in ass":
		return true
	}
	return false
}

func collectFinishCandidateTags(criteria []CriterionInfo, ctx tagFilterContext) []string {
	seen := make(map[string]struct{})
	var tags []string

	add := func(name string) {
		canonical, ok := ctx.tagIndex[strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			return
		}
		if !isFinishTagName(canonical) {
			return
		}
		key := strings.ToLower(canonical)
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		tags = append(tags, canonical)
	}

	for _, c := range criteria {
		text := strings.ToLower(c.Name + " " + c.Description)
		if strings.Contains(text, "creampie") || strings.Contains(text, "cumshot") ||
			strings.Contains(text, "finish") || strings.Contains(text, "cum") {
			for _, name := range c.ExistingTags {
				add(name)
			}
		}
	}

	for name := range ctx.tagIndex {
		if isFinishTagName(name) {
			add(name)
		}
	}

	return tags
}

var finishTypeToTagHints = map[string][]string{
	"creampie":     {"creampie", "show creampie"},
	"internal":     {"creampie", "show creampie"},
	"facial":       {"facial", "cum on face"},
	"cum_on_face":  {"facial", "cum on face"},
	"cum_on_mouth": {"cum on mouth"},
	"cum_in_mouth": {"cum in mouth"},
	"cum_on_tits":  {"cum on tits"},
	"cumshot":      {"cumshot"},
	"swallow":      {"swallow", "cum swallow"},
	"pullout":      {"cumshot"},
}

func parseFinishSummary(desc string) (visible bool, finishType string) {
	fields := parseSummaryLine(desc, "Finish summary:")
	if len(fields) == 0 {
		return false, ""
	}
	if fields["visible"] == "no" {
		return false, ""
	}
	if fields["type"] == "none" {
		return fields["visible"] == "yes", ""
	}
	return true, fields["type"]
}

func extractFinishHintsFromVision(visionLower string) []string {
	var hints []string
	add := func(items ...string) {
		hints = append(hints, items...)
	}

	if strings.Contains(visionLower, "creampie") ||
		strings.Contains(visionLower, "cream pie") ||
		strings.Contains(visionLower, "cum inside") ||
		strings.Contains(visionLower, "internal cum") ||
		strings.Contains(visionLower, "cum dripping from pussy") {
		add("creampie", "show creampie")
	}
	if strings.Contains(visionLower, "facial") || strings.Contains(visionLower, "cum on face") || strings.Contains(visionLower, "cum on her face") {
		add("facial", "cum on face")
	}
	if strings.Contains(visionLower, "cum on mouth") || strings.Contains(visionLower, "cum on lips") {
		add("cum on mouth")
	}
	if strings.Contains(visionLower, "cum in mouth") || strings.Contains(visionLower, "swallow") {
		add("cum in mouth", "swallow")
	}
	if strings.Contains(visionLower, "cumshot") || strings.Contains(visionLower, "ejacul") {
		add("cumshot")
	}
	if strings.Contains(visionLower, "cum on tits") || strings.Contains(visionLower, "cum on breasts") || strings.Contains(visionLower, "cum on boobs") {
		add("cum on tits")
	}

	return hints
}

func textFinishHints(texts ...string) []string {
	var hints []string
	seen := make(map[string]struct{})
	add := func(items ...string) {
		for _, item := range items {
			key := strings.ToLower(item)
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			hints = append(hints, item)
		}
	}

	for _, text := range texts {
		for _, hint := range filenameFinishHints(text) {
			add(hint)
		}
	}
	return hints
}

func filenameFinishHints(filename string) []string {
	lower := strings.ToLower(filename)
	var hints []string
	if strings.Contains(lower, "creampie") {
		hints = append(hints, "creampie")
	}
	if strings.Contains(lower, "facial") {
		hints = append(hints, "facial")
	}
	if strings.Contains(lower, "cum in mouth") || strings.Contains(lower, "cum-in-mouth") {
		hints = append(hints, "cum in mouth")
	}
	if strings.Contains(lower, "cum on mouth") || strings.Contains(lower, "cum-on-mouth") {
		hints = append(hints, "cum on mouth")
	}
	return hints
}

func visionDeniesFinish(visionLower string) bool {
	fields := parseSummaryLine(visionLower, "Finish summary:")
	if fields["visible"] == "no" {
		return true
	}
	return strings.Contains(visionLower, "no visible finish") ||
		strings.Contains(visionLower, "no climax visible")
}

func addFinishTag(output *SceneFillOutput, tag string, seen map[string]struct{}) {
	key := strings.ToLower(strings.TrimSpace(tag))
	if _, exists := seen[key]; exists {
		return
	}
	output.Tags = append(output.Tags, tag)
	seen[key] = struct{}{}
}

func matchFinishHint(hint string, candidates []string) (string, bool) {
	hintLower := strings.ToLower(hint)
	for _, tag := range candidates {
		tagLower := strings.ToLower(tag)
		if tagLower == hintLower || strings.Contains(tagLower, hintLower) || strings.Contains(hintLower, tagLower) {
			return tag, true
		}
	}
	return "", false
}

func applyFinishFromVision(output *SceneFillOutput, visionDesc, filename, title string, criteria []CriterionInfo, ctx tagFilterContext) {
	if visionDesc == "" {
		return
	}

	visionLower := strings.ToLower(visionDesc)
	if visionDeniesFinish(visionLower) {
		return
	}

	ratings := parseVisibilityRatings(visionDesc)
	_, finishType := parseFinishSummary(visionDesc)

	hintSeen := make(map[string]struct{})
	var hints []string
	addHints := func(items ...string) {
		for _, item := range items {
			key := strings.ToLower(item)
			if _, dup := hintSeen[key]; dup {
				continue
			}
			hintSeen[key] = struct{}{}
			hints = append(hints, item)
		}
	}

	if finishType != "" {
		if mapped, ok := finishTypeToTagHints[finishType]; ok {
			addHints(mapped...)
		} else {
			addHints(strings.ReplaceAll(finishType, "_", " "))
		}
	}

	if ratings["creampie"] >= 2 {
		addHints("creampie", "show creampie")
	}
	if ratings["cumshot"] >= 2 {
		addHints("cumshot")
	}
	if ratings["cum_on_mouth"] >= 2 {
		addHints("cum on mouth")
	}

	addHints(extractFinishHintsFromVision(visionLower)...)
	addHints(textFinishHints(filename, title)...)

	if len(hints) == 0 {
		return
	}

	seen := tagSet(output.Tags)
	candidates := collectFinishCandidateTags(criteria, ctx)

	for _, hint := range hints {
		if tag, ok := matchFinishHint(hint, candidates); ok {
			addFinishTag(output, tag, seen)
		}
	}
}

func tagSet(tags []string) map[string]struct{} {
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		seen[strings.ToLower(strings.TrimSpace(tag))] = struct{}{}
	}
	return seen
}

func removeDeniedAccessories(output *SceneFillOutput, visionDesc string) {
	if visionDesc == "" {
		return
	}

	output.Tags = filterTags(output.Tags, func(tag string) bool {
		if accessoryExplicitlyDenied(visionDesc, tag) {
			return false
		}
		tagLower := strings.ToLower(strings.TrimSpace(tag))
		for base := range accessoryTagKeywords {
			if accessoryExplicitlyDenied(visionDesc, base) && strings.Contains(tagLower, base) {
				return false
			}
		}
		return true
	})
}

func applyVisionInferredTags(output *SceneFillOutput, visionDesc, filename, title string, criteria []CriterionInfo, ctx tagFilterContext) {
	removeInvalidClothingNegations(output, visionDesc)
	removeDeniedAccessories(output, visionDesc)
	applyPosesFromVision(output, visionDesc, ctx)
	applyAccessoriesFromVision(output, visionDesc, criteria, ctx)
	applyFinishFromVision(output, visionDesc, filename, title, criteria, ctx)
}
