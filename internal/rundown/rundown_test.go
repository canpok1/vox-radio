package rundown_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/logging"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/rundown"
	"github.com/canpok1/vox-radio/internal/rundown/flow"
	sel "github.com/canpok1/vox-radio/internal/rundown/select"
	"github.com/canpok1/vox-radio/internal/script/summarize"
)

// --- mocks ---

type mockSelector struct {
	result           sel.SelectResult
	err              error
	called           bool
	receivedArticles []model.Article
}

func (m *mockSelector) Select(_ context.Context, _ config.CornerConfig, articles []model.Article) (sel.SelectResult, error) {
	m.called = true
	m.receivedArticles = articles
	return m.result, m.err
}

// appearanceCapturingSelector records the corner appearance values set via SetCornerAppearance.
type appearanceCapturingSelector struct {
	result        sel.SelectResult
	capturedCount []int
	capturedLast  []int
}

func (m *appearanceCapturingSelector) SetCornerAppearance(count, last int) {
	m.capturedCount = append(m.capturedCount, count)
	m.capturedLast = append(m.capturedLast, last)
}

func (m *appearanceCapturingSelector) Select(_ context.Context, _ config.CornerConfig, _ []model.Article) (sel.SelectResult, error) {
	return m.result, nil
}

var _ sel.CornerAppearanceSetter = (*appearanceCapturingSelector)(nil)

type mockSummarizer struct {
	byURL         map[string]model.Summary
	err           error
	receivedTexts map[string]string
}

func (m *mockSummarizer) Summarize(_ context.Context, a model.Article) (model.Summary, error) {
	if m.receivedTexts == nil {
		m.receivedTexts = make(map[string]string)
	}
	m.receivedTexts[a.URL] = a.Text()
	if m.err != nil {
		return model.Summary{}, m.err
	}
	if s, ok := m.byURL[a.URL]; ok {
		return s, nil
	}
	return model.Summary{URL: a.URL, Points: []string{"p1"}}, nil
}

// ensure mockSummarizer implements summarize.Summarizer
var _ summarize.Summarizer = (*mockSummarizer)(nil)

type mockFlowDesigner struct {
	flow            string
	err             error
	called          bool
	capturedCorners []config.CornerConfig
	capturedRundown model.Rundown
}

func (m *mockFlowDesigner) DesignFlow(_ context.Context, corner config.CornerConfig, _ flow.Position, _ model.RundownCorner, rd model.Rundown) (string, error) {
	m.called = true
	m.capturedCorners = append(m.capturedCorners, corner)
	m.capturedRundown = rd
	if m.err != nil {
		return "", m.err
	}
	return m.flow, nil
}

var _ flow.Designer = (*mockFlowDesigner)(nil)

type positionCapturingDesigner struct {
	positions *[]flow.Position
	flow      string
	err       error
}

func (d *positionCapturingDesigner) DesignFlow(_ context.Context, _ config.CornerConfig, pos flow.Position, _ model.RundownCorner, _ model.Rundown) (string, error) {
	*d.positions = append(*d.positions, pos)
	return d.flow, d.err
}

// --- helpers ---

func defaultCorner(title string) config.CornerConfig {
	return config.CornerConfig{Title: title, Content: "内容", LengthSec: 60}
}

func article(url string) model.Article {
	return model.Article{DedupKey: url, URL: url, Title: "記事: " + url, Description: "本文"}
}

func newRundowner(ms *mockSelector, msum *mockSummarizer, excludedDedupKeys []string, mfd *mockFlowDesigner, opts ...rundown.Option) *rundown.LLMRundowner {
	return rundown.NewLLMRundowner(ms, msum, mfd, excludedDedupKeys, opts...)
}

// --- tests ---

