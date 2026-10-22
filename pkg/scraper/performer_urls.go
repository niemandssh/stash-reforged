package scraper

import "github.com/stashapp/stash/pkg/models"

func performerURLList(p *models.ScrapedPerformer) []string {
	if p == nil {
		return nil
	}

	if len(p.URLs) > 0 {
		return append([]string(nil), p.URLs...)
	}

	if p.URL != nil && *p.URL != "" {
		return []string{*p.URL}
	}

	return nil
}

func setPerformerURLList(p *models.ScrapedPerformer, urls []string) {
	if p == nil {
		return
	}

	p.URLs = urls
	if len(urls) > 0 {
		p.URL = &urls[0]
	} else {
		p.URL = nil
	}
}
