//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
)

// llmStepRoute は fake LLM のルーティング1件分。プロンプト冒頭の見出し行で
// ステップを判別する（internal/cli/prompts/*.md の先頭行は各ステップで一意）。
type llmStepRoute struct {
	step   string
	header string
}

// llmRoutes はプロンプト見出し → ステップ名の対応表。
// 見出しが変わると fake が 500 を返して e2e が落ちるため、プロンプト変更の検知網を兼ねる。
var llmRoutes = []llmStepRoute{
	{step: "summarize", header: "# [0] 記事要約プロンプト"},
	{step: "select", header: "# [A] 記事選別プロンプト"},
	{step: "flow", header: "# [C] flow設計プロンプト"},
	{step: "write", header: "# [B] 台本生成プロンプト"},
	{step: "direct", header: "# [C] 演出プロンプト"},
	{step: "proofread", header: "# [C] 発音校正プロンプト"},
	{step: "summary", header: "# [D] 番組要約プロンプト"},
	{step: "corner_summary", header: "# [D] コーナー要約プロンプト"},
}

var urlInPromptRe = regexp.MustCompile(`"url":\s*"([^"]+)"`)

// fakeLLM は OpenAI 互換 /chat/completions を模倣するモックサーバー。
// レスポンスは各ステップの JSON Schema（クライアント側で検証される）に適合する固定値を返す。
type fakeLLM struct {
	server *httptest.Server

	mu    sync.Mutex
	calls []string // 受信したステップ名の履歴
}

func newFakeLLM() *fakeLLM {
	f := &fakeLLM{}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	return f
}

func (f *fakeLLM) URL() string { return f.server.URL }

func (f *fakeLLM) Close() { f.server.Close() }

// CallCount は指定ステップの受信回数を返す。
func (f *fakeLLM) CallCount(step string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if c == step {
			n++
		}
	}
	return n
}

type llmRequest struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func (f *fakeLLM) handle(w http.ResponseWriter, r *http.Request) {
	var req llmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("fake llm: decode request: %v", err), http.StatusBadRequest)
		return
	}
	if len(req.Messages) == 0 {
		http.Error(w, "fake llm: no messages", http.StatusBadRequest)
		return
	}

	// リトライ修復時もメッセージは追記されるだけなので、先頭メッセージで判別する。
	prompt := req.Messages[0].Content
	step := ""
	for _, route := range llmRoutes {
		if strings.HasPrefix(prompt, route.header) {
			step = route.step
			break
		}
	}
	if step == "" {
		firstLine, _, _ := strings.Cut(prompt, "\n")
		http.Error(w, fmt.Sprintf("fake llm: unknown prompt header %q (新ステップが追加された場合は e2e/fake_llm.go の llmRoutes を更新してください)", firstLine), http.StatusInternalServerError)
		return
	}

	f.mu.Lock()
	f.calls = append(f.calls, step)
	f.mu.Unlock()

	content, err := cannedResponse(step, prompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("fake llm: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp := map[string]any{
		"choices": []map[string]any{
			{"message": map[string]any{"content": content}},
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// cannedResponse はステップごとの固定レスポンス（choices[0].message.content の文字列）を返す。
// 各 JSON は対応するステップの JSON Schema（required / additionalProperties:false）に適合させること。
func cannedResponse(step, prompt string) (string, error) {
	switch step {
	case "summarize":
		return `{"summary":"記事の一行要約です。","points":["要点1","要点2","要点3"]}`, nil
	case "select":
		// 候補記事の URL は fake feed の動的ポートを含むため、プロンプト本文から抽出して全件選択する。
		matches := urlInPromptRe.FindAllStringSubmatch(prompt, -1)
		if len(matches) == 0 {
			return "", fmt.Errorf("select prompt contains no candidate urls")
		}
		urls := make([]string, 0, len(matches))
		seen := map[string]struct{}{}
		for _, m := range matches {
			if _, ok := seen[m[1]]; ok {
				continue
			}
			seen[m[1]] = struct{}{}
			urls = append(urls, m[1])
		}
		b, _ := json.Marshal(map[string]any{
			"selected_urls":    urls,
			"selection_reason": "テスト用に候補記事を全件選択",
		})
		return string(b), nil
	case "flow":
		return `{"flow":"導入の挨拶から始め、記事の内容を紹介し、感想を述べて締める。"}`, nil
	case "write":
		return `{"lines":[` +
			`{"speaker_role":"zundamon","text":"こんにちは、ずんだもんなのだ。"},` +
			`{"speaker_role":"metan","text":"四国めたんですわ。今日も始めましょう。"}` +
			`]}`, nil
	case "direct":
		return `{"insertions":[]}`, nil
	case "proofread":
		return `{"corrections":[]}`, nil
	case "summary":
		return `{"summary":"今回の番組のまとめです。テスト用の固定要約を返しています。",` +
			`"episode_title":"テスト回のサブタイトル",` +
			`"conversation_notes":[{"category":"雑談","character_ids":["zundamon"],"note":"テスト用の会話メモ"}]}`, nil
	case "corner_summary":
		return `{"summary":"コーナー内容のまとめです。","points":["コーナー要点1"]}`, nil
	}
	return "", fmt.Errorf("no canned response for step %q", step)
}