func TestLLMRundowner_Run_BakesCornerAppearance(t *testing.T) {
	ms := &mockSelector{result: sel.SelectResult{SelectedIDs: []string{"u1"}, SelectionReason: "理由"}}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, &mockSummarizer{}, nil, mfd)
	rd.SetCornerAppearances(map[string]cache.CornerAppearance{
		"tech": {Count: 2, LastEpisodeNumber: 3}, // 過去2回・前回は第3回 → 今回含め3回目
	})

	corners := []config.CornerConfig{
		{ID: "tech", Title: "テック", Content: "内容", LengthSec: 60},       // 記事ありコーナー
		{ID: "opening", Title: "オープニング", Content: "導入", LengthSec: 30}, // 記事なし・新コーナー
	}
	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "テック", Articles: []model.Article{{DedupKey: "u1", URL: "u1", Title: "t", Description: "b"}}},
			{CornerTitle: "オープニング", Articles: []model.Article{}},
		},
	}

	got, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tech := got.Corners[0]
	if tech.ID != "tech" {
		t.Errorf("Corners[0].ID: got %q, want tech", tech.ID)
	}
	if tech.AppearanceCount != 3 {
		t.Errorf("Corners[0].AppearanceCount: got %d, want 3 (past 2 + current)", tech.AppearanceCount)
	}
	if tech.LastEpisodeNumber != 3 {
		t.Errorf("Corners[0].LastEpisodeNumber: got %d, want 3", tech.LastEpisodeNumber)
	}
	opening := got.Corners[1]
	if opening.AppearanceCount != 1 {
		t.Errorf("Corners[1].AppearanceCount: got %d, want 1 (new corner)", opening.AppearanceCount)
	}
	if opening.LastEpisodeNumber != 0 {
		t.Errorf("Corners[1].LastEpisodeNumber: got %d, want 0 (new corner)", opening.LastEpisodeNumber)
	}
}

func TestLLMRundowner_Run_PassesCornerAppearanceToSelector(t *testing.T) {
	ms := &appearanceCapturingSelector{result: sel.SelectResult{SelectedIDs: []string{"u1"}, SelectionReason: "理由"}}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := rundown.NewLLMRundowner(ms, &mockSummarizer{}, mfd, nil)
	rd.SetCornerAppearances(map[string]cache.CornerAppearance{
		"tech": {Count: 4, LastEpisodeNumber: 5},
	})

	corners := []config.CornerConfig{{ID: "tech", Title: "テック", Content: "内容", LengthSec: 60}}
	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "テック", Articles: []model.Article{{DedupKey: "u1", URL: "u1", Title: "t", Description: "b"}}},
		},
	}

	if _, err := rd.Run(context.Background(), corners, articles, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ms.capturedCount) != 1 || ms.capturedCount[0] != 5 {
		t.Errorf("selector should receive appearanceCount 5 (4+1), got %v", ms.capturedCount)
	}
	if len(ms.capturedLast) != 1 || ms.capturedLast[0] != 5 {
		t.Errorf("selector should receive lastEpisodeNumber 5, got %v", ms.capturedLast)
	}
}

func TestLLMRundowner_Run_EmptyArticles_SkipsSelection(t *testing.T) {
	ms := &mockSelector{result: sel.SelectResult{SelectedIDs: []string{"u1"}, SelectionReason: "理由"}}
	mfd := &mockFlowDesigner{flow: "空コーナーの導入です"}
	rd := newRundowner(ms, &mockSummarizer{}, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "オープニング", Articles: []model.Article{}},
		},
	}
	corners := []config.CornerConfig{defaultCorner("オープニング")}

	got, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms.called {
		t.Error("Selector should not be called for corner with no articles")
	}
	if len(got.Corners) != 1 {
		t.Fatalf("len(Corners): got %d, want 1", len(got.Corners))
	}
	c := got.Corners[0]
	if c.Title != "オープニング" {
		t.Errorf("Title: got %q, want %q", c.Title, "オープニング")
	}
	if len(c.Articles) != 0 {
		t.Errorf("Articles should be empty, got %d", len(c.Articles))
	}
}

