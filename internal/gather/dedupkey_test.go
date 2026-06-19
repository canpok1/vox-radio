package gather

import (
	"strings"
	"testing"
)

func TestDedupKey_Format(t *testing.T) {
	k := dedupKey("https://feed.example.com/rss", "guid-123")
	if !strings.HasPrefix(k, "sha256:") {
		t.Errorf("dedupKey must start with sha256:, got %q", k)
	}
	if len(k) != len("sha256:")+64 {
		t.Errorf("dedupKey length: got %d, want %d", len(k), len("sha256:")+64)
	}
}

func TestDedupKey_Deterministic(t *testing.T) {
	k1 := dedupKey("https://feed.example.com/rss", "guid-123")
	k2 := dedupKey("https://feed.example.com/rss", "guid-123")
	if k1 != k2 {
		t.Errorf("same inputs must produce same key: %q != %q", k1, k2)
	}
}

func TestDedupKey_DifferentMaterial(t *testing.T) {
	k1 := dedupKey("https://feed.example.com/rss", "guid-123")
	k2 := dedupKey("https://feed.example.com/rss", "guid-456")
	if k1 == k2 {
		t.Errorf("different materials must produce different keys")
	}
}

func TestDedupKey_NamespaceIsolation(t *testing.T) {
	// 同じ material でも namespace が異なれば別のキーになる（フィード間のguid衝突を回避）
	k1 := dedupKey("https://feed1.example.com/rss", "item-1")
	k2 := dedupKey("https://feed2.example.com/rss", "item-1")
	if k1 == k2 {
		t.Errorf("same material with different namespace must produce different keys")
	}
}

func TestNormalizeContent(t *testing.T) {
	tests := []struct {
		name  string
		title string
		body  string
		want  string
	}{
		{
			name:  "basic",
			title: "タイトル",
			body:  "本文",
			want:  "タイトル\n本文",
		},
		{
			name:  "trim whitespace",
			title: "  タイトル  ",
			body:  "  本文  ",
			want:  "タイトル\n本文",
		},
		{
			name:  "collapse whitespace",
			title: "タイトル  スペース",
			body:  "本文\t\tタブ",
			want:  "タイトル スペース\n本文 タブ",
		},
		{
			name:  "empty body",
			title: "タイトル",
			body:  "",
			want:  "タイトル\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeContent(tt.title, tt.body)
			if got != tt.want {
				t.Errorf("normalizeContent(%q, %q) = %q, want %q", tt.title, tt.body, got, tt.want)
			}
		})
	}
}
