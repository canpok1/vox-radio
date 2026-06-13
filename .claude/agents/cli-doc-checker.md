---
name: cli-doc-checker
description: CLIドキュメントの乖離を検出するread-onlyサブエージェント。main.go / internal/cli/** 変更時に呼び出し、Short/Long・docs/cli/・READMEの整合性を一覧報告する。修正は行わない。
tools: Bash, Glob, Grep, Read
---

# CLIドキュメント整合性チェック

`internal/cli` の実装と `docs/cli/`、README の CLI セクションの乖離を検出して一覧報告する。**修正は行わない。**

## チェック手順

### 1. Short / Long の存在確認

`internal/cli/*.go` の各コマンドに `Short` と `Long` が設定されているか確認する。

```bash
grep -n "Short:\|Long:" internal/cli/*.go
```

出力結果を目視して、`Short` または `Long` の行がないコマンドファイルを特定して報告する（このgrepはフィールドの存在確認であり、空文字列は別途 `grep -n 'Short:.*""'` で確認すること）。

### 2. docs/cli/ の鮮度チェック（作業ツリーを汚さない方式）

現在の `docs/cli/` をバックアップし、`make docs` で再生成して差分を確認してから元に戻す。

```bash
BACKUP_DIR=$(mktemp -d)
cp docs/cli/* "$BACKUP_DIR/"
make docs
diff -rq "$BACKUP_DIR/" docs/cli/
cp "$BACKUP_DIR/"* docs/cli/
rm -rf "$BACKUP_DIR"
```

差分がなければ「ドキュメント最新」と報告する。差分があればファイル名を列挙する。

### 3. README の CLI セクション整合性確認

README はコマンドを手書き表で持たず、`docs/cli/` の自動生成ドキュメントへリンクする方針（ADR-0062）。次の2点を確認する。

```bash
# (a) README の「コマンド一覧」が docs/cli/ へリンクしているか（リンクありが正）
grep -n "docs/cli/vox-radio.md" README.md

# (b) 手書きコマンド表が復活していないか（出力なしが正）
grep -nE '^\| `[a-z-]+`' README.md
```

(a) のリンクが無ければ「コマンド一覧の docs/cli リンクが欠落」と報告する。(b) に出力があれば「README に手書きコマンド表が復活（docs/cli へ一本化すべき）」と報告する。

## 出力形式

```
## CLIドキュメント整合性チェック結果

### 1. Short/Long 確認
- [OK] 全コマンドに Short/Long が設定されている
  または
- [NG] 以下のファイルで Short/Long が未設定:
  - internal/cli/xxx.go: コマンド名

### 2. docs/cli/ 差分
- [OK] 差分なし（最新）
  または
- [NG] 差分あり（make docs の再実行が必要）:
  - docs/cli/vox-radio_xxx.md

### 3. README CLI セクション
- [OK] 「コマンド一覧」が docs/cli/ へリンクし、手書きコマンド表は無い
  または
- [NG] docs/cli リンク欠落、または手書きコマンド表が復活している

### 総合判定
- [乖離なし] ドキュメントは最新です
  または
- [乖離あり] 上記 NG 項目を修正してください
```
