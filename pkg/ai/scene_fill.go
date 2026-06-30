package ai

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/match"
	"github.com/stashapp/stash/pkg/models"
)

// SceneFillInput contains context passed to the AI for scene metadata extraction.
type SceneFillInput struct {
	Title                   string
	Code                    string
	Details                 string
	Director                string
	Date                    string
	ShootDate               string
	URLs                    []string
	Filename                string
	Performers              []string
	Studio                  string
	TagNames                []string
	PoseTagNames            []string
	Criteria                []CriterionInfo
	VideoPath               string
	VideoDuration           float64
	AIVisionPanelPaths      []string
	HasAIVisionPanels       bool
	SpriteVisionDescription string
}

// CriterionInfo describes a tag requirement category (color preset).
type CriterionInfo struct {
	Name         string
	Color        string
	Description  string
	Required     bool
	ExistingTags []string
}

// SceneFillOutput is the structured JSON returned by the AI.
type SceneFillOutput struct {
	Title         *string               `json:"title"`
	ShootDate     *string               `json:"shoot_date"`
	Date          *string               `json:"date"`
	Performers    []string              `json:"performers"`
	Studio        *string               `json:"studio"`
	Tags          []string              `json:"tags"`
	PerformerTags []PerformerTagsOutput `json:"performer_tags"`
}

// PerformerTagsOutput maps performer-specific tags.
type PerformerTagsOutput struct {
	PerformerName string   `json:"performer_name"`
	Tags          []string `json:"tags"`
}

// FilledSceneResult is the final result after matching to the database.
type FilledSceneResult struct {
	Scene         *models.ScrapedScene
	ShootDate     *string
	PerformerTags []PerformerTagsOutput
}

const systemPrompt = `You are a video metadata extraction assistant. Extract accurate scene metadata using the provided scene information, preview description, and web research results.

CRITICAL RULES:
1. For VISUAL tags (body parts, sex acts, positions, creampie, etc.): the Preview Grid Analysis is the PRIMARY and AUTHORITATIVE source. Web search must NOT override what is clearly visible in the preview.
2. Only report information confirmed by preview description, web research, filename, or existing metadata. NEVER invent or guess data.
3. Prefer official studio/publisher pages and reputable scene databases over user-generated sources for title, performers, studio, and dates only.
4. If sources conflict on visual content, trust the preview grid. If sources conflict on metadata (title/dates), prefer the most authoritative source or omit.
5. If you cannot determine a field with confidence, omit it or set it to null.
6. For title: if the current title is empty or looks like a filename, find the original/correct scene title and cross-check with other sources.
7. Prefer EXACT tag names from the provided category lists. You may suggest a NEW tag only when nothing in the lists fits and the concept is clearly visible in the preview — one short, specific name. Never create variations, synonyms, or numbered enumerations of the same idea.
8. NEVER output long lists of similar tags or guess every possible variant. One visible concept = at most one tag — EXCEPT pose tags: assign every distinct pose shown.
9. RATING TAGS: Many tags use the format "N category" (e.g. "0 tits", "5 tits"). The number is visibility/prominence (0=not shown, higher=more prominent).
   - NEVER include tags starting with "0 " (zero rating). Omit that category — the user will rate it manually.
   - NEVER include tags starting with "no " EXCEPT "no bra", "no pants", and "no underwear" when the preview confirms the performer is fully nude with no such clothing in ALL panels.
   - NEVER include other "no …" tags (e.g. "no creampie", "no cum"). Omit — the user will confirm manually.
   - NEVER include tags starting with "unknown ". Omit — the user will confirm manually.
   - Only assign numbered tags with rating 1 or higher when clearly supported by the preview.
10. CLOTHING / UNDERWEAR: underwear means panties/bra — NOT stockings, garters, or thigh-highs (those are separate accessory/hosiery tags). Assign colored bra/pant/pants tags only when that garment is clearly visible. Assign "no bra", "no pants", "no underwear" ONLY when Clothing summary says fully_nude_all_panels=yes AND bra_visible=no AND underwear_visible=no AND pants_visible=no. Stockings alone do not mean underwear is worn. If any panel shows panties or a bra, do NOT assign the corresponding "no …" tag.
11. ACCESSORIES: assign only accessories the preview clearly shows. Do NOT assign choker unless a choker necklace is explicitly described as visible. Match exact tag names from category lists.
12. Pick at most ONE positive-rated tag per color category when confident. Skip the category entirely if unsure or only 0/no/unknown tags would apply.
13. Include "creampie", "fake sperm", "real sperm", "show creampie" when the preview or Finish summary shows them — especially inspect the LAST panel. Do NOT assign negations.
14. EYE COLOR tags (e.g. "brown eyes", "blue eyes"): assign when the preview analysis states eye color and a matching tag exists in the category list.
15. FINISH / CUMSHOT: inspect the LAST panel and Finish summary first. Assign finish tags when visible — creampie, facial, cum on/in mouth, cumshot, etc. If creampie or cumshot ratings are 2+ or Finish summary visible=yes, assign matching tags from the lists. Filename/title may hint finish type but preview must not contradict it.
16. POSE TAGS: assign EVERY pose tag that appears in ANY panel (missionary, doggystyle, cowgirl, blowjob, etc.). A scene may have multiple poses at different times — include all that are clearly shown. Match exact pose tag names from the Pose Tags list.
17. Fill tag criteria categories only when you can assign a positive-rated or clearly matching tag from the preview analysis.
18. Performer-specific tags go in performer_tags (same rules: no "0 " or "unknown " tags; no "no …" except the clothing tags above when fully nude in all panels).
19. Dates must be YYYY-MM-DD format.
20. Return valid JSON only. Include every tag the preview clearly supports — scenes may legitimately have 50–70+ tags. Keep JSON compact and complete. NEVER pad with guesses or enumerate variants.
21. IMPREGNATE / BIRTH CONTROL: dialogue/plot themes only — NEVER assign from preview images. Assign only when title, details/description, or filename explicitly mentions impregnation or birth control/contraception. NEVER assign both impregnate and birth control on the same scene — they are mutually exclusive.`

