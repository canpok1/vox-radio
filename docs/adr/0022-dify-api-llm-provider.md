# 0022. LLM プロバイダに Dify API 経由の利用を追加する

- ステータス: 採用
- 日付: 2026-06-01

## コンテキスト

ADR 0002 は LLM プロバイダ切替を **OpenAI 互換 1 実装 ＋ `base_url`/`model` 差し替え**に統一し、列挙型での分岐を却下した。一方で LLM を Dify アプリ経由でも利用したい要望が生じた。Dify のアプリ API（`chat-messages` 等）は OpenAI 互換ではなく、リクエストでは `query` 等の限られた項目しか受け取らず、`model`・`temperature`・`response_format`(JSON Schema) は **Dify アプリ側設定に固定**され送信できない。このため `base_url`/`model` 差し替えでは吸収できず、ADR 0002 の前提が崩れる。

## 決定

`LLMConfig` に `provider` 列挙（`openai`/`dify`、既定 `openai`）を追加し、`llm` パッケージ内の factory で実装を選択する。`Client` インターフェース（`Complete`）は変更しない。Dify は **ゲートウェイ**として使い、vox-radio が組み立てた system+user プロンプトを `query` に渡す（プロンプトと JSON Schema 検証は vox-radio 側に温存）。`model`/`temperature`/`json_schema` は送れないため per-step temperature は Dify 利用時は無効とし、構造化出力は既存のスキーマ検証＋自己修復リトライで担保する。Dify アプリは単一とし、HTTP は `safejob/dify-sdk-go`（依存ゼロ・blocking chat 対応・MIT）の `RunBlock` を用いる。

## 結果

- 設定値 1 つで OpenAI 互換と Dify を切替でき、既定 `openai` で後方互換を維持できる。
- `ScriptGenerator`/`Client` 境界は不変のため、ドメイン層と各ステップは影響を受けない。
- ADR 0002 の「列挙型を持たない」原則は崩すが、対象は 2 値の限定的な分岐に留め、ワイヤープロトコルの統一思想は OpenAI 側で維持する。
- Dify 利用時は per-step temperature と strict schema が効かず、品質は検証＋リトライ依存になる。検証・修復ロジックは両実装で共有して重複を避ける。

## 検討した代替案

- **Dify の OpenAI 互換エンドポイントを使い ADR 0002 のまま吸収**: ライブラリ経由という要望に反し、Dify アプリ機能も活用できないため却下。
- **ステップごとに Dify アプリを分け per-step 制御を維持**: API キー・アプリ運用が 6 倍に膨らみ割に合わないため却下（単一アプリで割り切る）。
- **Dify 側にプロンプトを移管**: `prompts/*.md` と動的スキーマ生成の大規模な責務移動を伴い、変更範囲が過大なため却下。
