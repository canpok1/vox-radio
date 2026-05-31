package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "vox-radio",
		Short: "AI を使ったポッドキャスト制作ツール",
		Long: `vox-radio は AI を活用したポッドキャストエピソード制作 CLI ツールです。

記事収集・LLM による台本生成・音声合成・音声組み立て・コンテンツマニフェスト出力まで、
フルパイプラインをカバーします。`,
		SilenceUsage:      true,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}

	root.AddCommand(
		newInitCmd(),
		newCollectCmd(),
		newScriptCmd(),
		newSynthCmd(),
		newAssembleCmd(),
		newManifestCmd(),
		newRunCmd(),
	)

	return root
}

func Execute() error {
	return NewRootCmd().Execute()
}
