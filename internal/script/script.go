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
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/direct"
	"github.com/canpok1/vox-radio/internal/script/summarize"
	"github.com/canpok1/vox-radio/internal/script/write"
)

const regenThreshold = 0.20

type ScriptGenerator interface {
	Generate(ctx context.Context, articles model.Articles, corners []config.CornerConfig, chars map[string]config.CharacterConfig) (model.Script, error)
}

type LLMScriptGenerator struct {
	summarizer summarize.Summarizer
	writer     write.Writer
	director   direct.Director
	seCatalog  model.SECatalog
	workDir    string
	logger     *slog.Logger
}

// GeneratorOption configures a LLMScriptGenerator.
type GeneratorOption func(*LLMScriptGenerator)

// WithLogger sets the logger used for progress messages.
func WithLogger(l *slog.Logger) GeneratorOption {
	return func(g *LLMScriptGenerator) { g.logger = l }
}

func NewLLMScriptGenerator(
	s summarize.Summarizer,
	w write.Writer,
	d direct.Director,
	seCatalog model.SECatalog,
	workDir string,
	opts ...GeneratorOption,
) *LLMScriptGenerator {
	g := &LLMScriptGenerator{
		summarizer: s,
		writer:     w,
		director:   d,
		seCatalog:  seCatalog,
		workDir:    workDir,
		logger:     slog.Default(),
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

func (g *LLMScriptGenerator) Generate(ctx context.Context, articles model.Articles, corners []config.CornerConfig, chars map[string]config.CharacterConfig) (model.Script, error) {
	logger := g.logger.With("step", "script")
	start := time.Now()

	cornerArticlesMap := articles.CornerMap()

	totalArticles := 0
	for _, ca := range articles.Corners {
		totalArticles += len(ca.Articles)
	}

	sumLogger := g.logger.With("step", "script/summarize")
	sumLogger.Info(fmt.Sprintf("開始 (%d記事)", totalArticles))
	sumStart := time.Now()

	cornerSummaries := make([]model.CornerSummaries, 0, len(corners))
	done := 0
	for _, corner := range corners {
		arts := cornerArticlesMap[corner.Title]
		sums, err := g.summarizeAllWithProgress(ctx, arts, &done, totalArticles, sumLogger)
		if err != nil {
			return model.Script{}, err
		}
		cornerSummaries = append(cornerSummaries, model.CornerSummaries{
			CornerTitle: corner.Title,
			Summaries:   sums,
		})
	}
	sumLogger.Info(fmt.Sprintf("完了 (%.1fs)", time.Since(sumStart).Seconds()))

	allSummaries := model.Summaries{Corners: cornerSummaries}
	if err := g.saveIntermediate("summaries.json", allSummaries); err != nil {
		return model.Script{}, err
	}

	writeLogger := g.logger.With("step", "script/write")
	writeLogger.Info("開始")
	cornerLines, err := g.writeAll(ctx, corners, allSummaries, chars)
	if err != nil {
		return model.Script{}, err
	}
	cornerLines = g.regenIfNeeded(ctx, cornerLines, corners, allSummaries, chars)
	allLines := flatten(cornerLines)
	if err := g.saveIntermediate("lines.json", model.Lines{Lines: allLines}); err != nil {
		return model.Script{}, err
	}

	directLogger := g.logger.With("step", "script/direct")
	directLogger.Info("開始")
	scr, err := g.director.Direct(ctx, allLines, g.seCatalog)
	if err != nil {
		return model.Script{}, fmt.Errorf("direct: %w", err)
	}

	logger.Info(fmt.Sprintf("完了 (%dセグメント, %.1fs)", len(scr.Segments), time.Since(start).Seconds()))

	return scr, nil
}

func (g *LLMScriptGenerator) summarizeAllWithProgress(ctx context.Context, articles []model.Article, done *int, total int, logger *slog.Logger) ([]model.Summary, error) {
	summaries := make([]model.Summary, 0, len(articles))
	for _, a := range articles {
		*done++
		logger.Info(fmt.Sprintf("記事を要約中 (%d/%d)", *done, total))
		s, err := g.summarizer.Summarize(ctx, a)
		if err != nil {
			return nil, fmt.Errorf("summarize %q: %w", a.URL, err)
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func (g *LLMScriptGenerator) writeAll(ctx context.Context, corners []config.CornerConfig, sums model.Summaries, chars map[string]config.CharacterConfig) ([][]model.Line, error) {
	cornerSumsMap := sums.CornerMap()
	result := make([][]model.Line, len(corners))
	for i, corner := range corners {
		lines, err := g.writer.Write(ctx, corner, cornerSumsMap[corner.Title], chars)
		if err != nil {
			return nil, fmt.Errorf("write corner %q: %w", corner.Title, err)
		}
		result[i] = lines
	}
	return result, nil
}

func (g *LLMScriptGenerator) regenIfNeeded(ctx context.Context, cornerLines [][]model.Line, corners []config.CornerConfig, sums model.Summaries, chars map[string]config.CharacterConfig) [][]model.Line {
	if len(corners) == 0 {
		return cornerLines
	}
	totalTarget := 0
	for _, c := range corners {
		totalTarget += config.DurationSecToTargetChars(c.TargetDurationSec)
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

	cornerSumsMap := sums.CornerMap()
	worstIdx := 0
	worstDev := 0.0
	for i, corner := range corners {
		targetChars := config.DurationSecToTargetChars(corner.TargetDurationSec)
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
	if newLines, err := g.writer.Write(ctx, corner, cornerSumsMap[corner.Title], chars); err == nil {
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

func flatten(lines [][]model.Line) []model.Line {
	total := 0
	for _, l := range lines {
		total += len(l)
	}
	result := make([]model.Line, 0, total)
	for _, l := range lines {
		result = append(result, l...)
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
