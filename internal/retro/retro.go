// Package retro implements the KPT Problem/Try stages of the automatic improvement loop
// (ADR-0098): it turns accumulated per-episode analyses into a small set of in-progress
// problem/action pairs that get injected into the next episode's script generation.
package retro

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"gopkg.in/yaml.v3"
)

const tryFileHeader = `# vox-radio retro が生成する。手で編集しても次回の retro で全置換される。
# 恒久的に固定したい方針は episode-spec.yaml の script_note に書くこと。
# 適用を止めたいときはこのファイルを削除する（retro を実行しないだけでは注入は止まらない）。
`

// TryFile holds the problems retro is currently working on, each paired with the action being
// tried. LastEvaluatedEpisode is the newest episode already reflected in the counters; the
// keep-promotion loop counts only episodes beyond it so re-running retro without new episodes
// cannot inflate ClearStreak.
type TryFile struct {
	GeneratedAt          string    `yaml:"generated_at"`
	LastEvaluatedEpisode int       `yaml:"last_evaluated_episode"`
	Problems             []Problem `yaml:"problems"`
}

// Problem is one in-progress issue and the action being tried against it.
type Problem struct {
	ID               string `yaml:"id"`
	Problem          string `yaml:"problem"`
	Action           string `yaml:"action"`
	FirstSeenEpisode int    `yaml:"first_seen_episode"`
	LastSeenEpisode  int    `yaml:"last_seen_episode"`
	ClearStreak      int    `yaml:"clear_streak"` // consecutive newly-evaluated episodes without recurrence; promoted to keep at retro.keep_threshold
}

// LoadTryFile reads the try file at path. A missing file is not an error: it returns an empty
// TryFile (first run).
func LoadTryFile(path string) (TryFile, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return TryFile{Problems: make([]Problem, 0)}, nil
	}
	if err != nil {
		return TryFile{}, fmt.Errorf("read try file: %w", err)
	}
	var tf TryFile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return TryFile{}, fmt.Errorf("unmarshal try file: %w", err)
	}
	tf.Problems = model.NonNil(tf.Problems)
	return tf, nil
}

// MarshalTryFile renders tf as YAML with the standard header comment, the same bytes SaveTryFile
// writes to disk. Used by the retro command's --dry-run to preview without writing.
func MarshalTryFile(tf TryFile) ([]byte, error) {
	data, err := yaml.Marshal(tf)
	if err != nil {
		return nil, fmt.Errorf("marshal try file: %w", err)
	}
	return append([]byte(tryFileHeader), data...), nil
}

// SaveTryFile writes tf to path, fully replacing any existing content (retro does not do
// individual-item approval; ADR-0098).
func SaveTryFile(path string, tf TryFile) error {
	content, err := MarshalTryFile(tf)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write try file: %w", err)
	}
	return nil
}

const keepFileHeader = `# vox-radio retro が昇格・降格で更新する。既存項目の文面は書き換えない。
# 増えすぎたら episode-spec.yaml の script_note へ移し、ここから削除する。
# 適用を止めたいときはこのファイルを削除する。
`

// KeepFile holds actions whose effect has been confirmed: the problem stayed away for
// retro.keep_threshold consecutive newly-evaluated episodes. retro only appends promotions and
// removes demotions; it never rewrites the wording of an existing entry.
type KeepFile struct {
	GeneratedAt string `yaml:"generated_at"`
	Keeps       []Keep `yaml:"keeps"`
}

// Keep is one action whose effect has been confirmed and is now always injected.
type Keep struct {
	ID              string `yaml:"id"`
	Problem         string `yaml:"problem"`
	Action          string `yaml:"action"`
	ProvenAtEpisode int    `yaml:"proven_at_episode"`
}

// LoadKeepFile reads the keep file at path. A missing file is not an error: it returns an empty
// KeepFile.
func LoadKeepFile(path string) (KeepFile, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return KeepFile{Keeps: make([]Keep, 0)}, nil
	}
	if err != nil {
		return KeepFile{}, fmt.Errorf("read keep file: %w", err)
	}
	var kf KeepFile
	if err := yaml.Unmarshal(data, &kf); err != nil {
		return KeepFile{}, fmt.Errorf("unmarshal keep file: %w", err)
	}
	kf.Keeps = model.NonNil(kf.Keeps)
	return kf, nil
}

// MarshalKeepFile renders kf as YAML with the standard header comment, the same bytes
// SaveKeepFile writes to disk. Used by the retro command's --dry-run to preview without writing.
func MarshalKeepFile(kf KeepFile) ([]byte, error) {
	data, err := yaml.Marshal(kf)
	if err != nil {
		return nil, fmt.Errorf("marshal keep file: %w", err)
	}
	return append([]byte(keepFileHeader), data...), nil
}

