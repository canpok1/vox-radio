package collect_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

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
		_, err := c.Run(context.Background(), cfg, nil)
		if err == nil {
			t.Error("expected error for HTTP 404, got nil")
		}
	})

	t.Run("article returns error on non-200", func(t *testing.T) {
		cfg := config.FeedsConfig{
			Articles: []string{server.URL + "/article.html"},
		}
		c := collect.New(server.Client())
		_, err := c.Run(context.Background(), cfg, nil)
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
	result, err := c.Run(context.Background(), cfg, nil)
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
	result, err := c.Run(context.Background(), cfg, nil)
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
	result, err := c.Run(context.Background(), cfg, nil)
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

func TestCollector_Run_AtomAppliesMaxItems(t *testing.T) {
	atomData := loadTestdata(t, "feed_atom.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write(atomData)
	}))
	defer server.Close()

	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{
			{URL: server.URL + "/feed_atom.xml", MaxItems: 2},
		},
	}

	c := collect.New(server.Client())
	result, err := c.Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("articles count: got %d, want 2", len(result))
	}
}

func TestCollector_Run_AtomParsesFields(t *testing.T) {
	atomData := loadTestdata(t, "feed_atom.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write(atomData)
	}))
	defer server.Close()

	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{
			{URL: server.URL + "/feed_atom.xml", MaxItems: 1},
		},
	}

	c := collect.New(server.Client())
	result, err := c.Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("no articles returned")
	}
	art := result[0]
	if art.Title != "Atomフィード記事1" {
		t.Errorf("title: got %q, want %q", art.Title, "Atomフィード記事1")
	}
	if art.URL != "https://example.com/atom/1" {
		t.Errorf("url: got %q, want %q", art.URL, "https://example.com/atom/1")
	}
	if art.Body == "" {
		t.Error("body must not be empty")
	}
}

