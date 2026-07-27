package analyze_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/canpok1/vox-radio/internal/analyze"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
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

func testLines() model.ScriptLines {
	return model.ScriptLines{
		Corners: []model.CornerLines{
			{ID: "c1", Lines: []model.Line{
				{SpeakerRole: "zundamon", Text: "今日はAIについて話すのだ"},
				{SpeakerRole: "metan", Text: "そうですね"},
			}},
		},
	}
}

func TestComputeMetrics_AggregatesCornersAndSpeakers(t *testing.T) {
	corners := []config.CornerConfig{
		{ID: "c1", LengthSec: 60},
		{ID: "c2", LengthSec: 90},
	}
	lines := model.ScriptLines{
		Corners: []model.CornerLines{
			{
				ID: "c1",
				Lines: []model.Line{
					{SpeakerRole: "zundamon", Text: "こんにちは"},
					{SpeakerRole: "metan", Text: "どうも"},
				},
			},
			{
				ID: "c2",
				Lines: []model.Line{
					{SpeakerRole: "zundamon", Text: "また今度"},
				},
			},
		},
	}
	durations := map[string]float64{"c1": 58.5, "c2": 91.2}
	pr := &model.ProofreadResult{Corrections: []model.ProofreadCorrection{{}, {}}}

	got := analyze.ComputeMetrics(corners, lines, pr, durations)

	if len(got.Corners) != 2 {
		t.Fatalf("Corners len = %d, want 2", len(got.Corners))
	}
	if got.Corners[0].ID != "c1" || got.Corners[0].TargetLengthSec != 60 || got.Corners[0].ActualLengthSec != 58.5 || got.Corners[0].LineCount != 2 {
		t.Errorf("Corners[0] = %+v, unexpected", got.Corners[0])
	}
	if got.Corners[1].ID != "c2" || got.Corners[1].TargetLengthSec != 90 || got.Corners[1].ActualLengthSec != 91.2 || got.Corners[1].LineCount != 1 {
		t.Errorf("Corners[1] = %+v, unexpected", got.Corners[1])
	}

	bySpeaker := make(map[string]model.AnalysisSpeakerMetrics)
	for _, s := range got.Speakers {
		bySpeaker[s.CharacterID] = s
	}
	if bySpeaker["zundamon"].LineCount != 2 {
		t.Errorf("zundamon LineCount = %d, want 2", bySpeaker["zundamon"].LineCount)
	}
	if bySpeaker["zundamon"].CharCount != len([]rune("こんにちは"))+len([]rune("また今度")) {
		t.Errorf("zundamon CharCount = %d, want %d", bySpeaker["zundamon"].CharCount, len([]rune("こんにちは"))+len([]rune("また今度")))
	}
	if bySpeaker["metan"].LineCount != 1 {
		t.Errorf("metan LineCount = %d, want 1", bySpeaker["metan"].LineCount)
	}

	if got.ProofreadCorrections != 2 {
		t.Errorf("ProofreadCorrections = %d, want 2", got.ProofreadCorrections)
	}
}

func TestComputeMetrics_NilProofreadResultIsZero(t *testing.T) {
	got := analyze.ComputeMetrics(nil, model.ScriptLines{}, nil, nil)
	if got.ProofreadCorrections != 0 {
		t.Errorf("ProofreadCorrections = %d, want 0", got.ProofreadCorrections)
	}
}

func TestComputeMetrics_MissingDurationIsZeroNoPanic(t *testing.T) {
	corners := []config.CornerConfig{{ID: "c1", LengthSec: 60}}
	lines := model.ScriptLines{Corners: []model.CornerLines{{ID: "c1"}}}

	got := analyze.ComputeMetrics(corners, lines, nil, map[string]float64{})

	if got.Corners[0].ActualLengthSec != 0 {
		t.Errorf("ActualLengthSec = %v, want 0", got.Corners[0].ActualLengthSec)
	}
}

func TestComputeMetrics_EmptyInputsProduceNonNilSlices(t *testing.T) {
	got := analyze.ComputeMetrics(nil, model.ScriptLines{}, nil, nil)
	if got.Corners == nil {
		t.Error("Corners is nil, want empty non-nil slice")
	}
	if got.Speakers == nil {
		t.Error("Speakers is nil, want empty non-nil slice")
	}
}

func TestLLMAnalyzer_Analyze_SingleLLMCallReturnsFindingsAndPatterns(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{
			"findings": [{"aspect": "掛け合い", "severity": "high", "detail": "説明の羅列", "evidence": "セリフ引用"}],
			"patterns": [{"aspect": "つかみ", "detail": "天気の雑談から入った"}]
		}`),
	}
	a := analyze.NewLLMAnalyzer(mc, "program={{program}} corners={{corners}} lines={{lines}} metrics={{metrics}}", 0)

	program := config.ProgramConfig{Title: "テスト番組"}
	corners := []config.CornerConfig{{ID: "c1", Title: "C1", LengthSec: 60}}

	got, err := a.Analyze(context.Background(), program, corners, testLines(), nil, map[string]float64{"c1": 58})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mc.captured) != 1 {
		t.Fatalf("expected exactly 1 LLM call, got %d", len(mc.captured))
	}

	if len(got.Findings) != 1 || got.Findings[0].Severity != "high" {
		t.Errorf("Findings = %+v, want 1 high-severity finding", got.Findings)
	}
	if len(got.Patterns) != 1 || got.Patterns[0].Aspect != "つかみ" {
		t.Errorf("Patterns = %+v, want 1 つかみ pattern", got.Patterns)
	}
	if got.Metrics.Corners[0].ID != "c1" {
		t.Errorf("Metrics.Corners[0].ID = %q, want c1", got.Metrics.Corners[0].ID)
	}
}

func TestLLMAnalyzer_Analyze_NormalizesUnknownSeverityToMedium(t *testing.T) {
	mc := &mockClient{
		response: json.RawMessage(`{
			"findings": [{"aspect": "掛け合い", "severity": "critical", "detail": "d"}],
			"patterns": []
		}`),
	}
	a := analyze.NewLLMAnalyzer(mc, "{{lines}}", 0)

	got, err := a.Analyze(context.Background(), config.ProgramConfig{}, nil, testLines(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Findings) != 1 || got.Findings[0].Severity != "medium" {
		t.Errorf("Findings = %+v, want severity normalized to medium", got.Findings)
	}
}

func TestLLMAnalyzer_Analyze_TruncatesFindingsAndPatternsToFive(t *testing.T) {
	findings := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		findings = append(findings, `{"aspect":"a","severity":"low","detail":"d"}`)
	}
	patterns := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		patterns = append(patterns, `{"aspect":"a","detail":"d"}`)
	}
	mc := &mockClient{
		response: json.RawMessage(`{"findings":[` + join(findings) + `],"patterns":[` + join(patterns) + `]}`),
	}
	a := analyze.NewLLMAnalyzer(mc, "{{lines}}", 0)

	got, err := a.Analyze(context.Background(), config.ProgramConfig{}, nil, testLines(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Findings) != 5 {
		t.Errorf("Findings len = %d, want 5 (truncated)", len(got.Findings))
	}
	if len(got.Patterns) != 5 {
		t.Errorf("Patterns len = %d, want 5 (truncated)", len(got.Patterns))
	}
}

func join(items []string) string {
	out := ""
	for i, it := range items {
		if i > 0 {
			out += ","
		}
		out += it
	}
	return out
}
