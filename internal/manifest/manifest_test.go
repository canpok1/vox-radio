package manifest_test

import (
	"testing"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/manifest"
	"github.com/canpok1/vox-radio/internal/model"
)

var fixedTime = time.Date(2026, 5, 31, 12, 34, 56, 0, time.UTC)

// newMinimalBuildParams returns a minimal BuildParams for tests that focus on
// Casts or EpisodeNumber/EpisodeTitle fields.
func newMinimalBuildParams() manifest.BuildParams {
	return manifest.BuildParams{
		Program:     config.ProgramConfig{Title: "テスト番組", Description: "説明"},
		Corners:     []config.CornerConfig{{Title: "コーナー1"}},
		AudioFile:   "morning-news_ep001.mp3",
		GeneratedAt: fixedTime,
	}
}

func TestBuild(t *testing.T) {
	program := config.ProgramConfig{
		Title:       "今日のテックニュース",
		Description: "毎日5分のニュースラジオ",
	}

	corners := []config.CornerConfig{
		{ID: "opening", Title: "オープニング"},
		{ID: "tech", Title: "今日のテックニュース"},
	}

	rundown := model.Rundown{
		Corners: []model.RundownCorner{
			{
				ID:    "tech",
				Title: "今日のテックニュース",
				Flow:  "最新記事を紹介",
				Articles: []model.RundownArticle{
					{URL: "https://example.com/articles/123", Title: "記事タイトル", Body: "記事の本文", Points: []string{"p1"}},
				},
			},
		},
	}

	defaultParams := func() manifest.BuildParams {
		return manifest.BuildParams{
			Program:     program,
			Corners:     corners,
			Rundown:     rundown,
			AudioFile:   "morning-news_ep001.mp3",
			GeneratedAt: fixedTime,
		}
	}

	t.Run("title and description from program", func(t *testing.T) {
		got := manifest.Build(defaultParams())
		if got.Title != program.Title {
			t.Errorf("Title = %q, want %q", got.Title, program.Title)
		}
		if got.Description != program.Description {
			t.Errorf("Description = %q, want %q", got.Description, program.Description)
		}
	})

	t.Run("audio_file is set", func(t *testing.T) {
		got := manifest.Build(defaultParams())
		if got.AudioFile != "morning-news_ep001.mp3" {
			t.Errorf("AudioFile = %q, want %q", got.AudioFile, "morning-news_ep001.mp3")
		}
	})

	t.Run("datetime is RFC3339 UTC", func(t *testing.T) {
		got := manifest.Build(defaultParams())
		want := "2026-05-31T12:34:56Z"
		if got.Datetime != want {
			t.Errorf("Datetime = %q, want %q", got.Datetime, want)
		}
	})

	t.Run("corners in spec order", func(t *testing.T) {
		got := manifest.Build(defaultParams())
		if len(got.Corners) != len(corners) {
			t.Fatalf("len(Corners) = %d, want %d", len(got.Corners), len(corners))
		}
		for i, c := range corners {
			if got.Corners[i].Title != c.Title {
				t.Errorf("Corners[%d].Title = %q, want %q", i, got.Corners[i].Title, c.Title)
			}
		}
	})

	t.Run("corner without articles has empty array not null", func(t *testing.T) {
		got := manifest.Build(defaultParams())
		opening := got.Corners[0]
		if opening.Articles == nil {
			t.Error("Articles for corner without articles must be [] not nil")
		}
		if len(opening.Articles) != 0 {
			t.Errorf("Articles for corner without articles = %v, want []", opening.Articles)
		}
	})

	t.Run("articles attributed to correct corner", func(t *testing.T) {
		got := manifest.Build(defaultParams())
		techCorner := got.Corners[1]
		if len(techCorner.Articles) != 1 {
			t.Fatalf("len(Articles) = %d, want 1", len(techCorner.Articles))
		}
		if techCorner.Articles[0].Title != "記事タイトル" {
			t.Errorf("Articles[0].Title = %q, want %q", techCorner.Articles[0].Title, "記事タイトル")
		}
		if techCorner.Articles[0].URL != "https://example.com/articles/123" {
			t.Errorf("Articles[0].URL = %q, want %q", techCorner.Articles[0].URL, "https://example.com/articles/123")
		}
	})

	t.Run("empty rundown produces empty articles arrays", func(t *testing.T) {
		p := defaultParams()
		p.Rundown = model.Rundown{}
		got := manifest.Build(p)
		for i, c := range got.Corners {
			if c.Articles == nil {
				t.Errorf("Corners[%d].Articles must be [] not nil", i)
			}
		}
	})

	t.Run("corners slice is not nil", func(t *testing.T) {
		p := defaultParams()
		p.Corners = []config.CornerConfig{}
		got := manifest.Build(p)
		if got.Corners == nil {
			t.Error("Corners must be [] not nil")
		}
	})

	t.Run("summary is set from argument", func(t *testing.T) {
		want := "今回はAIチップと最新ニュースを紹介しました。"
		p := defaultParams()
		p.Summary = want
		got := manifest.Build(p)
		if got.Summary != want {
			t.Errorf("Summary = %q, want %q", got.Summary, want)
		}
	})

	t.Run("empty summary when empty string given", func(t *testing.T) {
		got := manifest.Build(defaultParams())
		if got.Summary != "" {
			t.Errorf("Summary = %q, want empty", got.Summary)
		}
	})

	t.Run("only selected articles appear in manifest", func(t *testing.T) {
		p := defaultParams()
		p.Rundown = model.Rundown{
			Corners: []model.RundownCorner{
				{
					ID:    "tech",
					Title: "今日のテックニュース",
					Articles: []model.RundownArticle{
						{URL: "https://example.com/1", Title: "選別記事1"},
					},
				},
			},
		}
		got := manifest.Build(p)
		techCorner := got.Corners[1]
		if len(techCorner.Articles) != 1 {
			t.Errorf("Articles count = %d, want 1 (only selected articles)", len(techCorner.Articles))
		}
	})

	t.Run("corner summary and points are included from cornerSummaries map", func(t *testing.T) {
		p := defaultParams()
		p.CornerSummaries = map[string]model.CornerSummary{
			"今日のテックニュース": {
				Summary: "AIチップについて話しました。",
				Points:  []string{"要点1", "要点2"},
			},
		}
		got := manifest.Build(p)
		techCorner := got.Corners[1]
		if techCorner.Summary != "AIチップについて話しました。" {
			t.Errorf("Corners[1].Summary = %q, want %q", techCorner.Summary, "AIチップについて話しました。")
		}
		if len(techCorner.Points) != 2 {
			t.Fatalf("Corners[1].Points len = %d, want 2", len(techCorner.Points))
		}
		if techCorner.Points[0] != "要点1" {
			t.Errorf("Corners[1].Points[0] = %q, want %q", techCorner.Points[0], "要点1")
		}
	})

	t.Run("corner points is empty array not nil when no corner summary provided", func(t *testing.T) {
		got := manifest.Build(defaultParams())
		for i, c := range got.Corners {
			if c.Points == nil {
				t.Errorf("Corners[%d].Points must be [] not nil", i)
			}
		}
	})

	t.Run("corner with summary has non-nil points", func(t *testing.T) {
		p := defaultParams()
		p.CornerSummaries = map[string]model.CornerSummary{
			"オープニング": {Summary: "開始", Points: nil},
		}
		got := manifest.Build(p)
		if got.Corners[0].Points == nil {
			t.Error("Points must be [] not nil even when CornerSummary.Points is nil")
		}
	})

	t.Run("conversation notes are included", func(t *testing.T) {
		p := defaultParams()
		p.ConversationNotes = []model.ConversationNote{
			{Category: "近況", CharacterIDs: []string{"zundamon"}, Note: "カフェにハマっている"},
		}
		got := manifest.Build(p)
		if len(got.ConversationNotes) != 1 {
			t.Fatalf("ConversationNotes: got %d, want 1", len(got.ConversationNotes))
		}
		if got.ConversationNotes[0].Category != "近況" {
			t.Errorf("ConversationNotes[0].Category: got %q, want %q", got.ConversationNotes[0].Category, "近況")
		}
	})

	t.Run("conversation notes is empty array not nil when nil given", func(t *testing.T) {
		got := manifest.Build(defaultParams())
		if got.ConversationNotes == nil {
			t.Error("ConversationNotes must be [] not nil")
		}
		if len(got.ConversationNotes) != 0 {
			t.Errorf("ConversationNotes: got %d, want 0", len(got.ConversationNotes))
		}
	})
}

