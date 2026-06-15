package slack_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	slackgo "github.com/slack-go/slack"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/slack"
)

func makeManifest() model.Manifest {
	return model.Manifest{
		Title:         "ずんだもんテックラジオ",
		EpisodeNumber: 42,
		EpisodeTitle:  "大規模言語モデルの最前線",
		Summary:       "今回はLLMの最新動向をお届けしました。",
		AudioFile:     "episode42.mp3",
		Corners: []model.ManifestCorner{
			{
				ID:      "news",
				Title:   "今週のニュース",
				Summary: "各社の新モデル発表が相次いだ一週間でした。",
				Articles: []model.ArticleRef{
					{Title: "OpenAIが新モデル発表", URL: "https://example.com/1"},
					{Title: "Googleの新Gemini登場", URL: "https://example.com/2"},
				},
			},
			{
				ID:      "deep",
				Title:   "深掘りコーナー",
				Summary: "エージェント設計のベストプラクティスを議論。",
				Articles: []model.ArticleRef{
					{Title: "エージェント設計論", URL: "https://example.com/3"},
				},
			},
		},
	}
}

// BuildParent

func TestBuildParent_FullManifest(t *testing.T) {
	m := makeManifest()
	tmpl := `🎙️ {{.Title}}{{if .EpisodeNumber}} 第{{.EpisodeNumber}}回{{end}}{{if .EpisodeTitle}}「{{.EpisodeTitle}}」{{end}}`

	got, err := slack.BuildParent(m, tmpl)
	if err != nil {
		t.Fatalf("BuildParent: unexpected error: %v", err)
	}
	want := "🎙️ ずんだもんテックラジオ 第42回「大規模言語モデルの最前線」"
	if got != want {
		t.Errorf("BuildParent = %q, want %q", got, want)
	}
}

func TestBuildParent_EpisodeNumberZero_OmitsEpisodeSegment(t *testing.T) {
	m := makeManifest()
	m.EpisodeNumber = 0
	tmpl := `🎙️ {{.Title}}{{if .EpisodeNumber}} 第{{.EpisodeNumber}}回{{end}}{{if .EpisodeTitle}}「{{.EpisodeTitle}}」{{end}}`

	got, err := slack.BuildParent(m, tmpl)
	if err != nil {
		t.Fatalf("BuildParent: unexpected error: %v", err)
	}
	if strings.Contains(got, "第0回") {
		t.Errorf("BuildParent should not contain 第0回 when EpisodeNumber is 0, got %q", got)
	}
	if strings.Contains(got, "第") {
		t.Errorf("BuildParent should not contain episode segment when EpisodeNumber is 0, got %q", got)
	}
}

func TestBuildParent_EmptyEpisodeTitle_OmitsEmptyQuotes(t *testing.T) {
	m := makeManifest()
	m.EpisodeTitle = ""
	tmpl := `🎙️ {{.Title}}{{if .EpisodeNumber}} 第{{.EpisodeNumber}}回{{end}}{{if .EpisodeTitle}}「{{.EpisodeTitle}}」{{end}}`

	got, err := slack.BuildParent(m, tmpl)
	if err != nil {
		t.Fatalf("BuildParent: unexpected error: %v", err)
	}
	if strings.Contains(got, "「」") {
		t.Errorf("BuildParent should not contain empty quotes 「」, got %q", got)
	}
	if got == "" {
		t.Error("BuildParent must not be empty")
	}
}

func TestBuildParent_InvalidTemplate_Error(t *testing.T) {
	m := makeManifest()
	_, err := slack.BuildParent(m, "{{invalid")
	if err == nil {
		t.Error("expected error for invalid template syntax")
	}
}

