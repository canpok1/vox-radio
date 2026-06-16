package feed

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/cache"
)

func TestExtractChannelMeta(t *testing.T) {
	tests := []struct {
		name       string
		entries    []cache.Entry
		wantTitle  string
		wantDesc   string
		wantAuthor string
	}{
		{
			name:       "empty entries returns zero value",
			entries:    []cache.Entry{},
			wantTitle:  "",
			wantDesc:   "",
			wantAuthor: "",
		},
		{
			name: "single entry returns its fields",
			entries: []cache.Entry{
				{Title: "タイトル", Description: "説明", Author: "著者"},
			},
			wantTitle:  "タイトル",
			wantDesc:   "説明",
			wantAuthor: "著者",
		},
		{
			name: "multiple entries returns latest (last) entry fields",
			entries: []cache.Entry{
				{Title: "古いタイトル", Description: "古い説明", Author: "古い著者"},
				{Title: "最新タイトル", Description: "最新説明", Author: "最新著者"},
			},
			wantTitle:  "最新タイトル",
			wantDesc:   "最新説明",
			wantAuthor: "最新著者",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractChannelMeta(tt.entries)
			if got.title != tt.wantTitle {
				t.Errorf("title = %q, want %q", got.title, tt.wantTitle)
			}
			if got.description != tt.wantDesc {
				t.Errorf("description = %q, want %q", got.description, tt.wantDesc)
			}
			if got.author != tt.wantAuthor {
				t.Errorf("author = %q, want %q", got.author, tt.wantAuthor)
			}
		})
	}
}
