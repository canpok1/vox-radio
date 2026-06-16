//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
)

// fixturesDir はこのテストファイルから見た fixtures ディレクトリの絶対パス。
var fixturesDir string

func init() {
	_, file, _, _ := runtime.Caller(0)
	fixturesDir = filepath.Join(filepath.Dir(file), "fixtures")
}

// scenarioState はシナリオ1本分の状態（作業ディレクトリ・環境変数・実行結果・モックサーバー）。
type scenarioState struct {
	workDir string
	env     map[string]string

	exitCode int
	stdout   string
	stderr   string

	llm      *fakeLLM
	voicevox *fakeVoicevox
	feed     *fakeFeed
	slack    *fakeSlack
}

func (s *scenarioState) close() {
	if s.llm != nil {
		s.llm.Close()
	}
	if s.voicevox != nil {
		s.voicevox.Close()
	}
	if s.feed != nil {
		s.feed.Close()
	}
	if s.slack != nil {
		s.slack.Close()
	}
}

// expand はステップ引数・fixture 内のプレースホルダをシナリオ固有の値へ展開する。
// 未起動のモックサーバーの URL は到達不能なダミー値になる（設定の strict パースは通る）。
func (s *scenarioState) expand(text string) string {
	llmURL := "http://e2e-llm.invalid"
	if s.llm != nil {
		llmURL = s.llm.URL()
	}
	feedURL := "http://e2e-feed.invalid"
	if s.feed != nil {
		feedURL = s.feed.URL()
	}
	voicevoxURL := "http://e2e-voicevox.invalid"
	if s.voicevox != nil {
		voicevoxURL = s.voicevox.URL()
	}
	slackURL := "http://e2e-slack.invalid"
	if s.slack != nil {
		slackURL = s.slack.URL()
	}
	r := strings.NewReplacer(
		"{{LLM_URL}}", llmURL,
		"{{FEED_URL}}", feedURL,
		"{{VOICEVOX_URL}}", voicevoxURL,
		"{{SLACK_URL}}", slackURL,
		"{{WORKDIR}}", s.workDir,
	)
	return r.Replace(text)
}

func (s *scenarioState) path(rel string) string {
	rel = s.expand(rel)
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(s.workDir, rel)
}

// --- Given: モックサーバー起動 ---

func (s *scenarioState) startLLM() error {
	s.llm = newFakeLLM()
	s.env["TEST_LLM_API_KEY"] = "e2e-test-key"
	return nil
}

func (s *scenarioState) startVoicevox() error {
	s.voicevox = newFakeVoicevox()
	s.env["VOX_RADIO_VOICEVOX_URL"] = s.voicevox.URL()
	return nil
}

func (s *scenarioState) startFeed() error {
	s.feed = newFakeFeed()
	return nil
}

func (s *scenarioState) startSlack() error {
	s.slack = newFakeSlack()
	s.env["VOX_RADIO_SLACK_API_URL"] = s.slack.URL()
	s.env["TEST_SLACK_BOT_TOKEN"] = "xoxb-e2e-test-token"
	s.env["E2E_SLACK_CHANNEL"] = slackTestChannel
	return nil
}

// --- Given: ファイル配置 ---

// placeFixtures はテスト用設定一式（vox-radio.yaml / episode-spec.yaml / feed-spec.yaml / slack-spec.yaml）を
// プレースホルダ展開しつつ作業ディレクトリへ配置する。
func (s *scenarioState) placeFixtures() error {
	files := map[string]string{
		"vox-radio.yaml.tmpl":    "vox-radio.yaml",
		"episode-spec.yaml.tmpl": "episode-spec.yaml",
		"feed-spec.yaml":         "feed-spec.yaml",
		"slack-spec.yaml":        "slack-spec.yaml",
	}
	for src, dst := range files {
		b, err := os.ReadFile(filepath.Join(fixturesDir, src))
		if err != nil {
			return fmt.Errorf("read fixture %s: %w", src, err)
		}
		if err := os.WriteFile(filepath.Join(s.workDir, dst), []byte(s.expand(string(b))), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
	}
	return nil
}

func (s *scenarioState) placeFixtureFileAs(src, dst string) error {
	b, err := os.ReadFile(filepath.Join(fixturesDir, src))
	if err != nil {
		return fmt.Errorf("read fixture %s: %w", src, err)
	}
	target := s.path(dst)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(s.expand(string(b))), 0o644)
}

