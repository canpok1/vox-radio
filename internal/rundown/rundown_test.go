package rundown_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

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

type mockSummarizer struct {
	byURL          map[string]model.Summary
	err            error
	receivedBodies map[string]string
}

type mockFetcher struct {
	bodyByURL  map[string]string
	err        error
	called     bool
	fetchedURL string
}

func (m *mockFetcher) FetchFullText(_ context.Context, url string) (string, error) {
	m.called = true
	m.fetchedURL = url
	if m.err != nil {
		return "", m.err
	}
	if body, ok := m.bodyByURL[url]; ok {
		return body, nil
	}
	return "default full text", nil
}

var _ rundown.ArticleFetcher = (*mockFetcher)(nil)

func (m *mockSummarizer) Summarize(_ context.Context, a model.Article) (model.Summary, error) {
	if m.receivedBodies == nil {
		m.receivedBodies = make(map[string]string)
	}
	m.receivedBodies[a.URL] = a.Body
	if m.err != nil {
		return model.Summary{}, m.err
	}
	if s, ok := m.byURL[a.URL]; ok {
		return s, nil
	}
	return model.Summary{URL: a.URL, Summary: "default", Points: []string{"p1"}}, nil
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
	return model.Article{URL: url, Title: "記事: " + url, Body: "本文"}
}

func newRundowner(ms *mockSelector, msum *mockSummarizer, mf *mockFetcher, excludedURLs []string, mfd *mockFlowDesigner, opts ...rundown.Option) *rundown.LLMRundowner {
	var fetcher rundown.ArticleFetcher
	if mf != nil {
		fetcher = mf
	}
	return rundown.NewLLMRundowner(ms, msum, mfd, fetcher, excludedURLs, opts...)
}

// --- tests ---

func TestLLMRundowner_Run_EmptyArticles_SkipsSelection(t *testing.T) {
	ms := &mockSelector{result: sel.SelectResult{SelectedURLs: []string{"u1"}, SelectionReason: "理由"}}
	mfd := &mockFlowDesigner{flow: "空コーナーの導入です"}
	rd := newRundowner(ms, &mockSummarizer{}, nil, nil, mfd)

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
	rd := newRundowner(ms, &mockSummarizer{}, nil, nil, mfd)

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
			SelectedURLs:    []string{"https://example.com/1"},
			SelectionReason: "AIチップ記事が最も関連性高い",
		},
	}
	msum := &mockSummarizer{
		byURL: map[string]model.Summary{
			"https://example.com/1": {
				URL:     "https://example.com/1",
				Summary: "要約テキスト",
				Points:  []string{"ポイント1"},
			},
		},
	}
	mfd := &mockFlowDesigner{flow: "記事1を紹介する"}
	rd := newRundowner(ms, msum, nil, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "テックニュース",
				Articles: []model.Article{
					{URL: "https://example.com/1", Title: "記事1", Body: "本文1"},
					{URL: "https://example.com/2", Title: "記事2", Body: "本文2"},
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
	if a.Summary != "要約テキスト" {
		t.Errorf("Summary: got %q, want %q", a.Summary, "要約テキスト")
	}
	if len(a.Points) != 1 || a.Points[0] != "ポイント1" {
		t.Errorf("Points: got %v, want [ポイント1]", a.Points)
	}
}

func TestLLMRundowner_Run_SelectorError(t *testing.T) {
	ms := &mockSelector{err: errors.New("LLM error")}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, &mockSummarizer{}, nil, nil, mfd)

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
	ms := &mockSelector{result: sel.SelectResult{SelectedURLs: []string{"u1"}, SelectionReason: "理由"}}
	msum := &mockSummarizer{err: errors.New("sum error")}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, msum, nil, nil, mfd)

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
		result: sel.SelectResult{SelectedURLs: []string{"u1"}, SelectionReason: "理由"},
	}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, &mockSummarizer{}, nil, nil, mfd)

	cornerTitles := []string{"オープニング", "テック", "エンディング"}
	var cornerArticles []model.CornerArticles
	for _, t := range cornerTitles {
		cornerArticles = append(cornerArticles, model.CornerArticles{
			CornerTitle: t,
			Articles:    []model.Article{{URL: "u1", Title: "t", Body: "b"}},
		})
	}

	var corners []config.CornerConfig
	for _, t := range cornerTitles {
		corners = append(corners, defaultCorner(t))
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
	rd := newRundowner(&mockSelector{}, &mockSummarizer{}, nil, nil, mfd)

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

func TestLLMRundowner_Run_ExcludedURLsFilteredBeforeSelect(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedURLs: []string{"https://example.com/2"}, SelectionReason: "理由"},
	}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, &mockSummarizer{}, nil, []string{"https://example.com/1"}, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "テック",
				Articles: []model.Article{
					{URL: "https://example.com/1", Title: "除外記事", Body: "body"},
					{URL: "https://example.com/2", Title: "対象記事", Body: "body"},
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
	rd := newRundowner(ms, &mockSummarizer{}, nil, []string{"https://example.com/1", "https://example.com/2"}, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "テック",
				Articles: []model.Article{
					{URL: "https://example.com/1", Title: "除外1", Body: "body"},
					{URL: "https://example.com/2", Title: "除外2", Body: "body"},
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

func TestLLMRundowner_Run_FetcherSuccess_BodyReplaced(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedURLs: []string{"https://example.com/1"}, SelectionReason: "理由"},
	}
	msum := &mockSummarizer{}
	mf := &mockFetcher{
		bodyByURL: map[string]string{
			"https://example.com/1": "全文テキスト",
		},
	}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, msum, mf, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "テック",
				Articles: []model.Article{
					{URL: "https://example.com/1", Title: "記事1", Body: "スニペット"},
				},
			},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テック")}

	_, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mf.called {
		t.Error("fetcher should have been called")
	}
	if body, ok := msum.receivedBodies["https://example.com/1"]; !ok || body != "全文テキスト" {
		t.Errorf("summarizer should receive full text body, got %q", body)
	}
}

