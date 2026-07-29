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
	PrevTryProblems  []Problem    // current try.yaml problems, with their real ClearStreak/FailStreak
	PrevKeeps        []Keep       // current keep.yaml entries
	PrevDropped      []Dropped    // current try.yaml dropped entries (carried forward, append-only)
	ProposedProblems []Problem    // LLMRetro.Run's `problems` output (already Go-ID-assigned)
	Recurrences      []Recurrence // LLMRetro.Run's recurrence judgments, by id
	NewEpisodes      []int        // episode numbers newly evaluated this round (> previous LastEvaluatedEpisode), ascending
	KeepThreshold    int          // consecutive non-recurring episodes required to promote to keep
	MaxTries         int          // cap on the final try problem count (0 = no cap)
	MaxFails         int          // consecutive recurring episodes after which a try problem is dropped (0 = no cap)
	LatestEpisode    int          // newest episode number evaluated; recorded as ProvenAtEpisode/DroppedAtEpisode
}

// ApplyCountsResult holds the next try/keep/dropped state and which ids moved this round (for logging).
type ApplyCountsResult struct {
	NextTry     []Problem
	NextKeep    []Keep
	NextDropped []Dropped
	PromotedIDs []string // try -> keep
	DemotedIDs  []string // keep -> try
	DroppedIDs  []string // try -> dropped
}

