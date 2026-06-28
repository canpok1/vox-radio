# Docker で番組を生成する

公式 Docker イメージ（`ghcr.io/canpok1/vox-radio`）を使うと、Go・ffmpeg を個別に用意せず、設定ファイルとコンテナだけで番組を生成できます。音声合成に使う VOICEVOX エンジンはイメージに含めず、公式イメージを併用します。

> **このページの位置づけ**
> ローカルで手早く動かしたい・ホスト環境を汚したくない場合の手段です。通常のインストールは [README のインストール](../../README.md#インストール)を参照してください。

## 前提

- Docker / Docker Compose が利用できること。
- API キー等を作業ディレクトリの `.env` に記入してあること（例: `GEMINI_API_KEY` / `SLACK_BOT_TOKEN` / `SLACK_CHANNEL_ID`）。

## 設定ファイルを用意する

設定ファイル（`vox-radio.yaml` / `episode-spec.yaml` など）は、コンテナ内の `init` で生成できます（ローカルに vox-radio を入れる必要はありません）。

```bash
# カレントディレクトリに記入済みサンプル一式を生成する
docker run --rm -v "$PWD:/work" -w /work ghcr.io/canpok1/vox-radio:latest init --sample --output-dir .
```

生成された YAML を編集し、番組内容・キャラクター・Slack 投稿先などを記入します（記入方法は [README の設定方法](../../README.md#設定方法)を参照）。

## compose サンプル

作業ディレクトリに `compose.yaml` として配置します。

```yaml
services:
  # 音声合成エンジン。同梱せず公式イメージを併用する。
  voicevox:
    image: voicevox/voicevox_engine:cpu-latest
    # ホストのブラウザ等から確認したい場合のみ ports を追加する（サービス間通信には不要）。
    # ports: ["50021:50021"]

  vox-radio:
    image: ghcr.io/canpok1/vox-radio:latest
    depends_on:
      - voicevox
    working_dir: /work
    volumes:
      # 作業ディレクトリ全体をマウントし、設定・出力・キャッシュをホスト側に永続化する。
      - ./:/work
    env_file: .env
    environment:
      # 接続先のホスト名 voicevox は compose のサービス名がそのまま使える。
      VOX_RADIO_VOICEVOX_URL: http://voicevox:50021
```

## 実行

`docker compose run` はイメージを自動取得するため、事前の `docker pull` は不要です。イメージの `ENTRYPOINT` は `vox-radio` なので、続けてサブコマンドだけを書きます。

```bash
# 番組生成
docker compose run --rm vox-radio episodegen --spec episode-spec.yaml --out-dir output

# Slack 投稿（最新回の manifest を選ぶ。-t で更新時刻の新しい順）
docker compose run --rm vox-radio slackpost --manifest "$(ls -t output/*_manifest.json | head -n1)" --spec slack-spec.yaml
```

生成から投稿までを一度に回す場合は、`ENTRYPOINT`（`vox-radio` 固定）をシェルに上書きしてコマンドを連結します。

```bash
docker compose run --rm --entrypoint sh vox-radio -c '
  vox-radio episodegen --spec episode-spec.yaml --out-dir output &&
  vox-radio slackpost --manifest "$(ls -t output/*_manifest.json | head -n1)" --spec slack-spec.yaml
'
```

## 仕組み・ポイント

- **状態の永続化** — 作業ディレクトリ（`./`）を `/work` にマウントするため、設定・`output/`（中間ファイル `output/intermediate/` を含む）・`.vox-radio/cache/`（過去回の履歴）がホスト側に残ります。回番号の連続性が保たれ、前回までを踏まえた番組になります。
- **VOICEVOX は別コンテナ** — イメージには含めません。`VOX_RADIO_VOICEVOX_URL` で接続先を指定します（compose では `http://voicevox:50021`）。
- **定期実行** — cron から `docker compose run --rm ...` を呼べば、ローカル端末でも定期生成・投稿ができます。GitHub Actions で回す場合は [GitHub Actions で定期投稿](github-actions-slack.md)を参照してください。

## 注意事項

- **Apple Silicon など arm64 環境の VOICEVOX** — VOICEVOX は CPU 版の arm64 イメージ（`cpu-arm64-*` 系タグ）を提供しています。`cpu-latest` が arm64 を含まない場合は、[Docker Hub のタグ一覧](https://hub.docker.com/r/voicevox/voicevox_engine/tags)で最新の arm64 向けタグを確認して指定してください。GPU（nvidia）版は arm64 では利用できません。
- **クレジット表記・利用規約** — 合成音声を公開する際の VOICEVOX のクレジット表記など、規約まわりの責任は利用者にあります。詳細は [DISCLAIMER.md](../../DISCLAIMER.md) を参照してください。
