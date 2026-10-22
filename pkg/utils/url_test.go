package utils

import "testing"

func TestURLFromHandle(t *testing.T) {
	type args struct {
		input   string
		siteURL string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "input is already a URL https",
			args: args{
				input:   "https://foo.com",
				siteURL: "https://bar.com",
			},
			want: "https://foo.com",
		},
		{
			name: "input is already a URL http",
			args: args{
				input:   "http://foo.com",
				siteURL: "https://bar.com",
			},
			want: "http://foo.com",
		},
		{
			name: "input is not a URL",
			args: args{
				input:   "foo",
				siteURL: "https://foo.com",
			},
			want: "https://foo.com/foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := URLFromHandle(tt.args.input, tt.args.siteURL); got != tt.want {
				t.Errorf("URLFromHandle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeImageURL(t *testing.T) {
	const malformed = "https://thumb-cdn77.xvideos-cdn.com/1f2429bd-104d-4c34-a64d-2619d6d37813/4/xv_20_p.avif');\thtml5player.setRelated(video_related);\thtml5player.setThumbSlide('https://thumb-cdn77.xvideos-cdn.com/1f2429bd-104d-4c34-a64d-2619d6d37813/4/mozaique_listing.jpg"

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "malformed xvideos script fragment",
			input: malformed,
			want:  "https://thumb-cdn77.xvideos-cdn.com/1f2429bd-104d-4c34-a64d-2619d6d37813/4/xv_20_p.avif",
		},
		{
			name:  "clean url unchanged",
			input: "https://example.com/cover.jpg",
			want:  "https://example.com/cover.jpg",
		},
		{
			name:  "base64 data url unchanged",
			input: "data:image/jpeg;base64,abc123",
			want:  "data:image/jpeg;base64,abc123",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeImageURL(tt.input); got != tt.want {
				t.Errorf("SanitizeImageURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
