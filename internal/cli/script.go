package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script"
	"github.com/canpok1/vox-radio/internal/script/direct"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"github.com/canpok1/vox-radio/internal/script/write"
	"github.com/spf13/cobra"
)

func newScriptCmd() *cobra.Command {
	var in string
	var out string
	var step string
	var specPath string

	cmd := &cobra.Command{
		Use:   "script",
		Short: "LLM を使って rundown から台本を生成する",
		Long: `多段階 LLM パイプライン（write → direct）を実行し、
02_rundown.json から 04_script.json を生成します。

vox-radio.yaml はカレントディレクトリから自動読み込みされます。
コーナー定義はプロファイルから取得します。

--step で単一ステージのみ実行できます:
  write      コーナーごとに台詞を書きます（03_lines.json を出力）
  direct     台詞に SE・話者を割り当てます（04_script.json を出力）

例:
  vox-radio episodegen script --in work/intermediate/02_rundown.json --out work/intermediate/04_script.json
  vox-radio episodegen script --out work/intermediate/04_script.json --step write
  vox-radio episodegen script --in work/intermediate/02_rundown.json --out work/intermediate/04_script.json --spec examples/tech.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("script", "")
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			cfg, p, err := loadConfigAndSpec(specPath)
			if err != nil {
				return err
			}

			llmClient := newLLMClient(cfg)

			prompts, err := loadPrompts()
			if err != nil {
				return fmt.Errorf("load prompts: %w", err)
			}

			workDir := filepath.Dir(out)
			if err := os.MkdirAll(workDir, 0o755); err != nil {
				return fmt.Errorf("create work dir: %w", err)
			}

			assetCatalog := buildAssetCatalog(p.Assets)

			switch step {
			case "":
				return runScriptFull(context.Background(), in, out, workDir, llmClient, cfg, p, prompts, assetCatalog, logger)
			case "write":
				return runScriptWrite(context.Background(), in, workDir, llmClient, cfg, p, prompts)
			case "direct":
				return runScriptDirect(context.Background(), workDir, out, llmClient, cfg.LLM, prompts, assetCatalog)
			default:
				return fmt.Errorf("unknown step %q: use write|direct", step)
			}
		},
	}

	cmd.Flags().StringVar(&in, "in", "", "02_rundown.json の入力パス（フルパイプラインまたは write ステップで必須）")
	cmd.Flags().StringVar(&out, "out", "", "04_script.json の出力先パス（必須）")
	cmd.Flags().StringVar(&step, "step", "", "単一ステップを実行する: write|direct")
	registerSpecFlag(cmd, &specPath)
	_ = cmd.MarkFlagRequired("out")

	return cmd
}

func runScriptFull(ctx context.Context, in, out, workDir string, c llm.Client, cfg *config.Config, p *config.EpisodeSpec, prompts map[string]string, assetCatalog model.AssetCatalog, logger *slog.Logger) error {
	if in == "" {
		return fmt.Errorf("--in is required for full pipeline")
	}
	rundown, err := readJSON[model.Rundown](in)
	if err != nil {
		return fmt.Errorf("read rundown: %w", err)
	}

	gen := script.NewLLMScriptGenerator(
		write.NewLLMWriter(c, prompts["write"], stepTemp(cfg.LLM, "write"), cfg),
		direct.NewLLMDirector(c, prompts["direct"], stepTemp(cfg.LLM, "direct")),
		assetCatalog,
		workDir,
		script.WithLogger(logger),
	)

	scr, err := gen.Generate(ctx, p.Program, rundown, p.Corners, cfg.Characters)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	return writeJSON(out, scr)
}

func runScriptWrite(ctx context.Context, in, workDir string, c llm.Client, cfg *config.Config, p *config.EpisodeSpec, prompts map[string]string) error {
	if in == "" {
		return fmt.Errorf("--in is required for write step")
	}
	rundown, err := readJSON[model.Rundown](in)
	if err != nil {
		return fmt.Errorf("read rundown: %w", err)
	}

	cornerMap := rundown.CornerMap()

	w := write.NewLLMWriter(c, prompts["write"], stepTemp(cfg.LLM, "write"), cfg)
	allCornerLines, err := script.WriteAll(ctx, w, p.Program, p.Corners, cornerMap, cfg.Characters)
	if err != nil {
		return fmt.Errorf("write corners: %w", err)
	}

	scriptLines := model.ScriptLines{Corners: script.BuildScriptLines(p.Corners, allCornerLines)}
	outPath := filepath.Join(workDir, "03_lines.json")
	if err := writeJSON(outPath, scriptLines); err != nil {
		return err
	}
	fmt.Printf("wrote %d lines to %s\n", scriptLines.TotalLines(), outPath)
	return nil
}

func runScriptDirect(ctx context.Context, workDir, out string, c llm.Client, llmCfg config.LLMConfig, prompts map[string]string, assetCatalog model.AssetCatalog) error {
	linesPath := filepath.Join(workDir, "03_lines.json")
	data, err := os.ReadFile(linesPath)
	if err != nil {
		return fmt.Errorf("read 03_lines.json: %w", err)
	}
	var scriptLines model.ScriptLines
	if err := json.Unmarshal(data, &scriptLines); err != nil {
		return fmt.Errorf("parse 03_lines.json: %w", err)
	}

	d := direct.NewLLMDirector(c, prompts["direct"], stepTemp(llmCfg, "direct"))
	scr, err := d.Direct(ctx, scriptLines.Corners, assetCatalog)
	if err != nil {
		return fmt.Errorf("direct: %w", err)
	}

	if err := writeJSON(out, scr); err != nil {
		return err
	}
	fmt.Printf("directed %d segments to %s\n", len(scr.Segments), out)
	return nil
}

func buildAssetCatalog(assets config.AssetsConfig) model.AssetCatalog {
	return model.AssetCatalog{
		SE: buildAssetEntries(assets.SE, func(e config.SEEntry) string { return e.Description }),
	}
}

// buildAssetEntries converts a config asset map to a sorted, non-nil slice of AssetCatalogEntry.
// slices.Sorted returns nil for empty iterators, causing JSON to marshal as null instead of [].
func buildAssetEntries[V any](m map[string]V, getDesc func(V) string) []model.AssetCatalogEntry {
	if len(m) == 0 {
		return make([]model.AssetCatalogEntry, 0)
	}
	keys := slices.Sorted(maps.Keys(m))
	entries := make([]model.AssetCatalogEntry, len(keys))
	for i, k := range keys {
		entries[i] = model.AssetCatalogEntry{
			Name:        k,
			Description: getDesc(m[k]),
		}
	}
	return entries
}

func stepTemp(llmCfg config.LLMConfig, name string) float64 {
	if s, ok := llmCfg.Steps[name]; ok && s.Temperature != nil {
		return *s.Temperature
	}
	return 0
}
