package embed

import "testing"

func TestApplyStartTime(t *testing.T) {
	src := "https://www.pornhub.com/embed/ph5abc123"
	got := ApplyStartTime(src, SitePornHub, 125)
	want := "https://www.pornhub.com/embed/ph5abc123?t=125"
	if got != want {
		t.Fatalf("ApplyStartTime() = %q, want %q", got, want)
	}

	got = ApplyStartTime(src, SitePornHub, 0.553269)
	want = "https://www.pornhub.com/embed/ph5abc123?t=0.553269"
	if got != want {
		t.Fatalf("ApplyStartTime() fractional = %q, want %q", got, want)
	}

	xvideosSrc := "https://www.xvideos.com/embedframe/abc123"
	got = ApplyStartTime(xvideosSrc, SiteXVideos, 90)
	want = "https://www.xvideos.com/embedframe/abc123?t=90"
	if got != want {
		t.Fatalf("ApplyStartTime() xvideos = %q, want %q", got, want)
	}

	if ApplyStartTime(src, SitePornHub, 0) != src {
		t.Fatal("expected zero start to leave URL unchanged")
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantSrc string
		wantSite Site
		wantErr error
	}{
		{
			name:    "pornhub page",
			url:     "https://www.pornhub.com/view_video.php?viewkey=ph5abc123",
			wantSrc: "https://www.pornhub.com/embed/ph5abc123",
			wantSite: SitePornHub,
		},
		{
			name:    "pornhub org page",
			url:     "https://www.pornhub.org/view_video.php?viewkey=6a325faa543f2",
			wantSrc: "https://www.pornhub.com/embed/6a325faa543f2",
			wantSite: SitePornHub,
		},
		{
			name:    "pornhub embed",
			url:     "https://www.pornhub.com/embed/ph5abc123",
			wantSrc: "https://www.pornhub.com/embed/ph5abc123",
			wantSite: SitePornHub,
		},
		{
			name:    "xvideos dot id",
			url:     "https://www.xvideos.com/video.abc123def/title_here",
			wantSrc: "https://www.xvideos.com/embedframe/abc123def",
			wantSite: SiteXVideos,
		},
		{
			name:    "xvideos numeric id",
			url:     "https://www.xvideos.com/video12345678/title_here",
			wantSrc: "https://www.xvideos.com/embedframe/12345678",
			wantSite: SiteXVideos,
		},
		{
			name:    "xvideos embed",
			url:     "https://www.xvideos.com/embedframe/abc123def",
			wantSrc: "https://www.xvideos.com/embedframe/abc123def",
			wantSite: SiteXVideos,
		},
		{
			name:    "xhamster page",
			url:     "https://xhamster.com/videos/some-title-12345678",
			wantSrc: "https://xhamster.com/xembed.php?video=12345678",
			wantSite: SiteXHamster,
		},
		{
			name:    "xhamster embed",
			url:     "https://xhamster.com/xembed.php?video=12345678",
			wantSrc: "https://xhamster.com/xembed.php?video=12345678",
			wantSite: SiteXHamster,
		},
		{
			name:    "xnxx page",
			url:     "https://www.xnxx.com/video-abc123/title",
			wantSrc: "https://www.xnxx.com/embedframe/abc123",
			wantSite: SiteXnxx,
		},
		{
			name:    "xnxx embed",
			url:     "https://www.xnxx.com/embedframe/abc123",
			wantSrc: "https://www.xnxx.com/embedframe/abc123",
			wantSite: SiteXnxx,
		},
		{
			name:    "unsupported",
			url:     "https://www.youtube.com/watch?v=abc",
			wantErr: ErrUnsupportedSite,
		},
		{
			name:    "empty",
			url:     "  ",
			wantErr: ErrInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSrc, gotSite, err := Parse(tt.url)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("Parse() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() unexpected error = %v", err)
			}
			if gotSrc != tt.wantSrc {
				t.Errorf("Parse() src = %q, want %q", gotSrc, tt.wantSrc)
			}
			if gotSite != tt.wantSite {
				t.Errorf("Parse() site = %q, want %q", gotSite, tt.wantSite)
			}
		})
	}
}

func TestPornHubViewPageURL(t *testing.T) {
	got := PornHubViewPageURL("https://www.pornhub.com/embed/phabc123")
	want := "https://www.pornhub.com/view_video.php?viewkey=phabc123"
	if got != want {
		t.Fatalf("PornHubViewPageURL(embed) = %q, want %q", got, want)
	}

	got = PornHubViewPageURL("https://www.pornhub.com/view_video.php?viewkey=6a2a9f6cda3d7")
	want = "https://www.pornhub.com/view_video.php?viewkey=6a2a9f6cda3d7"
	if got != want {
		t.Fatalf("PornHubViewPageURL(view) = %q, want %q", got, want)
	}
}
