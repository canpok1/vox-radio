package retro_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/retro"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

type mockClient struct {
	response json.RawMessage
	err      error
	captured []llm.CompletionRequest
}

func (m *mockClient) Complete(_ context.Context, req llm.CompletionRequest) (json.RawMessage, error) {
	m.captured = append(m.captured, req)
	return m.response, m.err
}

func TestLoadTryFile_MissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "try.yaml")

	tf, err := retro.LoadTryFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tf.Problems == nil {
		t.Error("Problems is nil, want empty non-nil slice")
	}
	if len(tf.Problems) != 0 {
		t.Errorf("Problems len = %d, want 0", len(tf.Problems))
	}
}

func TestNewTryFile_NonNilProblemsAndFields(t *testing.T) {
	tf := retro.NewTryFile(nil, nil, 5, "2026-07-27T10:00:00Z")
	if tf.GeneratedAt != "2026-07-27T10:00:00Z" || tf.LastEvaluatedEpisode != 5 {
		t.Errorf("NewTryFile fields = %+v, unexpected", tf)
	}
	if tf.Problems == nil {
		t.Error("Problems is nil, want empty non-nil slice")
	}
}

func TestNewKeepFile_NonNilKeepsAndFields(t *testing.T) {
	kf := retro.NewKeepFile(nil, "2026-07-27T10:00:00Z")
	if kf.GeneratedAt != "2026-07-27T10:00:00Z" {
		t.Errorf("GeneratedAt = %q, want 2026-07-27T10:00:00Z", kf.GeneratedAt)
	}
	if kf.Keeps == nil {
		t.Error("Keeps is nil, want empty non-nil slice")
	}
}

func TestSaveTryFile_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "try.yaml")
	want := retro.TryFile{
		GeneratedAt:          "2026-07-27T10:00:00Z",
		LastEvaluatedEpisode: 16,
		Problems: []retro.Problem{
			{ID: "p1", Problem: "掛け合いが説明調", Action: "疑問形で崩す", FirstSeenEpisode: 12, LastSeenEpisode: 16, ClearStreak: 0},
		},
	}

	if err := retro.SaveTryFile(path, want); err != nil {
		t.Fatalf("SaveTryFile: %v", err)
	}

	got, err := retro.LoadTryFile(path)
	if err != nil {
		t.Fatalf("LoadTryFile: %v", err)
	}
	if got.LastEvaluatedEpisode != 16 {
		t.Errorf("LastEvaluatedEpisode = %d, want 16", got.LastEvaluatedEpisode)
	}
	if len(got.Problems) != 1 || got.Problems[0].ID != "p1" {
		t.Fatalf("Problems = %+v, want 1 entry with id p1", got.Problems)
	}
	if got.Problems[0].Action != "疑問形で崩す" {
		t.Errorf("Action = %q, want 疑問形で崩す", got.Problems[0].Action)
	}
}

func TestSaveTryFile_RoundTrip_PreservesDroppedAndFailStreak(t *testing.T) {
	path := filepath.Join(t.TempDir(), "try.yaml")
	want := retro.TryFile{
		GeneratedAt:          "2026-07-29T10:00:00Z",
		LastEvaluatedEpisode: 83,
		Problems: []retro.Problem{
			{ID: "p2", Problem: "議論が単調", Action: "役割を固定しない", ClearStreak: 0, FailStreak: 2},
		},
		Dropped: []retro.Dropped{
			{ID: "p1", Problem: "尺超過", Action: "旧施策", FirstSeenEpisode: 79, DroppedAtEpisode: 83, FailStreak: 6},
		},
	}

	if err := retro.SaveTryFile(path, want); err != nil {
		t.Fatalf("SaveTryFile: %v", err)
	}

	got, err := retro.LoadTryFile(path)
	if err != nil {
		t.Fatalf("LoadTryFile: %v", err)
	}
	if len(got.Problems) != 1 || got.Problems[0].FailStreak != 2 {
		t.Fatalf("Problems = %+v, want 1 entry with FailStreak 2", got.Problems)
	}
	if len(got.Dropped) != 1 {
		t.Fatalf("Dropped = %+v, want 1 entry", got.Dropped)
	}
	d := got.Dropped[0]
	if d.ID != "p1" || d.Problem != "尺超過" || d.FirstSeenEpisode != 79 || d.DroppedAtEpisode != 83 || d.FailStreak != 6 {
		t.Errorf("Dropped[0] = %+v, unexpected", d)
	}
}

