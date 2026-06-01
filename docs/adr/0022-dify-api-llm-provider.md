# 0022. LLM プロバイダに Dify API 経由の利用を追加する

- ステータス: 採用
- 日付: 2026-06-01

## コンテキスト

ADR 0002 は LLM プロバイダ切替を **OpenAI 互換 1 実装 ＋ `base_url`/`model` 差し替え**に統一し、列挙型での分岐を却下した。一方で LLM を Dify アプリ経由でも利用したい要望が生じた。Dify はアプリ種別ごとにエンドポイントと入出力が異なり、いずれも `model`/`temperature`/`json_schema` は **Dify アプリ側設定に固定**され送信できない。このため `base_url`/`model` 差し替えでは吸収できない。

## 決定

`LLMConfig` に `provider` 列挙（`openai`/`dify-chat`、既定 `openai`）を追加し、`llm` パッケージ内の factory で実装を選択する。接続情報（`base_url`/`api_key_env`）と `model` は provider 別ブロックへ分離し、温度・`steps` 等は共通に保つ。`Client` インターフェース（`Complete`）は変更しない。Dify はアプリ種別でエンドポイントが異なるため **プロバイダ名に種別を含めて明示する**（`dify-chat` は `chat-messages` 専用。将来は `dify-workflow` を別実装で追加）。`dify-chat` は **ゲートウェイ**として使い、プロンプトを `query` に渡し `answer` を得る（JSON Schema 検証は vox-radio 側に温存）。`json_schema` は送れないため構造化出力は既存のスキーマ検証＋自己修復リトライで担保する。`temperature` は `dify-chat.inputs` の `${temperature}` プレースホルダーで per-step 値を渡せる（任意）。HTTP は `safejob/dify-sdk-go`（依存ゼロ・MIT）の blocking 呼び出しを用いる。

## 結果

- `provider` で OpenAI/Dify を切替える。接続情報・`model` は provider 別ブロックに再編し、既存 yaml は移行要。
- `Client` 境界は不変のため、ドメイン層と各ステップは影響を受けない。
- ADR 0002 の「列挙型を持たない」原則は崩すが、ワイヤープロトコルの統一思想は OpenAI 側で維持する。
- Dify 利用時は strict schema が効かず品質は検証＋リトライ依存。温度は Dify アプリ側で変数バインドした場合のみ効く。検証・修復ロジックは両実装で共有する。

## 検討した代替案

- **Dify の OpenAI 互換エンドポイントを使い ADR 0002 のまま吸収**: ライブラリ経由という要望に反し、Dify アプリ機能も活用できないため却下。
- **ステップごとに Dify アプリを分け per-step 制御を維持**: API キー・アプリ運用が 6 倍に膨らみ割に合わないため却下（単一アプリで割り切る）。
- **Dify 側にプロンプトを移管**: `prompts/*.md` と動的スキーマ生成の大規模な責務移動を伴い、変更範囲が過大なため却下。
- **1 プロバイダ＋ `app_type` で chat/workflow 両対応**: 設定・実装が複雑化するため却下し、種別を provider 値で分けた。
