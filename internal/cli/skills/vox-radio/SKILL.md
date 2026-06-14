---
name: vox-radio
description: vox-radio でラジオ番組を作る・更新する／設定ファイル（vox-radio.yaml・episode-spec.yaml・アセット設定・feed-spec.yaml・slack-spec.yaml）を編集する／設定や使い方を相談するスキル。質問に答えるだけで番組（mp3）まで仕上げられる。フィールド定義は同梱の references/*.md を参照し、check コマンドで検証する。
allowed-tools: Bash, Read, Grep, Glob, Edit, Write
---

## 概要

vox-radio の番組制作・設定編集・相談に対応するスキルです。

- 番組を一から作りたい・次回エピソードを作りたい → 「番組を作る・更新する」のフローに従う
- 設定の一部だけ直したい・使い方を相談したい・設定エラーを解消したい → 「リファレンスと検証コマンド」を起点に適宜対応する

迷ったら、まず下の「リファレンスと検証コマンド」を確認してください。

## リファレンスと検証コマンド

各設定ファイルのフィールド定義は同梱の `references/` ディレクトリを参照してください。設定に不備があるかは該当の検証コマンドで確認します（**終了コード 0 なら正常**）。

| ファイル | リファレンス | 検証コマンド |
|---|---|---|
| `vox-radio.yaml` | `references/vox-radio.md` | `vox-radio config check --config vox-radio.yaml` |
| `episode-spec.yaml` | `references/episode-spec.md` | `vox-radio episodegen check episode-spec.yaml --config vox-radio.yaml` |
| アセット設定 YAML | `references/assets.md` | `vox-radio assets check assets/assets.yaml` |
| `feed-spec.yaml` | `references/feed-spec.md` | `vox-radio feedgen check` |
| `slack-spec.yaml` | `references/slack-spec.md` | `vox-radio slackpost check` |

- 設定ファイルを変更したら、必ず該当の検証コマンドで確認すること。
- フィールド値を SKILL.md 本体に直接記載せず、常に `references/*.md` を参照すること。

> リファレンスの記載と check コマンドの挙動が食い違う場合は、check コマンドの検証結果を正としてください。
> 食い違いはスキルとバイナリの版ずれが原因のことがあります。下記「バージョン整合チェック」で版を揃えると解消します。

## バージョン整合チェック（作業の最初に実施）

スキルファイルはインストール時のバイナリと同一バージョンで配布されます。バイナリだけを更新すると古いスキルが残り、リファレンスと実際の挙動が食い違うことがあります。作業を始める前に版ずれを確認してください。

1. スキルディレクトリの `.skill-version`（インストール時に記録されたバイナリ版）を読む
2. `vox-radio --version` で現在のバイナリ版を取得する（出力形式: `vox-radio version X.Y.Z`）
3. 両者を比較し、以下に従う
   - **いずれかが `dev`、または `.skill-version` が存在しない**: 比較不能（ローカルビルド等）。警告を出すだけで、そのまま作業を続行する
   - **一致**: そのまま作業を続行する
   - **バイナリが新しい**: 確認のうえ `vox-radio install --skills --force` を実行してスキル（と `.skill-version`）を再生成し、最新の references で作業を続行する
   - **スキルが新しい（バイナリが古い）**: 確認のうえ、バイナリの更新手順（`scripts/install.sh` の再実行、または `go install`）を案内する。バイナリの自動更新は行わない（環境依存のため）

## 番組を作る・更新する

「質問に答えるだけで番組が仕上がる」標準フローです。ユーザーへの質問は、特定のツールに依存せず**対話形式で1問ずつ**行い、回答を得てから次へ進んでください（一度に多数の質問を並べない）。

### 1. 前提チェック

番組生成（`episodegen`）で失敗しないよう、着手前に次を確認します。不足していても「設定ファイルだけ作る」場合は続行できます（生成段で必要になる旨をユーザーに伝える）。

- 生成AIの API キー: `vox-radio.yaml` の LLM 設定で指定した環境変数（例: `api_key_env`）が設定されているか
- VOICEVOX Engine が起動しているか（既定 `http://localhost:50021`）
- `ffmpeg` / `ffprobe` が PATH にあるか

### 2. 設定があるか確認

- **設定ファイルが無い（新規）**: `vox-radio init` でテンプレートを生成してから編集する（`init` は既存ファイルを上書きしないため再実行しても安全）
- **設定ファイルがある（更新）**: 既存設定を読み込み、何を変えたいか（次回エピソード／コーナーやキャラの変更など）を確認する。`program.id` は引き継ぐこと（キャッシュ・放送回数の連続性を壊さないため）

### 3. ヒアリング

ユーザーに対話で質問し、番組の内容を引き出します。新規時は次の順で、更新時は「何を変えるか」を起点に必要な項目だけ尋ねます。

| 質問する内容 | 反映先（references で定義を確認） |
|---|---|
| 番組のテーマ・ジャンル | `episode-spec.yaml` の `program.description` |
| 番組タイトル | `program.title` / `program.id` |
| 出演キャラと役割 | `vox-radio.yaml` の `characters`、`episode-spec.yaml` の `casts` |
| コーナー構成（話題・流れ） | `corners[]`（title / content / cast / length_sec） |
| 各コーナーのネタ元（RSS / 記事 URL の有無） | `corners[].source` |
| 番組のおよその長さ | 各 `corners[].length_sec` |
| BGM・ジングル・効果音を入れるか | `assets/assets.yaml` ＋ `corners[]`（下記「5. アセットを設定する」で扱う） |

- 出演キャラには VOICEVOX の話者を割り当てる（`characters` の `styles` に話者ID、`default_style` に既定スタイル名）。
- BGM・ジングル・効果音を入れる場合は、下記「5. アセット（BGM・ジングル・効果音）を設定する」で設定する（音声ファイルはユーザー提供が前提）。
- コーナーの出現条件・ゲストの登場周期などの凝った設定は、初回は最小構成にとどめ「あとから追加できる」と案内する。必要になったら `references/*.md` を参照して足す。

### 4. 設定に反映・検証

ヒアリング結果を `references/*.md` の定義に沿って各設定ファイルへ反映し、該当の check コマンドが**終了コード 0 になるまで**修正と検証を繰り返します。

- キャラID（`vox-radio.yaml` の `characters` キー）は `episode-spec.yaml` の `corners[].cast` / `casts` と一致させること
- `episode-spec.yaml` の `assets_files` に列挙したパスが実際に存在することを確認すること
- `program.id` は必須（未設定だと `episodegen` / `episodegen check` でエラーになる）

### 5. アセット（BGM・ジングル・効果音）を設定する（任意）

ヒアリングでアセットを入れると決めた場合のみ実施します。**音声ファイルはエージェントが生成できない**ため、ユーザーに用意してもらう前提です。

1. 使う音声ファイルを `assets/` に置いてもらう（パスを確認する）。
2. `assets/assets.yaml` に各素材を登録する。`file` と `description`（何の音か・いつ使うか）は必ず設定し、音量・フェード・BGM のダッキング・無音除去は `references/assets.md` を参照して設定する。
3. 鳴らし方を決めて紐付ける。
   - 確定的に鳴らす（開始/終了ジングル・境界 SE・コーナー BGM）→ `episode-spec.yaml` の `corners[].start_audio` / `end_audio` / `bgm` に設定する。
   - 会話中に随時鳴らす SE → `description` と `program.direction` / `corners[].direction`（演出方針）を整え、台本生成（direct ステップ）の LLM 挿入に委ねる。
   - `episode-spec.yaml` の `assets_files` に `assets.yaml` を登録する。
4. 検証・試聴する。
   - `vox-radio assets check assets/assets.yaml` で検証する（終了コード 0 になるまで修正を繰り返す）。
   - `vox-radio assets preview assets/assets.yaml --id {type}:{key} --out preview.mp3` で素材ごとに試聴し、音量・フェード・ダッキング等を調整する。

> アセットID キー（`assets.yaml` の `jingle` / `se` / `bgm` のキー）は、`corners[]` の `start_audio.id` / `end_audio.id` / `bgm` と一致させること。

### 6. 番組を生成する

設定が検証を通過したら、番組を生成するかユーザーに確認します。

- **生成する**（前提チェックが整っている場合）: 次を実行し `output/episode.mp3` を生成する

  ```bash
  vox-radio episodegen --spec episode-spec.yaml
  ```

- **生成しない／前提が未整備**: 設定ファイルが完成した時点で完了とする（あとからユーザーが生成できる旨を伝える）

### 7. 完了報告

生成物のパス（`output/episode.mp3`、中間ファイルは `output/intermediate/`）を伝えます。合成音声を公開する場合は VOICEVOX のクレジット表記（例: `VOICEVOX:ずんだもん`）が必要な点も案内してください。

## それ以外（部分編集・相談・設定エラーの解消など）

上のフローに当てはまらない依頼（設定の一部だけ変更したい、使い方を相談したい、`check` のエラーを直したいなど）は、「リファレンスと検証コマンド」をもとに適宜対応してください。

- フィールドの意味や設定方法は `references/*.md` を参照する
- 設定エラーは該当の check コマンドでエラー内容を確認し、references を見て修正し、終了コード 0 になるまで検証を繰り返す
- 設定を変更したら必ず該当の check コマンドで検証する
