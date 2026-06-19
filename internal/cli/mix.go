package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/mix"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/spf13/cobra"
)

func newMixCmd() *cobra.Command {
	var in string
	var clipsDir string
	var out string
	var specPath string

	refURL := referenceURL("README.md")
	cmd := &cobra.Command{
		Use:   "mix",
		Short: "WAV クリップを MP3 エピソードに組み立てる",
		Long: fmt.Sprintf(`script.json と synth が生成したクリップディレクトリを読み込み、
ffmpeg を使ってイントロ・アウトロ・SE をミックスし、最終的な MP3 エピソードを生成します。

実行には ffmpeg および ffprobe が必要です。インストール手順は vox-radio の README を参照してください:
%s

例:
  vox-radio episodegen mix --in work/script.json --clips work/clips --out work/episode.mp3
  vox-radio episodegen mix --in work/script.json --clips work/clips --out work/episode.mp3 --spec episode-spec.yaml`, refURL),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("mix", logDirFlag(cmd))
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			if err := requireMediaTools(); err != nil {
				return err
			}

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
			var program config.ProgramConfig
			if specPath != "" {
				p, err := config.LoadEpisodeSpec(specPath)
				if err != nil {
					return fmt.Errorf("load spec: %w", err)
				}
				assetsConfig = p.Assets
				program = p.Program
			}

			a := mix.New(assetsConfig, program, mix.WithLogger(logger), mix.WithFFmpegWriter(logFile))
			result, err := a.Run(context.Background(), scr, clips, clipsDir, out, model.EpisodeMeta{})
			if err != nil {
				return err
			}

			fmt.Printf("assembled episode: duration=%.1fs, bytes=%d\n", result.DurationSec, result.Bytes)
			return nil
		},
	}

	cmd.Flags().StringVar(&in, "in", "", "script.json の入力パス（必須）")
	cmd.Flags().StringVar(&clipsDir, "clips", "", "clips.json と WAV ファイルを含むディレクトリ（必須）")
	cmd.Flags().StringVar(&out, "out", "", "MP3 の出力先パス（必須）")
	cmd.Flags().StringVar(&specPath, "spec", "", "アセット設定を含むエピソード仕様 YAML ファイルのパス（任意）")
	_ = cmd.MarkFlagRequired("in")
	_ = cmd.MarkFlagRequired("clips")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
