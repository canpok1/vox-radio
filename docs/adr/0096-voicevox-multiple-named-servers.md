# 0096. VOICEVOX 接続先を名前付き複数サーバー化しキャラクター単位で切り替える

- ステータス: 採用
- 日付: 2026-07-04

## コンテキスト

VOICEVOX NEMO は通常の VOICEVOX Engine とは別サーバー（別ポート）で動作するが、API（/audio_query, /synthesis, /version）は互換である。現状の設定は `voicevox.url` の単一 URL 前提（ADR-0042 の環境変数 `VOX_RADIO_VOICEVOX_URL` による上書き含む）で、キャラクターの speaker ID もサーバーとの紐づきを持たないため、VOICEVOX と NEMO の声を同一エピソードで併用できない。既存ユーザーの設定ファイル・環境変数（devcontainer / GitHub Actions / Docker ガイド）を壊さずに複数サーバーへ拡張する方法が必要になった。

## 決定

`voicevox.engines` に名前付きサーバー（名前 → `url`）を定義し、`characters.<id>.engine` で使用エンジンを指定する（省略時 `default`）。環境変数はエンジン名ごとに `VOX_RADIO_VOICEVOX_URL_<NAME>`（大文字化・`-`→`_` 正規化）で上書きし、既存の `VOX_RADIO_VOICEVOX_URL` は default 専用として維持する。`voicevox.url` のみの既存設定は暗黙の default エンジンとして従来通り動作させ、`url` と `engines` の同時指定はエラー。presets・startup_timeout_seconds は共通のまま、readiness チェックは定義済み全エンジンへ並行実施する。フィールド名は `characters.<id>.engine` との対応が直感的になるよう `servers` ではなく `engines` とする（VOICEVOX 自体も「エンジン」と呼ばれる）。

```yaml
voicevox:
  engines:
    default: { url: http://localhost:50021 }
    nemo: { url: http://localhost:50121 }
characters:
  anneli:
    engine: nemo
```

## 結果

- VOICEVOX と NEMO（および任意の互換サーバー）の声を1エピソード内で併用できる。
- 既存の設定ファイル・環境変数は無変更で動作し続ける（後方互換）。
- クライアント実装は共通の `VoicevoxClient` を再利用でき、追加はエンジン名→クライアントのルーティングのみ。
- トレードオフ: 定義済み全エンジンが起動チェック対象になるため、使わないエンジンは定義から外す運用が必要。環境変数名の正規化衝突（`nemo-v2` と `nemo_v2` 等）はバリデーションで検出する。

## 検討した代替案

- **スタイル単位でエンジンを指定**: 1キャラクターの声は通常1エンジンに属するため過剰。設定が冗長になり却下。
- **speaker ID にエンジンプレフィックスを埋め込む**（例: `nemo:0`）: styles マップの値が int でなくなり既存設定の互換性を壊すため却下。
- **エピソードで実際に使うエンジンのみ readiness チェック**: コマンドごとに使用エンジンの解決ロジックが必要で複雑化するため、シンプルな全エンジンチェックを採用（ユーザー確認済み）。