func TestLLMRundowner_Run_FetcherFailure_FallbackToFeedBody(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedURLs: []string{"https://example.com/1"}, SelectionReason: "理由"},
	}
	msum := &mockSummarizer{}
	mf := &mockFetcher{err: errors.New("connection refused")}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, msum, mf, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "テック",
				Articles: []model.Article{
					{URL: "https://example.com/1", Title: "記事1", Body: "フィードスニペット"},
				},
			},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テック")}

	_, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("should not return error on fetch failure (fallback to feed body): %v", err)
	}
	if body, ok := msum.receivedBodies["https://example.com/1"]; !ok || body != "フィードスニペット" {
		t.Errorf("summarizer should receive feed body as fallback, got %q", body)
	}
}

func TestLLMRundowner_Run_FetcherNil_SkipsFetch(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedURLs: []string{"https://example.com/1"}, SelectionReason: "理由"},
	}
	msum := &mockSummarizer{}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, msum, nil, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "テック",
				Articles: []model.Article{
					{URL: "https://example.com/1", Title: "記事1", Body: "フィードボディ"},
				},
			},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テック")}

	_, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body, ok := msum.receivedBodies["https://example.com/1"]; !ok || body != "フィードボディ" {
		t.Errorf("summarizer should receive original feed body when fetcher is nil, got %q", body)
	}
}

