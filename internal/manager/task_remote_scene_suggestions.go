package manager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scraper"
	"github.com/stashapp/stash/pkg/utils"
)

const (
	defaultRemoteSceneSuggestionsLimit = 12
	randomQueriesPerProvider           = 2
	popularQuery                       = "popular"
	enrichWorkers                      = 6

	xvideosSearchScraperID  = "builtin_xvideos_search"
	xvideosListingScraperID = "builtin_xvideos_listing"
)

var fallbackSearchQueries = []string{
	"amateur", "anal", "asian", "blonde", "brunette", "casting", "creampie",
	"ebony", "hardcore", "latina", "lesbian", "massage", "milf", "pov",
	"redhead", "threesome",
}

type remoteProviderKey string

const (
	providerPornhub  remoteProviderKey = "pornhub"
	providerXVideos  remoteProviderKey = "xvideos"
	providerXnxx     remoteProviderKey = "xnxx"
	providerXHamster remoteProviderKey = "xhamster"
)

type remoteProviderConfig struct {
	label             string
	scraperNameTokens []string
	urlPatterns       []string
	siteOrigin        string
	searchScraperID   string
	listingScraperID  string
	listingQuery      string
}

var remoteProviders = map[remoteProviderKey]remoteProviderConfig{
	providerPornhub: {
		label:             "PornHub",
		scraperNameTokens: []string{"pornhub"},
		urlPatterns:       []string{"pornhub.com", "pornhub.org"},
		siteOrigin:        "https://www.pornhub.com",
	},
	providerXVideos: {
		label:             "XVideos",
		scraperNameTokens: []string{"xvideos"},
		urlPatterns:       []string{"xvideos.com", "xvideos.red"},
		siteOrigin:        "https://www.xvideos.com",
		searchScraperID:   xvideosSearchScraperID,
		listingScraperID:  xvideosListingScraperID,
		listingQuery:      "best",
	},
	providerXnxx: {
		label:             "Xnxx",
		scraperNameTokens: []string{"xnxx"},
		urlPatterns:       []string{"xnxx.com"},
		siteOrigin:        "https://www.xnxx.com",
	},
	providerXHamster: {
		label:             "XHamster",
		scraperNameTokens: []string{"xhamster"},
		urlPatterns:       []string{"xhamster.com"},
		siteOrigin:        "https://xhamster.com",
	},
}

// RemoteSceneSuggestionsInput is the GraphQL input for fetching remote suggestions.
type RemoteSceneSuggestionsInput struct {
	Providers          []string `json:"providers" gqlgen:"providers"`
	MinDurationMinutes *int     `json:"min_duration_minutes" gqlgen:"min_duration_minutes"`
	Limit              *int     `json:"limit" gqlgen:"limit"`
	PriorityTagIds     []string `json:"priority_tag_ids" gqlgen:"priority_tag_ids"`
	StopTagIds         []string `json:"stop_tag_ids" gqlgen:"stop_tag_ids"`
	ContextTagID       *string  `json:"context_tag_id" gqlgen:"context_tag_id"`
	ContextPerformerID *string  `json:"context_performer_id" gqlgen:"context_performer_id"`
}

type configuredTagRef struct {
	id      string
	name    string
	aliases []string
}

type configuredPerformerRef struct {
	id      string
	name    string
	aliases []string
}

type remoteSuggestionItem struct {
	providerLabel string
	scraperID     string
	scene         *models.ScrapedScene
	priority      bool
}

type providerScraperIDs struct {
	searchScraperID  string
	listingScraperID string
}

type providerQuery struct {
	scraperID string
	query     string
	kind      string // popular | random
}

type remoteSceneSuggestionsTask struct {
	input RemoteSceneSuggestionsInput
	jobID int
}

// StartRemoteSceneSuggestions starts fetching remote suggestions in a separate
// goroutine (JobManager.Start) and returns the job ID immediately.
func (s *Manager) StartRemoteSceneSuggestions(ctx context.Context, input RemoteSceneSuggestionsInput) int {
	jobIDReady := make(chan int, 1)
	jobID := s.JobManager.Start(ctx, "Fetching remote scene suggestions", job.MakeJobExec(func(ctx context.Context, progress *job.Progress) error {
		id := <-jobIDReady
		task := &remoteSceneSuggestionsTask{input: input, jobID: id}
		err := task.Execute(ctx, progress)
		if err != nil {
			// Still store empty results so the client can finish cleanly.
			storeRemoteSceneSuggestionResults(id, nil)
		}
		return err
	}))
	jobIDReady <- jobID
	return jobID
}

