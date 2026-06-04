package slack_test

import (
	"strings"
	"testing"

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
				Title:   "今週のニュース",
				Summary: "各社の新モデル発表が相次いだ一週間でした。",
				Articles: []model.ArticleRef{
					{Title: "OpenAIが新モデル発表", URL: "https://example.com/1"},
					{Title: "Googleの新Gemini登場", URL: "https://example.com/2"},
				},
			},
			{
				Title:   "深掘りコーナー",
				Summary: "エージェント設計のベストプラクティスを議論。",
				Articles: []model.ArticleRef{
					{Title: "エージェント設計論", URL: "https://example.com/3"},
				},
			},
		},
	}
}

func makeTemplate() model.MessageTemplate {
	return model.MessageTemplate{
		Header:   "🎙️ {title} 第{episode_number}回「{episode_title}」",
		Fallback: "{title} 第{episode_number}回 を配信しました",
		Summary:  "*今回のまとめ*\n{summary}",
		Corner:   "*{corner_title}*\n{corner_summary}\n{articles}",
		Article:  " • <{url}|{title}>",
	}
}

func TestBuildHeader_FullManifest(t *testing.T) {
	manifest := makeManifest()
	tmpl := makeTemplate()

	header := slack.BuildHeader(manifest, tmpl)

	want := "🎙️ ずんだもんテックラジオ 第42回「大規模言語モデルの最前線」"
	if header != want {
		t.Errorf("BuildHeader = %q, want %q", header, want)
	}
}

func TestBuildHeader_EmptyEpisodeTitle_RemovesEmptyQuotes(t *testing.T) {
	manifest := makeManifest()
	manifest.EpisodeTitle = ""
	tmpl := makeTemplate()

	header := slack.BuildHeader(manifest, tmpl)

	if header == "" {
		t.Error("BuildHeader must not be empty")
	}
	if strings.Contains(header, "「」") {
		t.Errorf("BuildHeader should not contain empty quotes 「」, got %q", header)
	}
}

func TestBuildHeader_EpisodeNumberZero_RemovesEpisodeSegment(t *testing.T) {
	manifest := makeManifest()
	manifest.EpisodeNumber = 0
	tmpl := makeTemplate()

	header := slack.BuildHeader(manifest, tmpl)

	if strings.Contains(header, "第0回") {
		t.Errorf("BuildHeader should not contain 第0回, got %q", header)
	}
	if strings.Contains(header, "第") {
		t.Errorf("BuildHeader should not contain episode segment when EpisodeNumber is 0, got %q", header)
	}
}

func TestBuildFallback(t *testing.T) {
	manifest := makeManifest()
	tmpl := makeTemplate()

	fallback := slack.BuildFallback(manifest, tmpl)

	want := "ずんだもんテックラジオ 第42回 を配信しました"
	if fallback != want {
		t.Errorf("BuildFallback = %q, want %q", fallback, want)
	}
}

func TestBuildThreadBlocks_WithSummaryAndCorners(t *testing.T) {
	manifest := makeManifest()
	tmpl := makeTemplate()

	blocks, fallback := slack.BuildThreadBlocks(manifest, tmpl)

	if fallback == "" {
		t.Error("fallback must not be empty")
	}
	if len(blocks) == 0 {
		t.Fatal("blocks must not be empty when summary and corners are present")
	}

	// 要約 Section + Divider + コーナー Section×2 = 4ブロック以上
	if len(blocks) < 4 {
		t.Errorf("expected at least 4 blocks (summary+divider+corner×2), got %d", len(blocks))
	}

	// 最初のブロックは summary Section
	firstSection, ok := blocks[0].(*slackgo.SectionBlock)
	if !ok {
		t.Fatalf("blocks[0] should be SectionBlock, got %T", blocks[0])
	}
	if firstSection.Text == nil || firstSection.Text.Text == "" {
		t.Error("first section text must not be empty")
	}

	// 2番目は Divider
	_, ok = blocks[1].(*slackgo.DividerBlock)
	if !ok {
		t.Errorf("blocks[1] should be DividerBlock, got %T", blocks[1])
	}
}

func TestBuildThreadBlocks_EmptySummary_NoBummarySection(t *testing.T) {
	manifest := makeManifest()
	manifest.Summary = ""
	tmpl := makeTemplate()

	blocks, _ := slack.BuildThreadBlocks(manifest, tmpl)

	if len(blocks) == 0 {
		t.Fatal("blocks must not be empty when corners are present")
	}

	// 要約がないので最初のブロックは Divider ではなく Corner Section になる
	_, isDivider := blocks[0].(*slackgo.DividerBlock)
	if isDivider {
		t.Error("first block should not be Divider when summary is empty")
	}
}

func TestBuildThreadBlocks_NoCorners_NoDivider(t *testing.T) {
	manifest := makeManifest()
	manifest.Corners = nil
	tmpl := makeTemplate()

	blocks, _ := slack.BuildThreadBlocks(manifest, tmpl)

	// コーナーが0件なので divider 以降は出ない
	for _, b := range blocks {
		if _, ok := b.(*slackgo.DividerBlock); ok {
			t.Error("blocks should not contain Divider when no corners")
		}
	}
}

func TestBuildThreadBlocks_BothEmpty_NilBlocks(t *testing.T) {
	manifest := makeManifest()
	manifest.Summary = ""
	manifest.Corners = nil
	tmpl := makeTemplate()

	blocks, _ := slack.BuildThreadBlocks(manifest, tmpl)

	if len(blocks) != 0 {
		t.Errorf("expected empty blocks when both summary and corners are empty, got %d blocks", len(blocks))
	}
}

func TestBuildThreadBlocks_CornerEmptyArticles_NoArticleLines(t *testing.T) {
	manifest := makeManifest()
	manifest.Summary = ""
	manifest.Corners = []model.ManifestCorner{
		{Title: "タイトルのみ", Summary: "", Articles: nil},
	}
	tmpl := makeTemplate()

	blocks, _ := slack.BuildThreadBlocks(manifest, tmpl)

	if len(blocks) == 0 {
		t.Fatal("blocks must not be empty when corner title exists")
	}
	section, ok := blocks[0].(*slackgo.SectionBlock)
	if !ok {
		t.Fatalf("blocks[0] should be SectionBlock, got %T", blocks[0])
	}
	if section.Text == nil {
		t.Error("section text must not be nil")
	}
}

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
