# vox-radio 最新版への更新手順リファレンス

> vox-radio 本体の新バージョンがリリースされたとき、プロジェクトを最新版へ追随させるための手順です。
> 「vox-radio を最新版にして」「バージョンを上げて」「最新版に対応して」等の依頼や、SKILL.md の
> 「バージョン整合チェック」でバイナリが古いと判明したときに、この手順に沿って更新してください。

## ステップ1: 更新要否の判定（最新版の確認）

最新リリースタグと現在のバイナリ版を比較し、**同一なら更新不要として終了**します（この手順は何度実行しても安全です）。

```bash
# 最新リリースタグ（semver 降順の先頭）。例: v0.0.17
LATEST="$(git ls-remote --tags --refs --sort=-v:refname https://github.com/canpok1/vox-radio.git \
  | head -n1 | sed 's#.*/##')"
# 現在のバイナリ版。出力形式は "vox-radio version X.Y.Z"（タグは v 付き、--version は v なし）
CURRENT="$(vox-radio --version | awk '{print $NF}')"
echo "current=$CURRENT latest=$LATEST"
```

- `CURRENT` が `dev`（ローカルビルド）の場合は比較不能。警告を出すだけにとどめ、更新は行わない。
- `"v$CURRENT" == "$LATEST"` の場合は「既に最新版です」と報告して終了する。
- `"v$CURRENT" != "$LATEST"` の場合のみステップ2以降へ進む。

## ステップ2: バイナリの更新

最新リリースの `install.sh` を取得して実行し、バイナリを入れ替えます。**インストール位置は変更せず、現在と同じ場所へ入れ替える**こと。先に現在の設置先を確認し、その場所を `INSTALL_DIR` に指定して実行します。

```bash
# 現在の設置先を確認（同じ場所へ入れ替えるため）
INSTALL_DIR="$(dirname "$(command -v vox-radio)")"
echo "install dir: $INSTALL_DIR"

# 同じ場所へ最新版を入れる
curl -fsSL https://github.com/canpok1/vox-radio/releases/latest/download/install.sh \
  | INSTALL_DIR="$INSTALL_DIR" bash
vox-radio --version   # 更新後の版を確認
```

- 特定バージョンを入れる場合は `latest/download` をタグに置き換える（例: `releases/download/v0.0.17/install.sh`）。

## ステップ3: エージェントスキルの再インストール

バイナリと同じバージョンのスキル（`SKILL.md` ＋ `references/*.md`）へ更新します。設定ファイルのフィールド定義は
この `references/*.md` を正とするため、**設定の追随（ステップ4）より先に**実行します。

**現在スキルがインストールされている場所を維持して上書き**すること。この手順書（`update.md`）自身が
置かれているディレクトリが現在のスキル設置先（`<skills-dir>/vox-radio/`）なので、その親ディレクトリを
`--skills-dir` に指定する。既定の `.claude/skills` 以外（`--skills-dir` 指定でインストールした場合）に
置かれていることもあるため、思い込みで `.claude/skills` を使わないこと。

```bash
# 現在のスキル設置先の親ディレクトリ。例: 既定なら .claude/skills、別の場所ならそのパス
SKILLS_DIR="<この update.md がある vox-radio/ ディレクトリの親>"
vox-radio install --skills --skills-dir "$SKILLS_DIR" --force
git status "$SKILLS_DIR/vox-radio/"   # 差分を確認（references の変更が破壊的変更の手がかりになる）
```

- `--force` を付けないと既存スキルは上書きされない。更新時は `--force` を必ず付けること。
- 既定の `.claude/skills` に入っている場合は `--skills-dir` を省略してよい。
- `.skill-version` は常に新バイナリ版で上書きされ、版ずれが解消する。

## ステップ4: 設定ファイル等の最新版への追随

旧→新の変更点（破壊的変更・必須フィールド追加・既定値変更など）を把握し、各設定ファイルを追随させます。

1. **変更点の把握**: リリースノートを確認する。取得できない場合は、ステップ3で更新された
   `references/*.md` の差分（`git diff` / `git diff .claude/skills/vox-radio/`）と、ステップ5の `check`
   エラーを根拠にする。リリースノートは次で取得できる（ネットワーク制限時は省略可）。

   ```bash
   curl -fsSL "https://api.github.com/repos/canpok1/vox-radio/releases/tags/${LATEST}" \
     | sed -n 's/.*"body": "\(.*\)".*/\1/p'
   ```

2. **設定ファイルの更新**: 各設定ファイル（`vox-radio.yaml` / `episode-spec.yaml` / `feed-spec.yaml` /
   `slack-spec.yaml` / アセット設定 YAML）を、更新後の `references/*.md` の定義に沿って修正する。
   具体的な修正は SKILL.md 本体の「リファレンスと検証コマンド」「それ以外（部分編集…）」のフローに従う。
   - **現状の挙動を維持する形で修正する**こと（番組内容・出力・設定値の意味を変えない）。フィールド名変更・
     必須化・既定値変更などに対しては、これまでと同じ結果になるよう値を移行・補完する。
   - 現状の挙動を維持できない場合や、これまで設定していなかった項目を新たに設定する必要がある場合は、
     勝手に決めずユーザーに判断を仰ぐこと。

## ステップ5: 検証ループ

更新したすべての設定ファイルについて、該当の check コマンドが**終了コード 0 になるまで**修正と検証を繰り返します
（コマンド一覧は SKILL.md の「リファレンスと検証コマンド」を参照）。エラーが出たらステップ4へ戻る。

```bash
vox-radio config check --config vox-radio.yaml
vox-radio episodegen check episode-spec.yaml --config vox-radio.yaml
vox-radio feedgen check
vox-radio slackpost check
vox-radio assets check assets/assets.yaml
```

- プロジェクトに存在しない設定ファイルの check は省略してよい。

## 注意事項

- バイナリだけを更新するとスキルが古いまま残り、references と実際の挙動が食い違う。ステップ3の再インストールを必ず行うこと。
- 更新不要（最新版と同一）の場合は何も変更せず終了する。
