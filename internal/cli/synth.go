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
		Short: "台本から音声クリップを合成する",
		Long: `script.json を読み込み、VOICEVOX を呼び出して各台詞を WAV クリップに合成します。
出力ディレクトリには台詞ごとの WAV ファイルと clips.json マニフェストが格納されます。

共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。
voicevox.url フィールドで VOICEVOX エンジンの URL を指定します（デフォルト: http://localhost:50021）。
話者 ID は共通設定ファイルのキャラクターカタログから解決されます。

例:
  vox-radio episodegen synth --in work/script.json --out-dir work/clips`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("synth", "")
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			cfg, err := config.LoadConfig(configPath(cmd))
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

			engineURL := cfg.Voicevox.URL
			if engineURL == "" {
				engineURL = "http://localhost:50021"
			}

			s := synth.New(engineURL, cfg, synth.WithLogger(logger))
			meta, err := s.Run(context.Background(), scr, outDir)
			if err != nil {
				return err
			}

			fmt.Printf("synthesized %d clips to %s\n", len(meta.Clips), outDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&in, "in", "", "script.json の入力パス（必須）")
	cmd.Flags().StringVar(&outDir, "out-dir", "", "WAV クリップの出力ディレクトリ（必須）")
	_ = cmd.MarkFlagRequired("in")
	_ = cmd.MarkFlagRequired("out-dir")

	return cmd
}
