# CLIドキュメント反映ルール

`main.go` / `internal/cli/**` を変更したとき、またはサブコマンド・フラグ・引数・挙動を追加/変更/削除したときは、以下をすべて実施してコミットに含めること。

1. cobra コマンドの `Short` / `Long` / フラグ usage 文言を更新する
2. `make docs`（`go run ./tools/gendocs`）を実行して `docs/cli/` を再生成し、差分をコミットに含める
3. ルート README の CLI 概要・パイプライン記述に変更を反映する
