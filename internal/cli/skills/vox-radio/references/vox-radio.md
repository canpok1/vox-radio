# vox-radio.yaml（共通設定）リファレンス

> **フィールド定義の正**: `internal/config/config.go`。本ドキュメントとコードが齟齬する場合はコードを優先してください。

> **検証コマンド**: `vox-radio config check --config <パス>`

`vox-radio.yaml` はデフォルトでカレントディレクトリから読み込まれます。`--config` フラグで別パスを指定できます。

## `llm` セクション

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `provider` | string | 任意 | LLM プロバイダ。`openai`（デフォルト）または `dify-chat` |
| `temperature` | float64 | 任意 | 生成のランダム性（0.0〜1.0）。デフォルト: 0（Go ゼロ値） |
| `max_retries` | int | 任意 | APIリトライ回数。デフォルト: 0（Go ゼロ値） |
| `min_request_interval_ms` | *int | 任意 | リクエスト間隔（ミリ秒）。省略時は 4500ms |
| `steps` | map[string]LLMStepConfig | 任意 | ステップごとの設定（キー: ステップ名） |
| `openai` | OpenAIConfig | `provider: openai` 時必須 | OpenAI 互換プロバイダの接続設定 |
| `dify-chat` | DifyChatConfig | `provider: dify-chat` 時必須 | Dify chat-messages の接続設定 |

### `llm.openai` サブフィールド（`provider: openai` 時）

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `base_url` | string | 必須 | LLM API のベースURL（OpenAI 互換エンドポイント） |
| `api_key_env` | string | 必須 | APIキーを格納する環境変数名 |
| `model` | string | 必須 | 使用するモデル名 |

### `llm.dify-chat` サブフィールド（`provider: dify-chat` 時）

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `base_url` | string | 必須 | Dify API サーバーURL（例: `https://api.dify.ai/v1`） |
| `api_key_env` | string | 必須 | Dify API キーを格納する環境変数名 |
| `user` | string | 任意 | 利用者識別子。省略時は `vox-radio` |
| `inputs` | map[string]string | 任意 | Dify アプリに渡す変数。値に `${temperature}` プレースホルダーを書ける |

`inputs` の `${temperature}` プレースホルダーについて:
- 値が `"${temperature}"` だけの場合（完全一致）→ そのステップの temperature を **JSON 数値**で送信
- 値に `${temperature}` が含まれる場合（部分一致）→ 文字列として補間
- プレースホルダーを書かない場合 → temperature を inputs に含めない

> **注意**: inputs に temperature を載せても、Dify アプリ側でその変数をモデルパラメータにバインドしない限り効果はありません。

### `llm.steps.<step>` サブフィールド

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `temperature` | *float64 | 任意 | このステップの温度（省略時は `llm.temperature` を使用） |

組み込みステップ名: `summarize`（記事要約）、`plan`（台本設計）、`write`（セリフ執筆）、`direct`（ダイレクト生成）。

## `voicevox` セクション

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `url` | string | 必須 | VOICEVOX Engine のURL |
| `presets` | *VoicevoxPresets | 任意 | 抑揚・音高・話速プリセット定義。省略時はコード組込みのデフォルトプリセットが適用される |

`url` は環境変数 `VOX_RADIO_VOICEVOX_URL` で上書きできます。解決順は `VOX_RADIO_VOICEVOX_URL`（環境変数）> `voicevox.url` > 既定値 `http://localhost:50021` です。

### `voicevox.presets` サブフィールド

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `intonation` | map[string]float64 | 任意 | 抑揚プリセット（intonationScale, 0.0〜2.0）。省略時はデフォルト7段階が適用される |
| `pitch` | map[string]float64 | 任意 | 音高プリセット（pitchScale, -0.15〜0.15）。省略時はデフォルト7段階が適用される |
| `speed` | map[string]float64 | 任意 | 話速プリセット（speedScale, 0.5〜2.0）。省略時はデフォルト7段階が適用される |

## `cache` セクション

キャッシュは常に有効です。エピソード履歴は `episode-spec.yaml` の `program.id`（必須）をキーに JSONL へ保存されます。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `max_entries` | int | 任意 | JSONL に保持する最大エピソード数（超過分は古い行から削除）。デフォルト: 100 |
| `retention_days` | int | 任意 | 保持日数（超過した古い行は削除）。デフォルト: 90 |
| `llm_context_entries` | int | 任意 | LLM へ渡す直近エピソード件数。デフォルト: 10 |

## `security` セクション

省略時はすべて既定値が適用されます。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `prompt_injection` | PromptInjectionConfig | 任意 | プロンプトインジェクション対策設定 |

### `security.prompt_injection` サブフィールド

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `on_detect` | string | 任意 | 検出時の挙動。`exclude`（既定: 記事を丸ごと除外して継続）または `error`（パイプライン停止） |
| `max_body_chars` | int | 任意 | 記事本文の最大ルーン数。超過分は切り詰め。0 または省略で 3000 |

## `characters` セクション

`characters` はキャラID（文字列キー）をキーにしたマップです。プロファイルの `corners[].cast` で使用するIDを定義します。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `name` | string | 必須 | キャラクターの表示名 |
| `pronoun` | string | 任意 | 一人称代名詞（台本生成時に LLM へ渡す） |
| `speech_suffix` | []string | 任意 | 語尾パターン（台本生成時に LLM へ渡す） |
| `personality` | []string | 任意 | 性格特徴（台本生成時に LLM へ渡す） |
| `default_style` | string | 任意 | デフォルトの音声スタイル名（`styles` のキーと一致させること） |
| `styles` | map[string]int | 任意 | スタイル名 → VOICEVOX 話者ID のマップ |

`default_style` を指定した場合、その値は `styles` のキーとして存在しなければなりません（起動時検証あり）。