// FillScene analyzes a scene and returns AI-generated metadata matched to the database.
func FillScene(ctx context.Context, repo models.Repository, sceneID int) (*FilledSceneResult, error) {
	cfg := config.GetInstance()

	if !cfg.GetAISceneFillEnabled() {
		return nil, fmt.Errorf("AI scene fill is disabled in settings")
	}

	apiKey := cfg.GetAIDeepseekAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is not configured")
	}

	var input SceneFillInput
	var sceneHash string
	var tagFilterCtx tagFilterContext

	if err := repo.WithReadTxn(ctx, func(ctx context.Context) error {
		scene, err := repo.Scene.Find(ctx, sceneID)
		if err != nil {
			return err
		}
		if scene == nil {
			return fmt.Errorf("scene not found")
		}

		if err := scene.LoadRelationships(ctx, repo.Scene); err != nil {
			return err
		}

		input.Title = scene.GetTitle()
		input.Code = scene.Code
		input.Details = scene.Details
		input.Director = scene.Director
		if scene.Date != nil {
			input.Date = scene.Date.String()
		}
		if scene.ShootDate != nil {
			input.ShootDate = scene.ShootDate.String()
		}
		input.URLs = scene.URLs.List()

		for _, sp := range scene.ScenePerformers.List() {
			performer, err := repo.Performer.Find(ctx, sp.PerformerID)
			if err != nil {
				return err
			}
			if performer != nil {
				input.Performers = append(input.Performers, performer.Name)
			}
		}

		if scene.StudioID != nil {
			studio, err := repo.Studio.Find(ctx, *scene.StudioID)
			if err != nil {
				return err
			}
			if studio != nil {
				input.Studio = studio.Name
			}
		}

		for _, tagID := range scene.TagIDs.List() {
			tag, err := repo.Tag.Find(ctx, tagID)
			if err != nil {
				return err
			}
			if tag != nil {
				input.TagNames = append(input.TagNames, tag.Name)
			}
		}

		if scene.Path != "" {
			input.Filename = scene.Path
			input.VideoPath = scene.Path
		} else if len(scene.Files.List()) > 0 {
			primary := scene.Files.List()[0]
			input.Filename = primary.Base().Basename
			input.VideoPath = primary.Path
		}

		if err := scene.LoadPrimaryFile(ctx, repo.File); err == nil {
			if vf := scene.Files.Primary(); vf != nil {
				input.VideoPath = vf.Path
				input.VideoDuration = vf.Duration
			}
		}
		if input.VideoDuration == 0 && len(scene.Files.List()) > 0 {
			input.VideoDuration = scene.Files.List()[0].Duration
		}
		if input.VideoPath == "" && len(scene.Files.List()) > 0 {
			input.VideoPath = scene.Files.List()[0].Path
		}

		sceneHash = scene.GetHash(cfg.GetVideoFileNamingAlgorithm())

		presets, err := repo.ColorPreset.FindAll(ctx)
		if err != nil {
			return err
		}

		allTags, err := repo.Tag.All(ctx)
		if err != nil {
			return err
		}

		tagFilterCtx = buildTagFilterContext(allTags)

		tagsByColor := make(map[string][]string)
		for _, t := range allTags {
			if t.Color != "" {
				colorKey := strings.ToLower(t.Color)
				tagsByColor[colorKey] = append(tagsByColor[colorKey], t.Name)
			}
			if t.IsPoseTag {
				input.PoseTagNames = append(input.PoseTagNames, t.Name)
			}
		}

		for _, preset := range presets {
			if preset.TagRequirementsDescription == "" {
				continue
			}
			input.Criteria = append(input.Criteria, CriterionInfo{
				Name:         preset.Name,
				Color:        preset.Color,
				Description:  preset.TagRequirementsDescription,
				Required:     preset.RequiredForRequirements,
				ExistingTags: tagsByColor[strings.ToLower(preset.Color)],
			})
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if sceneHash != "" {
		paths := manager.GetInstance().Paths
		if paths.Scene.AIVisionPanelsExist(sceneHash) {
			input.AIVisionPanelPaths = paths.Scene.GetAIVisionPanelFilePaths(sceneHash)
			input.HasAIVisionPanels = true
		}
	}

	visionAPIKey := cfg.GetAIVisionAPIKey()
	if cfg.GetAIVisionEnabled() && visionAPIKey != "" {
		if err := manager.EnsureSceneAIVisionPanels(ctx, repo, sceneID); err != nil {
			return nil, fmt.Errorf("generating AI vision panels: %w", err)
		}

		if sceneHash != "" {
			input.AIVisionPanelPaths = manager.GetInstance().Paths.Scene.GetAIVisionPanelFilePaths(sceneHash)
			input.HasAIVisionPanels = true
		}

		visionClient := NewVisionClient(
			visionAPIKey,
			cfg.GetAIVisionAPIURL(),
			cfg.GetAIVisionModel(),
		)
		description, err := visionClient.DescribeSceneVision(ctx, input.AIVisionPanelPaths)
		if err != nil {
			return nil, fmt.Errorf("vision analysis failed: %w", err)
		}
		input.SpriteVisionDescription = description
	}

	userPrompt := buildUserPrompt(input)

	client := NewClient(apiKey, cfg.GetAIDeepseekModel())
	var output SceneFillOutput
	if err := client.ChatJSONWithWebSearch(ctx, systemPrompt, userPrompt, &output); err != nil {
		return nil, err
	}

	filterManualReviewTags(&output)
	applyFullNudityClothingTags(&output, input.SpriteVisionDescription, tagFilterCtx)
	applyVisionInferredTags(&output, input.SpriteVisionDescription, input.Filename, input.Title, input.Criteria, tagFilterCtx)
	filterAITags(&output, tagFilterCtx)
	applyDialogueOnlyTagRules(&output, input.Title, input.Details, input.Filename)

	scraped := buildScrapedScene(output)

	if err := repo.WithReadTxn(ctx, func(ctx context.Context) error {
		rel := match.SceneRelationships{
			PerformerFinder: repo.Performer,
			TagFinder:       repo.Tag,
			StudioFinder:    repo.Studio,
		}
		return rel.MatchRelationships(ctx, scraped, "")
	}); err != nil {
		return nil, err
	}

	return &FilledSceneResult{
		Scene:         scraped,
		ShootDate:     output.ShootDate,
		PerformerTags: output.PerformerTags,
	}, nil
}

func buildUserPrompt(input SceneFillInput) string {
	var b strings.Builder

	b.WriteString("Analyze this scene and extract metadata.\n\n")
	b.WriteString("## Current Scene Data\n")
	b.WriteString(fmt.Sprintf("Title: %s\n", input.Title))
	b.WriteString(fmt.Sprintf("Code: %s\n", input.Code))
	b.WriteString(fmt.Sprintf("Details: %s\n", input.Details))
	b.WriteString(fmt.Sprintf("Director: %s\n", input.Director))
	b.WriteString(fmt.Sprintf("Date: %s\n", input.Date))
	b.WriteString(fmt.Sprintf("Shoot date: %s\n", input.ShootDate))
	b.WriteString(fmt.Sprintf("Filename: %s\n", input.Filename))
	b.WriteString(fmt.Sprintf("URLs: %s\n", strings.Join(input.URLs, ", ")))
	b.WriteString(fmt.Sprintf("Current performers: %s\n", strings.Join(input.Performers, ", ")))
	b.WriteString(fmt.Sprintf("Current studio: %s\n", input.Studio))
	b.WriteString(fmt.Sprintf("Current tags: %s\n", strings.Join(input.TagNames, ", ")))

	if input.SpriteVisionDescription != "" {
		b.WriteString("\n## Preview Grid Analysis\n")
		b.WriteString("The following describes sampled frames from the video (multiple enlarged preview panels, chronological order):\n\n")
		b.WriteString(input.SpriteVisionDescription)
		b.WriteString("\n")
	} else {
		b.WriteString("\n## Preview Grid Analysis\n")
		b.WriteString("No visual preview analysis available. Rely on web search, filename, and existing metadata.\n")
	}

	searchQueries := buildSearchQueries(input)
	if len(searchQueries) > 0 {
		b.WriteString("\n## Suggested Web Search Queries\n")
		b.WriteString("Use web search with these queries (and variations) to find verified metadata:\n")
		for _, q := range searchQueries {
			b.WriteString(fmt.Sprintf("- %s\n", q))
		}
	}

	if len(input.URLs) > 0 {
		b.WriteString("\n## Scene URLs\n")
		b.WriteString("These URLs may identify the scene; search and verify them:\n")
		for _, u := range input.URLs {
			b.WriteString(fmt.Sprintf("- %s\n", u))
		}
	}

	b.WriteString("\n## Tag Criteria Categories\n")
	b.WriteString("Each color category may have at most ONE tag from its list when you are confident.\n")
	b.WriteString("NEVER assign tags starting with \"0 \" or \"unknown \" — skip those for manual review.\n")
	b.WriteString("Assign \"no bra\", \"no pants\", \"no underwear\" only when Clothing summary has fully_nude_all_panels=yes and bra/underwear/pants_visible=all no. Stockings are not underwear.\n")
	b.WriteString("Assign accessories only when clearly visible in preview — check Accessories summary yes values; do not guess choker.\n")
	b.WriteString("Assign finish tags (creampie, facial, cum on/in …) from Finish summary and last panel — creampie/cumshot ratings 2+ count.\n")
	b.WriteString("Assign ALL pose tags shown in any panel (missionary, doggystyle, cowgirl, blowjob, etc.) — multiple poses per scene are normal.\n")
	b.WriteString("impregnate / birth control: only from title, details, or filename — never from preview; never both on the same scene.\n")
	b.WriteString("Prefer exact names from the lists below; suggest a new tag only when nothing fits and the preview clearly supports it.\n")
	b.WriteString("NEVER enumerate variants or synonyms. For \"cum on …\" / \"cum in …\" tags: at most ONE each.\n")
	b.WriteString("Only assign positive ratings (1+) when preview clearly supports them.\n")
	for _, c := range input.Criteria {
		reqStr := "optional"
		if c.Required {
			reqStr = "REQUIRED"
		}
		b.WriteString(fmt.Sprintf("- [%s] %s (color %s): %s\n", reqStr, c.Name, c.Color, c.Description))
		if len(c.ExistingTags) > 0 {
			b.WriteString(fmt.Sprintf("  Pick ONE from: %s\n", formatTagsForPrompt(c.ExistingTags)))
			b.WriteString(formatRatingTagHint(c.ExistingTags))
		}
	}

	if len(input.PoseTagNames) > 0 {
		b.WriteString("\n## Pose Tags (assign ALL that appear in any panel)\n")
		b.WriteString(strings.Join(input.PoseTagNames, ", "))
		b.WriteString("\n")
	}

	b.WriteString("\n## Required Response JSON Format\n")
	b.WriteString(`{
  "title": "original scene title or null if unknown",
  "shoot_date": "YYYY-MM-DD or null",
  "date": "YYYY-MM-DD release date or null",
  "performers": ["performer names"],
  "studio": "studio name or null",
  "tags": ["scene tag names - positive ratings; no bra/pants/underwear only when fully nude in all panels"],
  "performer_tags": [{"performer_name": "name", "tags": ["performer-specific tags"]}]
}`)

	return b.String()
}

func buildSearchQueries(input SceneFillInput) []string {
	seen := make(map[string]struct{})
	var queries []string

	add := func(q string) {
		q = strings.TrimSpace(q)
		if q == "" {
			return
		}
		key := strings.ToLower(q)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		queries = append(queries, q)
	}

	add(input.Code)

	if input.Filename != "" {
		base := filepath.Base(input.Filename)
		ext := filepath.Ext(base)
		add(strings.TrimSuffix(base, ext))
	}

	if input.Title != "" {
		add(input.Title)
	}

	if input.Title != "" && input.Studio != "" {
		add(input.Title + " " + input.Studio)
	}

	if len(input.Performers) > 0 {
		performerQuery := strings.Join(input.Performers, " ")
		add(performerQuery)
		if input.Title != "" {
			add(input.Title + " " + performerQuery)
		}
		if input.Studio != "" {
			add(input.Studio + " " + performerQuery)
		}
	}

	if input.Code != "" && input.Studio != "" {
		add(input.Code + " " + input.Studio)
	}

	return queries
}

// formatRatingTagHint adds guidance when tags use numeric visibility ratings.
func formatRatingTagHint(tags []string) string {
	var rated []string
	for _, t := range tags {
		parts := strings.SplitN(t, " ", 2)
		if len(parts) == 2 && isDigits(parts[0]) {
			rated = append(rated, t)
		}
	}
	if len(rated) == 0 {
		return ""
	}
	return "  (Rating scale: 1+=visible/prominent. Skip 0-rated tags. \"no bra\"/\"no pants\"/\"no underwear\" only when fully nude in all panels.)\n"
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isManualReviewTag reports tags the user prefers to assign themselves (zero ratings, negations).
func isManualReviewTag(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if isAllowedClothingNegationTag(lower) {
		return false
	}
	if strings.HasPrefix(lower, "no ") || strings.HasPrefix(lower, "unknown ") {
		return true
	}
	parts := strings.SplitN(lower, " ", 2)
	return len(parts) == 2 && parts[0] == "0"
}

var allowedClothingNegationTags = map[string]struct{}{
	"no bra":       {},
	"no pants":     {},
	"no underwear": {},
}

func isAllowedClothingNegationTag(lower string) bool {
	_, ok := allowedClothingNegationTags[lower]
	return ok
}

var fullNudityClothingNegations = []struct {
	tagName   string
	conflicts []string
}{
	{"no bra", []string{"bra"}},
	{"no pants", []string{"pants", "pant"}},
	{"no underwear", []string{"underwear", "pant", "pants"}},
}

func visionIndicatesFullyNudeAllPanels(desc string) bool {
	lower := strings.ToLower(desc)
	if strings.Contains(lower, "fully_nude_all_panels=yes") {
		return true
	}
	if strings.Contains(lower, "fully_nude_all_panels=no") ||
		strings.Contains(lower, "fully_nude_all_panels=partial") {
		return false
	}

	clothedPhrases := []string{
		"clothed intro", "starts clothed", "becomes nude", "clothed then nude",
		"partially clothed", "partially nude", "removes clothes", "taking off",
		"stripping", "undressing",
	}
	for _, phrase := range clothedPhrases {
		if strings.Contains(lower, phrase) {
			return false
		}
	}

	nudePhrases := []string{
		"fully nude in all", "completely nude in all", "nude throughout",
		"nude in all panels", "nude in every", "naked in all",
		"no clothing in any", "fully naked throughout",
	}
	for _, phrase := range nudePhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}

	return false
}

func hasConflictingClothingTag(tags []string, conflictWords []string) bool {
	for _, tag := range tags {
		lower := strings.ToLower(strings.TrimSpace(tag))
		if isAllowedClothingNegationTag(lower) {
			continue
		}
		for _, word := range conflictWords {
			if strings.Contains(lower, word) {
				return true
			}
		}
	}
	return false
}

func applyFullNudityClothingTags(output *SceneFillOutput, visionDesc string, ctx tagFilterContext) {
	if visionDesc == "" || !visionAllowsClothingNegation(visionDesc) {
		return
	}

	seen := make(map[string]struct{}, len(output.Tags))
	for _, tag := range output.Tags {
		seen[strings.ToLower(strings.TrimSpace(tag))] = struct{}{}
	}

	for _, candidate := range fullNudityClothingNegations {
		canonical, ok := ctx.tagIndex[strings.ToLower(candidate.tagName)]
		if !ok {
			continue
		}
		key := strings.ToLower(canonical)
		if _, exists := seen[key]; exists {
			continue
		}
		if hasConflictingClothingTag(output.Tags, candidate.conflicts) {
			continue
		}
		output.Tags = append(output.Tags, canonical)
		seen[key] = struct{}{}
	}
}

func filterManualReviewTags(output *SceneFillOutput) {
	output.Tags = filterOutManualReviewTags(output.Tags)
	for i := range output.PerformerTags {
		output.PerformerTags[i].Tags = filterOutManualReviewTags(output.PerformerTags[i].Tags)
	}
}

const (
	maxAISceneTags          = 100
	maxAICumFinishPerPrefix = 1
	maxTagsInPrompt         = 40
	maxAITagNameLen         = 80
)

var cumFinishPrefixes = []string{"cum on ", "cum in "}

type tagFilterContext struct {
	tagIndex      map[string]string
	cumFinishInDB map[string]struct{}
	poseTags      []string
}

// cumFinishBodyTerms — suffix words that make a new "cum on/in …" tag plausible.
var cumFinishBodyTerms = []string{
	"mouth", "face", "tits", "tit", "boob", "breast", "ass", "butt", "pussy",
	"vagina", "stomach", "belly", "tongue", "lip", "chin", "cheek", "eye",
	"hair", "back", "chest", "nipple", "throat", "neck", "hand", "foot", "feet",
	"leg", "arm", "body", "clit", "anus", "forehead", "nose", "ear", "shoulder",
	"hip", "thigh", "knee", "toe", "finger", "wrist", "elbow", "waist", "navel",
	"pubic", "cunt", "areola", "cleavage", "abs", "pec", "glute", "labia",
	"penis", "cock", "dick", "balls", "ballsack",
}

func buildTagFilterContext(allTags []*models.Tag) tagFilterContext {
	ctx := tagFilterContext{
		tagIndex:      make(map[string]string, len(allTags)),
		cumFinishInDB: make(map[string]struct{}),
	}
	for _, t := range allTags {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			continue
		}
		lower := strings.ToLower(name)
		ctx.tagIndex[lower] = name
		if t.IsPoseTag {
			ctx.poseTags = append(ctx.poseTags, name)
		}
		if _, ok := cumFinishPrefix(lower); ok {
			ctx.cumFinishInDB[lower] = struct{}{}
		}
	}
	return ctx
}

func cumFinishPrefix(lower string) (string, bool) {
	for _, prefix := range cumFinishPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return prefix, true
		}
	}
	return "", false
}

