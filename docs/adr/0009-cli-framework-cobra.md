# 0009. CLIフレームワークにcobraを採用

- ステータス: 採用
- 日付: 2026-05-30

## コンテキスト

`main.go` は標準 `flag` パッケージ + `os.Args[1]` の手書きディスパッチで実装されていた。この構造では `vox-radio --help` / `-h` が `unknown command: --help` となり、トップレベルの help が機能しない。また各サブコマンドの usage を手書きする負担が大きく、新コマンド追加のたびにディスパッチと usage の両方を書く必要があった。

後続 Issue #33 では cobra/doc を用いたドキュメント自動生成も予定しており、CLIフレームワーク採用が共通の土台となる。

## 決定

`github.com/spf13/cobra` を採用し、`internal/cli/` パッケージ内にルートコマンドと各サブコマンドを実装する。`RunE` + `SilenceUsage = true` / `SilenceErrors = true` を設定し、実行時エラーで usage が無駄に出ないようにする。必須フラグは `MarkFlagRequired` で明示する。

## 結果

- `vox-radio --help` でルート help に全コマンド一覧が自動生成される
- `vox-radio <command> --help` で各フラグ説明が自動生成される
- 新コマンド追加時に help の手書きが不要になる
- cobra/doc による Markdown ドキュメント自動生成（Issue #33）の土台が整う
- 依存が増える（cobra + pflag）がいずれも広く使われる安定ライブラリ

## 検討した代替案

**flag維持（現状のまま）**: トップレベル help の欠如と usage 手書き負担が解消されず、Issue #33 の土台も作れない。却下。

**urfave/cli**: cobra と同等の機能を持つが、cobra のほうが Go エコシステムで採用実績が多く、cobra/doc など周辺ツールが充実している。kubectl・gh など主要 CLI も cobra を採用しており、ドキュメントが豊富。却下。
