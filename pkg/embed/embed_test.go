package embed

import (
	"strings"
	"testing"
)

func TestVKVideoViewPageURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "page url",
			url:  "https://vk.com/video-62622863_169801357",
			want: "https://vk.com/video-62622863_169801357",
		},
		{
			name: "embed url",
			url:  "https://vk.com/video_ext.php?oid=-62622863&id=169801357",
			want: "https://vk.com/video-62622863_169801357",
		},
		{
			name: "embed url with hash",
			url:  "https://vk.com/video_ext.php?oid=-62622863&id=169801357&hash=abc123",
			want: "https://vk.com/video-62622863_169801357",
		},
		{
			name: "positive oid",
			url:  "https://vk.com/video12345_67890",
			want: "https://vk.com/video12345_67890",
		},
		{
			name: "vkvideo.ru page",
			url:  "https://vkvideo.ru/video12345_67890",
			want: "https://vk.com/video12345_67890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VKVideoViewPageURL(tt.url)
			if got != tt.want {
				t.Errorf("VKVideoViewPageURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantSrc string
		want    Site
		wantErr error
	}{
		{
			name:    "pornhub embed",
			url:     "https://www.pornhub.com/embed/ph123456789",
			wantSrc: "https://www.pornhub.com/embed/ph123456789",
			want:    SitePornHub,
		},
		{
			name:    "xvideos embed",
			url:     "https://www.xvideos.com/embedframe/abc123",
			wantSrc: "https://www.xvideos.com/embedframe/abc123",
			want:    SiteXVideos,
		},
		{
			name:    "xhamster embed legacy",
			url:     "https://xhamster.com/xembed.php?video=123456",
			wantSrc: "https://xhamster.com/xembed.php?video=123456",
			want:    SiteXHamster,
		},
		{
			name:    "xhamster embed modern",
			url:     "https://xhamster.com/embed/xhCvkeh",
			wantSrc: "https://xhamster.com/embed/xhCvkeh",
			want:    SiteXHamster,
		},
		{
			name:    "xhamster page modern short id",
			url:     "https://xhamster.com/videos/19-years-old-trans-has-her-first-threesome-xhCvkeh",
			wantSrc: "https://xhamster.com/embed/xhCvkeh",
			want:    SiteXHamster,
		},
		{
			name:    "xhamster page legacy numeric id",
			url:     "https://xhamster.com/videos/some-old-title-1234567",
			wantSrc: "https://xhamster.com/xembed.php?video=1234567",
			want:    SiteXHamster,
		},
		{
			name:    "xnxx embed",
			url:     "https://www.xnxx.com/embedframe/abc123",
			wantSrc: "https://www.xnxx.com/embedframe/abc123",
			want:    SiteXnxx,
		},
		{
			name:    "xnxx page",
			url:     "https://www.xnxx.com/video-10dkxj42/some_slug",
			wantSrc: "https://www.xnxx.com/embedframe/10dkxj42",
			want:    SiteXnxx,
		},
		{
			name:    "xnxx numeric page",
			url:     "https://www.xnxx.com/video12345/some_slug",
			wantSrc: "https://www.xnxx.com/embedframe/12345",
			want:    SiteXnxx,
		},
		{
			name:    "vk video page",
			url:     "https://vk.com/video-62622863_169801357",
			wantSrc: "https://vk.com/video_ext.php?oid=-62622863&id=169801357&hash=",
			want:    SiteVKVideo,
		},
		{
			name:    "vk video page positive oid",
			url:     "https://vk.com/video12345_67890",
			wantSrc: "https://vk.com/video_ext.php?oid=12345&id=67890&hash=",
			want:    SiteVKVideo,
		},
		{
			name:    "vk video embed with hash",
			url:     "https://vk.com/video_ext.php?oid=-62622863&id=169801357&hash=abc123",
			wantSrc: "https://vk.com/video_ext.php?oid=-62622863&id=169801357&hash=abc123",
			want:    SiteVKVideo,
		},
		{
			name:    "vk video embed oid first",
			url:     "https://vk.com/video_ext.php?oid=-62622863&id=169801357",
			wantSrc: "https://vk.com/video_ext.php?oid=-62622863&id=169801357&hash=",
			want:    SiteVKVideo,
		},
		{
			name:    "vkvideo.ru page",
			url:     "https://vkvideo.ru/video-62622863_169801357",
			wantSrc: "https://vk.com/video_ext.php?oid=-62622863&id=169801357&hash=",
			want:    SiteVKVideo,
		},
		{
			name:    "noodlemagazine watch page",
			url:     "https://hot.noodlemagazine.com/watch/-136203049_456239707",
			wantSrc: "",
			want:    SiteJWPlayer,
		},
		{
			name:    "ukdevilz watch page",
			url:     "https://w0w.ukdevilz.com/watch/-204376816_456239053",
			wantSrc: "",
			want:    SiteJWPlayer,
		},
		{
			name:    "mat6tube watch page",
			url:     "https://wow.mat6tube.com/watch/-165330904_456245253",
			wantSrc: "",
			want:    SiteJWPlayer,
		},
		{
			name:    "newgrounds portal view",
			url:     "https://www.newgrounds.com/portal/view/1039229",
			wantSrc: "",
			want:    SiteNewgrounds,
		},
		{
			name:    "playerjs page fallback",
			url:     "https://www.youtube.com/watch?v=abc",
			wantSrc: "",
			want:    SitePlayerJS,
		},
		{
			name:    "empty",
			url:     "  ",
			wantErr: ErrInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSrc, gotSite, gotErr := Parse(tt.url)

			if tt.wantErr != nil {
				if gotErr != tt.wantErr {
					t.Errorf("Parse(%q) error = %v, wantErr %v", tt.url, gotErr, tt.wantErr)
				}
				return
			}

			if gotSrc != tt.wantSrc {
				t.Errorf("Parse(%q) src = %q, want %q", tt.url, gotSrc, tt.wantSrc)
			}
			if gotSite != tt.want {
				t.Errorf("Parse(%q) site = %q, want %q", tt.url, gotSite, tt.want)
			}
		})
	}
}

