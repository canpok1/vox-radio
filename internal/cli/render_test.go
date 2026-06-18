package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
	"github.com/canpok1/vox-radio/internal/testutil"
)

func writeManifestForRenderTest(t *testing.T, dir string) string {
	t.Helper()
	manifest := map[string]any{
		"title":          "テスト番組",
		"episode_number": 3,
		"episode_title":  "テスト回",
		"summary":        "今回のまとめ",
		"audio_file":     "ep3.mp3",
		"corners": []any{
			map[string]any{
				"id":    "news",
				"title": "ニュースコーナー",
				"articles": []any{
					map[string]any{"title": "記事A", "url": "https://example.com/a"},
					map[string]any{"title": "記事B", "url": ""},
				},
			},
		},
	}
	data, _ := json.Marshal(manifest)
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

func TestRenderCmd_BasicOutput(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeManifestForRenderTest(t, dir)
	tmplPath := testutil.WriteTempFile(t, "test.tmpl", []byte(`{{.Title}} 第{{.EpisodeNumber}}回`))

	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"render", "--manifest", manifestPath, "--template", tmplPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "テスト番組") {
		t.Errorf("output should contain title, got: %q", buf.String())
	}
}

func TestRenderCmd_URLSkip(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeManifestForRenderTest(t, dir)
	tmplPath := testutil.WriteTempFile(t, "test.tmpl",
		[]byte(`{{range .Corners}}{{range .Articles}}{{if .URL}}{{.Title}}{{end}}{{end}}{{end}}`))

	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"render", "--manifest", manifestPath, "--template", tmplPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "記事A") {
		t.Errorf("output should contain 記事A, got: %q", out)
	}
	if strings.Contains(out, "記事B") {
		t.Errorf("output should NOT contain URL-less 記事B, got: %q", out)
	}
}

func TestRenderCmd_OutputFile(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeManifestForRenderTest(t, dir)
	tmplPath := testutil.WriteTempFile(t, "test.tmpl", []byte(`{{.Title}}`))
	outputPath := filepath.Join(dir, "out.txt")

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"render", "--manifest", manifestPath, "--template", tmplPath, "--output", outputPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !strings.Contains(string(data), "テスト番組") {
		t.Errorf("output file should contain title, got: %q", string(data))
	}
}

func TestRenderCmd_MissingManifest_Error(t *testing.T) {
	tmplPath := testutil.WriteTempFile(t, "test.tmpl", []byte(`{{.Title}}`))

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"render", "--template", tmplPath})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error when --manifest is missing")
	}
}

func TestRenderCmd_MissingTemplate_Error(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeManifestForRenderTest(t, dir)

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"render", "--manifest", manifestPath})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error when --template is missing")
	}
}

func TestRenderCmd_TemplateFileNotFound_Error(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeManifestForRenderTest(t, dir)

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"render", "--manifest", manifestPath, "--template", filepath.Join(dir, "nonexistent.tmpl")})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error when template file does not exist")
	}
}

func TestRenderCmd_TemplateParseError_Error(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeManifestForRenderTest(t, dir)
	tmplPath := testutil.WriteTempFile(t, "test.tmpl", []byte(`{{.Title`)) // 構文エラー

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"render", "--manifest", manifestPath, "--template", tmplPath})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for template parse error")
	}
}

func TestRenderCmd_TemplateString_BasicOutput(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeManifestForRenderTest(t, dir)

	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"render", "--manifest", manifestPath, "--template-string", "{{.Title}} 第{{.EpisodeNumber}}回"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "テスト番組") {
		t.Errorf("output should contain title, got: %q", buf.String())
	}
}

func TestRenderCmd_TemplateBothFlags_Error(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeManifestForRenderTest(t, dir)
	tmplPath := testutil.WriteTempFile(t, "test.tmpl", []byte(`{{.Title}}`))

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"render", "--manifest", manifestPath, "--template", tmplPath, "--template-string", "{{.Title}}"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error when both --template and --template-string are specified")
	}
}

func TestRenderCmd_TemplateString_OutputFile(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeManifestForRenderTest(t, dir)
	outputPath := filepath.Join(dir, "out.txt")

	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"render", "--manifest", manifestPath, "--template-string", "{{.Title}}", "--output", outputPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !strings.Contains(string(data), "テスト番組") {
		t.Errorf("output file should contain title, got: %q", string(data))
	}
}

func TestRootHelp_ContainsRender(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})
	_ = cmd.Execute()

	out := buf.String()
	if !strings.Contains(out, "render") {
		t.Errorf("root help should contain render, got: %s", out)
	}
}
