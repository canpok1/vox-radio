# episode-spec.yaml（エピソード仕様）リファレンス

> **フィールド定義の正**: `internal/config/config.go`。本ドキュメントとコードが齟齬する場合はコードを優先してください。

> **検証コマンド**: `vox-radio episodegen check <パス> --config vox-radio.yaml`

`--spec` フラグで指定するジャンル別設定ファイルです。`vox-radio init` で生成されるテンプレートは `episode-spec.yaml` という名前です。詳細は [examples/README.md](examples/README.md) も参照してください。

## `program` セクション

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `title` | string | 任意 | 番組タイトル |
| `description` | string | 任意 | 番組の説明（LLM への指示に使用） |
| `summary_length` | int | 任意 | 番組全体サマリーの目安文字数。未指定時はデフォルト 200 文字 |

## `corners` セクション

`corners` はコーナー定義のリストです。番組を構成するセグメントを順番に記述します。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `title` | string | 必須 | コーナータイトル |
| `content` | string | 任意 | コーナーの内容説明（台本生成 LLM への指示に使用） |
| `direction` | string | 任意 | コーナーの演出説明（演出生成 LLM への指示に使用。SE の挿入タイミングなど）。台本生成 LLM へは渡されない |
| `cast` | map[string]string | 任意 | キャラID → 役割説明のマップ（キーは `vox-radio.yaml` の `characters` のキーと一致させること） |
| `length_sec` | int | 任意 | このコーナーの目標収録時間（秒）。台本生成時に文字数（≈7文字/秒）へ換算される |
| `summary_length` | int | 任意 | コーナーサマリーの目安文字数。未指定時はデフォルト 100 文字 |
| `source` | SourceConfig | 任意 | データソース（省略するとこのコーナーの収集はスキップ） |
| `start_audio` | AudioRef | 任意 | コーナー開始境界音声。`type` に `jingle`（BGM停止後再生）または `se`（BGMの下で再生）を指定し、`id` に `assets` の該当マップのキーを指定する。コーナー本編の前に確定的に挿入される |
| `end_audio` | AudioRef | 任意 | コーナー終了境界音声。`type`/`id` は `start_audio` と同様。コーナー本編の後に確定的に挿入される |
| `bgm` | string | 任意 | コーナー中 BGM のキー名（`assets.bgm` のキーと一致させること）。コーナー本編を開始/停止セグメントで挟む |
| `start_pause_sec` | float64 | 任意 | コーナー先頭（`start_audio` より前）に挿入する無音時間（秒）。0 または省略時は挿入しない |
| `end_pause_sec` | float64 | 任意 | コーナー末尾（`end_audio` より後）に挿入する無音時間（秒）。0 または省略時は挿入しない |
| `condition` | EpisodeCondition | 任意 | コーナーの出現条件（省略すると毎回必ず出る固定コーナー） |

### `corners[].condition` サブフィールド

`condition` を設定すると、回番号が条件に合致したときのみそのコーナーが採用されます。`condition` を省略したコーナーは毎回必ず採用される固定コーナーとなります。

```yaml
corners:
  - title: "オープニング"       # condition なし → 毎回必須
    content: "挨拶と自己紹介"
    cast: { zundamon: "MC" }
    length_sec: 30

  - title: "月いちスペシャル"
    content: "5回に1回だけやるスペシャル企画"
    cast: { zundamon: "MC", metan: "MC" }
    length_sec: 120
    condition:
      every: 5                  # 5の倍数回（5,10,15,…）に採用

  - title: "通常トーク"
    content: "月いちスペシャルを行わない回の通常コーナー"
    cast: { zundamon: "MC", metan: "MC" }
    length_sec: 120
    condition:
      not:
        every: 5               # 5の倍数でない回（1,2,3,4,6,…）に採用

  - title: "今週の一冊"
    content: "おすすめの本を紹介"
    cast: { zundamon: "ボケ", metan: "解説" }
    length_sec: 120
    condition:
      every: 2                  # 偶数回に採用
      not:
        episodes: [6]           # ただし第6回は除く（2,4,8,10,…）

  - title: "エンディング"        # condition なし → 毎回必須
    content: "締めの挨拶"
    cast: { zundamon: "MC" }
    length_sec: 30
```

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `condition.episodes` | []int | 任意 | 採用する回番号の明示リスト（各値は 1 以上） |
| `condition.every` | int | 任意 | 周期的な採用（N の倍数回に採用。1 以上） |
| `condition.offset` | int | 任意 | `every` と組み合わせる剰余（`episodeNumber % every == offset` で採用。未指定=0 で倍数回） |
| `condition.not` | EpisodeCondition | 任意 | この条件に合致する回を除外（補集合） |

