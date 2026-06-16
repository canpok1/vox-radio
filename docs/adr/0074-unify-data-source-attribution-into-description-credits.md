# 0074. データソース帰属を description クレジット節に統合し feed.credit を廃止する

- ステータス: 採用
- 日付: 2026-06-16
- 旧決定を置換: [0065-propagate-asset-and-character-credits-to-outputs.md](0065-propagate-asset-and-character-credits-to-outputs.md)

## コンテキスト

ADR 0065 では「既存の `feed.credit`（`<itunes:author>`）は配信者表記として残す」と決定したが、実際の利用を振り返ると `feed.credit` に入っていたのはデータソース帰属（「気象庁「防災情報XML」を加工して作成」）であり、itunes:author の本来用途（著者名）と異なる使い方だった。

また、番組固定のデータソース帰属をアセット・キャラクターのクレジットとは別ファイル（`feed-spec.yaml`）で管理していたため、クレジットが分散していた。

## 決定

1. `feed.credit`（`feed-spec.yaml`）を廃止し、item `itunes:author` へのクレジット出力をやめる（channel `itunes:author` は番組著者名として継続）。
2. `ProgramConfig` に `credits []string`（`episode-spec.yaml` の `program.credits`）を新設し、番組固定のデータソース帰属をここに記入する。
3. `manifest.Build`/`CollectCredits` で `Program.Credits` をアセット・キャラクターのクレジットより先頭に統合し、`manifest.credits` へ集約する（重複排除あり）。
4. feed 生成時、統合された `manifest.credits` を各エピソードの `<description>` 末尾クレジット節へ自動追記する（ADR 0065 で確立済みの仕組みを流用）。

## 結果

- データソース帰属とアセット・キャラクタークレジットが `manifest.credits` に一元化され、description クレジット節に自動転記される。
- `feed.credit` は strict ロードで unknown key エラーになるため、旧設定ファイルを持つユーザーは `program.credits` への移行が必要（リリースノートで案内）。

## 検討した代替案

- **`feed.credit` を保持して用途を限定**: 既存ユーザーへの破壊的変更を避けられるが、帰属と著者名が同一フィールドに混在する混乱を解消できない。
- **`feed.credit` を description にも流す**: 廃止せず description に追記する折衷案。`feed-spec.yaml` と `episode-spec.yaml` への分散が残るため不採用。
