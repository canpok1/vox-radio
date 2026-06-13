package cli

import (
	"embed"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed skills
var skillsFS embed.FS

const defaultSkillsDir = ".claude/skills"

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
			return runInstallSkills(cmd, skillsDir, force)
		},
	}

	cmd.Flags().BoolVar(&installSkillsFlag, "skills", false, "エージェントスキルを <skills-dir>/vox-radio/ にインストールする")
	cmd.Flags().BoolVar(&force, "force", false, "既存ファイルを上書きする")
	cmd.Flags().StringVar(&skillsDir, "skills-dir", defaultSkillsDir, "スキルのインストール先ディレクトリ（このディレクトリ下に vox-radio/ を作成する）")

	return cmd
}

func runInstallSkills(cmd *cobra.Command, skillsDir string, force bool) error {
	const srcDir = "skills/vox-radio"
	dstDir := filepath.Join(skillsDir, "vox-radio")

	return writeEmbeddedTree(cmd, skillsFS, srcDir, dstDir, force)
}
