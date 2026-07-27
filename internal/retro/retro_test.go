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

	got, _, lastEvaluated, err := r.Run(context.Background(), config.ProgramConfig{Title: "テスト"}, entries, nil, nil, 3)
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

	got, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, current, nil, 3)
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

	got, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, current, nil, 3)
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

	got, _, _, err := r.Run(context.Background(), config.ProgramConfig{}, entries, nil, nil, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("got %d problems, want 3 (truncated to max_tries)", len(got))
	}
}