// ApplyCounts computes the next try/keep/dropped problem sets. Counting (ClearStreak/FailStreak)
// and promotion/demotion/drop are done here in Go, not by the LLM (ADR-0098): the LLM only judges,
// per id, which of NewEpisodes it recurred in (Recurrences); everything else is arithmetic.
//
// A try problem's ClearStreak grows by one per non-recurring episode in NewEpisodes and resets to
// 0 on recurrence; FailStreak does the opposite (grows on recurrence, resets on non-recurrence) —
// the two are mutually exclusive per episode. LastSeenEpisode/Problem/Action are updated from
// ProposedProblems on recurrence, since the previous action evidently did not work. Once
// ClearStreak reaches KeepThreshold the problem is promoted to keep; once FailStreak exceeds
// MaxFails the problem is dropped instead (a problem cannot do both in the same round, since
// reaching one resets the other to 0).
//
// A keep entry is never rewritten while it survives (ADR-0098): only a recurrence demotes it back
// to try (Problem/Action taken from ProposedProblems for that id). A demoted entry is subject to
// the same MaxFails drop check as any other try problem (a keep entry that immediately relapses
// chronically should not occupy a try slot either), but is not re-considered for promotion in the
// same round.
//
// Problems in ProposedProblems whose id is not already tracked (in PrevTryProblems or PrevKeeps)
// are brand new and are appended to try, unless their text matches a PrevDropped entry (retro must
// not resurrect a dropped problem the LLM proposed anyway, despite being told not to). The final
// try list is truncated to MaxTries (existing problems kept ahead of newly-added ones) when
// MaxTries > 0.
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
	nextDropped := append([]Dropped(nil), in.PrevDropped...)
	promoted := make([]string, 0)
	demoted := make([]string, 0)
	dropped := make([]string, 0)

	// dropIfChronic appends p to nextDropped and reports true if its FailStreak exceeds MaxFails.
	// Shared by the try-problem loop and the keep-demotion loop, since both can produce a Problem
	// whose FailStreak already exceeds the chronic-failure threshold.
	dropIfChronic := func(p Problem) bool {
		if in.MaxFails <= 0 || p.FailStreak <= in.MaxFails {
			return false
		}
		nextDropped = append(nextDropped, Dropped{
			ID: p.ID, Problem: p.Problem, Action: p.Action,
			FirstSeenEpisode: p.FirstSeenEpisode, DroppedAtEpisode: in.LatestEpisode,
			FailStreak: p.FailStreak,
		})
		dropped = append(dropped, p.ID)
		return true
	}

	for _, p := range in.PrevTryProblems {
		clearStreak, failStreak, recurred, lastSeen := nextStreaks(p.ClearStreak, p.FailStreak, in.NewEpisodes, recurredEpisodesByID[p.ID])
		p.ClearStreak = clearStreak
		p.FailStreak = failStreak
		if recurred {
			p.LastSeenEpisode = lastSeen
			if lp, ok := proposedByID[p.ID]; ok {
				p.Problem = lp.Problem
				p.Action = lp.Action
			}
		}

		if dropIfChronic(p) {
			continue
		}
		if in.KeepThreshold > 0 && p.ClearStreak >= in.KeepThreshold {
			nextKeep = append(nextKeep, Keep{ID: p.ID, Problem: p.Problem, Action: p.Action, ProvenAtEpisode: in.LatestEpisode})
			promoted = append(promoted, p.ID)
			continue
		}
		nextTry = append(nextTry, p)
	}

	existingIDs := trackedIDs(in.PrevTryProblems, in.PrevKeeps)
	existingProblemTexts := problemTextIndex(in.PrevTryProblems, in.PrevKeeps)
	droppedProblemTexts := make(map[string]bool, len(in.PrevDropped))
	for _, d := range in.PrevDropped {
		droppedProblemTexts[normalizeProblemText(d.Problem)] = true
	}

	for _, k := range in.PrevKeeps {
		clearStreak, failStreak, recurred, lastSeen := nextStreaks(0, 0, in.NewEpisodes, recurredEpisodesByID[k.ID])
		if !recurred {
			nextKeep = append(nextKeep, k) // survives unchanged; retro never rewrites a kept entry's wording
			continue
		}
		demotedProblem := Problem{ID: k.ID, Problem: k.Problem, Action: k.Action, ClearStreak: clearStreak, FailStreak: failStreak, LastSeenEpisode: lastSeen}
		if lp, ok := proposedByID[k.ID]; ok {
			demotedProblem.Problem = lp.Problem
			demotedProblem.Action = lp.Action
			demotedProblem.FirstSeenEpisode = lp.FirstSeenEpisode
		}
		if !dropIfChronic(demotedProblem) {
			nextTry = append(nextTry, demotedProblem)
		}
		demoted = append(demoted, k.ID)
	}

	for _, lp := range in.ProposedProblems {
		if existingIDs[lp.ID] {
			continue
		}
		// Backstop for assignIDs: a proposed problem whose text matches a tracked problem/keep is
		// a continuation retro.go failed to resolve to an existing id, not a genuine new problem.
		// Appending it here would duplicate the entry and split its ClearStreak across two ids.
		if _, seen := existingProblemTexts[normalizeProblemText(lp.Problem)]; seen {
			continue
		}
		// Backstop for the prompt instruction: a proposed problem matching a dropped entry's text
		// must not be resurrected even if the LLM proposes it anyway.
		if droppedProblemTexts[normalizeProblemText(lp.Problem)] {
			continue
		}
		nextTry = append(nextTry, lp)
	}

	if in.MaxTries > 0 && len(nextTry) > in.MaxTries {
		nextTry = nextTry[:in.MaxTries]
	}

	return ApplyCountsResult{
		NextTry:     model.NonNil(nextTry),
		NextKeep:    model.NonNil(nextKeep),
		NextDropped: model.NonNil(nextDropped),
		PromotedIDs: model.NonNil(promoted),
		DemotedIDs:  model.NonNil(demoted),
		DroppedIDs:  model.NonNil(dropped),
	}
}

// nextStreaks walks newEpisodes (ascending) and returns the updated ClearStreak/FailStreak, whether
// any recurrence happened this round, and the last recurring episode (0 if none recurred — callers
// should keep the previous LastSeenEpisode in that case). The two streaks are exclusive: each
// episode resets exactly one of them to 0 while incrementing the other.
func nextStreaks(currentClearStreak, currentFailStreak int, newEpisodes []int, recurredEpisodes []int) (clearStreak, failStreak int, recurred bool, lastSeen int) {
	recurredSet := make(map[int]bool, len(recurredEpisodes))
	for _, ep := range recurredEpisodes {
		recurredSet[ep] = true
	}

	clearStreak = currentClearStreak
	failStreak = currentFailStreak
	for _, ep := range newEpisodes {
		if recurredSet[ep] {
			clearStreak = 0
			failStreak++
			lastSeen = ep
			recurred = true
		} else {
			clearStreak++
			failStreak = 0
		}
	}
	return clearStreak, failStreak, recurred, lastSeen
}
