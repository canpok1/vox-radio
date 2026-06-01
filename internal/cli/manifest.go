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
	var rundownPath string
	var audioPath string
	var out string
	var scriptPath string
	var linesPath string
	var promptsDir string

	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "エピソードのコンテンツマニフェスト JSON を生成する",
		Long: `エピソードの内容を記述する manifest.json を生成します。
タイトル・説明・要約・日時・音声ファイル名・各コーナーの記事情報・会話メモを含みます。

マニフェストは別の配信サービスが RSS フィードを生成する際に使用することを想定しており、
フルパイプラインを再実行せずに済みます。

--script を指定すると、vox-radio.yaml の LLM 設定を使って
LLM が生成した番組要約と会話メモ（conversation_notes）をマニフェストに追加します。
会話メモはキャラの近況・掛け合い・感想・ハプニング・継続ネタなど
rundown（記事の事実）に含まれない会話情報を幅広く記録します。

--lines を指定すると、vox-radio.yaml の LLM 設定を使って
LLM が各コーナーの台本からコーナー単位の要約を生成してマニフェストに追加します。

例:
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --rundown output/intermediate/02_rundown.json --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --script output/intermediate/04_script.json --audio output/episode.mp3 --out output/manifest.json
  vox-radio manifest --profile sample-profiles/tech_profile.yaml --lines output/intermediate/03_lines.json --audio output/episode.mp3 --out output/manifest.json`,
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

			var rd model.Rundown
			if rundownPath != "" {
				rd, err = readJSON[model.Rundown](rundownPath)
				if err != nil {
					return fmt.Errorf("read rundown: %w", err)
				}
			}

			var programSummary model.ProgramSummary
			var cornerSummaries map[string]model.CornerSummary
			if scriptPath != "" || linesPath != "" {
				cfg, err := config.LoadConfig("vox-radio.yaml")
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}
				llmClient := newLLMClient(cfg)

				if scriptPath != "" {
					scr, err := readJSON[model.Script](scriptPath)
					if err != nil {
						return fmt.Errorf("read script: %w", err)
					}

					summaryPromptData, err := os.ReadFile(filepath.Join(promptsDir, "summary.md"))
					if err != nil {
						return fmt.Errorf("read summary.md: %w", err)
					}

					s := programsummary.NewLLMProgramSummarizer(llmClient, string(summaryPromptData), stepTemp(cfg.LLM, "summary"))
					programSummary, err = s.Summarize(context.Background(), scr)
					if err != nil {
						return fmt.Errorf("summarize program: %w", err)
					}
				}

				if linesPath != "" {
					scriptLines, err := readJSON[model.ScriptLines](linesPath)
					if err != nil {
						return fmt.Errorf("read lines: %w", err)
					}

					cornerSummaryPromptData, err := os.ReadFile(filepath.Join(promptsDir, "corner_summary.md"))
					if err != nil {
						return fmt.Errorf("read corner_summary.md: %w", err)
					}

					cs := programsummary.NewLLMCornerSummarizer(llmClient, string(cornerSummaryPromptData), stepTemp(cfg.LLM, "corner_summary"))
					cornerSummaries = make(map[string]model.CornerSummary, len(scriptLines.Corners))
					for _, cl := range scriptLines.Corners {
						result, err := cs.SummarizeCorner(context.Background(), cl)
						if err != nil {
							return fmt.Errorf("summarize corner %s: %w", cl.Title, err)
						}
						cornerSummaries[cl.Title] = result
					}
				}
			}

			m := manifest.Build(p.Program, p.Corners, rd, filepath.Base(audioPath), time.Now().UTC(), programSummary.Summary, cornerSummaries, programSummary.ConversationNotes)

			if err := writeJSON(out, m); err != nil {
				return err
			}

			manifestLogger.Info("完了")
			fmt.Printf("manifest written to %s\n", out)
			return nil
		},
	}

	registerProfileFlag(cmd, &profilePath)
	cmd.Flags().StringVar(&rundownPath, "rundown", "", "02_rundown.json のパス（任意）。省略するとコーナーの記事は空になる")
	cmd.Flags().StringVar(&audioPath, "audio", "", "音声ファイルのパス。ファイル名のみマニフェストに記録される（必須）")
	cmd.Flags().StringVar(&out, "out", "", "manifest.json の出力先パス（必須）")
	cmd.Flags().StringVar(&scriptPath, "script", "", "04_script.json のパス（任意）。指定すると LLM が台本から番組要約を生成する")
	cmd.Flags().StringVar(&linesPath, "lines", "", "03_lines.json のパス（任意）。指定すると LLM がコーナー台本からコーナー単位要約を生成する")
	cmd.Flags().StringVar(&promptsDir, "prompts", "prompts", "プロンプトテンプレートを含むディレクトリ（--script / --lines 指定時に使用）")
	_ = cmd.MarkFlagRequired("audio")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
