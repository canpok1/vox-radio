# 0096. VOICEVOX 接続先を名前付き複数サーバー化しキャラクター単位で切り替える

- ステータス: 採用
- 日付: 2026-07-04

## コンテキスト

VOICEVOX NEMO は通常の VOICEVOX Engine とは別サーバー（別ポート）で動作するが、API（/audio_query, /synthesis, /version）は互換である。現状の設定は `voicevox.url` の単一 URL 前提（ADR-0042 の環境変数 `VOX_RADIO_VOICEVOX_URL` による上書き含む）で、キャラクターの speaker ID もサーバーとの紐づきを持たないため、VOICEVOX と NEMO の声を同一エピソードで併用できない。既存ユーザーの設定ファイル・環境変数（devcontainer / GitHub Actions / Docker ガイド）を壊さずに複数サーバーへ拡張する方法が必要になった。

## 決定

`voicevox.servers` に名前付きサーバー（名前 → `url`）を定義し、`characters.<id>.engine` で使用サーバーを指定する（省略時 `default`）。環境変数はサーバー名ごとに `VOX_RADIO_VOICEVOX_URL_<NAME>`（大文字化・`-`→`_` 正規化）で上書きし、既存の `VOX_RADIO_VOICEVOX_URL` は default 専用として維持する。`voicevox.url` のみの既存設定は暗黙の default サーバーとして従来通り動作させ、`url` と `servers` の同時指定はエラー。presets・startup_timeout_seconds は共通のまま、readiness チェックは定義済み全サーバーへ並行実施する。

```yaml
voicevox:
  servers:
    default: { url: http://localhost:50021 }
    nemo: { url: http://localhost:50121 }
characters:
  anneli:
    engine: nemo
```

## 結果

- VOICEVOX と NEMO（および任意の互換サーバー）の声を1エピソード内で併用できる。
- 既存の設定ファイル・環境変数は無変更で動作し続ける（後方互換）。
- クライアント実装は共通の `VoicevoxClient` を再利用でき、追加はサーバー名→クライアントのルーティングのみ。
- トレードオフ: 定義済み全サーバーが起動チェック対象になるため、使わないサーバーは定義から外す運用が必要。環境変数名の正規化衝突（`nemo-v2` と `nemo_v2` 等）はバリデーションで検出する。

## 検討した代替案

- **スタイル単位でサーバーを指定**: 1キャラクターの声は通常1エンジンに属するため過剰。設定が冗長になり却下。
- **speaker ID にサーバープレフィックスを埋め込む**（例: `nemo:0`）: styles マップの値が int でなくなり既存設定の互換性を壊すため却下。
- **エピソードで実際に使うサーバーのみ readiness チェック**: コマンドごとに使用サーバーの解決ロジックが必要で複雑化するため、シンプルな全サーバーチェックを採用（ユーザー確認済み）。
