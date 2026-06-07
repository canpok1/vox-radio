//go:build eval

package eval_test

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/eval"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"github.com/canpok1/vox-radio/internal/script/write"
)

// writeCornerOutline mirrors the program-level corner outline passed to write.md.
type writeCornerOutline struct {
	Title string `json:"title"`
}

// writeProgram mirrors the programForPrompt passed to write.md.
type writeProgram struct {
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Corners     []writeCornerOutline `json:"corners"`
}

// writeCastEntry mirrors castEntryForPrompt passed to write.md.
type writeCastEntry struct {
	ID          string `json:"id"`
	ProgramRole string `json:"program_role"`
	CornerRole  string `json:"corner_role,omitempty"`
}

// writeCorner mirrors cornerForPrompt passed to write.md.
type writeCorner struct {
	Title       string           `json:"title"`
	Content     string           `json:"content"`
	ScriptNote  string           `json:"script_note,omitempty"`
	Cast        []writeCastEntry `json:"cast"`
	TargetChars int              `json:"target_chars"`
}

// writeArticle mirrors RundownArticle passed to write.md.
type writeArticle struct {
	URL       string   `json:"url"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	Points    []string `json:"points"`
	Source    string   `json:"source,omitempty"`
	Author    string   `json:"author,omitempty"`
	Published string   `json:"published,omitempty"`
}

// writeCase is one entry in the write testdata files.
type writeCase struct {
	Name              string              `json:"name"`
	Category          string              `json:"category"`
	EpisodeNumber     string              `json:"episode_number"`
	RecordedAt        string              `json:"recorded_at"`
	Timezone          string              `json:"timezone"`
	Program           writeProgram        `json:"program"`
	Corner            writeCorner         `json:"corner"`
	Articles          []writeArticle      `json:"articles"`
	Flow              string              `json:"flow"`
	CastInfo          string              `json:"cast_info"`
	PresetInfo        string              `json:"preset_info"`
	GuestInfo         string              `json:"guest_info"`
	ProgramScriptNote string              `json:"program_script_note"`
	PreviousCorners   string              `json:"previous_corners"`
	PastEpisodes      string              `json:"past_episodes"`
	ValidSpeakerRoles []string            `json:"valid_speaker_roles"`
	ValidStyles       map[string][]string `json:"valid_styles"`
	ValidIntonation   []string            `json:"valid_intonation"`
	ValidPitch        []string            `json:"valid_pitch"`
	ValidSpeed        []string            `json:"valid_speed"`
	Expectation       string              `json:"expectation,omitempty"`
}

// writeLine mirrors model.Line for parsing write output.
type writeLine struct {
	SpeakerRole string `json:"speaker_role"`
	Style       string `json:"style,omitempty"`
	Intonation  string `json:"intonation,omitempty"`
	Pitch       string `json:"pitch,omitempty"`
	Speed       string `json:"speed,omitempty"`
	Text        string `json:"text"`
}

// writeOutput mirrors model.Lines for parsing write output.
type writeOutput struct {
	Lines []writeLine `json:"lines"`
}

// writeJudgeSchema is the JSON schema for the write judge LLM output.
var writeJudgeSchema = json.RawMessage(`{
  "type": "object",
  "required": ["scores"],
  "properties": {
    "scores": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["criterion", "score", "reason"],
        "properties": {
          "criterion": {
            "type": "string",
            "enum": ["content_fidelity", "character_consistency", "structure_compliance", "naturalness", "schema_compliance"]
          },
          "score": {"type": "integer", "minimum": 1, "maximum": 5},
          "reason": {"type": "string"}
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`)

// runWrite calls the write.md LLM with the given case inputs and returns raw JSON output.
func runWrite(ctx context.Context, t *testing.T, client llm.Client, promptTemplate, programJSON, cornerJSON, articlesJSON string, ec writeCase) (json.RawMessage, error) {
	t.Helper()
	prompt := strings.NewReplacer(
		"{{program}}", programJSON,
		"{{corner}}", cornerJSON,
		"{{articles}}", articlesJSON,
		"{{flow}}", ec.Flow,
		"{{cast_info}}", ec.CastInfo,
		"{{preset_info}}", ec.PresetInfo,
		"{{guest_info}}", ec.GuestInfo,
		"{{program_script_note}}", ec.ProgramScriptNote,
		"{{previous_corners}}", ec.PreviousCorners,
		"{{past_episodes}}", ec.PastEpisodes,
		"{{episode_number}}", ec.EpisodeNumber,
		"{{recorded_at}}", ec.RecordedAt,
		"{{timezone}}", ec.Timezone,
	).Replace(promptTemplate)

	schema := write.BuildLinesSchema(ec.ValidIntonation, ec.ValidPitch, ec.ValidSpeed)

	return client.Complete(ctx, llm.CompletionRequest{
		Messages:   []llm.Message{{Role: "user", Content: prompt}},
		JSONSchema: schema,
	})
}

func TestWriteEval(t *testing.T) {
	requireGeminiKey(t)

	targetClient, judgeClient := buildEvalClients(t)

	threshold, err := getEnvFloat("VOX_EVAL_WRITE_THRESHOLD", 4.0)
	if err != nil {
		t.Fatalf("parse VOX_EVAL_WRITE_THRESHOLD: %v", err)
	}

	sampleSize, seed := loadSampleParams(t)

	writePrompt, err := eval.LoadPrompt("write")
	if err != nil {
		t.Fatalf("load write prompt: %v", err)
	}

	judgePrompt := loadTestdataString(t, "write_judge.md")

	regressionCases := loadCasesJSON[writeCase](t, "write_regression_cases.json")
	poolCases := loadCasesJSON[writeCase](t, "write_pool_cases.json")

	allCases, caseByName := buildHarnessCases(t, regressionCases, poolCases, sampleSize, seed, func(c writeCase) string { return c.Name })

	ctx := context.Background()

	runEvalHarness(ctx, t, allCases, harnessConfig{
		Criteria:    eval.AllWriteCriteria,
		JudgeClient: judgeClient,
		JudgePrompt: judgePrompt,
		JudgeSchema: writeJudgeSchema,
		Threshold:   threshold,
		RunCase: func(ctx context.Context, t *testing.T, c harnessCase) (map[string]string, error) {
			ec := caseByName[c.Name]

			programJSON, err := json.Marshal(ec.Program)
			if err != nil {
				return nil, fmt.Errorf("marshal program for case %s: %w", c.Name, err)
			}
			cornerJSON, err := json.Marshal(ec.Corner)
			if err != nil {
				return nil, fmt.Errorf("marshal corner for case %s: %w", c.Name, err)
			}
			articlesJSON, err := json.Marshal(struct {
				Articles []writeArticle `json:"articles"`
			}{Articles: ec.Articles})
			if err != nil {
				return nil, fmt.Errorf("marshal articles for case %s: %w", c.Name, err)
			}

			raw, err := runWrite(ctx, t, targetClient, writePrompt, string(programJSON), string(cornerJSON), string(articlesJSON), ec)
			if err != nil {
				return nil, err
			}

			var output writeOutput
			if err := json.Unmarshal(raw, &output); err != nil {
				return nil, fmt.Errorf("unmarshal write output for case %s: %w", c.Name, err)
			}

			// Mechanical verification: speaker_role and style must be within declared valid values.
			validRoles := make(map[string]bool, len(ec.ValidSpeakerRoles))
			for _, r := range ec.ValidSpeakerRoles {
				validRoles[r] = true
			}
			for i, line := range output.Lines {
				if !validRoles[line.SpeakerRole] {
					t.Errorf("*** CONSTRAINT VIOLATION *** [%s] lines[%d].speaker_role=%q is not a valid character ID (valid: %v)",
						c.Name, i, line.SpeakerRole, ec.ValidSpeakerRoles)
				}
				if line.Style != "" {
					validStylesForChar := ec.ValidStyles[line.SpeakerRole]
					if !slices.Contains(validStylesForChar, line.Style) {
						t.Errorf("*** CONSTRAINT VIOLATION *** [%s] lines[%d].style=%q is not valid for speaker %q (valid: %v)",
							c.Name, i, line.Style, line.SpeakerRole, validStylesForChar)
					}
				}
			}

			outputJSON, err := json.Marshal(output)
			if err != nil {
				return nil, fmt.Errorf("marshal write output for case %s: %w", c.Name, err)
			}

			isFinalCorner := len(ec.Program.Corners) > 0 &&
				ec.Program.Corners[len(ec.Program.Corners)-1].Title == ec.Corner.Title
			isFinalCornerStr := strconv.FormatBool(isFinalCorner)

			previousCornersStr := ec.PreviousCorners
			if previousCornersStr == "" {
				previousCornersStr = "（なし）"
			}

			return map[string]string{
				"corner":           string(cornerJSON),
				"articles":         string(articlesJSON),
				"flow":             ec.Flow,
				"cast_info":        ec.CastInfo,
				"program":          string(programJSON),
				"previous_corners": previousCornersStr,
				"is_final_corner":  isFinalCornerStr,
				"write_output":     string(outputJSON),
				"expectation":      eval.ResolveExpectation(ec.Expectation),
			}, nil
		},
	})
}
