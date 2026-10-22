package scraper

import (
	"strings"

	"github.com/stashapp/stash/pkg/logger"
)

const (
	XvideosSearchScraperID  = "builtin_xvideos_search"
	XvideosListingScraperID = "builtin_xvideos_listing"
)

// Community Xvideos scraper only supports sceneByURL; these builtins add search/listing.
// Search pages use <p class="title"><a title="...">; listing (/best) uses
// <div class="title" title="..."><a>... — selectors cover both layouts.
//
// Important:
// - Image must use @data-src only (@src is blank.gif).
// - Do not use contains(@class,"thumb"): it also matches thumb-inside and
//   returns the same img twice, mis-aligning Title/URL/Image rows.
// - Go's regexp (RE2) does not support (?!...) lookaheads in replace.
const xvideosSearchSceneSearch = `
    common:
      $item: //div[contains(@class,"thumb-block") and (.//div[@class="title" or starts-with(@class,"title ")] or .//p[@class="title" or starts-with(@class,"title ")])]
    scene:
      Title: $item//div[@class="title" or starts-with(@class,"title ")]/@title|$item//p[@class="title" or starts-with(@class,"title ")]/a/@title
      URL:
        selector: $item//div[@class="title" or starts-with(@class,"title ")]/a/@href|$item//p[@class="title" or starts-with(@class,"title ")]/a/@href
        postProcess:
          - replace:
              - regex: ^
                with: https://www.xvideos.com
      Image:
        selector: $item//div[@class="thumb"]/a/img/@data-src
        postProcess:
          - replace:
              - regex: THUMBNUM
                with: "20"
      Duration:
        selector: $item//p[contains(@class,"metadata")]//span[@class="duration"]|$item//div[contains(@class,"video-metadata")]//span[@class="duration"]
        postProcess:
          - javascript: |
              if (!value) return "";
              const h = value.match(/(\d+)\s*h/i);
              const m = value.match(/(\d+)\s*min/i);
              const s = value.match(/(\d+)\s*sec/i);
              let sec = 0;
              if (h) sec += parseInt(h[1], 10) * 3600;
              if (m) sec += parseInt(m[1], 10) * 60;
              if (s) sec += parseInt(s[1], 10);
              if (!h && !m && !s) {
                const n = parseInt(value, 10);
                if (!isNaN(n)) sec = n;
              }
              return sec > 0 ? String(sec) : "";
`

const xvideosSceneScraper = `
  sceneScraper:
    scene:
      Title: //h2[@class="page-title"]/text()[1]
      URL: //link[@rel="alternate" and @hreflang="x-default"]/@href
      Image:
        selector: //script[contains(text(), "setThumbUrl169")]/text()
        postProcess:
          - replace:
              - regex: ^.+(?:setThumbUrl169|url)\('(.+?\.jpg).+$
                with: $1
`

const xvideosSearchScraperConfig = `
name: Xvideos Search
sceneByName:
  action: scrapeXPath
  queryURL: https://www.xvideos.com/?k={}
  scraper: sceneSearch
sceneByQueryFragment:
  action: scrapeXPath
  queryURL: "{url}"
  scraper: sceneScraper

xPathScrapers:
  sceneSearch:` + xvideosSearchSceneSearch + xvideosSceneScraper

const xvideosListingScraperConfig = `
name: Xvideos Listing
sceneByName:
  action: scrapeXPath
  queryURL: https://www.xvideos.com/{}
  scraper: sceneSearch
sceneByQueryFragment:
  action: scrapeXPath
  queryURL: "{url}"
  scraper: sceneScraper

xPathScrapers:
  sceneSearch:` + xvideosSearchSceneSearch + xvideosSceneScraper

func getXvideosSearchScraper(globalConfig GlobalConfig) scraper {
	return loadBuiltinScraper(globalConfig, XvideosSearchScraperID, xvideosSearchScraperConfig)
}

func getXvideosListingScraper(globalConfig GlobalConfig) scraper {
	return loadBuiltinScraper(globalConfig, XvideosListingScraperID, xvideosListingScraperConfig)
}

func loadBuiltinScraper(globalConfig GlobalConfig, id string, yml string) scraper {
	c, err := loadConfigFromYAML(id, strings.NewReader(yml))
	if err != nil {
		logger.Fatalf("Error loading builtin scraper %s: %s", id, err.Error())
	}

	return newGroupScraper(*c, globalConfig)
}
