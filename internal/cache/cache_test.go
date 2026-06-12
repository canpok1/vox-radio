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

func TestManager_Append_CompactsWhenExceedsMaxEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.jsonl")

	corner := cache.CornerEntry{Title: "コーナー", Summary: "要約", Articles: []cache.ArticleEntry{{URL: "https://example.com/1"}}}
	existing := []cache.Entry{
		{ProgramID: "p", Datetime: "2026-01-01T00:00:00Z", Title: "古い1", Corners: []cache.CornerEntry{corner}},
		{ProgramID: "p", Datetime: "2026-01-02T00:00:00Z", Title: "古い2", Corners: []cache.CornerEntry{corner}},
	}
	writeJSONL(t, path, existing)

	m := cache.New(path)
	newEntry := cache.Entry{
		ProgramID: "p",
		Datetime:  "2026-01-03T00:00:00Z",
		Title:     "新しい",
		Corners:   []cache.CornerEntry{corner},
	}

	if err := m.Append(newEntry, 2, 9999); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := m.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// compact keeps all 3 entries (no deletion)
	if len(got) != 3 {
		t.Fatalf("got %d entries after compacting maxEntries=2, want 3 (all kept)", len(got))
	}
	// oldest entry should have heavy corner fields stripped but identity (Title) preserved
	if got[0].Title != "古い1" {
		t.Errorf("Entry[0].Title: got %q, want %q", got[0].Title, "古い1")
	}
	if len(got[0].Corners) != 1 {
		t.Fatalf("Entry[0].Corners: got %d corners, want 1 (identity kept for appearance counting)", len(got[0].Corners))
	}
	if got[0].Corners[0].Title != "コーナー" {
		t.Errorf("Entry[0].Corners[0].Title: got %q, want コーナー (kept)", got[0].Corners[0].Title)
	}
	if got[0].Corners[0].Summary != "" || len(got[0].Corners[0].Articles) != 0 {
		t.Errorf("Entry[0].Corners[0] heavy fields should be stripped, got Summary=%q Articles=%d", got[0].Corners[0].Summary, len(got[0].Corners[0].Articles))
	}
	// newer entries should keep full data
	if got[1].Title != "古い2" {
		t.Errorf("Entry[1].Title: got %q, want %q", got[1].Title, "古い2")
	}
	if len(got[1].Corners) != 1 {
		t.Errorf("Entry[1].Corners: got %d corners, want 1 (kept)", len(got[1].Corners))
	}
	if got[2].Title != "新しい" {
		t.Errorf("Entry[2].Title: got %q, want %q", got[2].Title, "新しい")
	}
}

