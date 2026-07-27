package model_test

import (
	"encoding/json"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

func TestAnalysis_JSONRoundTrip(t *testing.T) {
	a := model.Analysis{
		Metrics: model.AnalysisMetrics{
			Corners: []model.AnalysisCornerMetrics{
				{ID: "c1", TargetLengthSec: 60, ActualLengthSec: 58.5, LineCount: 10},
			},
			Speakers: []model.AnalysisSpeakerMetrics{
				{CharacterID: "zundamon", LineCount: 5, CharCount: 120},
			},
			ProofreadCorrections: 2,
		},
		Findings: []model.AnalysisFinding{
			{Aspect: "掛け合い", Severity: "high", Detail: "説明の羅列に寄っている", Evidence: "ずんだもん「〜なのだ」"},
		},
		Patterns: []model.AnalysisPattern{
			{Aspect: "つかみ", Detail: "天気の雑談から入った"},
		},
	}

	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got model.Analysis
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Metrics.Corners[0].ID != "c1" {
		t.Errorf("Corners[0].ID = %q, want c1", got.Metrics.Corners[0].ID)
	}
	if got.Metrics.Speakers[0].CharacterID != "zundamon" {
		t.Errorf("Speakers[0].CharacterID = %q, want zundamon", got.Metrics.Speakers[0].CharacterID)
	}
	if got.Metrics.ProofreadCorrections != 2 {
		t.Errorf("ProofreadCorrections = %d, want 2", got.Metrics.ProofreadCorrections)
	}
	if len(got.Findings) != 1 || got.Findings[0].Severity != "high" {
		t.Errorf("Findings = %+v, want 1 high-severity finding", got.Findings)
	}
	if len(got.Patterns) != 1 || got.Patterns[0].Aspect != "つかみ" {
		t.Errorf("Patterns = %+v, want 1 つかみ pattern", got.Patterns)
	}
}

func TestAnalysis_EmptySlicesMarshalAsEmptyArray(t *testing.T) {
	a := model.Analysis{
		Metrics: model.AnalysisMetrics{
			Corners:  make([]model.AnalysisCornerMetrics, 0),
			Speakers: make([]model.AnalysisSpeakerMetrics, 0),
		},
		Findings: make([]model.AnalysisFinding, 0),
		Patterns: make([]model.AnalysisPattern, 0),
	}

	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(m["findings"]) != "[]" {
		t.Errorf("findings = %s, want []", m["findings"])
	}
	if string(m["patterns"]) != "[]" {
		t.Errorf("patterns = %s, want []", m["patterns"])
	}

	var metrics map[string]json.RawMessage
	if err := json.Unmarshal(m["metrics"], &metrics); err != nil {
		t.Fatalf("unmarshal metrics: %v", err)
	}
	if string(metrics["corners"]) != "[]" {
		t.Errorf("metrics.corners = %s, want []", metrics["corners"])
	}
	if string(metrics["speakers"]) != "[]" {
		t.Errorf("metrics.speakers = %s, want []", metrics["speakers"])
	}
}
