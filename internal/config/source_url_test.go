package config

import (
	"testing"
)

func TestResolveFileURL(t *testing.T) {
	tests := []struct {
		name    string
		specDir string
		raw     string
		want    string
		wantErr bool
	}{
		{
			name:    "https URL passthrough",
			specDir: "/cfg",
			raw:     "https://example.com/feed.xml",
			want:    "https://example.com/feed.xml",
		},
		{
			name:    "http URL passthrough",
			specDir: "/cfg",
			raw:     "http://example.com/feed.xml",
			want:    "http://example.com/feed.xml",
		},
		{
			name:    "absolute file URL stays as-is",
			specDir: "/cfg",
			raw:     "file:///abs/x.xml",
			want:    "file:///abs/x.xml",
		},
		{
			name:    "relative file URL resolved against specDir",
			specDir: "/cfg",
			raw:     "file://feeds/x.xml",
			want:    "file:///cfg/feeds/x.xml",
		},
		{
			name:    "relative file URL with deeper path",
			specDir: "/home/user/spec",
			raw:     "file://subdir/feed.xml",
			want:    "file:///home/user/spec/subdir/feed.xml",
		},
		{
			name:    "empty string passthrough",
			specDir: "/cfg",
			raw:     "",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveFileURL(tt.specDir, tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveFileURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resolveFileURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
