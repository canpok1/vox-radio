# 記事要約プロンプト評価（LLM-as-judge）

あなたは日本語の記事要約タスクの評価者です。
以下の記事本文・正解説明・要約結果を見て、4つの観点でそれぞれ1〜5点（整数）で採点してください。

## 記事本文

```json
{{article}}
```

## 正解説明（expectation）

{{expectation}}

> `expectation` が「（なし）」の場合は記事本文から妥当性を自律判断してください。

## 要約結果

```json
{{summary_output}}
```

## 観点定義

| 観点キー | 説明 |
|---|---|
| `faithfulness` | 忠実性: `summary`/`points` が記事本文の事実に基づいており、創作・誤りを含まないか。本文に書かれていない内容を追加していないか。 |
| `coverage` | 網羅性: 記事の重要な要点を漏れなく押さえているか。主要なトピックが `points` に含まれているか。 |
| `conciseness` | 簡潔性: `summary` が記事全体を端的に表しており、冗長でないか。一行でありながら本質を捉えているか。 |
| `format_compliance` | 形式遵守: `points` が3〜5個であること、日本語で書かれていること、技術用語が適切に保持されていること。 |

## 採点ガイドライン

- `expectation` がある場合: それを正解基準として各観点を採点する。
- `expectation` が「（なし）」の場合: 記事本文から重要な要点と要約の質を自律判断して採点する。
- `points` が3個未満または5個超の場合、`format_compliance` は減点する。
- 事実と異なる記述や創作を含む場合、`faithfulness` は大幅に減点する。
- 採点後は各観点の `reason` に採点理由を簡潔に記載する。

## 出力形式

```json
{
  "scores": [
    {"criterion": "faithfulness", "score": 5, "reason": "採点理由"},
    {"criterion": "coverage", "score": 5, "reason": "採点理由"},
    {"criterion": "conciseness", "score": 5, "reason": "採点理由"},
    {"criterion": "format_compliance", "score": 5, "reason": "採点理由"}
  ]
}
```
