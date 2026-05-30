package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-radio/internal/assemble"
	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/spf13/cobra"
)

func newAssembleCmd() *cobra.Command {
	var in string
	var clipsDir string
	var out string
	var configDir string

	cmd := &cobra.Command{
		Use:   "assemble",
		Short: "Assemble WAV clips into an MP3 episode",
		Long: `Read script.json and the clips directory produced by synth, then use ffmpeg
to mix intro/outro/SE and produce a final MP3 episode file.

Example:
  vox-radio assemble --in work/script.json --clips work/clips --out work/episode.mp3
  vox-radio assemble --in work/script.json --clips work/clips --out work/episode.mp3 --config config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			scriptData, err := os.ReadFile(in)
			if err != nil {
				return fmt.Errorf("read script: %w", err)
			}
			var scr model.Script
			if err := json.Unmarshal(scriptData, &scr); err != nil {
				return fmt.Errorf("parse script: %w", err)
			}

			clipsData, err := os.ReadFile(filepath.Join(clipsDir, "clips.json"))
			if err != nil {
				return fmt.Errorf("read clips.json: %w", err)
			}
			var clips model.ClipsMeta
			if err := json.Unmarshal(clipsData, &clips); err != nil {
				return fmt.Errorf("parse clips.json: %w", err)
			}

			var assetsConfig config.AssetsConfig
			var showConfig model.ShowConfig
			if configDir != "" {
				cfg, err := config.Load(configDir)
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}
				assetsConfig = cfg.Assets
				showConfig = cfg.Show
			} else {
				showConfig = model.ShowConfig{SegmentPauseSec: 0.3}
			}

			a := assemble.New(assetsConfig, showConfig)
			result, err := a.Run(context.Background(), scr, clips, clipsDir, out)
			if err != nil {
				return err
			}

			fmt.Printf("assembled episode: duration=%.1fs, bytes=%d\n", result.DurationSec, result.Bytes)
			return nil
		},
	}

	cmd.Flags().StringVar(&in, "in", "", "input script.json path (required)")
	cmd.Flags().StringVar(&clipsDir, "clips", "", "directory containing clips.json and WAV files (required)")
	cmd.Flags().StringVar(&out, "out", "", "output mp3 path (required)")
	cmd.Flags().StringVar(&configDir, "config", "", "config directory for assets (optional)")
	_ = cmd.MarkFlagRequired("in")
	_ = cmd.MarkFlagRequired("clips")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
