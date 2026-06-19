# vox-radio アーキテクチャルール

vox-radio のプロダクトコード（Go）を疎結合・保守しやすい構成に保つためのルール。現状の良い構造を維持・強制し、理想形への方向付けを行う。`.go` ファイルを変更するときは本ドキュメントの依存ルールに違反しないこと。

## 1. 層構造の全体像

依存は必ず下方向のみ。逆方向・層飛ばしの import を追加しない（cli は合成点のため全層に依存してよい）。

```
cmd/vox-radio
    ↓
internal/cli（CLI層: 唯一の合成点。ロード・検証・依存注入）
    ↓
internal/pipeline（オーケストレーション層: interface 経由でステップを実行）
    ↓
ドメイン層（gather / rundown / script / synth / mix / manifest / slack / feed / cache / cast / eval / render）
    ↓
internal/config（設定層: 共有設定のロード・検証・Effective値）
    ↓
internal/model（データ層: 型定義と純粋関数のみ）
    ↓
基盤層（fileio / httpretry / logging / mediainfo / testutil）
```

## 2. 層の定義と依存ルール

各層が import してよい `internal/` パッケージを以下に限定する。

| 層 | パッケージ | import してよい internal パッケージ |
|---|---|---|
| 基盤 | `fileio` `httpretry` `logging` `mediainfo` `testutil` | なし（標準ライブラリのみ） |
| データ | `model` | **なし**（型定義と純粋関数のみ。ファイルI/O・YAML/JSONロード・環境変数参照を置かない） |
| 設定 | `config` | `fileio` のみ |
| ドメイン | `gather` `rundown(+flow/select/prompt)` `script(+write/direct/llm/summarize/summary)` `synth` `mix` `manifest` `slack` `feed` `cache` `cast` `eval` `render` | `model` `config` と基盤層。ドメイン間の横断依存は下記の許容リストのみ |
| オーケストレーション | `pipeline` | `model` `config` `fileio` `manifest` のみ。**ステップ実装パッケージ（gather/rundown/script/synth/mix 等）の import 禁止**（interface 経由で扱う） |
| CLI | `cli` | すべて可（唯一の合成点） |

### ドメイン間の許容横断エッジ

以下は現状の正当な横断依存として許容する。これ以外のドメイン間依存を追加する場合は、本表の更新と理由の記録（必要なら ADR）を行うこと。

- `rundown` → `rundown/flow` `rundown/select` `script/summarize`（要約器の再利用）
- `rundown` → `cache`（過去エピソード文脈の参照）
- `rundown/flow` `rundown/select` → `rundown/prompt` `script/llm`（共有LLMクライアント）
- `script` → `script/direct` `script/write`
- `script/write` → `cache`（過去エピソード文脈の参照）
- `script/write` → `script/llm`（LLMクライアントの利用）
- `script/direct` `script/summarize` `script/summary` → `script/llm`
- `feed` → `cache`（エピソード履歴がフィードの正データ）
- `eval` → `script/llm`（LLM-as-judge）
- `slack` → `render`（メッセージレンダリング）

### 検証

依存ルールは **depguard**（golangci-lint 内蔵）によって CI で自動検証される。`make lint` を実行することで違反を検出できる。

```bash
make lint
```

本表（§2）が depguard 設定（`.golangci.yml` の `linters.settings.depguard.rules`）の正（SSOT）である。表を更新したら depguard 設定も同期すること。

補助的な手動確認コマンド（全パッケージの依存エッジ一覧）:

```bash
go list -f '{{.ImportPath}} => {{join .Imports " "}}' ./internal/... | grep canpok1
```

## 3. ロードと依存注入のルール

