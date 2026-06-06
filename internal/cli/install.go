package cli

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed skills
var skillsFS embed.FS

func newInstallCmd() *cobra.Command {
	var installSkillsFlag bool
	var force bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "vox-radio のエージェントスキルやリソースをインストールする",
		Long: `vox-radio のエージェントスキルやリソースを現在のプロジェクトへインストールします。

--skills フラグを指定すると、LLM エージェント向けのスキルファイル一式を
.claude/skills/vox-radio/ 配下にインストールします。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !installSkillsFlag {
				return cmd.Help()
			}
			return runInstallSkills(cmd, force)
		},
	}

	cmd.Flags().BoolVar(&installSkillsFlag, "skills", false, "エージェントスキルを .claude/skills/vox-radio/ にインストールする")
	cmd.Flags().BoolVar(&force, "force", false, "既存ファイルを上書きする")

	return cmd
}

func runInstallSkills(cmd *cobra.Command, force bool) error {
	const srcDir = "skills/vox-radio"
	const dstDir = ".claude/skills/vox-radio"

	return fs.WalkDir(skillsFS, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("rel path: %w", err)
		}
		dst := filepath.Join(dstDir, rel)
		content, err := skillsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		return writeSkillFile(cmd, dst, content, force)
	})
}

func writeSkillFile(cmd *cobra.Command, path string, content []byte, force bool) error {
	if _, err := os.Stat(path); err == nil && !force {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "skip: %s already exists\n", path)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created: %s\n", path)
	return nil
}
