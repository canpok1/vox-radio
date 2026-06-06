package write

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
)

// CastAssignment はコーナー内のキャスト割り当て情報。
// 番組全体での役割とコーナー固有の役割の両方を保持する。
type CastAssignment struct {
	CharacterID string // キャラID
	Type        string // config.CastTypeRegular | config.CastTypeGuest
	ProgramRole string // 番組全体での役割（casts[].role）
	CornerRole  string // このコーナーでの役割（corners[].cast）。未指定は ""
}

// castEntryForPrompt はプロンプトに渡すキャスト情報の1エントリ。
type castEntryForPrompt struct {
	ID          string `json:"id"`
	ProgramRole string `json:"program_role"`
	CornerRole  string `json:"corner_role,omitempty"`
}

// cornerForPrompt is the subset of corner data passed to the LLM.
// TargetChars is computed from LengthSec via config.DurationSecToTargetChars.
type cornerForPrompt struct {
	Title       string               `json:"title"`
	Content     string               `json:"content"`
	Cast        []castEntryForPrompt `json:"cast"`
	TargetChars int                  `json:"target_chars"`
}

// cornerOutline is the program-level outline of a corner (title only).
// Other corners' content is intentionally omitted so the LLM cannot "spoil"
// topics that belong to other corners. The current corner's full content is
// still provided separately via cornerForPrompt ({{corner}}).
type cornerOutline struct {
	Title string `json:"title"`
}

// previousCornerForPrompt is the context of a previously generated corner, passed to the LLM.
// Only title and lines (speaker_role + text) are included; voice presets are excluded.
type previousCornerForPrompt struct {
	Title string                  `json:"title"`
	Lines []previousLineForPrompt `json:"lines"`
}

// previousLineForPrompt holds only the fields needed for intra-episode context.
type previousLineForPrompt struct {
	SpeakerRole string `json:"speaker_role"`
	Text        string `json:"text"`
}

// programForPrompt is the program structure passed to the LLM to prevent fabrication.
type programForPrompt struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Corners     []cornerOutline `json:"corners"`
}

type Writer interface {
	Write(ctx context.Context, program config.ProgramConfig, corner config.CornerConfig, assignments []CastAssignment, allCorners []config.CornerConfig, previousCorners []model.CornerLines, articles []model.RundownArticle, flow string, chars map[string]config.CharacterConfig) ([]model.Line, error)
}

type LLMWriter struct {
	client         llm.Client
	promptTemplate string
	temperature    float64
	config         *config.Config
	pastEpisodes   []cache.Entry
	episodeNumber  int
	casts          []model.RundownCast
	recordedAt     string // RFC3339 in program timezone; empty means unknown
	timezone       string // IANA timezone name; empty means unknown
}

// NewLLMWriter creates an LLMWriter. Pass nil for cfg to use default presets.
func NewLLMWriter(client llm.Client, promptTemplate string, temperature float64, cfg *config.Config) *LLMWriter {
	return &LLMWriter{client: client, promptTemplate: promptTemplate, temperature: temperature, config: cfg}
}

// SetPastEpisodes configures recent past episodes to inject into the script generation prompt.
func (w *LLMWriter) SetPastEpisodes(eps []cache.Entry) {
	w.pastEpisodes = eps
}

// SetEpisodeNumber configures the episode number to inject into the script generation prompt.
// 0 means unknown (will be rendered as "（不明）").
func (w *LLMWriter) SetEpisodeNumber(n int) {
	w.episodeNumber = n
}

// SetCasts configures the confirmed cast members to inject into the script generation prompt.
func (w *LLMWriter) SetCasts(casts []model.RundownCast) {
	w.casts = casts
}

// SetRecordedAt configures the recording time to inject into the script generation prompt.
// t is formatted as RFC3339 in loc; loc.String() is used as the timezone name.
func (w *LLMWriter) SetRecordedAt(t time.Time, loc *time.Location) {
	w.recordedAt = t.In(loc).Format(time.RFC3339)
	w.timezone = loc.String()
}