func (s *scenarioState) createFileWithContent(rel string, content *godog.DocString) error {
	target := s.path(rel)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(s.expand(content.Content)), 0o644)
}

func (s *scenarioState) setEnv(name, value string) error {
	s.env[name] = s.expand(value)
	return nil
}

// --- When: コマンド実行 ---

func (s *scenarioState) runCommand(command string) error {
	command = s.expand(command)
	args := strings.Fields(command)
	if len(args) == 0 {
		return fmt.Errorf("empty command")
	}
	if args[0] != "vox-radio" {
		return fmt.Errorf("command must start with %q: %s", "vox-radio", command)
	}

	cmd := exec.Command(binaryPath, args[1:]...)
	cmd.Dir = s.workDir
	cmd.Env = s.buildEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	s.stdout = stdout.String()
	s.stderr = stderr.String()
	s.exitCode = 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			s.exitCode = exitErr.ExitCode()
		} else {
			return fmt.Errorf("run %s: %w", command, err)
		}
	}
	return nil
}

// buildEnv は親プロセスの環境変数から vox-radio に影響するものを取り除き、
// シナリオで明示設定した変数を加えた環境を返す（devcontainer 等の環境差異を排除する）。
func (s *scenarioState) buildEnv() []string {
	drop := func(key string) bool {
		switch key {
		case "VOX_RADIO_VOICEVOX_URL", "VOX_RADIO_SLACK_API_URL", "GEMINI_API_KEY", "SLACK_BOT_TOKEN":
			return true
		}
		return false
	}
	env := make([]string, 0, len(os.Environ())+len(s.env))
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if !drop(key) {
			env = append(env, kv)
		}
	}
	for k, v := range s.env {
		env = append(env, k+"="+v)
	}
	return env
}

// --- Then: 検証 ---

func (s *scenarioState) assertExitCode(want int) error {
	if s.exitCode != want {
		return fmt.Errorf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", s.exitCode, want, s.stdout, s.stderr)
	}
	return nil
}

func (s *scenarioState) assertExitCodeNonZero() error {
	if s.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want non-zero\nstdout:\n%s\nstderr:\n%s", s.stdout, s.stderr)
	}
	return nil
}

func (s *scenarioState) assertFileExists(rel string) error {
	if _, err := os.Stat(s.path(rel)); err != nil {
		return fmt.Errorf("file %s should exist: %w", rel, err)
	}
	return nil
}

func (s *scenarioState) assertFileNotExists(rel string) error {
	_, err := os.Stat(s.path(rel))
	if err == nil {
		return fmt.Errorf("file %s should not exist", rel)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", rel, err)
	}
	return nil
}

func (s *scenarioState) assertFileNotEmpty(rel string) error {
	info, err := os.Stat(s.path(rel))
	if err != nil {
		return fmt.Errorf("file %s should exist: %w", rel, err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("file %s is empty", rel)
	}
	return nil
}

func (s *scenarioState) assertFileLineCount(rel string, want int) error {
	b, err := os.ReadFile(s.path(rel))
	if err != nil {
		return err
	}
	got := len(strings.Split(strings.TrimRight(string(b), "\n"), "\n"))
	if got != want {
		return fmt.Errorf("file %s has %d lines, want %d", rel, got, want)
	}
	return nil
}

func (s *scenarioState) assertFileContains(rel, substr string) error {
	b, err := os.ReadFile(s.path(rel))
	if err != nil {
		return err
	}
	substr = s.expand(substr)
	if !strings.Contains(string(b), substr) {
		return fmt.Errorf("file %s does not contain %q\ncontent:\n%s", rel, substr, truncate(string(b), 2000))
	}
	return nil
}

func (s *scenarioState) assertStdoutContains(substr string) error {
	substr = s.expand(substr)
	if !strings.Contains(s.stdout, substr) {
		return fmt.Errorf("stdout does not contain %q\nstdout:\n%s\nstderr:\n%s", substr, s.stdout, s.stderr)
	}
	return nil
}

func (s *scenarioState) assertStdoutNotContains(substr string) error {
	substr = s.expand(substr)
	if strings.Contains(s.stdout, substr) {
		return fmt.Errorf("stdout should not contain %q\nstdout:\n%s", substr, s.stdout)
	}
	return nil
}

func (s *scenarioState) assertStderrContains(substr string) error {
	substr = s.expand(substr)
	if !strings.Contains(s.stderr, substr) {
		return fmt.Errorf("stderr does not contain %q\nstdout:\n%s\nstderr:\n%s", substr, s.stdout, s.stderr)
	}
	return nil
}

// jsonLookup はドット区切りパス（配列は数値インデックス）で JSON ドキュメントの値を引く。
func jsonLookup(doc any, path string) (any, error) {
	cur := doc
	for _, part := range strings.Split(path, ".") {
		switch v := cur.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("key %q not found", part)
			}
			cur = val
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("array index %q is not a number", part)
			}
			if idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("array index %d out of range (len=%d)", idx, len(v))
			}
			cur = v[idx]
		default:
			return nil, fmt.Errorf("cannot descend into %T at %q", cur, part)
		}
	}
	return cur, nil
}

