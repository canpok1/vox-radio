package cli

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//go:embed templates/vox-radio.yaml
var configTemplate []byte

//go:embed templates/profile.yaml
var profileTemplate []byte

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Generate template config files in the current directory",
		Long: `Generate vox-radio.yaml (common settings) and profile.yaml (program profile)
in the current directory.

Existing files are skipped individually to prevent accidental overwrites.
If both files already exist, nothing is generated.

After generation, edit the files to configure your LLM API key, program
content, and audio asset paths, then run the pipeline:

  vox-radio run --profile profile.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := generateFile(cmd, "vox-radio.yaml", configTemplate); err != nil {
				return err
			}
			return generateFile(cmd, "profile.yaml", profileTemplate)
		},
	}
}

func generateFile(cmd *cobra.Command, path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "skip: %s already exists\n", path)
		return nil
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created: %s\n", path)
	return nil
}
