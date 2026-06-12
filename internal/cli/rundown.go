package cli

import (
	"context"
	"fmt"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/rundown"
	"github.com/canpok1/vox-radio/internal/rundown/flow"
	sel "github.com/canpok1/vox-radio/internal/rundown/select"
	"github.com/canpok1/vox-radio/internal/script/summarize"
	"github.com/spf13/cobra"
)

func newRundownCmd() *cobra.Command {
	var in string
	var out string
	var specPath string

	cmd := &cobra.Command{
		Use:   "rundown",
		Short: "収集記事から番組設計図（rundown）を生成する",
		Long: `LLM を使って収集記事を選別し、コーナーごとの話の流れと要約を含む
02_rundown.json を生成します。

共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。
コーナー定義はエピソード仕様から取得します。

例:
  vox-radio episodegen rundown --in work/intermediate/01_articles.json --out work/intermediate/02_rundown.json --spec sample/episode-spec.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("rundown", logDirFlag(cmd))
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			cfg, p, err := loadConfigAndSpec(configPath(cmd), specPath)
			if err != nil {
				return err
			}

			llmClient := newLLMClient(cfg)

			prompts, err := loadPrompts()
			if err != nil {
				return fmt.Errorf("load prompts: %w", err)
			}

			articles, err := readJSON[model.Articles](in)
			if err != nil {
				return fmt.Errorf("read articles: %w", err)
			}

			entries, episodeNumber, err := loadCacheEntries(p.Program.ID)
			if err != nil {
				return err
			}
			corners := resolveCorners(p.Corners, episodeNumber)
			castAppearances := cache.CastAppearances(entries)
			cornerAppearances := cache.CornerAppearances(entries)

			selector := sel.NewLLMSelector(llmClient, prompts["select"], stepTemp(cfg.LLM, "select"))
			summarizer := summarize.NewLLMSummarizer(llmClient, prompts["summarize"], stepTemp(cfg.LLM, "summarize"))
			flowDesigner := flow.NewLLMDesigner(llmClient, prompts["flow"], stepTemp(cfg.LLM, "flow"))
			casts := selectCasts(p.Casts, episodeNumber, castAppearances)
			selector.SetCasts(casts)
			rd := rundown.NewLLMRundowner(selector, summarizer, flowDesigner, nil, rundown.WithLogger(logger))
			rd.SetCornerAppearances(cornerAppearances)

			result, err := rd.Run(context.Background(), corners, articles, casts)
			if err != nil {
				return fmt.Errorf("rundown: %w", err)
			}

			if err := writeJSON(out, result); err != nil {
				return err
			}

			fmt.Printf("rundown written to %s\n", out)
			return nil
		},
	}

	cmd.Flags().StringVar(&in, "in", "", "01_articles.json の入力パス（必須）")
	cmd.Flags().StringVar(&out, "out", "", "02_rundown.json の出力先パス（必須）")
	registerSpecFlag(cmd, &specPath)
	_ = cmd.MarkFlagRequired("in")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
