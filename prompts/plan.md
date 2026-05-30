# [A] テーマ決定プロンプト

以下の記事要約群と番組設定を元に、今日のラジオ番組の構成（rundown）を作成してください。

## 番組設定

```json
{{show_config}}
```

## 記事要約一覧

```json
{{summaries}}
```

## 出力形式

以下のJSON形式で回答してください。

```json
{
  "corners": [
    {
      "title": "コーナータイトル",
      "topic": "取り上げるトピック",
      "points": ["話すべき要点1", "話すべき要点2"],
      "target_chars": 500,
      "summary_urls": ["https://example.com/articles/123"]
    }
  ]
}
```

## 注意事項

- コーナー数は番組設定の `corners` に従ってください
- 各コーナーの `target_chars` の合計が番組設定の `target_chars` になるように配分してください
- 最も興味深い記事・トピックを優先して取り上げてください
- コーナー間でトピックが重複しないようにしてください
- `summary_urls` には関連する記事要約のURLを列挙してください
