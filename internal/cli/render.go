package cli

import (
	"fmt"
	"os"

	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/render"
	"github.com/spf13/cobra"
)

func newRenderCmd() *cobra.Command {
	var manifestPath string
	var templatePath string
	var templateString string
	var outputPath string

	refURL := referenceURL("internal/cli/skills/vox-radio/references/manifest.md")

	cmd := &cobra.Command{
		Use:   "render",
		Short: "manifest を text/template でレンダリングして出力する",
		Long: fmt.Sprintf(`manifest.json と text/template を入力に、レンダリング結果を標準出力（または --output ファイル）へ書き出します。

テンプレートはファイル（--template）またはインライン文字列（--template-string）で指定します。両方の同時指定は不可。

よく使うトップレベルフィールド:
  .Title          — 番組タイトル
  .EpisodeNumber  — 回番号（int）
  .EpisodeTitle   — サブタイトル
  .AudioFile      — 音声ファイル名
  .Summary        — 全体要約
  .Datetime       — 配信日時
  .Author         — 著者

テンプレート関数:
  corner "<id>"     — 指定 ID のコーナーを返す（見つからない場合は nil）
  hasLinks <corner> — コーナーに URL 付き記事が 1 件以上あれば true

全フィールド・コーナー・関数の一覧:
  %s

例（ファイル指定）:
  vox-radio render --manifest output/manifest.json --template release-note.tmpl

例（インライン指定・CI での値抽出）:
  vox-radio render --manifest output/manifest.json --template-string '{{.EpisodeNumber}}'
  vox-radio render --manifest output/manifest.json --template-string '第{{.EpisodeNumber}}回 {{.EpisodeTitle}}'`, refURL),
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest, err := readJSON[model.Manifest](manifestPath)
			if err != nil {
				return fmt.Errorf("load manifest: %w", err)
			}

			var tmplText string
			if cmd.Flags().Changed("template-string") {
				tmplText = templateString
			} else {
				tmplBytes, err := os.ReadFile(templatePath)
				if err != nil {
					return fmt.Errorf("load template: %w", err)
				}
				tmplText = string(tmplBytes)
			}

			result, err := render.Render(tmplText, manifest)
			if err != nil {
				return fmt.Errorf("render: %w", err)
			}

			if outputPath != "" {
				if err := os.WriteFile(outputPath, []byte(result), 0o644); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
				return nil
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), result)
			return err
		},
	}

	cmd.Flags().StringVar(&manifestPath, "manifest", "", "manifest.json ファイルのパス（必須）")
	cmd.Flags().StringVar(&templatePath, "template", "", "text/template ファイルのパス（--template-string と排他）")
	cmd.Flags().StringVar(&templateString, "template-string", "", "テンプレート文字列（--template と排他、CI での値抽出に便利）")
	cmd.Flags().StringVar(&outputPath, "output", "", "出力先ファイルのパス（省略時は標準出力）")
	_ = cmd.MarkFlagRequired("manifest")
	cmd.MarkFlagsMutuallyExclusive("template", "template-string")
	cmd.MarkFlagsOneRequired("template", "template-string")

	return cmd
}