func (t *remoteSceneSuggestionsTask) Execute(ctx context.Context, progress *job.Progress) error {
	progress.Indefinite()

	limit := defaultRemoteSceneSuggestionsLimit
	if t.input.Limit != nil && *t.input.Limit > 0 {
		limit = *t.input.Limit
	}
	providerLimit := int(math.Max(3, math.Ceil(float64(limit)*2.0/3.0)))

	minMinutes := 10
	if t.input.MinDurationMinutes != nil {
		minMinutes = *t.input.MinDurationMinutes
	}
	if minMinutes < 0 {
		minMinutes = 0
	}
	minSeconds := minMinutes * 60

	providers := t.input.Providers
	if len(providers) == 0 {
		providers = []string{
			string(providerPornhub),
			string(providerXVideos),
			string(providerXnxx),
			string(providerXHamster),
		}
	}

	cache := instance.ScraperCache
	if cache == nil {
		storeRemoteSceneSuggestionResults(t.jobID, nil)
		return nil
	}

	scrapers := cache.ListScrapers([]scraper.ScrapeContentType{scraper.ScrapeContentTypeScene})

	priorityTags, err := t.loadConfiguredTags(ctx, t.input.PriorityTagIds)
	if err != nil {
		logger.Warnf("remote suggestions: load priority tags: %v", err)
	}
	stopTags, err := t.loadConfiguredTags(ctx, t.input.StopTagIds)
	if err != nil {
		logger.Warnf("remote suggestions: load stop tags: %v", err)
	}

	var contextTag *configuredTagRef
	if t.input.ContextTagID != nil && *t.input.ContextTagID != "" {
		tags, loadErr := t.loadConfiguredTags(ctx, []string{*t.input.ContextTagID})
		if loadErr != nil {
			logger.Warnf("remote suggestions: load context tag: %v", loadErr)
		} else if len(tags) > 0 {
			contextTag = &tags[0]
		}
	}

	var contextPerformer *configuredPerformerRef
	if t.input.ContextPerformerID != nil && *t.input.ContextPerformerID != "" {
		p, loadErr := t.loadConfiguredPerformer(ctx, *t.input.ContextPerformerID)
		if loadErr != nil {
			logger.Warnf("remote suggestions: load context performer: %v", loadErr)
		} else {
			contextPerformer = p
		}
	}

	type enabledProvider struct {
		key        remoteProviderKey
		label      string
		scraperIDs providerScraperIDs
	}

	var enabled []enabledProvider
	for _, raw := range providers {
		key := remoteProviderKey(strings.ToLower(strings.TrimSpace(raw)))
		cfg, ok := remoteProviders[key]
		if !ok {
			continue
		}
		ids := resolveProviderScrapers(key, cfg, scrapers)
		if ids.searchScraperID == "" && ids.listingScraperID == "" {
			continue
		}
		enabled = append(enabled, enabledProvider{key: key, label: cfg.label, scraperIDs: ids})
	}

	if len(enabled) == 0 {
		storeRemoteSceneSuggestionResults(t.jobID, nil)
		return nil
	}

	preferredQueries := preferredSearchQueries(priorityTags)
	contextOnly := contextTag != nil || contextPerformer != nil
	searchQueries := t.buildSearchQueries(ctx, contextTag, contextPerformer, preferredQueries, len(enabled)*randomQueriesPerProvider)
	// For performer/tag pages, search only by that context — never mix in
	// "popular"/priority queries, or results get filtered to empty.
	queryPreferred := preferredQueries
	if contextOnly {
		queryPreferred = searchQueries
	}

	type providerResult struct {
		items []remoteSuggestionItem
	}

	results := make([]providerResult, len(enabled))
	var wg sync.WaitGroup

	for i, ep := range enabled {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func(idx int, ep enabledProvider) {
			defer wg.Done()
			if ctx.Err() != nil {
				return
			}
			items := t.scrapeProvider(ctx, ep.key, ep.label, ep.scraperIDs, searchQueries, queryPreferred, contextOnly, idx, providerLimit, minSeconds, stopTags, contextTag, contextPerformer, priorityTags)
			results[idx] = providerResult{items: items}
		}(i, ep)
	}
	wg.Wait()

	if ctx.Err() != nil {
		storeRemoteSceneSuggestionResults(t.jobID, nil)
		return ctx.Err()
	}

	merged := make([]remoteSuggestionItem, 0)
	for _, r := range results {
		merged = append(merged, r.items...)
	}

	byURL := make(map[string]remoteSuggestionItem)
	for _, item := range merged {
		url := sceneSourceURL(item.scene)
		if url == "" {
			continue
		}
		if _, exists := byURL[url]; !exists {
			byURL[url] = item
		}
	}

	candidates := make([]remoteSuggestionItem, 0, len(byURL))
	for _, item := range byURL {
		candidates = append(candidates, item)
	}
	candidates = t.filterExistingSceneURLs(ctx, candidates)
	shuffleRemoteSuggestions(candidates)

	// Priority/stop/context matching needs page tags. Enrich a larger pool first,
	// then select — truncating before enrich made priority almost never apply.
	needsDetailEnrich := len(priorityTags) > 0 || len(stopTags) > 0 || contextTag != nil || contextPerformer != nil
	enrichPool := candidates
	if !needsDetailEnrich {
		if len(enrichPool) > limit {
			enrichPool = enrichPool[:limit]
		}
	} else {
		poolCap := limit * 4
		if poolCap < 24 {
			poolCap = 24
		}
		if poolCap > 48 {
			poolCap = 48
		}
		if len(enrichPool) > poolCap {
			enrichPool = enrichPool[:poolCap]
		}
	}

	progress.SetTotal(len(enrichPool))

	type enrichOutcome struct {
		item *remoteSuggestionItem
	}
	outcomes := make([]enrichOutcome, len(enrichPool))
	var enrichWG sync.WaitGroup
	sem := make(chan struct{}, enrichWorkers)

	for i, item := range enrichPool {
		if ctx.Err() != nil {
			break
		}
		enrichWG.Add(1)
		go func(idx int, item remoteSuggestionItem) {
			defer enrichWG.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			progress.ExecuteTask("Enriching suggestion", func() {
				out := t.enrichSuggestion(ctx, item, minSeconds, needsDetailEnrich)
				if out == nil {
					return
				}
				if sceneMatchesConfiguredTags(out.scene, stopTags) {
					return
				}
				if contextTag != nil && !sceneMatchesConfiguredTags(out.scene, []configuredTagRef{*contextTag}) {
					return
				}
				if contextPerformer != nil && !sceneMatchesConfiguredPerformer(out.scene, *contextPerformer) {
					return
				}
				out.priority = sceneMatchesConfiguredTags(out.scene, priorityTags)
				outcomes[idx] = enrichOutcome{item: out}
			})
			progress.Increment()
		}(i, item)
	}
	enrichWG.Wait()

	enriched := make([]remoteSuggestionItem, 0, len(enrichPool))
	for _, outcome := range outcomes {
		if outcome.item != nil {
			enriched = append(enriched, *outcome.item)
		}
	}

	if ctx.Err() != nil {
		storeRemoteSceneSuggestionResults(t.jobID, nil)
		return ctx.Err()
	}

	priorityItems := make([]remoteSuggestionItem, 0)
	regularItems := make([]remoteSuggestionItem, 0)
	for _, item := range enriched {
		if item.priority {
			priorityItems = append(priorityItems, item)
		} else {
			regularItems = append(regularItems, item)
		}
	}
	shuffleRemoteSuggestions(priorityItems)
	shuffleRemoteSuggestions(regularItems)

	final := append(priorityItems, regularItems...)
	if len(final) > limit {
		final = final[:limit]
	}

	// Search/enrich keep remote CDN URLs for speed; browsers can't load them
	// (hotlink protection). Materialize only the final carousel set to base64.
	materializeSuggestionImages(ctx, final)

	scenes := make([]*models.ScrapedScene, 0, len(final))
	for _, item := range final {
		scenes = append(scenes, item.scene)
	}

	storeRemoteSceneSuggestionResults(t.jobID, scenes)
	return nil
}

