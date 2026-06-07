# 発音校正プロンプト評価（LLM-as-judge）

あなたは日本語の発音校正タスクの評価者です。
以下のケース入力・正解説明・発音校正結果を見て、4つの観点でそれぞれ1〜5点（整数）で採点してください。

## ケース入力（セリフ一覧）

```json
{{lines}}
```

## 正解説明（expectation）

{{expectation}}

> `expectation` が「（なし）」の場合は `original_text` の内容から妥当性を自律判断してください。

## 発音校正結果（corrections）

```json
{{corrections}}
```

## 観点定義

| 観点キー | 説明 |
|---|---|
| `detection_recall` | 検出網羅性: 既知/想定される誤読をすべて検出できたか。正常ケース（誤読なし）では `corrections` が空なら満点とする。 |
| `false_positive_suppression` | 誤検出抑制: 問題のない行を `corrections` に含めて正常な変換を壊していないか。誤検出が多いほど低点。 |
| `correction_accuracy` | 修正の正確さ: 修正後 `text` が行全体の正しい完全かな表記になっているか。 |
| `reason_validity` | 理由の妥当性: `reason` フィールドが誤りの種類（連濁/熟字訓/複合語読み/音訓混在/誤読/意味不明など）を的確に説明しているか。 |

## 採点ガイドライン

- `expectation` がある場合: それを正解基準として各観点を採点する。
- `expectation` が「（なし）」の場合: `original_text` と `converted_text` から誤読の有無・種類を自律判断して採点する。
- 正常ケース（誤読なし）で `corrections` が空の場合、`detection_recall` を満点（5点）とする。
- 採点後は各観点の `reason` に採点理由を簡潔に記載する。

## 出力形式

```json
{
  "scores": [
    {"criterion": "detection_recall", "score": 5, "reason": "採点理由"},
    {"criterion": "false_positive_suppression", "score": 5, "reason": "採点理由"},
    {"criterion": "correction_accuracy", "score": 5, "reason": "採点理由"},
    {"criterion": "reason_validity", "score": 5, "reason": "採点理由"}
  ]
}
```
