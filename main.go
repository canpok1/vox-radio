package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-radio/internal/assemble"
	"github.com/canpok1/vox-radio/internal/collect"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/synth"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: vox-radio <command>")
		fmt.Fprintln(os.Stderr, "Commands: collect, synth, assemble")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "collect":
		if err := runCollect(os.Args[2:]); err != nil {
			log.Fatalf("collect: %v", err)
		}
	case "synth":
		if err := runSynth(os.Args[2:]); err != nil {
			log.Fatalf("synth: %v", err)
		}
	case "assemble":
		if err := runAssemble(os.Args[2:]); err != nil {
			log.Fatalf("assemble: %v", err)
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
