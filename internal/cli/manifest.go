package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/manifest"
	"github.com/canpok1/vox-radio/internal/model"
	programsummary "github.com/canpok1/vox-radio/internal/script/summary"
	"github.com/spf13/cobra"
)

func newManifestCmd() *cobra.Command {
	var specPath string
	var rundownPath string
	var audioPath string
	var out string
	var linesPath string
	var scriptPath string
	var clipsPath string
	var timelinePath string

	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "エピソードのコンテンツマニフェスト JSON を生成する",
		Long: `エピソードの内容を記述する manifest.json を生成します。
タイトル・説明・要約・日時・音声ファイル名・各コーナーの記事情報・会話メモを含みます。

マニフェストは別の配信サービスが RSS フィードを生成する際に使用することを想定しており、
フルパイプラインを再実行せずに済みます。

--lines を指定すると、共通設定ファイルの LLM 設定を使って
LLM が 03_lines.json（元表記のセリフ）から番組要約・会話メモ・コーナー単位要約を生成してマニフェストに追加します。
共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。
会話メモはキャラの近況・掛け合い・感想・ハプニング・継続ネタなど
rundown（記事の事実）に含まれない会話情報を幅広く記録します。

--lines または --script を指定すると、実際に使用されたアセット・キャラクターの
クレジット情報を自動収集してマニフェストに含めます（OtoLogic https://otologic.jp / CC BY 4.0 等）。

例:
  vox-radio episodegen manifest --spec episode-spec.yaml --audio output/episode.mp3 --out output/manifest.json
  vox-radio episodegen manifest --spec episode-spec.yaml --rundown output/intermediate/02_rundown.json --audio output/episode.mp3 --out output/manifest.json
  vox-radio episodegen manifest --spec episode-spec.yaml --lines output/intermediate/03_lines.json --audio output/episode.mp3 --out output/manifest.json
  vox-radio episodegen manifest --spec episode-spec.yaml --lines output/intermediate/03_lines.json --script output/intermediate/04_script.json --audio output/episode.mp3 --out output/manifest.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("manifest", logDirFlag(cmd))
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			manifestLogger := logger.With("step", "manifest")
			manifestLogger.Info("開始")

			p, err := config.LoadEpisodeSpec(specPath)
			if err != nil {
				return fmt.Errorf("load spec: %w", err)
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
			var chars map[string]config.CharacterConfig
			var lines *model.ScriptLines
			var script *model.Script
			var clips *model.ClipsMeta
			var cornerDurations map[string]float64

			if linesPath != "" || scriptPath != "" {
				cfg, err := config.LoadConfig(configPath(cmd))
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}
				chars = cfg.Characters

				if linesPath != "" {
					llmClient := newLLMClient(cfg)

					prompts, err := loadPrompts()
					if err != nil {
						return fmt.Errorf("load prompts: %w", err)
					}

					scriptLines, err := readJSON[model.ScriptLines](linesPath)
					if err != nil {
						return fmt.Errorf("read lines: %w", err)
					}
					lines = &scriptLines

					ps := programsummary.NewLLMProgramSummarizer(llmClient, prompts["summary"], stepTemp(cfg.LLM, "summary"), p.Program.EffectiveSummaryLength(), programsummary.WithLogger(logger))
					programSummary, err = ps.Summarize(context.Background(), scriptLines)
					if err != nil {
						return fmt.Errorf("summarize program: %w", err)
					}

					cs := programsummary.NewLLMCornerSummarizer(llmClient, prompts["corner_summary"], stepTemp(cfg.LLM, "corner_summary"), programsummary.WithLogger(logger))
					cornerSummaries = make(map[string]model.CornerSummary, len(scriptLines.Corners))
					for _, cl := range scriptLines.Corners {
						result, err := cs.SummarizeCorner(context.Background(), cl, p.CornerSummaryLength(cl.Title))
						if err != nil {
							return fmt.Errorf("summarize corner %s: %w", cl.Title, err)
						}
						cornerSummaries[cl.Title] = result
					}
				}

				if scriptPath != "" {
					scr, err := readJSON[model.Script](scriptPath)
					if err != nil {
						return fmt.Errorf("read script: %w", err)
					}
					script = &scr
				}
			}

			if clipsPath != "" {
				cm, err := readJSON[model.ClipsMeta](clipsPath)
				if err != nil {
					return fmt.Errorf("read clips: %w", err)
				}
				clips = &cm
			}

			if timelinePath != "" {
				tl, err := readJSON[model.Timeline](timelinePath)
				if err != nil {
					return fmt.Errorf("read timeline: %w", err)
				}
				cornerDurations = make(map[string]float64, len(tl.Corners))
				for _, ct := range tl.Corners {
					cornerDurations[ct.ID] = ct.DurationSec
				}
			}

			m := manifest.Build(manifest.BuildParams{
				Program:           p.Program,
				Corners:           p.Corners,
				Rundown:           rd,
				AudioFile:         filepath.Base(audioPath),
				GeneratedAt:       time.Now().UTC(),
				Summary:           programSummary.Summary,
				CornerSummaries:   cornerSummaries,
				ConversationNotes: programSummary.ConversationNotes,
				EpisodeTitle:      programSummary.EpisodeTitle,
				Assets:            p.Assets,
				Characters:        chars,
				Lines:             lines,
				Script:            script,
				Clips:             clips,
				CornerDurations:   cornerDurations,
			})

			if err := writeJSON(out, m); err != nil {
				return err
			}

			manifestLogger.Info("完了")
			fmt.Printf("manifest written to %s\n", out)
			return nil
		},
	}

	registerSpecFlag(cmd, &specPath)
	cmd.Flags().StringVar(&rundownPath, "rundown", "", "02_rundown.json のパス（任意）。省略するとコーナーの記事は空になる")
	cmd.Flags().StringVar(&audioPath, "audio", "", "音声ファイルのパス。ファイル名のみマニフェストに記録される（必須）")
	cmd.Flags().StringVar(&out, "out", "", "manifest.json の出力先パス（必須）")
	cmd.Flags().StringVar(&linesPath, "lines", "", "03_lines.json のパス（任意）。指定すると LLM が元表記のセリフから番組要約・会話メモ・コーナー単位要約を生成する")
	cmd.Flags().StringVar(&scriptPath, "script", "", "04_script.json のパス（任意）。指定すると SE アセットのクレジットを自動収集する")
	cmd.Flags().StringVar(&clipsPath, "clips", "", "05_clips/clips.json のパス（任意）。指定するとコーナー別の speech_sec・char_count をマニフェストに追加する")
	cmd.Flags().StringVar(&timelinePath, "timeline", "", "06_timeline.json のパス（任意）。指定するとコーナー別の duration_sec をマニフェストに追加する")
	_ = cmd.MarkFlagRequired("audio")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