func TestApplyStartTime(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		site        Site
		startSec    float64
		wantPrefix  string
		wantSuffix  string
		wantContains string
	}{
		{
			name: "pornhub",
			src:  "https://www.pornhub.com/embed/ph123456789",
			site: SitePornHub,
			startSec: 90,
			wantContains: "t=90",
		},
		{
			name: "xvideos",
			src:  "https://www.xvideos.com/embedframe/abc123",
			site: SiteXVideos,
			startSec: 90,
			wantContains: "t=90",
		},
		{
			name: "vk video",
			src:  "https://vk.com/video_ext.php?oid=-62622863&id=169801357&hash=",
			site: SiteVKVideo,
			startSec: 90,
			wantContains: "t=90",
		},
		{
			name: "zero start",
			src:  "https://www.pornhub.com/embed/ph123456789",
			site: SitePornHub,
			startSec: 0,
			wantPrefix: "https://www.pornhub.com/embed/ph123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyStartTime(tt.src, tt.site, tt.startSec)

			if tt.wantPrefix != "" && got != tt.wantPrefix {
				t.Errorf("ApplyStartTime(%q, %v, %v) = %q, want %q", tt.src, tt.site, tt.startSec, got, tt.wantPrefix)
			}
			if tt.wantSuffix != "" && got != tt.wantSuffix {
				t.Errorf("ApplyStartTime(%q, %v, %v) = %q, want %q", tt.src, tt.site, tt.startSec, got, tt.wantSuffix)
			}
			if tt.wantContains != "" {
				if got != tt.wantPrefix+tt.wantContains && !strings.Contains(got, tt.wantContains) {
					t.Errorf("ApplyStartTime(%q, %v, %v) = %q, want containing %q", tt.src, tt.site, tt.startSec, got, tt.wantContains)
				}
			}
		})
	}
}