func TestLLMRundowner_Run_NoArticleCorner_FlowNotEmpty(t *testing.T) {
	ms := &mockSelector{}
	mfd := &mockFlowDesigner{flow: "番組全体の導入です"}
	rd := newRundowner(ms, &mockSummarizer{}, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "オープニング", Articles: []model.Article{}},
		},
	}
	corners := []config.CornerConfig{defaultCorner("オープニング")}

	got, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := got.Corners[0]
	if c.Flow == "" {
		t.Error("Flow should not be empty after phase 2 flow design")
	}
	if c.Flow != "番組全体の導入です" {
		t.Errorf("Flow: got %q, want %q", c.Flow, "番組全体の導入です")
	}
}

func TestLLMRundowner_Run_SelectsAndSummarizes(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{
			SelectedIDs:     []string{"https://example.com/1"},
			SelectionReason: "AIチップ記事が最も関連性高い",
		},
	}
	msum := &mockSummarizer{
		byURL: map[string]model.Summary{
			"https://example.com/1": {
				URL:    "https://example.com/1",
				Points: []string{"ポイント1"},
			},
		},
	}
	mfd := &mockFlowDesigner{flow: "記事1を紹介する"}
	rd := newRundowner(ms, msum, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "テックニュース",
				Articles: []model.Article{
					{DedupKey: "https://example.com/1", URL: "https://example.com/1", Title: "記事1", Description: "本文1"},
					{DedupKey: "https://example.com/2", URL: "https://example.com/2", Title: "記事2", Description: "本文2"},
				},
			},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テックニュース")}

	got, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Corners) != 1 {
		t.Fatalf("len(Corners): got %d, want 1", len(got.Corners))
	}
	c := got.Corners[0]
	if c.Title != "テックニュース" {
		t.Errorf("Title: got %q, want %q", c.Title, "テックニュース")
	}
	if c.SelectionReason != "AIチップ記事が最も関連性高い" {
		t.Errorf("SelectionReason: got %q, want %q", c.SelectionReason, "AIチップ記事が最も関連性高い")
	}
	if c.Flow != "記事1を紹介する" {
		t.Errorf("Flow: got %q, want %q", c.Flow, "記事1を紹介する")
	}
	if len(c.Articles) != 1 {
		t.Fatalf("len(Articles): got %d, want 1", len(c.Articles))
	}
	a := c.Articles[0]
	if a.URL != "https://example.com/1" {
		t.Errorf("URL: got %q, want %q", a.URL, "https://example.com/1")
	}
	if a.Title != "記事1" {
		t.Errorf("Title: got %q, want %q", a.Title, "記事1")
	}
	// Description is the feed-derived text, transferred as-is to RundownArticle
	if a.Description != "本文1" {
		t.Errorf("Description: got %q, want %q", a.Description, "本文1")
	}
	if len(a.Points) != 1 || a.Points[0] != "ポイント1" {
		t.Errorf("Points: got %v, want [ポイント1]", a.Points)
	}
}

func TestLLMRundowner_Run_DescriptionPassedToRundownArticle(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedIDs: []string{"https://example.com/1"}, SelectionReason: "理由"},
	}
	msum := &mockSummarizer{}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, msum, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "テック",
				Articles: []model.Article{
					{DedupKey: "https://example.com/1", URL: "https://example.com/1", Title: "記事1", Description: "フィードテキスト"},
				},
			},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テック")}

	got, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Corners) != 1 || len(got.Corners[0].Articles) != 1 {
		t.Fatalf("unexpected structure: %+v", got)
	}
	a := got.Corners[0].Articles[0]
	if a.Description != "フィードテキスト" {
		t.Errorf("Description: got %q, want %q", a.Description, "フィードテキスト")
	}
	if a.Body != "" {
		t.Errorf("Body should be empty for feed article, got %q", a.Body)
	}
	// summarizer should receive Description via Text()
	if text, ok := msum.receivedTexts["https://example.com/1"]; !ok || text != "フィードテキスト" {
		t.Errorf("summarizer should receive feed description via Text(), got %q", text)
	}
}