func (w *LLMWriter) Write(ctx context.Context, program config.ProgramConfig, corner config.CornerConfig, assignments []CastAssignment, allCorners []config.CornerConfig, previousCorners []model.CornerLines, articles []model.RundownArticle, flow string, chars map[string]config.CharacterConfig) ([]model.Line, error) {
	castEntries := make([]castEntryForPrompt, len(assignments))
	for i, a := range assignments {
		castEntries[i] = castEntryForPrompt{
			ID:          a.CharacterID,
			ProgramRole: a.ProgramRole,
			CornerRole:  a.CornerRole,
		}
	}

	promptCorner := cornerForPrompt{
		Title:       corner.Title,
		Content:     corner.Content,
		Cast:        castEntries,
		TargetChars: config.DurationSecToTargetChars(corner.LengthSec),
	}
	cornerJSON, err := json.Marshal(promptCorner)
	if err != nil {
		return nil, fmt.Errorf("marshal corner: %w", err)
	}

	articlesJSON, err := json.Marshal(struct {
		Articles []model.RundownArticle `json:"articles"`
	}{Articles: articles})
	if err != nil {
		return nil, fmt.Errorf("marshal articles: %w", err)
	}

	outlines := make([]cornerOutline, len(allCorners))
	for i, c := range allCorners {
		outlines[i] = cornerOutline{Title: c.Title}
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
	castInfo := buildCastInfo(assignments, chars)
	presetInfo := buildPresetInfo(presets)

	castOverviewStr := formatCastInfo(w.casts)
	pastEpisodesStr := formatPastEpisodes(w.pastEpisodes)

	episodeNumberStr := "（不明）"
	if w.episodeNumber > 0 {
		episodeNumberStr = fmt.Sprintf("%d", w.episodeNumber)
	}

	previousCornersStr := "（なし）"
	if len(previousCorners) > 0 {
		prompts := make([]previousCornerForPrompt, len(previousCorners))
		for i, pc := range previousCorners {
			lines := make([]previousLineForPrompt, len(pc.Lines))
			for j, l := range pc.Lines {
				lines[j] = previousLineForPrompt{SpeakerRole: l.SpeakerRole, Text: l.Text}
			}
			prompts[i] = previousCornerForPrompt{Title: pc.Title, Lines: lines}
		}
		if b, err := json.Marshal(prompts); err == nil {
			previousCornersStr = string(b)
		}
	}

	recordedAtStr := w.recordedAt
	if recordedAtStr == "" {
		recordedAtStr = "（不明）"
	}
	timezoneStr := w.timezone
	if timezoneStr == "" {
		timezoneStr = "（不明）"
	}

	prompt := strings.NewReplacer(
		"{{program}}", string(programJSON),
		"{{corner}}", string(cornerJSON),
		"{{articles}}", string(articlesJSON),
		"{{flow}}", flow,
		"{{cast_info}}", castInfo,
		"{{preset_info}}", presetInfo,
		"{{past_episodes}}", pastEpisodesStr,
		"{{previous_corners}}", previousCornersStr,
		"{{episode_number}}", episodeNumberStr,
		"{{guest_info}}", castOverviewStr,
		"{{recorded_at}}", recordedAtStr,
		"{{timezone}}", timezoneStr,
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
// Includes both program role and corner role (if specified).
func buildCastInfo(assignments []CastAssignment, chars map[string]config.CharacterConfig) string {
	var sb strings.Builder
	for _, a := range assignments {
		ch, ok := chars[a.CharacterID]
		if !ok {
			fmt.Fprintf(&sb, "- %s（番組ロール: %s）\n", a.CharacterID, a.ProgramRole)
			continue
		}
		styleNames := make([]string, 0, len(ch.Styles))
		for s := range ch.Styles {
			styleNames = append(styleNames, s)
		}
		sort.Strings(styleNames)

		roleStr := fmt.Sprintf("番組ロール: %s", a.ProgramRole)
		if a.CornerRole != "" {
			roleStr += fmt.Sprintf(" / コーナーロール: %s", a.CornerRole)
		}
		fmt.Fprintf(&sb, "- %s（%s）: 名前=%s、一人称=%s、語尾=[%s]、性格=[%s]、スタイル=[%s]（デフォルト: %s）\n",
			a.CharacterID, roleStr,
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

// formatCastInfo formats the episode's cast overview for the prompt.
// regular type → 常連扱い（紹介・見送り不要）
// guest type → 最初のコーナーで紹介・中間で同席継続・最後で見送り（既存演出）
func formatCastInfo(casts []model.RundownCast) string {
	if len(casts) == 0 {
		return "（なし）この回はゲストのいない通常回です。ゲストの存在に一切触れないでください。"
	}

	var guests []model.RundownCast
	for _, c := range casts {
		if c.Type == config.CastTypeGuest {
			guests = append(guests, c)
		}
	}

	if len(guests) == 0 {
		return "（ゲストなし）この回はレギュラーメンバーのみの通常回です。ゲストの存在に一切触れないでください。"
	}

	var sb strings.Builder
	sb.WriteString("この回は以下のゲストが番組を通して（最初から最後まで）出演します:\n")
	for _, g := range guests {
		appearanceInfo := "（今回が初出演）"
		if past := g.PastAppearanceCount(); past > 0 {
			appearanceInfo = fmt.Sprintf("（過去%d回出演）", past)
		}
		fmt.Fprintf(&sb, "- %s（役割: %s）%s\n", g.CharacterID, g.Role, appearanceInfo)
	}
	sb.WriteString("\nゲスト演出ルール:\n")
	sb.WriteString("- 最初のコーナーでゲストを自然に紹介・登場させる。\n")
	sb.WriteString("- 中間のコーナーではゲストが継続して同席している前提で会話する（途中で急に登場・退場させない）。\n")
	sb.WriteString("- 最後のコーナーでゲストを見送る。\n")
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