func TestCollector_Run_WarnOnEmptyFeed(t *testing.T) {
	emptyFeed := `<?xml version="1.0"?><rss version="2.0"><channel><title>Empty</title></channel></rss>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(emptyFeed))
	}))
	defer server.Close()

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{
			{URL: server.URL + "/empty.xml", MaxItems: 5},
		},
	}

	c := collect.New(server.Client(), collect.WithLogger(logger))
	result, err := c.Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("expected no error for empty feed, got: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("articles count: got %d, want 0", len(result))
	}
	if !strings.Contains(buf.String(), "WARN") {
		t.Errorf("expected WARN log for empty feed, got: %q", buf.String())
	}
}

func TestCollector_Run_ExcludesURLsAndFillsFromLater(t *testing.T) {
	// feed.xml has 3 articles: article/1, article/2, article/3
	// Exclude article/1 by DedupKey; with maxItems=2 should return article/2 and article/3
	rssData := loadTestdata(t, "feed.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server.Close()

	feedURL := server.URL + "/feed.xml"
	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{
			{URL: feedURL, MaxItems: 2},
		},
	}
	// article/1 has no GUID, so material = normalizeContent(title, body)
	excluded := map[string]struct{}{
		collect.FeedDedupKey(feedURL, "", "AIチップの最新動向", "新型AIチップが発表された。従来比2倍の性能を実現する。"): {},
	}

	c := collect.New(server.Client())
	result, err := c.Run(context.Background(), cfg, excluded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("articles count: got %d, want 2", len(result))
	}
	if result[0].URL != "https://example.com/article/2" {
		t.Errorf("result[0].URL: got %q, want %q", result[0].URL, "https://example.com/article/2")
	}
	if result[1].URL != "https://example.com/article/3" {
		t.Errorf("result[1].URL: got %q, want %q", result[1].URL, "https://example.com/article/3")
	}
}

func TestCollector_Run_WarnWhenInsufficientNonExcludedArticles(t *testing.T) {
	// feed.xml has 3 articles; exclude article/1 and article/2; want 3 but only 1 available
	rssData := loadTestdata(t, "feed.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server.Close()

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	feedURL := server.URL + "/feed.xml"
	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{
			{URL: feedURL, MaxItems: 3},
		},
	}
	excluded := map[string]struct{}{
		collect.FeedDedupKey(feedURL, "", "AIチップの最新動向", "新型AIチップが発表された。従来比2倍の性能を実現する。"):  {},
		collect.FeedDedupKey(feedURL, "", "オープンソース活動の活発化", "オープンソースプロジェクトへの参加者が増加している。"): {},
	}

	c := collect.New(server.Client(), collect.WithLogger(logger))
	result, err := c.Run(context.Background(), cfg, excluded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("articles count: got %d, want 1 (only non-excluded)", len(result))
	}
	if result[0].URL != "https://example.com/article/3" {
		t.Errorf("result[0].URL: got %q, want %q", result[0].URL, "https://example.com/article/3")
	}
	if !strings.Contains(buf.String(), "WARN") {
		t.Errorf("expected WARN log for insufficient articles, got: %q", buf.String())
	}
}

func TestCollector_Run_UnlimitedWithExcludedDedupKeys(t *testing.T) {
	// maxItems=0 means unlimited; exclude article/1 by DedupKey; should return article/2 and article/3
	rssData := loadTestdata(t, "feed.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server.Close()

	feedURL := server.URL + "/feed.xml"
	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{
			{URL: feedURL, MaxItems: 0},
		},
	}
	excluded := map[string]struct{}{
		collect.FeedDedupKey(feedURL, "", "AIチップの最新動向", "新型AIチップが発表された。従来比2倍の性能を実現する。"): {},
	}

	c := collect.New(server.Client())
	result, err := c.Run(context.Background(), cfg, excluded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("articles count: got %d, want 2 (all non-excluded)", len(result))
	}
}

func TestCollector_RunAll_SkipsSourcelessCorners(t *testing.T) {
	corners := []config.CornerConfig{
		{Title: "ソースなし", Content: "挨拶のみ"},
		{Title: "ソースあり", Content: "ニュース", Source: &config.SourceConfig{}},
	}

	c := collect.New(nil)
	result, err := c.RunAll(context.Background(), corners, nil)
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
	result, err := c.RunAll(context.Background(), corners, nil)
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

func TestCollector_RunAll_ExcludesDedupKeysViaFeed(t *testing.T) {
	// feed.xml has 3 articles; exclude article/1 by DedupKey; maxItems=2 should return article/2 and article/3
	rssData := loadTestdata(t, "feed.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server.Close()

	feedURL := server.URL + "/feed.xml"
	corners := []config.CornerConfig{
		{
			Title: "ニュース",
			Source: &config.SourceConfig{
				Feeds: []config.FeedEntry{{URL: feedURL, MaxItems: 2}},
			},
		},
	}
	excludedDedupKeys := []string{
		collect.FeedDedupKey(feedURL, "", "AIチップの最新動向", "新型AIチップが発表された。従来比2倍の性能を実現する。"),
	}

	c := collect.New(server.Client())
	result, err := c.RunAll(context.Background(), corners, excludedDedupKeys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Corners) != 1 {
		t.Fatalf("corners count: got %d, want 1", len(result.Corners))
	}
	articles := result.Corners[0].Articles
	if len(articles) != 2 {
		t.Errorf("articles count: got %d, want 2", len(articles))
	}
	if len(articles) > 0 && articles[0].URL != "https://example.com/article/2" {
		t.Errorf("articles[0].URL: got %q, want %q", articles[0].URL, "https://example.com/article/2")
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
	result, err := c.Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("articles count: got %d, want 2 (1 RSS + 1 individual)", len(result))
	}
}

func TestCollector_RunAll_LogsStartAndComplete(t *testing.T) {
	rssData := loadTestdata(t, "feed.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server.Close()

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	corners := []config.CornerConfig{
		{
			Title: "テックニュース",
			Source: &config.SourceConfig{
				Feeds: []config.FeedEntry{{URL: server.URL + "/feed.xml", MaxItems: 1}},
			},
		},
	}

	c := collect.New(server.Client(), collect.WithLogger(logger))
	_, err := c.RunAll(context.Background(), corners, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "開始") {
		t.Errorf("should log start: %q", logs)
	}
	if !strings.Contains(logs, "完了") {
		t.Errorf("should log complete: %q", logs)
	}
}

func TestCollector_RunAll_LogsPerCornerProgress(t *testing.T) {
	rssData := loadTestdata(t, "feed.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server.Close()

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

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
				Feeds: []config.FeedEntry{{URL: server.URL + "/feed.xml", MaxItems: 1}},
			},
		},
	}

	c := collect.New(server.Client(), collect.WithLogger(logger))
	_, err := c.RunAll(context.Background(), corners, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "1/2") {
		t.Errorf("should log per-corner progress (1/2): %q", logs)
	}
	if !strings.Contains(logs, "2/2") {
		t.Errorf("should log per-corner progress (2/2): %q", logs)
	}
}

func TestCollector_FetchFullText_ReturnsBody(t *testing.T) {
	htmlData := loadTestdata(t, "article.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(htmlData)
	}))
	defer server.Close()

	c := collect.New(server.Client())
	body, err := c.FetchFullText(context.Background(), server.URL+"/article.html")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "最初のパラグラフ") {
		t.Errorf("body should contain article text, got: %q", body)
	}
}

func TestCollector_FetchFullText_ReturnsErrorOnHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := collect.New(server.Client())
	_, err := c.FetchFullText(context.Background(), server.URL+"/notfound.html")
	if err == nil {
		t.Error("expected error for HTTP 404, got nil")
	}
}

func TestCollector_Run_RSS_ExtractsSourceAuthorPublished(t *testing.T) {
	rssData := loadTestdata(t, "feed_with_meta.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server.Close()

	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("time.LoadLocation: %v", err)
	}
	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{
			{URL: server.URL + "/feed_with_meta.xml"},
		},
	}

	c := collect.New(server.Client(), collect.WithLocation(loc))
	result, err := c.Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) < 1 {
		t.Fatal("no articles returned")
	}

	// 1件目: dc:creator=山田太郎, pubDate=2026-06-06T10:00:00+00:00
	// Asia/Tokyo に変換すると 2026-06-06T19:00:00+09:00
	art := result[0]
	if art.Source != "メタ情報テストフィード" {
		t.Errorf("Source: got %q, want %q", art.Source, "メタ情報テストフィード")
	}
	if art.Author != "山田太郎" {
		t.Errorf("Author: got %q, want %q", art.Author, "山田太郎")
	}
	if art.Published != "2026-06-06T19:00:00+09:00" {
		t.Errorf("Published: got %q, want %q", art.Published, "2026-06-06T19:00:00+09:00")
	}
}

func TestCollector_Run_RSS_EmailAuthorExcluded(t *testing.T) {
	rssData := loadTestdata(t, "feed_with_meta.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server.Close()

	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{
			{URL: server.URL + "/feed_with_meta.xml"},
		},
	}

	c := collect.New(server.Client())
	result, err := c.Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) < 2 {
		t.Fatalf("need at least 2 articles, got %d", len(result))
	}

	// 2件目: dc:creator=author@example.com → @ を含むため空
	art := result[1]
	if art.Author != "" {
		t.Errorf("email author should be excluded (got %q, want empty)", art.Author)
	}
}

func TestCollector_Run_RSS_PublishedEmptyWhenNil(t *testing.T) {
	rssData := loadTestdata(t, "feed_with_meta.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server.Close()

	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{
			{URL: server.URL + "/feed_with_meta.xml"},
		},
	}

	c := collect.New(server.Client())
	result, err := c.Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) < 3 {
		t.Fatalf("need at least 3 articles, got %d", len(result))
	}

	// 3件目: pubDate なし → Published = ""
	art := result[2]
	if art.Published != "" {
		t.Errorf("Published should be empty when no pubDate, got %q", art.Published)
	}
}

func TestCollector_Run_RSS_SetsDedupKey(t *testing.T) {
	rssData := loadTestdata(t, "feed.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server.Close()

	feedURL := server.URL + "/feed.xml"
	cfg := config.FeedsConfig{
		Feeds: []config.FeedEntry{{URL: feedURL, MaxItems: 1}},
	}

	c := collect.New(server.Client())
	result, err := c.Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("no articles returned")
	}
	art := result[0]
	if art.DedupKey == "" {
		t.Error("DedupKey must not be empty for RSS article")
	}
	// DedupKey は sha256: プレフィックスを持つ
	if len(art.DedupKey) != len("sha256:")+64 {
		t.Errorf("DedupKey has unexpected length: %q", art.DedupKey)
	}
}

func TestCollector_Run_Article_SetsDedupKey(t *testing.T) {
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
	result, err := c.Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("no articles returned")
	}
	art := result[0]
	if art.DedupKey == "" {
		t.Error("DedupKey must not be empty for HTML article")
	}
	if len(art.DedupKey) != len("sha256:")+64 {
		t.Errorf("DedupKey has unexpected length: %q", art.DedupKey)
	}
}

func TestCollector_Run_RSS_NamespaceIsolation(t *testing.T) {
	// 2つのフィードで同じ内容の記事でも DedupKey は異なること（フィード間衝突回避）
	rssData := loadTestdata(t, "feed.xml")
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(rssData)
	}))
	defer server2.Close()

	c := collect.New(server1.Client())
	result1, err := c.Run(context.Background(), config.FeedsConfig{Feeds: []config.FeedEntry{{URL: server1.URL + "/feed.xml", MaxItems: 1}}}, nil)
	if err != nil {
		t.Fatalf("feed1: %v", err)
	}
	result2, err := c.Run(context.Background(), config.FeedsConfig{Feeds: []config.FeedEntry{{URL: server2.URL + "/feed.xml", MaxItems: 1}}}, nil)
	if err != nil {
		t.Fatalf("feed2: %v", err)
	}
	if len(result1) == 0 || len(result2) == 0 {
		t.Fatal("no articles from one of the feeds")
	}
	if result1[0].DedupKey == result2[0].DedupKey {
		t.Errorf("same content from different feed URLs must have different DedupKeys, both got %q", result1[0].DedupKey)
	}
}
