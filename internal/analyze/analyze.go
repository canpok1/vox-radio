// Package analyze computes the post-generation analysis for a single episode (ADR-0098):
// mechanical metrics plus LLM-derived findings and dialogue patterns.
package analyze

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/logging"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

// maxFindings and maxPatterns cap the number of items returned per analysis, even if the LLM
// returns more (ADR-0098: keep the injected text small and force it to prioritize).
const (
	maxFindings = 5
	maxPatterns = 5
)

var validSeverities = map[string]bool{"high": true, "medium": true, "low": true}

// ComputeMetrics aggregates indicators that need no LLM judgement: per-corner target vs actual
// length and line count, per-speaker line/character counts, and the proofread correction count.
// cornerDurations missing an entry (e.g. a corner skipped by mix) yields ActualLengthSec 0.
func ComputeMetrics(corners []config.CornerConfig, lines model.ScriptLines, pr *model.ProofreadResult, cornerDurations map[string]float64) model.AnalysisMetrics {
	lineCountByCornerID := make(map[string]int, len(lines.Corners))
	speakerOrder := make([]string, 0)
	lineCountBySpeaker := make(map[string]int)
	charCountBySpeaker := make(map[string]int)
	for _, c := range lines.Corners {
		lineCountByCornerID[c.ID] += len(c.Lines)
		for _, line := range c.Lines {
			if _, ok := lineCountBySpeaker[line.SpeakerRole]; !ok {
				speakerOrder = append(speakerOrder, line.SpeakerRole)
			}
			lineCountBySpeaker[line.SpeakerRole]++
			charCountBySpeaker[line.SpeakerRole] += len([]rune(line.Text))
		}
	}

	cornerMetrics := make([]model.AnalysisCornerMetrics, len(corners))
	for i, c := range corners {
		cornerMetrics[i] = model.AnalysisCornerMetrics{
			ID:              c.ID,
			TargetLengthSec: c.LengthSec,
			ActualLengthSec: cornerDurations[c.ID],
			LineCount:       lineCountByCornerID[c.ID],
		}
	}

	speakerMetrics := make([]model.AnalysisSpeakerMetrics, len(speakerOrder))
	for i, id := range speakerOrder {
		speakerMetrics[i] = model.AnalysisSpeakerMetrics{
			CharacterID: id,
			LineCount:   lineCountBySpeaker[id],
			CharCount:   charCountBySpeaker[id],
		}
	}

	corrections := 0
	if pr != nil {
		corrections = len(pr.Corrections)
	}

	return model.AnalysisMetrics{
		Corners:              cornerMetrics,
		Speakers:             speakerMetrics,
		ProofreadCorrections: corrections,
	}
}

var analysisSchema = json.RawMessage(`{
  "type": "object",
  "required": ["findings", "patterns"],
  "properties": {
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["aspect", "severity", "detail"],
        "properties": {
          "aspect":   {"type": "string"},
          "severity": {"type": "string"},
          "detail":   {"type": "string"},
          "evidence": {"type": "string"}
        },
        "additionalProperties": false
      }
    },
    "patterns": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["aspect", "detail"],
        "properties": {
          "aspect": {"type": "string"},
          "detail": {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

// LLMAnalyzer computes metrics and asks an LLM for qualitative findings and dialogue patterns.
type LLMAnalyzer struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
	logger         *slog.Logger
}

// Option configures an LLMAnalyzer.
type Option func(*LLMAnalyzer)

// WithLogger returns an option that sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(a *LLMAnalyzer) { a.logger = l }
}

// NewLLMAnalyzer creates a new LLMAnalyzer.
func NewLLMAnalyzer(client llm.Client, promptTemplate string, temperature float64, opts ...Option) *LLMAnalyzer {
	a := &LLMAnalyzer{client: client, promptTemplate: promptTemplate, temperature: temperature}
	for _, opt := range opts {
		opt(a)
	}
	if a.logger == nil {
		a.logger = slog.Default()
	}
	a.logger = a.logger.With("step", "analyze")
	return a
}

type programInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type cornerInfo struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	TargetLengthSec int    `json:"target_length_sec"`
}

type speechEntry struct {
	CornerID string `json:"corner_id"`
	Speaker  string `json:"speaker"`
	Text     string `json:"text"`
}

type analysisResponse struct {
	Findings []model.AnalysisFinding `json:"findings"`
	Patterns []model.AnalysisPattern `json:"patterns"`
}

// Analyze computes metrics, asks the LLM for findings and patterns in a single call, and returns
// the combined result. Findings and patterns beyond maxFindings/maxPatterns are dropped, and any
// finding severity outside {high, medium, low} is normalized to "medium".
func (a *LLMAnalyzer) Analyze(ctx context.Context, program config.ProgramConfig, corners []config.CornerConfig, lines model.ScriptLines, pr *model.ProofreadResult, cornerDurations map[string]float64) (model.Analysis, error) {
	done := logging.StartStep(ctx, a.logger, "開始")
	defer func() { done("") }()

	metrics := ComputeMetrics(corners, lines, pr, cornerDurations)

	programJSON, err := json.Marshal(programInfo{Title: program.Title, Description: program.Description})
	if err != nil {
		return model.Analysis{}, fmt.Errorf("marshal program: %w", err)
	}

	cornerInfos := make([]cornerInfo, len(corners))
	for i, c := range corners {
		cornerInfos[i] = cornerInfo{ID: c.ID, Title: c.Title, TargetLengthSec: c.LengthSec}
	}
	cornersJSON, err := json.Marshal(cornerInfos)
	if err != nil {
		return model.Analysis{}, fmt.Errorf("marshal corners: %w", err)
	}

	entries := make([]speechEntry, 0)
	for _, c := range lines.Corners {
		for _, line := range c.Lines {
			if line.Text == "" {
				continue
			}
			entries = append(entries, speechEntry{CornerID: c.ID, Speaker: line.SpeakerRole, Text: line.Text})
		}
	}
	linesJSON, err := json.Marshal(entries)
	if err != nil {
		return model.Analysis{}, fmt.Errorf("marshal lines: %w", err)
	}

	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return model.Analysis{}, fmt.Errorf("marshal metrics: %w", err)
	}

	prompt := strings.NewReplacer(
		"{{program}}", string(programJSON),
		"{{corners}}", string(cornersJSON),
		"{{lines}}", string(linesJSON),
		"{{metrics}}", string(metricsJSON),
	).Replace(a.promptTemplate)

	raw, err := a.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  analysisSchema,
		Temperature: a.temperature,
	})
	if err != nil {
		return model.Analysis{}, fmt.Errorf("llm complete: %w", err)
	}

	var resp analysisResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return model.Analysis{}, fmt.Errorf("unmarshal response: %w", err)
	}

	findings := model.NonNil(resp.Findings)
	for i := range findings {
		if !validSeverities[findings[i].Severity] {
			findings[i].Severity = "medium"
		}
	}
	if len(findings) > maxFindings {
		findings = findings[:maxFindings]
	}

	patterns := model.NonNil(resp.Patterns)
	if len(patterns) > maxPatterns {
		patterns = patterns[:maxPatterns]
	}

	return model.Analysis{
		Metrics:  metrics,
		Findings: findings,
		Patterns: patterns,
	}, nil
}
