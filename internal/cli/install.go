package cli

import (
	"embed"
	"fmt"
	"io/fs"
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
		content, err := skillsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		return writeFile(cmd, filepath.Join(dstDir, rel), content, force)
	})
}
