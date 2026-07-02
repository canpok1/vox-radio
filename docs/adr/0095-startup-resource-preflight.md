# 0095. 起動時リソース preflight を全コマンド共通機構で導入する（ADR-0088 改訂）

- ステータス: 採用
- 日付: 2026-07-02

## コンテキスト

`episodegen` は gather → rundown → script（LLM）→ synth（VOICEVOX）→ mix（ffmpeg）の順に実行する。VOICEVOX 未到達でも、LLM でセリフ生成等のコストを消費した**後**の synth 段で初めて接続エラーになり非効率だった。ADR-0088 の起動待機ポーリング（`Synth.Run` 冒頭の `waitForReady`）も synth 段でしか走らず、LLM 消費を防げない。

リソース確認は散発的（ffmpeg は PATH 確認、VOICEVOX は synth 段、LLM キーは初回呼び出し時）で、必要リソースを起動直後に一括検証して早期に失敗を返す共通機構が必要になった。

## 決定

各コマンドが使う外部リソース（ffmpeg/ffprobe・VOICEVOX・LLM キー）を、RunE 先頭（設定ロード後・LLM/パイプライン着手前）で共通機構により検証する。

- `checkResources(checks ...func() error) error` で複数チェックを実行し、失敗を集約して1エラーに列挙する（fail-fast せず不足を一覧提示）。
- VOICEVOX は `synth.CheckReadiness` が既存 `waitForReady` を流用し `EffectiveStartupTimeout()`（既定60秒, 0で無効）ぶん待機。ADR-0088 の待機を synth 段から起動時へ**前倒し**する（部分改訂）。`Synth.Run` の待機は多重防御のため維持。
- LLM は `api_key_env` の環境変数が非空かのみ検証（実 ping はしない・無コスト）。
- 対象は外部リソースを使うコマンドのみ（episodegen・rundown・script・synth・mix・assets preview）。依存のないコマンドは呼ばない。slackpost は既存 `VerifyScopes` が副作用前チェックとして機能するため対象外。

## 結果

- episodegen で LLM 消費前に VOICEVOX 未到達・ffmpeg 欠落・LLM キー未設定を検知でき、無駄なコストを防げる。不足リソースは一括で一覧提示され、修正の往復が減る。
- ADR-0088 の待機挙動（エンジン起動直後の許容）はそのまま前倒しされ後方互換を保つ。
- トレードオフ: VOICEVOX 不在時は起動時に最大60秒待って失敗する（従来と同じで発生位置が早まるだけ）。`startup_timeout_seconds: 0` で無効化可能。

## 検討した代替案

- **LLM を実際に ping**: 無効キー/ネットワークを起動時に確定できるが、毎回わずかな API コスト・レイテンシが生じ ping 用メソッドの新設も必要。無効キーは最初の LLM 呼び出しで早期・安価に判明するため、キー存在検証に留めた。
- **VOICEVOX を即時 fail-fast**: 起動待機を廃すると ADR-0088 の「エンジン起動直後」許容が失われるため却下。既存待機を流用する。
- **共通 `PersistentPreRunE` に集約**: コマンドごとに必要リソースが異なり、設定ロード後でないと接続先が定まらないため、各 RunE で必要チェックのみ呼ぶ方式にした。
