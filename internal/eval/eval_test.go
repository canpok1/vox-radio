package eval

import (
	"errors"
	"slices"
	"testing"
)

func TestResolveExpectation(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "（なし）"},
		{"some text", "some text"},
	}
	for _, tt := range tests {
		if got := ResolveExpectation(tt.input); got != tt.want {
			t.Errorf("ResolveExpectation(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCriterionValues(t *testing.T) {
	criteria := []Criterion{
		CriterionDetectionRecall,
		CriterionFalsePositiveSuppression,
		CriterionCorrectionAccuracy,
		CriterionReasonValidity,
		CriterionFaithfulness,
		CriterionCoverage,
		CriterionConciseness,
		CriterionSpecificity,
		CriterionFormatCompliance,
		CriterionSummaryQuality,
		CriterionEpisodeTitleQuality,
		CriterionNotesFaithfulness,
		CriterionNotesCoverage,
	}
	for _, c := range criteria {
		if c == "" {
			t.Errorf("criterion should not be empty string")
		}
	}
}

func TestAllCriteria(t *testing.T) {
	want := []Criterion{
		CriterionDetectionRecall,
		CriterionFalsePositiveSuppression,
		CriterionCorrectionAccuracy,
		CriterionReasonValidity,
	}
	if !slices.Equal(AllCriteria, want) {
		t.Errorf("AllCriteria = %v, want %v", AllCriteria, want)
	}
}

func TestAllSummarizeCriteria(t *testing.T) {
	want := []Criterion{
		CriterionFaithfulness,
		CriterionCoverage,
		CriterionConciseness,
		CriterionFormatCompliance,
	}
	if !slices.Equal(AllSummarizeCriteria, want) {
		t.Errorf("AllSummarizeCriteria = %v, want %v", AllSummarizeCriteria, want)
	}
}

func TestAllCornerSummaryCriteria(t *testing.T) {
	want := []Criterion{
		CriterionFaithfulness,
		CriterionCoverage,
		CriterionSpecificity,
		CriterionFormatCompliance,
	}
	if !slices.Equal(AllCornerSummaryCriteria, want) {
		t.Errorf("AllCornerSummaryCriteria = %v, want %v", AllCornerSummaryCriteria, want)
	}
}

func TestAllSummaryCriteria(t *testing.T) {
	want := []Criterion{
		CriterionSummaryQuality,
		CriterionEpisodeTitleQuality,
		CriterionNotesFaithfulness,
		CriterionNotesCoverage,
	}
	if !slices.Equal(AllSummaryCriteria, want) {
		t.Errorf("AllSummaryCriteria = %v, want %v", AllSummaryCriteria, want)
	}
}

func TestAggregateScores_Average(t *testing.T) {
	results := []CaseResult{
		{
			CaseName: "case1",
			Scores: []ScoreEntry{
				{Criterion: CriterionDetectionRecall, Score: 5},
				{Criterion: CriterionFalsePositiveSuppression, Score: 4},
			},
		},
		{
			CaseName: "case2",
			Scores: []ScoreEntry{
				{Criterion: CriterionDetectionRecall, Score: 3},
				{Criterion: CriterionFalsePositiveSuppression, Score: 4},
			},
		},
	}

	agg := AggregateScores(results)
	// overall: (5+4+3+4)/4 = 4.0
	if agg.Overall != 4.0 {
		t.Errorf("overall = %v, want 4.0", agg.Overall)
	}
	// detection_recall: (5+3)/2 = 4.0
	if agg.ByCriterion[CriterionDetectionRecall] != 4.0 {
		t.Errorf("detection_recall = %v, want 4.0", agg.ByCriterion[CriterionDetectionRecall])
	}
	// false_positive_suppression: (4+4)/2 = 4.0
	if agg.ByCriterion[CriterionFalsePositiveSuppression] != 4.0 {
		t.Errorf("false_positive = %v, want 4.0", agg.ByCriterion[CriterionFalsePositiveSuppression])
	}
}

func TestAggregateScores_Empty(t *testing.T) {
	agg := AggregateScores(nil)
	if agg.Overall != 0 {
		t.Errorf("overall for empty = %v, want 0", agg.Overall)
	}
}

func TestIsInconclusive_NetworkError(t *testing.T) {
	err := errors.New("http do: dial tcp: connection refused")
	if !IsInconclusive(err) {
		t.Error("network error should be inconclusive")
	}
}

func TestIsInconclusive_RateLimitStatus(t *testing.T) {
	err := errors.New("api error (status 429): rate limit exceeded")
	if !IsInconclusive(err) {
		t.Error("rate limit error should be inconclusive")
	}
}

func TestIsInconclusive_QualityFailure(t *testing.T) {
	err := errors.New("quality below threshold: 3.5 < 4.0")
	if IsInconclusive(err) {
		t.Error("quality failure should not be inconclusive")
	}
}

func TestIsInconclusive_Nil(t *testing.T) {
	if IsInconclusive(nil) {
		t.Error("nil error should not be inconclusive")
	}
}

func TestBuildLLMConfig_Defaults(t *testing.T) {
	t.Setenv("TEST_API_KEY", "test-key")
	t.Setenv("TEST_MODEL", "")
	t.Setenv("TEST_INTERVAL_MS", "")

	cfg := BuildLLMConfig("TEST_API_KEY", "TEST_MODEL", "TEST_INTERVAL_MS")
	if cfg.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want test-key", cfg.APIKey)
	}
	if cfg.Model != "gemini-3.1-flash-lite" {
		t.Errorf("Model = %q, want gemini-3.1-flash-lite", cfg.Model)
	}
	if cfg.MinRequestIntervalMS != 4500 {
		t.Errorf("MinRequestIntervalMS = %d, want 4500", cfg.MinRequestIntervalMS)
	}
}

func TestBuildLLMConfig_EnvOverride(t *testing.T) {
	t.Setenv("TEST_API_KEY2", "key123")
	t.Setenv("TEST_MODEL2", "gemini-2.0-flash")
	t.Setenv("TEST_INTERVAL_MS2", "2000")

	cfg := BuildLLMConfig("TEST_API_KEY2", "TEST_MODEL2", "TEST_INTERVAL_MS2")
	if cfg.Model != "gemini-2.0-flash" {
		t.Errorf("Model = %q, want gemini-2.0-flash", cfg.Model)
	}
	if cfg.MinRequestIntervalMS != 2000 {
		t.Errorf("MinRequestIntervalMS = %d, want 2000", cfg.MinRequestIntervalMS)
	}
}
