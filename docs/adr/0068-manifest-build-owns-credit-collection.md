# 0068. manifest.Build がクレジット収集を内包する

- ステータス: 採用
- 日付: 2026-06-14

## コンテキスト

`manifest.Build` の呼び出し元が2箇所（`internal/cli/manifest.go` と `internal/pipeline/pipeline.go`）あり、両者がそれぞれ `manifest.CollectCredits` を呼んで `BuildParams.Credits` に手配線する設計だった。フルパイプライン `episodegen` で `pipeline.go` 側の配線が漏れて manifest に credits が転記されないバグが発生した（#445）。バグ修正で配線は追加されたが、「呼び出し側が Credits を手で渡す」という手配線忘れリスク構造が残った。

## 決定

`manifest.Build` が内部で `CollectCredits` を呼ぶ。`BuildParams.Credits []string` を除去し、代わりにクレジット収集に必要なソースデータ（`Assets`, `Characters`, `Lines`, `Script`）を `BuildParams` のフィールドとして持つ。`Casts` は既存の `Rundown.Casts` から取得する。

## 結果

- **良い面**: 呼び出し側が Credits を渡し忘れるバグクラスを設計で消せる。`manifest.Build` の責務が「manifest を構築する」で一貫し、Credits だけ漏れるシームがなくなる。
- **悪い面**: `BuildParams` にクレジット収集用フィールドが増え、Credits だけをテストで直接注入できなくなる。
- **トレードオフ**: `CollectCredits` は LLM 不要の純粋関数のため Build に内包しても副作用は生じない。

## 検討した代替案

- **現状維持（手配線）**: 呼び出し側が毎回 `CollectCredits` を呼ぶ。配線忘れのリスクが残るため却下。
- **Build の引数を CreditParams に変える**: `BuildParams` に `CreditParams` をネストする案。フラット化と大差なく、既存コードとの乖離が大きくなるため却下。
