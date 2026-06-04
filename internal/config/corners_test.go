package config

import (
	"fmt"
	"testing"
)

func TestResolveCornersForEpisode(t *testing.T) {
	fixed := CornerConfig{Title: "固定コーナー", LengthSec: 30}
	cond1 := CornerConfig{
		Title:     "条件コーナーA",
		LengthSec: 60,
		Condition: &EpisodeCondition{Episodes: []int{1, 3, 5}},
	}
	cond2 := CornerConfig{
		Title:     "条件コーナーB",
		LengthSec: 60,
		Condition: &EpisodeCondition{Every: 2},
	}
	corners := []CornerConfig{fixed, cond1, cond2}

	tests := []struct {
		name          string
		episodeNumber int
		wantTitles    []string
	}{
		{
			name:          "固定コーナーは常に採用",
			episodeNumber: 1,
			wantTitles:    []string{"固定コーナー", "条件コーナーA"},
		},
		{
			name:          "偶数回はevery:2の条件コーナーのみ採用",
			episodeNumber: 2,
			wantTitles:    []string{"固定コーナー", "条件コーナーB"},
		},
		{
			name:          "奇数回でepisodesに含まれる回",
			episodeNumber: 3,
			wantTitles:    []string{"固定コーナー", "条件コーナーA"},
		},
		{
			name:          "いずれの条件にも合致しない回",
			episodeNumber: 7,
			wantTitles:    []string{"固定コーナー"},
		},
		{
			name:          "回番号不明(0)は全コーナー採用",
			episodeNumber: 0,
			wantTitles:    []string{"固定コーナー", "条件コーナーA", "条件コーナーB"},
		},
		{
			name:          "回番号不明(<0)は全コーナー採用",
			episodeNumber: -1,
			wantTitles:    []string{"固定コーナー", "条件コーナーA", "条件コーナーB"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveCornersForEpisode(corners, tt.episodeNumber)
			if len(got) != len(tt.wantTitles) {
				t.Fatalf("ResolveCornersForEpisode(_, %d) len = %d, want %d; got titles: %v",
					tt.episodeNumber, len(got), len(tt.wantTitles), cornerTitles(got))
			}
			for i, c := range got {
				if c.Title != tt.wantTitles[i] {
					t.Errorf("ResolveCornersForEpisode(_, %d)[%d].Title = %q, want %q",
						tt.episodeNumber, i, c.Title, tt.wantTitles[i])
				}
			}
		})
	}
}

func TestResolveCornersForEpisode_Not(t *testing.T) {
	condEvery5 := CornerConfig{
		Title:     "5回に1回",
		LengthSec: 60,
		Condition: &EpisodeCondition{Every: 5},
	}
	condNotEvery5 := CornerConfig{
		Title:     "5の倍数でない回",
		LengthSec: 60,
		Condition: &EpisodeCondition{Not: &EpisodeCondition{Every: 5}},
	}
	corners := []CornerConfig{condEvery5, condNotEvery5}

	tests := []struct {
		name          string
		episodeNumber int
		wantTitles    []string
	}{
		{
			name:          "5の倍数回: every:5 のみ採用",
			episodeNumber: 5,
			wantTitles:    []string{"5回に1回"},
		},
		{
			name:          "5の倍数でない回: not:{every:5} のみ採用",
			episodeNumber: 7,
			wantTitles:    []string{"5の倍数でない回"},
		},
		{
			name:          "10回: every:5 のみ採用",
			episodeNumber: 10,
			wantTitles:    []string{"5回に1回"},
		},
		{
			name:          "1回: not:{every:5} のみ採用",
			episodeNumber: 1,
			wantTitles:    []string{"5の倍数でない回"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveCornersForEpisode(corners, tt.episodeNumber)
			if len(got) != len(tt.wantTitles) {
				t.Fatalf("ResolveCornersForEpisode(_, %d) len = %d, want %d; got titles: %v",
					tt.episodeNumber, len(got), len(tt.wantTitles), cornerTitles(got))
			}
			for i, c := range got {
				if c.Title != tt.wantTitles[i] {
					t.Errorf("ResolveCornersForEpisode(_, %d)[%d].Title = %q, want %q",
						tt.episodeNumber, i, c.Title, tt.wantTitles[i])
				}
			}
		})
	}
}

func TestResolveCornersForEpisode_PreservesOrder(t *testing.T) {
	corners := []CornerConfig{
		{Title: "A", LengthSec: 10, Condition: &EpisodeCondition{Episodes: []int{1}}},
		{Title: "B", LengthSec: 10},
		{Title: "C", LengthSec: 10, Condition: &EpisodeCondition{Every: 2}},
		{Title: "D", LengthSec: 10},
	}
	got := ResolveCornersForEpisode(corners, 2)
	wantTitles := []string{"B", "C", "D"}
	if len(got) != len(wantTitles) {
		t.Fatalf("got %d corners, want %d", len(got), len(wantTitles))
	}
	for i, c := range got {
		if c.Title != wantTitles[i] {
			t.Errorf("order[%d]: got %q, want %q", i, c.Title, wantTitles[i])
		}
	}
}