func TestSaveTryFile_IncludesHeaderComment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "try.yaml")
	if err := retro.SaveTryFile(path, retro.TryFile{Problems: make([]retro.Problem, 0)}); err != nil {
		t.Fatalf("SaveTryFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("file is empty")
	}
	if data[0] != '#' {
		t.Errorf("file should start with a comment header, got: %s", string(data[:min(30, len(data))]))
	}
}

func TestLoadKeepFile_MissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keep.yaml")

	kf, err := retro.LoadKeepFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kf.Keeps == nil || len(kf.Keeps) != 0 {
		t.Errorf("Keeps = %+v, want empty non-nil slice", kf.Keeps)
	}
}

func TestSaveKeepFile_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "keep.yaml")
	want := retro.KeepFile{
		GeneratedAt: "2026-07-27T10:00:00Z",
		Keeps: []retro.Keep{
			{ID: "p1", Problem: "掛け合いが説明調", Action: "疑問形で崩す", ProvenAtEpisode: 19},
		},
	}

	if err := retro.SaveKeepFile(path, want); err != nil {
		t.Fatalf("SaveKeepFile: %v", err)
	}

	got, err := retro.LoadKeepFile(path)
	if err != nil {
		t.Fatalf("LoadKeepFile: %v", err)
	}
	if len(got.Keeps) != 1 || got.Keeps[0].ID != "p1" || got.Keeps[0].ProvenAtEpisode != 19 {
		t.Fatalf("Keeps = %+v, unexpected", got.Keeps)
	}
}

func TestSaveKeepFile_IncludesHeaderComment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keep.yaml")
	if err := retro.SaveKeepFile(path, retro.KeepFile{Keeps: make([]retro.Keep, 0)}); err != nil {
		t.Fatalf("SaveKeepFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(data) == 0 || data[0] != '#' {
		t.Errorf("file should start with a comment header, got: %s", string(data[:min(30, len(data))]))
	}
}

func TestKeepContentLength_SumsProblemAndActionRunes(t *testing.T) {
	kf := retro.KeepFile{Keeps: []retro.Keep{
		{Problem: "あい", Action: "うえお"}, // 2 + 3 runes
		{Problem: "x", Action: "y"},    // 1 + 1
	}}
	if got := retro.KeepContentLength(kf); got != 7 {
		t.Errorf("KeepContentLength = %d, want 7", got)
	}
}

func TestMarshalTryFile_EmptyProblemsRendersEmptyArray(t *testing.T) {
	content, err := retro.MarshalTryFile(retro.TryFile{GeneratedAt: "x", Problems: make([]retro.Problem, 0)})
	if err != nil {
		t.Fatalf("MarshalTryFile: %v", err)
	}
	if !strings.Contains(string(content), "problems: []") {
		t.Errorf("expected \"problems: []\" in output, got: %s", content)
	}
}

func TestFilterAnalyzed_SkipsNilAnalysis(t *testing.T) {
	entries := []cache.Entry{
		{EpisodeNumber: 1, Analysis: nil},
		{EpisodeNumber: 2, Analysis: &model.Analysis{}},
		{EpisodeNumber: 3, Analysis: nil},
	}
	got := retro.FilterAnalyzed(entries)
	if len(got) != 1 || got[0].EpisodeNumber != 2 {
		t.Errorf("FilterAnalyzed = %+v, want only episode 2", got)
	}
}

func TestLLMRetro_Run_SingleCallAssignsNewIDs(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"problem": "p1の内容", "action": "a1", "first_seen_episode": 10, "last_seen_episode": 12}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}} {{current_problems}} {{program}} {{max_tries}}", 0)

	entries := []cache.Entry{
		{EpisodeNumber: 12, Analysis: &model.Analysis{}},
	}

	got, _, lastEvaluated, err := r.Run(context.Background(), config.ProgramConfig{Title: "テスト"}, entries, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mc.captured) != 1 {
		t.Fatalf("expected exactly 1 LLM call, got %d", len(mc.captured))
	}
	if lastEvaluated != 12 {
		t.Errorf("lastEvaluated = %d, want 12", lastEvaluated)
	}
	if len(got) != 1 || got[0].ID != "p1" {
		t.Fatalf("got = %+v, want 1 problem with id p1", got)
	}
	if got[0].ClearStreak != 0 {
		t.Errorf("ClearStreak = %d, want 0", got[0].ClearStreak)
	}
}

