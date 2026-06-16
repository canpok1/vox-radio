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
<skills-dir>/vox-radio/ 配下にインストールします（既定: .claude/skills/vox-radio/）。

このとき、インストール元バイナリのバージョンを <skills-dir>/vox-radio/.skill-version に
記録します（スキルとバイナリの版ずれ検知に使用）。.skill-version は生成ファイルのため
--force の有無に関わらず常に最新バージョンで上書きされます。`,
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
	if err := writeEmbeddedTree(cmd, skillsFS, "skills/vox-radio", dstDir, force, nil); err != nil {
		return err
	}
	// 版スタンプはインストール元バイナリの版を記録する生成ファイル。
	// スキルがバイナリとの版ずれを検知できるよう、--force の有無に関わらず常に上書きする。
	return writeFile(cmd, filepath.Join(dstDir, ".skill-version"), []byte(version+"\n"), true)
}
