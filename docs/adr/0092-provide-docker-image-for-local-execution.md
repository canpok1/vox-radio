# 0092. ローカル実行向けに公式 Docker イメージを提供する（ADR-0006 改訂）

- ステータス: 採用
- 日付: 2026-06-28

## コンテキスト

配布は prebuilt binary（install.sh）・Homebrew Cask（[ADR-0089](0089-distribute-via-homebrew-cask-with-ffmpeg-dependency.md)）・エージェントスキル（[ADR-0039](0039-distribute-vox-radio-as-installable-agent-skill.md)）の3経路がある。だがローカル実行には Go・ffmpeg・VOICEVOX の個別準備が要り、とくに Apple Silicon の ffmpeg 導入の手間は[ADR-0070](0070-keep-ffmpeg-over-mp3-alternatives.md)で未解決のまま残っていた。一方 [ADR-0006](0006-ghpages-hosting-github-actions-runtime.md) は「運用用 docker-compose / Dockerfile は作らない」と決めていたが、「設定ファイルを置いてコンテナを起動するだけ」で動かせる需要があり、依存準備の手間を一掃できる。

## 決定

公式 Docker イメージを提供し、ADR-0006 の「運用用 Dockerfile は作らない」方針を改訂する。

- イメージは vox-radio バイナリ + ffmpeg のみの軽量構成（プロンプトはバイナリ埋め込み＝[ADR-0023](0023-embed-prompts-in-binary.md)）。
- VOICEVOX は同梱せず、公式イメージ `voicevox/voicevox_engine` を併用する。
- `compose.yaml` はリポジトリに同梱せず、README/ドキュメントにサンプルとして記載する。
- 配布は ghcr.io。リリース CI から `vX.Y.Z` と `latest` を push し、amd64 / arm64 のマルチアーキに対応する。

## 結果

- 利用者は設定ファイルと compose だけで起動でき、Go・ffmpeg・VOICEVOX の個別準備が不要になる。Apple Silicon の ffmpeg 手間（ADR-0070）も解消する。
- 配布チャネルが4つに増え、CI（マルチアーキビルド・publish）と保守の負担が増える。
- compose を同梱しないため、利用者はドキュメントのサンプルをコピーする必要がある（その分リポジトリの保守対象は増えない）。
- VOICEVOX 非同梱のため、利用者は別途 VOICEVOX コンテナを用意する（公式イメージで容易）。

## 検討した代替案

- **VOICEVOX 同梱の単一イメージ**: 1コマンドで完結するが、イメージが +1GB 超に肥大し、同梱配布のライセンス/クレジット責任と CPU/GPU・アーキ差を抱えるため却下。
- **compose.yaml のリポジトリ同梱**: 利用者の手間は減るが、運用ファイルをリポジトリで保守する負担と ADR-0006 の精神からドキュメント記載に留める。
- **Docker Hub 配布**: 知名度と短い名前は利点だが、匿名 pull のレート制限があり CI 連携に追加シークレットが要る。ghcr.io は public で pull 無制限・CI 連携が容易なため主とする（将来 Docker Hub 併用も可）。