func TestLLMRundowner_WithLogger_ExcludedArticlesLogFormat(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewTextHandler(&buf, slog.LevelInfo)
	logger := slog.New(handler)

	ms := &mockSelector{
		result: sel.SelectResult{SelectedURLs: []string{"https://example.com/2"}, SelectionReason: "理由"},
	}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, &mockSummarizer{}, nil, []string{"https://example.com/1"}, mfd, rundown.WithLogger(logger))

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "今日のテックニュース",
				Articles: []model.Article{
					{URL: "https://example.com/1", Title: "除外記事", Body: "body"},
					{URL: "https://example.com/2", Title: "対象記事", Body: "body"},
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

func TestLLMRundowner_Run_FetcherEmptyBody_FallbackToFeedBody(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedURLs: []string{"https://example.com/1"}, SelectionReason: "理由"},
	}
	msum := &mockSummarizer{}
	mf := &mockFetcher{
		bodyByURL: map[string]string{
			"https://example.com/1": "",
		},
	}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, msum, mf, nil, mfd)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{
				CornerTitle: "テック",
				Articles: []model.Article{
					{URL: "https://example.com/1", Title: "記事1", Body: "フィードスニペット"},
				},
			},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テック")}

	_, err := rd.Run(context.Background(), corners, articles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body, ok := msum.receivedBodies["https://example.com/1"]; !ok || body != "フィードスニペット" {
		t.Errorf("summarizer should receive feed body when full text is empty, got %q", body)
	}
}

func TestLLMRundowner_Run_FlowDesignerCalledForAllCorners(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedURLs: []string{"u1"}, SelectionReason: "理由"},
	}
	mfd := &mockFlowDesigner{flow: "generated flow"}
	rd := newRundowner(ms, &mockSummarizer{}, nil, nil, mfd)

	cornerTitles := []string{"オープニング", "テックニュース", "エンディング"}
	cornerArticles := []model.CornerArticles{
		{CornerTitle: "オープニング", Articles: []model.Article{}},
		{CornerTitle: "テックニュース", Articles: []model.Article{{URL: "u1", Title: "t", Body: "b"}}},
		{CornerTitle: "エンディング", Articles: []model.Article{}},
	}
	var corners []config.CornerConfig
	for _, t := range cornerTitles {
		corners = append(corners, defaultCorner(t))
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
		result: sel.SelectResult{SelectedURLs: []string{"u1"}, SelectionReason: "理由"},
	}

	var capturedPositions []flow.Position
	customDesigner := &positionCapturingDesigner{positions: &capturedPositions, flow: "flow"}
	rd := rundown.NewLLMRundowner(ms, &mockSummarizer{}, customDesigner, nil, nil)

	corners := []config.CornerConfig{
		defaultCorner("オープニング"),
		defaultCorner("テックニュース"),
		defaultCorner("エンディング"),
	}
	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "オープニング", Articles: []model.Article{}},
			{CornerTitle: "テックニュース", Articles: []model.Article{{URL: "u1", Title: "t", Body: "b"}}},
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
		result: sel.SelectResult{SelectedURLs: []string{"u1"}, SelectionReason: "選別理由"},
	}
	msum := &mockSummarizer{
		byURL: map[string]model.Summary{
			"u1": {URL: "u1", Summary: "要約", Points: []string{"p1"}},
		},
	}
	casts := []model.RundownCast{{CharacterID: "zundamon", Role: "MC", Type: "regular"}}
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(ms, msum, nil, nil, mfd)

	corners := []config.CornerConfig{defaultCorner("テックニュース")}
	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "テックニュース", Articles: []model.Article{{URL: "u1", Title: "t", Body: "b"}}},
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
		result: sel.SelectResult{SelectedURLs: []string{"u1"}, SelectionReason: "理由"},
	}
	mfd := &mockFlowDesigner{err: errors.New("flow design error")}
	rd := newRundowner(ms, &mockSummarizer{}, nil, nil, mfd)

	corners := []config.CornerConfig{defaultCorner("テック")}
	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "テック", Articles: []model.Article{{URL: "u1", Title: "t", Body: "b"}}},
		},
	}

	_, err := rd.Run(context.Background(), corners, articles, nil)
	if err == nil {
		t.Fatal("expected error from FlowDesigner, got nil")
	}
}

func TestLLMRundowner_Run_CastsSetInRundown(t *testing.T) {
	mfd := &mockFlowDesigner{flow: "flow"}
	rd := newRundowner(&mockSelector{}, &mockSummarizer{}, nil, nil, mfd)

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