func TestManager_Append_CompactsOldEntriesByRetentionDays(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.jsonl")

	oldDatetime := time.Now().AddDate(0, 0, -100).Format(time.RFC3339)
	recentDatetime := time.Now().AddDate(0, 0, -1).Format(time.RFC3339)

	corner := cache.CornerEntry{Title: "コーナー", Summary: "要約", Articles: []cache.ArticleEntry{{URL: "https://example.com/1"}}}
	existing := []cache.Entry{
		{ProgramID: "p", Datetime: oldDatetime, Title: "古すぎる", Corners: []cache.CornerEntry{corner}},
		{ProgramID: "p", Datetime: recentDatetime, Title: "最近", Corners: []cache.CornerEntry{corner}},
	}
	writeJSONL(t, path, existing)

	m := cache.New(path)
	newEntry := cache.Entry{
		ProgramID: "p",
		Datetime:  time.Now().Format(time.RFC3339),
		Title:     "今",
		Corners:   []cache.CornerEntry{corner},
	}

	if err := m.Append(newEntry, 100, 90); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := m.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// compact keeps all 3 entries (no deletion)
	if len(got) != 3 {
		t.Fatalf("got %d entries (retention_days=90 should compact but not delete entry 100 days old), want 3", len(got))
	}
	// oldest entry should have heavy corner fields stripped but identity (Title) preserved
	if got[0].Title != "古すぎる" {
		t.Errorf("Entry[0].Title: got %q, want %q", got[0].Title, "古すぎる")
	}
	if len(got[0].Corners) != 1 {
		t.Fatalf("Entry[0].Corners: got %d corners, want 1 (identity kept for appearance counting)", len(got[0].Corners))
	}
	if got[0].Corners[0].Summary != "" || len(got[0].Corners[0].Articles) != 0 {
		t.Errorf("Entry[0].Corners[0] heavy fields should be stripped, got Summary=%q Articles=%d", got[0].Corners[0].Summary, len(got[0].Corners[0].Articles))
	}
	// recent entries should keep full data
	if got[1].Title != "最近" {
		t.Errorf("Entry[1].Title: got %q, want %q", got[1].Title, "最近")
	}
	if len(got[1].Corners) != 1 {
		t.Errorf("Entry[1].Corners: got %d corners, want 1 (kept)", len(got[1].Corners))
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
	const key1 = "sha256:aaa1111111111111111111111111111111111111111111111111111111111111"
	const key2 = "sha256:bbb2222222222222222222222222222222222222222222222222222222222222"
	m := model.Manifest{
		Title:    "テストエピソード",
		Summary:  "全体要約",
		Datetime: "2026-06-01T07:00:00+09:00",
		Corners: []model.ManifestCorner{
			{
				Title: "コーナーA",
				Articles: []model.ArticleRef{
					{DedupKey: key1, Title: "記事1", URL: "https://example.com/1"},
					{DedupKey: key2, Title: "記事2", URL: "https://example.com/2"},
				},
			},
		},
	}
	rd := model.Rundown{
		Corners: []model.RundownCorner{
			{
				Title: "コーナーA",
				Articles: []model.RundownArticle{
					{DedupKey: key1, URL: "https://example.com/1", Title: "記事1", Summary: "記事1の要約", Points: []string{"ポイント1"}},
				},
			},
		},
	}

	got := cache.BuildEntryFromManifest("prog-id", m, rd, 0, 0)

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
	if a1.DedupKey != key1 {
		t.Errorf("Articles[0].DedupKey: got %q, want %q", a1.DedupKey, key1)
	}
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
	if a2.DedupKey != key2 {
		t.Errorf("Articles[1].DedupKey: got %q, want %q", a2.DedupKey, key2)
	}
	if a2.URL != "https://example.com/2" {
		t.Errorf("Articles[1].URL: got %q, want %q", a2.URL, "https://example.com/2")
	}
	if a2.Summary != "" {
		t.Errorf("Articles[1].Summary: expected empty for unknown DedupKey, got %q", a2.Summary)
	}
	if len(a2.Points) != 0 {
		t.Errorf("Articles[1].Points: expected empty, got %v", a2.Points)
	}
}

func TestBuildEntryFromManifest_EmptyCorners(t *testing.T) {
	m := model.Manifest{Title: "空", Datetime: "2026-06-01T00:00:00Z"}
	rd := model.Rundown{}

	got := cache.BuildEntryFromManifest("p", m, rd, 0, 0)
	if got.Corners == nil {
		t.Error("Corners should be non-nil for empty manifest")
	}
	if len(got.Corners) != 0 {
		t.Errorf("Corners: got %d, want 0", len(got.Corners))
	}
}

func TestBuildEntryFromManifest_CornerSummaryAndPointsIncluded(t *testing.T) {
	m := model.Manifest{
		Title:    "エピソード",
		Datetime: "2026-06-01T00:00:00Z",
		Corners: []model.ManifestCorner{
			{
				Title:   "コーナーA",
				Summary: "コーナーAの会話要約",
				Points:  []string{"要点1", "要点2"},
			},
		},
	}
	rd := model.Rundown{}

	got := cache.BuildEntryFromManifest("p", m, rd, 0, 0)

	if len(got.Corners) != 1 {
		t.Fatalf("Corners: got %d, want 1", len(got.Corners))
	}
	c := got.Corners[0]
	if c.Summary != "コーナーAの会話要約" {
		t.Errorf("Corners[0].Summary: got %q, want %q", c.Summary, "コーナーAの会話要約")
	}
	if len(c.Points) != 2 {
		t.Fatalf("Corners[0].Points: got %d, want 2", len(c.Points))
	}
	if c.Points[0] != "要点1" {
		t.Errorf("Corners[0].Points[0]: got %q, want %q", c.Points[0], "要点1")
	}
}

func TestBuildEntryFromManifest_CornerPointsNeverNil(t *testing.T) {
	m := model.Manifest{
		Title:    "エピソード",
		Datetime: "2026-06-01T00:00:00Z",
		Corners: []model.ManifestCorner{
			{Title: "コーナーA"},
		},
	}
	rd := model.Rundown{}

	got := cache.BuildEntryFromManifest("p", m, rd, 0, 0)

	if len(got.Corners) != 1 {
		t.Fatalf("Corners: got %d, want 1", len(got.Corners))
	}
	if got.Corners[0].Points == nil {
		t.Error("Corners[0].Points must be [] not nil")
	}
}

func TestBuildEntryFromManifest_ConversationNotesCopied(t *testing.T) {
	m := model.Manifest{
		Title:    "エピソード",
		Datetime: "2026-06-01T00:00:00Z",
		Corners:  []model.ManifestCorner{},
		ConversationNotes: []model.ConversationNote{
			{Category: "近況", CharacterIDs: []string{"zundamon"}, Note: "カフェにハマっている"},
		},
	}
	rd := model.Rundown{}

	got := cache.BuildEntryFromManifest("p", m, rd, 0, 0)

	if len(got.ConversationNotes) != 1 {
		t.Fatalf("ConversationNotes: got %d, want 1", len(got.ConversationNotes))
	}
	n := got.ConversationNotes[0]
	if n.Category != "近況" {
		t.Errorf("ConversationNotes[0].Category: got %q, want %q", n.Category, "近況")
	}
	if len(n.CharacterIDs) != 1 || n.CharacterIDs[0] != "zundamon" {
		t.Errorf("ConversationNotes[0].CharacterIDs: got %v, want [zundamon]", n.CharacterIDs)
	}
	if n.Note != "カフェにハマっている" {
		t.Errorf("ConversationNotes[0].Note: got %q, want %q", n.Note, "カフェにハマっている")
	}
}

func TestBuildEntryFromManifest_EpisodeNumberAndTitleCopied(t *testing.T) {
	m := model.Manifest{
		Title:         "エピソード",
		Datetime:      "2026-06-01T00:00:00Z",
		EpisodeNumber: 7,
		EpisodeTitle:  "今週の技術ニュース",
		Corners:       []model.ManifestCorner{},
	}
	rd := model.Rundown{}

	got := cache.BuildEntryFromManifest("p", m, rd, 0, 0)

	if got.EpisodeNumber != 7 {
		t.Errorf("EpisodeNumber: got %d, want 7", got.EpisodeNumber)
	}
	if got.EpisodeTitle != "今週の技術ニュース" {
		t.Errorf("EpisodeTitle: got %q, want %q", got.EpisodeTitle, "今週の技術ニュース")
	}
}

func TestNextEpisodeNumber_NoEntries_ReturnsOne(t *testing.T) {
	got := cache.NextEpisodeNumber([]cache.Entry{})
	if got != 1 {
		t.Errorf("NextEpisodeNumber(empty): got %d, want 1", got)
	}
}

func TestNextEpisodeNumber_LatestHasEpisodeNumber_ReturnsNextNumber(t *testing.T) {
	entries := []cache.Entry{
		{EpisodeNumber: 3},
		{EpisodeNumber: 5},
	}
	got := cache.NextEpisodeNumber(entries)
	if got != 6 {
		t.Errorf("NextEpisodeNumber(latest=5): got %d, want 6", got)
	}
}

func TestNextEpisodeNumber_LegacyEntries_ReturnsLenPlusOne(t *testing.T) {
	entries := []cache.Entry{
		{EpisodeNumber: 0},
		{EpisodeNumber: 0},
		{EpisodeNumber: 0},
	}
	got := cache.NextEpisodeNumber(entries)
	if got != 4 {
		t.Errorf("NextEpisodeNumber(3 legacy entries): got %d, want 4", got)
	}
}

func TestBuildEntryFromManifest_NewFields_Populated(t *testing.T) {
	m := model.Manifest{
		Title:       "エピソード",
		Datetime:    "2026-06-01T00:00:00Z",
		Description: "番組説明テキスト",
		AudioFile:   "episode.mp3",
		Corners:     []model.ManifestCorner{},
	}
	rd := model.Rundown{}

	got := cache.BuildEntryFromManifest("p", m, rd, 12345678, 1800)

	if got.Description != "番組説明テキスト" {
		t.Errorf("Description: got %q, want %q", got.Description, "番組説明テキスト")
	}
	if got.AudioFile != "episode.mp3" {
		t.Errorf("AudioFile: got %q, want %q", got.AudioFile, "episode.mp3")
	}
	if got.Bytes != 12345678 {
		t.Errorf("Bytes: got %d, want 12345678", got.Bytes)
	}
	if got.DurationSec != 1800 {
		t.Errorf("DurationSec: got %d, want 1800", got.DurationSec)
	}
}

func TestCompact_KeepsAllEntries(t *testing.T) {
	corner := cache.CornerEntry{Title: "コーナー", Summary: "要約"}
	entries := []cache.Entry{
		{Datetime: "2026-01-01T00:00:00Z", Title: "e1", Corners: []cache.CornerEntry{corner}},
		{Datetime: "2026-01-02T00:00:00Z", Title: "e2", Corners: []cache.CornerEntry{corner}},
		{Datetime: "2026-01-03T00:00:00Z", Title: "e3", Corners: []cache.CornerEntry{corner}},
	}

	got := cache.Compact(entries, 2, 9999)

	if len(got) != 3 {
		t.Fatalf("Compact: got %d entries, want 3 (all kept)", len(got))
	}
}

func TestCompact_StripsCornerHeavyFieldsAndEmptiesNotes_ForEntriesOutsideMaxEntries(t *testing.T) {
	corner := cache.CornerEntry{ID: "c1", Title: "コーナー", Summary: "要約", Points: []string{"p"}, Articles: []cache.ArticleEntry{{URL: "u"}}}
	note := model.ConversationNote{Category: "近況", Note: "メモ"}
	entries := []cache.Entry{
		{Datetime: "2026-01-01T00:00:00Z", Title: "e1", Corners: []cache.CornerEntry{corner}, ConversationNotes: []model.ConversationNote{note}},
		{Datetime: "2026-01-02T00:00:00Z", Title: "e2", Corners: []cache.CornerEntry{corner}, ConversationNotes: []model.ConversationNote{note}},
		{Datetime: "2026-01-03T00:00:00Z", Title: "e3", Corners: []cache.CornerEntry{corner}, ConversationNotes: []model.ConversationNote{note}},
	}

	got := cache.Compact(entries, 2, 9999)

	// entry[0] is outside maxEntries=2 window: identity kept, heavy fields stripped
	if len(got[0].Corners) != 1 {
		t.Fatalf("Compact: entry[0].Corners should keep identity (1), got %d", len(got[0].Corners))
	}
	if got[0].Corners[0].ID != "c1" || got[0].Corners[0].Title != "コーナー" {
		t.Errorf("Compact: entry[0].Corners[0] identity should be kept, got %+v", got[0].Corners[0])
	}
	if got[0].Corners[0].Summary != "" || len(got[0].Corners[0].Points) != 0 || len(got[0].Corners[0].Articles) != 0 {
		t.Errorf("Compact: entry[0].Corners[0] heavy fields should be stripped, got %+v", got[0].Corners[0])
	}
	if len(got[0].ConversationNotes) != 0 {
		t.Errorf("Compact: entry[0].ConversationNotes should be empty (compacted), got %d", len(got[0].ConversationNotes))
	}
	// entries[1,2] are within window, should keep full data
	if len(got[1].Corners) != 1 || got[1].Corners[0].Summary != "要約" {
		t.Errorf("Compact: entry[1].Corners should keep full data, got %+v", got[1].Corners)
	}
	if len(got[2].Corners) != 1 || got[2].Corners[0].Summary != "要約" {
		t.Errorf("Compact: entry[2].Corners should keep full data, got %+v", got[2].Corners)
	}
}

func TestCompact_StripsCornerHeavyFields_ForOldEntries(t *testing.T) {
	corner := cache.CornerEntry{ID: "c1", Title: "コーナー", Summary: "要約"}
	oldDatetime := time.Now().AddDate(0, 0, -100).Format(time.RFC3339)
	recentDatetime := time.Now().AddDate(0, 0, -1).Format(time.RFC3339)

	entries := []cache.Entry{
		{Datetime: oldDatetime, Title: "old", Corners: []cache.CornerEntry{corner}},
		{Datetime: recentDatetime, Title: "recent", Corners: []cache.CornerEntry{corner}},
	}

	got := cache.Compact(entries, 100, 90)

	// old entry (outside retention_days=90): identity kept, heavy fields stripped
	if len(got[0].Corners) != 1 || got[0].Corners[0].ID != "c1" {
		t.Fatalf("Compact: old entry.Corners should keep identity, got %+v", got[0].Corners)
	}
	if got[0].Corners[0].Summary != "" {
		t.Errorf("Compact: old entry.Corners[0].Summary should be stripped, got %q", got[0].Corners[0].Summary)
	}
	// recent entry should keep full data
	if len(got[1].Corners) != 1 || got[1].Corners[0].Summary != "要約" {
		t.Errorf("Compact: recent entry.Corners should keep full data, got %+v", got[1].Corners)
	}
}

func TestCompact_KeepsLightweightFields(t *testing.T) {
	corner := cache.CornerEntry{Title: "コーナー", Summary: "要約"}
	entries := []cache.Entry{
		{
			Datetime:    "2026-01-01T00:00:00Z",
			Title:       "e1",
			Summary:     "要約",
			Description: "番組説明",
			AudioFile:   "episode.mp3",
			Bytes:       12345,
			DurationSec: 600,
			Corners:     []cache.CornerEntry{corner},
		},
		{Datetime: "2026-01-02T00:00:00Z", Title: "e2", Corners: []cache.CornerEntry{corner}},
		{Datetime: "2026-01-03T00:00:00Z", Title: "e3", Corners: []cache.CornerEntry{corner}},
	}

	got := cache.Compact(entries, 2, 9999)

	// entry[0] is compacted, but lightweight fields must be preserved
	e0 := got[0]
	if e0.Title != "e1" {
		t.Errorf("Compact: entry[0].Title: got %q, want e1", e0.Title)
	}
	if e0.Summary != "要約" {
		t.Errorf("Compact: entry[0].Summary: got %q, want 要約", e0.Summary)
	}
	if e0.Description != "番組説明" {
		t.Errorf("Compact: entry[0].Description: got %q, want 番組説明", e0.Description)
	}
	if e0.AudioFile != "episode.mp3" {
		t.Errorf("Compact: entry[0].AudioFile: got %q, want episode.mp3", e0.AudioFile)
	}
	if e0.Bytes != 12345 {
		t.Errorf("Compact: entry[0].Bytes: got %d, want 12345", e0.Bytes)
	}
	if e0.DurationSec != 600 {
		t.Errorf("Compact: entry[0].DurationSec: got %d, want 600", e0.DurationSec)
	}
	// corner identity is kept (for appearance counting) but heavy fields are stripped
	if len(e0.Corners) != 1 {
		t.Fatalf("Compact: entry[0].Corners should keep identity (1), got %d", len(e0.Corners))
	}
	if e0.Corners[0].Summary != "" {
		t.Errorf("Compact: entry[0].Corners[0].Summary should be stripped, got %q", e0.Corners[0].Summary)
	}
}

func TestCompact_PreservesCasts(t *testing.T) {
	casts := []cache.CastEntry{
		{CharacterID: "zundamon", Type: "regular"},
		{CharacterID: "guest1", Type: "guest"},
	}
	corner := cache.CornerEntry{Title: "コーナー", Summary: "要約"}
	entries := []cache.Entry{
		{Datetime: "2026-01-01T00:00:00Z", Title: "e1", Casts: casts, Corners: []cache.CornerEntry{corner}},
		{Datetime: "2026-01-02T00:00:00Z", Title: "e2", Casts: casts, Corners: []cache.CornerEntry{corner}},
		{Datetime: "2026-01-03T00:00:00Z", Title: "e3", Casts: casts, Corners: []cache.CornerEntry{corner}},
	}

	got := cache.Compact(entries, 2, 9999)

	// entry[0] is compacted (Corners emptied) but Casts must be preserved
	if len(got[0].Casts) != 2 {
		t.Errorf("Compact: entry[0].Casts: got %d, want 2 (preserved even when compacted)", len(got[0].Casts))
	}
	if got[0].Casts[0].CharacterID != "zundamon" {
		t.Errorf("Compact: entry[0].Casts[0].CharacterID: got %q, want zundamon", got[0].Casts[0].CharacterID)
	}
}

func TestCastAppearances_EmptyEntries(t *testing.T) {
	got := cache.CastAppearances([]cache.Entry{})
	if len(got) != 0 {
		t.Errorf("CastAppearances(empty): got %d entries, want 0", len(got))
	}
}

func TestCastAppearances_LegacyEntriesWithNoCasts(t *testing.T) {
	entries := []cache.Entry{
		{Title: "e1"}, // legacy: no Casts field
		{Title: "e2"},
	}
	got := cache.CastAppearances(entries)
	// legacy entries contribute nothing, map should be empty
	if len(got) != 0 {
		t.Errorf("CastAppearances(legacy): got %d entries, want 0", len(got))
	}
}

func TestCastAppearances_CountsAndLastEpisodeNumber(t *testing.T) {
	entries := []cache.Entry{
		{EpisodeNumber: 1, Casts: []cache.CastEntry{
			{CharacterID: "zundamon", Type: "regular"},
			{CharacterID: "guest1", Type: "guest"},
		}},
		{EpisodeNumber: 2, Casts: []cache.CastEntry{
			{CharacterID: "zundamon", Type: "regular"},
		}},
		{EpisodeNumber: 5, Casts: []cache.CastEntry{
			{CharacterID: "zundamon", Type: "regular"},
			{CharacterID: "guest1", Type: "guest"},
		}},
	}
	got := cache.CastAppearances(entries)

	if got["zundamon"].Count != 3 {
		t.Errorf("zundamon.Count: got %d, want 3", got["zundamon"].Count)
	}
	if got["zundamon"].LastEpisodeNumber != 5 {
		t.Errorf("zundamon.LastEpisodeNumber: got %d, want 5", got["zundamon"].LastEpisodeNumber)
	}
	if got["guest1"].Count != 2 {
		t.Errorf("guest1.Count: got %d, want 2", got["guest1"].Count)
	}
	if got["guest1"].LastEpisodeNumber != 5 {
		t.Errorf("guest1.LastEpisodeNumber: got %d, want 5", got["guest1"].LastEpisodeNumber)
	}
}

func TestCornerAppearances_EmptyEntries(t *testing.T) {
	got := cache.CornerAppearances([]cache.Entry{})
	if len(got) != 0 {
		t.Errorf("CornerAppearances(empty): got %d, want 0", len(got))
	}
}

func TestCornerAppearances_IgnoresCornersWithoutID(t *testing.T) {
	entries := []cache.Entry{
		{EpisodeNumber: 1, Corners: []cache.CornerEntry{{Title: "レガシー"}}}, // no ID
		{EpisodeNumber: 2, Corners: []cache.CornerEntry{{ID: "", Title: "空ID"}}},
	}
	got := cache.CornerAppearances(entries)
	if len(got) != 0 {
		t.Errorf("CornerAppearances(no-id): got %d, want 0 (ID-less corners ignored)", len(got))
	}
}

func TestCornerAppearances_CountsAndLastEpisodeNumber(t *testing.T) {
	entries := []cache.Entry{
		{EpisodeNumber: 1, Corners: []cache.CornerEntry{{ID: "opening"}, {ID: "weather"}}},
		{EpisodeNumber: 2, Corners: []cache.CornerEntry{{ID: "opening"}}},
		{EpisodeNumber: 5, Corners: []cache.CornerEntry{{ID: "opening"}, {ID: "weather"}}},
	}
	got := cache.CornerAppearances(entries)

	if got["opening"].Count != 3 {
		t.Errorf("opening.Count: got %d, want 3", got["opening"].Count)
	}
	if got["opening"].LastEpisodeNumber != 5 {
		t.Errorf("opening.LastEpisodeNumber: got %d, want 5", got["opening"].LastEpisodeNumber)
	}
	if got["weather"].Count != 2 {
		t.Errorf("weather.Count: got %d, want 2", got["weather"].Count)
	}
	if got["weather"].LastEpisodeNumber != 5 {
		t.Errorf("weather.LastEpisodeNumber: got %d, want 5", got["weather"].LastEpisodeNumber)
	}
}

func TestBuildEntryFromManifest_CornerIDCopied(t *testing.T) {
	m := model.Manifest{
		Title:    "エピソード",
		Datetime: "2026-06-01T00:00:00Z",
		Corners: []model.ManifestCorner{
			{ID: "opening", Title: "オープニング", Summary: "概要"},
		},
	}
	rd := model.Rundown{}

	got := cache.BuildEntryFromManifest("p", m, rd, 0, 0)

	if len(got.Corners) != 1 {
		t.Fatalf("Corners: got %d, want 1", len(got.Corners))
	}
	if got.Corners[0].ID != "opening" {
		t.Errorf("Corners[0].ID: got %q, want opening", got.Corners[0].ID)
	}
}

func TestBuildEntryFromManifest_CastsCopied(t *testing.T) {
	m := model.Manifest{
		Title:    "エピソード",
		Datetime: "2026-06-01T00:00:00Z",
		Corners:  []model.ManifestCorner{},
		Casts: []model.RundownCast{
			{CharacterID: "zundamon", Role: "MC", Type: "regular", AppearanceCount: 3},
			{CharacterID: "guest1", Role: "ゲスト", Type: "guest", AppearanceCount: 0},
		},
	}
	rd := model.Rundown{}

	got := cache.BuildEntryFromManifest("p", m, rd, 0, 0)

	if len(got.Casts) != 2 {
		t.Fatalf("BuildEntryFromManifest: Casts: got %d, want 2", len(got.Casts))
	}
	if got.Casts[0].CharacterID != "zundamon" {
		t.Errorf("Casts[0].CharacterID: got %q, want zundamon", got.Casts[0].CharacterID)
	}
	if got.Casts[0].Type != "regular" {
		t.Errorf("Casts[0].Type: got %q, want regular", got.Casts[0].Type)
	}
	if got.Casts[1].CharacterID != "guest1" {
		t.Errorf("Casts[1].CharacterID: got %q, want guest1", got.Casts[1].CharacterID)
	}
}

func TestBuildEntryFromManifest_CastsNeverNil(t *testing.T) {
	m := model.Manifest{
		Title:    "エピソード",
		Datetime: "2026-06-01T00:00:00Z",
		Corners:  []model.ManifestCorner{},
		// Casts is nil
	}
	rd := model.Rundown{}

	got := cache.BuildEntryFromManifest("p", m, rd, 0, 0)

	if got.Casts == nil {
		t.Error("BuildEntryFromManifest: Casts must be [] not nil")
	}
	if len(got.Casts) != 0 {
		t.Errorf("BuildEntryFromManifest: Casts: got %d, want 0", len(got.Casts))
	}
}

func TestPastDedupKeys_ExtractsAllKeys(t *testing.T) {
	entries := []cache.Entry{
		{Corners: []cache.CornerEntry{
			{Articles: []cache.ArticleEntry{
				{DedupKey: "sha256:aaa"},
				{DedupKey: "sha256:bbb"},
			}},
		}},
		{Corners: []cache.CornerEntry{
			{Articles: []cache.ArticleEntry{
				{DedupKey: "sha256:ccc"},
			}},
		}},
	}
	got := cache.PastDedupKeys(entries)
	if len(got) != 3 {
		t.Fatalf("PastDedupKeys: got %d keys, want 3", len(got))
	}
}

func TestPastDedupKeys_DeduplicatesKeys(t *testing.T) {
	entries := []cache.Entry{
		{Corners: []cache.CornerEntry{
			{Articles: []cache.ArticleEntry{{DedupKey: "sha256:aaa"}}},
		}},
		{Corners: []cache.CornerEntry{
			{Articles: []cache.ArticleEntry{{DedupKey: "sha256:aaa"}}},
		}},
	}
	got := cache.PastDedupKeys(entries)
	if len(got) != 1 {
		t.Fatalf("PastDedupKeys: got %d keys (expected dedup=1)", len(got))
	}
}

func TestPastDedupKeys_SkipsEmptyKeys(t *testing.T) {
	// 旧エントリで DedupKey が空の場合はスキップ
	entries := []cache.Entry{
		{Corners: []cache.CornerEntry{
			{Articles: []cache.ArticleEntry{
				{URL: "https://example.com/1"}, // DedupKey なし（旧エントリ）
				{DedupKey: "sha256:aaa"},
			}},
		}},
	}
	got := cache.PastDedupKeys(entries)
	if len(got) != 1 {
		t.Fatalf("PastDedupKeys: got %d keys, want 1 (empty DedupKey skipped)", len(got))
	}
	if got[0] != "sha256:aaa" {
		t.Errorf("PastDedupKeys: got %q, want sha256:aaa", got[0])
	}
}

func TestPastDedupKeys_EmptyEntries(t *testing.T) {
	got := cache.PastDedupKeys([]cache.Entry{})
	if got == nil {
		t.Error("PastDedupKeys: expected non-nil slice for empty entries")
	}
	if len(got) != 0 {
		t.Errorf("PastDedupKeys: expected empty slice, got %d", len(got))
	}
}
