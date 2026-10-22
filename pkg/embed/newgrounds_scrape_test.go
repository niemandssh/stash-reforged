package embed

import "testing"

func TestExtractNewgroundsTagsFromHTML(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []string
	}{
		{
			name: "legacy tags block",
			html: `<dd class="tags"><a href="/search/conduct/movies?terms=nsfw">nsfw</a><a href="/search/conduct/movies?terms=parody">parody</a></dd>`,
			want: []string{"nsfw", "parody"},
		},
		{
			name: "sidestats tags block",
			html: `<dl class="sidestats"><dt>Tags</dt><dd><a href="/search/conduct/movies?terms=kneesocks-daemon">kneesocks-daemon</a><a href="/search/conduct/movies?terms=nsfw">nsfw</a></dd></dl>`,
			want: []string{"kneesocks-daemon", "nsfw"},
		},
		{
			name: "filters javascript placeholders",
			html: `<dd class="tags"><a>(function($) { alert(1); });</a><a>valid-tag</a></dd>`,
			want: []string{"valid-tag"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractNewgroundsTagsFromHTML(tt.html)
			if len(got) != len(tt.want) {
				t.Fatalf("tags = %#v, want %#v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("tags[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestEnrichNewgroundsSceneFromHTML(t *testing.T) {
	meta := &SceneMetadata{}
	enrichNewgroundsSceneFromHTML(meta, `<html>
<meta itemprop="uploadDate" content="2023-02-01T16:30:52-05:00">
<meta property="og:image" content="https://example.com/image.png">
<meta property="og:description" content="Example details">
<dl class="sidestats"><dt>Tags</dt><dd><a>parody</a><a>nsfw</a></dd></dl>
</html>`)

	if meta.Date != "2023-02-01" {
		t.Fatalf("date = %q", meta.Date)
	}
	if meta.Image != "https://example.com/image.png" {
		t.Fatalf("image = %q", meta.Image)
	}
	if meta.Details != "Example details" {
		t.Fatalf("details = %q", meta.Details)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "parody" {
		t.Fatalf("tags = %#v", meta.Tags)
	}
}
