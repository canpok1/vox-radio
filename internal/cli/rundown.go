package cli

import (
	"context"
	"fmt"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/rundown"
	sel "github.com/canpok1/vox-radio/internal/rundown/select"
	"github.com/canpok1/vox-radio/internal/script/summarize"
	"github.com/spf13/cobra"
)

func newRundownCmd() *cobra.Command {
	var in string
	var out string
	var profilePath string

	cmd := &cobra.Command{
		Use:   "rundown",
		Short: "収集記事から番組設計図（rundown）を生成する",
		Long: `LLM を使って収集記事を選別し、コーナーごとの話の流れと要約を含む
02_rundown.json を生成します。

vox-radio.yaml はカレントディレクトリから自動読み込みされます。
コーナー定義はプロファイルから取得します。

例:
  vox-radio rundown --in work/intermediate/01_articles.json --out work/intermediate/02_rundown.json --profile sample-profiles/tech_profile.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("rundown", "")
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()
			_ = logger

			cfg, p, err := loadConfigAndProfile(profilePath)
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

			selector := sel.NewLLMSelector(llmClient, prompts["select"], stepTemp(cfg.LLM, "select"))
			summarizer := summarize.NewLLMSummarizer(llmClient, prompts["summarize"], stepTemp(cfg.LLM, "summarize"))
			rd := rundown.NewLLMRundowner(selector, summarizer)

			result, err := rd.Run(context.Background(), p.Corners, articles)
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
	registerProfileFlag(cmd, &profilePath)
	_ = cmd.MarkFlagRequired("in")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
