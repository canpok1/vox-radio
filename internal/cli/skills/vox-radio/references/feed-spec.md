# feed-spec.yaml（フィード生成設定）リファレンス

> **フィールド定義の正**: `internal/model/feed_spec.go`。本ドキュメントとコードが齟齬する場合はコードを優先してください。

> **検証コマンド**: `vox-radio feedgen check`

`feed-spec.yaml` は `vox-radio feedgen` / `vox-radio feedgen check` が使用するフィード生成設定ファイルです。`vox-radio init` で生成されるテンプレートを元に編集してください。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `program_id` | string | 必須 | feedgen がキャッシュから対象エピソードを絞り込むキー。`episode-spec.yaml` の `program.id` と一致させること |
| `feed.language` | string | 必須 | 言語コード（RSS channel language）。例: `ja` |
| `feed.author` | string | 必須 | 配信者名（itunes:author） |
| `feed.email` | string | 必須 | 連絡先メールアドレス（itunes:email） |
| `feed.site_url` | string | 必須 | 番組サイト URL（RSS channel link） |
| `feed.audio_url_template` | string | 必須 | 各エピソード音声ファイルの URL テンプレート。`{episode_number}` と `{audio_file}` が cache の値で置換される |
| `feed.category` | string | 任意 | iTunes カテゴリ。空文字でタグ省略 |
| `feed.explicit` | bool | 任意 | 露骨な表現の有無（itunes:explicit）。デフォルト: false |
| `feed.cover_image_url` | string | 任意 | カバー画像 URL（itunes:image）。空文字でタグ省略 |
| `feed.credit` | string | 任意 | クレジット表記（各 item の itunes:author）。空文字で省略 |
| `output.public` | string | 任意 | `feed.xml` を書き出すディレクトリ。デフォルト: `public` |

必須フィールドの欠落・URL/email 形式・`audio_url_template` のプレースホルダ（`{episode_number}` / `{audio_file}`）は `vox-radio feedgen check` で検証されます。意味検証エラーは全件まとめて報告されます。
