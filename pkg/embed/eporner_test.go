package embed

import "testing"

func TestParseEporner(t *testing.T) {
	for _, u := range []string{
		"https://eporner.com/video-gtiMvcOoxLP/beach-ball-babe-alex-tanner/",
		"https://www.eporner.com/video-gtiMvcOoxLP/beach-ball-babe-alex-tanner/",
	} {
		if _, site, err := Parse(u); err != nil || site != SiteEporner {
			t.Fatalf("%s: site=%s err=%v", u, site, err)
		}
	}
}

func TestEpornerHashParam(t *testing.T) {
	if epornerHashParam("00000000000000010000000000000024") != "01010" {
		t.Fatal("unexpected hash param")
	}
}
