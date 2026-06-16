# episode-spec.yaml（エピソード仕様）リファレンス

> **検証の正**: 設定が正しいかは下記の検証コマンドの結果で判断してください。本ドキュメントと実際の挙動が食い違う場合は、スキルとバイナリの版ずれが原因のことがあります。SKILL.md の「バージョン整合チェック」に従ってスキル / バイナリを揃えてください。

> **検証コマンド**: `vox-radio episodegen check <パス> --config vox-radio.yaml`

`--spec` フラグで指定するジャンル別設定ファイルです。`vox-radio init` で生成されるテンプレートは `episode-spec.yaml` という名前です。記入済みのサンプル一式は `vox-radio init --sample` で `sample/` に生成できます。

## `program` セクション

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `id` | string | 必須 | 番組を識別するID。キャッシュファイル名（`.vox-radio/cache/{id}.jsonl`）とキャッシュエントリの記録キーに使用。日替わりコーナーやゲストの登場回もこのIDをキーに数える |
| `title` | string | 任意 | 番組タイトル |
| `author` | string | 任意 | 番組の作者名。生成 MP3 のアーティストタグ（ID3 TPE1）に埋め込まれる。空の場合はタグが省略される |
| `description` | string | 任意 | 番組の説明（LLM への指示に使用）。RSSフィード・Slack通知にも露出する公開フィールド |
| `direction` | string | 任意 | 番組全体の演出方針（direct ステップのみに渡る）。SE・pause の挿入タイミングに関する指示。台本生成・manifest・feed・Slack には渡されない |
| `script_note` | string | 任意 | 番組全体の台本指示（write ステップのみに渡る）。非公開フィールド。manifest・feed・Slack には露出しない。コーナーを問わず全台本に適用したいルールや注意事項を記述する |
| `summary_length` | int | 任意 | 番組全体サマリーの目安文字数。未指定時はデフォルト 200 文字 |
| `chars_per_minute` | int | 任意 | 台本の文字数換算に使用する1分あたりの目安文字数。台本生成時の `length_sec` → 目標文字数換算に使用。未指定時はデフォルト 420（= 7文字/秒×60） |
| `audio_quality` | string | 任意 | 生成 MP3 の音質プリセット。`high`（約245kbps）/ `standard`（約190kbps、**既定**）/ `low`（約130kbps）。未指定または空は `standard` として扱われる |

## `corners` セクション

`corners` はコーナー定義のリストです。番組を構成するセグメントを順番に記述します。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `id` | string | 必須 | コーナーを回をまたいで同定する安定キー（番組内で一意）。過去回の扱い回数（新コーナー/久しぶりの復活コーナー）の集計・演出に使う。`title` を変更しても `id` で履歴を追える。未設定・重複はバリデーションエラー |
| `title` | string | 必須 | コーナータイトル（番組内で一意） |
| `content` | string | 任意 | コーナーの内容説明（台本生成 LLM への指示に使用） |
| `direction` | string | 任意 | コーナーの演出方針（direct ステップのみに渡る）。SE の挿入タイミングなど。台本生成 LLM へは渡されない |
| `script_note` | string | 任意 | コーナー個別の台本指示（write ステップのみに渡る）。非公開フィールド。manifest・feed・Slack には露出しない。このコーナーのやり取りの細かい指示を記述する |
| `cast` | map[string]string | 任意 | キャラID → 役割説明のマップ（キーは `vox-radio.yaml` の `characters` のキーと一致させること） |
| `length_sec` | int | 任意 | このコーナーの目標収録時間（秒）。台本生成時に `program.chars_per_minute`（既定 420/分）で文字数へ換算される |
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
  - id: "opening"
    title: "オープニング"       # condition なし → 毎回必須
    content: "挨拶と自己紹介"
    cast: { zundamon: "MC" }
    length_sec: 30

  - id: "monthly-special"
    title: "月いちスペシャル"
    content: "5回に1回だけやるスペシャル企画"
    cast: { zundamon: "MC", metan: "MC" }
    length_sec: 120
    condition:
      every: 5                  # 5の倍数回（5,10,15,…）に採用

  - id: "normal-talk"
    title: "通常トーク"
    content: "月いちスペシャルを行わない回の通常コーナー"
    cast: { zundamon: "MC", metan: "MC" }
    length_sec: 120
    condition:
      not:
        every: 5               # 5の倍数でない回（1,2,3,4,6,…）に採用

  - id: "weekly-book"
    title: "今週の一冊"
    content: "おすすめの本を紹介"
    cast: { zundamon: "ボケ", metan: "解説" }
    length_sec: 120
    condition:
      every: 2                  # 偶数回に採用
      not:
        episodes: [6]           # ただし第6回は除く（2,4,8,10,…）

  - id: "ending"
    title: "エンディング"        # condition なし → 毎回必須
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
- キャッシュ読み込みに失敗した場合はエラー停止します
- 採用されたコーナーは元の `corners` 配列の順序を維持したまま台本に出力されます