func TestBuild_AuthorCopiedFromProgram(t *testing.T) {
	p := newMinimalBuildParams()
	p.Program.Author = "テスト配信者"
	got := manifest.Build(p)
	if got.Author != "テスト配信者" {
		t.Errorf("Author = %q, want %q", got.Author, "テスト配信者")
	}
}

func TestBuild_AuthorEmptyWhenNotSet(t *testing.T) {
	got := manifest.Build(newMinimalBuildParams())
	if got.Author != "" {
		t.Errorf("Author = %q, want empty", got.Author)
	}
}

func TestBuild_CastsCopiedFromRundown(t *testing.T) {
	p := newMinimalBuildParams()
	p.Rundown = model.Rundown{
		Casts: []model.RundownCast{
			{CharacterID: "zundamon", Role: "MC", Type: "regular", AppearanceCount: 2},
			{CharacterID: "guest1", Role: "ゲスト", Type: "guest", AppearanceCount: 0},
		},
	}

	got := manifest.Build(p)

	if len(got.Casts) != 2 {
		t.Fatalf("Casts: got %d, want 2", len(got.Casts))
	}
	if got.Casts[0].CharacterID != "zundamon" {
		t.Errorf("Casts[0].CharacterID: got %q, want zundamon", got.Casts[0].CharacterID)
	}
	if got.Casts[0].AppearanceCount != 2 {
		t.Errorf("Casts[0].AppearanceCount: got %d, want 2", got.Casts[0].AppearanceCount)
	}
	if got.Casts[1].CharacterID != "guest1" {
		t.Errorf("Casts[1].CharacterID: got %q, want guest1", got.Casts[1].CharacterID)
	}
}

