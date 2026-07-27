package retro_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/retro"
)

func TestNewEpisodeNumbers_FiltersToNewerThanLastEvaluated(t *testing.T) {
	entries := []cache.Entry{
		{EpisodeNumber: 14},
		{EpisodeNumber: 15},
		{EpisodeNumber: 16},
	}
	got := retro.NewEpisodeNumbers(entries, 15)
	if len(got) != 1 || got[0] != 16 {
		t.Errorf("NewEpisodeNumbers = %v, want [16]", got)
	}
}

func TestNewEpisodeNumbers_NoneNewerReturnsEmpty(t *testing.T) {
	entries := []cache.Entry{{EpisodeNumber: 10}, {EpisodeNumber: 12}}
	got := retro.NewEpisodeNumbers(entries, 12)
	if len(got) != 0 {
		t.Errorf("NewEpisodeNumbers = %v, want empty (no episode newer than last evaluated)", got)
	}
}

func TestApplyCounts_NonRecurringEpisodesAccumulateClearStreak(t *testing.T) {
	prevTry := []retro.Problem{
		{ID: "p1", Problem: "説明調", Action: "崩す", ClearStreak: 1},
	}
	result := retro.ApplyCounts(retro.ApplyCountsInput{
		PrevTryProblems: prevTry,
		PrevKeeps:       nil,
		ProposedProblems: []retro.Problem{
			{ID: "p1", Problem: "説明調", Action: "崩す"},
		},
		Recurrences:   nil, // p1 did not recur
		NewEpisodes:   []int{5, 6},
		KeepThreshold: 5,
		LatestEpisode: 6,
	})

	if len(result.NextTry) != 1 {
		t.Fatalf("NextTry = %+v, want 1 entry", result.NextTry)
	}
	if result.NextTry[0].ClearStreak != 3 {
		t.Errorf("ClearStreak = %d, want 3 (1 + 2 non-recurring episodes)", result.NextTry[0].ClearStreak)
	}
}

func TestApplyCounts_RecurrenceResetsStreakUpdatesLastSeenAndAction(t *testing.T) {
	prevTry := []retro.Problem{
		{ID: "p1", Problem: "説明調", Action: "旧施策", ClearStreak: 2, LastSeenEpisode: 3},
	}
	result := retro.ApplyCounts(retro.ApplyCountsInput{
		PrevTryProblems: prevTry,
		ProposedProblems: []retro.Problem{
			{ID: "p1", Problem: "説明調", Action: "新施策"},
		},
		Recurrences:   []retro.Recurrence{{ID: "p1", Episodes: []int{5}}},
		NewEpisodes:   []int{4, 5, 6},
		KeepThreshold: 5,
		LatestEpisode: 6,
	})

	if len(result.NextTry) != 1 {
		t.Fatalf("NextTry = %+v, want 1 entry", result.NextTry)
	}
	got := result.NextTry[0]
	if got.ClearStreak != 1 {
		t.Errorf("ClearStreak = %d, want 1 (reset at ep5, then +1 for ep6)", got.ClearStreak)
	}
	if got.LastSeenEpisode != 5 {
		t.Errorf("LastSeenEpisode = %d, want 5", got.LastSeenEpisode)
	}
	if got.Action != "新施策" {
		t.Errorf("Action = %q, want rewritten action after recurrence", got.Action)
	}
}

func TestApplyCounts_NoNewEpisodesDoesNotInflateStreak(t *testing.T) {
	prevTry := []retro.Problem{
		{ID: "p1", Problem: "x", Action: "y", ClearStreak: 2},
	}
	result := retro.ApplyCounts(retro.ApplyCountsInput{
		PrevTryProblems:  prevTry,
		ProposedProblems: []retro.Problem{{ID: "p1", Problem: "x", Action: "y"}},
		Recurrences:      nil,
		NewEpisodes:      nil, // no new episodes since last retro run
		KeepThreshold:    5,
		LatestEpisode:    0,
	})

	if result.NextTry[0].ClearStreak != 2 {
		t.Errorf("ClearStreak = %d, want unchanged at 2 (no new episodes)", result.NextTry[0].ClearStreak)
	}
}

func TestApplyCounts_PromotesToKeepAtThreshold(t *testing.T) {
	prevTry := []retro.Problem{
		{ID: "p1", Problem: "説明調", Action: "崩す", ClearStreak: 2},
	}
	result := retro.ApplyCounts(retro.ApplyCountsInput{
		PrevTryProblems:  prevTry,
		ProposedProblems: []retro.Problem{{ID: "p1", Problem: "説明調", Action: "崩す"}},
		Recurrences:      nil,
		NewEpisodes:      []int{7},
		KeepThreshold:    3,
		LatestEpisode:    7,
	})

	if len(result.NextTry) != 0 {
		t.Errorf("NextTry = %+v, want empty (promoted)", result.NextTry)
	}
	if len(result.NextKeep) != 1 {
		t.Fatalf("NextKeep = %+v, want 1 entry", result.NextKeep)
	}
	if result.NextKeep[0].ID != "p1" {
		t.Errorf("promoted keep ID = %q, want p1 (id carried over)", result.NextKeep[0].ID)
	}
	if result.NextKeep[0].ProvenAtEpisode != 7 {
		t.Errorf("ProvenAtEpisode = %d, want 7 (latest episode)", result.NextKeep[0].ProvenAtEpisode)
	}
	if len(result.PromotedIDs) != 1 || result.PromotedIDs[0] != "p1" {
		t.Errorf("PromotedIDs = %v, want [p1]", result.PromotedIDs)
	}
}

