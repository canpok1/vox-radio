# 0030. 設定ファイルを episode-spec / feed-spec に統一し「プロファイル」概念を廃止する

- ステータス: 採用
- 日付: 2026-06-03

## コンテキスト

ADR-0010 で設定を共通設定（Config）とジャンル別設定（Profile）に二分割したが、ADR-0026 以降 `feedgen` は `profile.yaml` に一切依存しなくなり、`profile.yaml` は実質 `episodegen` 専用の番組定義ファイルとなった。しかし命名が実態と乖離している。

- 「プロファイル（profile）」という概念名が「episodegen 専用の番組定義」を表せていない。
- `profile.yaml` はコマンドとの対応が読み取りづらく、`feedgen.yaml` とも命名規則が揃っていない。
- フラグも `--profile`（episodegen）と `--config`（feedgen）で不統一。

ADR-0026 補足で `distribution.yaml`→`feedgen.yaml` へのリネームが行われた前例があり、今回も非互換リネームの方針を踏襲する。

## 決定

- `profile.yaml` → `episode-spec.yaml`、`feedgen.yaml` → `feed-spec.yaml` に対称リネームする。
- Go シンボルを `Profile`→`EpisodeSpec`、`FeedgenConfig`→`FeedSpec`、`LoadProfile`→`LoadEpisodeSpec`、`LoadFeedgen`→`LoadFeedSpec` 等に更新する。
- CLI フラグを両コマンドとも `--spec` に統一（episodegen: `--profile`→`--spec`、feedgen: `--config`→`--spec`）。
- `profile check` コマンドを `episodegen check` へ移設し、top-level `profile` コマンドを削除する。
- 後方互換エイリアスは設けず、即時置換（破壊的変更）とする。
- 過去 ADR（0010/0011/0026 等）の本文は履歴として不変とし、本 ADR-0030 で用語更新を記録する。

## 結果

**良い影響**: ファイル名・フラグ名・コマンド名が `<ドメイン>-spec.yaml` / `--spec` / `<cmd> check` で統一され、対称性が高まる。`feedgen` の `--config` が消えて共通設定コマンド `config` との「config」の語の重複が解消される。

**悪い影響・トレードオフ**: 既存ユーザーは `profile.yaml`・`feedgen.yaml` のリネームと CI/スクリプトのフラグ更新が必要。`sample-profiles/` ディレクトリも `examples/` に移動する。

## 検討した代替案

- **`--profile` を維持し `feedgen` 側だけ `--spec` に変える**: 両コマンド間の不統一が残り命名の一貫性が改善されない。
- **`episode.yaml` / `feed.yaml` とする**: `spec` を付けることで「定義ファイル」であることが明示でき自己文書化効果が高い。`spec` を採用。
- **後方互換エイリアスを残す**: 設定ファイルのエイリアスは二重管理コストが大きく、名称変更の恩恵が薄れるため不採用。