func TestLLMRundowner_Run_SelectorError(t *testing.T) {
	ms := &mockSelector{err: errors.New("LLM error")}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, &mockSummarizer{}, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "テック", Articles: []model.Article{article("u1")}},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テック")}

	_, err := rd.Run(context.Background(), corners, articles, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMRundowner_Run_SummarizerError(t *testing.T) {
	ms := &mockSelector{result: sel.SelectResult{SelectedIDs: []string{"u1"}, SelectionReason: "理由"}}
	msum := &mockSummarizer{err: errors.New("sum error")}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, msum, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "テック", Articles: []model.Article{article("u1")}},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テック")}

	_, err := rd.Run(context.Background(), corners, articles, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMRundowner_Run_PreservesCornerOrder(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedIDs: []string{"u1"}, SelectionReason: "理由"},
	}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, &mockSummarizer{}, nil, mfd)

	cornerTitles := []string{"オープニング", "テック", "エンディング"}
	var cornerArticles []model.CornerArticles
	for _, title := range cornerTitles {
		cornerArticles = append(cornerArticles, model.CornerArticles{
			CornerTitle: title,
			Articles:    []model.Article{{DedupKey: "u1", URL: "u1", Title: "t", Description: "b"}},
		})
	}

	var corners []config.CornerConfig
	for _, title := range cornerTitles {
		corners = append(corners, defaultCorner(title))
	}

	got, err := rd.Run(context.Background(), corners, model.Articles{Corners: cornerArticles}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Corners) != 3 {
		t.Fatalf("len(Corners): got %d, want 3", len(got.Corners))
	}
	for i, want := range cornerTitles {
		if got.Corners[i].Title != want {
			t.Errorf("Corners[%d].Title: got %q, want %q", i, got.Corners[i].Title, want)
		}
	}
}

func TestLLMRundowner_Run_ArticlesNotNilForEmptyCorner(t *testing.T) {
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(&mockSelector{}, &mockSummarizer{}, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "オープニング", Articles: []model.Article{}},
		},
	}
	corners := []config.CornerConfig{defaultCorner("オープニング")}

	got, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Corners[0].Articles == nil {
		t.Error("Articles must not be nil (JSON should marshal as [] not null)")
	}
}

func TestLLMRundowner_Run_ExcludedDedupKeysFilteredBeforeSelect(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedIDs: []string{"https://example.com/2"}, SelectionReason: "理由"},
	}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, &mockSummarizer{}, []string{"https://example.com/1"}, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "テック",
				Articles: []model.Article{
					{DedupKey: "https://example.com/1", URL: "https://example.com/1", Title: "除外記事", Description: "body"},
					{DedupKey: "https://example.com/2", URL: "https://example.com/2", Title: "対象記事", Description: "body"},
				},
			},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テック")}

	_, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ms.receivedArticles) != 1 {
		t.Fatalf("selector should receive 1 article after exclusion, got %d", len(ms.receivedArticles))
	}
	if ms.receivedArticles[0].URL != "https://example.com/2" {
		t.Errorf("selector should receive non-excluded article URL, got %q", ms.receivedArticles[0].URL)
	}
}

func TestLLMRundowner_Run_AllArticlesExcluded_EmptyCorner(t *testing.T) {
	ms := &mockSelector{}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, &mockSummarizer{}, []string{"https://example.com/1", "https://example.com/2"}, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "テック",
				Articles: []model.Article{
					{DedupKey: "https://example.com/1", URL: "https://example.com/1", Title: "除外1", Description: "body"},
					{DedupKey: "https://example.com/2", URL: "https://example.com/2", Title: "除外2", Description: "body"},
				},
			},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テック")}

	got, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms.called {
		t.Error("selector should not be called when all articles are excluded")
	}
	if len(got.Corners) != 1 {
		t.Fatalf("should have 1 corner, got %d", len(got.Corners))
	}
	if len(got.Corners[0].Articles) != 0 {
		t.Errorf("corner should have no articles when all excluded, got %d", len(got.Corners[0].Articles))
	}
}

