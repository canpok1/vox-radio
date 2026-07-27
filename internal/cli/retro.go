package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/retro"
	"github.com/canpok1/vox-radio/internal/script/write"
	"github.com/spf13/cobra"
)

// retroTryItems converts a loaded try file's problems into write.RetroTryItem for injection.
// Returns nil when there is nothing to inject: a missing try file (LoadTryFile already yields an
// empty Problems slice for that case) or a try file whose problems have all been resolved/removed.
// This is how injection stops without a dedicated on/off flag (ADR-0098).
func retroTryItems(tf retro.TryFile) []write.RetroTryItem {
	if len(tf.Problems) == 0 {
		return nil
	}
	items := make([]write.RetroTryItem, len(tf.Problems))
	for i, p := range tf.Problems {
		items[i] = write.RetroTryItem{Problem: p.Problem, Action: p.Action}
	}
	return items
}

// retroKeepItems converts a loaded keep file's entries into write.RetroTryItem for injection.
// Returns nil when there is nothing to inject: a missing keep file (LoadKeepFile already yields an
// empty Keeps slice for that case) or an empty keep file.
func retroKeepItems(kf retro.KeepFile) []write.RetroTryItem {
	if len(kf.Keeps) == 0 {
		return nil
	}
	items := make([]write.RetroTryItem, len(kf.Keeps))
	for i, k := range kf.Keeps {
		items[i] = write.RetroTryItem{Problem: k.Problem, Action: k.Action}
	}
	return items
}

func newRetroCmd() *cobra.Command {
	var specPath string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "retro",
		Short: "過去回の分析から課題と施策を抽出し try ファイルを更新する（自動改善ループ）",
		Long: `蓄積された過去回の分析（analyze の出力）から、反復して現れる課題を見つけ、
次に試す施策と組にして .vox-radio/programs/{program.id}/try.yaml へ記録します。

問題が retro.keep_threshold 回連続で再発しなければ、その施策は実証済みとして
.vox-radio/programs/{program.id}/keep.yaml へ昇格し、try からは外れて常に適用されます。
keep の問題が再発した場合は try へ降格します。

try/keep ファイルの内容は episodegen 実行時に write（台本生成）プロンプトへ自動的に注入されます。
retro を実行しなくても、既存の try/keep ファイルは注入され続けます。適用を止めたいときは
該当するファイルを削除してください。

retro は毎回 try ファイルを全置換します（個別項目の承認は行いません）。恒久的に固定したい
方針は episode-spec.yaml の script_note に書いてください。keep が増えすぎた場合も script_note へ
移し keep から削除することを検討してください。

共通設定ファイルのパスは --config フラグで指定します（省略時は vox-radio.yaml）。

例:
  vox-radio retro --spec episode-spec.yaml
  vox-radio retro --spec episode-spec.yaml --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, logFile, err := setupLogger("retro", logDirFlag(cmd))
			if err != nil {
				return fmt.Errorf("setup logger: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			cfg, p, err := loadConfigAndSpec(configPath(cmd), specPath)
			if err != nil {
				return err
			}

			if err := checkResources(func() error { return requireLLMKey(cfg) }); err != nil {
				return err
			}

			if p.Program.SingleShot {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "single_shot 番組はキャッシュに保存しないため retro を実行できません")
				return nil
			}

			entries, _, err := loadCacheEntries(p.Program.ID)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "キャッシュが存在しません。episodegen を実行してから retro を実行してください")
				return nil
			}

			recent := cache.Recent(entries, cfg.Retro.EffectiveAnalysisEntries())
			analyzed := retro.FilterAnalyzed(recent)
			if len(analyzed) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "分析付きのエピソードがありません。episodegen を実行して分析を蓄積してください")
				return nil
			}

			tryPath := programTryPath(p.Program.ID)
			tf, err := retro.LoadTryFile(tryPath)
			if err != nil {
				return err
			}
			keepPath := programKeepPath(p.Program.ID)
			kf, err := retro.LoadKeepFile(keepPath)
			if err != nil {
				return err
			}

			llmClient := newLLMClient(cfg)
			prompts, err := loadPrompts()
			if err != nil {
				return fmt.Errorf("load prompts: %w", err)
			}

			r := retro.NewLLMRetro(llmClient, prompts["retro"], stepTemp(cfg.LLM, "retro"), retro.WithLogger(logger))
			proposed, recurrences, lastEvaluated, err := r.Run(context.Background(), p.Program, analyzed, tf.Problems, kf.Keeps, cfg.Retro.EffectiveMaxTries())
			if err != nil {
				return fmt.Errorf("retro: %w", err)
			}

			newEpisodes := retro.NewEpisodeNumbers(analyzed, tf.LastEvaluatedEpisode)
			result := retro.ApplyCounts(retro.ApplyCountsInput{
				PrevTryProblems:  tf.Problems,
				PrevKeeps:        kf.Keeps,
				ProposedProblems: proposed,
				Recurrences:      recurrences,
				NewEpisodes:      newEpisodes,
				KeepThreshold:    cfg.Retro.EffectiveKeepThreshold(),
				MaxTries:         cfg.Retro.EffectiveMaxTries(),
				LatestEpisode:    lastEvaluated,
			})

			newTF := retro.TryFile{
				GeneratedAt:          time.Now().UTC().Format(time.RFC3339),
				LastEvaluatedEpisode: lastEvaluated,
				Problems:             result.NextTry,
			}
			newKF := retro.KeepFile{
				GeneratedAt: time.Now().UTC().Format(time.RFC3339),
				Keeps:       result.NextKeep,
			}

			if len(result.PromotedIDs) > 0 {
				logger.Info("keepへ昇格", "ids", result.PromotedIDs)
			}
			if len(result.DemotedIDs) > 0 {
				logger.Info("tryへ降格", "ids", result.DemotedIDs)
			}

			keepLength := cfg.Retro.EffectiveKeepLength()
			if n := retro.KeepContentLength(newKF); n > keepLength {
				logger.Warn("keepの分量が上限を超えています。episode-spec.yaml の script_note へ移すことを検討してください", "length", n, "limit", keepLength)
			}

			if dryRun {
				tryContent, err := retro.MarshalTryFile(newTF)
				if err != nil {
					return err
				}
				keepContent, err := retro.MarshalKeepFile(newKF)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprint(cmd.OutOrStdout(), string(tryContent))
				_, _ = fmt.Fprint(cmd.OutOrStdout(), string(keepContent))
				return nil
			}

			if err := retro.SaveTryFile(tryPath, newTF); err != nil {
				return err
			}
			if err := retro.SaveKeepFile(keepPath, newKF); err != nil {
				return err
			}
			logger.Info("try/keepファイルを更新", "try_problems", len(newTF.Problems), "keeps", len(newKF.Keeps))
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "try file written to %s (%d problems)\n", tryPath, len(newTF.Problems))
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "keep file written to %s (%d keeps)\n", keepPath, len(newKF.Keeps))
			return nil
		},
	}

	registerSpecFlag(cmd, &specPath)
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "try ファイルを書き込まず標準出力に出力する")

	return cmd
}
