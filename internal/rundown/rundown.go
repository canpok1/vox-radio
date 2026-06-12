package rundown

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/rundown/flow"
	sel "github.com/canpok1/vox-radio/internal/rundown/select"
	"github.com/canpok1/vox-radio/internal/script/summarize"
)

// Rundowner generates a Rundown from collected articles.
type Rundowner interface {
	Run(ctx context.Context, corners []config.CornerConfig, articles model.Articles, casts []model.RundownCast) (model.Rundown, error)
}

// Option configures an LLMRundowner.
type Option func(*LLMRundowner)

// WithLogger sets the logger used for log output.
func WithLogger(l *slog.Logger) Option {
	return func(r *LLMRundowner) { r.logger = l }
}

// LLMRundowner uses Selector + Summarizer + FlowDesigner to produce a Rundown.
type LLMRundowner struct {
	selector          sel.Selector
	summarizer        summarize.Summarizer
	flowDesigner      flow.Designer
	excludedDedupKeys map[string]struct{}
	cornerAppearances map[string]cache.CornerAppearance
	logger            *slog.Logger
}

// SetCornerAppearances configures per-corner appearance stats (keyed by corner ID) aggregated from
// cache history. Run bakes the count (including this episode) and last episode number into each
// RundownCorner and passes them to the selector. nil/missing IDs are treated as new corners.
func (r *LLMRundowner) SetCornerAppearances(m map[string]cache.CornerAppearance) {
	r.cornerAppearances = m
}

// NewLLMRundowner creates a LLMRundowner.
// excludedDedupKeys is the set of article DedupKeys to exclude before selection (nil = no exclusion).
func NewLLMRundowner(selector sel.Selector, summarizer summarize.Summarizer, designer flow.Designer, excludedDedupKeys []string, opts ...Option) *LLMRundowner {
	excluded := make(map[string]struct{}, len(excludedDedupKeys))
	for _, k := range excludedDedupKeys {
		excluded[k] = struct{}{}
	}
	r := &LLMRundowner{
		selector:          selector,
		summarizer:        summarizer,
		flowDesigner:      designer,
		excludedDedupKeys: excluded,
		logger:            slog.Default(),
	}
	for _, opt := range opts {
		opt(r)
	}
	r.logger = r.logger.With("step", "rundown")
	return r
}

func (r *LLMRundowner) Run(ctx context.Context, corners []config.CornerConfig, articles model.Articles, casts []model.RundownCast) (model.Rundown, error) {
	articleMap := articles.CornerMap()
	rundownCorners := make([]model.RundownCorner, 0, len(corners))

	// フェーズ1: 各コーナーの記事選別・要約・選別理由を収集
	for _, corner := range corners {
		cornerArticles := articleMap[corner.Title]

		// 扱い回数文脈（今回含む・初回=1）と前回出演回番号を算出。
		// キャスト（ADR-0040）と異なり LLM へは -1 の境界変換をせず生値のまま渡す（ADR-0052）。
		appearance := r.cornerAppearances[corner.ID]
		appearanceCount := appearance.Count + 1
		lastEpisodeNumber := appearance.LastEpisodeNumber

		filtered := make([]model.Article, 0, len(cornerArticles))
		for _, a := range cornerArticles {
			if _, excluded := r.excludedDedupKeys[a.DedupKey]; !excluded {
				filtered = append(filtered, a)
			}
		}
		if n := len(cornerArticles) - len(filtered); n > 0 {
			r.logger.Info("excluded past articles", "corner", corner.Title, "count", n)
		}
		cornerArticles = filtered

		if len(cornerArticles) == 0 {
			rundownCorners = append(rundownCorners, model.RundownCorner{
				ID:                corner.ID,
				Title:             corner.Title,
				Articles:          make([]model.RundownArticle, 0),
				AppearanceCount:   appearanceCount,
				LastEpisodeNumber: lastEpisodeNumber,
			})
			continue
		}

		// 選別 LLM にこのコーナーの扱い回数文脈を渡す（supplementary interface）
		if s, ok := r.selector.(sel.CornerAppearanceSetter); ok {
			s.SetCornerAppearance(appearanceCount, lastEpisodeNumber)
		}

		selected, err := r.selector.Select(ctx, corner, cornerArticles)
		if err != nil {
			return model.Rundown{}, fmt.Errorf("select corner %q: %w", corner.Title, err)
		}

		// Build DedupKey→Article index for fast lookup
		articleByDedupKey := make(map[string]model.Article, len(cornerArticles))
		for _, a := range cornerArticles {
			articleByDedupKey[a.DedupKey] = a
		}

		rdArticles := make([]model.RundownArticle, 0, len(selected.SelectedIDs))
		for _, id := range selected.SelectedIDs {
			a, ok := articleByDedupKey[id]
			if !ok {
				continue
			}
			sum, err := r.summarizer.Summarize(ctx, a)
			if err != nil {
				return model.Rundown{}, fmt.Errorf("summarize %q: %w", id, err)
			}
			rdArticles = append(rdArticles, model.NewRundownArticle(
				a.DedupKey, a.URL, a.Title, a.Description, a.Body, sum.Points,
				a.Source, a.Author, a.Published,
			))
		}

		rundownCorners = append(rundownCorners, model.RundownCorner{
			ID:                corner.ID,
			Title:             corner.Title,
			SelectionReason:   selected.SelectionReason,
			Articles:          rdArticles,
			AppearanceCount:   appearanceCount,
			LastEpisodeNumber: lastEpisodeNumber,
		})
	}

	// キャストをセット
	rd := model.Rundown{
		Corners: rundownCorners,
		Casts:   casts,
	}

	// フェーズ2: 番組構成全体を文脈に全コーナーの flow を設計
	last := len(corners) - 1
	for i, corner := range corners {
		designed, err := r.flowDesigner.DesignFlow(ctx, corner, flow.PositionFor(i, last), rd.Corners[i], rd)
		if err != nil {
			return model.Rundown{}, fmt.Errorf("design flow for corner %q: %w", corner.Title, err)
		}
		rd.Corners[i].Flow = designed
	}

	return rd, nil
}