func TestLLMRetro_Run_PromptContainsCurrentDropped(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{"problems": []}`)}
	r := retro.NewLLMRetro(mc, "dropped={{current_dropped}}", 0)
	currentDropped := []retro.Dropped{
		{ID: "p1", Problem: "尺超過の課題", Action: "旧施策", FirstSeenEpisode: 79, DroppedAtEpisode: 83, FailStreak: 6},
	}
	entries := []cache.Entry{{EpisodeNumber: 90, Analysis: &model.Analysis{}}}

	_, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, nil, nil, currentDropped, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mc.captured) != 1 {
		t.Fatalf("expected exactly 1 LLM call, got %d", len(mc.captured))
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "尺超過の課題") {
		t.Errorf("prompt should contain the dropped problem text, got: %s", prompt)
	}
	if strings.Contains(prompt, "旧施策") {
		t.Errorf("prompt should not expose the dropped entry's action (irrelevant once dropped), got: %s", prompt)
	}
}

func TestLLMRetro_Run_PromptContainsProgramRetroNote(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{"problems": []}`)}
	r := retro.NewLLMRetro(mc, "{{program}}", 0)
	program := config.ProgramConfig{RetroNote: "尺の乖離ではなく掛け合いの質に注力してほしい"}
	entries := []cache.Entry{{EpisodeNumber: 1, Analysis: &model.Analysis{}}}

	_, _, _, err := r.Run(context.Background(), program, entries, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mc.captured) != 1 {
		t.Fatalf("expected exactly 1 LLM call, got %d", len(mc.captured))
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "尺の乖離ではなく掛け合いの質に注力してほしい") {
		t.Errorf("prompt should contain program.RetroNote, got: %s", prompt)
	}
}

