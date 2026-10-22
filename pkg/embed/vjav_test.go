package embed

import "testing"

func TestParseVJavPage(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "vjav videos page", url: "https://vjav.com/videos/685149/mayu-komikawa-3072-horny-lady-in-the-office/"},
		{name: "vjav embed page", url: "https://vjav.com/embed/685149/"},
		{name: "videovjav embed page", url: "https://videovjav.com/embed/685149/"},
		{name: "invalid path", url: "https://vjav.com/channels/jav/", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, site, err := Parse(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (site=%q)", site)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if site != SiteVJav {
				t.Fatalf("site = %q, want %q", site, SiteVJav)
			}
		})
	}
}

func TestVJavBase164Decode(t *testing.T) {
	// "/get_file/1/test/" encoded with the site's custom alphabet.
	encoded := "L2dldF9maWxlLzEvdGVzdC8~"
	got, err := vjavBase164Decode(encoded)
	if err != nil {
		t.Fatalf("vjavBase164Decode() error = %v", err)
	}
	want := "/get_file/1/test/"
	if got != want {
		t.Fatalf("vjavBase164Decode() = %q, want %q", got, want)
	}
}

func TestSupportsNativeStreamingVJav(t *testing.T) {
	if !SupportsNativeStreaming(SiteVJav) {
		t.Fatal("expected SiteVJav to support native streaming")
	}
}
