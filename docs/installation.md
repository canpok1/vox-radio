# インストールガイド

vox-radio の詳細なインストール手順とオプションです。最短手順はルート [README](../README.md#インストール) を参照してください。

## インストールスクリプト

最新リリースの `install.sh` を GitHub Releases から取得して実行します。スクリプトは OS / アーキテクチャを判定し、対応するバイナリをダウンロード・チェックサム検証して設置します。

```bash
curl -fsSL https://github.com/canpok1/vox-radio/releases/latest/download/install.sh | bash
```

## 設置先（INSTALL_DIR）

既定の設置先は `$HOME/.local/bin` です。ユーザー権限のみで完結し、`sudo` は不要です。ディレクトリが無い場合は自動で作成します。別の場所（書き込み権限のあるディレクトリ）に入れる場合は環境変数 `INSTALL_DIR` を指定します。指定先に書き込み権限が無い場合はエラーで停止し、別ディレクトリの指定を促します。

```bash
curl -fsSL https://github.com/canpok1/vox-radio/releases/latest/download/install.sh | INSTALL_DIR=$HOME/bin bash
```

`$HOME/.local/bin` が `PATH` に含まれていない場合は、インストール後に表示される案内に従い、shell の設定ファイル（例: `~/.bashrc`, `~/.zshrc`）へ次の行を追記してください。

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## バージョンの指定

`latest/download` は常に最新リリースを指します。特定バージョンを入れる場合は、URL の `latest/download` をリリースタグに置き換えます。

```bash
curl -fsSL https://github.com/canpok1/vox-radio/releases/download/v0.0.16/install.sh | bash
```

利用可能なバージョンは [GitHub Releases](https://github.com/canpok1/vox-radio/releases) で確認できます。既に同じ／より新しいバージョンが入っている場合、インストールはスキップされます。

## 必要なコマンド・対応環境

- 必要コマンド: `curl`（または `wget`）・`tar`・`sha256sum`（または `shasum`）
- 対応 OS: Linux / macOS
- 対応アーキテクチャ: x86_64 / arm64
