package script

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/direct"
	"github.com/canpok1/vox-radio/internal/script/plan"
	"github.com/canpok1/vox-radio/internal/script/summarize"
	"github.com/canpok1/vox-radio/internal/script/write"
)

const regenThreshold = 0.20

type ScriptGenerator interface {
	Generate(ctx context.Context, articles []model.Article, show model.ShowConfig) (model.Script, error)
}

type LLMScriptGenerator struct {
	summarizer summarize.Summarizer
	planner    plan.Planner
	writer     write.Writer
	director   direct.Director
	seCatalog  model.SECatalog
	workDir    string
}

func NewLLMScriptGenerator(
	s summarize.Summarizer,
	p plan.Planner,
	w write.Writer,
	d direct.Director,
	seCatalog model.SECatalog,
	workDir string,
) *LLMScriptGenerator {
	return &LLMScriptGenerator{
		summarizer: s,
		planner:    p,
		writer:     w,
		director:   d,
		seCatalog:  seCatalog,
		workDir:    workDir,
	}
}

func (g *LLMScriptGenerator) Generate(ctx context.Context, articles []model.Article, show model.ShowConfig) (model.Script, error) {
	summaries, err := g.summarizeAll(ctx, articles)
	if err != nil {
		return model.Script{}, err
	}
	if err := g.saveIntermediate("summaries.json", model.Summaries{Summaries: summaries}); err != nil {
		return model.Script{}, err
	}

	rundown, err := g.planner.Plan(ctx, summaries, show)
	if err != nil {
		return model.Script{}, fmt.Errorf("plan: %w", err)
	}
	if err := g.saveIntermediate("rundown.json", rundown); err != nil {
		return model.Script{}, err
	}

	cornerLines, summaryByURL, err := g.writeAll(ctx, rundown, summaries, show)
	if err != nil {
		return model.Script{}, err
	}
	cornerLines = g.regenIfNeeded(ctx, cornerLines, rundown, summaryByURL, show)
	allLines := flatten(cornerLines)
	if err := g.saveIntermediate("lines.json", model.Lines{Lines: allLines}); err != nil {
		return model.Script{}, err
	}

	scr, err := g.director.Direct(ctx, allLines, g.seCatalog)
	if err != nil {
		return model.Script{}, fmt.Errorf("direct: %w", err)
	}

	return scr, nil
}

func (g *LLMScriptGenerator) summarizeAll(ctx context.Context, articles []model.Article) ([]model.Summary, error) {
	summaries := make([]model.Summary, 0, len(articles))
	for _, a := range articles {
		s, err := g.summarizer.Summarize(ctx, a)
		if err != nil {
			return nil, fmt.Errorf("summarize %q: %w", a.URL, err)
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func (g *LLMScriptGenerator) writeAll(ctx context.Context, rundown model.Rundown, summaries []model.Summary, show model.ShowConfig) ([][]model.Line, map[string]model.Summary, error) {
	summaryByURL := SummaryByURL(summaries)

	result := make([][]model.Line, len(rundown.Corners))
	for i, corner := range rundown.Corners {
		relevant := CornerSummaries(corner, summaryByURL)
		lines, err := g.writer.Write(ctx, corner, relevant, show)
		if err != nil {
			return nil, nil, fmt.Errorf("write corner %q: %w", corner.Title, err)
		}
		result[i] = lines
	}
	return result, summaryByURL, nil
}

func (g *LLMScriptGenerator) regenIfNeeded(ctx context.Context, cornerLines [][]model.Line, rundown model.Rundown, summaryByURL map[string]model.Summary, show model.ShowConfig) [][]model.Line {
	if show.TargetChars <= 0 || len(rundown.Corners) == 0 {
		return cornerLines
	}

	totalChars := 0
	for _, lines := range cornerLines {
		totalChars += countChars(lines)
	}

	deviation := float64(max(totalChars-show.TargetChars, show.TargetChars-totalChars)) / float64(show.TargetChars)
	if deviation <= regenThreshold {
		return cornerLines
	}

	worstIdx := 0
	worstDev := 0.0
	for i, corner := range rundown.Corners {
		if corner.TargetChars <= 0 {
			continue
		}
		actual := countChars(cornerLines[i])
		dev := float64(max(actual-corner.TargetChars, corner.TargetChars-actual)) / float64(corner.TargetChars)
		if dev > worstDev {
			worstDev = dev
			worstIdx = i
		}
	}

	corner := rundown.Corners[worstIdx]
	relevant := CornerSummaries(corner, summaryByURL)
	if newLines, err := g.writer.Write(ctx, corner, relevant, show); err == nil {
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

// SummaryByURL builds a URL-keyed lookup map from a summaries slice.
func SummaryByURL(summaries []model.Summary) map[string]model.Summary {
	m := make(map[string]model.Summary, len(summaries))
	for _, s := range summaries {
		m[s.URL] = s
	}
	return m
}

// CornerSummaries returns the summaries relevant to a corner, in summary_urls order.
func CornerSummaries(corner model.Corner, byURL map[string]model.Summary) []model.Summary {
	result := make([]model.Summary, 0, len(corner.SummaryURLs))
	for _, u := range corner.SummaryURLs {
		if s, ok := byURL[u]; ok {
			result = append(result, s)
		}
	}
	return result
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
