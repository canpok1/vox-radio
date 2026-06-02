package write

import (
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/model"
)

func TestFormatPastEpisodes(t *testing.T) {
	tests := []struct {
		name            string
		eps             []cache.Entry
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:         "空エピソード",
			eps:          []cache.Entry{},
			wantContains: []string{"（なし）"},
		},
		{
			name: "全フィールドあり単一エピソード",
			eps: []cache.Entry{
				{
					ProgramID: "prog-id",
					Title:     "エピソードタイトル",
					Datetime:  "2024-01-01T10:00:00Z",
					Summary:   "先週の要約",
					Corners: []cache.CornerEntry{
						{
							Title:   "コーナー1",
							Summary: "コーナー概要",
							Points:  []string{"ポイント1", "ポイント2"},
							Articles: []cache.ArticleEntry{
								{Title: "記事タイトル", URL: "https://example.com/old"},
							},
						},
					},
					ConversationNotes: []model.ConversationNote{
						{
							Category:     "近況",
							CharacterIDs: []string{"zundamon", "metan"},
							Note:         "ずんだもんが元気だった",
						},
					},
				},
			},
			wantContains: []string{
				"## 過去のエピソード（新しい順）",
				"### 2024-01-01T10:00:00Z",
				"概要: 先週の要約",
				"- コーナー1: コーナー概要",
				"  ・ポイント1",
				"  ・ポイント2",
				"会話メモ:",
				"- (近況/zundamon,metan) ずんだもんが元気だった",
			},
			wantNotContains: []string{
				"prog-id",
				"エピソードタイトル",
				"https://example.com/old",
				"記事タイトル",
			},
		},
		{
			name: "Summary空で概要行省略",
			eps: []cache.Entry{
				{
					Datetime: "2024-01-01T10:00:00Z",
					Summary:  "",
					Corners:  []cache.CornerEntry{{Title: "コーナー1", Summary: "コーナー概要"}},
				},
			},
			wantNotContains: []string{"概要:"},
		},
		{
			name: "Corners空でコーナー行省略",
			eps: []cache.Entry{
				{
					Datetime:          "2024-01-01T10:00:00Z",
					Summary:           "エピソード概要",
					Corners:           []cache.CornerEntry{},
					ConversationNotes: []model.ConversationNote{},
				},
			},
			wantContains:    []string{"概要: エピソード概要"},
			wantNotContains: []string{"コーナー"},
		},
		{
			name: "ConversationNotes空で会話メモセクション省略",
			eps: []cache.Entry{
				{
					Datetime:          "2024-01-01T10:00:00Z",
					Summary:           "エピソード概要",
					ConversationNotes: []model.ConversationNote{},
				},
			},
			wantNotContains: []string{"会話メモ:"},
		},
		{
			name: "コーナーのPoints空でbullet省略",
			eps: []cache.Entry{
				{
					Datetime: "2024-01-01T10:00:00Z",
					Corners: []cache.CornerEntry{
						{Title: "コーナー1", Summary: "コーナー概要", Points: []string{}},
					},
				},
			},
			wantContains:    []string{"- コーナー1: コーナー概要"},
			wantNotContains: []string{"・"},
		},
		{
			name: "CharacterIDs空で会話メモにスラッシュなし",
			eps: []cache.Entry{
				{
					Datetime: "2024-01-01T10:00:00Z",
					ConversationNotes: []model.ConversationNote{
						{Category: "感想", CharacterIDs: []string{}, Note: "楽しかった"},
					},
				},
			},
			wantContains:    []string{"- (感想) 楽しかった"},
			wantNotContains: []string{"感想/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPastEpisodes(tt.eps)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("formatPastEpisodes() should contain %q\ngot:\n%s", want, got)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(got, notWant) {
					t.Errorf("formatPastEpisodes() should NOT contain %q\ngot:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestFormatPastEpisodes_MultipleEpisodes_NewestFirst(t *testing.T) {
	eps := []cache.Entry{
		{Datetime: "2024-01-01T10:00:00Z", Summary: "古い概要"},
		{Datetime: "2024-01-08T10:00:00Z", Summary: "新しい概要"},
	}
	got := formatPastEpisodes(eps)

	idxNew := strings.Index(got, "新しい概要")
	idxOld := strings.Index(got, "古い概要")
	if idxNew == -1 {
		t.Error("formatPastEpisodes() should contain '新しい概要'")
	}
	if idxOld == -1 {
		t.Error("formatPastEpisodes() should contain '古い概要'")
	}
	if idxNew != -1 && idxOld != -1 && idxNew > idxOld {
		t.Errorf("'新しい概要' (idx=%d) should appear before '古い概要' (idx=%d) in output:\n%s", idxNew, idxOld, got)
	}
}

func TestFormatPastEpisodes_EpisodeNumber_IsDisplayed(t *testing.T) {
	eps := []cache.Entry{
		{Datetime: "2024-01-01T10:00:00Z", EpisodeNumber: 3, EpisodeTitle: "今週のAI特集", Summary: "概要"},
	}
	got := formatPastEpisodes(eps)
	if !strings.Contains(got, "第3回") {
		t.Errorf("formatPastEpisodes() should contain '第3回', got:\n%s", got)
	}
	if !strings.Contains(got, "今週のAI特集") {
		t.Errorf("formatPastEpisodes() should contain '今週のAI特集', got:\n%s", got)
	}
}

func TestFormatPastEpisodes_EpisodeNumberWithoutTitle_IsDisplayed(t *testing.T) {
	eps := []cache.Entry{
		{Datetime: "2024-01-01T10:00:00Z", EpisodeNumber: 5, Summary: "概要"},
	}
	got := formatPastEpisodes(eps)
	if !strings.Contains(got, "第5回") {
		t.Errorf("formatPastEpisodes() should contain '第5回', got:\n%s", got)
	}
	// No episode title: heading should be "第5回 <datetime>" without parenthesized title
	if strings.Contains(got, "第5回（") {
		t.Errorf("formatPastEpisodes() should NOT contain '第5回（' when no title, got:\n%s", got)
	}
}

func TestFormatPastEpisodes_LegacyEntry_NoEpisodeNumberDisplay(t *testing.T) {
	eps := []cache.Entry{
		{Datetime: "2024-01-01T10:00:00Z", EpisodeNumber: 0, Summary: "概要"},
	}
	got := formatPastEpisodes(eps)
	if strings.Contains(got, "第0回") {
		t.Errorf("formatPastEpisodes() should NOT contain '第0回' for legacy entries, got:\n%s", got)
	}
}

func TestFormatPastEpisodes_NewTechnicalFields_NotLeakedToLLM(t *testing.T) {
	eps := []cache.Entry{
		{
			Datetime:    "2024-01-01T10:00:00Z",
			Summary:     "エピソード概要",
			Description: "番組説明テキスト",
			AudioFile:   "episode.mp3",
			Bytes:       12345678,
			DurationSec: 1800,
		},
	}
	got := formatPastEpisodes(eps)

	for _, notWant := range []string{"番組説明テキスト", "episode.mp3", "12345678", "1800"} {
		if strings.Contains(got, notWant) {
			t.Errorf("formatPastEpisodes() should NOT contain %q\ngot:\n%s", notWant, got)
		}
	}
	if !strings.Contains(got, "エピソード概要") {
		t.Errorf("formatPastEpisodes() should contain 'エピソード概要'")
	}
}
