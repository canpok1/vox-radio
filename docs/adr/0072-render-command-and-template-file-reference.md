# 0072. 汎用 render コマンドを追加し、テンプレをファイルパス参照に統一する

- ステータス: 採用
- 日付: 2026-06-15

## コンテキスト

[ADR-0071](0071-slackpost-text-template-message-spec.md) で `slackpost` のメッセージ整形を `text/template` ベースへ移行することを決めた。一方、利用側のリポジトリでは GitHub Release ノートを manifest から jq 等で組み立てており（紹介記事は `url != null and url != ""` で抽出）、この組み立てを専用コマンドとして本体に取り込みたい。

slackpost と同じ「manifest を text/template でレンダリングする」仕組みを汎用化すれば、リリースノート生成も同一のテンプレ記法・同一の URL 区別で実現でき、jq 依存を撤去できる。また ADR-0071 はテンプレを `slack-spec.yaml` にインライン記載する方針だったが、render はテンプレファイルを引数で受け取るため、両者でテンプレを共有するにはテンプレの持ち方をそろえる必要がある。

## 決定

1. **汎用 `render` コマンドを追加する**。`vox-radio render --manifest <path> --template <path> [--output <path>]` で manifest を任意の `text/template` ファイルでレンダリングし、標準出力（または `--output` でファイル）へ書き出す。リリースノートは代表的ユースケースで、他の文言組み立てにも使える。Slack 固有の 3000 文字ブロック分割は行わず、生テキストをそのまま出力する。

2. **レンダリング核を共有パッケージに抽出する**。`text/template` のパース・実行、FuncMap（`corner "<id>"` / `hasLinks`）、manifest 全体のデータ文脈を共有パッケージ（`internal/render` 仮）に切り出し、`slackpost` と `render` の双方が利用する。Slack 固有のブロック分割は `internal/slack` 側に残す。

3. **テンプレの持ち方をファイルパス参照へ統一する（ADR-0071 の改定）**。`slack-spec.yaml` の `parent` / `thread` / `fallback` はテンプレ本文ではなく**テンプレファイルのパス**を持つ（省略時は組み込みデフォルト）。これにより slackpost と render が同一の `.tmpl` を参照・共有でき、テンプレ管理が一貫する。`init` は `slack-spec.yaml`（パス指定）とデフォルト `.tmpl` を生成し、双方に初見ユーザー向けの説明コメントを付ける。

## 結果

- リリースノート生成を本体コマンドで賄え、jq による抽出ロジックを撤去できる。slackpost とリリースノートで同一のテンプレ記法・URL 区別に統一される。
- レンダリング核が 1 箇所に集約され、slackpost と render の重複実装を避けられる。
- `slack-spec.yaml` はテンプレ本文ではなくパスを持つ形に変わる（ADR-0071 のインライン方針からの破壊的変更）。テンプレファイルが増えるが、共有・バージョン管理・補完が効く。

## 検討した代替案

- **リリースノート専用コマンド**（`releasenote` + 専用 spec）: 用途は明確だが、文言組み立ての用途ごとにコマンドが増える。汎用 render なら 1 コマンドで賄える。
- **テンプレを spec.yaml にインラインのまま維持**（ADR-0071 のまま）: render とテンプレを共有できず、同種テンプレを二重管理することになる。
- **render もインライン spec を受け取る**: テンプレファイルを引数にする方が単純で、パイプ（`> RELEASE_NOTES.md`）にも素直。