func TestResolveCornersByTitles(t *testing.T) {
	corners := []CornerConfig{
		{Title: "A", LengthSec: 30},
		{Title: "B", LengthSec: 60},
		{Title: "C", LengthSec: 90},
	}

	t.Run("タイトル順にコーナーを返す", func(t *testing.T) {
		got, err := ResolveCornersByTitles(corners, []string{"C", "A"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d corners, want 2", len(got))
		}
		if got[0].Title != "C" || got[1].Title != "A" {
			t.Errorf("titles = %v, want [C A]", cornerTitles(got))
		}
	})

	t.Run("存在しないタイトルはエラー", func(t *testing.T) {
		_, err := ResolveCornersByTitles(corners, []string{"A", "X"})
		if err == nil {
			t.Error("expected error for unknown title, got nil")
		}
	})

	t.Run("空のtitlesは空スライスを返す", func(t *testing.T) {
		got, err := ResolveCornersByTitles(corners, []string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d corners, want 0", len(got))
		}
	})
}

func TestValidateEpisodeSpecCorners_Valid(t *testing.T) {
	tests := []struct {
		name    string
		corners []CornerConfig
	}{
		{
			name: "conditionなし（固定コーナー）のみ",
			corners: []CornerConfig{
				{Title: "A", LengthSec: 30},
				{Title: "B", LengthSec: 60},
			},
		},
		{
			name: "episodes指定の条件コーナー",
			corners: []CornerConfig{
				{Title: "A", LengthSec: 30},
				{Title: "B", LengthSec: 60, Condition: &EpisodeCondition{Episodes: []int{1, 3}}},
			},
		},
		{
			name: "every指定の条件コーナー",
			corners: []CornerConfig{
				{Title: "A", LengthSec: 30},
				{Title: "B", LengthSec: 60, Condition: &EpisodeCondition{Every: 2}},
			},
		},
		{
			name: "episodes+every両方指定",
			corners: []CornerConfig{
				{Title: "A", LengthSec: 30, Condition: &EpisodeCondition{Episodes: []int{1}, Every: 3}},
			},
		},
		{
			name: "not のみ指定",
			corners: []CornerConfig{
				{Title: "A", LengthSec: 30, Condition: &EpisodeCondition{Not: &EpisodeCondition{Every: 5}}},
			},
		},
		{
			name: "every + not 指定",
			corners: []CornerConfig{
				{Title: "A", LengthSec: 30, Condition: &EpisodeCondition{Every: 2, Not: &EpisodeCondition{Episodes: []int{6}}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &EpisodeSpec{Corners: tt.corners}
			if err := ValidateEpisodeSpecCorners(p); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateEpisodeSpecCorners_Error(t *testing.T) {
	tests := []struct {
		name    string
		corners []CornerConfig
	}{
		{
			name: "タイトルが重複",
			corners: []CornerConfig{
				{Title: "A", LengthSec: 30},
				{Title: "A", LengthSec: 60},
			},
		},
		{
			name: "conditionがあるがepisodesもeveryもnotも未設定",
			corners: []CornerConfig{
				{Title: "A", LengthSec: 30, Condition: &EpisodeCondition{}},
			},
		},
		{
			name: "not の中身が空",
			corners: []CornerConfig{
				{Title: "A", LengthSec: 30, Condition: &EpisodeCondition{Not: &EpisodeCondition{}}},
			},
		},
		{
			name: "episodesの値が0",
			corners: []CornerConfig{
				{Title: "A", LengthSec: 30, Condition: &EpisodeCondition{Episodes: []int{0}}},
			},
		},
		{
			name: "episodesの値が負数",
			corners: []CornerConfig{
				{Title: "A", LengthSec: 30, Condition: &EpisodeCondition{Episodes: []int{-1}}},
			},
		},
		{
			name: "everyが負数",
			corners: []CornerConfig{
				{Title: "A", LengthSec: 30, Condition: &EpisodeCondition{Every: -1}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &EpisodeSpec{Corners: tt.corners}
			if err := ValidateEpisodeSpecCorners(p); err == nil {
				t.Errorf("expected error for %q, got nil", tt.name)
			}
		})
	}
}

func TestResolveCornersForEpisode_Offset_ThreeRotation(t *testing.T) {
	corners := []CornerConfig{
		{Title: "コーナーA", LengthSec: 60, Condition: &EpisodeCondition{Every: 3, Offset: 1}}, // 1,4,7,...
		{Title: "コーナーB", LengthSec: 60, Condition: &EpisodeCondition{Every: 3, Offset: 2}}, // 2,5,8,...
		{Title: "コーナーC", LengthSec: 60, Condition: &EpisodeCondition{Every: 3, Offset: 0}}, // 3,6,9,...
	}

	tests := []struct {
		episodeNumber int
		wantTitle     string
	}{
		{episodeNumber: 1, wantTitle: "コーナーA"},
		{episodeNumber: 2, wantTitle: "コーナーB"},
		{episodeNumber: 3, wantTitle: "コーナーC"},
		{episodeNumber: 4, wantTitle: "コーナーA"},
		{episodeNumber: 5, wantTitle: "コーナーB"},
		{episodeNumber: 6, wantTitle: "コーナーC"},
		{episodeNumber: 7, wantTitle: "コーナーA"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("episode%d", tt.episodeNumber), func(t *testing.T) {
			got := ResolveCornersForEpisode(corners, tt.episodeNumber)
			if len(got) != 1 {
				t.Fatalf("episode %d: got %d corners, want 1; titles: %v",
					tt.episodeNumber, len(got), cornerTitles(got))
			}
			if got[0].Title != tt.wantTitle {
				t.Errorf("episode %d: got %q, want %q", tt.episodeNumber, got[0].Title, tt.wantTitle)
			}
		})
	}
}

func cornerTitles(corners []CornerConfig) []string {
	titles := make([]string, len(corners))
	for i, c := range corners {
		titles[i] = c.Title
	}
	return titles
}