// filterExistingSceneURLs drops candidates whose source URL is already
// attached to a scene in the library.
func (t *remoteSceneSuggestionsTask) filterExistingSceneURLs(ctx context.Context, items []remoteSuggestionItem) []remoteSuggestionItem {
	urlSet := make(map[string]struct{}, len(items))
	patterns := make([]string, 0, len(items))
	for _, item := range items {
		url := sceneSourceURL(item.scene)
		if url == "" {
			continue
		}
		if _, ok := urlSet[url]; ok {
			continue
		}
		urlSet[url] = struct{}{}
		patterns = append(patterns, regexp.QuoteMeta(url))
	}
	if len(patterns) == 0 {
		return items
	}

	existing := make(map[string]struct{})
	perPage := -1
	err := instance.Repository.WithReadTxn(ctx, func(ctx context.Context) error {
		result, err := instance.Repository.Scene.Query(ctx, models.SceneQueryOptions{
			QueryOptions: models.QueryOptions{
				FindFilter: &models.FindFilterType{PerPage: &perPage},
			},
			SceneFilter: &models.SceneFilterType{
				URL: &models.StringCriterionInput{
					Value:    "^(" + strings.Join(patterns, "|") + ")$",
					Modifier: models.CriterionModifierMatchesRegex,
				},
			},
		})
		if err != nil {
			return err
		}

		for _, id := range result.IDs {
			urls, err := instance.Repository.Scene.GetURLs(ctx, id)
			if err != nil {
				return err
			}
			for _, u := range urls {
				if _, ok := urlSet[u]; ok {
					existing[u] = struct{}{}
				}
			}
		}
		return nil
	})
	if err != nil {
		logger.Warnf("remote suggestions: filter existing scene urls: %v", err)
		return items
	}
	if len(existing) == 0 {
		return items
	}

	filtered := make([]remoteSuggestionItem, 0, len(items))
	for _, item := range items {
		url := sceneSourceURL(item.scene)
		if url != "" {
			if _, ok := existing[url]; ok {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func (t *remoteSceneSuggestionsTask) scrapeProvider(
	ctx context.Context,
	key remoteProviderKey,
	label string,
	scraperIDs providerScraperIDs,
	searchQueries []string,
	preferredQueries []string,
	contextOnly bool,
	providerIndex int,
	providerLimit int,
	minSeconds int,
	stopTags []configuredTagRef,
	contextTag *configuredTagRef,
	contextPerformer *configuredPerformerRef,
	priorityTags []configuredTagRef,
) []remoteSuggestionItem {
	cfg := remoteProviders[key]
	queries := buildProviderQueries(key, cfg, scraperIDs, searchQueries, preferredQueries, contextOnly, providerIndex)
	if len(queries) == 0 {
		return nil
	}

	var popularScenes, randomScenes []*models.ScrapedScene
	for _, q := range queries {
		if ctx.Err() != nil {
			return nil
		}
		scenes, err := scrapeScenesByName(ctx, q.scraperID, q.query)
		if err != nil {
			logger.Debugf("remote suggestions: scrape %s %q: %v", q.scraperID, q.query, err)
			continue
		}
		for _, scene := range scenes {
			normalized := normalizeScrapedScene(scene, cfg.siteOrigin)
			if q.kind == "popular" {
				popularScenes = append(popularScenes, normalized)
			} else {
				randomScenes = append(randomScenes, normalized)
			}
		}
	}

	// Do not require context tag/performer match here: search hits rarely include
	// those relationships until URL enrich. Filtering now yields an empty set.
	passes := func(scene *models.ScrapedScene) bool {
		if sceneSourceURL(scene) == "" {
			return false
		}
		if sceneMatchesConfiguredTags(scene, stopTags) {
			return false
		}
		if dur, ok := getSceneDurationSeconds(scene); ok && minSeconds > 0 && dur < float64(minSeconds) {
			return false
		}
		return true
	}

	halfLimit := int(math.Ceil(float64(providerLimit) / 2))
	popularPicked := filterAndShuffleScenes(popularScenes, passes, halfLimit)
	randomPicked := filterAndShuffleScenes(randomScenes, passes, providerLimit-len(popularPicked))
	combined := append(popularPicked, randomPicked...)

	if len(combined) < providerLimit {
		used := make(map[string]struct{}, len(combined))
		for _, scene := range combined {
			used[sceneSourceURL(scene)] = struct{}{}
		}
		leftovers := make([]*models.ScrapedScene, 0)
		for _, scene := range append(popularScenes, randomScenes...) {
			url := sceneSourceURL(scene)
			if url == "" {
				continue
			}
			if _, ok := used[url]; ok {
				continue
			}
			if !passes(scene) {
				continue
			}
			leftovers = append(leftovers, scene)
		}
		shuffleScenes(leftovers)
		need := providerLimit - len(combined)
		if len(leftovers) > need {
			leftovers = leftovers[:need]
		}
		combined = append(combined, leftovers...)
	}

	shuffleScenes(combined)
	items := make([]remoteSuggestionItem, 0, len(combined))
	scraperID := scraperIDs.searchScraperID
	if scraperID == "" {
		scraperID = scraperIDs.listingScraperID
	}
	for _, scene := range combined {
		items = append(items, remoteSuggestionItem{
			providerLabel: label,
			scraperID:     scraperID,
			scene:         scene,
			priority:      sceneMatchesConfiguredTags(scene, priorityTags),
		})
	}
	return items
}

func (t *remoteSceneSuggestionsTask) enrichSuggestion(
	ctx context.Context,
	item remoteSuggestionItem,
	minSeconds int,
	fetchDetails bool,
) *remoteSuggestionItem {
	sourceURL := sceneSourceURL(item.scene)
	if sourceURL == "" {
		return nil
	}

	merged := cloneScrapedScene(item.scene)
	clearPlaceholderSceneImage(merged)

	if _, ok := getSceneDurationSeconds(merged); !ok {
		if duration, err := ResolveEmbedDuration(sourceURL); err == nil && duration > 0 {
			d := int(math.Round(duration))
			merged.Duration = &d
		}
	}

	if !passesMinDuration(merged, minSeconds) {
		return nil
	}

	needsURLScrape := fetchDetails ||
		merged.Title == nil || strings.TrimSpace(stringPtrValue(merged.Title)) == "" ||
		merged.Image == nil || strings.TrimSpace(stringPtrValue(merged.Image)) == ""

	if needsURLScrape {
		if enriched, err := scrapeSceneURLSkipImages(ctx, sourceURL); err == nil && enriched != nil {
			merged = mergeScrapedScene(enriched, merged, sourceURL)
			clearPlaceholderSceneImage(merged)
		}
	}

	if !passesMinDuration(merged, minSeconds) {
		return nil
	}

	return &remoteSuggestionItem{
		providerLabel: item.providerLabel,
		scraperID:     item.scraperID,
		scene:         merged,
		priority:      item.priority,
	}
}

func stringPtrValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func clearPlaceholderSceneImage(scene *models.ScrapedScene) {
	if scene == nil || scene.Image == nil {
		return
	}
	if !isUsableRemoteImageURL(scene.Image) {
		scene.Image = nil
	}
}

func isUsableRemoteImageURL(img *string) bool {
	if img == nil {
		return false
	}
	value := strings.TrimSpace(*img)
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "data:") {
		return true
	}
	lower := strings.ToLower(value)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return false
	}
	if strings.Contains(lower, "blank.gif") ||
		strings.Contains(lower, "lightbox-blank") ||
		strings.Contains(lower, "thumbnum") ||
		strings.Contains(lower, "javascript:") {
		return false
	}
	return true
}

func materializeSuggestionImages(ctx context.Context, items []remoteSuggestionItem) {
	if instance.ScraperCache == nil {
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, enrichWorkers)

	for i := range items {
		scene := items[i].scene
		if scene == nil || !isUsableRemoteImageURL(scene.Image) {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(*scene.Image), "data:") {
			continue
		}

		referer := sceneSourceURL(scene)
		wg.Add(1)
		go func(s *models.ScrapedScene, ref string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			if ctx.Err() != nil {
				return
			}
			if err := downloadRemoteSceneImage(ctx, s, ref); err != nil {
				logger.Debugf("remote suggestions: download image: %v", err)
				// Drop hotlink-blocked URLs so the UI doesn't show a broken icon.
				if s.Image != nil && strings.HasPrefix(*s.Image, "http") {
					s.Image = nil
				}
			}
		}(scene, referer)
	}
	wg.Wait()
}

func (t *remoteSceneSuggestionsTask) buildSearchQueries(
	ctx context.Context,
	contextTag *configuredTagRef,
	contextPerformer *configuredPerformerRef,
	preferredQueries []string,
	count int,
) []string {
	if contextTag != nil {
		queries := []string{}
		if strings.TrimSpace(contextTag.name) != "" {
			queries = append(queries, strings.TrimSpace(contextTag.name))
		}
		for _, alias := range contextTag.aliases {
			if strings.TrimSpace(alias) != "" {
				queries = append(queries, strings.TrimSpace(alias))
			}
		}
		shuffleStrings(queries)
		return queries
	}

	if contextPerformer != nil {
		queries := []string{}
		if strings.TrimSpace(contextPerformer.name) != "" {
			queries = append(queries, strings.TrimSpace(contextPerformer.name))
		}
		for _, alias := range contextPerformer.aliases {
			if strings.TrimSpace(alias) != "" {
				queries = append(queries, strings.TrimSpace(alias))
			}
		}
		shuffleStrings(queries)
		return uniqueNonEmptyStrings(queries)
	}

	queries := loadRandomSearchQueries(ctx, count)
	if len(preferredQueries) == 0 {
		return queries
	}

	// Prefer configured priority tag names so remote results are more likely to match.
	combined := append([]string{}, preferredQueries...)
	combined = append(combined, queries...)
	return uniqueNonEmptyStrings(combined)
}

func preferredSearchQueries(tags []configuredTagRef) []string {
	queries := make([]string, 0, len(tags)*2)
	seen := make(map[string]struct{})
	add := func(raw string) {
		q := strings.TrimSpace(raw)
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
	for _, tag := range tags {
		add(tag.name)
		for _, alias := range tag.aliases {
			add(alias)
		}
	}
	shuffleStrings(queries)
	return queries
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func (t *remoteSceneSuggestionsTask) loadConfiguredTags(ctx context.Context, ids []string) ([]configuredTagRef, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	intIDs := make([]int, 0, len(ids))
	for _, id := range ids {
		n, err := strconv.Atoi(id)
		if err != nil {
			continue
		}
		intIDs = append(intIDs, n)
	}
	if len(intIDs) == 0 {
		return nil, nil
	}

	var tags []*models.Tag
	err := instance.Repository.WithReadTxn(ctx, func(ctx context.Context) error {
		var err error
		tags, err = instance.Repository.Tag.FindMany(ctx, intIDs)
		if err != nil {
			return err
		}
		for _, tag := range tags {
			if tag == nil {
				continue
			}
			_ = tag.LoadAliases(ctx, instance.Repository.Tag)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	refs := make([]configuredTagRef, 0, len(tags))
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		refs = append(refs, configuredTagRef{
			id:      strconv.Itoa(tag.ID),
			name:    tag.Name,
			aliases: tag.Aliases.List(),
		})
	}
	return refs, nil
}

func (t *remoteSceneSuggestionsTask) loadConfiguredPerformer(ctx context.Context, id string) (*configuredPerformerRef, error) {
	intID, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	var performer *models.Performer
	err = instance.Repository.WithReadTxn(ctx, func(ctx context.Context) error {
		var err error
		performer, err = instance.Repository.Performer.Find(ctx, intID)
		if err != nil || performer == nil {
			return err
		}
		return performer.LoadAliases(ctx, instance.Repository.Performer)
	})
	if err != nil || performer == nil {
		return nil, err
	}

	return &configuredPerformerRef{
		id:      strconv.Itoa(performer.ID),
		name:    performer.Name,
		aliases: performer.Aliases.List(),
	}, nil
}

func loadRandomSearchQueries(ctx context.Context, count int) []string {
	queries := make([]string, 0, count*3)
	sortBy := "random"
	perPage := count * 2
	if perPage < 12 {
		perPage = 12
	}
	findFilter := &models.FindFilterType{
		PerPage: &perPage,
		Sort:    &sortBy,
	}

	_ = instance.Repository.WithReadTxn(ctx, func(ctx context.Context) error {
		performers, _, err := instance.Repository.Performer.Query(ctx, nil, findFilter)
		if err == nil {
			for _, p := range performers {
				if p != nil && strings.TrimSpace(p.Name) != "" {
					queries = append(queries, strings.TrimSpace(p.Name))
				}
			}
		}
		tags, _, err := instance.Repository.Tag.Query(ctx, nil, findFilter)
		if err == nil {
			for _, tag := range tags {
				if tag != nil && strings.TrimSpace(tag.Name) != "" {
					queries = append(queries, strings.TrimSpace(tag.Name))
				}
			}
		}
		return nil
	})

	fallback := append([]string{}, fallbackSearchQueries...)
	shuffleStrings(fallback)
	for _, q := range fallback {
		if len(queries) >= count*3 {
			break
		}
		queries = append(queries, q)
	}

	shuffleStrings(queries)
	return queries
}

func resolveProviderScrapers(key remoteProviderKey, cfg remoteProviderConfig, scrapers []*scraper.Scraper) providerScraperIDs {
	result := providerScraperIDs{}

	supportsName := func(s *scraper.Scraper) bool {
		if s == nil || s.Scene == nil {
			return false
		}
		for _, t := range s.Scene.SupportedScrapes {
			if t == scraper.ScrapeTypeName {
				return true
			}
		}
		return false
	}

	findByID := func(id string) *scraper.Scraper {
		for _, s := range scrapers {
			if s != nil && s.ID == id {
				return s
			}
		}
		return nil
	}

	if cfg.searchScraperID != "" {
		if s := findByID(cfg.searchScraperID); supportsName(s) {
			result.searchScraperID = s.ID
		}
	}
	if cfg.listingScraperID != "" {
		if s := findByID(cfg.listingScraperID); supportsName(s) {
			result.listingScraperID = s.ID
		}
	}

	if result.searchScraperID == "" {
		for _, s := range scrapers {
			if !supportsName(s) {
				continue
			}
			if s.ID == cfg.listingScraperID {
				continue
			}
			if matchesProviderScraper(s, cfg) {
				result.searchScraperID = s.ID
				break
			}
		}
	}

	return result
}

func matchesProviderScraper(s *scraper.Scraper, cfg remoteProviderConfig) bool {
	name := strings.ToLower(s.Name)
	id := strings.ToLower(s.ID)
	for _, token := range cfg.scraperNameTokens {
		if strings.Contains(name, token) || strings.Contains(id, token) {
			return true
		}
	}
	if s.Scene != nil {
		for _, url := range s.Scene.Urls {
			lower := strings.ToLower(url)
			for _, pattern := range cfg.urlPatterns {
				if strings.Contains(lower, pattern) {
					return true
				}
			}
		}
	}
	return false
}

func buildProviderQueries(
	key remoteProviderKey,
	cfg remoteProviderConfig,
	scraperIDs providerScraperIDs,
	searchQueries []string,
	preferredQueries []string,
	contextOnly bool,
	providerIndex int,
) []providerQuery {
	queries := make([]providerQuery, 0)

	randomQueries := make([]string, 0, randomQueriesPerProvider)

	preferred := append([]string{}, preferredQueries...)
	shuffleStrings(preferred)
	for _, q := range preferred {
		if len(randomQueries) >= randomQueriesPerProvider {
			break
		}
		if strings.TrimSpace(q) == "" {
			continue
		}
		randomQueries = append(randomQueries, strings.TrimSpace(q))
	}

	shuffled := append([]string{}, searchQueries...)
	shuffleStrings(shuffled)
	for _, q := range shuffled {
		if len(randomQueries) >= randomQueriesPerProvider {
			break
		}
		if strings.TrimSpace(q) == "" {
			continue
		}
		// Skip duplicates already taken from preferred queries.
		dup := false
		for _, existing := range randomQueries {
			if strings.EqualFold(existing, q) {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		randomQueries = append(randomQueries, strings.TrimSpace(q))
	}

	if !contextOnly {
		for len(randomQueries) < randomQueriesPerProvider {
			if len(fallbackSearchQueries) > 0 {
				randomQueries = append(randomQueries, fallbackSearchQueries[rand.Intn(len(fallbackSearchQueries))])
			} else {
				randomQueries = append(randomQueries, "video "+strconv.Itoa(providerIndex+len(randomQueries)+1))
			}
		}
	}

	if len(randomQueries) == 0 {
		return queries
	}

	searchID := scraperIDs.searchScraperID
	if searchID == "" {
		searchID = scraperIDs.listingScraperID
	}
	if searchID == "" {
		return queries
	}

	if contextOnly {
		// Performer/tag context: only query for that name/aliases.
		for _, q := range randomQueries {
			queries = append(queries, providerQuery{
				scraperID: searchID,
				query:     q,
				kind:      "random",
			})
		}
		return queries
	}

	if key == providerXVideos {
		if scraperIDs.listingScraperID != "" {
			listingQuery := cfg.listingQuery
			if listingQuery == "" {
				listingQuery = "best"
			}
			queries = append(queries, providerQuery{
				scraperID: scraperIDs.listingScraperID,
				query:     listingQuery,
				kind:      "popular",
			})
		}
		if scraperIDs.searchScraperID != "" {
			for _, q := range randomQueries {
				queries = append(queries, providerQuery{
					scraperID: scraperIDs.searchScraperID,
					query:     q,
					kind:      "random",
				})
			}
		}
		return queries
	}

	if scraperIDs.searchScraperID == "" {
		return queries
	}

	popular := popularQuery
	if key == providerXHamster {
		// "popular" as a search term returns very few results on XHamster.
		if len(preferred) > 0 {
			popular = preferred[0]
		} else {
			popular = "amateur"
		}
	} else if len(preferred) > 0 && providerIndex%2 == 1 {
		// Alternate popular slot with a priority tag search for more matches.
		popular = preferred[providerIndex%len(preferred)]
	}

	queries = append(queries, providerQuery{
		scraperID: scraperIDs.searchScraperID,
		query:     popular,
		kind:      "popular",
	})
	for _, q := range randomQueries {
		queries = append(queries, providerQuery{
			scraperID: scraperIDs.searchScraperID,
			query:     q,
			kind:      "random",
		})
	}
	return queries
}

func scrapeScenesByName(ctx context.Context, scraperID, query string) ([]*models.ScrapedScene, error) {
	if query == "" || scraperID == "" || instance.ScraperCache == nil {
		return nil, nil
	}
	if instance.ScraperCache.GetScraper(scraperID) == nil {
		logger.Debugf("remote suggestions: scraper %s not found, skipping", scraperID)
		return nil, nil
	}

	content, err := instance.ScraperCache.ScrapeName(ctx, scraperID, query, scraper.ScrapeContentTypeScene)
	if err != nil {
		if errors.Is(err, scraper.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	scenes := scrapedScenesFromContent(content)
	for _, scene := range scenes {
		clearPlaceholderSceneImage(scene)
	}
	return scenes, nil
}

func scrapeSceneURLSkipImages(ctx context.Context, url string) (*models.ScrapedScene, error) {
	if strings.TrimSpace(url) == "" || instance.ScraperCache == nil {
		return nil, nil
	}

	content, err := instance.ScraperCache.ScrapeURL(ctx, url, scraper.ScrapeContentTypeScene)
	if err != nil {
		if errors.Is(err, scraper.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if content == nil {
		return nil, nil
	}
	scenes := scrapedScenesFromContent([]scraper.ScrapedContent{content})
	if len(scenes) == 0 {
		return nil, nil
	}
	clearPlaceholderSceneImage(scenes[0])
	return scenes[0], nil
}

func downloadRemoteSceneImage(ctx context.Context, s *models.ScrapedScene, referer string) error {
	if s == nil || s.Image == nil {
		return nil
	}

	imageURL := strings.TrimSpace(*s.Image)
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		return nil
	}

	imageURL = utils.SanitizeImageURL(imageURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return err
	}

	userAgent := ""
	if instance.Config != nil {
		userAgent = instance.Config.GetScraperUserAgent()
	}
	if userAgent == "" {
		userAgent = utils.GetUserAgent()
	}
	req.Header.Set("User-Agent", userAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	} else if req.URL.Scheme != "" {
		req.Header.Set("Referer", req.URL.Scheme+"://"+req.Host+"/")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("http error %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}

	img := "data:" + contentType + ";base64," + utils.GetBase64StringFromData(body)
	s.Image = &img
	return nil
}

func scrapedScenesFromContent(content []scraper.ScrapedContent) []*models.ScrapedScene {
	ret := make([]*models.ScrapedScene, 0, len(content))
	for _, c := range content {
		if c == nil {
			continue
		}
		switch s := c.(type) {
		case *models.ScrapedScene:
			ret = append(ret, s)
		case models.ScrapedScene:
			cp := s
			ret = append(ret, &cp)
		}
	}
	return ret
}

func normalizeSceneURL(url, siteOrigin string) string {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return trimmed
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "//") {
		return "https:" + trimmed
	}
	if strings.HasPrefix(trimmed, "/") {
		return siteOrigin + trimmed
	}
	return siteOrigin + "/" + trimmed
}

func normalizeScrapedScene(scene *models.ScrapedScene, siteOrigin string) *models.ScrapedScene {
	if scene == nil {
		return nil
	}
	out := cloneScrapedScene(scene)
	rawURL := sceneSourceURL(out)
	normalized := normalizeSceneURL(rawURL, siteOrigin)
	if normalized == "" {
		return out
	}

	urls := out.URLs
	if len(urls) == 0 && out.URL != nil {
		urls = []string{*out.URL}
	}
	normalizedURLs := make([]string, 0, len(urls))
	seen := make(map[string]struct{})
	for _, u := range urls {
		n := normalizeSceneURL(u, siteOrigin)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		normalizedURLs = append(normalizedURLs, n)
	}
	if len(normalizedURLs) == 0 {
		normalizedURLs = []string{normalized}
	}
	out.URL = &normalized
	out.URLs = normalizedURLs
	return out
}

func cloneScrapedScene(scene *models.ScrapedScene) *models.ScrapedScene {
	if scene == nil {
		return nil
	}
	cp := *scene
	if scene.URLs != nil {
		cp.URLs = append([]string{}, scene.URLs...)
	}
	if scene.Tags != nil {
		cp.Tags = append([]*models.ScrapedTag{}, scene.Tags...)
	}
	if scene.Performers != nil {
		cp.Performers = append([]*models.ScrapedPerformer{}, scene.Performers...)
	}
	return &cp
}

func mergeScrapedScene(fromURL, fallback *models.ScrapedScene, sourceURL string) *models.ScrapedScene {
	if fromURL == nil {
		return fallback
	}
	merged := cloneScrapedScene(fromURL)
	clearPlaceholderSceneImage(merged)
	// Prefer a usable search thumbnail when URL-scrape image is missing/invalid.
	// Also prefer the search thumb when both are remote URLs: listing thumbs are
	// usually more reliable CDN paths than og:image variants.
	if fallback != nil && isUsableRemoteImageURL(fallback.Image) {
		if !isUsableRemoteImageURL(merged.Image) {
			merged.Image = fallback.Image
		}
	}
	if _, ok := getSceneDurationSeconds(merged); !ok {
		if dur, ok := getSceneDurationSeconds(fallback); ok {
			d := int(math.Round(dur))
			merged.Duration = &d
		}
	}
	if len(merged.URLs) == 0 {
		if fallback != nil && len(fallback.URLs) > 0 {
			merged.URLs = append([]string{}, fallback.URLs...)
		} else {
			merged.URLs = []string{sourceURL}
		}
	}
	return merged
}

func sceneSourceURL(scene *models.ScrapedScene) string {
	if scene == nil {
		return ""
	}
	if len(scene.URLs) > 0 && strings.TrimSpace(scene.URLs[0]) != "" {
		return strings.TrimSpace(scene.URLs[0])
	}
	if scene.URL != nil {
		return strings.TrimSpace(*scene.URL)
	}
	return ""
}

func getSceneDurationSeconds(scene *models.ScrapedScene) (float64, bool) {
	if scene == nil {
		return 0, false
	}
	if scene.Duration != nil && *scene.Duration > 0 {
		return float64(*scene.Duration), true
	}
	if scene.File != nil && scene.File.Duration != nil && *scene.File.Duration > 0 {
		return *scene.File.Duration, true
	}
	return 0, false
}

func passesMinDuration(scene *models.ScrapedScene, minSeconds int) bool {
	if minSeconds <= 0 {
		return true
	}
	dur, ok := getSceneDurationSeconds(scene)
	if !ok {
		return false
	}
	return dur >= float64(minSeconds)
}

func normalizeTagName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func buildTagNameSet(tagRefs []configuredTagRef) map[string]struct{} {
	names := make(map[string]struct{})
	for _, tag := range tagRefs {
		if tag.name != "" {
			names[normalizeTagName(tag.name)] = struct{}{}
		}
		for _, alias := range tag.aliases {
			if alias != "" {
				names[normalizeTagName(alias)] = struct{}{}
			}
		}
	}
	return names
}

func sceneMatchesConfiguredTags(scene *models.ScrapedScene, tagRefs []configuredTagRef) bool {
	if scene == nil || len(tagRefs) == 0 {
		return false
	}
	idSet := make(map[string]struct{}, len(tagRefs))
	for _, tag := range tagRefs {
		idSet[tag.id] = struct{}{}
	}
	nameSet := buildTagNameSet(tagRefs)

	for _, tag := range scene.Tags {
		if tag == nil {
			continue
		}
		if tag.StoredID != nil {
			if _, ok := idSet[*tag.StoredID]; ok {
				return true
			}
		}
		if tag.Name != "" {
			if _, ok := nameSet[normalizeTagName(tag.Name)]; ok {
				return true
			}
		}
	}

	// Fallback: some sites expose weak/empty tag lists; still treat an exact
	// tag/alias mention in the title as a priority/stop match.
	if scene.Title != nil {
		title := normalizeTagName(*scene.Title)
		for name := range nameSet {
			if name != "" && strings.Contains(title, name) {
				return true
			}
		}
	}

	return false
}

func sceneMatchesConfiguredPerformer(scene *models.ScrapedScene, performer configuredPerformerRef) bool {
	if scene == nil {
		return false
	}
	names := make(map[string]struct{})
	if strings.TrimSpace(performer.name) != "" {
		names[normalizeTagName(performer.name)] = struct{}{}
	}
	for _, alias := range performer.aliases {
		if strings.TrimSpace(alias) != "" {
			names[normalizeTagName(alias)] = struct{}{}
		}
	}
	for _, p := range scene.Performers {
		if p == nil {
			continue
		}
		if p.StoredID != nil && *p.StoredID == performer.id {
			return true
		}
		if p.Name != nil {
			if _, ok := names[normalizeTagName(*p.Name)]; ok {
				return true
			}
		}
	}
	// Search pages / weak scrapers often omit performers; title usually has the name.
	if scene.Title != nil {
		title := normalizeTagName(*scene.Title)
		for name := range names {
			if name != "" && strings.Contains(title, name) {
				return true
			}
		}
	}
	return false
}

func filterAndShuffleScenes(scenes []*models.ScrapedScene, passes func(*models.ScrapedScene) bool, limit int) []*models.ScrapedScene {
	filtered := make([]*models.ScrapedScene, 0, len(scenes))
	for _, scene := range scenes {
		if passes(scene) {
			filtered = append(filtered, scene)
		}
	}
	shuffleScenes(filtered)
	if limit < 0 {
		limit = 0
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
}

func shuffleScenes(scenes []*models.ScrapedScene) {
	rand.Shuffle(len(scenes), func(i, j int) {
		scenes[i], scenes[j] = scenes[j], scenes[i]
	})
}

func shuffleRemoteSuggestions(items []remoteSuggestionItem) {
	rand.Shuffle(len(items), func(i, j int) {
		items[i], items[j] = items[j], items[i]
	})
}

func shuffleStrings(items []string) {
	rand.Shuffle(len(items), func(i, j int) {
		items[i], items[j] = items[j], items[i]
	})
}
