package cache_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/model"
)

func writeJSONL(t *testing.T, path string, entries []cache.Entry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
}

func TestManager_Load_FileNotExist(t *testing.T) {
	m := cache.New(filepath.Join(t.TempDir(), "nonexistent.jsonl"))
	entries, err := m.Load()
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Load: expected empty slice, got %d entries", len(entries))
	}
}

func TestManager_Load_ValidJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.jsonl")

	want := []cache.Entry{
		{
			ProgramID: "p1",
			Datetime:  "2026-05-01T07:00:00+09:00",
			Title:     "エピソード1",
			Summary:   "要約1",
			Corners: []cache.CornerEntry{
				{
					Title: "コーナーA",
					Articles: []cache.ArticleEntry{
						{Title: "記事1", URL: "https://example.com/1", Summary: "記事要約1", Points: []string{"p1"}},
					},
				},
			},
		},
		{
			ProgramID: "p1",
			Datetime:  "2026-05-02T07:00:00+09:00",
			Title:     "エピソード2",
			Summary:   "要約2",
			Corners:   []cache.CornerEntry{},
		},
	}
	writeJSONL(t, path, want)

	m := cache.New(path)
	got, err := m.Load()
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Load: got %d entries, want 2", len(got))
	}
	if got[0].ProgramID != "p1" {
		t.Errorf("Entry[0].ProgramID: got %q, want %q", got[0].ProgramID, "p1")
	}
	if got[0].Title != "エピソード1" {
		t.Errorf("Entry[0].Title: got %q, want %q", got[0].Title, "エピソード1")
	}
	if len(got[0].Corners) != 1 || len(got[0].Corners[0].Articles) != 1 {
		t.Errorf("Entry[0].Corners: unexpected structure %+v", got[0].Corners)
	}
}

func TestManager_Append_CreatesFileAndAppends(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "cache.jsonl")

	m := cache.New(path)
	entry := cache.Entry{
		ProgramID: "prog",
		Datetime:  time.Now().Format(time.RFC3339),
		Title:     "テストエピソード",
		Summary:   "要約",
		Corners:   []cache.CornerEntry{},
	}

	if err := m.Append(entry, 100, 90); err != nil {
		t.Fatalf("Append: unexpected error: %v", err)
	}

	got, err := m.Load()
	if err != nil {
		t.Fatalf("Load after Append: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1", len(got))
	}
	if got[0].Title != "テストエピソード" {
		t.Errorf("Entry.Title: got %q, want %q", got[0].Title, "テストエピソード")
	}
}

func TestManager_Append_PrunesWhenExceedsMaxEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.jsonl")

	existing := []cache.Entry{
		{ProgramID: "p", Datetime: "2026-01-01T00:00:00Z", Title: "古い1", Corners: []cache.CornerEntry{}},
		{ProgramID: "p", Datetime: "2026-01-02T00:00:00Z", Title: "古い2", Corners: []cache.CornerEntry{}},
	}
	writeJSONL(t, path, existing)

	m := cache.New(path)
	newEntry := cache.Entry{
		ProgramID: "p",
		Datetime:  "2026-01-03T00:00:00Z",
		Title:     "新しい",
		Corners:   []cache.CornerEntry{},
	}

	if err := m.Append(newEntry, 2, 9999); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := m.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries after pruning maxEntries=2, want 2", len(got))
	}
	// oldest entry should be pruned
	if got[0].Title != "古い2" {
		t.Errorf("Entry[0].Title: got %q, want %q", got[0].Title, "古い2")
	}
	if got[1].Title != "新しい" {
		t.Errorf("Entry[1].Title: got %q, want %q", got[1].Title, "新しい")
	}
}

func TestManager_Append_PrunesOldEntriesByRetentionDays(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.jsonl")

	oldDatetime := time.Now().AddDate(0, 0, -100).Format(time.RFC3339)
	recentDatetime := time.Now().AddDate(0, 0, -1).Format(time.RFC3339)

	existing := []cache.Entry{
		{ProgramID: "p", Datetime: oldDatetime, Title: "古すぎる", Corners: []cache.CornerEntry{}},
		{ProgramID: "p", Datetime: recentDatetime, Title: "最近", Corners: []cache.CornerEntry{}},
	}
	writeJSONL(t, path, existing)

	m := cache.New(path)
	newEntry := cache.Entry{
		ProgramID: "p",
		Datetime:  time.Now().Format(time.RFC3339),
		Title:     "今",
		Corners:   []cache.CornerEntry{},
	}

	if err := m.Append(newEntry, 100, 90); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := m.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries (retention_days=90 should drop entry 100 days old), want 2", len(got))
	}
	if got[0].Title != "最近" {
		t.Errorf("Entry[0].Title: got %q, want %q", got[0].Title, "最近")
	}
}

func TestRecent_FewEntries(t *testing.T) {
	entries := []cache.Entry{
		{Title: "e1"},
		{Title: "e2"},
	}
	got := cache.Recent(entries, 5)
	if len(got) != 2 {
		t.Errorf("Recent(2 entries, n=5): got %d, want 2", len(got))
	}
}

func TestRecent_ExactN(t *testing.T) {
	entries := []cache.Entry{
		{Title: "e1"},
		{Title: "e2"},
		{Title: "e3"},
	}
	got := cache.Recent(entries, 2)
	if len(got) != 2 {
		t.Fatalf("Recent: got %d, want 2", len(got))
	}
	if got[0].Title != "e2" {
		t.Errorf("Recent[0].Title: got %q, want %q", got[0].Title, "e2")
	}
	if got[1].Title != "e3" {
		t.Errorf("Recent[1].Title: got %q, want %q", got[1].Title, "e3")
	}
}

