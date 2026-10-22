package embed

import "testing"

func TestParseTNAFlix(t *testing.T) {
	for _, u := range []string{
		"https://www.tnaflix.com/es/amateur-porn/Unique-Dual-POV-Fuck%21/video25073598",
		"tnaflix.com/amateur-porn/some-title/video25073598",
	} {
		if _, site, err := Parse(u); err != nil || site != SiteTNAFlix {
			t.Fatalf("%s: site=%s err=%v", u, site, err)
		}
	}
}
