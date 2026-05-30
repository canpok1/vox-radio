package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/synth"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: vox-radio <command>")
		fmt.Fprintln(os.Stderr, "Commands: synth")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "synth":
		if err := runSynth(os.Args[2:]); err != nil {
			log.Fatalf("synth: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
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
