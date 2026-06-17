package cli

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// ldflags で上書きされる。未指定時は "dev"
var version = "dev"

// loadEnvFile は path の env ファイルを読み込む。
// explicit=false（デフォルト）のときはファイルが存在しなければ無視。
// explicit=true（--env-file 明示指定）のときはファイル不在でエラー。
// ファイルが存在し解析に失敗した場合は常にエラーを返す。
// 既存の OS 環境変数は上書きしない（godotenv.Load のデフォルト挙動）。
func loadEnvFile(path string, explicit bool) error {
	if explicit {
		return godotenv.Load(path)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return godotenv.Load(path)
}

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "vox-radio",
		Version: version,
		Short:   "AI を使ったポッドキャスト制作ツール",
		Long: `vox-radio は AI を活用したポッドキャストエピソード制作 CLI ツールです。

記事収集・LLM による台本生成・音声合成・音声組み立て・コンテンツマニフェスト出力まで、
フルパイプラインをカバーします。`,
		SilenceUsage:      true,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}

	root.PersistentFlags().String("config", DefaultConfigPath, "共通設定 YAML ファイル（vox-radio.yaml）のパス")
	root.PersistentFlags().String("log-dir", defaultLogDir, "ログ出力ディレクトリのパス")
	root.PersistentFlags().String("env-file", ".env", "環境変数を読み込む env ファイルのパス（未指定時は .env を自動読込、不在は無視）")

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		envFile, err := cmd.Flags().GetString("env-file")
		if err != nil {
			return err
		}
		return loadEnvFile(envFile, cmd.Flags().Changed("env-file"))
	}

	root.AddCommand(
		newInitCmd(),
		newInstallCmd(),
		newEpisodegenCmd(),
		newConfigCmd(),
		newFeedgenCmd(),
		newAssetsCmd(),
		newSlackpostCmd(),
		newRenderCmd(),
	)

	return root
}

func Execute() error {
	return NewRootCmd().Execute()
}