func (s *scenarioState) lookupJSON(rel, path string) (any, error) {
	b, err := os.ReadFile(s.path(rel))
	if err != nil {
		return nil, err
	}
	var doc any
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", rel, err)
	}
	val, err := jsonLookup(doc, path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	return val, nil
}

func (s *scenarioState) assertJSONString(rel, path, want string) error {
	val, err := s.lookupJSON(rel, path)
	if err != nil {
		return err
	}
	got, ok := val.(string)
	if !ok {
		return fmt.Errorf("%s %s is %T, want string", rel, path, val)
	}
	want = s.expand(want)
	if got != want {
		return fmt.Errorf("%s %s = %q, want %q", rel, path, got, want)
	}
	return nil
}

func (s *scenarioState) assertJSONNumber(rel, path string, want int) error {
	val, err := s.lookupJSON(rel, path)
	if err != nil {
		return err
	}
	got, ok := val.(float64)
	if !ok {
		return fmt.Errorf("%s %s is %T, want number", rel, path, val)
	}
	if int(got) != want {
		return fmt.Errorf("%s %s = %v, want %d", rel, path, got, want)
	}
	return nil
}

func (s *scenarioState) assertJSONNonEmptyString(rel, path string) error {
	val, err := s.lookupJSON(rel, path)
	if err != nil {
		return err
	}
	got, ok := val.(string)
	if !ok {
		return fmt.Errorf("%s %s is %T, want string", rel, path, val)
	}
	if got == "" {
		return fmt.Errorf("%s %s is empty", rel, path)
	}
	return nil
}

func (s *scenarioState) assertJSONTrue(rel, path string) error {
	val, err := s.lookupJSON(rel, path)
	if err != nil {
		return err
	}
	got, ok := val.(bool)
	if !ok {
		return fmt.Errorf("%s %s is %T, want bool", rel, path, val)
	}
	if !got {
		return fmt.Errorf("%s %s = false, want true", rel, path)
	}
	return nil
}

func (s *scenarioState) assertJSONArrayLen(rel, path string, want int) error {
	val, err := s.lookupJSON(rel, path)
	if err != nil {
		return err
	}
	arr, ok := val.([]any)
	if !ok {
		return fmt.Errorf("%s %s is %T, want array", rel, path, val)
	}
	if len(arr) != want {
		return fmt.Errorf("%s %s has %d elements, want %d", rel, path, len(arr), want)
	}
	return nil
}

func (s *scenarioState) assertJSONArrayLenAtLeast(rel, path string, want int) error {
	val, err := s.lookupJSON(rel, path)
	if err != nil {
		return err
	}
	arr, ok := val.([]any)
	if !ok {
		return fmt.Errorf("%s %s is %T, want array", rel, path, val)
	}
	if len(arr) < want {
		return fmt.Errorf("%s %s has %d elements, want >= %d", rel, path, len(arr), want)
	}
	return nil
}

func (s *scenarioState) assertSlackReceived(method string) error {
	if s.slack == nil {
		return fmt.Errorf("fake slack server is not running")
	}
	if s.slack.Received(method) == 0 {
		return fmt.Errorf("fake slack did not receive %q", method)
	}
	return nil
}

