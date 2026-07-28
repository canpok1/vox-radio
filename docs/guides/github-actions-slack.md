# GitHub Actions で定期的に Slack へ投稿する

GitHub Actions のスケジュール実行で、定期的に番組を生成して Slack へ投稿する最小構成のサンプルです。

> **このページの位置づけ**
> GitHub（リポジトリ・Secrets/Variables）と Slack（Bot・チャンネル）側の準備は vox-radio の責務の**範囲外**で、下記「前提」が整っていることを前提とします。

## 前提

- 設定ファイル（`vox-radio.yaml` / `episode-spec.yaml` / `slack-spec.yaml`）がリポジトリのルートにコミット済みであること（`vox-radio init` で生成し、内容を記入したもの）。
- Slack Bot を作成し、必要な権限（`chat:write` / `files:write` / `files:read`）を付与済みであること（[README の Slack投稿](../../README.md#slack投稿)を参照）。
- リポジトリに次の値を登録済みであること。

  | 値 | 登録先 | 既定の環境変数名 | 対応する設定 |
  |---|---|---|---|
  | Gemini API キー | Secret | `GEMINI_API_KEY` | `vox-radio.yaml` の LLM 設定 |
  | Slack Bot トークン | Secret | `SLACK_BOT_TOKEN` | `vox-radio.yaml` の `slack.bot_token_env` |
  | 投稿先チャンネル ID | Variable | `SLACK_CHANNEL_ID` | `slack-spec.yaml` の `slack.channel_env` |

  環境変数名は設定ファイル側で変更できます。変えた場合はワークフローの `env:` の名前も合わせてください。
- v1.0.5 以前からアップグレードした場合、キャッシュは旧パス（`.vox-radio/cache`）に保存されています。サンプルワークフローの旧パス併記と移行ステップは、この履歴を新パスへ引き継ぐためのものです（新規導入時は不要ですが、入れておいても無害です）。

## サンプルワークフロー

`.github/workflows/vox-radio-slack.yml` として配置します。

```yaml
name: vox-radio-slack

on:
  schedule:
    # UTC 表記。例は毎日 12:15 JST 相当（00分は混雑するため避ける）。
    - cron: "15 3 * * *"
  workflow_dispatch:

concurrency:               # 同時に複数実行されないようにする（キャッシュの取り合いを防ぐ）
  group: vox-radio-slack
  cancel-in-progress: false

jobs:
  generate-and-post:
    runs-on: ubuntu-latest
    services:
      # 音声合成に VOICEVOX エンジンが必要。ランナーの localhost にマップされる。
      voicevox:
        image: voicevox/voicevox_engine:cpu-latest
        ports:
          - 50021:50021
    steps:
      - uses: actions/checkout@v4

      # 1. キャッシュ復元。実行ごとに新しいキーで保存し、直近のキャッシュを引き継ぐ書き方。
      # 旧パス（.vox-radio/cache）は移行期の併記（理由は次のステップのコメントを参照）。
      - uses: actions/cache@v4
        with:
          path: |
            .vox-radio/cache
            .vox-radio/programs
          key: vox-radio-cache-${{ github.run_id }}
          restore-keys: vox-radio-cache-

      # 2. 旧レイアウト（v1.0.5 以前）のキャッシュを新レイアウトへ移す。自動移行は行われないため
      # このステップが無いと、旧パスを復元しても回番号が 1 に戻る。
      # 何度実行しても安全（移行済みなら何もしない）。
      - name: Migrate cache layout
        run: |
          for f in .vox-radio/cache/*.jsonl; do
            [ -e "$f" ] || continue
            id=$(basename "$f" .jsonl)
            mkdir -p ".vox-radio/programs/$id"
            mv "$f" ".vox-radio/programs/$id/cache.jsonl"
          done

      - name: Install ffmpeg
        run: sudo apt-get update && sudo apt-get install -y ffmpeg

      - name: Install vox-radio
        run: curl -fsSL https://github.com/canpok1/vox-radio/releases/latest/download/install.sh | bash

      # 3. 番組生成
      - name: Generate episode
        env:
          GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
          VOX_RADIO_VOICEVOX_URL: http://127.0.0.1:50021
        run: vox-radio episodegen --spec episode-spec.yaml --out-dir output

      # 4. Slack 投稿
      - name: Post to Slack
        env:
          SLACK_BOT_TOKEN: ${{ secrets.SLACK_BOT_TOKEN }}
          SLACK_CHANNEL_ID: ${{ vars.SLACK_CHANNEL_ID }}
        run: |
          MANIFEST=$(ls output/*_manifest.json | head -n1)
          vox-radio slackpost --manifest "$MANIFEST" --spec slack-spec.yaml
```

## 仕組み

1. **キャッシュ復元** — 過去回の履歴（`.vox-radio/programs/`）を `actions/cache` で復元し、前回までの内容を踏まえた番組を生成します。初回（キャッシュ無し）でも動作し、回番号は 1 から始まります。
2. **キャッシュレイアウトの移行** — 旧パス（`.vox-radio/cache`）に復元されたファイルがあれば `.vox-radio/programs/` へ移します。
3. **番組生成** — `episodegen` が記事収集から音声合成までを一括実行し（詳細は[README の番組生成](../../README.md#番組生成)）、`output/` に mp3 とマニフェストを出力します。
4. **Slack 投稿** — `slackpost` がマニフェストをもとに mp3 を Slack へ直接アップロードします。GitHub Release など公開 URL の準備は不要です。
5. **キャッシュ保存** — 新しい履歴を含む `.vox-radio/programs/` は、復元に使った `actions/cache` の post 処理で自動保存されます。専用ステップは要りません。
6. **VOICEVOX NEMO などを併用する場合** — `services` にサービスを追加してポートをマップし、`vox-radio.yaml` の `voicevox.engines` でエンジンを定義したうえで、エンジンごとに `VOX_RADIO_VOICEVOX_URL_<エンジン名を大文字化し - を _ に置換した名前>` 環境変数を設定します（詳細は[vox-radio.md](../../internal/cli/skills/vox-radio/references/vox-radio.md)）。

## 注意事項

- **キャッシュの保持期間** — `actions/cache` は 7 日間アクセスのないキャッシュを自動削除します。実行間隔が空くと過去回の文脈（番組の連続性）が失われます。日次など定期実行のサンプル用途では問題になりません。連続性を確実に保ちたい場合は、専用ブランチへコミットして永続化する方式もあります（その分ワークフローは長くなります）。
- **クレジット表記・利用規約** — 合成音声を公開する際の VOICEVOX のクレジット表記など、規約まわりの責任は利用者にあります。詳細は[DISCLAIMER.md](../../DISCLAIMER.md)を参照してください。
- **移行完了後の片付け** — 旧パスの併記と「Migrate cache layout」ステップは、すべての履歴が新パスへ移った後は不要です（残しても無害ですが）。確認できたらワークフローから両方をまとめて外してください。