// SaveKeepFile writes kf to path, fully replacing any existing content. Callers must not rewrite
// the Problem/Action of a surviving entry (ADR-0098): only promotions (append) and demotions
// (remove) should change kf between rounds.
func SaveKeepFile(path string, kf KeepFile) error {
	content, err := MarshalKeepFile(kf)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write keep file: %w", err)
	}
	return nil
}

// KeepContentLength returns the total character count of keep's problem/action text, used to
// check against retro.keep_length (a warning-only threshold; ADR-0098 never truncates keep).
func KeepContentLength(kf KeepFile) int {
	n := 0
	for _, k := range kf.Keeps {
		n += len([]rune(k.Problem)) + len([]rune(k.Action))
	}
	return n
}

// FilterAnalyzed returns the subset of entries with a non-nil Analysis, preserving order.
// Entries compacted before analyze was introduced (or predating it) are skipped.
func FilterAnalyzed(entries []cache.Entry) []cache.Entry {
	out := make([]cache.Entry, 0, len(entries))
	for _, e := range entries {
		if e.Analysis != nil {
			out = append(out, e)
		}
	}
	return out
}

// LLMRetro asks an LLM to propose the next set of problems/actions from recent analyzed episodes
// and the current try file's problems.
type LLMRetro struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
	logger         *slog.Logger
}

// Option configures an LLMRetro.
type Option func(*LLMRetro)

// WithLogger returns an option that sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(r *LLMRetro) { r.logger = l }
}

// NewLLMRetro creates a new LLMRetro.
func NewLLMRetro(client llm.Client, promptTemplate string, temperature float64, opts ...Option) *LLMRetro {
	r := &LLMRetro{client: client, promptTemplate: promptTemplate, temperature: temperature}
	for _, opt := range opts {
		opt(r)
	}
	if r.logger == nil {
		r.logger = slog.Default()
	}
	r.logger = r.logger.With("step", "retro")
	return r
}