**N者ローテーションの例（`every` + `offset`）**

```yaml
corners:
  - id: "corner-a"
    title: "コーナーA"
    condition: { every: 3, offset: 1 }   # 1,4,7,… 回に採用
  - id: "corner-b"
    title: "コーナーB"
    condition: { every: 3, offset: 2 }   # 2,5,8,… 回に採用
  - id: "corner-c"
    title: "コーナーC"
    condition: { every: 3, offset: 0 }   # 3,6,9,… 回に採用
```

### `corners[].source` サブフィールド

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `feeds` | []FeedEntry | 任意 | RSS/Atom フィードのリスト |
| `articles` | []string | 任意 | 個別記事 URL のリスト。`https://` のほか `file://` によるローカルファイル指定も可能（後述） |

### `corners[].source.feeds[]` サブフィールド

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `url` | string | 必須 | フィードの URL。`https://` のほか `file://` によるローカルファイル指定も可能（後述） |
| `max_items` | int | 任意 | 過去使用URLを除外したうえで確保する最大記事数。デフォルト: 0（実質無制限）。除外で減った分はフィード内の後続記事で補う。 |

### `file://` プロトコルによるローカルフィード・記事の読み込み

アクセス制限などで RSS/Atom フィードに直接アクセスできない場合、手動ダウンロードした XML ファイルをローカルから読み込めます。`feeds[].url` および `articles[]` に `file://` プロトコルを使ったパスを指定します。

**絶対パス指定**（そのまま使用）:

```yaml
source:
  feeds:
    - url: "file:///home/user/downloads/feed.xml"
```

**相対パス指定**（spec ファイルのディレクトリを基準に解決）:

```yaml
# /home/user/myshow/episode-spec.yaml に記述した場合、
# /home/user/myshow/feeds/feed.xml を読み込む
source:
  feeds:
    - url: "file://feeds/feed.xml"
  articles:
    - "file://articles/article.html"
```

> **注意**: ファイルが存在しない場合は `episodegen collect` 実行時にエラーになります。

## `casts` セクション（出演者名簿）

`casts` は番組に登場する出演者をキャラID（`vox-radio.yaml` の `characters` のキー）をキーとするマップで宣言します。`corners[].cast` に書くキャラは、必ずここで宣言してください。`type` で毎回出演（`regular`）かゲスト（`guest`）かを指定します。ゲストは指定した条件に合致した回だけ出演し、その回はオープニングからエンディングまで、採用された全コーナーにゲストが通しで出演します（コーナーの `cast` への明示追記は不要）。

