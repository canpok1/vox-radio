package rundown_test

import (
	"context"
	"errors"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/rundown"
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

// --- helpers ---

func defaultCorner(title string) config.CornerConfig {
	return config.CornerConfig{Title: title, Content: "内容", LengthSec: 60}
}

func article(url string) model.Article {
	return model.Article{URL: url, Title: "記事: " + url, Body: "本文"}
}

// --- tests ---

func TestLLMRundowner_Run_EmptyArticles_SkipsSelection(t *testing.T) {
	ms := &mockSelector{result: sel.SelectResult{SelectedURLs: []string{"u1"}, Flow: "flow"}}
	rd := rundown.NewLLMRundowner(ms, &mockSummarizer{}, nil)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "オープニング", Articles: []model.Article{}},
		},
	}
	corners := []config.CornerConfig{defaultCorner("オープニング")}

	got, err := rd.Run(context.Background(), corners, articles)
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
	if c.Flow != "" {
		t.Errorf("Flow should be empty for no-article corner, got %q", c.Flow)
	}
	if len(c.Articles) != 0 {
		t.Errorf("Articles should be empty, got %d", len(c.Articles))
	}
}

func TestLLMRundowner_Run_SelectsAndSummarizes(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{
			SelectedURLs: []string{"https://example.com/1"},
			Flow:         "記事1を紹介する",
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
	rd := rundown.NewLLMRundowner(ms, msum, nil)

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

	got, err := rd.Run(context.Background(), corners, articles)
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
	rd := rundown.NewLLMRundowner(ms, &mockSummarizer{}, nil)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "テック", Articles: []model.Article{article("u1")}},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テック")}

	_, err := rd.Run(context.Background(), corners, articles)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMRundowner_Run_SummarizerError(t *testing.T) {
	ms := &mockSelector{result: sel.SelectResult{SelectedURLs: []string{"u1"}, Flow: "f"}}
	msum := &mockSummarizer{err: errors.New("sum error")}
	rd := rundown.NewLLMRundowner(ms, msum, nil)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "テック", Articles: []model.Article{article("u1")}},
		},
	}
	corners := []config.CornerConfig{defaultCorner("テック")}

	_, err := rd.Run(context.Background(), corners, articles)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLLMRundowner_Run_PreservesCornerOrder(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedURLs: []string{"u1"}, Flow: "flow"},
	}
	rd := rundown.NewLLMRundowner(ms, &mockSummarizer{}, nil)

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

	got, err := rd.Run(context.Background(), corners, model.Articles{Corners: cornerArticles})
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
	rd := rundown.NewLLMRundowner(&mockSelector{}, &mockSummarizer{}, nil)

	articles := model.Articles{
		Corners: []model.CornerArticles{
			{CornerTitle: "オープニング", Articles: []model.Article{}},
		},
	}
	corners := []config.CornerConfig{defaultCorner("オープニング")}

	got, err := rd.Run(context.Background(), corners, articles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Corners[0].Articles == nil {
		t.Error("Articles must not be nil (JSON should marshal as [] not null)")
	}
}

func TestLLMRundowner_Run_ExcludedURLsFilteredBeforeSelect(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedURLs: []string{"https://example.com/2"}, Flow: "flow"},
	}
	rd := rundown.NewLLMRundowner(ms, &mockSummarizer{}, nil)
	rd.SetExcludedURLs([]string{"https://example.com/1"})

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

	_, err := rd.Run(context.Background(), corners, articles)
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
	rd := rundown.NewLLMRundowner(ms, &mockSummarizer{}, nil)
	rd.SetExcludedURLs([]string{"https://example.com/1", "https://example.com/2"})

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

	got, err := rd.Run(context.Background(), corners, articles)
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
		result: sel.SelectResult{SelectedURLs: []string{"https://example.com/1"}, Flow: "flow"},
	}
	msum := &mockSummarizer{}
	mf := &mockFetcher{
		bodyByURL: map[string]string{
			"https://example.com/1": "全文テキスト",
		},
	}
	rd := rundown.NewLLMRundowner(ms, msum, mf)

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

	_, err := rd.Run(context.Background(), corners, articles)
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
		result: sel.SelectResult{SelectedURLs: []string{"https://example.com/1"}, Flow: "flow"},
	}
	msum := &mockSummarizer{}
	mf := &mockFetcher{err: errors.New("connection refused")}
	rd := rundown.NewLLMRundowner(ms, msum, mf)

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

	_, err := rd.Run(context.Background(), corners, articles)
	if err != nil {
		t.Fatalf("should not return error on fetch failure (fallback to feed body): %v", err)
	}
	if body, ok := msum.receivedBodies["https://example.com/1"]; !ok || body != "フィードスニペット" {
		t.Errorf("summarizer should receive feed body as fallback, got %q", body)
	}
}

func TestLLMRundowner_Run_FetcherNil_SkipsFetch(t *testing.T) {
	ms := &mockSelector{
		result: sel.SelectResult{SelectedURLs: []string{"https://example.com/1"}, Flow: "flow"},
	}
	msum := &mockSummarizer{}
	rd := rundown.NewLLMRundowner(ms, msum, nil)

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

	_, err := rd.Run(context.Background(), corners, articles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body, ok := msum.receivedBodies["https://example.com/1"]; !ok || body != "フィードボディ" {
		t.Errorf("summarizer should receive original feed body when fetcher is nil, got %q", body)
	}
}
