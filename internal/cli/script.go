package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/script"
	"github.com/canpok1/vox-radio/internal/script/direct"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"github.com/canpok1/vox-radio/internal/script/summarize"
	"github.com/canpok1/vox-radio/internal/script/write"
	"github.com/spf13/cobra"
)

func newScriptCmd() *cobra.Command {
	var in string
	var out string
	var step string
	var profilePath string
	var promptsDir string

	cmd := &cobra.Command{
		Use:   "script",
		Short: "Generate a script from collected articles using LLM",
		Long: `Run the multi-stage LLM pipeline (summarize → write → direct) to
produce script.json from articles.json.

vox-radio.yaml is automatically loaded from the current directory.
Corner definitions come from the profile (no plan step).

Use --step to run a single stage independently:
  summarize  Summarize each article per corner (writes summaries.json)
  write      Write lines per corner (writes lines.json)
  direct     Assign SE/speakers to lines (writes script.json)

Example:
  vox-radio script --in work/articles.json --out work/script.json
  vox-radio script --out work/script.json --step write
  vox-radio script --in work/articles.json --out work/script.json --profile profiles/tech/profile.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, p, err := loadConfigAndProfile(profilePath)
			if err != nil {
				return err
			}

			apiKey := os.Getenv(cfg.LLM.APIKeyEnv)
			llmClient := llm.NewClient(llm.Config{
				BaseURL:     cfg.LLM.BaseURL,
				APIKey:      apiKey,
				Model:       cfg.LLM.Model,
				Temperature: cfg.LLM.Temperature,
				MaxRetries:  cfg.LLM.MaxRetries,
			})

			prompts, err := loadPrompts(promptsDir)
			if err != nil {
				return fmt.Errorf("load prompts: %w", err)
			}

			workDir := filepath.Dir(out)
			if err := os.MkdirAll(workDir, 0o755); err != nil {
				return fmt.Errorf("create work dir: %w", err)
			}

			seCatalog := buildSECatalog(p.Assets)

			switch step {
			case "":
				return runScriptFull(context.Background(), in, out, workDir, llmClient, cfg.LLM, p, cfg.Characters, prompts, seCatalog)
			case "summarize":
				return runScriptSummarize(context.Background(), in, workDir, llmClient, cfg.LLM, prompts)
			case "write":
				return runScriptWrite(context.Background(), workDir, llmClient, cfg.LLM, p, cfg.Characters, prompts)
			case "direct":
				return runScriptDirect(context.Background(), workDir, out, llmClient, cfg.LLM, prompts, seCatalog)
			default:
				return fmt.Errorf("unknown step %q: use summarize|write|direct", step)
			}
		},
	}

	cmd.Flags().StringVar(&in, "in", "", "input articles.json path (required for full pipeline or summarize step)")
	cmd.Flags().StringVar(&out, "out", "", "output script.json path (required)")
	cmd.Flags().StringVar(&step, "step", "", "run a single step: summarize|write|direct")
	cmd.Flags().StringVar(&profilePath, "profile", "profiles/test/profile.yaml", "profile YAML file path")
	cmd.Flags().StringVar(&promptsDir, "prompts", "prompts", "directory containing prompt templates")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}

func runScriptFull(ctx context.Context, in, out, workDir string, c llm.Client, llmCfg config.LLMConfig, p *config.Profile, chars map[string]config.CharacterConfig, prompts map[string]string, seCatalog model.SECatalog) error {
	if in == "" {
		return fmt.Errorf("--in is required for full pipeline")
	}
	articles, err := readArticles(in)
	if err != nil {
		return err
	}

	gen := script.NewLLMScriptGenerator(
		summarize.NewLLMSummarizer(c, prompts["summarize"], stepTemp(llmCfg, "summarize")),
		write.NewLLMWriter(c, prompts["write"], stepTemp(llmCfg, "write")),
		direct.NewLLMDirector(c, prompts["direct"], stepTemp(llmCfg, "direct")),
		seCatalog,
		workDir,
	)

	scr, err := gen.Generate(ctx, articles, p.Corners, chars)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	return writeJSON(out, scr)
}

func runScriptSummarize(ctx context.Context, in, workDir string, c llm.Client, llmCfg config.LLMConfig, prompts map[string]string) error {
	if in == "" {
		return fmt.Errorf("--in is required for summarize step")
	}
	articles, err := readArticles(in)
	if err != nil {
		return err
	}

	s := summarize.NewLLMSummarizer(c, prompts["summarize"], stepTemp(llmCfg, "summarize"))
	cornerSummaries := make([]model.CornerSummaries, 0, len(articles.Corners))
	totalCount := 0
	for _, ca := range articles.Corners {
		sums := make([]model.Summary, 0, len(ca.Articles))
		for _, a := range ca.Articles {
			sum, err := s.Summarize(ctx, a)
			if err != nil {
				return fmt.Errorf("summarize %q: %w", a.URL, err)
			}
			sums = append(sums, sum)
		}
		cornerSummaries = append(cornerSummaries, model.CornerSummaries{
			CornerTitle: ca.CornerTitle,
			Summaries:   sums,
		})
		totalCount += len(sums)
	}

	outPath := filepath.Join(workDir, "summaries.json")
	if err := writeJSON(outPath, model.Summaries{Corners: cornerSummaries}); err != nil {
		return err
	}
	fmt.Printf("summarized %d articles to %s\n", totalCount, outPath)
	return nil
}

func runScriptWrite(ctx context.Context, workDir string, c llm.Client, llmCfg config.LLMConfig, p *config.Profile, chars map[string]config.CharacterConfig, prompts map[string]string) error {
	summariesPath := filepath.Join(workDir, "summaries.json")
	summariesData, err := os.ReadFile(summariesPath)
	if err != nil {
		return fmt.Errorf("read summaries.json: %w", err)
	}
	var sums model.Summaries
	if err := json.Unmarshal(summariesData, &sums); err != nil {
		return fmt.Errorf("parse summaries.json: %w", err)
	}

	cornerSumsMap := make(map[string][]model.Summary, len(sums.Corners))
	for _, cs := range sums.Corners {
		cornerSumsMap[cs.CornerTitle] = cs.Summaries
	}

	w := write.NewLLMWriter(c, prompts["write"], stepTemp(llmCfg, "write"))
	allLines := make([]model.Line, 0)
	for _, corner := range p.Corners {
		cornerSums := cornerSumsMap[corner.Title]
		lines, err := w.Write(ctx, corner, cornerSums, chars)
		if err != nil {
			return fmt.Errorf("write corner %q: %w", corner.Title, err)
		}
		allLines = append(allLines, lines...)
	}

	outPath := filepath.Join(workDir, "lines.json")
	if err := writeJSON(outPath, model.Lines{Lines: allLines}); err != nil {
		return err
	}
	fmt.Printf("wrote %d lines to %s\n", len(allLines), outPath)
	return nil
}

func runScriptDirect(ctx context.Context, workDir, out string, c llm.Client, llmCfg config.LLMConfig, prompts map[string]string, seCatalog model.SECatalog) error {
	linesPath := filepath.Join(workDir, "lines.json")
	data, err := os.ReadFile(linesPath)
	if err != nil {
		return fmt.Errorf("read lines.json: %w", err)
	}
	var linesWrapper model.Lines
	if err := json.Unmarshal(data, &linesWrapper); err != nil {
		return fmt.Errorf("parse lines.json: %w", err)
	}

	d := direct.NewLLMDirector(c, prompts["direct"], stepTemp(llmCfg, "direct"))
	scr, err := d.Direct(ctx, linesWrapper.Lines, seCatalog)
	if err != nil {
		return fmt.Errorf("direct: %w", err)
	}

	if err := writeJSON(out, scr); err != nil {
		return err
	}
	fmt.Printf("directed %d segments to %s\n", len(scr.Segments), out)
	return nil
}

func loadPrompts(dir string) (map[string]string, error) {
	names := []string{"summarize", "write", "direct"}
	prompts := make(map[string]string, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name+".md"))
		if err != nil {
			return nil, fmt.Errorf("read %s.md: %w", name, err)
		}
		prompts[name] = string(data)
	}
	return prompts, nil
}

func buildSECatalog(assets config.AssetsConfig) model.SECatalog {
	names := make([]string, 0, len(assets.SE))
	for name := range assets.SE {
		names = append(names, name)
	}
	sort.Strings(names)
	return model.SECatalog{Names: names}
}

func stepTemp(llmCfg config.LLMConfig, name string) float64 {
	if s, ok := llmCfg.Steps[name]; ok && s.Temperature != nil {
		return *s.Temperature
	}
	return 0
}

func readArticles(path string) (model.Articles, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Articles{}, fmt.Errorf("read %s: %w", path, err)
	}
	var articles model.Articles
	if err := json.Unmarshal(data, &articles); err != nil {
		return model.Articles{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return articles, nil
}
