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
	result sel.SelectResult
	err    error
	called bool
}

func (m *mockSelector) Select(_ context.Context, _ config.CornerConfig, _ []model.Article) (sel.SelectResult, error) {
	m.called = true
	return m.result, m.err
}

type mockSummarizer struct {
	byURL map[string]model.Summary
	err   error
}

func (m *mockSummarizer) Summarize(_ context.Context, a model.Article) (model.Summary, error) {
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
	return config.CornerConfig{Title: title, Content: "内容", TargetDurationSec: 60}
}

func article(url string) model.Article {
	return model.Article{URL: url, Title: "記事: " + url, Body: "本文"}
}

// --- tests ---

func TestLLMRundowner_Run_EmptyArticles_SkipsSelection(t *testing.T) {
	ms := &mockSelector{result: sel.SelectResult{SelectedURLs: []string{"u1"}, Flow: "flow"}}
	rd := rundown.NewLLMRundowner(ms, &mockSummarizer{})

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
	rd := rundown.NewLLMRundowner(ms, msum)

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
	rd := rundown.NewLLMRundowner(ms, &mockSummarizer{})

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
	rd := rundown.NewLLMRundowner(ms, msum)

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
	rd := rundown.NewLLMRundowner(ms, &mockSummarizer{})

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
	rd := rundown.NewLLMRundowner(&mockSelector{}, &mockSummarizer{})

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
