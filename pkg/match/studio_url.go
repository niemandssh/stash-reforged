package match

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/studio"
)

var excludedStudioHostSuffixes = []string{
	"twitter.com",
	"x.com",
	"instagram.com",
	"onlyfans.com",
	"linktr.ee",
	"fansly.com",
	"patreon.com",
	"reddit.com",
	"facebook.com",
	"tiktok.com",
	"youtube.com",
	"pornolab.net",
}

var excludedStudioNames = []string{
	"pornolab",
}

var nonDomainTLDs = map[string]struct{}{
	"php": {}, "html": {}, "htm": {}, "asp": {}, "aspx": {}, "jsp": {},
}

var studioReferenceSeparators = []string{" / ", " | ", ", ", " /"}

// WebsiteHostCandidates returns normalized host strings used to match studios.
func WebsiteHostCandidates(rawURL string) []string {
	normalized := normalizeStudioReference(rawURL)
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Host == "" {
		return nil
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return nil
	}

	stripped := strings.TrimPrefix(host, "www.")
	candidates := []string{host, stripped}

	parts := strings.Split(stripped, ".")
	if len(parts) >= 2 {
		candidates = append(candidates, strings.Join(parts[len(parts)-2:], "."))
		candidates = append(candidates, parts[len(parts)-2])
	}

	seen := make(map[string]struct{}, len(candidates))
	unique := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		unique = append(unique, candidate)
	}

	return unique
}

// SecondLevelDomainName returns the second-level domain label for studio naming.
func SecondLevelDomainName(rawURL string) string {
	normalized := normalizeStudioReference(rawURL)
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Host == "" {
		if looksLikeDomain(rawURL) {
			parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(rawURL), "www."), ".")
			if len(parts) >= 2 {
				return parts[0]
			}
		}
		return ""
	}

	stripped := strings.TrimPrefix(parsed.Hostname(), "www.")
	parts := strings.Split(stripped, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}

	return ""
}

func isExcludedPerformerHost(host string) bool {
	host = strings.ToLower(strings.TrimPrefix(host, "www."))
	for _, suffix := range excludedStudioHostSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

func isExcludedStudioName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, excluded := range excludedStudioNames {
		if name == excluded {
			return true
		}
	}
	return false
}

func isExcludedStudioReference(reference string) bool {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return false
	}

	if isExcludedStudioName(reference) {
		return true
	}

	if looksLikeDomain(reference) || strings.Contains(reference, "://") {
		normalized := normalizeStudioReference(reference)
		parsed, err := url.Parse(normalized)
		if err == nil && parsed.Host != "" && isExcludedPerformerHost(parsed.Hostname()) {
			return true
		}

		if name := SecondLevelDomainName(normalized); isExcludedStudioName(name) {
			return true
		}
	}

	return false
}

func looksLikeDomain(reference string) bool {
	reference = strings.TrimSpace(reference)
	if reference == "" || strings.Contains(reference, " ") {
		return false
	}

	if strings.Contains(reference, "://") {
		parsed, err := url.Parse(reference)
		return err == nil && parsed.Host != ""
	}

	parts := strings.Split(reference, ".")
	if len(parts) >= 2 {
		tld := strings.ToLower(parts[len(parts)-1])
		if strings.ContainsAny(tld, "?#") {
			return false
		}
		if _, ok := nonDomainTLDs[tld]; ok {
			return false
		}
		return tld != ""
	}

	return false
}

func normalizeStudioReference(reference string) string {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return ""
	}

	if strings.Contains(reference, "://") {
		return reference
	}

	if looksLikeDomain(reference) {
		return "https://" + reference
	}

	return reference
}

func splitStudioReferences(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	for _, separator := range studioReferenceSeparators {
		if strings.Contains(value, separator) {
			parts := strings.Split(value, separator)
			results := make([]string, 0, len(parts))
			for _, part := range parts {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					results = append(results, trimmed)
				}
			}
			if len(results) > 1 {
				return results
			}
		}
	}

	if parts := splitSlashSeparatedStudioReferences(value); len(parts) > 1 {
		return parts
	}

	return []string{value}
}

func splitMalformedCompoundStudioURL(value string) []string {
	if !strings.Contains(value, "://") {
		return nil
	}

	parsed, err := url.Parse(normalizeStudioReference(value))
	if err != nil || parsed.Host == "" {
		return nil
	}

	host := parsed.Hostname()
	if !strings.Contains(host, " ") {
		return nil
	}

	path := strings.Trim(parsed.Path, "/")
	if path == "" {
		return nil
	}

	return []string{strings.TrimSpace(host), strings.TrimSpace(path)}
}

