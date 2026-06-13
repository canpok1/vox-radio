---
paths:
  - "main.go"
  - "internal/cli/**"
---

# CLIドキュメント反映ルール

`main.go` / `internal/cli/**` を変更したとき、またはサブコマンド・フラグ・引数・挙動を追加/変更/削除したときは、以下をすべて実施してコミットに含めること。

1. cobra コマンドの `Short` / `Long` / フラグ usage 文言を更新する
2. `make docs`（`go run ./tools/gendocs`）を実行して `docs/cli/` を再生成し、差分をコミットに含める
3. ルート README の CLI 概要・パイプライン記述に変更を反映する

## サブコマンドを削除した場合の追加手順

サブコマンドを削除したときは、上記の共通手順に加えて以下を実施すること。

4. `make docs` は削除したコマンドのドキュメントを自動削除しないため、`git rm docs/cli/{command}.md` を手動で実行する
5. `grep -rn "{command}" internal/cli/` で `internal/cli/*_test.go` のサブコマンドリスト（`TestRootHelp` / `TestSubcommandHelp` 等）に削除コマンドへの参照が残っていないか確認し、残っていれば修正する
6. `root.go` の親コマンドへの登録（`AddCommand`）と `Long` 説明文から削除コマンドの記述を除去する
