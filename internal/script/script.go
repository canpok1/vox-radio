package script

import (
	"context"
	"fmt"
	"log/slog"
	"unicode/utf8"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/logging"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/direct"
	"github.com/canpok1/vox-radio/internal/script/write"
)

type ScriptGenerator interface {
	Generate(ctx context.Context, program config.ProgramConfig, rundown model.Rundown, corners []config.CornerConfig, chars map[string]config.CharacterConfig) (model.Script, model.ScriptLines, *model.ProofreadResult, error)
}

type LLMScriptGenerator struct {
	writer          write.Writer
	director        direct.Director
	assetCatalog    model.AssetCatalog
	logger          *slog.Logger
	regenThreshold  float64
	regenMaxRetries int
}

// GeneratorOption configures a LLMScriptGenerator.
type GeneratorOption func(*LLMScriptGenerator)

// WithLogger sets the logger used for progress messages.
func WithLogger(l *slog.Logger) GeneratorOption {
	return func(g *LLMScriptGenerator) { g.logger = l }
}

// WithRegenConfig sets the regenIfNeeded per-corner deviation threshold and max retry count.
// Defaults to config.DefaultScriptRegenThreshold / config.DefaultScriptRegenMaxRetries when not set.
func WithRegenConfig(threshold float64, maxRetries int) GeneratorOption {
	return func(g *LLMScriptGenerator) {
		g.regenThreshold = threshold
		g.regenMaxRetries = maxRetries
	}
}

