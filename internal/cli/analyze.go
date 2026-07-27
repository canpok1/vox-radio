package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/canpok1/vox-radio/internal/analyze"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"github.com/spf13/cobra"
)

// runAndSaveAnalysis reads the intermediate files an episodegen run just produced (lines and
// timeline are always present; proofread is present only when that step found corrections),
// computes the analysis, and writes it to layout.Analysis(). Used by the main episodegen flow;
// the standalone `episodegen analyze` command builds its inputs from explicit flags instead.
func runAndSaveAnalysis(ctx context.Context, cfg *config.Config, prompts map[string]string, llmClient llm.Client, p *config.EpisodeSpec, layout fileio.EpisodeLayout, logger *slog.Logger) error {
	lines, err := readJSON[model.ScriptLines](layout.Lines())
	if err != nil {
		return fmt.Errorf("read lines: %w", err)
	}

	pr, err := readOptionalJSON[model.ProofreadResult](layout.Proofread())
	if err != nil {
		return fmt.Errorf("read proofread: %w", err)
	}

	tl, err := readJSON[model.Timeline](layout.Timeline())
	if err != nil {
		return fmt.Errorf("read timeline: %w", err)
	}

	a, err := runAnalyzeStep(ctx, cfg, prompts, llmClient, p.Program, p.Corners, lines, pr, tl.Map(), logger)
	if err != nil {
		return fmt.Errorf("analyze: %w", err)
	}

	if err := writeJSON(layout.Analysis(), a); err != nil {
		return err
	}
	logger.Info("分析を出力", "findings", len(a.Findings), "patterns", len(a.Patterns))
	return nil
}

// runAnalyzeStep builds an LLMAnalyzer from cfg/prompts and computes the episode's analysis.
// Shared by the standalone `episodegen analyze` command and the main episodegen pipeline.
func runAnalyzeStep(ctx context.Context, cfg *config.Config, prompts map[string]string, llmClient llm.Client, program config.ProgramConfig, corners []config.CornerConfig, lines model.ScriptLines, pr *model.ProofreadResult, cornerDurations map[string]float64, logger *slog.Logger) (model.Analysis, error) {
	analyzer := analyze.NewLLMAnalyzer(llmClient, prompts["analyze"], stepTemp(cfg.LLM, "analyze"), analyze.WithLogger(logger))
	return analyzer.Analyze(ctx, program, corners, lines, pr, cornerDurations)
}

// readOptionalJSON reads a JSON file at path into a new T. It returns (nil, nil) when path is
// empty or the file does not exist, since several analyze inputs (proofread result) are only
// written when that step actually produced output.
func readOptionalJSON[T any](path string) (*T, error) {
	if path == "" {
		return nil, nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	v, err := readJSON[T](path)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func newAnalyzeCmd() *cobra.Command {
	var specPath string
	var linesPath string
	var proofreadPath string
	var timelinePath string
	var out string

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "台本とタイミング情報から回の分析（07_analysis.json）を生成する",
		Long: `コーナーの目標尺・実測尺・話者別セリフ数などの機械集計指標に加え、
LLM でこの回の課題（findings）と会話の型（patterns）を抽出し、07_analysis.json を生成します。

episodegen 本体は番組生成の一部として analyze を自動実行しキャッシュへ蓄積します。
このコマンドは既存回の中間ファイルから分析を作り直したいとき（抽出観点を変えた場合など）に使います。

共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。

例:
  vox-radio episodegen analyze --spec episode-spec.yaml --lines output/intermediate/prog_ep001/03_lines.json --timeline output/intermediate/prog_ep001/06_timeline.json --out output/intermediate/prog_ep001/07_analysis.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("analyze", logDirFlag(cmd))
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			cfg, p, err := loadConfigAndSpec(configPath(cmd), specPath)
			if err != nil {
				return err
			}

			if err := checkResources(func() error { return requireLLMKey(cfg) }); err != nil {
				return err
			}

			lines, err := readJSON[model.ScriptLines](linesPath)
			if err != nil {
				return fmt.Errorf("read lines: %w", err)
			}

			pr, err := readOptionalJSON[model.ProofreadResult](proofreadPath)
			if err != nil {
				return fmt.Errorf("read proofread: %w", err)
			}

			var cornerDurations map[string]float64
			if timelinePath != "" {
				tl, err := readJSON[model.Timeline](timelinePath)
				if err != nil {
					return fmt.Errorf("read timeline: %w", err)
				}
				cornerDurations = tl.Map()
			}

			ids := make([]string, len(lines.Corners))
			for i, c := range lines.Corners {
				ids[i] = c.ID
			}
			corners, err := config.ResolveCornersByIDs(p.Corners, ids)
			if err != nil {
				return fmt.Errorf("resolve corners: %w", err)
			}

			llmClient := newLLMClient(cfg)
			prompts, err := loadPrompts()
			if err != nil {
				return fmt.Errorf("load prompts: %w", err)
			}

			a, err := runAnalyzeStep(context.Background(), cfg, prompts, llmClient, p.Program, corners, lines, pr, cornerDurations, logger)
			if err != nil {
				return fmt.Errorf("analyze: %w", err)
			}

			if err := writeJSON(out, a); err != nil {
				return err
			}
			fmt.Printf("analysis written to %s (findings=%d, patterns=%d)\n", out, len(a.Findings), len(a.Patterns))
			return nil
		},
	}

	registerSpecFlag(cmd, &specPath)
	cmd.Flags().StringVar(&linesPath, "lines", "", "03_lines.json のパス（必須）")
	cmd.Flags().StringVar(&proofreadPath, "proofread", "", "04_proofread.json のパス（任意。無ければ校正修正0件として扱う）")
	cmd.Flags().StringVar(&timelinePath, "timeline", "", "06_timeline.json のパス（任意。無ければコーナー実測尺は0として扱う）")
	cmd.Flags().StringVar(&out, "out", "", "07_analysis.json の出力先パス（必須）")
	_ = cmd.MarkFlagRequired("lines")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
