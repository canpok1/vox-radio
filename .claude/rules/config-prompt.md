# 設定YAML・プロンプト編集ルール

対象ファイル: `*.yaml` / `profiles/**/*.yaml` / `prompts/*.md` / `internal/config/testdata/*.yaml`

## 設定YAMLの編集

`vox-radio.yaml` や `profiles/**/profile.yaml` など設定YAMLを編集するときは、以下を確認すること。

- **デフォルト値はコード定数から確認する** — Issue本文や設計ドキュメントの記載よりGoコードを正とする。`grep -r 'Default' internal/` で `const DefaultXxx` を確認してから記入する。
- **設定フィールドを追加・変更したらコード配線を確認する** — フィールドを追加・変更したら `grep -rn 'フィールド名' internal/` でコード参照箇所を確認し、宣言だけで未使用（サイレント無効化）になっていないか検証する。

## プロンプトファイルの編集

`prompts/*.md` を編集するときは、以下を確認すること。

- **出力JSONスキーマは `internal/model/*.go` と照合する** — プロンプトに記載するJSONフィールドは `internal/model/*.go` の `json` タグと1フィールドずつ照合し、設計ドキュメントのサンプルではなくGoの構造体定義を正とする。

## テストデータの編集

`internal/config/testdata/*.yaml` などテストデータを編集するときは、以下を確認すること。

- **必須フィールドを明示的に設定する** — `PubDate` 等の必須フィールドを省略すると後でテスト失敗になるため、すべての必須フィールドを明示的に含めること。
