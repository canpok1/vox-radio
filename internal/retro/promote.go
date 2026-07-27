package retro

import (
	"sort"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/model"
)

// NewEpisodeNumbers returns the distinct episode numbers among entries that are newer than
// lastEvaluated, sorted ascending. Passing the previous TryFile.LastEvaluatedEpisode here is what
// keeps re-running retro without new episodes from inflating ClearStreak (ADR-0098).
func NewEpisodeNumbers(entries []cache.Entry, lastEvaluated int) []int {
	seen := make(map[int]bool)
	out := make([]int, 0, len(entries))
	for _, e := range entries {
		if e.EpisodeNumber > lastEvaluated && !seen[e.EpisodeNumber] {
			seen[e.EpisodeNumber] = true
			out = append(out, e.EpisodeNumber)
		}
	}
	sort.Ints(out)
	return out
}

// ApplyCountsInput holds everything ApplyCounts needs to compute the next try/keep state.
type ApplyCountsInput struct {
	PrevTryProblems  []Problem    // current try.yaml problems, with their real ClearStreak
	PrevKeeps        []Keep       // current keep.yaml entries
	ProposedProblems []Problem    // LLMRetro.Run's `problems` output (already Go-ID-assigned)
	Recurrences      []Recurrence // LLMRetro.Run's recurrence judgments, by id
	NewEpisodes      []int        // episode numbers newly evaluated this round (> previous LastEvaluatedEpisode), ascending
	KeepThreshold    int          // consecutive non-recurring episodes required to promote to keep
	MaxTries         int          // cap on the final try problem count (0 = no cap)
	LatestEpisode    int          // newest episode number evaluated; recorded as ProvenAtEpisode on promotion
}

// ApplyCountsResult holds the next try/keep state and which ids moved this round (for logging).
type ApplyCountsResult struct {
	NextTry     []Problem
	NextKeep    []Keep
	PromotedIDs []string // try -> keep
	DemotedIDs  []string // keep -> try
}

// ApplyCounts computes the next try/keep problem sets. Counting (ClearStreak) and promotion/
// demotion are done here in Go, not by the LLM (ADR-0098): the LLM only judges, per id, which of
// NewEpisodes it recurred in (Recurrences); everything else is arithmetic.
//
// A try problem's ClearStreak grows by one per non-recurring episode in NewEpisodes and resets to
// 0 on recurrence (LastSeenEpisode and Action are updated from ProposedProblems at that point,
// since the previous action evidently did not work). Once ClearStreak reaches KeepThreshold the
// problem is promoted to keep and removed from try.
//
// A keep entry is never rewritten while it survives (ADR-0098): only a recurrence demotes it back
// to try (fresh, ClearStreak 0, Problem/Action taken from ProposedProblems for that id).
//
// Problems in ProposedProblems whose id is not already tracked (in PrevTryProblems or PrevKeeps)
// are brand new and are appended to try. The final try list is truncated to MaxTries (existing
// problems kept ahead of newly-added ones) when MaxTries > 0.
func ApplyCounts(in ApplyCountsInput) ApplyCountsResult {
	proposedByID := make(map[string]Problem, len(in.ProposedProblems))
	for _, p := range in.ProposedProblems {
		proposedByID[p.ID] = p
	}
	recurredEpisodesByID := make(map[string][]int, len(in.Recurrences))
	for _, r := range in.Recurrences {
		recurredEpisodesByID[r.ID] = r.Episodes
	}

	nextTry := make([]Problem, 0, len(in.PrevTryProblems))
	nextKeep := make([]Keep, 0, len(in.PrevKeeps))
	promoted := make([]string, 0)
	demoted := make([]string, 0)

	for _, p := range in.PrevTryProblems {
		streak, recurred, lastSeen := nextClearStreak(p.ClearStreak, in.NewEpisodes, recurredEpisodesByID[p.ID])
		p.ClearStreak = streak
		if recurred {
			p.LastSeenEpisode = lastSeen
			if lp, ok := proposedByID[p.ID]; ok {
				p.Problem = lp.Problem
				p.Action = lp.Action
			}
		}

		if in.KeepThreshold > 0 && p.ClearStreak >= in.KeepThreshold {
			nextKeep = append(nextKeep, Keep{ID: p.ID, Problem: p.Problem, Action: p.Action, ProvenAtEpisode: in.LatestEpisode})
			promoted = append(promoted, p.ID)
			continue
		}
		nextTry = append(nextTry, p)
	}

	existingIDs := trackedIDs(in.PrevTryProblems, in.PrevKeeps)

	for _, k := range in.PrevKeeps {
		episodes := recurredEpisodesByID[k.ID]
		if len(episodes) == 0 {
			nextKeep = append(nextKeep, k) // survives unchanged; retro never rewrites a kept entry's wording
			continue
		}
		demotedProblem := Problem{ID: k.ID, Problem: k.Problem, Action: k.Action, ClearStreak: 0, LastSeenEpisode: episodes[len(episodes)-1]}
		if lp, ok := proposedByID[k.ID]; ok {
			demotedProblem.Problem = lp.Problem
			demotedProblem.Action = lp.Action
			demotedProblem.FirstSeenEpisode = lp.FirstSeenEpisode
		}
		nextTry = append(nextTry, demotedProblem)
		demoted = append(demoted, k.ID)
	}

	for _, lp := range in.ProposedProblems {
		if !existingIDs[lp.ID] {
			nextTry = append(nextTry, lp)
		}
	}

	if in.MaxTries > 0 && len(nextTry) > in.MaxTries {
		nextTry = nextTry[:in.MaxTries]
	}

	return ApplyCountsResult{
		NextTry:     model.NonNil(nextTry),
		NextKeep:    model.NonNil(nextKeep),
		PromotedIDs: model.NonNil(promoted),
		DemotedIDs:  model.NonNil(demoted),
	}
}

// nextClearStreak walks newEpisodes (ascending) and returns the updated streak, whether any
// recurrence happened this round, and the last recurring episode (0 if none recurred — callers
// should keep the previous LastSeenEpisode in that case).
func nextClearStreak(currentStreak int, newEpisodes []int, recurredEpisodes []int) (streak int, recurred bool, lastSeen int) {
	recurredSet := make(map[int]bool, len(recurredEpisodes))
	for _, ep := range recurredEpisodes {
		recurredSet[ep] = true
	}

	streak = currentStreak
	for _, ep := range newEpisodes {
		if recurredSet[ep] {
			streak = 0
			lastSeen = ep
			recurred = true
		} else {
			streak++
		}
	}
	return streak, recurred, lastSeen
}
