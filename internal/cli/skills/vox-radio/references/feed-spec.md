# feed-spec.yaml（フィード生成設定）リファレンス

> **検証の正**: 設定が正しいかは下記の検証コマンドの結果で判断してください。本ドキュメントと実際の挙動が食い違う場合は、スキルとバイナリの版ずれが原因のことがあります。SKILL.md の「バージョン整合チェック」に従ってスキル / バイナリを揃えてください。

> **検証コマンド**: `vox-radio feedgen check`

`feed-spec.yaml` は `vox-radio feedgen` / `vox-radio feedgen check` が使用するフィード生成設定ファイルです。`vox-radio init` で生成されるテンプレートを元に編集してください。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `feed.language` | string | 必須 | 言語コード（RSS channel language）。例: `ja` |
| `feed.email` | string | 必須 | 連絡先メールアドレス（itunes:email） |
| `feed.site_url` | string | 必須 | 番組サイト URL（RSS channel link） |
| `feed.audio_url_template` | string | 必須 | 各エピソード音声ファイルの URL テンプレート。`{episode_number}` と `{audio_file}` が cache の値で置換される |
| `feed.category` | string | 任意 | iTunes カテゴリ。空文字でタグ省略 |
| `feed.explicit` | bool | 任意 | 露骨な表現の有無（itunes:explicit）。デフォルト: false |
| `feed.cover_image_url` | string | 任意 | カバー画像 URL（itunes:image）。空文字でタグ省略 |
| `feed.credit` | string | 任意 | 配信者クレジット表記（各 item の itunes:author）。空文字で省略 |
| `feed.credits_header` | string | 任意 | `<description>` 内クレジット節の見出し文字列。デフォルト: `クレジット` |
| `output.public` | string | 任意 | `feed.xml` を書き出すディレクトリ。デフォルト: `public` |

必須フィールドの欠落・URL/email 形式・`audio_url_template` のプレースホルダ（`{episode_number}` / `{audio_file}`）は `vox-radio feedgen check` で検証されます。意味検証エラーは全件まとめて報告されます。

## アセット・キャラクターのクレジット自動転記

`vox-radio episodegen manifest --lines 03_lines.json --script 04_script.json` を実行すると、実際に使ったアセット・キャラクターの `credit` フィールドが自動収集されて `manifest.json` の `credits` へ格納されます。

フィード生成時、`credits` が非空のエピソードは各 item の `<description>` 末尾に以下の形式でクレジット節が自動追記されます。

```
{要約テキスト}

クレジット
OtoLogic https://otologic.jp / CC BY 4.0
VOICEVOX:ずんだもん
```

アセット設定（`assets.yaml`）と共通設定（`vox-radio.yaml` の `characters`）の各エントリに `credit` フィールドを記入することで有効になります。
