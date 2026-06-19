# 0088. VOICEVOX 起動待機ポーリングを vox-radio 本体に組み込む

- ステータス: 採用
- 日付: 2026-06-19

## コンテキスト

CI（`demo-release.yml`）では VOICEVOX サービスコンテナの起動完了を `curl /version` のループで確認してから音声合成を実行していた。同じレース条件はローカル利用でも起きる（エンジン起動直後の実行）が、ローカルには対策がなかった。

既存の `httpretry` は 5xx/429 をリトライするが、接続レベルのエラー（connection refused）は即返しするため、エンジン未起動時にそのまま失敗する。GitHub Actions のサービスコンテナヘルスチェック（`--health-cmd`）はコンテナ内に `curl` がない可能性が高く見送り、サードパーティ製アクションは依存増加につき不採用とした。

## 決定

`Synth.Run` 冒頭で `waitForReady(ctx, client, timeout, interval)` ヘルパーを呼び、`/version` が 200 を返すまでポーリングしてから合成を開始する。

- `VoicevoxClient` に `Version(ctx) (string, error)` を追加（`GET /version`）。httpretry なしの専用クライアントで呼び出し、失敗を即返して外側ループに委ねる
- `VoicevoxConfig.StartupTimeoutSeconds *int`（yaml: `startup_timeout_seconds`）と `EffectiveStartupTimeout()` を追加。デフォルト 60 秒、`0` で無効
- ポーリング間隔: 1 秒固定（名前付き定数 `pollIntervalDefault`）

## 結果

**良い点**: CI・ローカル双方で起動タイミングのレース条件が解消。CI から `Wait for VOICEVOX` ステップを削除でき、依存なしで同等の待機が実現できる。エンジン起動済みの場合は即進むためペナルティなし。

**注意点**: デフォルト 60 秒の待機が有効になるため、VOICEVOX が存在しない環境では最大 60 秒後にタイムアウトエラーとなる。`startup_timeout_seconds: 0` で待機を無効化できる。

## 検討した代替案

- **`httpretry` に接続エラー対応を追加**: 全クライアントに影響するため却下。`Version` 専用の素の HTTP クライアントを使う方が局所的。
- **CLI フラグで待機秒数を受け取る**: `docs/cli/` 再生成が必要になり負担が大きい。設定ファイルで完結させる方が既存パターンと整合する。
- **サードパーティ製 GitHub Actions**: 依存追加につき不採用。本体機能として持つことでローカルも同時に改善できる。