func splitSlashSeparatedStudioReferences(value string) []string {
	if isExcludedStudioReference(value) {
		return nil
	}

	if parts := splitMalformedCompoundStudioURL(value); len(parts) > 1 {
		return parts
	}

	stripped := value
	switch {
	case strings.HasPrefix(strings.ToLower(stripped), "https://"):
		stripped = stripped[len("https://"):]
	case strings.HasPrefix(strings.ToLower(stripped), "http://"):
		stripped = stripped[len("http://"):]
	}

	if !strings.Contains(stripped, "/") {
		if looksLikeDomain(stripped) {
			return nil
		}

		parsed, err := url.Parse(normalizeStudioReference(value))
		if err == nil && parsed.Host != "" && !strings.Contains(parsed.Hostname(), " ") {
			return nil
		}

		return nil
	}

	parts := strings.Split(stripped, "/")
	results := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			results = append(results, trimmed)
		}
	}

	if len(results) <= 1 || !isCompoundSlashParts(results) {
		return nil
	}

	if isExcludedStudioReference(results[0]) {
		return nil
	}

	return results
}

func isCompoundSlashParts(parts []string) bool {
	if len(parts) < 2 {
		return false
	}

	hasSpaceName := false
	domainCount := 0
	nonDomainCount := 0
	firstIsDomain := looksLikeDomain(parts[0])

	for _, part := range parts {
		if looksLikeDomain(part) {
			domainCount++
			continue
		}

		nonDomainCount++
		if strings.Contains(part, " ") {
			hasSpaceName = true
		}
	}

	if hasSpaceName {
		return true
	}

	if domainCount > 1 {
		return true
	}

	if firstIsDomain && nonDomainCount > 0 {
		return false
	}

	if domainCount == 1 && nonDomainCount == 1 {
		lastPart := parts[len(parts)-1]
		return strings.Contains(lastPart, ".") || strings.Contains(lastPart, " ")
	}

	return nonDomainCount > 1 && domainCount == 0
}

func isCompoundStudioReference(value string) bool {
	return len(splitStudioReferences(value)) > 1
}

// IsStudioCandidateURL reports whether a URL should be treated as a studio website
// rather than a performer profile link.
func IsStudioCandidateURL(rawURL string) bool {
	normalized := normalizeStudioReference(rawURL)
	parsed, err := url.Parse(normalized)
	if err == nil && parsed.Host != "" && isExcludedPerformerHost(parsed.Hostname()) {
		return false
	}

	if isCompoundStudioReference(rawURL) {
		return true
	}

	if err != nil || parsed.Host == "" {
		return looksLikeDomain(rawURL)
	}

	path := strings.Trim(parsed.Path, "/")
	if path == "" {
		return true
	}

	segments := strings.Split(path, "/")
	if len(segments) == 1 {
		switch strings.ToLower(strings.TrimSpace(segments[0])) {
		case "en", "ru", "de", "fr", "es", "it", "pt", "ja", "zh":
			return true
		}
	}

	return false
}

func findStudioByWebsite(ctx context.Context, qb StudioFinder, rawURL string) (*models.Studio, error) {
	if isExcludedStudioReference(rawURL) {
		return nil, nil
	}

	candidates := WebsiteHostCandidates(rawURL)
	if len(candidates) == 0 {
		return nil, nil
	}

	for _, candidate := range candidates {
		if isExcludedStudioName(candidate) {
			continue
		}

		st, err := studio.ByAlias(ctx, qb, candidate)
		if err != nil {
			return nil, err
		}
		if st != nil {
			return st, nil
		}
	}

	for _, candidate := range candidates {
		if isExcludedStudioName(candidate) {
			continue
		}

		pp := 1
		ret, count, err := qb.Query(ctx, &models.StudioFilterType{
			URL: &models.StringCriterionInput{
				Value:    candidate,
				Modifier: models.CriterionModifierIncludes,
			},
		}, &models.FindFilterType{
			PerPage: &pp,
		})
		if err != nil {
			return nil, err
		}
		if count > 0 && len(ret) > 0 {
			return ret[0], nil
		}
	}

	for _, candidate := range candidates {
		if isExcludedStudioName(candidate) {
			continue
		}

		st, err := findStudioByName(ctx, qb, candidate)
		if err != nil {
			return nil, err
		}
		if st != nil {
			return st, nil
		}
	}

	return nil, nil
}

func findStudioByName(ctx context.Context, qb StudioFinder, name string) (*models.Studio, error) {
	if isExcludedStudioName(name) {
		return nil, nil
	}

	st, err := studio.ByName(ctx, qb, name)
	if err != nil {
		return nil, err
	}
	if st != nil {
		return st, nil
	}

	return studio.ByAlias(ctx, qb, name)
}

func scrapedStudioFromMatch(st *models.Studio) *models.ScrapedStudio {
	if st == nil {
		return nil
	}

	id := strconv.Itoa(st.ID)
	url := st.URL
	return &models.ScrapedStudio{
		StoredID: &id,
		Name:     st.Name,
		URL:      stringPtrIfNotEmpty(url),
	}
}

