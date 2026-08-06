//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
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
	{step: "analyze", header: "# [D] 番組分析プロンプト"},
	{step: "retro", header: "# [D] 振り返り（retro）プロンプト"},
}

var idInPromptRe = regexp.MustCompile(`"id":\s*"([^"]+)"`)

// cornersInPromptRe は演出プロンプトに差し込まれたコーナー別セリフ列（{{corners}}）の
// JSON 配列を切り出す。
var cornersInPromptRe = regexp.MustCompile(`(?s)## コーナー別セリフ列\s*` + "```" + `json\s*(\[.*?\])\s*` + "```")

// directLineEntries は演出ステップの line_voices / line_conversions を、プロンプトに
// 含まれる実際のセリフ列から組み立てる。実プロンプトは全セリフ分の出力を要求するため、
// 固定件数を返すモックだと本番で起きない「一部の行だけ無指定」を検証してしまう。
func directLineEntries(prompt string) (voices, conversions []map[string]any, err error) {
	m := cornersInPromptRe.FindStringSubmatch(prompt)
	if m == nil {
		return nil, nil, fmt.Errorf("direct prompt contains no corners JSON")
	}

	var corners []struct {
		Lines []struct {
			Text string `json:"text"`
		} `json:"lines"`
	}
	if err := json.Unmarshal([]byte(m[1]), &corners); err != nil {
		return nil, nil, fmt.Errorf("unmarshal corners in direct prompt: %w", err)
	}

	voices = make([]map[string]any, 0)
	conversions = make([]map[string]any, 0)
	for ci, c := range corners {
		for li, line := range c.Lines {
			voices = append(voices, map[string]any{
				"corner_index": ci,
				"line_index":   li,
				"intonation":   "標準",
			})
			conversions = append(conversions, map[string]any{
				"corner_index": ci,
				"line_index":   li,
				"text":         line.Text,
			})
		}
	}
	return voices, conversions, nil
}

// fakeLLM は OpenAI 互換 /chat/completions を模倣するモックサーバー。
// レスポンスは各ステップの JSON Schema（クライアント側で検証される）に適合する固定値を返す。
type fakeLLM struct {
	server *httptest.Server
}

func newFakeLLM() *fakeLLM {
	f := &fakeLLM{}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	return f
}

func (f *fakeLLM) URL() string { return f.server.URL }

func (f *fakeLLM) Close() { f.server.Close() }

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
		return `{"points":["要点1","要点2","要点3"]}`, nil
	case "select":
		// 候補記事の ID は DedupKey（sha256:...）のためプロンプト本文から抽出して全件選択する。
		matches := idInPromptRe.FindAllStringSubmatch(prompt, -1)
		if len(matches) == 0 {
			return "", fmt.Errorf("select prompt contains no candidate ids")
		}
		ids := make([]string, 0, len(matches))
		seen := map[string]struct{}{}
		for _, m := range matches {
			if _, ok := seen[m[1]]; ok {
				continue
			}
			seen[m[1]] = struct{}{}
			ids = append(ids, m[1])
		}
		b, _ := json.Marshal(map[string]any{
			"selected_ids":     ids,
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
		voices, conversions, err := directLineEntries(prompt)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(map[string]any{
			"insertions":       []any{},
			"pause_insertions": []any{},
			"line_conversions": conversions,
			"line_voices":      voices,
		})
		return string(b), nil
	case "proofread":
		return `{"corrections":[]}`, nil
	case "summary":
		return `{"summary":"今回の番組のまとめです。テスト用の固定要約を返しています。",` +
			`"episode_title":"テスト回のサブタイトル",` +
			`"conversation_notes":[{"category":"雑談","character_ids":["zundamon"],"note":"テスト用の会話メモ"}]}`, nil
	case "corner_summary":
		return `{"summary":"コーナー内容のまとめです。","points":["コーナー要点1"]}`, nil
	case "analyze":
		return `{"findings":[` +
			`{"aspect":"掛け合い","severity":"low","detail":"テスト用の固定分析結果です。","evidence":"ずんだもん「テストなのだ」"}` +
			`],"patterns":[` +
			`{"aspect":"つかみ","detail":"テスト用の固定パターンです。"}` +
			`]}`, nil
	case "retro":
		return `{"problems":[` +
			`{"id":"","problem":"テスト用の固定問題です。","action":"テスト用の固定施策です。","first_seen_episode":1,"last_seen_episode":1}` +
			`],"recurrences":[]}`, nil
	}
	return "", fmt.Errorf("no canned response for step %q", step)
}
