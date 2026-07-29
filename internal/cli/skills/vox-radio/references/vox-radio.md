# vox-radio.yaml（共通設定）リファレンス

> **検証の正**: 設定が正しいかは下記の検証コマンドの結果で判断してください。本ドキュメントと実際の挙動が食い違う場合は、スキルとバイナリの版ずれが原因のことがあります。SKILL.md の「バージョン整合チェック」に従ってスキル / バイナリを揃えてください。

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
| `url` | string | 任意 | VOICEVOX Engine のURL（単一サーバーモード）。`engines` と同時指定不可 |
| `engines` | map[string]VoicevoxEngineConfig | 任意 | 名前付き複数 VOICEVOX 互換サーバー（例: VOICEVOX NEMO 併用）。`url` と同時指定不可 |
| `startup_timeout_seconds` | int | 任意 | 音声合成前に VOICEVOX の起動を待つ最大秒数。省略時はデフォルト 60 秒。`0` で待機を無効化。全エンジン共通で、定義済み全エンジンへ並行適用される |
| `presets` | *VoicevoxPresets | 任意 | 抑揚・音高・話速プリセット定義。省略時はコード組込みのデフォルトプリセットが適用される。全エンジン共通 |

`url` は環境変数 `VOX_RADIO_VOICEVOX_URL` で上書きできます。解決順は `VOX_RADIO_VOICEVOX_URL`（環境変数）> `voicevox.url` > 既定値 `http://localhost:50021` です。`url` のみを指定した設定は、暗黙のエンジン名 `default` として動作します（後方互換）。

### `voicevox.engines.<name>` サブフィールド

複数の VOICEVOX 互換サーバー（例: 通常の VOICEVOX Engine と VOICEVOX NEMO）を名前付きで定義し、`characters.<id>.engine` でキャラクターごとに使用エンジンを切り替えられます。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `url` | string | エンジンごとの環境変数上書きがなければ必須 | このエンジンの URL |

- エンジン名は `[a-z0-9_-]+`（英小文字・数字・`-`・`_`）のみ使用できます。
- エンジンごとの URL は環境変数 `VOX_RADIO_VOICEVOX_URL_<エンジン名を大文字化し `-` を `_` に置換した名前>` で上書きできます（例: `nemo` → `VOX_RADIO_VOICEVOX_URL_NEMO`）。
- エンジン名 `default` のみ、既存の `VOX_RADIO_VOICEVOX_URL`（エンジン別の環境変数がない場合）と既定値 `http://localhost:50021`（`url` 省略時）が追加のフォールバックとして働きます。それ以外のエンジン名は `url` かエンジン別環境変数のいずれかが必須です。
- 正規化後の環境変数名が複数のエンジン名で衝突する場合（例: `nemo-v2` と `nemo_v2`）はエラーになります。
- 起動時に定義済み全エンジンへ並行で疎通確認（readiness チェック）が行われます。

```yaml
voicevox:
  engines:
    default:
      url: http://localhost:50021
    nemo:
      url: http://localhost:50121
characters:
  anneli:
    engine: nemo   # このキャラクターは nemo エンジンで合成する（省略時は default）
```

### `voicevox.presets` サブフィールド

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `intonation` | map[string]float64 | 任意 | 抑揚プリセット（intonationScale, 0.0〜2.0）。省略時はデフォルト7段階が適用される |
| `pitch` | map[string]float64 | 任意 | 音高プリセット（pitchScale, -0.15〜0.15）。省略時はデフォルト7段階が適用される |
| `speed` | map[string]float64 | 任意 | 話速プリセット（speedScale, 0.5〜2.0）。省略時はデフォルト7段階が適用される |

## `pronunciation` セクション

固有名詞の読み方辞書（任意・トップレベル）。`表記: 読み方` のマップで、人名・作品名・略語など誤読しやすい語を意図した読みで読ませたいときに使います。省略時は空（置換なし）。

```yaml
pronunciation:
  宮本武蔵: みやもとむさし
  源氏物語: げんじものがたり
  NHK: えぬえいちけー
```