func NewLLMScriptGenerator(
	w write.Writer,
	d direct.Director,
	assetCatalog model.AssetCatalog,
	opts ...GeneratorOption,
) *LLMScriptGenerator {
	g := &LLMScriptGenerator{
		writer:          w,
		director:        d,
		assetCatalog:    assetCatalog,
		logger:          slog.Default(),
		regenThreshold:  config.DefaultScriptRegenThreshold,
		regenMaxRetries: config.DefaultScriptRegenMaxRetries,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

func (g *LLMScriptGenerator) Generate(ctx context.Context, program config.ProgramConfig, rundown model.Rundown, corners []config.CornerConfig, chars map[string]config.CharacterConfig) (model.Script, model.ScriptLines, *model.ProofreadResult, error) {
	scriptLogger := g.logger.With("step", "script")
	scriptDone := logging.StartStep(ctx, scriptLogger, "開始")

	cornerMap := rundown.CornerMap()

	// コーナーごとのキャスト割り当てを生成（休演キャストを除外）
	allAssignments := make([][]write.CastAssignment, len(corners))
	for i, corner := range corners {
		allAssignments[i] = MergeCornerCast(corner, rundown.Casts)
	}

	writeLogger := g.logger.With("step", "script/write")
	writeDone := logging.StartStep(ctx, writeLogger, "開始")
	cornerLines, err := WriteAll(ctx, g.writer, program, corners, allAssignments, cornerMap, chars)
	if err != nil {
		return model.Script{}, model.ScriptLines{}, nil, err
	}
	cornerLines = g.regenIfNeeded(ctx, program, cornerLines, corners, allAssignments, cornerMap, chars)
	scriptLines := model.ScriptLines{Direction: program.Direction, Corners: BuildScriptLines(corners, cornerLines)}
	writeDone(fmt.Sprintf("%dコーナー", len(corners)))

	directLogger := g.logger.With("step", "script/direct")
	directDone := logging.StartStep(ctx, directLogger, "開始")
	scr, pr, err := g.director.Direct(ctx, scriptLines.Corners, g.assetCatalog, program.Direction)
	if err != nil {
		return model.Script{}, model.ScriptLines{}, nil, fmt.Errorf("direct: %w", err)
	}

	if pr != nil {
		directLogger.Info("校正完了", "count", len(pr.Corrections))
	}
	directDone("")

	scriptDone(fmt.Sprintf("%dセグメント", len(scr.Segments)))

	return scr, scriptLines, pr, nil
}

// WriteAll writes lines for each corner in order, passing previously generated corners as context.
func WriteAll(ctx context.Context, w write.Writer, program config.ProgramConfig, corners []config.CornerConfig, assignments [][]write.CastAssignment, cornerMap map[string]model.RundownCorner, chars map[string]config.CharacterConfig) ([][]model.Line, error) {
	result := make([][]model.Line, len(corners))
	previous := make([]model.CornerLines, 0, len(corners))
	for i, corner := range corners {
		rc := cornerMap[corner.ID]
		a := assignments[i]
		if cas, ok := w.(write.CornerAppearanceSetter); ok {
			cas.SetCornerAppearance(rc.AppearanceCount, rc.LastEpisodeNumber)
		}
		lines, err := w.Write(ctx, program, corner, a, corners, previous, rc.Articles, rc.Flow, chars)
		if err != nil {
			return nil, fmt.Errorf("write corner %q: %w", corner.Title, err)
		}
		result[i] = lines
		previous = append(previous, model.CornerLines{Title: corner.Title, Lines: lines})
	}
	return result, nil
}

// MergeCornerCast generates per-corner cast assignments from the episode's cast set.
// Only episode cast members (from Select result) appear in assignments.
// corner.Cast provides corner-specific role annotations for cast members present in the episode set.
// Cast members not in the episode set are excluded even if listed in corner.Cast.
func MergeCornerCast(corner config.CornerConfig, casts []model.RundownCast) []write.CastAssignment {
	assignments := make([]write.CastAssignment, 0, len(casts))
	for _, c := range casts {
		assignments = append(assignments, write.CastAssignment{
			CharacterID: c.CharacterID,
			Type:        c.Type,
			ProgramRole: c.Role,
			CornerRole:  corner.Cast[c.CharacterID], // "" if not in corner cast
		})
	}
	// casts は Select() によりすでに charID 昇順
	return assignments
}

// buildPreviousCorners assembles the first n corners into a []model.CornerLines for context passing.
func buildPreviousCorners(corners []config.CornerConfig, cornerLines [][]model.Line, n int) []model.CornerLines {
	previous := make([]model.CornerLines, n)
	for i := range n {
		previous[i] = model.CornerLines{Title: corners[i].Title, Lines: cornerLines[i]}
	}
	return previous
}

// cornerDeviation is a corner's index paired with its |actual-target|/target deviation.
type cornerDeviation struct {
	idx int
	dev float64
}

// exceedingCorners returns the corners (by index) whose written character count deviates from
// target_chars by more than threshold, in corner order. Corners with target_chars<=0 (e.g. no
// length_sec/target_chars configured) are skipped, matching the previous total-based check.
func exceedingCorners(corners []config.CornerConfig, cornerLines [][]model.Line, charsPerMin int, threshold float64) []cornerDeviation {
	exceeding := make([]cornerDeviation, 0, len(corners))
	for i, corner := range corners {
		target := corner.EffectiveTargetChars(charsPerMin)
		if target <= 0 {
			continue
		}
		dev := absDeviation(countChars(cornerLines[i]), target)
		if dev > threshold {
			exceeding = append(exceeding, cornerDeviation{idx: i, dev: dev})
		}
	}
	return exceeding
}

// regenIfNeeded regenerates any corner whose written character count deviates from target_chars
// by more than g.regenThreshold, retrying up to g.regenMaxRetries times. Judgment is per corner
// (not the episode total): a corner far over target and another far under can cancel out in the
// total while both individually need fixing. Corners still exceeding after the last retry are
// left as-is and logged as a warning.
func (g *LLMScriptGenerator) regenIfNeeded(ctx context.Context, program config.ProgramConfig, cornerLines [][]model.Line, corners []config.CornerConfig, allAssignments [][]write.CastAssignment, cornerMap map[string]model.RundownCorner, chars map[string]config.CharacterConfig) [][]model.Line {
	if len(corners) == 0 {
		return cornerLines
	}
	logger := g.logger.With("step", "script/regen")
	charsPerMin := program.EffectiveCharsPerMinute()

	for attempt := 0; ; attempt++ {
		exceeding := exceedingCorners(corners, cornerLines, charsPerMin, g.regenThreshold)
		if len(exceeding) == 0 {
			return cornerLines
		}
		if attempt >= g.regenMaxRetries {
			for _, e := range exceeding {
				logger.Warn("目標文字数からの乖離が閾値を超えたまま打ち切りました", "corner_id", corners[e.idx].ID, "deviation", e.dev, "threshold", g.regenThreshold)
			}
			return cornerLines
		}
		for _, e := range exceeding {
			corner := corners[e.idx]
			logger.Info("目標文字数からの乖離により再生成します", "corner_id", corner.ID, "deviation", e.dev, "threshold", g.regenThreshold, "attempt", attempt+1)
			rc := cornerMap[corner.ID]
			previous := buildPreviousCorners(corners, cornerLines, e.idx)
			if cas, ok := g.writer.(write.CornerAppearanceSetter); ok {
				cas.SetCornerAppearance(rc.AppearanceCount, rc.LastEpisodeNumber)
			}
			if newLines, err := g.writer.Write(ctx, program, corner, allAssignments[e.idx], corners, previous, rc.Articles, rc.Flow, chars); err == nil {
				cornerLines[e.idx] = newLines
			}
		}
	}
}

// BuildScriptLines converts per-corner config and line slices into a []model.CornerLines.
// Asset fields (StartAudio, EndAudio, BGM) are transferred from CornerConfig
// so they are available during deterministic segment injection in the direct step.
func BuildScriptLines(corners []config.CornerConfig, cornerLines [][]model.Line) []model.CornerLines {
	result := make([]model.CornerLines, len(corners))
	for i, corner := range corners {
		result[i] = model.CornerLines{
			ID:            corner.ID,
			Title:         corner.Title,
			Direction:     corner.Direction,
			Lines:         cornerLines[i],
			StartAudio:    audioRefToCornerAudio(corner.StartAudio),
			EndAudio:      audioRefToCornerAudio(corner.EndAudio),
			BGM:           corner.EffectiveBGM(),
			StartPauseSec: corner.EffectiveStartPauseSec(),
			EndPauseSec:   corner.EffectiveEndPauseSec(),
		}
	}
	return result
}

func audioRefToCornerAudio(ref *config.AudioRef) *model.CornerAudio {
	if ref == nil {
		return nil
	}
	return &model.CornerAudio{
		Type:      model.SegmentType(ref.Type),
		AssetName: ref.ID,
	}
}

func countChars(lines []model.Line) int {
	total := 0
	for _, l := range lines {
		total += utf8.RuneCountInString(l.Text)
	}
	return total
}

func absDeviation(actual, target int) float64 {
	return float64(max(actual-target, target-actual)) / float64(target)
}
