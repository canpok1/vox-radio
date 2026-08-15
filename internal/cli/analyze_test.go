package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

type analyzeMockClient struct {
	response json.RawMessage
	err      error
}

func (m *analyzeMockClient) Complete(_ context.Context, _ llm.CompletionRequest) (json.RawMessage, error) {
	return m.response, m.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRunAndSaveAnalysis_WritesAnalysisFile(t *testing.T) {
	dir := chdirTemp(t)
	layout := fileio.EpisodeLayout{OutDir: dir, ProgramID: "prog", EpisodeNumber: 1}

	lines := model.ScriptLines{Corners: []model.CornerLines{
		{ID: "c1", Lines: []model.Line{{SpeakerRole: "zundamon", Text: "こんにちは"}}},
	}}
	if err := writeJSON(layout.Lines(), lines); err != nil {
		t.Fatalf("write lines: %v", err)
	}
	if err := writeJSON(layout.Timeline(), model.Timeline{Corners: []model.CornerTiming{{ID: "c1", DurationSec: 12}}}); err != nil {
		t.Fatalf("write timeline: %v", err)
	}
	// layout.Proofread() intentionally not written (optional).

	mc := &analyzeMockClient{response: json.RawMessage(`{"findings":[],"patterns":[]}`)}
	cfg := &config.Config{}
	p := &config.EpisodeSpec{Program: config.ProgramConfig{Title: "テスト"}, Corners: []config.CornerConfig{{ID: "c1", LengthSec: 30}}}

	err := runAndSaveAnalysis(context.Background(), cfg, map[string]string{"analyze": "{{lines}}"}, mc, p, layout, testLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := readJSON[model.Analysis](layout.Analysis())
	if err != nil {
		t.Fatalf("read written analysis: %v", err)
	}
	if len(got.Metrics.Corners) != 1 || got.Metrics.Corners[0].ID != "c1" {
		t.Errorf("Metrics.Corners = %+v, want 1 entry for c1", got.Metrics.Corners)
	}
}

// A corner skipped by the rundown never reaches the lines file, so it must not appear in the
// metrics as a 0-second corner.
func TestRunAndSaveAnalysis_ExcludesSkippedCorners(t *testing.T) {
	dir := chdirTemp(t)
	layout := fileio.EpisodeLayout{OutDir: dir, ProgramID: "prog", EpisodeNumber: 1}

	lines := model.ScriptLines{Corners: []model.CornerLines{
		{ID: "c1", Lines: []model.Line{{SpeakerRole: "zundamon", Text: "こんにちは"}}},
	}}
	if err := writeJSON(layout.Lines(), lines); err != nil {
		t.Fatalf("write lines: %v", err)
	}
	if err := writeJSON(layout.Timeline(), model.Timeline{Corners: []model.CornerTiming{{ID: "c1", DurationSec: 12}}}); err != nil {
		t.Fatalf("write timeline: %v", err)
	}

	mc := &analyzeMockClient{response: json.RawMessage(`{"findings":[],"patterns":[]}`)}
	cfg := &config.Config{}
	p := &config.EpisodeSpec{Program: config.ProgramConfig{Title: "テスト"}, Corners: []config.CornerConfig{
		{ID: "c1", LengthSec: 30},
		{ID: "c2", LengthSec: 120},
	}}

	err := runAndSaveAnalysis(context.Background(), cfg, map[string]string{"analyze": "{{lines}}"}, mc, p, layout, testLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := readJSON[model.Analysis](layout.Analysis())
	if err != nil {
		t.Fatalf("read written analysis: %v", err)
	}
	if len(got.Metrics.Corners) != 1 || got.Metrics.Corners[0].ID != "c1" {
		t.Errorf("Metrics.Corners = %+v, want 1 entry for c1 (skipped c2 must not appear)", got.Metrics.Corners)
	}
}

func TestRunAndSaveAnalysis_LLMErrorPropagates(t *testing.T) {
	dir := chdirTemp(t)
	layout := fileio.EpisodeLayout{OutDir: dir, ProgramID: "prog", EpisodeNumber: 1}

	if err := writeJSON(layout.Lines(), model.ScriptLines{}); err != nil {
		t.Fatalf("write lines: %v", err)
	}
	if err := writeJSON(layout.Timeline(), model.Timeline{}); err != nil {
		t.Fatalf("write timeline: %v", err)
	}

	mc := &analyzeMockClient{err: errors.New("llm unavailable")}
	cfg := &config.Config{}
	p := &config.EpisodeSpec{}

	err := runAndSaveAnalysis(context.Background(), cfg, map[string]string{"analyze": "{{lines}}"}, mc, p, layout, testLogger())
	if err == nil {
		t.Fatal("expected error when LLM fails, got nil")
	}
}

func TestAppendToCache_MissingAnalysisFile_Succeeds(t *testing.T) {
	dir := chdirTemp(t)
	layout := fileio.EpisodeLayout{OutDir: dir, ProgramID: "prog", EpisodeNumber: 1}

	if err := writeJSON(layout.Manifest(), model.Manifest{Title: "t", Corners: []model.ManifestCorner{}}); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := writeJSON(layout.Rundown(), model.Rundown{}); err != nil {
		t.Fatalf("write rundown: %v", err)
	}
	// layout.Analysis() intentionally not written.

	mgr := cache.New(programCachePath("prog"))
	if err := appendToCache(mgr, layout, config.CacheConfig{}, 5, testLogger()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := mgr.Load()
	if err != nil {
		t.Fatalf("load cache: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Analysis != nil {
		t.Errorf("Analysis: got %+v, want nil (no analysis file)", entries[0].Analysis)
	}
}

func TestAppendToCache_ReadsAnalysisFile(t *testing.T) {
	dir := chdirTemp(t)
	layout := fileio.EpisodeLayout{OutDir: dir, ProgramID: "prog", EpisodeNumber: 1}

	if err := writeJSON(layout.Manifest(), model.Manifest{Title: "t", Corners: []model.ManifestCorner{}}); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := writeJSON(layout.Rundown(), model.Rundown{}); err != nil {
		t.Fatalf("write rundown: %v", err)
	}
	analysis := model.Analysis{Findings: []model.AnalysisFinding{{Aspect: "a", Severity: "low", Detail: "d"}}}
	if err := writeJSON(layout.Analysis(), analysis); err != nil {
		t.Fatalf("write analysis: %v", err)
	}

	mgr := cache.New(programCachePath("prog"))
	if err := appendToCache(mgr, layout, config.CacheConfig{}, 5, testLogger()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := mgr.Load()
	if err != nil {
		t.Fatalf("load cache: %v", err)
	}
	if len(entries) != 1 || entries[0].Analysis == nil || len(entries[0].Analysis.Findings) != 1 {
		t.Fatalf("Analysis not propagated: %+v", entries)
	}
}