- **ドメインパッケージの公開関数は、ロード済みの構造体を受け取る。** ファイルパスを受け取って内部で `Load*` しない（ユニットテストにファイル配置が必要になり、層の責務も崩れるため）。
- **`config.LoadConfig` / 各 spec の Load・Validate の呼び出しは `cli` 層のみ。** ロード → 検証 → 構造体注入の流れを `cli`（`util.go` の `loadConfigAndSpec` 等）に集約する。
- **`os.Getenv` の呼び出しは `cli` 層のみ。** 例外は ADR で明示された env override（`VOX_RADIO_VOICEVOX_URL`: ADR-0042、`VOX_RADIO_SLACK_API_URL`: ADR-0055）の `Effective*` メソッドに限る。
- **spec の置き場所は「複数ドメインが共有する設定は `config`、単一ドメイン専用の spec はそのドメインパッケージ」**とする。
  - 共有: `vox-radio.yaml` / `episode-spec.yaml` → `internal/config`
  - 専用: `feed-spec.yaml` → `internal/feed`、`slack-spec.yaml` → `internal/slack`（型・Load・Validate・Effective値をドメインに置く）

## 4. interface 設計のルール

- **interface は利用側で定義する**（現状踏襲。例: `pipeline.Gatherer` / `pipeline.Scripter`、`synth.VoicevoxClient`、`slack.Poster`、`script/llm.Client`）。
- **interface の引数・戻り値に具象ドメインパッケージの型を使わない。** `model` の型・基本型・利用側パッケージ定義の型のみとする（具象型を返すと利用側が実装パッケージへ依存してしまう）。
- **ステップ間のデータ受け渡しは戻り値で行う。**「実装が副作用でファイルを書き、呼び出し側が読み戻す」暗黙のファイル契約を作らない。中間ファイルの書き出しはオーケストレーター（`pipeline`）または `cli` の責務とする。
- 実装ごとに追加操作が必要な場合は supplementary interface（例: `CornerAppearanceSetter`）を使い、型アサーションで任意適用する（`.claude/rules/go-file.md` 参照）。

## 5. 中間成果物とファイルレイアウト

各パイプラインステップは独立コマンドとして再実行可能であること（ADR-0004 / ADR-0028）。ステップ間はファイルベースで疎結合にする。

| ファイル | 書き手 | 内容 |
|---|---|---|
| `intermediate/{program.id}_ep{NNN}/01_articles.json` | gather | 収集記事（`model.Articles`） |
| `intermediate/{program.id}_ep{NNN}/02_rundown.json` | rundown | 選別・フロー設計（`model.Rundown`） |
| `intermediate/{program.id}_ep{NNN}/03_lines.json` | script(write) | 元表記のセリフ（`model.ScriptLines`） |
| `intermediate/{program.id}_ep{NNN}/04_script.json` | script(direct) | 演出済み台本（`model.Script`） |
| `intermediate/{program.id}_ep{NNN}/05_clips/` + `clips.json` | synth | 音声クリップ（`model.ClipsMeta`） |
| `output/{program.id}_ep{NNN}.mp3` | mix | 完成音声 |
| `output/{program.id}_ep{NNN}_manifest.json` | manifest | エピソードマニフェスト（`model.Manifest`） |

- **パス・ファイル名は `internal/fileio/paths.go` の定数・関数のみ使用する。**`"03_lines.json"` 等のリテラル直書きを禁止する（`fileio.FileLines` 等の定数を使う）。回数別の出力パス（mp3・マニフェスト・中間ファイル）は `fileio.EpisodeLayout` のメソッドで解決する。

## 6. エラー処理・ログ

- エラーは `fmt.Errorf("文脈プレフィックス: %w", err)` で一貫してラップする（例: `fmt.Errorf("gather: %w", err)`）。
- 判定は `errors.Is` / `errors.As`、複数バリデーションエラーの集約は `errors.Join` を使う。
- ログは `slog` を使い、logger はオプション注入（`WithLogger` パターン）で渡す。
- グローバル可変状態を作らない。パッケージレベル `var` は読み取り専用の定数的データ・`embed.FS`・JSONスキーマに限る。

## 7. サイズ・シグネチャの目安

- **引数が6個以上になる関数は params struct 化する。**
- 1ファイル500行・1関数80行を超えたら責務分割を検討する（ffmpeg フィルタ構築のようなドメイン固有の複雑性は許容しつつ、フェーズ単位のヘルパー抽出を優先する）。

## 8. 既知の違反（リファクタリング対象）

以下は本ルール策定時点（2026-06）に現存する違反。対応タスクは Todoist（dev/vox-radio）に登録済み。**各タスクの完了時に該当項目をこの節から削除すること。**

| 違反 | 該当ルール | 対応タスク |
|---|---|---|
