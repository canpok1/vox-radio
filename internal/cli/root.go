package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "vox-radio",
		Short: "AI-powered podcast production tool",
		Long: `vox-radio is a CLI tool for producing AI-generated podcast episodes.

It covers the full pipeline: collecting articles, generating scripts via LLM,
synthesizing voice clips, assembling audio, and publishing RSS feeds.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newCollectCmd(),
		newScriptCmd(),
		newSynthCmd(),
		newAssembleCmd(),
		newPublishCmd(),
		newPruneCmd(),
	)

	return root
}

func Execute() error {
	return NewRootCmd().Execute()
}
