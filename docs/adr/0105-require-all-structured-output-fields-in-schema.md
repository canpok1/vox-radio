# 0105. 構造化出力で必ず得たいフィールドは JSON Schema の required に明示する（ADR-0104 改訂）

- ステータス: 採用
- 日付: 2026-08-06

## コンテキスト

ADR-0104 実装後に実機（`gemini-3.1-flash-lite`）で通したところ、speech セグメント 64 件すべてで `intonation` / `pitch` / `speed` が空だった。生レスポンスに返っていたのは `insertions` と `pause_insertions` だけで、`line_voices` は**キーごと存在しなかった**。

`internal/script/llm` は `response_format: json_schema` を `strict: true` で送る。この形式では required に無いプロパティは生成されず、プロンプトに「全セリフを対象にする」と書いても効かない。direct のスキーマは required が `["insertions"]` だけだった。

同じ理由で `line_conversions` も返らず、助詞の「は→わ」変換が効いていなかった。ADR-0104 以前からの状態で、ADR-0103 が挙げた誤読多発の一因でもあったとみられる。

## 決定

direct のレスポンススキーマで、必ず得たいものを required に明示する。

1. トップレベルの required に `insertions` / `pause_insertions` / `line_conversions` / `line_voices` の全配列を並べる。
2. `line_voices` の要素では `intonation` のみ required に加え、`pitch` / `speed` は任意のまま残す。

`intonation` だけ必須にするのは実機比較による。3 つとも必須にすると `line_voices` が最初のコーナー分（11/64 件）で尽きる（2 回とも再現）。`intonation` のみなら全 64 件が返り、任意の `pitch` / `speed` も生成される。

`direct.md` の「フィールドごと省略してかまいません」も、`intonation` が必須である旨へ改める（ADR-0104 の当該決定の改訂）。

## 結果

- 良い影響: 抑揚が全セリフに付き、同一話者の連続セリフでも感情の切れ目で切り替わる（ADR-0104 の狙いが実際に得られる）。`line_conversions` も復活する。
- 悪い影響: レスポンスが必ず全セリフ分になり direct の出力トークンが増える。`pitch` / `speed` はモデル次第で欠落しうる非対称な仕様が残る。
- 波及: strict でスキーマを送る他ステップにも同じ罠があり、required 漏れは横断的に点検する価値がある。

## 検討した代替案

- **プロンプトの指示を強める**: required が優先されるため、文言を変えても生成されない。
- **`pitch` / `speed` も必須にする**: 出力が最初のコーナーで尽き、大半のセリフが無指定に戻る。
- **`strict: false` / `json_object` へ落とす**: 任意フィールドは返るが、enum 逸脱や型崩れの検証がリトライ任せになりスキーマ検証の利点を失う。
