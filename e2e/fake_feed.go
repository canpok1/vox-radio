//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

// fakeFeed は RSS フィードと記事ページを配信するモックサーバー。
// /feed.xml は記事2件の RSS 2.0 を返し、各 item の link は同サーバーの /articles/N を指す。
type fakeFeed struct {
	server *httptest.Server
}

func newFakeFeed() *fakeFeed {
	f := &fakeFeed{}
	mux := http.NewServeMux()
	mux.HandleFunc("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
		base := f.server.URL
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>E2Eテストフィード</title>
    <link>%[1]s</link>
    <description>e2e テスト用のフィードです</description>
    <item>
      <title>テスト記事1</title>
      <link>%[1]s/articles/1</link>
      <description>テスト記事1の概要です。新しい技術が発表されました。</description>
      <pubDate>Mon, 01 Jun 2026 09:00:00 +0900</pubDate>
    </item>
    <item>
      <title>テスト記事2</title>
      <link>%[1]s/articles/2</link>
      <description>テスト記事2の概要です。便利なツールが公開されました。</description>
      <pubDate>Tue, 02 Jun 2026 09:00:00 +0900</pubDate>
    </item>
  </channel>
</rss>`, base)
	})
	mux.HandleFunc("/articles/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="ja">
<head><title>テスト記事</title></head>
<body>
<article>
<h1>テスト記事の本文</h1>
<p>これは e2e テスト用の記事本文です。記事の詳細な内容がここに書かれています。</p>
<p>本文の続きです。要約の材料になる文章がもう少し続きます。</p>
</article>
</body>
</html>`)
	})
	f.server = httptest.NewServer(mux)
	return f
}

func (f *fakeFeed) URL() string { return f.server.URL }

func (f *fakeFeed) Close() { f.server.Close() }
