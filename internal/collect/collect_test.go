package collect_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/collect"
	"github.com/canpok1/vox-radio/internal/config"
)

func TestCollector_Run_ErrorOnHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	t.Run("feed returns error on non-200", func(t *testing.T) {
		cfg := config.FeedsConfig{
			Feeds: []config.FeedEntry{
				{URL: server.URL + "/feed.xml", MaxItems: 1},
			},
		}
		c := collect.New(server.Client())
		_, err := c.Run(context.Background(), cfg)
		if err == nil {
			t.Error("expected error for HTTP 404, got nil")
		}
	})

	t.Run("article returns error on non-200", func(t *testing.T) {
		cfg := config.FeedsConfig{
			Articles: []string{server.URL + "/article.html"},
		}
		c := collect.New(server.Client())
		_, err := c.Run(context.Background(), cfg)
		if err == nil {
			t.Error("expected error for HTTP 404, got nil")
		}
	})
}

func loadTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("loadTestdata: %v", err)
	}
	return data
}

func TestCollector_Run_RSSAppliesMaxItems(t *testing.T) {
	rssData := loadTestdata(t, "feed.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server.Close()

	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{
			{URL: server.URL + "/feed.xml", MaxItems: 2},
		},
	}

	c := collect.New(server.Client())
	result, err := c.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("articles count: got %d, want 2", len(result))
	}
}

func TestCollector_Run_RSSParsesFields(t *testing.T) {
	rssData := loadTestdata(t, "feed.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server.Close()

	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{
			{URL: server.URL + "/feed.xml", MaxItems: 1},
		},
	}

	c := collect.New(server.Client())
	result, err := c.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("no articles returned")
	}
	art := result[0]
	if art.Title != "AIチップの最新動向" {
		t.Errorf("title: got %q, want %q", art.Title, "AIチップの最新動向")
	}
	if art.URL != "https://example.com/article/1" {
		t.Errorf("url: got %q, want %q", art.URL, "https://example.com/article/1")
	}
	if art.Body == "" {
		t.Error("body must not be empty")
	}
}

func TestCollector_Run_ArticleExtractsBody(t *testing.T) {
	htmlData := loadTestdata(t, "article.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(htmlData)
	}))
	defer server.Close()

	cfg := config.FeedsConfig{
		Articles: []string{server.URL + "/article.html"},
	}

	c := collect.New(server.Client())
	result, err := c.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("articles count: got %d, want 1", len(result))
	}
	art := result[0]
	if art.Title != "テスト記事タイトル" {
		t.Errorf("title: got %q, want %q", art.Title, "テスト記事タイトル")
	}
	if !strings.Contains(art.Body, "最初のパラグラフ") {
		t.Errorf("body should contain article text, got: %q", art.Body)
	}
	if strings.Contains(art.Body, "ナビゲーション") {
		t.Errorf("body should not contain nav text, got: %q", art.Body)
	}
}

func TestCollector_RunAll_SkipsSourcelessCorners(t *testing.T) {
	corners := []config.CornerConfig{
		{Title: "ソースなし", Content: "挨拶のみ"},
		{Title: "ソースあり", Content: "ニュース", Source: &config.SourceConfig{}},
	}

	c := collect.New(nil)
	result, err := c.RunAll(context.Background(), corners)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Corners) != 1 {
		t.Fatalf("corners count: got %d, want 1 (source-less skipped)", len(result.Corners))
	}
	if result.Corners[0].CornerTitle != "ソースあり" {
		t.Errorf("corner title: got %q, want %q", result.Corners[0].CornerTitle, "ソースあり")
	}
}

func TestCollector_RunAll_CollectsPerCorner(t *testing.T) {
	rssData := loadTestdata(t, "feed.xml")
	htmlData := loadTestdata(t, "article.html")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".xml") {
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write(rssData)
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(htmlData)
		}
	}))
	defer server.Close()

	corners := []config.CornerConfig{
		{
			Title: "コーナーA",
			Source: &config.SourceConfig{
				Feeds: []config.FeedEntry{{URL: server.URL + "/feed.xml", MaxItems: 1}},
			},
		},
		{
			Title: "コーナーB",
			Source: &config.SourceConfig{
				Articles: []string{server.URL + "/article.html"},
			},
		},
	}

	c := collect.New(server.Client())
	result, err := c.RunAll(context.Background(), corners)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Corners) != 2 {
		t.Fatalf("corners count: got %d, want 2", len(result.Corners))
	}
	if result.Corners[0].CornerTitle != "コーナーA" {
		t.Errorf("corners[0].title: got %q, want %q", result.Corners[0].CornerTitle, "コーナーA")
	}
	if len(result.Corners[0].Articles) != 1 {
		t.Errorf("corners[0].articles count: got %d, want 1", len(result.Corners[0].Articles))
	}
	if result.Corners[1].CornerTitle != "コーナーB" {
		t.Errorf("corners[1].title: got %q, want %q", result.Corners[1].CornerTitle, "コーナーB")
	}
	if len(result.Corners[1].Articles) != 1 {
		t.Errorf("corners[1].articles count: got %d, want 1", len(result.Corners[1].Articles))
	}
}

func TestCollector_Run_CombinesFeedsAndArticles(t *testing.T) {
	rssData := loadTestdata(t, "feed.xml")
	htmlData := loadTestdata(t, "article.html")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".xml") {
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write(rssData)
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(htmlData)
		}
	}))
	defer server.Close()

	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{
			{URL: server.URL + "/feed.xml", MaxItems: 1},
		},
		Articles: []string{server.URL + "/article.html"},
	}

	c := collect.New(server.Client())
	result, err := c.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("articles count: got %d, want 2 (1 RSS + 1 individual)", len(result))
	}
}