- `condition.episodes` と `condition.every` の両方を指定した場合は **論理和**（どちらかに合致すれば採用）
- `condition.not` に合致する回は除外される。`not` 単独指定（`episodes`/`every` を省略）すると「合致しない回すべて」を表現できる
- 肯定条件（`episodes`/`every`）を省略すると「常に真」として扱われ、`not` 単独で補集合を表現できる
- `condition.episodes`・`condition.every`・`condition.not` のいずれも未設定の場合、および `not` の中身が空の場合はバリデーションエラー
- キャッシュが無効または `program.id` が未設定で回番号が不明な場合、条件付きコーナーを含む全コーナーが採用されます（警告ログが出力されます）
- 採用されたコーナーは元の `corners` 配列の順序を維持したまま台本に出力されます

**N者ローテーションの例（`every` + `offset`）**

```yaml
corners:
  - title: "コーナーA"
    condition: { every: 3, offset: 1 }   # 1,4,7,… 回に採用
  - title: "コーナーB"
    condition: { every: 3, offset: 2 }   # 2,5,8,… 回に採用
  - title: "コーナーC"
    condition: { every: 3, offset: 0 }   # 3,6,9,… 回に採用
```

### `corners[].source` サブフィールド

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `feeds` | []FeedEntry | 任意 | RSS/Atom フィードのリスト |
| `articles` | []string | 任意 | 個別記事 URL のリスト |

### `corners[].source.feeds[]` サブフィールド

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `url` | string | 必須 | フィードの URL |
| `max_items` | int | 任意 | 過去使用URLを除外したうえで確保する最大記事数。デフォルト: 0（実質無制限）。除外で減った分はフィード内の後続記事で補う。 |

## `guests` セクション（省略可）

`guests` はゲスト出演者の設定をキャラID（`vox-radio.yaml` の `characters` のキー）をキーとするマップで定義します。指定した条件に合致した回だけゲストが出演し、その回はオープニングからエンディングまで全コーナーにゲストが通しで出演します。

```yaml
guests:
  zunko:                             # キー = characters に定義済みのキャラID
    role: 古参リスナー出身の常連ゲスト  # 全コーナーの cast にマージされる役割説明
    condition:
      episodes: [3, 10, 20]          # 第3回・第10回・第20回に登場（明示リスト）
  metan:
    role: 業界に詳しい解説ゲスト
    condition:
      every: 5                       # 5, 10, 15, … の回に登場（定期出演）
  sora:
    role: フリーランスエンジニア
    condition:
      not:
        every: 5                     # 5の倍数以外の回に登場（metan がいない回すべて）
```

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `role` | string | 任意 | 全コーナーの cast にマージされるゲストの役割説明 |
| `condition.episodes` | []int | 任意 | 出演する回番号の明示リスト（各値は 1 以上） |
| `condition.every` | int | 任意 | 周期的な出演（N の倍数回に出演。1 以上） |
| `condition.offset` | int | 任意 | `every` と組み合わせる剰余（`episodeNumber % every == offset` で出演。未指定=0 で倍数回） |
| `condition.not` | EpisodeCondition | 任意 | この条件に合致する回を除外（補集合） |

- `condition.episodes` と `condition.every` の両方を指定した場合は **論理和**（どちらかに合致すれば出演）
- `condition.not` に合致する回は除外される。`not` 単独指定（`episodes`/`every` を省略）すると「合致しない回すべて」を表現できる
- 肯定条件（`episodes`/`every`）を省略すると「常に真」として扱われ、`not` 単独で補集合を表現できる
- `condition.episodes`・`condition.every`・`condition.not` のいずれも未設定の場合はバリデーションエラー
- キャッシュが無効または `program.id` が未設定で回番号が不明な場合、ゲストは出演しません（警告ログが出力されます）

**N者ローテーションの例（`every` + `offset`）**

```yaml
guests:
  alice:
    role: ゲストA
    condition: { every: 3, offset: 1 }   # 1,4,7,… 回に出演
  bob:
    role: ゲストB
    condition: { every: 3, offset: 2 }   # 2,5,8,… 回に出演
  carol:
    role: ゲストC
    condition: { every: 3, offset: 0 }   # 3,6,9,… 回に出演
```

## `assets_files` フィールド

`assets_files` はアセット設定ファイル（ジングル・SE・BGM を定義した YAML）のパスリストです。バイナリ素材は別途用意してください。

- `assets_files` の各パスは**プロファイルファイルのディレクトリ**を基準に解決されます
- アセット設定ファイル内の `file:` 相対パスは**そのアセット設定ファイルのディレクトリ**を基準に解決されます（アセット設定ファイルと音声素材をひとまとめに配布できます）
- 複数ファイルを指定した場合は後勝ちでマージされます（共通アセット集＋番組固有アセットの組み合わせが可能）
- `assets_files` を省略した場合はアセットが空となります（アセット不要なプロファイルで許容）

```yaml
# プロファイルファイルからの参照
assets_files:
  - assets/assets.yaml          # 共通アセット集
  - assets/my-assets.yaml       # 番組固有アセット（後勝ちでマージ）
```

アセット設定ファイルのフォーマットは `references/assets.md` を参照してください。
