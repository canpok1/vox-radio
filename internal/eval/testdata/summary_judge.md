# 番組要約プロンプト評価（LLM-as-judge）

あなたはラジオ番組の要約タスクの評価者です。
以下のセリフ一覧・正解説明・要約結果を見て、4つの観点でそれぞれ1〜5点（整数）で採点してください。

## セリフ一覧

```json
{{script_lines}}
```

## 正解説明（expectation）

{{expectation}}

> `expectation` が「（なし）」の場合はセリフ一覧から妥当性を自律判断してください。

## 要約結果

```json
{{summary_output}}
```

## 観点定義

| 観点キー | 説明 |
|---|---|
| `summary_quality` | 要約品質: `summary` がこの回固有の内容を具体的に記述しており、固定説明文でなく `summary_length` 程度の分量か。リスナーがエピソードを選ぶ参考になるか。 |
| `episode_title_quality` | サブタイトル品質: `episode_title` が15〜30文字程度・番組名や「第○回」等の固定フレーズを含まない・この回の内容を端的に表しているか。 |
| `notes_faithfulness` | 会話メモ忠実性: `conversation_notes` の各メモが実際に語られた・起きたことのみを記述しており、創作・推測を含まないか。`character_ids` がセリフ一覧の `speaker` に実在するIDのみを使用しているか。 |
| `notes_coverage` | 会話メモ網羅性: 事実の紹介・解説以外の会話要素（近況・掛け合い・感想・ハプニング・継続ネタ等）を幅広く拾えているか。セリフに個人的な会話が無い場合は空配列が正解。 |

## 採点ガイドライン

- `expectation` がある場合: それを正解基準として各観点を採点する。
- `expectation` が「（なし）」の場合: セリフ一覧から重要な要点と要約の質を自律判断して採点する。
- `summary` が汎用的な番組説明文（「この回ではさまざまな話題を取り上げました」等）のみで具体性がない場合、`summary_quality` は減点する。
- `episode_title` が「第○回」「番組名」等の固定フレーズを含む場合や、30文字を大きく超える・15文字を大きく下回る場合は `episode_title_quality` を減点する。
- セリフに書かれていない事実や創作を `conversation_notes` に含む場合、`notes_faithfulness` は大幅に減点する。
- セリフ一覧に存在しない `speaker` のIDを `character_ids` に使った場合、`notes_faithfulness` は大幅に減点する。
- セリフに個人的な会話・近況・掛け合いが存在するのに `conversation_notes` が空配列の場合、`notes_coverage` は減点する。
- セリフが純粋な事実紹介・解説のみで個人的な会話要素がない場合、`conversation_notes` が空配列なら `notes_coverage` は満点（5点）とする。
- 採点後は各観点の `reason` に採点理由を簡潔に記載する。

## 出力形式

```json
{
  "scores": [
    {"criterion": "summary_quality", "score": 5, "reason": "採点理由"},
    {"criterion": "episode_title_quality", "score": 5, "reason": "採点理由"},
    {"criterion": "notes_faithfulness", "score": 5, "reason": "採点理由"},
    {"criterion": "notes_coverage", "score": 5, "reason": "採点理由"}
  ]
}
```
