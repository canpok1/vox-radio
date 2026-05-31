package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/synth"
	"github.com/spf13/cobra"
)

func newSynthCmd() *cobra.Command {
	var in string
	var outDir string

	cmd := &cobra.Command{
		Use:   "synth",
		Short: "Synthesize voice clips from a script",
		Long: `Read script.json and call VOICEVOX to synthesize each line into WAV clips.
The output directory will contain per-line WAV files and a clips.json manifest.

vox-radio.yaml is automatically loaded from the current directory.
The voicevox.url field specifies the VOICEVOX engine URL (default: http://localhost:50021).

Example:
  vox-radio synth --in work/script.json --out-dir work/clips`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig("vox-radio.yaml")
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			data, err := os.ReadFile(in)
			if err != nil {
				return fmt.Errorf("read script: %w", err)
			}
			var scr model.Script
			if err := json.Unmarshal(data, &scr); err != nil {
				return fmt.Errorf("parse script: %w", err)
			}

			showConfig := model.ShowConfig{
				DefaultSpeaker: 3,
				Speakers:       map[string]int{},
			}

			engineURL := cfg.Voicevox.URL
			if engineURL == "" {
				engineURL = "http://localhost:50021"
			}

			s := synth.New(engineURL, showConfig, cfg)
			meta, err := s.Run(context.Background(), scr, outDir)
			if err != nil {
				return err
			}

			fmt.Printf("synthesized %d clips to %s\n", len(meta.Clips), outDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&in, "in", "", "input script.json path (required)")
	cmd.Flags().StringVar(&outDir, "out-dir", "", "output directory for WAV clips (required)")
	_ = cmd.MarkFlagRequired("in")
	_ = cmd.MarkFlagRequired("out-dir")

	return cmd
}
