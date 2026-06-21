package gather

import (
	"strings"
	"testing"
)

func TestFeedDedupKey_WithGUID(t *testing.T) {
	// GUID が非空なら link・content より優先
	k := FeedDedupKey("https://feed.example.com/rss", "guid-abc", "https://link.example.com", "タイトル", "本文")
	kNoLinkNoContent := FeedDedupKey("https://feed.example.com/rss", "guid-abc", "", "別タイトル", "別本文")
	if k != kNoLinkNoContent {
		t.Errorf("GUID が同じなら link・content が異なってもキーが同一のはず: %q vs %q", k, kNoLinkNoContent)
	}
}

func TestFeedDedupKey_WithLinkOnly(t *testing.T) {
	// GUID が空 かつ link が非空なら link を material に使う
	k := FeedDedupKey("https://feed.example.com/rss", "", "https://link.example.com/1", "タイトル", "本文")
	kSameURL := FeedDedupKey("https://feed.example.com/rss", "", "https://link.example.com/1", "別タイトル", "別本文")
	if k != kSameURL {
		t.Errorf("GUID なし・同一 link なら content が違ってもキー同一のはず: %q vs %q", k, kSameURL)
	}
	kDiffURL := FeedDedupKey("https://feed.example.com/rss", "", "https://link.example.com/2", "タイトル", "本文")
	if k == kDiffURL {
		t.Errorf("link が異なれば別キーになるはず: %q == %q", k, kDiffURL)
	}
}

func TestFeedDedupKey_FallsBackToContent(t *testing.T) {
	// GUID・link ともに空 → normalizeContent(title, body) を material に使う
	k := FeedDedupKey("https://feed.example.com/rss", "", "", "タイトル", "本文")
	kSame := FeedDedupKey("https://feed.example.com/rss", "", "", "タイトル", "本文")
	if k != kSame {
		t.Errorf("同一入力で異なるキーが出るはず: %q vs %q", k, kSame)
	}
	kDiff := FeedDedupKey("https://feed.example.com/rss", "", "", "別タイトル", "別本文")
	if k == kDiff {
		t.Errorf("content が異なれば別キーになるはず: %q == %q", k, kDiff)
	}
}

func TestFeedDedupKey_GUIDTakesPrecedenceOverLink(t *testing.T) {
	kGUID := FeedDedupKey("https://feed.example.com/rss", "guid-abc", "https://link.example.com", "", "")
	kLink := FeedDedupKey("https://feed.example.com/rss", "", "https://link.example.com", "", "")
	if kGUID == kLink {
		t.Errorf("GUID と link で material が異なるのでキーは別になるはず: %q == %q", kGUID, kLink)
	}
}

func TestLinksDedupKey_StableByFilePathAndURL(t *testing.T) {
	k1 := LinksDedupKey("/data/links.txt", "https://example.com/article")
	k2 := LinksDedupKey("/data/links.txt", "https://example.com/article")
	if k1 != k2 {
		t.Errorf("同一 filePath・lineURL で同一キーになるはず: %q vs %q", k1, k2)
	}
	kDiffURL := LinksDedupKey("/data/links.txt", "https://example.com/other")
	if k1 == kDiffURL {
		t.Errorf("lineURL が異なれば別キーになるはず: %q == %q", k1, kDiffURL)
	}
	kDiffFile := LinksDedupKey("/data/other.txt", "https://example.com/article")
	if k1 == kDiffFile {
		t.Errorf("filePath が異なれば別キーになるはず: %q == %q", k1, kDiffFile)
	}
}

func TestLinksDedupKey_ContentIndependent(t *testing.T) {
	// ページ内容が変わっても同一ファイルパス＋同一 URL なら同一キー
	k := LinksDedupKey("/data/links.txt", "https://example.com/page")
	// content 変化を模擬するが kLinks に引数はないため、キー自体は不変
	if k == "" {
		t.Error("LinksDedupKey must not be empty")
	}
	if !strings.HasPrefix(k, "sha256:") {
		t.Errorf("LinksDedupKey must start with sha256:, got %q", k)
	}
}

func TestTextDedupKey_StableByContent(t *testing.T) {
	k1 := TextDedupKey("/data/ref.txt", "参考情報", "本文テキスト")
	k2 := TextDedupKey("/data/ref.txt", "参考情報", "本文テキスト")
	if k1 != k2 {
		t.Errorf("同一 filePath・title・body で同一キーになるはず: %q vs %q", k1, k2)
	}
	kDiffBody := TextDedupKey("/data/ref.txt", "参考情報", "更新された本文")
	if k1 == kDiffBody {
		t.Errorf("本文が変われば別キーになるはず（再採用可）: %q == %q", k1, kDiffBody)
	}
	kDiffFile := TextDedupKey("/data/other.txt", "参考情報", "本文テキスト")
	if k1 == kDiffFile {
		t.Errorf("filePath が異なれば別キーになるはず: %q == %q", k1, kDiffFile)
	}
}

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
