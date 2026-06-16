# 0079. サンプル episode-spec を text/template で単一ソース化する

- ステータス: 採用
- 日付: 2026-06-16

## コンテキスト

ADR-0078 で `init --sample-with-assets` を追加した際、共通ファイルは `templates-sample` を再利用し `episode-spec.yaml` のみを別ツリー（`templates-sample-with-assets`）でオーバーレイする方式を採った。しかし2つの `episode-spec.yaml` は番組本文（`program` / `casts` / `corners` の台本・長さ・`condition`・`source`）約140行が完全一致で、差分はヘッダ/フッタのコメント・`corner_defaults`・`opening`/`ending` の `start_audio`/`end_audio` の十数行のみ。本文が丸ごと複製されているため、将来本文を編集する際に両ファイルの手動同期が必要で更新漏れリスクがあった。差分が散在するため単純な部分オーバーレイや連結では解消できない。

## 決定

本文を単一テンプレート `templates-sample/episode-spec.yaml.tmpl` に集約し、`text/template`（`struct { WithAssets bool }`）の `{{if .WithAssets}}` で音割り当て差分のみを分岐させる。`init` は埋め込みテンプレートをレンダリングして `episode-spec.yaml` を書き出す（`--sample` は `WithAssets=false`、`--sample-with-assets` は `true`）。`templates-sample-with-assets` ディレクトリと `sampleWithAssetsFS` 埋め込みは廃止する。現行2ファイルとのバイト等価はゴールデンテストで担保する。ADR-0078 の「episode-spec.yaml のみオーバーレイする」生成メカニズムは本 ADR で更新する。

## 結果

- 番組本文が単一ソース化され、二重管理と同期漏れリスクが解消される。
- ゴールデン比較テストで移行のバイト等価と意図しないテンプレ変更を検知できる。`text/template` は render・slackpost で既に採用済みで前例と整合する。
- トレードオフ: テンプレートが標準 YAML でなくなり、教育用コメントにテンプレート命令が混ざるため可読性は低下する（ユーザー合意のうえ許容）。`--sample` 系もレンダリング経由になる。

## 検討した代替案

- **ドリフト検知テスト**: 2ファイルを残し本文一致を CI で検証する案。同期漏れは防げるが本文の二重管理（2ファイル編集）は残るため却下。
- **base＋パッチ生成（ゴールデン）**: base から行アンカー挿入で音入り版を生成する案。アンカー挿入が脆く実装が重いため却下。
- **現状維持**: 安定した教育用フィクスチャとして放置する案。更新漏れリスクが残るため却下。