func filterAITags(output *SceneFillOutput, ctx tagFilterContext) {
	output.Tags = filterAITagList(output.Tags, ctx)
	for i := range output.PerformerTags {
		output.PerformerTags[i].Tags = filterAITagList(output.PerformerTags[i].Tags, ctx)
	}
}

func filterAITagList(tags []string, ctx tagFilterContext) []string {
	filtered := make([]string, 0, len(tags))
	seen := make(map[string]struct{})
	cumFinishCount := make(map[string]int)

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || len(tag) > maxAITagNameLen {
			continue
		}

		lower := strings.ToLower(tag)
		if canonical, ok := ctx.tagIndex[lower]; ok {
			tag = canonical
			lower = strings.ToLower(tag)
		} else if !isAcceptableNewTag(tag, ctx) {
			continue
		}

		if _, dup := seen[lower]; dup {
			continue
		}

		if prefix, isFinish := cumFinishPrefix(lower); isFinish {
			if cumFinishCount[prefix] >= maxAICumFinishPerPrefix {
				continue
			}
			cumFinishCount[prefix]++
		}

		seen[lower] = struct{}{}
		filtered = append(filtered, tag)
		if len(filtered) >= maxAISceneTags {
			break
		}
	}

	return filtered
}

func isAcceptableNewTag(tag string, ctx tagFilterContext) bool {
	lower := strings.ToLower(strings.TrimSpace(tag))
	if _, ok := cumFinishPrefix(lower); ok {
		return isValidCumFinishTag(lower, ctx)
	}
	// Reject tags that look like abstract/meta enumeration spam.
	if strings.Contains(lower, " multiverse") || strings.Contains(lower, " universe") {
		return false
	}
	return true
}

