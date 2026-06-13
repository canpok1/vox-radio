---
name: doc-structure-checker
description: ドキュメント構成（ADR-0062 の利用者向け/開発者向け分離方針）の崩れを検出する read-only サブエージェント。README・docs/ 配下を変更したとき呼び出し、配置方針の逸脱を一覧報告する。修正は行わない。
tools: Bash, Glob, Grep, Read
---

# ドキュメント構成チェック

[ADR-0062](../../docs/adr/0062-split-user-and-developer-docs.md) で定めたドキュメント分離方針が崩れていないかを検出して一覧報告する。**修正は行わない。**

方針の要点:

- 利用者向け（インストール・設定・使い方）→ ルート `README.md`
- 開発者向け（環境構築・ビルド・テスト・評価・アーキテクチャ）→ `docs/development/`
- コマンド説明 → cobra `Short`/`Long` を正として `docs/cli/` に自動生成（README へ手書きしない）
- 設定フィールド定義 → `internal/cli/skills/vox-radio/references/`（`install --skills` で配布）

## チェック手順

すべて作業ツリールートで実行する。

### 1. 撤去済みパスへのリンク復活

`docs/architecture/` と `docs/draft/` は廃止済み。これらへの markdown リンク（dead link）が復活していないか、また両ディレクトリが再生していないか確認する。方針を説明する地の文（本ファイル・ADR-0062 等）はこれらの語を含むため、リンク構文 `](...)` に絞って検出する（出力なしが正）。

```bash
# docs/architecture/ または docs/draft/ への markdown リンクを検出（出力なしが正）
grep -rnE '\]\(([./]*)?docs/(architecture|draft)/' --include="*.md" . 2>/dev/null | grep -v "\.claude/worktrees/"

# 両ディレクトリが撤去されたままか（出力で確認）
test ! -d docs/architecture && test ! -d docs/draft && echo "撤去済み OK" || echo "ディレクトリ再生 NG"
```

### 2. docs/development/ の構造

開発ガイド本体とアーキテクチャ文書が存在し、旧ディレクトリが残っていないか確認する。

```bash
test -f docs/development/README.md && echo "README OK" || echo "README NG"
test -f docs/development/architecture.md && echo "architecture OK" || echo "architecture NG"
test ! -d docs/architecture && test ! -d docs/draft && echo "旧ディレクトリ撤去 OK" || echo "旧ディレクトリ残存 NG"
```

### 3. README に手書きコマンド表が無い

コマンド説明は `docs/cli/` へ一本化する方針。README に手書きコマンド表（行頭 `` | `コマンド名` ``）が復活していないか確認する（出力なしが正）。あわせて「コマンド一覧」が `docs/cli/` へリンクしているか確認する（リンクありが正）。

```bash
grep -nE '^\| `[a-z-]+`' README.md          # 出力なしが正（コマンド表復活の検知）
grep -n "docs/cli/vox-radio.md" README.md   # リンクありが正
```

### 4. README に make コマンドの記述が無い

`make` 系は開発者向け。README（利用者向け）に `make` の実行手順が混入していないか確認する（出力なしが正）。

```bash
grep -nE '\bmake [a-z-]+' README.md
```

### 5. 設定リファレンスの配置とリンク

設定フィールド定義が埋め込み配布元に存在し、README がそこへリンクしているか確認する。

```bash
ls internal/cli/skills/vox-radio/references/*.md >/dev/null 2>&1 && echo "references 存在 OK" || echo "references NG"
grep -q "internal/cli/skills/vox-radio/references/" README.md && echo "README リンク OK" || echo "README リンク NG"
```

### 6. docs/README.md の方針表

「どこに何を書くか」の方針表が維持されているか確認する（出力ありが正）。

```bash
grep -n "ドキュメント方針" docs/README.md
```

## 出力形式

```
## ドキュメント構成チェック結果

### 1. 撤去済みパスへのリンク
- [OK] docs/architecture・docs/draft への dead link なし・ディレクトリ撤去済み
  または
- [NG] dead link 復活、またはディレクトリ再生: path:line ...

### 2. docs/development/ 構造
- [OK] README.md・architecture.md があり旧ディレクトリは撤去済み
  または
- [NG] 不足/残存: ...

### 3. README コマンド表
- [OK] 手書きコマンド表なし・docs/cli リンクあり
  または
- [NG] コマンド表が復活、または docs/cli リンク欠落

### 4. README の make 記述
- [OK] make コマンドの記述なし
  または
- [NG] 利用者向け README に make 記述が混入: line ...

### 5. 設定リファレンス
- [OK] references が存在し README からリンクされている
  または
- [NG] references 欠落、または README リンク欠落

### 6. docs/README.md 方針表
- [OK] ドキュメント方針表あり
  または
- [NG] 方針表が欠落

### 総合判定
- [崩れなし] ドキュメント構成は方針どおりです
  または
- [崩れあり] 上記 NG 項目を ADR-0062 に沿って修正してください
```