var retroSchema = json.RawMessage(`{
  "type": "object",
  "required": ["problems", "recurrences"],
  "properties": {
    "problems": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["problem", "action", "first_seen_episode", "last_seen_episode"],
        "properties": {
          "id":                  {"type": "string"},
          "problem":             {"type": "string"},
          "action":              {"type": "string"},
          "first_seen_episode":  {"type": "integer"},
          "last_seen_episode":   {"type": "integer"}
        },
        "additionalProperties": false
      }
    },
    "recurrences": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "episodes"],
        "properties": {
          "id":       {"type": "string"},
          "episodes": {"type": "array", "items": {"type": "integer"}}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

type programForPrompt struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Direction   string `json:"direction"`
	ScriptNote  string `json:"script_note"`
}

type analysisForPrompt struct {
	EpisodeNumber int                     `json:"episode_number"`
	Findings      []model.AnalysisFinding `json:"findings"`
	Patterns      []model.AnalysisPattern `json:"patterns"`
}

type currentProblemForPrompt struct {
	ID      string `json:"id"`
	Problem string `json:"problem"`
	Action  string `json:"action"`
}

type currentKeepForPrompt struct {
	ID      string `json:"id"`
	Problem string `json:"problem"`
	Action  string `json:"action"`
}

// llmProblem is one entry of the LLM's raw response, before Go-side ID assignment.
type llmProblem struct {
	ID               string `json:"id"`
	Problem          string `json:"problem"`
	Action           string `json:"action"`
	FirstSeenEpisode int    `json:"first_seen_episode"`
	LastSeenEpisode  int    `json:"last_seen_episode"`
}

// llmRecurrence is one id's recurrence judgment from the LLM: the episode numbers (within the
// evaluated window) in which that problem/keep was judged to have recurred. Go computes
// ClearStreak from this; the LLM never counts.
type llmRecurrence struct {
	ID       string `json:"id"`
	Episodes []int  `json:"episodes"`
}

// Recurrence is the exported form of llmRecurrence, returned by Run.
type Recurrence struct {
	ID       string
	Episodes []int
}

type retroResponse struct {
	Problems    []llmProblem    `json:"problems"`
	Recurrences []llmRecurrence `json:"recurrences"`
}

// Run asks the LLM for the next problems/actions and for each tracked id's (try problems and keep
// entries) recurrence within the evaluated episodes. Returns the proposed problems with
// Go-assigned IDs, the recurrence judgments, and the newest episode number among entries (0 if
// entries is empty). entries must already be filtered to Analysis != nil (see FilterAnalyzed) and
// are passed to the LLM in the given order (oldest to newest, matching cache.Recent's natural
// order) so it can compare patterns across episodes.
func (r *LLMRetro) Run(ctx context.Context, program config.ProgramConfig, entries []cache.Entry, current []Problem, currentKeeps []Keep, maxTries int) ([]Problem, []Recurrence, int, error) {
	programJSON, err := json.Marshal(programForPrompt{
		Title:       program.Title,
		Description: program.Description,
		Direction:   program.Direction,
		ScriptNote:  program.ScriptNote,
	})
	if err != nil {
		return nil, nil, 0, fmt.Errorf("marshal program: %w", err)
	}

	lastEvaluated := 0
	analyses := make([]analysisForPrompt, 0, len(entries))
	for _, e := range entries {
		if e.EpisodeNumber > lastEvaluated {
			lastEvaluated = e.EpisodeNumber
		}
		analyses = append(analyses, analysisForPrompt{
			EpisodeNumber: e.EpisodeNumber,
			Findings:      e.Analysis.Findings,
			Patterns:      e.Analysis.Patterns,
		})
	}
	analysesJSON, err := json.Marshal(analyses)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("marshal analyses: %w", err)
	}

	currentForPrompt := make([]currentProblemForPrompt, len(current))
	for i, p := range current {
		currentForPrompt[i] = currentProblemForPrompt{ID: p.ID, Problem: p.Problem, Action: p.Action}
	}
	currentJSON, err := json.Marshal(currentForPrompt)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("marshal current problems: %w", err)
	}

	currentKeepsForPrompt := make([]currentKeepForPrompt, len(currentKeeps))
	for i, k := range currentKeeps {
		currentKeepsForPrompt[i] = currentKeepForPrompt{ID: k.ID, Problem: k.Problem, Action: k.Action}
	}
	currentKeepsJSON, err := json.Marshal(currentKeepsForPrompt)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("marshal current keeps: %w", err)
	}

	prompt := strings.NewReplacer(
		"{{program}}", string(programJSON),
		"{{analyses}}", string(analysesJSON),
		"{{current_problems}}", string(currentJSON),
		"{{current_keeps}}", string(currentKeepsJSON),
		"{{max_tries}}", strconv.Itoa(maxTries),
	).Replace(r.promptTemplate)

	raw, err := r.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  retroSchema,
		Temperature: r.temperature,
	})
	if err != nil {
		return nil, nil, 0, fmt.Errorf("llm complete: %w", err)
	}

	var resp retroResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, nil, 0, fmt.Errorf("unmarshal response: %w", err)
	}

	recurrences := make([]Recurrence, len(resp.Recurrences))
	for i, rec := range resp.Recurrences {
		recurrences[i] = Recurrence(rec)
	}

	return assignIDs(current, currentKeeps, resp.Problems, maxTries), recurrences, lastEvaluated, nil
}

// assignIDs assigns sequential IDs (p1, p2, ...) to raw problems that are not a genuine
// continuation of a current problem or keep entry, preserving the id only when it matches one of
// their ids. IDs are assigned by Go, not trusted from the LLM: even if the LLM echoes back a
// non-empty id for what is actually a new problem (hallucinated or copied from elsewhere), it is
// not in currentIDs and gets overridden here, avoiding duplicate or invented ids. The result is
// truncated to maxTries.
func assignIDs(current []Problem, currentKeeps []Keep, raw []llmProblem, maxTries int) []Problem {
	currentIDs := make(map[string]bool, len(current)+len(currentKeeps))
	used := make(map[int]bool)
	for _, p := range current {
		currentIDs[p.ID] = true
		if n, ok := parseProblemID(p.ID); ok {
			used[n] = true
		}
	}
	for _, k := range currentKeeps {
		currentIDs[k.ID] = true
		if n, ok := parseProblemID(k.ID); ok {
			used[n] = true
		}
	}

	next := 1
	nextID := func() string {
		for used[next] {
			next++
		}
		id := fmt.Sprintf("p%d", next)
		used[next] = true
		next++
		return id
	}

	result := make([]Problem, 0, len(raw))
	for _, p := range raw {
		id := p.ID
		if id == "" || !currentIDs[id] {
			id = nextID()
		}
		result = append(result, Problem{
			ID:               id,
			Problem:          p.Problem,
			Action:           p.Action,
			FirstSeenEpisode: p.FirstSeenEpisode,
			LastSeenEpisode:  p.LastSeenEpisode,
			ClearStreak:      0,
		})
	}
	if len(result) > maxTries {
		result = result[:maxTries]
	}
	return result
}

func parseProblemID(id string) (int, bool) {
	if !strings.HasPrefix(id, "p") {
		return 0, false
	}
	n, err := strconv.Atoi(id[1:])
	if err != nil {
		return 0, false
	}
	return n, true
}