func isValidCumFinishTag(lower string, ctx tagFilterContext) bool {
	if _, ok := ctx.cumFinishInDB[lower]; ok {
		return true
	}
	prefix, ok := cumFinishPrefix(lower)
	if !ok {
		return false
	}
	suffix := strings.TrimSpace(strings.TrimPrefix(lower, prefix))
	if suffix == "" {
		return false
	}
	for _, term := range cumFinishBodyTerms {
		if suffix == term || strings.Contains(suffix, term) {
			return true
		}
	}
	return false
}

func formatTagsForPrompt(tags []string) string {
	if len(tags) <= maxTagsInPrompt {
		return strings.Join(tags, ", ")
	}

	shown := append([]string(nil), tags[:maxTagsInPrompt]...)
	return strings.Join(shown, ", ") + fmt.Sprintf(" … (%d more — pick at most ONE from this category)", len(tags)-maxTagsInPrompt)
}

func filterOutManualReviewTags(tags []string) []string {
	filtered := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag == "" || isManualReviewTag(tag) {
			continue
		}
		filtered = append(filtered, tag)
	}
	return filtered
}

func buildScrapedScene(output SceneFillOutput) *models.ScrapedScene {
	scraped := &models.ScrapedScene{}

	if output.Title != nil && *output.Title != "" {
		scraped.Title = output.Title
	}
	if output.Date != nil && *output.Date != "" {
		scraped.Date = output.Date
	}

	for _, name := range output.Performers {
		if name == "" {
			continue
		}
		n := name
		scraped.Performers = append(scraped.Performers, &models.ScrapedPerformer{Name: &n})
	}

	for _, tagName := range output.Tags {
		if tagName == "" {
			continue
		}
		n := tagName
		scraped.Tags = append(scraped.Tags, &models.ScrapedTag{Name: n})
	}

	if output.Studio != nil && *output.Studio != "" {
		scraped.Studio = &models.ScrapedStudio{Name: *output.Studio}
	}

	return scraped
}