func scrapedStudioProposalFromReference(reference string) *models.ScrapedStudio {
	reference = strings.TrimSpace(reference)
	if reference == "" || isExcludedStudioReference(reference) {
		return nil
	}

	if looksLikeDomain(reference) || strings.Contains(reference, "://") {
		normalized := normalizeStudioReference(reference)
		name := SecondLevelDomainName(normalized)
		if name == "" {
			return nil
		}

		return &models.ScrapedStudio{
			Name: name,
			URL:  stringPtrIfNotEmpty(normalized),
		}
	}

	return &models.ScrapedStudio{
		Name: reference,
	}
}

func studioProposalFromReference(reference string) (*models.ScrapedStudio, bool) {
	proposal := scrapedStudioProposalFromReference(reference)
	if proposal == nil {
		return nil, false
	}
	return proposal, true
}

func stringPtrIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// StudioFromReference matches an existing studio or proposes a new one.
func StudioFromReference(ctx context.Context, qb StudioFinder, reference string) (*models.ScrapedStudio, bool) {
	reference = strings.TrimSpace(reference)
	if reference == "" || isExcludedStudioReference(reference) {
		return nil, false
	}

	if looksLikeDomain(reference) || strings.Contains(reference, "://") {
		normalized := normalizeStudioReference(reference)
		st, err := findStudioByWebsite(ctx, qb, normalized)
		if err != nil {
			return nil, false
		}
		if st != nil {
			return scrapedStudioFromMatch(st), true
		}

		if IsStudioCandidateURL(reference) {
			return studioProposalFromReference(reference)
		}

		return nil, false
	}

	st, err := findStudioByName(ctx, qb, reference)
	if err != nil {
		return nil, false
	}
	if st != nil {
		return scrapedStudioFromMatch(st), true
	}

	if isCompoundStudioReference(reference) {
		return studioProposalFromReference(reference)
	}

	return nil, false
}

// StudioFromWebsiteURL matches an existing studio or proposes a new one from a website URL.
func StudioFromWebsiteURL(ctx context.Context, qb StudioFinder, rawURL string) (*models.ScrapedStudio, bool) {
	return StudioFromReference(ctx, qb, rawURL)
}

func combineCompoundStudioResults(matched, proposals []*models.ScrapedStudio) *models.ScrapedStudio {
	var primary *models.ScrapedStudio

	if len(proposals) > 0 {
		primary = proposals[0]
	} else {
		primary = matched[0]
		matched = matched[1:]
	}

	if len(matched) > 0 {
		if primary.Parent == nil {
			primary.Parent = matched[0]
		}
		current := primary.Parent
		for i := 1; i < len(matched); i++ {
			if current.Parent == nil {
				current.Parent = matched[i]
			}
			current = current.Parent
		}
	}

	if len(proposals) > 1 && primary.StoredID == nil {
		current := primary
		for i := 1; i < len(proposals); i++ {
			if current.Parent == nil {
				current.Parent = proposals[i]
			}
			current = current.Parent
		}
	}

	return primary
}

func studioFromValue(ctx context.Context, qb StudioFinder, value string) (*models.ScrapedStudio, bool) {
	if isExcludedStudioReference(value) {
		return nil, false
	}

	references := splitStudioReferences(value)
	if len(references) <= 1 {
		return StudioFromReference(ctx, qb, value)
	}

	var matched []*models.ScrapedStudio
	var proposals []*models.ScrapedStudio

	for _, reference := range references {
		if studioResult, ok := StudioFromReference(ctx, qb, reference); ok && studioResult != nil {
			if studioResult.StoredID != nil {
				matched = append(matched, studioResult)
			} else {
				proposals = append(proposals, studioResult)
			}
			continue
		}

		if proposal := scrapedStudioProposalFromReference(reference); proposal != nil {
			proposals = append(proposals, proposal)
		}
	}

	if len(matched) == 0 && len(proposals) == 0 {
		return nil, false
	}

	return combineCompoundStudioResults(matched, proposals), true
}

func studioReferenceFromTitle(title string) (string, bool) {
	title = strings.TrimSpace(title)
	if !strings.HasPrefix(title, "[") {
		return "", false
	}

	end := strings.Index(title, "]")
	if end <= 1 {
		return "", false
	}

	return strings.TrimSpace(title[1:end]), true
}

// StudioFromTitle extracts a studio from a leading [studio] prefix in the title.
func StudioFromTitle(ctx context.Context, qb StudioFinder, title string) (*models.ScrapedStudio, bool) {
	reference, ok := studioReferenceFromTitle(title)
	if !ok {
		return nil, false
	}

	return studioFromValue(ctx, qb, reference)
}

// ExtractStudioURLs splits performer URLs into remaining performer URLs and an optional studio.
func ExtractStudioURLs(ctx context.Context, qb StudioFinder, urls []string) ([]string, *models.ScrapedStudio) {
	remaining := make([]string, 0, len(urls))
	var foundStudio *models.ScrapedStudio

	for _, rawURL := range urls {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			continue
		}

		studioResult, extracted := studioFromValue(ctx, qb, rawURL)
		if extracted {
			if foundStudio == nil && studioResult != nil {
				foundStudio = studioResult
			}
			continue
		}

		remaining = append(remaining, rawURL)
	}

	return remaining, foundStudio
}
