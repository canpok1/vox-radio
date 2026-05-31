---
name: cli-doc-review
description: main.go / internal/cli/** を変更したとき、CLIドキュメントの乖離を検出・修正するスキル。Short/Long文言・docs/cli/自動生成ドキュメント・READMEのCLIセクションを整合させてコミットする。
allowed-tools: Bash, Read, Grep, Glob, Edit
---

## このスキルの起動タイミング

- **手動実行**: ユーザーが `cli-doc-review` を明示的に呼んだとき。
- **LLM 判断による自動起動**: `main.go` または `internal/cli/**` を変更した直後（CLIドキュメント反映ルールに従い漏れを防ぐ）。

## 手順

### ステップ1: 乖離を検出する

以下のチェックを順に実施する。

#### 1a. Short / Long の確認

```bash
grep -n "Short:\|Long:" internal/cli/*.go
```

`Short` または `Long` が空文字列・未設定のコマンドを特定する。

#### 1b. docs/cli/ の鮮度チェック

作業ツリーを汚さないよう `docs/cli/` をバックアップしてから再生成し、差分を確認して元に戻す。

```bash
TMPDIR=$(mktemp -d)
cp docs/cli/* "$TMPDIR/"
make docs 2>/dev/null
diff -rq "$TMPDIR/" docs/cli/
cp "$TMPDIR/"* docs/cli/
rm -rf "$TMPDIR"
```

差分があれば対象ファイルを記録する。

#### 1c. README の CLI セクション確認

```bash
# 実装側コマンド名（vox-radio ルートコマンドを除く）
grep -h 'Use:' internal/cli/*.go | grep -v '"vox-radio"' | grep -oE '"[a-z-]+"' | tr -d '"' | sort -u

# README のコマンドテーブル記載コマンド名（行頭が "| `コマンド名`" の行のみを対象）
grep -E '^\| `[a-z-]+`' README.md | sed 's/^| `\([a-z-]*\)`.*/\1/' | sort -u
```

実装にあって README に記載がないコマンドを特定する。

すべてのチェックで乖離がなければ「ドキュメント最新。修正不要。」と報告して終了する。

---

### ステップ2: docs/cli/ を再生成する

ステップ1b で差分があった場合は `make docs` を実行してドキュメントを更新する。

```bash
make docs
```

---

### ステップ3: Short / Long を修正する

ステップ1a で未設定が見つかった場合、対象の `.go` ファイルを修正する。

- `Short`: 動詞始まりの英語1行（例: `Collect articles from RSS feeds`）
- `Long`: 詳細説明 + `Example:` セクション（他のコマンドを参考にする）

修正後は再度 `make docs` を実行してドキュメントに反映させる。

---

### ステップ4: README を修正する

ステップ1c で不足コマンドが見つかった場合、README のコマンドテーブルを更新する。

- 追加コマンド: パイプライン順序（collect → script → synth → assemble → publish → prune / run）に合わせて行を挿入する
- 削除コマンド: 対応行を削除する

---

### ステップ5: 修正をコミットする

```bash
git add docs/cli/ README.md internal/cli/
git commit -m "docs: CLIドキュメントを更新する"
```

コミットメッセージは変更内容に応じて調整する（例: `docs: run コマンドを README に追記する`）。

---

### ステップ6: 再チェック

ステップ1a〜1c を再度実行して乖離がなくなったことを確認する。

乖離が残っている場合はステップ2〜5を繰り返す。
