package render_test

import (
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/render"
)

func makeManifest() model.Manifest {
	return model.Manifest{
		Title:         "テスト番組",
		EpisodeNumber: 1,
		EpisodeTitle:  "テスト回",
		Summary:       "今回のまとめ",
		Corners: []model.ManifestCorner{
			{
				ID:    "news",
				Title: "ニュースコーナー",
				Articles: []model.ArticleRef{
					{Title: "記事A", URL: "https://example.com/a"},
					{Title: "記事B", URL: ""},
				},
			},
			{
				ID:    "tech",
				Title: "テックコーナー",
				Articles: []model.ArticleRef{
					{Title: "記事C", URL: "https://example.com/c"},
				},
			},
		},
	}
}

func TestRender_BasicTemplate(t *testing.T) {
	m := makeManifest()
	tmpl := `タイトル: {{.Title}}`

	out, err := render.Render(tmpl, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "タイトル: テスト番組" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestRender_CornerFunction(t *testing.T) {
	m := makeManifest()
	tmpl := `{{with corner "news"}}{{.Title}}{{end}}`

	out, err := render.Render(tmpl, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ニュースコーナー" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestRender_CornerFunction_NotFound(t *testing.T) {
	m := makeManifest()
	tmpl := `{{with corner "missing"}}{{.Title}}{{else}}notfound{{end}}`

	out, err := render.Render(tmpl, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "notfound" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestRender_HasLinksFunction_True(t *testing.T) {
	m := makeManifest()
	tmpl := `{{range .Corners}}{{if hasLinks .}}{{.ID}}{{end}}{{end}}`

	out, err := render.Render(tmpl, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// newsもtechもURLありの記事を持つ
	if !strings.Contains(out, "news") || !strings.Contains(out, "tech") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestRender_HasLinksFunction_False(t *testing.T) {
	m := model.Manifest{
		Corners: []model.ManifestCorner{
			{
				ID:    "nolinks",
				Title: "リンクなし",
				Articles: []model.ArticleRef{
					{Title: "記事A", URL: ""},
				},
			},
		},
	}
	tmpl := `{{range .Corners}}{{if hasLinks .}}HASLINKS{{else}}NOLINKS{{end}}{{end}}`

	out, err := render.Render(tmpl, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "NOLINKS" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestRender_URLSkip(t *testing.T) {
	m := makeManifest()
	// URLありの記事のみリスト化するテンプレ
	tmpl := `{{range .Corners}}{{range .Articles}}{{if .URL}}- {{.Title}}
{{end}}{{end}}{{end}}`

	out, err := render.Render(tmpl, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "記事A") {
		t.Errorf("output should contain 記事A: %q", out)
	}
	if !strings.Contains(out, "記事C") {
		t.Errorf("output should contain 記事C: %q", out)
	}
	if strings.Contains(out, "記事B") {
		t.Errorf("output should NOT contain URL-less 記事B: %q", out)
	}
}

func TestRender_ParseError(t *testing.T) {
	m := makeManifest()
	tmpl := `{{.Title` // 構文エラー

	_, err := render.Render(tmpl, m)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestRender_ExecutionError(t *testing.T) {
	m := makeManifest()
	// 存在しないフィールドを option strict で使うと error
	// Go text/template はデフォルトでゼロ値になるが、
	// ここでは関数の型不一致でエラーを起こす
	tmpl := `{{corner 123}}` // corner は string 引数が必要

	_, err := render.Render(tmpl, m)
	if err == nil {
		t.Fatal("expected execution error, got nil")
	}
}