```yaml
casts:
  zundamon:
    type: regular                      # regular = 毎回出演 / guest = 決まった回だけ出演
    role: "MC。進行役。ボケ担当。"        # 番組全体での役割（プロンプトに渡す）
    # condition を省略すると毎回出演（regular のデフォルト）
  metan:
    type: regular
    role: "MC。相棒。ツッコミ担当。"
    # お休み条件の例（5回目だけ出演しない）:
    # condition:
    #   not:
    #     episodes: [5]
  zunko:                               # キー = characters に定義済みのキャラID
    type: guest                        # guest は condition が必須
    role: "古参リスナー出身の常連ゲスト"
    condition:
      episodes: [3, 10, 20]            # 第3・10・20回に出演（明示リスト）
```

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `type` | string | 必須 | `regular`（毎回または条件付きで出演）または `guest`（条件付きで出演。`condition` 必須） |
| `role` | string | 任意 | 番組全体での役割説明（プロンプトに渡す。ゲストは出演回の全コーナーの cast にマージされる） |
| `condition.episodes` | []int | 任意 | 出演する回番号の明示リスト（各値は 1 以上） |
| `condition.every` | int | 任意 | 周期的な出演（N の倍数回に出演。1 以上） |
| `condition.offset` | int | 任意 | `every` と組み合わせる剰余（`episodeNumber % every == offset` で出演。未指定=0 で倍数回） |
| `condition.not` | EpisodeCondition | 任意 | この条件に合致する回を除外（補集合） |

- `type` は `regular` または `guest` のいずれか。それ以外はバリデーションエラー
- `type: guest` は `condition` が必須（省略するとバリデーションエラー）
- `type: regular` は `condition` を省略すると毎回出演。`condition` を書くと条件に合致した回だけ出演
- `condition.episodes` と `condition.every` の両方を指定した場合は **論理和**（どちらかに合致すれば出演）
- `condition.not` に合致する回は除外される。`not` 単独指定（`episodes`/`every` を省略）すると「合致しない回すべて」を表現できる
- 肯定条件（`episodes`/`every`）を省略すると「常に真」として扱われ、`not` 単独で補集合を表現できる
- キャストのキャラ ID は `vox-radio.yaml` の `characters` に存在しなければバリデーションエラー
- キャッシュ読み込みに失敗した場合はエラー停止します

**3人のゲストを順番に出す例（`every` + `offset` で均等に分担）**

```yaml
casts:
  alice:
    type: guest
    role: ゲストA
    condition: { every: 3, offset: 1 }   # 3回おきに出演、初回は1回目。→ 第1,4,7,… 回に出演。
  bob:
    type: guest
    role: ゲストB
    condition: { every: 3, offset: 2 }   # 3回おきに出演、初回は2回目。→ 第2,5,8,… 回に出演。
  carol:
    type: guest
    role: ゲストC
    condition: { every: 3, offset: 0 }   # 3回おきに出演、初回は3回目。→ 第3,6,9,… 回に出演。
```

## `corner_defaults` セクション（コーナー共通デフォルト）

`corner_defaults` は全コーナーに共通で適用するデフォルト値を設定します。省略可能です。各コーナーで値を書けば上書きでき、空値（`""` や `{}`）を書けば無効化できます。

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `bgm` | string | 任意 | 全コーナーのデフォルト BGM キー名（`assets.bgm` のキーと一致させること） |
| `start_audio` | AudioRef | 任意 | 全コーナーのデフォルト開始境界音声（`type`/`id` は `corners[].start_audio` と同様） |
| `end_audio` | AudioRef | 任意 | 全コーナーのデフォルト終了境界音声（`type`/`id` は `corners[].end_audio` と同様） |
| `start_pause_sec` | float64 | 任意 | 全コーナーのデフォルト開始前の無音時間（秒） |
| `end_pause_sec` | float64 | 任意 | 全コーナーのデフォルト終了後の無音時間（秒） |

### 継承・上書き・無効化の挙動

| コーナーの設定 | 動作 |
|---|---|
| フィールドを書かない（省略） | `corner_defaults` の値を継承 |
| 具体的な値を書く | `corner_defaults` の値を上書き |
| `bgm: ""` または `start_audio: {}` / `end_audio: {}` と書く | 無効化（デフォルトを継承せず音声なし） |

```yaml
corner_defaults:
  bgm: talk_bgm           # 全コーナーのデフォルトBGM
  start_pause_sec: 1.0    # 全コーナーのデフォルト開始前の無音

corners:
  - id: "opening"
    title: "オープニング"
    # bgm・start_pause_sec を省略 → corner_defaults を継承

  - id: "main"
    title: "メイン"
    bgm: special_bgm      # corner_defaults.bgm を上書き

  - id: "ending"
    title: "エンディング"
    bgm: ""               # corner_defaults.bgm を無効化（BGMなし）
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