func TestLLMRetro_Run_EmptyProgramRetroNote_BackwardCompatible(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{"problems": []}`)}
	r := retro.NewLLMRetro(mc, "{{program}}", 0)
	entries := []cache.Entry{{EpisodeNumber: 1, Analysis: &model.Analysis{}}}

	_, _, _, err := r.Run(context.Background(), config.ProgramConfig{Title: "テスト"}, entries, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, `"retro_note":""`) {
		t.Errorf("empty RetroNote should marshal as an empty retro_note field (unchanged behavior), got: %s", prompt)
	}
}

func TestLLMRetro_Run_PreservesExistingIDAndAssignsNextForNew(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"id": "p2", "problem": "継続中の問題", "action": "改善版の施策", "first_seen_episode": 10, "last_seen_episode": 16},
			{"problem": "新規の問題", "action": "新しい施策", "first_seen_episode": 16, "last_seen_episode": 16}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}}", 0)

	current := []retro.Problem{
		{ID: "p2", Problem: "継続中の問題", Action: "旧施策", FirstSeenEpisode: 10, LastSeenEpisode: 14},
	}
	entries := []cache.Entry{{EpisodeNumber: 16, Analysis: &model.Analysis{}}}

	got, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, current, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d problems, want 2", len(got))
	}
	if got[0].ID != "p2" {
		t.Errorf("continuing problem ID = %q, want p2 (preserved)", got[0].ID)
	}
	if got[0].Action != "改善版の施策" {
		t.Errorf("continuing problem Action = %q, want rewritten action", got[0].Action)
	}
	if got[1].ID != "p1" {
		t.Errorf("new problem ID = %q, want p1 (next unused)", got[1].ID)
	}
}

func TestLLMRetro_Run_IgnoresHallucinatedIDForNewProblem(t *testing.T) {
	// The LLM returns a non-empty id ("p9") for a problem that does not exist in `current`.
	// Go must not trust it (it could collide with a future real p9), so it gets reassigned.
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"id": "p9", "problem": "実は新規の問題", "action": "a", "first_seen_episode": 5, "last_seen_episode": 5}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}}", 0)
	current := []retro.Problem{{ID: "p1", Problem: "既存", Action: "a", FirstSeenEpisode: 1, LastSeenEpisode: 1}}
	entries := []cache.Entry{{EpisodeNumber: 5, Analysis: &model.Analysis{}}}

	got, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, current, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d problems, want 1", len(got))
	}
	if got[0].ID == "p9" {
		t.Errorf("ID = %q, the hallucinated id p9 (not in current) must not be trusted verbatim", got[0].ID)
	}
	if got[0].ID != "p2" {
		t.Errorf("ID = %q, want p2 (next unused, since p1 is taken by current)", got[0].ID)
	}
}

func TestLLMRetro_Run_MatchesContinuingProblemByTextWhenIDMissing(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"problem": "掛け合いが説明調", "action": "改善版の施策", "first_seen_episode": 10, "last_seen_episode": 16}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}}", 0)
	current := []retro.Problem{
		{ID: "p1", Problem: "掛け合いが説明調", Action: "旧施策", FirstSeenEpisode: 10, LastSeenEpisode: 14},
	}
	entries := []cache.Entry{{EpisodeNumber: 16, Analysis: &model.Analysis{}}}

	got, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, current, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "p1" {
		t.Errorf("got = %+v, want single problem matched to p1 by text (no duplicate created)", got)
	}
}

func TestLLMRetro_Run_MatchesContinuingProblemDespiteWhitespaceDifferences(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"problem": "掛け合いが　記事の要点を\t並べる説明に寄りがち", "action": "施策", "first_seen_episode": 10, "last_seen_episode": 16}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}}", 0)
	current := []retro.Problem{
		{ID: "p1", Problem: "掛け合いが 記事の要点を 並べる説明に寄りがち", Action: "旧施策"},
	}
	entries := []cache.Entry{{EpisodeNumber: 16, Analysis: &model.Analysis{}}}

	got, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, current, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "p1" {
		t.Errorf("got = %+v, want matched to p1 despite full-width space/tab vs half-width space", got)
	}
}

func TestLLMRetro_Run_UnknownIDFallsBackToProblemTextMatch(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"id": "p99", "problem": "掛け合いが説明調", "action": "改善版の施策", "first_seen_episode": 10, "last_seen_episode": 16}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}}", 0)
	current := []retro.Problem{
		{ID: "p1", Problem: "掛け合いが説明調", Action: "旧施策"},
	}
	entries := []cache.Entry{{EpisodeNumber: 16, Analysis: &model.Analysis{}}}

	got, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, current, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "p1" {
		t.Errorf("got = %+v, want matched to p1 via text fallback despite unknown id p99", got)
	}
}

func TestLLMRetro_Run_KnownIDTakesPriorityOverProblemTextMatch(t *testing.T) {
	// current has two entries; the LLM's id (p2) and the problem text (matches p1's text) disagree.
	// A known id must win outright, without falling back to text matching.
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"id": "p2", "problem": "問題A", "action": "施策", "first_seen_episode": 10, "last_seen_episode": 16}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}}", 0)
	current := []retro.Problem{
		{ID: "p1", Problem: "問題A", Action: "施策1"},
		{ID: "p2", Problem: "問題B", Action: "施策2"},
	}
	entries := []cache.Entry{{EpisodeNumber: 16, Analysis: &model.Analysis{}}}

	got, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, current, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "p2" {
		t.Errorf("got = %+v, want id p2 (LLM id takes priority even though problem text matches p1)", got)
	}
}

func TestLLMRetro_Run_MatchesKeepEntryByProblemTextWhenIDMissing(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"problem": "定着済みの問題", "action": "再発時の新施策", "first_seen_episode": 10, "last_seen_episode": 16}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}}", 0)
	currentKeeps := []retro.Keep{{ID: "p3", Problem: "定着済みの問題", Action: "実証済み施策"}}
	entries := []cache.Entry{{EpisodeNumber: 16, Analysis: &model.Analysis{}}}

	got, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, nil, currentKeeps, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "p3" {
		t.Errorf("got = %+v, want matched to keep id p3", got)
	}
}

func TestLLMRetro_Run_KeepProblemEchoedWithoutRecurrence_DoesNotDuplicateIntoTry(t *testing.T) {
	// If the LLM mistakenly echoes a keep entry back in "problems" with no id, matching it to the
	// keep's id (rather than minting a fresh try id) must not create a second, try-side copy of an
	// already-kept problem when nothing recurred.
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"problem": "定着済みの問題", "action": "実証済み施策", "first_seen_episode": 3, "last_seen_episode": 6}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}}", 0)
	currentKeeps := []retro.Keep{{ID: "p3", Problem: "定着済みの問題", Action: "実証済み施策", ProvenAtEpisode: 5}}
	entries := []cache.Entry{{EpisodeNumber: 6, Analysis: &model.Analysis{}}}

	proposed, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, nil, currentKeeps, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(proposed) != 1 || proposed[0].ID != "p3" {
		t.Fatalf("proposed = %+v, want matched to keep id p3", proposed)
	}

	result := retro.ApplyCounts(retro.ApplyCountsInput{
		PrevKeeps:        currentKeeps,
		ProposedProblems: proposed,
		NewEpisodes:      []int{6},
		KeepThreshold:    5,
		LatestEpisode:    6,
	})

	if len(result.NextTry) != 0 {
		t.Errorf("NextTry = %+v, want empty (no duplicate of the kept problem)", result.NextTry)
	}
	if len(result.NextKeep) != 1 || result.NextKeep[0].ID != "p3" {
		t.Fatalf("NextKeep = %+v, want p3 unchanged", result.NextKeep)
	}
}

func TestLLMRetro_Run_ProblemTextMatchPrefersTryOverKeep(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"problem": "共通の問題文", "action": "施策", "first_seen_episode": 10, "last_seen_episode": 16}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}}", 0)
	current := []retro.Problem{{ID: "p1", Problem: "共通の問題文", Action: "施策1"}}
	currentKeeps := []retro.Keep{{ID: "p9", Problem: "共通の問題文", Action: "施策9"}}
	entries := []cache.Entry{{EpisodeNumber: 16, Analysis: &model.Analysis{}}}

	got, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, current, currentKeeps, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "p1" {
		t.Errorf("got = %+v, want try's id p1 preferred over keep's id p9 for a duplicated problem text", got)
	}
}

func TestLLMRetro_Run_ContinuingProblemWithMissingID_PreservesClearStreakThroughApplyCounts(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"problem": "掛け合いが説明調", "action": "旧施策", "first_seen_episode": 10, "last_seen_episode": 16}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}}", 0)
	current := []retro.Problem{
		{ID: "p1", Problem: "掛け合いが説明調", Action: "旧施策", FirstSeenEpisode: 10, LastSeenEpisode: 14, ClearStreak: 2},
	}
	entries := []cache.Entry{{EpisodeNumber: 16, Analysis: &model.Analysis{}}}

	proposed, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, current, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(proposed) != 1 || proposed[0].ID != "p1" {
		t.Fatalf("proposed = %+v, want matched to p1 by text", proposed)
	}

	result := retro.ApplyCounts(retro.ApplyCountsInput{
		PrevTryProblems:  current,
		ProposedProblems: proposed,
		NewEpisodes:      []int{15, 16},
		KeepThreshold:    5,
		LatestEpisode:    16,
	})

	if len(result.NextTry) != 1 {
		t.Fatalf("NextTry = %+v, want exactly 1 entry (missing id must not create a duplicate)", result.NextTry)
	}
	if result.NextTry[0].ID != "p1" {
		t.Errorf("ID = %q, want p1", result.NextTry[0].ID)
	}
	if result.NextTry[0].ClearStreak != 4 {
		t.Errorf("ClearStreak = %d, want 4 (2 + 2 non-recurring new episodes, carried over rather than reset)", result.NextTry[0].ClearStreak)
	}
}

func TestLLMRetro_Run_ContinuingProblemWithMissingID_RecurrenceRewritesAction(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"problem": "掛け合いが説明調", "action": "改善版の施策", "first_seen_episode": 10, "last_seen_episode": 16}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}}", 0)
	current := []retro.Problem{
		{ID: "p1", Problem: "掛け合いが説明調", Action: "旧施策", FirstSeenEpisode: 10, LastSeenEpisode: 14, ClearStreak: 2},
	}
	entries := []cache.Entry{{EpisodeNumber: 16, Analysis: &model.Analysis{}}}

	proposed, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, current, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(proposed) != 1 || proposed[0].ID != "p1" {
		t.Fatalf("proposed = %+v, want matched to p1 by text", proposed)
	}

	result := retro.ApplyCounts(retro.ApplyCountsInput{
		PrevTryProblems:  current,
		ProposedProblems: proposed,
		Recurrences:      []retro.Recurrence{{ID: "p1", Episodes: []int{16}}},
		NewEpisodes:      []int{16},
		KeepThreshold:    5,
		LatestEpisode:    16,
	})

	if len(result.NextTry) != 1 || result.NextTry[0].ID != "p1" {
		t.Fatalf("NextTry = %+v, want single entry p1", result.NextTry)
	}
	if result.NextTry[0].Action != "改善版の施策" {
		t.Errorf("Action = %q, want rewritten from the matched proposed problem", result.NextTry[0].Action)
	}
	if result.NextTry[0].ClearStreak != 0 {
		t.Errorf("ClearStreak = %d, want 0 (reset by recurrence)", result.NextTry[0].ClearStreak)
	}
}

func TestLLMRetro_Run_TruncatesToMaxTries(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{
		"problems": [
			{"problem": "1", "action": "a", "first_seen_episode": 1, "last_seen_episode": 1},
			{"problem": "2", "action": "a", "first_seen_episode": 1, "last_seen_episode": 1},
			{"problem": "3", "action": "a", "first_seen_episode": 1, "last_seen_episode": 1},
			{"problem": "4", "action": "a", "first_seen_episode": 1, "last_seen_episode": 1}
		]
	}`)}
	r := retro.NewLLMRetro(mc, "{{analyses}}", 0)
	entries := []cache.Entry{{EpisodeNumber: 1, Analysis: &model.Analysis{}}}

	got, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("got %d problems, want 3 (truncated to max_tries)", len(got))
	}
}

