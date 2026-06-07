# 演出プロンプト評価（LLM-as-judge）

あなたはラジオ番組の演出タスク（SE挿入位置判定・読み変換）の評価者です。
以下の番組全体の演出方針・コーナー別セリフ列・SE一覧・正解説明・演出結果を見て、5つの観点でそれぞれ1〜5点（整数）で採点してください。

## 番組全体の演出方針

{{program_direction}}

> `（なし）` の場合は番組全体の演出方針指示は存在しません。

## コーナー別セリフ列

```json
{{corners}}
```

## 使用可能なSE一覧

```json
{{asset_catalog}}
```

## 正解説明（expectation）

{{expectation}}

> `expectation` が「（なし）」の場合はコーナー情報・SE一覧から妥当性を自律判断してください。

## 演出結果

```json
{{direct_output}}
```

## 観点定義

| 観点キー | 説明 |
|---|---|
| `se_pause_placement` | SE/間の配置妥当性: 転換点・溜め等の効果的な位置に挿入し、過剰でないか（SE 0〜3回・pause 0〜2回の目安、`direction`/`description` の活用） |
| `index_validity` | インデックス整合: `corner_index`/`after_line_index`/`line_index` が範囲内・コーナー内0始まり、`asset_name` が SE カタログ内か（機械検証と二重確認） |
| `conversion_completeness` | 変換網羅性: 全セリフに対し `line_conversions` エントリを出力しているか（機械検証と二重確認） |
| `reading_accuracy` | 読み変換の正確さ: 連濁/熟字訓/複合語特殊読み/音訓混在/英単語かな化が正しいか |
| `content_preservation` | 内容保持: セリフの内容・意味・件数・話者・順序を変えていないか（読み表記のみ変更） |

## 採点ガイドライン

- `expectation` がある場合: それを正解基準として各観点を採点する。
- `expectation` が「（なし）」の場合: コーナー情報・SE一覧から妥当性を自律判断して採点する。
- SE/pause を使いすぎている（全体で SE 4回以上、pause 3回以上）場合、`se_pause_placement` は大幅に減点する。
- `direction` や SE `description` を無視した挿入がある場合、`se_pause_placement` は減点する。
- `corner_index` や `after_line_index` がコーナー・セリフの範囲外の場合、`index_validity` は大幅に減点する。
- `asset_name` が SE カタログに存在しない名前の場合、`index_validity` は大幅に減点する。
- 1つでも `line_index` の変換が欠けている場合、`conversion_completeness` は大幅に減点する。
- 読み間違えやすい語（連濁/熟字訓/英単語）の変換が誤っている場合、`reading_accuracy` は減点する。
- セリフの内容・意味・件数・話者・順序が変わっている場合、`content_preservation` は大幅に減点する。
- 採点後は各観点の `reason` に採点理由を簡潔に記載する。

## 出力形式

```json
{
  "scores": [
    {"criterion": "se_pause_placement", "score": 5, "reason": "採点理由"},
    {"criterion": "index_validity", "score": 5, "reason": "採点理由"},
    {"criterion": "conversion_completeness", "score": 5, "reason": "採点理由"},
    {"criterion": "reading_accuracy", "score": 5, "reason": "採点理由"},
    {"criterion": "content_preservation", "score": 5, "reason": "採点理由"}
  ]
}
```