func TestLLMRundowner_WithLogger_ExcludedArticlesLogFormat(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewTextHandler(&buf, slog.LevelInfo)
	logger := slog.New(handler)

	ms := &mockSelector{
		result: sel.SelectResult{SelectedIDs: []string{"https://example.com/2"}, SelectionReason: "理由"},
	}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, &mockSummarizer{}, []string{"https://example.com/1"}, mfd, rundown.WithLogger(logger))

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "今日のテックニュース",
				Articles: []model.Article{
					{DedupKey: "https://example.com/1", URL: "https://example.com/1", Title: "除外記事", Description: "body"},
					{DedupKey: "https://example.com/2", URL: "https://example.com/2", Title: "対象記事", Description: "body"},
				},
			},
		},
	}
	corners := []config.CornerConfig{defaultCorner("今日のテックニュース")}

	_, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "rundown: excluded past articles") {
		t.Errorf("log output should contain 'rundown: excluded past articles', got: %q", got)
	}
	if !strings.Contains(got, "corner=今日のテックニュース") {
		t.Errorf("log output should contain 'corner=今日のテックニュース', got: %q", got)
	}
	if !strings.Contains(got, "count=1") {
		t.Errorf("log output should contain 'count=1', got: %q", got)
	}
}

func TestLLMRundowner_Run_FlowDesignerCalledForAllCorners(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedIDs: []string{"u1"}, SelectionReason: "理由"},
	}
	mfd := &mockFlowDesigner{flow: "generated flow"}
	rd := newRundowner(ms, &mockSummarizer{}, nil, mfd)

	cornerTitles := []string{"オープニング", "テックニュース", "エンディング"}
	cornerArticles := []model.CornerArticles{
		{CornerTitle: "オープニング", Articles: []model.Article{}},
		{CornerTitle: "テックニュース", Articles: []model.Article{{DedupKey: "u1", URL: "u1", Title: "t", Description: "b"}}},
		{CornerTitle: "エンディング", Articles: []model.Article{}},
	}
	var corners []config.CornerConfig
	for _, title := range cornerTitles {
		corners = append(corners, defaultCorner(title))
	}

	got, err := rd.Run(context.Background(), corners, model.Articles{Corners: cornerArticles}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mfd.capturedCorners) != 3 {
		t.Errorf("FlowDesigner should be called for all 3 corners, got %d calls", len(mfd.capturedCorners))
	}
	for _, c := range got.Corners {
		if c.Flow != "generated flow" {
			t.Errorf("corner %q: Flow should be 'generated flow', got %q", c.Title, c.Flow)
		}
	}
}

func TestLLMRundowner_Run_PositionOpeningForFirstCorner(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedIDs: []string{"u1"}, SelectionReason: "理由"},
	}

	var capturedPositions []flow.Position
	customDesigner := &positionCapturingDesigner{positions: &capturedPositions, flow: "flow"}
	rd := rundown.NewLLMRundowner(ms, &mockSummarizer{}, customDesigner, nil)

	corners := []config.CornerConfig{
		defaultCorner("オープニング"),
		defaultCorner("テックニュース"),
		defaultCorner("エンディング"),
	}
	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "オープニング", Articles: []model.Article{}},
			{CornerTitle: "テックニュース", Articles: []model.Article{{DedupKey: "u1", URL: "u1", Title: "t", Description: "b"}}},
			{CornerTitle: "エンディング", Articles: []model.Article{}},
		},
	}

	_, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedPositions) != 3 {
		t.Fatalf("expected 3 position calls, got %d", len(capturedPositions))
	}
	if capturedPositions[0] != flow.PositionOpening {
		t.Errorf("first corner position: got %q, want %q", capturedPositions[0], flow.PositionOpening)
	}
	if capturedPositions[1] != flow.PositionMiddle {
		t.Errorf("middle corner position: got %q, want %q", capturedPositions[1], flow.PositionMiddle)
	}
	if capturedPositions[2] != flow.PositionEnding {
		t.Errorf("last corner position: got %q, want %q", capturedPositions[2], flow.PositionEnding)
	}
}