func TestLLMRetro_Run_PromptContainsScripts(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{"problems": []}`)}
	r := retro.NewLLMRetro(mc, "scripts={{scripts}}", 0)
	entries := []cache.Entry{{
		EpisodeNumber: 90,
		Analysis:      &model.Analysis{},
		Corners: []cache.CornerEntry{{
			ID:    "opening",
			Title: "オープニング",
			Lines: []cache.LineEntry{{SpeakerRole: "zundamon", Text: "今日は記事がないのだ"}},
		}},
	}}

	_, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prompt := mc.captured[0].Messages[0].Content
	if !strings.Contains(prompt, "今日は記事がないのだ") {
		t.Errorf("prompt should contain the actual line, got: %s", prompt)
	}
	if !strings.Contains(prompt, `"speaker":"zundamon"`) {
		t.Errorf("prompt should use the speaker key (matching analyze.md), got: %s", prompt)
	}
	if !strings.Contains(prompt, `"corner_id":"opening"`) {
		t.Errorf("prompt should carry the corner id, got: %s", prompt)
	}
}

func TestLLMRetro_Run_ScriptsExcludeEpisodesWithoutLines(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{"problems": []}`)}
	r := retro.NewLLMRetro(mc, "scripts={{scripts}}", 0)
	entries := []cache.Entry{
		// 台本を持たない回（機能追加前・保持期間外）は {{scripts}} に現れない
		{EpisodeNumber: 89, Analysis: &model.Analysis{}, Corners: []cache.CornerEntry{{ID: "opening", Title: "オープニング"}}},
		{EpisodeNumber: 90, Analysis: &model.Analysis{}, Corners: []cache.CornerEntry{{
			ID:    "opening",
			Title: "オープニング",
			Lines: []cache.LineEntry{{SpeakerRole: "metan", Text: "はじめるわよ"}},
		}}},
	}

	_, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prompt := mc.captured[0].Messages[0].Content
	if strings.Contains(prompt, `"episode_number":89`) {
		t.Errorf("episode 89 has no lines and should be omitted from scripts, got: %s", prompt)
	}
	if !strings.Contains(prompt, `"episode_number":90`) {
		t.Errorf("episode 90 has lines and should be present in scripts, got: %s", prompt)
	}
}