- 台本テキストを LLM による読み変換にかける**前段**で、セリフ中に現れた登録表記を読み方へ置換します。置換済みのテキストが LLM に渡るため、LLM は登録した読みを引き継いでかな化します。同じ表記が複数回現れればすべて置換されます。
- 未登録の語は変換されず、そのまま残ります。
- 登録した表記が部分的に重なる場合は、長い表記を優先して置換します（例: `東京` と `東京駅` を登録すると `東京駅` を優先）。
- 同じ表記に複数の読み方を定義（読みの衝突）すると、YAML の重複キーとして読み込み時にエラーになります。

## `cache` セクション

キャッシュは常に有効です。エピソード履歴は `episode-spec.yaml` の `program.id`（必須）をキーに JSONL へ保存されます。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `max_entries` | int | 任意 | JSONL に保持する最大エピソード数（超過分は古い行から削除）。デフォルト: 100 |
| `retention_days` | int | 任意 | 保持日数（超過した古い行は削除）。デフォルト: 90 |
| `llm_context_entries` | int | 任意 | LLM へ渡す直近エピソード件数。デフォルト: 10 |

## `retro` セクション

`retro` コマンド（過去回の分析から課題と施策を抽出し、番組生成の台本生成へ注入する自動改善ループ）の設定です。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `max_tries` | int | 任意 | 同時に試行中の問題数の上限。デフォルト: 3 |
| `analysis_entries` | int | 任意 | retro が参照する直近の分析件数。デフォルト: 5 |
| `keep_threshold` | int | 任意 | 実証済み（keep）へ昇格させるのに必要な、問題が連続で発生しなかった回数。デフォルト: 3 |
| `keep_length` | int | 任意 | keep の分量の警告しきい値（文字数）。超えても切り捨てない（警告のみ）。デフォルト: 600 |
| `max_fails` | int | 任意 | 同じ問題が連続で再発してよい回数の上限。超えると試行中の問題一覧から破棄され（`dropped` へ記録）、再提案されなくなる。デフォルト: 5 |

## `script` セクション

台本生成（write ステップ）が書いたコーナーの文字数が `corners[].target_chars`（非推奨の `length_sec` 使用時は換算後の文字数）から乖離しているとき、コーナー単位で書き直す `regenIfNeeded` の設定です。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `regen_threshold` | float64 | 任意 | 書き直しを行う目標文字数からの乖離率（`\|実際の文字数 − target_chars\| ÷ target_chars`）。コーナーごとに判定する。デフォルト: 0.20（20%） |
| `regen_max_retries` | int | 任意 | 乖離が続くコーナーへの書き直し試行回数の上限。上限に達しても乖離が閾値を超えたままの場合は諦めて警告ログを出す。デフォルト: 1 |

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
| `name` | string | 任意 | キャラクターの表示名。省略/空欄にすると匿名キャラクターとして扱われ、名前が LLM に渡らず、他の出演者もこの人物への呼びかけ自体を行わない |
| `pronoun` | string | 任意 | 一人称代名詞（台本生成時に LLM へ渡す） |
| `speech_suffix` | []string | 任意 | 語尾パターン（台本生成時に LLM へ渡す） |
| `personality` | []string | 任意 | 性格特徴（台本生成時に LLM へ渡す） |
| `default_style` | string | 任意 | デフォルトの音声スタイル名（`styles` のキーと一致させること） |
| `styles` | map[string]int | 任意 | スタイル名 → VOICEVOX 話者ID のマップ |
| `credit` | string | 任意 | キャラクターのクレジット表記（例: `VOICEVOX:ずんだもん`）。設定すると manifest の `credits` へ自動収集される |
| `engine` | string | 任意 | 音声合成に使う `voicevox.engines` のエンジン名。省略時は `default` |

`default_style` を指定した場合、その値は `styles` のキーとして存在しなければなりません（起動時検証あり）。

`engine` を指定した場合、`voicevox.engines` に同名のエンジンが定義されていなければなりません（起動時検証あり）。`voicevox.engines` を使わない単一サーバーモードでは `engine` は省略するか `default` のみ指定できます。

## `slack` セクション

Slack 通知（`slackpost` コマンド）で使う Bot トークンの環境変数名を指定します。省略可能で、Slack 連携を使わない場合は不要です。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `bot_token_env` | string | 任意 | Slack Bot トークン（`xoxb-...`）を格納する環境変数名。`slackpost` での Slack 投稿時に使用 |

通知先チャンネルやメッセージテンプレートなど、配信ごとの詳細設定は `slack-spec.yaml` 側で行います。フォーマットは `references/slack-spec.md` を参照してください。