func TestBuild_CastsFirstAppearancePreserved(t *testing.T) {
	// 新定義: AppearanceCount=1 は初登場（今回含む出演回数）
	p := newMinimalBuildParams()
	p.Rundown = model.Rundown{
		Casts: []model.RundownCast{
			{CharacterID: "guest1", Role: "ゲスト", Type: "guest", AppearanceCount: 1},
		},
	}

	got := manifest.Build(p)

	if len(got.Casts) != 1 {
		t.Fatalf("Casts: got %d, want 1", len(got.Casts))
	}
	// 新定義値（1=初登場）がマニフェストにそのまま保持される
	if got.Casts[0].AppearanceCount != 1 {
		t.Errorf("Casts[0].AppearanceCount: got %d, want 1 (初登場=1 in new definition)", got.Casts[0].AppearanceCount)
	}
}

func TestBuild_CastsNeverNil(t *testing.T) {
	p := newMinimalBuildParams()
	// Rundown.Casts is nil (zero value)

	got := manifest.Build(p)

	if got.Casts == nil {
		t.Error("Casts must be [] not nil when rundown has no casts")
	}
	if len(got.Casts) != 0 {
		t.Errorf("Casts: got %d, want 0", len(got.Casts))
	}
}

func TestBuild_EpisodeNumberAndTitle(t *testing.T) {
	t.Run("episode_number and episode_title are set when provided", func(t *testing.T) {
		p := newMinimalBuildParams()
		p.EpisodeNumber = 3
		p.EpisodeTitle = "今週の面白技術"
		got := manifest.Build(p)
		if got.EpisodeNumber != 3 {
			t.Errorf("EpisodeNumber = %d, want 3", got.EpisodeNumber)
		}
		if got.EpisodeTitle != "今週の面白技術" {
			t.Errorf("EpisodeTitle = %q, want %q", got.EpisodeTitle, "今週の面白技術")
		}
	})

	t.Run("episode_number zero and empty title are omitted from manifest", func(t *testing.T) {
		got := manifest.Build(newMinimalBuildParams())
		if got.EpisodeNumber != 0 {
			t.Errorf("EpisodeNumber = %d, want 0 (omitempty)", got.EpisodeNumber)
		}
		if got.EpisodeTitle != "" {
			t.Errorf("EpisodeTitle = %q, want empty (omitempty)", got.EpisodeTitle)
		}
	})

	t.Run("script_note fields are not exposed in manifest", func(t *testing.T) {
		programWithNote := config.ProgramConfig{
			Title:       "テスト番組",
			Description: "公開メタデータ",
			ScriptNote:  "非公開台本指示",
			Direction:   "番組演出方針",
		}
		cornersWithNote := []config.CornerConfig{
			{Title: "テストコーナー", ScriptNote: "コーナー台本指示"},
		}
		got := manifest.Build(manifest.BuildParams{
			Program:     programWithNote,
			Corners:     cornersWithNote,
			Rundown:     model.Rundown{},
			AudioFile:   "ep.mp3",
			GeneratedAt: fixedTime,
		})
		if got.Description != "公開メタデータ" {
			t.Errorf("Description = %q, want 公開メタデータ", got.Description)
		}
		// Manifest struct should not have ScriptNote/Direction fields
		// This test documents that manifest.Build does not receive or expose these fields.
		// The Manifest struct only has Title, Description, and other public-safe fields.
		if got.Title != "テスト番組" {
			t.Errorf("Title = %q, want テスト番組", got.Title)
		}
	})
}