func (s *scenarioState) assertSlackNotReceived(method string) error {
	if s.slack == nil {
		return fmt.Errorf("fake slack server is not running")
	}
	if n := s.slack.Received(method); n > 0 {
		return fmt.Errorf("fake slack received %q %d times, want 0", method, n)
	}
	return nil
}

func (s *scenarioState) clearSlackReceived() error {
	if s.slack == nil {
		return fmt.Errorf("fake slack server is not running")
	}
	s.slack.ClearReceived()
	return nil
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "...(truncated)"
}

// InitializeScenario は godog のステップ定義を登録する。
func InitializeScenario(sc *godog.ScenarioContext) {
	s := &scenarioState{env: map[string]string{}}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		dir, err := os.MkdirTemp("", "vox-radio-scenario-*")
		if err != nil {
			return ctx, err
		}
		s.workDir = dir
		return ctx, nil
	})
	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		s.close()
		if s.workDir != "" {
			_ = os.RemoveAll(s.workDir)
		}
		return ctx, nil
	})

	// Given
	sc.Step(`^モックLLMサーバーが起動している$`, s.startLLM)
	sc.Step(`^モックVOICEVOXサーバーが起動している$`, s.startVoicevox)
	sc.Step(`^モックフィードサーバーが起動している$`, s.startFeed)
	sc.Step(`^モックSlackサーバーが起動している$`, s.startSlack)
	sc.Step(`^テスト用設定一式を配置する$`, s.placeFixtures)
	sc.Step(`^fixture "([^"]*)" をファイル "([^"]*)" として配置する$`, s.placeFixtureFileAs)
	sc.Step(`^ファイル "([^"]*)" を以下の内容で作成する:$`, s.createFileWithContent)
	sc.Step(`^環境変数 "([^"]*)" に "([^"]*)" を設定する$`, s.setEnv)
	sc.Step(`^モックSlackサーバーの受信記録をクリアする$`, s.clearSlackReceived)

	// When
	sc.Step(`^"([^"]*)" を実行する$`, s.runCommand)

	// Then
	sc.Step(`^終了コードは (\d+) である$`, s.assertExitCode)
	sc.Step(`^終了コードは 0 以外である$`, s.assertExitCodeNonZero)
	sc.Step(`^ファイル "([^"]*)" が存在する$`, s.assertFileExists)
	sc.Step(`^ファイル "([^"]*)" が存在しない$`, s.assertFileNotExists)
	sc.Step(`^ファイル "([^"]*)" のサイズは 0 より大きい$`, s.assertFileNotEmpty)
	sc.Step(`^ファイル "([^"]*)" の行数は (\d+) である$`, s.assertFileLineCount)
	sc.Step(`^ファイル "([^"]*)" に "([^"]*)" を含む$`, s.assertFileContains)
	sc.Step(`^標準出力に "([^"]*)" を含む$`, s.assertStdoutContains)
	sc.Step(`^標準出力に "([^"]*)" を含まない$`, s.assertStdoutNotContains)
	sc.Step(`^標準エラーに "([^"]*)" を含む$`, s.assertStderrContains)
	sc.Step(`^JSONファイル "([^"]*)" のキー "([^"]*)" は文字列 "([^"]*)" である$`, s.assertJSONString)
	sc.Step(`^JSONファイル "([^"]*)" のキー "([^"]*)" は数値 (\d+) である$`, s.assertJSONNumber)
	sc.Step(`^JSONファイル "([^"]*)" のキー "([^"]*)" は空でない文字列である$`, s.assertJSONNonEmptyString)
	sc.Step(`^JSONファイル "([^"]*)" のキー "([^"]*)" は真である$`, s.assertJSONTrue)
	sc.Step(`^JSONファイル "([^"]*)" の配列 "([^"]*)" の要素数は (\d+) である$`, s.assertJSONArrayLen)
	sc.Step(`^JSONファイル "([^"]*)" の配列 "([^"]*)" の要素数は (\d+) 以上である$`, s.assertJSONArrayLenAtLeast)
	sc.Step(`^モックSlackサーバーは "([^"]*)" を受信した$`, s.assertSlackReceived)
	sc.Step(`^モックSlackサーバーは "([^"]*)" を受信していない$`, s.assertSlackNotReceived)
}
