package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/canpok1/vox-radio/internal/assemble"
	"github.com/canpok1/vox-radio/internal/collect"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/publish"
	"github.com/canpok1/vox-radio/internal/publish/hosting/local"
	"github.com/canpok1/vox-radio/internal/script"
	"github.com/canpok1/vox-radio/internal/script/direct"
	"github.com/canpok1/vox-radio/internal/script/llm"
	"github.com/canpok1/vox-radio/internal/script/plan"
	"github.com/canpok1/vox-radio/internal/script/summarize"
	"github.com/canpok1/vox-radio/internal/script/write"
	"github.com/canpok1/vox-radio/internal/synth"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: vox-radio <command>")
		fmt.Fprintln(os.Stderr, "Commands: collect, script, synth, assemble, publish, prune")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "collect":
		if err := runCollect(os.Args[2:]); err != nil {
			log.Fatalf("collect: %v", err)
		}
	case "script":
		if err := runScript(os.Args[2:]); err != nil {
			log.Fatalf("script: %v", err)
		}
	case "synth":
		if err := runSynth(os.Args[2:]); err != nil {
			log.Fatalf("synth: %v", err)
		}
	case "assemble":
		if err := runAssemble(os.Args[2:]); err != nil {
			log.Fatalf("assemble: %v", err)
		}
	case "publish":
		if err := runPublish(os.Args[2:]); err != nil {
			log.Fatalf("publish: %v", err)
		}
	case "prune":
		if err := runPrune(os.Args[2:]); err != nil {
			log.Fatalf("prune: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runCollect(args []string) error {
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	configDir := fs.String("config", "config", "config directory containing feeds.yaml (default: config)")
	out := fs.String("out", "", "output articles.json path (required)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: vox-radio collect --out <articles.json> [--config <config_dir>]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		fs.Usage()
		return fmt.Errorf("--out is required")
	}

	cfg, err := config.Load(*configDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	c := collect.New(nil)
	articles, err := c.Run(context.Background(), cfg.Feeds)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	data, err := json.MarshalIndent(articles, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal articles: %w", err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		return fmt.Errorf("write articles: %w", err)
	}

	fmt.Printf("collected %d articles to %s\n", len(articles.Articles), *out)
	return nil
}

func runSynth(args []string) error {
	fs := flag.NewFlagSet("synth", flag.ContinueOnError)
	in := fs.String("in", "", "input script.json path (required)")
	outDir := fs.String("out-dir", "", "output directory for WAV clips (required)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: vox-radio synth --in <script.json> --out-dir <clips>")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *in == "" {
		fs.Usage()
		return fmt.Errorf("--in is required")
	}
	if *outDir == "" {
		fs.Usage()
		return fmt.Errorf("--out-dir is required")
	}

	data, err := os.ReadFile(*in)
	if err != nil {
		return fmt.Errorf("read script: %w", err)
	}
	var script model.Script
	if err := json.Unmarshal(data, &script); err != nil {
		return fmt.Errorf("parse script: %w", err)
	}

	showConfig := model.ShowConfig{
		DefaultSpeaker: 3,
		Speakers:       map[string]int{},
	}

	engineURL := os.Getenv("VOICEVOX_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:50021"
	}

	s := synth.New(engineURL, showConfig)
	meta, err := s.Run(context.Background(), script, *outDir)
	if err != nil {
		return err
	}

	fmt.Printf("synthesized %d clips to %s\n", len(meta.Clips), *outDir)
	return nil
}

func runAssemble(args []string) error {
	fs := flag.NewFlagSet("assemble", flag.ContinueOnError)
	in := fs.String("in", "", "input script.json path (required)")
	clipsDir := fs.String("clips", "", "directory containing clips.json and WAV files (required)")
	out := fs.String("out", "", "output mp3 path (required)")
	configDir := fs.String("config", "", "config directory for assets (optional)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: vox-radio assemble --in <script.json> --clips <dir> --out <mp3>")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *in == "" {
		fs.Usage()
		return fmt.Errorf("--in is required")
	}
	if *clipsDir == "" {
		fs.Usage()
		return fmt.Errorf("--clips is required")
	}
	if *out == "" {
		fs.Usage()
		return fmt.Errorf("--out is required")
	}

	scriptData, err := os.ReadFile(*in)
	if err != nil {
		return fmt.Errorf("read script: %w", err)
	}
	var script model.Script
	if err := json.Unmarshal(scriptData, &script); err != nil {
		return fmt.Errorf("parse script: %w", err)
	}

	clipsData, err := os.ReadFile(filepath.Join(*clipsDir, "clips.json"))
	if err != nil {
		return fmt.Errorf("read clips.json: %w", err)
	}
	var clips model.ClipsMeta
	if err := json.Unmarshal(clipsData, &clips); err != nil {
		return fmt.Errorf("parse clips.json: %w", err)
	}

	var assetsConfig config.AssetsConfig
	var showConfig model.ShowConfig
	if *configDir != "" {
		cfg, err := config.Load(*configDir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		assetsConfig = cfg.Assets
		showConfig = cfg.Show
	} else {
		showConfig = model.ShowConfig{SegmentPauseSec: 0.3}
	}

	a := assemble.New(assetsConfig, showConfig)
	result, err := a.Run(context.Background(), script, clips, *clipsDir, *out)
	if err != nil {
		return err
	}

	fmt.Printf("assembled episode: duration=%.1fs, bytes=%d\n", result.DurationSec, result.Bytes)
	return nil
}

func runPublish(args []string) error {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	in := fs.String("in", "", "input mp3 path (required)")
	date := fs.String("date", "", "episode date YYYY-MM-DD (default: today)")
	titleFlag := fs.String("title", "", "episode title (default: <date> <podcast.title>)")
	descFlag := fs.String("description", "", "episode description")
	configDir := fs.String("config", "config", "config directory containing podcast.yaml")
	outDir := fs.String("out-dir", "", "output directory for local hosting (required)")
	baseURL := fs.String("base-url", "", "base URL for audio/feed URLs (default: site_url from podcast.yaml)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: vox-radio publish --in <mp3> --out-dir <dir> [--date <YYYY-MM-DD>] [options]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *in == "" {
		fs.Usage()
		return fmt.Errorf("--in is required")
	}
	if *outDir == "" {
		fs.Usage()
		return fmt.Errorf("--out-dir is required")
	}

	cfg, err := config.Load(*configDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	h := local.New(*outDir, resolveSiteURL(*baseURL, cfg.Podcast.SiteURL))
	publisher := publish.New(h, cfg.Podcast)

	opts := publish.Options{
		Date:        *date,
		Title:       *titleFlag,
		Description: *descFlag,
	}

	if err := publisher.Run(context.Background(), *in, opts); err != nil {
		return err
	}

	effectiveDate := *date
	if effectiveDate == "" {
		effectiveDate = "(today)"
	}
	fmt.Printf("published episode for %s to %s\n", effectiveDate, *outDir)
	return nil
}

func runPrune(args []string) error {
	fs := flag.NewFlagSet("prune", flag.ContinueOnError)
	outDir := fs.String("out-dir", "", "output directory for local hosting (required)")
	configDir := fs.String("config", "config", "config directory containing podcast.yaml")
	baseURL := fs.String("base-url", "", "base URL for audio/feed URLs (default: site_url from podcast.yaml)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: vox-radio prune --out-dir <dir> [--config <dir>] [--base-url <url>]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outDir == "" {
		fs.Usage()
		return fmt.Errorf("--out-dir is required")
	}

	cfg, err := config.Load(*configDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	keep := cfg.Podcast.MaxItems
	if keep <= 0 {
		keep = publish.DefaultKeep
	}

	h := local.New(*outDir, resolveSiteURL(*baseURL, cfg.Podcast.SiteURL))
	pruner := publish.NewPruner(h, keep)

	if err := pruner.Run(context.Background()); err != nil {
		return err
	}

	fmt.Printf("pruned to %d episodes in %s\n", keep, *outDir)
	return nil
}

func resolveSiteURL(override, configURL string) string {
	if override != "" {
		return override
	}
	return configURL
}

func runScript(args []string) error {
	fs := flag.NewFlagSet("script", flag.ContinueOnError)
	in := fs.String("in", "", "input articles.json path (required for full pipeline or summarize step)")
	out := fs.String("out", "", "output script.json path (required)")
	step := fs.String("step", "", "run a single step: summarize|plan|write|direct")
	configDir := fs.String("config", "config", "config directory containing llm.yaml, show.yaml, assets.yaml")
	promptsDir := fs.String("prompts", "prompts", "directory containing prompt templates")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: vox-radio script --out <script.json> [--in <articles.json>] [--step summarize|plan|write|direct]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		fs.Usage()
		return fmt.Errorf("--out is required")
	}

	cfg, err := config.Load(*configDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	apiKey := os.Getenv(cfg.LLM.APIKeyEnv)
	llmClient := llm.NewClient(llm.Config{
		BaseURL:     cfg.LLM.BaseURL,
		APIKey:      apiKey,
		Model:       cfg.LLM.Model,
		Temperature: cfg.LLM.Temperature,
		MaxRetries:  cfg.LLM.MaxRetries,
	})

	prompts, err := loadPrompts(*promptsDir)
	if err != nil {
		return fmt.Errorf("load prompts: %w", err)
	}

	workDir := filepath.Dir(*out)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}

	seCatalog := buildSECatalog(cfg.Assets)

	switch *step {
	case "":
		return runScriptFull(context.Background(), *in, *out, workDir, llmClient, cfg, prompts, seCatalog)
	case "summarize":
		return runScriptSummarize(context.Background(), *in, workDir, llmClient, cfg, prompts)
	case "plan":
		return runScriptPlan(context.Background(), workDir, llmClient, cfg, prompts)
	case "write":
		return runScriptWrite(context.Background(), workDir, llmClient, cfg, prompts)
	case "direct":
		return runScriptDirect(context.Background(), workDir, *out, llmClient, cfg, prompts, seCatalog)
	default:
		return fmt.Errorf("unknown step %q: use summarize|plan|write|direct", *step)
	}
}

func runScriptFull(ctx context.Context, in, out, workDir string, c llm.Client, cfg *config.Config, prompts map[string]string, seCatalog model.SECatalog) error {
	if in == "" {
		return fmt.Errorf("--in is required for full pipeline")
	}
	articles, err := readArticles(in)
	if err != nil {
		return err
	}

	gen := script.NewLLMScriptGenerator(
		summarize.NewLLMSummarizer(c, prompts["summarize"], stepTemp(cfg, "summarize")),
		plan.NewLLMPlanner(c, prompts["plan"], stepTemp(cfg, "plan")),
		write.NewLLMWriter(c, prompts["write"], stepTemp(cfg, "write")),
		direct.NewLLMDirector(c, prompts["direct"], stepTemp(cfg, "direct")),
		seCatalog,
		workDir,
	)

	scr, err := gen.Generate(ctx, articles.Articles, cfg.Show)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	return writeJSON(out, scr)
}

func runScriptSummarize(ctx context.Context, in, workDir string, c llm.Client, cfg *config.Config, prompts map[string]string) error {
	if in == "" {
		return fmt.Errorf("--in is required for summarize step")
	}
	articles, err := readArticles(in)
	if err != nil {
		return err
	}

	s := summarize.NewLLMSummarizer(c, prompts["summarize"], stepTemp(cfg, "summarize"))
	summaries := make([]model.Summary, 0, len(articles.Articles))
	for _, a := range articles.Articles {
		sum, err := s.Summarize(ctx, a)
		if err != nil {
			return fmt.Errorf("summarize %q: %w", a.URL, err)
		}
		summaries = append(summaries, sum)
	}

	out := filepath.Join(workDir, "summaries.json")
	if err := writeJSON(out, model.Summaries{Summaries: summaries}); err != nil {
		return err
	}
	fmt.Printf("summarized %d articles to %s\n", len(summaries), out)
	return nil
}

func runScriptPlan(ctx context.Context, workDir string, c llm.Client, cfg *config.Config, prompts map[string]string) error {
	summariesPath := filepath.Join(workDir, "summaries.json")
	data, err := os.ReadFile(summariesPath)
	if err != nil {
		return fmt.Errorf("read summaries.json: %w", err)
	}
	var sums model.Summaries
	if err := json.Unmarshal(data, &sums); err != nil {
		return fmt.Errorf("parse summaries.json: %w", err)
	}

	p := plan.NewLLMPlanner(c, prompts["plan"], stepTemp(cfg, "plan"))
	rundown, err := p.Plan(ctx, sums.Summaries, cfg.Show)
	if err != nil {
		return fmt.Errorf("plan: %w", err)
	}

	out := filepath.Join(workDir, "rundown.json")
	if err := writeJSON(out, rundown); err != nil {
		return err
	}
	fmt.Printf("planned %d corners to %s\n", len(rundown.Corners), out)
	return nil
}

func runScriptWrite(ctx context.Context, workDir string, c llm.Client, cfg *config.Config, prompts map[string]string) error {
	summariesPath := filepath.Join(workDir, "summaries.json")
	summariesData, err := os.ReadFile(summariesPath)
	if err != nil {
		return fmt.Errorf("read summaries.json: %w", err)
	}
	var sums model.Summaries
	if err := json.Unmarshal(summariesData, &sums); err != nil {
		return fmt.Errorf("parse summaries.json: %w", err)
	}

	rundownPath := filepath.Join(workDir, "rundown.json")
	rundownData, err := os.ReadFile(rundownPath)
	if err != nil {
		return fmt.Errorf("read rundown.json: %w", err)
	}
	var rundown model.Rundown
	if err := json.Unmarshal(rundownData, &rundown); err != nil {
		return fmt.Errorf("parse rundown.json: %w", err)
	}

	summaryByURL := script.SummaryByURL(sums.Summaries)

	w := write.NewLLMWriter(c, prompts["write"], stepTemp(cfg, "write"))
	allLines := make([]model.Line, 0)
	for _, corner := range rundown.Corners {
		relevant := script.CornerSummaries(corner, summaryByURL)
		lines, err := w.Write(ctx, corner, relevant, cfg.Show)
		if err != nil {
			return fmt.Errorf("write corner %q: %w", corner.Title, err)
		}
		allLines = append(allLines, lines...)
	}

	out := filepath.Join(workDir, "lines.json")
	if err := writeJSON(out, model.Lines{Lines: allLines}); err != nil {
		return err
	}
	fmt.Printf("wrote %d lines to %s\n", len(allLines), out)
	return nil
}

func runScriptDirect(ctx context.Context, workDir, out string, c llm.Client, cfg *config.Config, prompts map[string]string, seCatalog model.SECatalog) error {
	linesPath := filepath.Join(workDir, "lines.json")
	data, err := os.ReadFile(linesPath)
	if err != nil {
		return fmt.Errorf("read lines.json: %w", err)
	}
	var linesWrapper model.Lines
	if err := json.Unmarshal(data, &linesWrapper); err != nil {
		return fmt.Errorf("parse lines.json: %w", err)
	}

	d := direct.NewLLMDirector(c, prompts["direct"], stepTemp(cfg, "direct"))
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
	names := []string{"summarize", "plan", "write", "direct"}
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

func stepTemp(cfg *config.Config, name string) float64 {
	if s, ok := cfg.LLM.Steps[name]; ok && s.Temperature != nil {
		return *s.Temperature
	}
	// 0 causes the LLM client to fall back to its configured global temperature.
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

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