func TestLLMRetro_Run_NoScripts_RendersEmptyArray(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{"problems": []}`)}
	r := retro.NewLLMRetro(mc, "scripts={{scripts}}", 0)
	entries := []cache.Entry{{EpisodeNumber: 1, Analysis: &model.Analysis{}}}

	_, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prompt := mc.captured[0].Messages[0].Content; prompt != "scripts=[]" {
		t.Errorf("scripts should render as an empty array when no entry has lines, got: %s", prompt)
	}
}

func TestLLMRetro_Run_PromptContainsMetrics(t *testing.T) {
	mc := &mockClient{response: json.RawMessage(`{"problems": []}`)}
	r := retro.NewLLMRetro(mc, "analyses={{analyses}}", 0)
	entries := []cache.Entry{{
		EpisodeNumber: 90,
		Analysis: &model.Analysis{Metrics: model.AnalysisMetrics{
			Corners:              []model.AnalysisCornerMetrics{{ID: "tech_news", TargetChars: 1280, ActualChars: 1257, CharsPerSec: 6.19}},
			Speakers:             []model.AnalysisSpeakerMetrics{{CharacterID: "zundamon", LineCount: 33, CharCount: 816}},
			ProofreadCorrections: 6,
		}},
	}}

	_, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prompt := mc.captured[0].Messages[0].Content
	for _, want := range []string{`"target_chars":1280`, `"actual_chars":1257`, `"char_count":816`, `"proofread_corrections":6`} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt should contain metrics field %s, got: %s", want, prompt)
		}
	}
}
