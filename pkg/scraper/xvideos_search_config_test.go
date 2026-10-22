package scraper

import (
	"strings"
	"testing"
)

func TestXvideosBuiltinConfigsLoad(t *testing.T) {
	for _, tc := range []struct {
		id  string
		yml string
	}{
		{XvideosSearchScraperID, xvideosSearchScraperConfig},
		{XvideosListingScraperID, xvideosListingScraperConfig},
	} {
		c, err := loadConfigFromYAML(tc.id, strings.NewReader(tc.yml))
		if err != nil {
			t.Fatalf("%s: %v\nYAML:\n%s", tc.id, err, tc.yml)
		}
		if c.Name == "" {
			t.Fatalf("%s: empty name", tc.id)
		}
	}
}
