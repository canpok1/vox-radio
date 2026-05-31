package write

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

// cornerForPrompt is the subset of corner data passed to the LLM.
// TargetChars is computed from TargetDurationSec via config.DurationSecToTargetChars.
type cornerForPrompt struct {
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Cast        map[string]string `json:"cast"`
	TargetChars int               `json:"target_chars"`
}

// cornerOutline is the program-level outline of a corner (title+content only, no cast).
type cornerOutline struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// programForPrompt is the program structure passed to the LLM to prevent fabrication.
type programForPrompt struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Corners     []cornerOutline `json:"corners"`
}

type Writer interface {
	Write(ctx context.Context, program config.ProgramConfig, corner config.CornerConfig, allCorners []config.CornerConfig, summaries []model.Summary, chars map[string]config.CharacterConfig) ([]model.Line, error)
}

type LLMWriter struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
	config         *config.Config
}

// NewLLMWriter creates an LLMWriter. Pass nil for cfg to use default presets.
func NewLLMWriter(client llm.Client, promptTemplate string, temperature float64, cfg *config.Config) *LLMWriter {
	return &LLMWriter{client: client, promptTemplate: promptTemplate, temperature: temperature, config: cfg}
}

func (w *LLMWriter) Write(ctx context.Context, program config.ProgramConfig, corner config.CornerConfig, allCorners []config.CornerConfig, summaries []model.Summary, chars map[string]config.CharacterConfig) ([]model.Line, error) {
	promptCorner := cornerForPrompt{
		Title:       corner.Title,
		Content:     corner.Content,
		Cast:        corner.Cast,
		TargetChars: config.DurationSecToTargetChars(corner.TargetDurationSec),
	}
	cornerJSON, err := json.Marshal(promptCorner)
	if err != nil {
		return nil, fmt.Errorf("marshal corner: %w", err)
	}

	summariesJSON, err := json.Marshal(struct {
		Summaries []model.Summary `json:"summaries"`
	}{Summaries: summaries})
	if err != nil {
		return nil, fmt.Errorf("marshal summaries: %w", err)
	}

	outlines := make([]cornerOutline, len(allCorners))
	for i, c := range allCorners {
		outlines[i] = cornerOutline{Title: c.Title, Content: c.Content}
	}
	promptProgram := programForPrompt{
		Title:       program.Title,
		Description: program.Description,
		Corners:     outlines,
	}
	programJSON, err := json.Marshal(promptProgram)
	if err != nil {
		return nil, fmt.Errorf("marshal program: %w", err)
	}

	presets := w.effectivePresets()
	castInfo := buildCastInfo(corner.Cast, chars)
	presetInfo := buildPresetInfo(presets)

	prompt := strings.NewReplacer(
		"{{program}}", string(programJSON),
		"{{corner}}", string(cornerJSON),
		"{{summary}}", string(summariesJSON),
		"{{cast_info}}", castInfo,
		"{{preset_info}}", presetInfo,
	).Replace(w.promptTemplate)

	schema := buildLinesSchema(presets)

	raw, err := w.client.Complete(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema:  schema,
		Temperature: w.temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("llm complete: %w", err)
	}

	var resp model.Lines
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal lines: %w", err)
	}

	return resp.Lines, nil
}

func (w *LLMWriter) effectivePresets() config.VoicevoxPresets {
	if w.config == nil {
		return config.VoicevoxConfig{}.EffectivePresets()
	}
	return w.config.Voicevox.EffectivePresets()
}

// buildLinesSchema generates a JSON Schema for the lines response, with intonation/pitch/speed
// enum values derived from the given presets.
func buildLinesSchema(presets config.VoicevoxPresets) json.RawMessage {
	intonationEnum := sortedKeys(presets.Intonation)
	pitchEnum := sortedKeys(presets.Pitch)
	speedEnum := sortedKeys(presets.Speed)

	intonationEnumJSON, _ := json.Marshal(intonationEnum)
	pitchEnumJSON, _ := json.Marshal(pitchEnum)
	speedEnumJSON, _ := json.Marshal(speedEnum)

	return json.RawMessage(fmt.Sprintf(`{
  "type": "object",
  "required": ["lines"],
  "properties": {
    "lines": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["speaker_role", "text"],
        "properties": {
          "speaker_role": {"type": "string"},
          "style":        {"type": "string"},
          "intonation":   {"type": "string", "enum": %s},
          "pitch":        {"type": "string", "enum": %s},
          "speed":        {"type": "string", "enum": %s},
          "text":         {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`, intonationEnumJSON, pitchEnumJSON, speedEnumJSON))
}

// buildPresetInfo formats available preset names for each axis for the prompt.
func buildPresetInfo(presets config.VoicevoxPresets) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "抑揚（intonation）: [%s]\n", strings.Join(sortedKeys(presets.Intonation), ", "))
	fmt.Fprintf(&sb, "音高（pitch）: [%s]\n", strings.Join(sortedKeys(presets.Pitch), ", "))
	fmt.Fprintf(&sb, "話速（speed）: [%s]\n", strings.Join(sortedKeys(presets.Speed), ", "))
	return sb.String()
}

// buildCastInfo formats cast assignments with character catalog features for the prompt.
func buildCastInfo(cast map[string]string, chars map[string]config.CharacterConfig) string {
	var sb strings.Builder
	for charID, role := range cast {
		ch, ok := chars[charID]
		if !ok {
			fmt.Fprintf(&sb, "- %s（%s）\n", charID, role)
			continue
		}
		styleNames := make([]string, 0, len(ch.Styles))
		for s := range ch.Styles {
			styleNames = append(styleNames, s)
		}
		sort.Strings(styleNames)
		fmt.Fprintf(&sb, "- %s（%s）: 名前=%s、一人称=%s、語尾=[%s]、性格=[%s]、スタイル=[%s]（デフォルト: %s）\n",
			charID, role,
			ch.Name,
			ch.Pronoun,
			strings.Join(ch.SpeechSuffix, ", "),
			strings.Join(ch.Personality, ", "),
			strings.Join(styleNames, ", "),
			ch.DefaultStyle,
		)
	}
	return sb.String()
}

func sortedKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
