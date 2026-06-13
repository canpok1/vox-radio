package cli

import (
	"embed"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed skills
var skillsFS embed.FS

func newInstallCmd() *cobra.Command {
	var installSkillsFlag bool
	var force bool
	var skillsDir string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "vox-radio のエージェントスキルやリソースをインストールする",
		Long: `vox-radio のエージェントスキルやリソースを現在のプロジェクトへインストールします。

--skills フラグを指定すると、LLM エージェント向けのスキルファイル一式を
<skills-dir>/vox-radio/ 配下にインストールします（既定: .claude/skills/vox-radio/）。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !installSkillsFlag {
				return cmd.Help()
			}
			return runInstallSkills(cmd, filepath.Join(skillsDir, "vox-radio"), force)
		},
	}

	cmd.Flags().BoolVar(&installSkillsFlag, "skills", false, "エージェントスキルを <skills-dir>/vox-radio/ にインストールする")
	cmd.Flags().BoolVar(&force, "force", false, "既存ファイルを上書きする")
	cmd.Flags().StringVar(&skillsDir, "skills-dir", ".claude/skills", "スキルのインストール先ディレクトリ（このディレクトリ下に vox-radio/ を作成する）")

	return cmd
}

func runInstallSkills(cmd *cobra.Command, dstDir string, force bool) error {
	return writeEmbeddedTree(cmd, skillsFS, "skills/vox-radio", dstDir, force)
}