func TestBuildParent_TrimsWhitespace(t *testing.T) {
	m := makeManifest()
	tmpl := "  {{.Title}}  "

	got, err := slack.BuildParent(m, tmpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != strings.TrimSpace(got) {
		t.Errorf("BuildParent should trim whitespace, got %q", got)
	}
}

// BuildFallback

func TestBuildFallback_BasicRender(t *testing.T) {
	m := makeManifest()
	tmpl := `{{.Title}}{{if .EpisodeNumber}} 第{{.EpisodeNumber}}回{{end}} を配信しました`

	got, err := slack.BuildFallback(m, tmpl)
	if err != nil {
		t.Fatalf("BuildFallback: unexpected error: %v", err)
	}
	want := "ずんだもんテックラジオ 第42回 を配信しました"
	if got != want {
		t.Errorf("BuildFallback = %q, want %q", got, want)
	}
}

func TestBuildFallback_InvalidTemplate_Error(t *testing.T) {
	m := makeManifest()
	_, err := slack.BuildFallback(m, "{{invalid")
	if err == nil {
		t.Error("expected error for invalid template syntax")
	}
}

// BuildThread

func TestBuildThread_RendersTemplate(t *testing.T) {
	m := makeManifest()
	tmpl := `{{.Summary}}`

	got, err := slack.BuildThread(m, tmpl)
	if err != nil {
		t.Fatalf("BuildThread: unexpected error: %v", err)
	}
	if got != m.Summary {
		t.Errorf("BuildThread = %q, want %q", got, m.Summary)
	}
}

func TestBuildThread_URLSkip(t *testing.T) {
	m := makeManifest()
	m.Corners = []model.ManifestCorner{
		{
			Title: "テック",
			Articles: []model.ArticleRef{
				{Title: "URL付き記事", URL: "https://example.com"},
				{Title: "URLなし記事", URL: ""},
			},
		},
	}
	tmpl := `{{range .Corners}}{{range .Articles}}{{if .URL}} • <{{.URL}}|{{.Title}}>
{{end}}{{end}}{{end}}`

	got, err := slack.BuildThread(m, tmpl)
	if err != nil {
		t.Fatalf("BuildThread: unexpected error: %v", err)
	}
	if !strings.Contains(got, "URL付き記事") {
		t.Errorf("BuildThread should contain URL付き記事, got: %q", got)
	}
	if strings.Contains(got, "URLなし記事") {
		t.Errorf("BuildThread should NOT contain URLなし記事, got: %q", got)
	}
}

func TestBuildThread_CornerFunction(t *testing.T) {
	m := makeManifest()
	tmpl := `{{with corner "news"}}コーナー: {{.Title}}{{end}}`

	got, err := slack.BuildThread(m, tmpl)
	if err != nil {
		t.Fatalf("BuildThread: unexpected error: %v", err)
	}
	if !strings.Contains(got, "今週のニュース") {
		t.Errorf("BuildThread should contain corner title via corner function, got: %q", got)
	}
}

func TestBuildThread_CornerFunctionNotFound_ReturnsEmpty(t *testing.T) {
	m := makeManifest()
	tmpl := `{{with corner "nonexistent"}}{{.Title}}{{end}}`

	got, err := slack.BuildThread(m, tmpl)
	if err != nil {
		t.Fatalf("BuildThread: unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("BuildThread corner function with unknown ID should return empty, got: %q", got)
	}
}

func TestBuildThread_HasLinksFunction(t *testing.T) {
	m := model.Manifest{
		Corners: []model.ManifestCorner{
			{
				ID:    "withlinks",
				Title: "リンクあり",
				Articles: []model.ArticleRef{
					{Title: "A", URL: "https://example.com"},
				},
			},
			{
				ID:    "nolinks",
				Title: "リンクなし",
				Articles: []model.ArticleRef{
					{Title: "B", URL: ""},
				},
			},
		},
	}
	tmpl := `{{range .Corners}}{{if hasLinks .}}{{.Title}}
{{end}}{{end}}`

	got, err := slack.BuildThread(m, tmpl)
	if err != nil {
		t.Fatalf("BuildThread: unexpected error: %v", err)
	}
	if !strings.Contains(got, "リンクあり") {
		t.Errorf("BuildThread should include corner with links, got: %q", got)
	}
	if strings.Contains(got, "リンクなし") {
		t.Errorf("BuildThread should exclude corner without links, got: %q", got)
	}
}

func TestBuildThread_InvalidTemplate_Error(t *testing.T) {
	m := makeManifest()
	_, err := slack.BuildThread(m, "{{invalid")
	if err == nil {
		t.Error("expected error for invalid template syntax")
	}
}

func TestBuildThread_TrimsWhitespace(t *testing.T) {
	m := makeManifest()
	tmpl := "  {{.Title}}  "

	got, err := slack.BuildThread(m, tmpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != strings.TrimSpace(got) {
		t.Errorf("BuildThread should trim whitespace, got %q", got)
	}
}

// SplitIntoSectionBlocks

func TestSplitIntoSectionBlocks_EmptyText_ReturnsNil(t *testing.T) {
	blocks := slack.SplitIntoSectionBlocks("")
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks for empty text, got %d", len(blocks))
	}
}

func TestSplitIntoSectionBlocks_ShortText_OneBlock(t *testing.T) {
	text := "短いテキスト"
	blocks := slack.SplitIntoSectionBlocks(text)
	if len(blocks) != 1 {
		t.Errorf("expected 1 block for short text, got %d", len(blocks))
	}
	section, ok := blocks[0].(*slackgo.SectionBlock)
	if !ok {
		t.Fatalf("block[0] should be SectionBlock, got %T", blocks[0])
	}
	if section.Text.Text != text {
		t.Errorf("section text: got %q, want %q", section.Text.Text, text)
	}
}

func TestSplitIntoSectionBlocks_LongText_MultipleBlocks(t *testing.T) {
	// 100文字 * 35行 = 3500文字（3000文字超）
	line := strings.Repeat("あ", 100)
	var sb strings.Builder
	for i := 0; i < 35; i++ {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	text := strings.TrimRight(sb.String(), "\n")

	blocks := slack.SplitIntoSectionBlocks(text)
	if len(blocks) < 2 {
		t.Errorf("expected multiple blocks for text > 3000 chars, got %d", len(blocks))
	}
	// 各ブロックは 3000 rune 以下
	for i, b := range blocks {
		section, ok := b.(*slackgo.SectionBlock)
		if !ok {
			t.Fatalf("block[%d] is not SectionBlock, got %T", i, b)
		}
		count := utf8.RuneCountInString(section.Text.Text)
		if count > 3000 {
			t.Errorf("block[%d] has %d runes, exceeds 3000", i, count)
		}
	}
}

func TestSplitIntoSectionBlocks_ExactlyAtLimit_OneBlock(t *testing.T) {
	// ちょうど 3000 文字
	text := strings.Repeat("a", 3000)
	blocks := slack.SplitIntoSectionBlocks(text)
	if len(blocks) != 1 {
		t.Errorf("expected 1 block for exactly 3000 chars, got %d", len(blocks))
	}
}

func TestSplitIntoSectionBlocks_AllBlocksWithinLimit(t *testing.T) {
	// 長いテキストを作成
	line := strings.Repeat("x", 200)
	var sb strings.Builder
	for i := 0; i < 30; i++ {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	text := strings.TrimRight(sb.String(), "\n")

	blocks := slack.SplitIntoSectionBlocks(text)
	for i, b := range blocks {
		section, ok := b.(*slackgo.SectionBlock)
		if !ok {
			t.Fatalf("block[%d] is not SectionBlock", i)
		}
		count := utf8.RuneCountInString(section.Text.Text)
		if count > 3000 {
			t.Errorf("block[%d] has %d runes, exceeds 3000", i, count)
		}
	}
}

// BuildAudioTitle (unchanged behavior)

func TestBuildAudioTitle_WithEpisodeTitle(t *testing.T) {
	manifest := makeManifest()
	title := slack.BuildAudioTitle(manifest)
	if title != manifest.EpisodeTitle {
		t.Errorf("BuildAudioTitle = %q, want episode_title %q", title, manifest.EpisodeTitle)
	}
}

func TestBuildAudioTitle_WithoutEpisodeTitle(t *testing.T) {
	manifest := makeManifest()
	manifest.EpisodeTitle = ""
	title := slack.BuildAudioTitle(manifest)
	if title != manifest.Title {
		t.Errorf("BuildAudioTitle = %q, want title %q", title, manifest.Title)
	}
}