func TestLLMRundowner_Run_FlowDesignerReceivesRundownContext(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedIDs: []string{"u1"}, SelectionReason: "選別理由"},
	}
	msum := &mockSummarizer{
		byURL: map[string]model.Summary{
			"u1": {URL: "u1", Points: []string{"p1"}},
		},
	}
	casts := []model.RundownCast{{CharacterID: "zundamon", Role: "MC", Type: "regular"}}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, msum, nil, mfd)

	corners := []config.CornerConfig{defaultCorner("テックニュース")}
	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "テックニュース", Articles: []model.Article{{DedupKey: "u1", URL: "u1", Title: "t", Description: "b"}}},
		},
	}

	_, err := rd.Run(context.Background(), corners, articles, casts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mfd.capturedRundown.Casts) != 1 || mfd.capturedRundown.Casts[0].CharacterID != "zundamon" {
		t.Errorf("FlowDesigner should receive rundown with casts, got: %+v", mfd.capturedRundown.Casts)
	}
}

func TestLLMRundowner_Run_FlowDesignerError(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedIDs: []string{"u1"}, SelectionReason: "理由"},
	}
	mfd := &mockFlowDesigner{err: errors.New("flow design error")}
	rd := newRundowner(ms, &mockSummarizer{}, nil, mfd)

	corners := []config.CornerConfig{defaultCorner("テック")}
	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "テック", Articles: []model.Article{{DedupKey: "u1", URL: "u1", Title: "t", Description: "b"}}},
		},
	}

	_, err := rd.Run(context.Background(), corners, articles, nil)
	if err == nil {
		t.Fatal("expected error from FlowDesigner, got nil")
	}
}

func TestLLMRundowner_Run_CastsSetInRundown(t *testing.T) {
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(&mockSelector{}, &mockSummarizer{}, nil, mfd)

	casts := []model.RundownCast{
		{CharacterID: "metan", Role: "アシスタント", Type: "regular"},
	}
	corners := []config.CornerConfig{defaultCorner("オープニング")}
	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "オープニング", Articles: []model.Article{}},
		},
	}

	got, err := rd.Run(context.Background(), corners, articles, casts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Casts) != 1 || got.Casts[0].CharacterID != "metan" {
		t.Errorf("Rundown.Casts should contain passed casts, got: %+v", got.Casts)
	}
}

func TestLLMRundowner_Run_PropagatesSourceAuthorPublished(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{
			SelectedIDs:     []string{"https://example.com/meta/1"},
			SelectionReason: "出典テスト",
		},
	}
	msum := &mockSummarizer{
		byURL: map[string]model.Summary{
			"https://example.com/meta/1": {
				URL:    "https://example.com/meta/1",
				Points: []string{"ポイント1"},
			},
		},
	}
	mfd := &mockFlowDesigner{flow: "フロー"}
	rd := newRundowner(ms, msum, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "テック",
				Articles: []model.Article{
					{
						DedupKey:    "https://example.com/meta/1",
						URL:         "https://example.com/meta/1",
						Title:       "メタ記事",
						Description: "本文",
						Source:      "テストフィード",
						Author:      "山田太郎",
						Published:   "2026-06-06T19:00:00+09:00",
					},
				},
			},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テック")}

	got, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Corners) != 1 || len(got.Corners[0].Articles) != 1 {
		t.Fatalf("unexpected structure: %+v", got)
	}
	a := got.Corners[0].Articles[0]
	if a.Source != "テストフィード" {
		t.Errorf("Source: got %q, want %q", a.Source, "テストフィード")
	}
	if a.Author != "山田太郎" {
		t.Errorf("Author: got %q, want %q", a.Author, "山田太郎")
	}
	if a.Published != "2026-06-06T19:00:00+09:00" {
		t.Errorf("Published: got %q, want %q", a.Published, "2026-06-06T19:00:00+09:00")
	}
}
