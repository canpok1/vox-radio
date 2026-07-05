//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// slackTestChannel は fixture の slack-spec.yaml と合わせたテスト用チャンネル ID。
const slackTestChannel = "C0123456789"

const slackTestFileID = "F-E2E-001"

// fakeSlack は slack-go が呼ぶ Slack Web API を模倣するモックサーバー。
// アップロードは3段階（files.getUploadURLExternal → upload先POST → files.completeUploadExternal）、
// その後 files.info（スレッド ts 解決）と chat.postMessage（スレッド返信）が呼ばれる。
type fakeSlack struct {
	server *httptest.Server

	mu       sync.Mutex
	received []string // 受信した API メソッド名の履歴
}

func newFakeSlack() *fakeSlack {
	f := &fakeSlack{}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	return f
}

func (f *fakeSlack) URL() string { return f.server.URL }

func (f *fakeSlack) Close() { f.server.Close() }

// Received は指定 API メソッドの受信回数を返す。
func (f *fakeSlack) Received(method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, m := range f.received {
		if m == method {
			n++
		}
	}
	return n
}

// ClearReceived は受信履歴をクリアする（再開シナリオの検証用）。
func (f *fakeSlack) ClearReceived() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.received = nil
}

func (f *fakeSlack) handle(w http.ResponseWriter, r *http.Request) {
	method := strings.TrimPrefix(r.URL.Path, "/")

	f.mu.Lock()
	f.received = append(f.received, method)
	f.mu.Unlock()

	switch method {
	case "auth.test":
		// プリフライトのスコープ検証用。付与済みスコープをヘッダで返す。
		w.Header().Set("X-OAuth-Scopes", "chat:write,files:write,files:read")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"ok":true,"url":"https://example.slack.com/","team":"T","user":"bot","team_id":"T1","user_id":"U1","bot_id":"B1"}`)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	switch method {
	case "files.getUploadURLExternal":
		_, _ = fmt.Fprintf(w, `{"ok":true,"upload_url":"%s/upload","file_id":"%s"}`, f.server.URL, slackTestFileID)
	case "upload":
		_, _ = w.Write([]byte(`OK`))
	case "files.completeUploadExternal":
		_, _ = fmt.Fprintf(w, `{"ok":true,"files":[{"id":"%s","title":"e2e"}]}`, slackTestFileID)
	case "files.info":
		_, _ = fmt.Fprintf(w, `{"ok":true,"file":{"id":"%s","shares":{"public":{"%s":[{"ts":"1700000000.000100"}]}}},"comments":[]}`,
			slackTestFileID, slackTestChannel)
	case "chat.postMessage":
		_, _ = fmt.Fprintf(w, `{"ok":true,"channel":"%s","ts":"1700000000.000200"}`, slackTestChannel)
	default:
		http.Error(w, fmt.Sprintf(`{"ok":false,"error":"fake slack: unknown method %s"}`, method), http.StatusNotFound)
	}
}
