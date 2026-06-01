package script

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/direct"
	"github.com/canpok1/vox-radio/internal/script/write"
)

const regenThreshold = 0.20

type ScriptGenerator interface {
	Generate(ctx context.Context, program config.ProgramConfig, rundown model.Rundown, corners []config.CornerConfig, chars map[string]config.CharacterConfig) (model.Script, error)
}

type LLMScriptGenerator struct {
	writer       write.Writer
	director     direct.Director
	assetCatalog model.AssetCatalog
	workDir      string
	logger       *slog.Logger
}

// GeneratorOption configures a LLMScriptGenerator.
type GeneratorOption func(*LLMScriptGenerator)

// WithLogger sets the logger used for progress messages.
func WithLogger(l *slog.Logger) GeneratorOption {
	return func(g *LLMScriptGenerator) { g.logger = l }
}

func NewLLMScriptGenerator(
	w write.Writer,
	d direct.Director,
	assetCatalog model.AssetCatalog,
	workDir string,
	opts ...GeneratorOption,
) *LLMScriptGenerator {
	g := &LLMScriptGenerator{
		writer:       w,
		director:     d,
		assetCatalog: assetCatalog,
		workDir:      workDir,
		logger:       slog.Default(),
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

func (g *LLMScriptGenerator) Generate(ctx context.Context, program config.ProgramConfig, rundown model.Rundown, corners []config.CornerConfig, chars map[string]config.CharacterConfig) (model.Script, error) {
	start := time.Now()

	cornerMap := rundown.CornerMap()

	g.logger.With("step", "script/write").Info("開始")
	cornerLines, err := g.writeAll(ctx, program, corners, cornerMap, chars)
	if err != nil {
		return model.Script{}, err
	}
	cornerLines = g.regenIfNeeded(ctx, program, cornerLines, corners, cornerMap, chars)
	scriptLines := BuildScriptLines(corners, cornerLines)
	if err := g.saveIntermediate(fileio.FileLines, model.ScriptLines{Corners: scriptLines}); err != nil {
		return model.Script{}, err
	}

	g.logger.With("step", "script/direct").Info("開始")
	scr, err := g.director.Direct(ctx, scriptLines, g.assetCatalog)
	if err != nil {
		return model.Script{}, fmt.Errorf("direct: %w", err)
	}

	g.logger.With("step", "script").Info(fmt.Sprintf("完了 (%dセグメント, %.1fs)", len(scr.Segments), time.Since(start).Seconds()))

	return scr, nil
}

func (g *LLMScriptGenerator) writeAll(ctx context.Context, program config.ProgramConfig, corners []config.CornerConfig, cornerMap map[string]model.RundownCorner, chars map[string]config.CharacterConfig) ([][]model.Line, error) {
	result := make([][]model.Line, len(corners))
	previous := make([]model.CornerLines, 0, len(corners))
	for i, corner := range corners {
		rc := cornerMap[corner.Title]
		lines, err := g.writer.Write(ctx, program, corner, corners, previous, rc.Articles, rc.Flow, chars)
		if err != nil {
			return nil, fmt.Errorf("write corner %q: %w", corner.Title, err)
		}
		result[i] = lines
		previous = append(previous, model.CornerLines{Title: corner.Title, Lines: lines})
	}
	return result, nil
}

// buildPreviousCorners assembles the first n corners into a []model.CornerLines for context passing.
func buildPreviousCorners(corners []config.CornerConfig, cornerLines [][]model.Line, n int) []model.CornerLines {
	previous := make([]model.CornerLines, n)
	for i := range n {
		previous[i] = model.CornerLines{Title: corners[i].Title, Lines: cornerLines[i]}
	}
	return previous
}

func (g *LLMScriptGenerator) regenIfNeeded(ctx context.Context, program config.ProgramConfig, cornerLines [][]model.Line, corners []config.CornerConfig, cornerMap map[string]model.RundownCorner, chars map[string]config.CharacterConfig) [][]model.Line {
	if len(corners) == 0 {
		return cornerLines
	}
	totalTarget := 0
	for _, c := range corners {
		totalTarget += config.DurationSecToTargetChars(c.LengthSec)
	}
	if totalTarget <= 0 {
		return cornerLines
	}

	totalChars := 0
	for _, lines := range cornerLines {
		totalChars += countChars(lines)
	}

	if absDeviation(totalChars, totalTarget) <= regenThreshold {
		return cornerLines
	}

	worstIdx := 0
	worstDev := 0.0
	for i, corner := range corners {
		targetChars := config.DurationSecToTargetChars(corner.LengthSec)
		if targetChars <= 0 {
			continue
		}
		actual := countChars(cornerLines[i])
		dev := absDeviation(actual, targetChars)
		if dev > worstDev {
			worstDev = dev
			worstIdx = i
		}
	}
	if worstDev == 0 {
		return cornerLines
	}

	corner := corners[worstIdx]
	rc := cornerMap[corner.Title]
	previous := buildPreviousCorners(corners, cornerLines, worstIdx)
	if newLines, err := g.writer.Write(ctx, program, corner, corners, previous, rc.Articles, rc.Flow, chars); err == nil {
		cornerLines[worstIdx] = newLines
	}
	return cornerLines
}

func (g *LLMScriptGenerator) saveIntermediate(filename string, v any) error {
	if g.workDir == "" {
		return nil
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filename, err)
	}
	path := filepath.Join(g.workDir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filename, err)
	}
	return nil
}

// BuildScriptLines converts per-corner config and line slices into a []model.CornerLines.
// Asset fields (StartJingle, EndJingle, BGM) are transferred from CornerConfig
// so they are available during deterministic segment injection in the direct step.
func BuildScriptLines(corners []config.CornerConfig, cornerLines [][]model.Line) []model.CornerLines {
	result := make([]model.CornerLines, len(corners))
	for i, corner := range corners {
		result[i] = model.CornerLines{
			Title:       corner.Title,
			Direction:   corner.Direction,
			Lines:       cornerLines[i],
			StartJingle: corner.StartJingle,
			EndJingle:   corner.EndJingle,
			BGM:         corner.BGM,
		}
	}
	return result
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
