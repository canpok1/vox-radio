package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/manifest"
	"github.com/canpok1/vox-radio/internal/model"
	programsummary "github.com/canpok1/vox-radio/internal/script/summary"
	"github.com/spf13/cobra"
)

func newManifestCmd() *cobra.Command {
	var profilePath string
	var articlesPath string
	var audioPath string
	var out string
	var scriptPath string
	var promptsDir string

	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "エピソードのコンテンツマニフェスト JSON を生成する",
		Long: `エピソードの内容を記述する manifest.json を生成します。
タイトル・説明・要約・日時・音声ファイル名・各コーナーの記事情報を含みます。

マニフェストは別の配信サービスが RSS フィードを生成する際に使用することを想定しており、
フルパイプラインを再実行せずに済みます。

--script を指定すると、vox-radio.yaml の LLM 設定を使って
LLM が生成した要約をマニフェストに追加します。

例:
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --articles output/intermediate/01_articles.json --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --script output/intermediate/04_script.json --audio output/episode.mp3 --out output/manifest.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("manifest", "")
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			manifestLogger := logger.With("step", "manifest")
			manifestLogger.Info("開始")

			p, err := config.LoadProfile(profilePath)
			if err != nil {
				return fmt.Errorf("load profile: %w", err)
			}

			var articles model.Articles
			if articlesPath != "" {
				var err error
				articles, err = readJSON[model.Articles](articlesPath)
				if err != nil {
					return fmt.Errorf("read articles: %w", err)
				}
			}

			var programSummary string
			if scriptPath != "" {
				scr, err := readJSON[model.Script](scriptPath)
				if err != nil {
					return fmt.Errorf("read script: %w", err)
				}

				cfg, err := config.LoadConfig("vox-radio.yaml")
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}

				summaryPromptData, err := os.ReadFile(filepath.Join(promptsDir, "summary.md"))
				if err != nil {
					return fmt.Errorf("read summary.md: %w", err)
				}

				llmClient := newLLMClient(cfg)

				s := programsummary.NewLLMProgramSummarizer(llmClient, string(summaryPromptData), stepTemp(cfg.LLM, "summary"))
				programSummary, err = s.Summarize(context.Background(), scr)
				if err != nil {
					return fmt.Errorf("summarize program: %w", err)
				}
			}

			m := manifest.Build(p.Program, p.Corners, articles, filepath.Base(audioPath), time.Now().UTC(), programSummary)

			if err := writeJSON(out, m); err != nil {
				return err
			}

			manifestLogger.Info("完了")
			fmt.Printf("manifest written to %s\n", out)
			return nil
		},
	}

	registerProfileFlag(cmd, &profilePath)
	cmd.Flags().StringVar(&articlesPath, "articles", "", "articles.json のパス（任意）。省略するとコーナーの記事は空になる")
	cmd.Flags().StringVar(&audioPath, "audio", "", "音声ファイルのパス。ファイル名のみマニフェストに記録される（必須）")
	cmd.Flags().StringVar(&out, "out", "", "manifest.json の出力先パス（必須）")
	cmd.Flags().StringVar(&scriptPath, "script", "", "script.json のパス（任意）。指定すると LLM が台本から要約を生成する")
	cmd.Flags().StringVar(&promptsDir, "prompts", "prompts", "プロンプトテンプレートを含むディレクトリ（--script 指定時に使用）")
	_ = cmd.MarkFlagRequired("audio")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