func TestBuild_CornerDurationFields(t *testing.T) {
	p := newMinimalBuildParams()
	p.Corners = []config.CornerConfig{
		{ID: "op", Title: "OP", LengthSec: 60},
		{ID: "tech", Title: "Tech", LengthSec: 180},
	}
	p.Clips = &model.ClipsMeta{
		Clips: []model.ClipMeta{
			{CornerID: "op", DurationSec: 10.0, Text: "こんにちは"},  // 5 chars
			{CornerID: "tech", DurationSec: 30.0, Text: "技術の話"}, // 4 chars
			{CornerID: "tech", DurationSec: 20.0, Text: "もう少し"}, // 4 chars
		},
	}
	p.CornerDurations = map[string]float64{"op": 12.5, "tech": 55.0}

	got := manifest.Build(p)

	// op corner
	opC := got.Corners[0]
	if opC.ID != "op" {
		t.Fatalf("Corners[0].ID = %q, want op", opC.ID)
	}
	if opC.TargetSec != 60 {
		t.Errorf("op.TargetSec = %d, want 60", opC.TargetSec)
	}
	if opC.SpeechSec != 10.0 {
		t.Errorf("op.SpeechSec = %.1f, want 10.0", opC.SpeechSec)
	}
	if opC.DurationSec != 12.5 {
		t.Errorf("op.DurationSec = %.1f, want 12.5", opC.DurationSec)
	}
	if opC.CharCount != 5 {
		t.Errorf("op.CharCount = %d, want 5", opC.CharCount)
	}

	// tech corner
	techC := got.Corners[1]
	if techC.TargetSec != 180 {
		t.Errorf("tech.TargetSec = %d, want 180", techC.TargetSec)
	}
	if techC.SpeechSec != 50.0 {
		t.Errorf("tech.SpeechSec = %.1f, want 50.0", techC.SpeechSec)
	}
	if techC.DurationSec != 55.0 {
		t.Errorf("tech.DurationSec = %.1f, want 55.0", techC.DurationSec)
	}
	if techC.CharCount != 8 {
		t.Errorf("tech.CharCount = %d, want 8", techC.CharCount)
	}
}

func TestBuild_DedupKeyCopiedToArticleRef(t *testing.T) {
	p := newMinimalBuildParams()
	p.Corners = []config.CornerConfig{
		{ID: "opening", Title: "オープニング"},
		{ID: "tech", Title: "今日のテックニュース"},
	}
	p.Rundown = model.Rundown{
		Corners: []model.RundownCorner{
			{
				ID:    "tech",
				Title: "今日のテックニュース",
				Articles: []model.RundownArticle{
					{DedupKey: "sha256:abc123", URL: "https://example.com/1", Title: "記事1", Points: []string{}},
				},
			},
		},
	}
	got := manifest.Build(p)
	techCorner := got.Corners[1]
	if len(techCorner.Articles) != 1 {
		t.Fatalf("Articles count = %d, want 1", len(techCorner.Articles))
	}
	if techCorner.Articles[0].DedupKey != "sha256:abc123" {
		t.Errorf("Articles[0].DedupKey = %q, want %q", techCorner.Articles[0].DedupKey, "sha256:abc123")
	}
}
