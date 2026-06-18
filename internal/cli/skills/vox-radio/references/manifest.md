# manifest テンプレートフィールドリファレンス

> manifest.json を入力とする `render` / `slackpost` のテンプレートで参照できるフィールドと関数の一覧です。
>
> テンプレートは Go 標準の [`text/template`](https://pkg.go.dev/text/template) 記法を使います。

## 命名規則

manifest.json のキーは **snake_case**（例: `episode_number`）ですが、テンプレートのデータ文脈は **PascalCase**（例: `.EpisodeNumber`）です。

```
manifest.json キー → テンプレートフィールド
episode_number   → .EpisodeNumber
episode_title    → .EpisodeTitle
audio_file       → .AudioFile
```

## Manifest（ルートオブジェクト）

| フィールド | 型 | 説明 |
|---|---|---|
| `.Title` | string | 番組タイトル |
| `.EpisodeNumber` | int | 回番号（0 で偽扱い） |
| `.EpisodeTitle` | string | サブタイトル |
| `.Author` | string | 著者名 |
| `.Description` | string | 番組説明 |
| `.Summary` | string | エピソード全体の要約 |
| `.Datetime` | string | 配信日時（ISO 8601 形式の文字列） |
| `.AudioFile` | string | 音声ファイル名 |
| `.Corners` | []ManifestCorner | コーナー一覧（下記参照） |
| `.ConversationNotes` | []ConversationNote | 会話メモ一覧（下記参照） |
| `.Casts` | []RundownCast | 出演者一覧（下記参照） |
| `.Credits` | []string | クレジット文字列一覧 |

## ManifestCorner（`.Corners` の要素）

| フィールド | 型 | 説明 |
|---|---|---|
| `.ID` | string | コーナー ID |
| `.Title` | string | コーナータイトル |
| `.Summary` | string | コーナー要約 |
| `.Points` | []string | 要点リスト |
| `.Articles` | []ArticleRef | 記事一覧（下記参照） |
| `.TargetSec` | int | 目標尺（秒）。未設定の場合は 0 |
| `.SpeechSec` | float64 | 発話尺（秒）。未設定の場合は 0 |
| `.DurationSec` | float64 | 実際の再生尺（秒）。未設定の場合は 0 |
| `.CharCount` | int | 台本の文字数。未設定の場合は 0 |

## ArticleRef（`.Corners[].Articles` の要素）

| フィールド | 型 | 説明 |
|---|---|---|
| `.Title` | string | 記事タイトル |
| `.URL` | string | 記事 URL（空の場合あり） |
| `.DedupKey` | string | 内部用: 重複判定キー（sha256:hex）。テンプレートでは通常使用しない |

## ConversationNote（`.ConversationNotes` の要素）

| フィールド | 型 | 説明 |
|---|---|---|
| `.Category` | string | カテゴリラベル（例: 近況・掛け合い・感想・ハプニング・継続ネタ） |
| `.CharacterIDs` | []string | 関係するキャラクター ID 一覧（キャラクター固有でない場合は空） |
| `.Note` | string | メモ本文 |

## RundownCast（`.Casts` の要素）

| フィールド | 型 | 説明 |
|---|---|---|
| `.CharacterID` | string | キャラクター ID |
| `.Role` | string | 役割（例: MC・ゲスト） |
| `.Type` | string | 種別: `"regular"` または `"guest"` |
| `.AppearanceCount` | int | 今回を含む出演回数（1 = 初登場） |
| `.LastEpisodeNumber` | int | 前回出演回番号（未出演の場合は 0） |

### RundownCast のメソッド

| メソッド | 戻り値 | 説明 |
|---|---|---|
| `.PastAppearanceCount` | int | 今回を除く過去出演回数（AppearanceCount − 1、最小 0） |

## テンプレート関数

| 関数 | 説明 |
|---|---|
| `corner "<id>"` | 指定 ID の `*ManifestCorner` を返す（見つからない場合は `nil`） |
| `hasLinks <corner>` | コーナーに URL 付き記事が 1 件以上あれば `true` |

Go 標準の `eq` / `ne` / `if` / `range` / `with` / `index` 等もすべて使えます。

## レシピ集

### 値の抽出（CI での jq 代替）

```bash
# 回番号を取り出す
vox-radio render --manifest manifest.json --template-string '{{.EpisodeNumber}}'

# リリースタイトルを組み立てる
vox-radio render --manifest manifest.json --template-string '第{{.EpisodeNumber}}回 {{.EpisodeTitle}}'
```

### URL なし記事のスキップ

```
{{range .Corners}}{{range .Articles}}{{if .URL}}{{.Title}}
{{end}}{{end}}{{end}}
```

### 特定コーナーを ID で取り出す

```
{{with corner "news"}}
{{.Title}}: {{.Summary}}
{{end}}
```

### URL 付き記事があるコーナーだけ展開

```
{{range .Corners}}{{if hasLinks .}}
{{.Title}}
{{range .Articles}}{{if .URL}} - {{.Title}} ({{.URL}})
{{end}}{{end}}{{end}}{{end}}
```

### 出演者一覧

```
{{range .Casts}}{{.CharacterID}}（{{.Role}}・{{.AppearanceCount}}回目）
{{end}}
```
