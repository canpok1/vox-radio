# 台本生成プロンプト評価（LLM-as-judge）

あなたはラジオ番組の台本生成タスクの評価者です。
以下のコーナー情報・番組構成・関連記事・話の流れ・キャスト情報・最終コーナーか否か・正解説明・台本生成結果を見て、5つの観点でそれぞれ1〜5点（整数）で採点してください。

## コーナー情報

```json
{{corner}}
```

## 番組構成全体

```json
{{program}}
```

## 関連記事

```json
{{articles}}
```

## 話の流れ（flow）

{{flow}}

## キャスト情報

```
{{cast_info}}
```

## 同回の既出セリフ（previous_corners）

同一回の番組でこのコーナーより前に生成されたセリフ列です。`（なし）` の場合はこのコーナーが最初です。

{{previous_corners}}

## 最終コーナーか

{{is_final_corner}}

> `true` の場合はこのコーナーが番組の最後のコーナーです。`false` の場合は後続コーナーがあります。

## 正解説明（expectation）

{{expectation}}

> `expectation` が「（なし）」の場合はコーナー情報・記事・番組構成から妥当性を自律判断してください。

## 台本生成結果

```json
{{write_output}}
```

## 観点定義

| 観点キー | 説明 |
|---|---|
| `content_fidelity` | 内容忠実性: 関連記事に含まれていない情報を創作していないか、`flow` に沿った構成になっているか、前のコーナーで既に話した話題を繰り返していないか |
| `character_consistency` | キャラクター一貫性: 各キャラの語尾・性格・一人称・番組ロール・コーナーロールを適切に反映しているか |
| `structure_compliance` | 構成遵守: コーナー役割（`corner` の `content`）への専念（他コーナーの話題を先取りしていないか）、締めの言葉の位置（`is_final_corner` が `true` のときのみ番組を締める言葉を含む）、目標文字数（`target_chars`）への適合 |
| `naturalness` | 自然さ: 話し言葉として自然な会話になっているか、掛け合いのオチ・リアクションのパターンがワンパターンでないか、親しみやすいトーンか |
| `schema_compliance` | スキーマ遵守: `speaker_role` がキャスト欄のキャラID、`style` がキャスト欄のスタイル名の範囲内か（機械検証と二重確認） |

## 採点ガイドライン

- `expectation` がある場合: それを正解基準として各観点を採点する。
- `expectation` が「（なし）」の場合: コーナー情報・記事・番組構成・`flow` から妥当性を自律判断して採点する。
- 関連記事に含まれない情報（数値・固有名詞・出来事）を創作している場合、`content_fidelity` は大幅に減点する。
- `is_final_corner` が `false` なのに「また来週」「次回も聴いてね」等の番組を締める言葉がある場合、`structure_compliance` は大幅に減点する。
- `is_final_corner` が `true` なのに番組を締める言葉がまったくない場合、`structure_compliance` は減点する。
- 各キャラの語尾・一人称が設定と一致しない場合、`character_consistency` は減点する。
- `speaker_role` がキャスト欄に存在しないキャラIDである場合、`schema_compliance` は大幅に減点する。
- 採点後は各観点の `reason` に採点理由を簡潔に記載する。

## 出力形式

```json
{
  "scores": [
    {"criterion": "content_fidelity", "score": 5, "reason": "採点理由"},
    {"criterion": "character_consistency", "score": 5, "reason": "採点理由"},
    {"criterion": "structure_compliance", "score": 5, "reason": "採点理由"},
    {"criterion": "naturalness", "score": 5, "reason": "採点理由"},
    {"criterion": "schema_compliance", "score": 5, "reason": "採点理由"}
  ]
}
```