func TestRecent_ZeroN(t *testing.T) {
	entries := []cache.Entry{{Title: "e1"}}
	got := cache.Recent(entries, 0)
	if len(got) != 0 {
		t.Errorf("Recent(n=0): expected empty, got %d", len(got))
	}
}

func TestPastURLs_ExtractsAllURLs(t *testing.T) {
	entries := []cache.Entry{
		{
			Corners: []cache.CornerEntry{
				{Articles: []cache.ArticleEntry{
					{URL: "https://example.com/1"},
					{URL: "https://example.com/2"},
				}},
			},
		},
		{
			Corners: []cache.CornerEntry{
				{Articles: []cache.ArticleEntry{
					{URL: "https://example.com/3"},
				}},
			},
		},
	}
	got := cache.PastURLs(entries)
	if len(got) != 3 {
		t.Fatalf("PastURLs: got %d URLs, want 3", len(got))
	}
}

func TestPastURLs_DeduplicatesURLs(t *testing.T) {
	entries := []cache.Entry{
		{
			Corners: []cache.CornerEntry{
				{Articles: []cache.ArticleEntry{
					{URL: "https://example.com/1"},
				}},
			},
		},
		{
			Corners: []cache.CornerEntry{
				{Articles: []cache.ArticleEntry{
					{URL: "https://example.com/1"},
				}},
			},
		},
	}
	got := cache.PastURLs(entries)
	if len(got) != 1 {
		t.Fatalf("PastURLs: got %d URLs (expected dedup=1)", len(got))
	}
}

func TestPastURLs_EmptyEntries(t *testing.T) {
	got := cache.PastURLs([]cache.Entry{})
	if got == nil {
		t.Error("PastURLs: expected non-nil slice for empty entries")
	}
	if len(got) != 0 {
		t.Errorf("PastURLs: expected empty slice, got %d", len(got))
	}
}

func TestBuildEntryFromManifest_BasicMapping(t *testing.T) {
	m := model.Manifest{
		Title:    "テストエピソード",
		Summary:  "全体要約",
		Datetime: "2026-06-01T07:00:00+09:00",
		Corners: []model.ManifestCorner{
			{
				Title: "コーナーA",
				Articles: []model.ArticleRef{
					{Title: "記事1", URL: "https://example.com/1"},
					{Title: "記事2", URL: "https://example.com/2"},
				},
			},
		},
	}
	rd := model.Rundown{
		Corners: []model.RundownCorner{
			{
				Title: "コーナーA",
				Articles: []model.RundownArticle{
					{URL: "https://example.com/1", Title: "記事1", Summary: "記事1の要約", Points: []string{"ポイント1"}},
				},
			},
		},
	}

	got := cache.BuildEntryFromManifest("prog-id", m, rd)

	if got.ProgramID != "prog-id" {
		t.Errorf("ProgramID: got %q, want %q", got.ProgramID, "prog-id")
	}
	if got.Title != "テストエピソード" {
		t.Errorf("Title: got %q, want %q", got.Title, "テストエピソード")
	}
	if got.Summary != "全体要約" {
		t.Errorf("Summary: got %q, want %q", got.Summary, "全体要約")
	}
	if got.Datetime != "2026-06-01T07:00:00+09:00" {
		t.Errorf("Datetime: got %q, want %q", got.Datetime, "2026-06-01T07:00:00+09:00")
	}
	if len(got.Corners) != 1 {
		t.Fatalf("Corners: got %d, want 1", len(got.Corners))
	}
	if got.Corners[0].Title != "コーナーA" {
		t.Errorf("Corners[0].Title: got %q, want %q", got.Corners[0].Title, "コーナーA")
	}
	if len(got.Corners[0].Articles) != 2 {
		t.Fatalf("Corners[0].Articles: got %d, want 2", len(got.Corners[0].Articles))
	}

	// Article with rundown data should have summary and points merged
	a1 := got.Corners[0].Articles[0]
	if a1.URL != "https://example.com/1" {
		t.Errorf("Articles[0].URL: got %q, want %q", a1.URL, "https://example.com/1")
	}
	if a1.Summary != "記事1の要約" {
		t.Errorf("Articles[0].Summary: got %q, want %q", a1.Summary, "記事1の要約")
	}
	if len(a1.Points) != 1 || a1.Points[0] != "ポイント1" {
		t.Errorf("Articles[0].Points: got %v, want [ポイント1]", a1.Points)
	}

	// Article without rundown data should still be included, with empty summary/points
	a2 := got.Corners[0].Articles[1]
	if a2.URL != "https://example.com/2" {
		t.Errorf("Articles[1].URL: got %q, want %q", a2.URL, "https://example.com/2")
	}
	if a2.Summary != "" {
		t.Errorf("Articles[1].Summary: expected empty for unknown URL, got %q", a2.Summary)
	}
	if len(a2.Points) != 0 {
		t.Errorf("Articles[1].Points: expected empty, got %v", a2.Points)
	}
}

func TestBuildEntryFromManifest_EmptyCorners(t *testing.T) {
	m := model.Manifest{Title: "空", Datetime: "2026-06-01T00:00:00Z"}
	rd := model.Rundown{}

	got := cache.BuildEntryFromManifest("p", m, rd)
	if got.Corners == nil {
		t.Error("Corners should be non-nil for empty manifest")
	}
	if len(got.Corners) != 0 {
		t.Errorf("Corners: got %d, want 0", len(got.Corners))
	}
}
