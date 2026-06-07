# flow設計プロンプト評価（LLM-as-judge）

あなたはラジオ番組の flow 設計タスクの評価者です。
以下のコーナー情報・番組内位置・選別済み記事・選別理由・番組構成全体・正解説明・flow設計結果を見て、4つの観点でそれぞれ1〜5点（整数）で採点してください。

## コーナー情報

```json
{{corner}}
```

## 番組内位置

{{position}}

## 選別済み記事

```json
{{articles}}
```

## 選別理由

{{selection_reason}}

## 番組構成全体

```json
{{program}}
```

## 正解説明（expectation）

{{expectation}}

> `expectation` が「（なし）」の場合はコーナー情報・番組内位置・記事情報から妥当性を自律判断してください。

## flow設計結果

{{flow_output}}

## 観点定義

| 観点キー | 説明 |
|---|---|
| `position_role_fit` | 位置別役割適合: `position` の役割（opening=導入・締めない / ending=締め / middle=つなぎ）を満たしているか。opening なのに番組を締める内容になっていないか。ending なのに次の話題を引き出す流れになっていないか。 |
| `consistency` | 整合性: 番組構成全体・他コーナーと矛盾・重複しない flow か。特定コーナー名をハードコードせず番組構成から読み取った内容に基づいているか。 |
| `article_alignment` | 記事整合: 記事ありコーナーでは選別済み記事・選別理由に沿った紹介順・繋ぎになっているか。記事なしコーナーでも番組構成に沿った矛盾のない flow になっているか。 |
| `actionability` | 指針性: 台本作成の指針として具体的な構成・繋ぎ方が記述できているか。抽象的な方針にとどまらず、何をどの順で話すかが読み取れるか。 |

## 採点ガイドライン

- `expectation` がある場合: それを正解基準として各観点を採点する。
- `expectation` が「（なし）」の場合: コーナー情報・番組内位置・記事情報・番組構成から妥当性を自律判断して採点する。
- opening なのに番組を締める内容（「本日はここまで」等）がある場合、`position_role_fit` は大幅に減点する。
- 他コーナーの内容と重複した紹介をしている場合、`consistency` は減点する。
- 記事ありコーナーで記事の内容と無関係な flow の場合、`article_alignment` は大幅に減点する。
- 「自然につなぐ」のみで具体的な構成が書かれていない場合、`actionability` は減点する。
- 採点後は各観点の `reason` に採点理由を簡潔に記載する。

## 出力形式

```json
{
  "scores": [
    {"criterion": "position_role_fit", "score": 5, "reason": "採点理由"},
    {"criterion": "consistency", "score": 5, "reason": "採点理由"},
    {"criterion": "article_alignment", "score": 5, "reason": "採点理由"},
    {"criterion": "actionability", "score": 5, "reason": "採点理由"}
  ]
}
```
