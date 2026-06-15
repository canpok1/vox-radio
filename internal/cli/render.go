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
	var outputPath string

	cmd := &cobra.Command{
		Use:   "render",
		Short: "manifest を text/template でレンダリングして出力する",
		Long: `manifest.json と text/template ファイルを入力に、レンダリング結果を標準出力（または --output ファイル）へ書き出します。

テンプレートのデータ文脈は manifest 全体です。以下のテンプレート関数が使えます:
  corner "<id>"  — 指定 ID のコーナーを返す（見つからない場合は nil）
  hasLinks <corner> — コーナーに URL 付き記事が 1 件以上あれば true

URL なし記事のスキップは {{if .URL}} でテンプレ側に表現できます。

例:
  vox-radio render --manifest output/manifest.json --template release-note.tmpl
  vox-radio render --manifest output/manifest.json --template release-note.tmpl --output RELEASE_NOTES.md`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest, err := readJSON[model.Manifest](manifestPath)
			if err != nil {
				return fmt.Errorf("load manifest: %w", err)
			}

			tmplBytes, err := os.ReadFile(templatePath)
			if err != nil {
				return fmt.Errorf("load template: %w", err)
			}

			result, err := render.Render(string(tmplBytes), manifest)
			if err != nil {
				return fmt.Errorf("render: %w", err)
			}

			if outputPath != "" {
				if err := writeStringToFile(outputPath, result); err != nil {
					return err
				}
				return nil
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), result)
			return err
		},
	}

	cmd.Flags().StringVar(&manifestPath, "manifest", "", "manifest.json ファイルのパス（必須）")
	cmd.Flags().StringVar(&templatePath, "template", "", "text/template ファイルのパス（必須）")
	cmd.Flags().StringVar(&outputPath, "output", "", "出力先ファイルのパス（省略時は標準出力）")
	_ = cmd.MarkFlagRequired("manifest")
	_ = cmd.MarkFlagRequired("template")

	return cmd
}

func writeStringToFile(path, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	_, writeErr := fmt.Fprint(f, content)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