func TestApplyCounts_DemotesRecurringKeepAndDoesNotRewriteSurvivors(t *testing.T) {
	prevKeeps := []retro.Keep{
		{ID: "p1", Problem: "掛け合い説明調", Action: "崩す", ProvenAtEpisode: 10},
		{ID: "p2", Problem: "別の問題", Action: "別施策", ProvenAtEpisode: 12},
	}
	result := retro.ApplyCounts(retro.ApplyCountsInput{
		PrevKeeps: prevKeeps,
		ProposedProblems: []retro.Problem{
			{ID: "p1", Problem: "掛け合い説明調", Action: "新しい施策", FirstSeenEpisode: 15, LastSeenEpisode: 16},
		},
		Recurrences:   []retro.Recurrence{{ID: "p1", Episodes: []int{16}}},
		NewEpisodes:   []int{15, 16},
		KeepThreshold: 3,
		LatestEpisode: 16,
	})

	if len(result.NextKeep) != 1 || result.NextKeep[0].ID != "p2" {
		t.Fatalf("NextKeep = %+v, want only p2 to survive", result.NextKeep)
	}
	if result.NextKeep[0].Problem != "別の問題" || result.NextKeep[0].Action != "別施策" {
		t.Errorf("surviving keep entry was rewritten: %+v", result.NextKeep[0])
	}
	if len(result.NextTry) != 1 || result.NextTry[0].ID != "p1" {
		t.Fatalf("NextTry = %+v, want demoted p1", result.NextTry)
	}
	if result.NextTry[0].Action != "新しい施策" {
		t.Errorf("demoted problem Action = %q, want rewritten action from proposed problems", result.NextTry[0].Action)
	}
	if result.NextTry[0].ClearStreak != 0 {
		t.Errorf("demoted problem ClearStreak = %d, want 0", result.NextTry[0].ClearStreak)
	}
	if len(result.DemotedIDs) != 1 || result.DemotedIDs[0] != "p1" {
		t.Errorf("DemotedIDs = %v, want [p1]", result.DemotedIDs)
	}
}

func TestApplyCounts_NoIDCollisionBetweenTryAndKeep(t *testing.T) {
	prevTry := []retro.Problem{{ID: "p1", Problem: "a", Action: "a", ClearStreak: 2}}
	prevKeeps := []retro.Keep{{ID: "p2", Problem: "b", Action: "b", ProvenAtEpisode: 1}}

	result := retro.ApplyCounts(retro.ApplyCountsInput{
		PrevTryProblems:  prevTry,
		PrevKeeps:        prevKeeps,
		ProposedProblems: []retro.Problem{{ID: "p1", Problem: "a", Action: "a"}},
		Recurrences:      nil,
		NewEpisodes:      []int{2},
		KeepThreshold:    3,
		LatestEpisode:    2,
	})

	allIDs := make(map[string]bool)
	for _, p := range result.NextTry {
		if allIDs[p.ID] {
			t.Fatalf("id %q appears in both try and keep", p.ID)
		}
		allIDs[p.ID] = true
	}
	for _, k := range result.NextKeep {
		if allIDs[k.ID] {
			t.Fatalf("id %q appears in both try and keep", k.ID)
		}
		allIDs[k.ID] = true
	}
}

func TestApplyCounts_NewProblemNotInPrevIsAdded(t *testing.T) {
	result := retro.ApplyCounts(retro.ApplyCountsInput{
		ProposedProblems: []retro.Problem{
			{ID: "p1", Problem: "新規", Action: "施策", FirstSeenEpisode: 1, LastSeenEpisode: 1},
		},
		NewEpisodes:   []int{1},
		KeepThreshold: 3,
		LatestEpisode: 1,
	})

	if len(result.NextTry) != 1 || result.NextTry[0].ID != "p1" {
		t.Fatalf("NextTry = %+v, want new problem p1", result.NextTry)
	}
	if result.NextTry[0].ClearStreak != 0 {
		t.Errorf("new problem ClearStreak = %d, want 0", result.NextTry[0].ClearStreak)
	}
}

func TestApplyCounts_TruncatesFinalTryToMaxTries(t *testing.T) {
	prevTry := []retro.Problem{
		{ID: "p1", Problem: "1", Action: "a"},
		{ID: "p2", Problem: "2", Action: "a"},
		{ID: "p3", Problem: "3", Action: "a"},
	}
	result := retro.ApplyCounts(retro.ApplyCountsInput{
		PrevTryProblems: prevTry,
		ProposedProblems: []retro.Problem{
			{ID: "p1", Problem: "1", Action: "a"},
			{ID: "p2", Problem: "2", Action: "a"},
			{ID: "p3", Problem: "3", Action: "a"},
			{ID: "p4", Problem: "4 new", Action: "a", FirstSeenEpisode: 1, LastSeenEpisode: 1},
		},
		NewEpisodes:   []int{1},
		KeepThreshold: 100, // nothing promotes
		MaxTries:      3,
		LatestEpisode: 1,
	})

	if len(result.NextTry) != 3 {
		t.Errorf("NextTry len = %d, want 3 (truncated to MaxTries)", len(result.NextTry))
	}
}
