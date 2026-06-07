package eval

import (
	"os"
	"strconv"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

// Criterion represents a scoring dimension for LLM-as-judge evaluation.
type Criterion string

const (
	// Proofread evaluation criteria.
	CriterionDetectionRecall          Criterion = "detection_recall"
	CriterionFalsePositiveSuppression Criterion = "false_positive_suppression"
	CriterionCorrectionAccuracy       Criterion = "correction_accuracy"
	CriterionReasonValidity           Criterion = "reason_validity"

	// Shared evaluation criteria (used by multiple prompts).
	CriterionFaithfulness     Criterion = "faithfulness"
	CriterionCoverage         Criterion = "coverage"
	CriterionFormatCompliance Criterion = "format_compliance"

	// Summarize-only evaluation criteria.
	CriterionConciseness Criterion = "conciseness"

	// CornerSummary-only evaluation criteria.
	CriterionSpecificity Criterion = "specificity"
)

// AllCriteria lists all proofread scoring dimensions in canonical order.
var AllCriteria = []Criterion{
	CriterionDetectionRecall,
	CriterionFalsePositiveSuppression,
	CriterionCorrectionAccuracy,
	CriterionReasonValidity,
}

// AllSummarizeCriteria lists all summarize scoring dimensions in canonical order.
var AllSummarizeCriteria = []Criterion{
	CriterionFaithfulness,
	CriterionCoverage,
	CriterionConciseness,
	CriterionFormatCompliance,
}

// AllCornerSummaryCriteria lists all corner_summary scoring dimensions in canonical order.
var AllCornerSummaryCriteria = []Criterion{
	CriterionFaithfulness,
	CriterionCoverage,
	CriterionSpecificity,
	CriterionFormatCompliance,
}

// ScoreEntry holds the score and reason for one criterion.
type ScoreEntry struct {
	Criterion Criterion `json:"criterion"`
	Score     int       `json:"score"`
	Reason    string    `json:"reason"`
}

// CaseResult holds the judge scores for one evaluation case.
type CaseResult struct {
	CaseName string
	SetType  string // "regression" or "generalization"
	Scores   []ScoreEntry
}

// Aggregation holds aggregated scores.
type Aggregation struct {
	Overall     float64
	ByCriterion map[Criterion]float64
}

// AggregateScores computes the overall average and per-criterion averages.
func AggregateScores(results []CaseResult) Aggregation {
	if len(results) == 0 {
		return Aggregation{ByCriterion: make(map[Criterion]float64)}
	}

	sums := make(map[Criterion]float64)
	counts := make(map[Criterion]int)
	total := 0.0
	totalN := 0

	for _, r := range results {
		for _, s := range r.Scores {
			sums[s.Criterion] += float64(s.Score)
			counts[s.Criterion]++
			total += float64(s.Score)
			totalN++
		}
	}

	byC := make(map[Criterion]float64, len(sums))
	for c, sum := range sums {
		byC[c] = sum / float64(counts[c])
	}

	return Aggregation{
		Overall:     total / float64(totalN),
		ByCriterion: byC,
	}
}

// inconclusivePatterns are error substrings that indicate infrastructure failures
// rather than quality failures.
var inconclusivePatterns = []string{
	"http do:",
	"dial tcp",
	"connection refused",
	"status 429",
	"rate limit",
	"context deadline exceeded",
	"context canceled",
	"timeout",
	"EOF",
	"no such host",
	"status 5",
}

// ResolveExpectation returns s if non-empty, or the sentinel "（なし）" used by
// judge prompts when no expected result is provided.
func ResolveExpectation(s string) string {
	if s == "" {
		return "（なし）"
	}
	return s
}

// IsInconclusive returns true when err represents a transient infrastructure
// problem (network, rate limit, API outage) rather than a quality failure.
func IsInconclusive(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, p := range inconclusivePatterns {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}

// BuildLLMConfig constructs an llm.Config from environment variables.
// apiKeyEnv, modelEnv, intervalEnv are the names of the env vars to read.
func BuildLLMConfig(apiKeyEnv, modelEnv, intervalEnv string) llm.Config {
	model := os.Getenv(modelEnv)
	if model == "" {
		model = llm.DefaultModel
	}

	intervalMS := config.DefaultMinRequestIntervalMS
	if v := os.Getenv(intervalEnv); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			intervalMS = n
		}
	}

	return llm.Config{
		APIKey:               os.Getenv(apiKeyEnv),
		Model:                model,
		MinRequestIntervalMS: intervalMS,
	}
}
