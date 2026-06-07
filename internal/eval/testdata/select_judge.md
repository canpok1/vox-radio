# 記事選別プロンプト評価（LLM-as-judge）

あなたはラジオ番組の記事選別タスクの評価者です。
以下のキャスト情報・コーナー情報・候補記事・正解説明・選別結果を見て、4つの観点でそれぞれ1〜5点（整数）で採点してください。

## キャスト情報

```json
{{casts}}
```

## コーナー情報

```json
{{corner}}
```

## 候補記事（タイトルと URL）

```json
{{articles}}
```

## 正解説明（expectation）

{{expectation}}

> `expectation` が「（なし）」の場合はコーナー情報・候補記事から妥当性を自律判断してください。

## 選別結果

```json
{{select_output}}
```

## 観点定義

| 観点キー | 説明 |
|---|---|
| `relevance` | 適合性: コーナーの趣旨（`content`）・目標放送時間（`target_duration_seconds`）に合った記事を選べているか。コーナーの趣旨と無関係な記事を選んでいないか。 |
| `constraint_compliance` | 制約遵守: `selected_urls` が候補記事内の URL のみを使用しているか・最低1件以上選ばれているか・URL 形式が正しいか。 |
| `ordering_quality` | 紹介順の質: 選ばれた記事の紹介順が番組の流れとして自然・妥当か。関連性の高い記事が適切な順序で並んでいるか。 |
| `reason_validity` | 選別理由の妥当性: `selection_reason` が選択意図・紹介順の根拠を的確に説明しているか。なぜその記事をその順で選んだか理由が明確か。 |

## 採点ガイドライン

- `expectation` がある場合: それを正解基準として各観点を採点する。
- `expectation` が「（なし）」の場合: コーナー情報・候補記事から妥当性を自律判断して採点する。
- 候補に存在しない URL が `selected_urls` に含まれる場合、`constraint_compliance` は大幅に減点する。
- `selected_urls` が空の場合（最低1件制約違反）、`constraint_compliance` は大幅に減点する。
- コーナー趣旨と明らかに無関係な記事が多く選ばれている場合、`relevance` は減点する。
- `selection_reason` が空文字や「理由なし」のみの場合、`reason_validity` は大幅に減点する。
- 採点後は各観点の `reason` に採点理由を簡潔に記載する。

## 出力形式

```json
{
  "scores": [
    {"criterion": "relevance", "score": 5, "reason": "採点理由"},
    {"criterion": "constraint_compliance", "score": 5, "reason": "採点理由"},
    {"criterion": "ordering_quality", "score": 5, "reason": "採点理由"},
    {"criterion": "reason_validity", "score": 5, "reason": "採点理由"}
  ]
}
```
