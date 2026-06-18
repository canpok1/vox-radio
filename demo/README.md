# vox-radio 紹介デモ番組

設定ファイルから生成AIと VOICEVOX でラジオ番組を自動生成する CLI ツール **vox-radio** の機能を、ずんだもんとめたんが紹介するデモ番組です。

| ファイル | 内容 |
|---|---|
| `vox-radio.yaml` | 共通設定（LLM・VOICEVOX・キャラクター） |
| `episode-spec.yaml` | 番組構成（コーナー・出演者・アセット参照） |

BGM・ジングル・効果音はリポジトリ同梱の [`../sample-assets/`](../sample-assets/) を参照します（このディレクトリには音源を持ちません）。

## 前提

[ルートの README](../README.md) のとおり、生成AIの API キー・VOICEVOX Engine（既定 `http://localhost:50021`）・ffmpeg が必要です。

## 生成手順

1. vox-radio をインストール

   ```bash
   curl -fsSL https://github.com/canpok1/vox-radio/releases/latest/download/install.sh | bash
   ```

2. このディレクトリへ移動

   ```bash
   cd demo
   ```

3. `.env` を作成し、必要な環境変数を設定する（`.env` は Git 管理対象外）

   ```bash
   echo "GEMINI_API_KEY=<your-key>" > .env
   ```

   - 環境変数 `GEMINI_API_KEY` を設定済みなら、この行は不要です。
   - VOICEVOX Engine が既定の `http://localhost:50021` 以外で動いていて、環境変数 `VOX_RADIO_VOICEVOX_URL` も未設定の場合は、接続先も設定します（または `vox-radio.yaml` の `voicevox.url` を変更）。

     ```bash
     echo "VOX_RADIO_VOICEVOX_URL=http://<host>:<port>" >> .env
     ```

4. 番組を生成

   ```bash
   vox-radio episodegen --spec episode-spec.yaml
   ```

   `output/vox-radio-demo_ep001.mp3` が生成されます。

## クレジット

生成した音声を公開する場合は、VOICEVOX（例: `VOICEVOX:ずんだもん` / `VOICEVOX:四国めたん`）と効果音（[`../sample-assets/CREDITS.md`](../sample-assets/CREDITS.md)）のクレジット表記が必要です。
