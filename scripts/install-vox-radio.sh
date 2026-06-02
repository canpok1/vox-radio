#!/usr/bin/env bash
# vox-radio バイナリを GitHub Releases から取得して INSTALL_DIR に配置する。
#
# 使い方:
#   bash install-vox-radio.sh [VERSION]
#   curl -fsSL https://raw.githubusercontent.com/canpok1/vox-radio/main/scripts/install-vox-radio.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/canpok1/vox-radio/main/scripts/install-vox-radio.sh | bash -s -- v0.0.1
#
# 引数:
#   VERSION  取得するリリースタグ (例: v0.0.1)。省略時は最新版を自動解決。
#
# 環境変数:
#   INSTALL_DIR  バイナリ設置先ディレクトリ (デフォルト: /usr/local/bin)
set -euo pipefail

REPO="canpok1/vox-radio"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

usage() {
  echo "Usage: $0 [VERSION]" >&2
  echo "  VERSION: release tag (e.g. v0.0.1). Omit to install the latest." >&2
  echo "" >&2
  echo "Environment variables:" >&2
  echo "  INSTALL_DIR  Installation directory (default: /usr/local/bin)" >&2
}

# ---- 引数パース ----
if [[ $# -gt 1 ]]; then
  usage
  exit 1
fi

if [[ ${1:-} == "-h" || ${1:-} == "--help" ]]; then
  usage
  exit 0
fi

VERSION="${1:-}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: $1 が必要です" >&2
    exit 1
  fi
}

# ---- 必須コマンド検出 ----
if command -v curl >/dev/null 2>&1; then
  DOWNLOADER="curl"
elif command -v wget >/dev/null 2>&1; then
  DOWNLOADER="wget"
else
  echo "ERROR: curl または wget が必要です" >&2
  exit 1
fi

require_cmd tar

# sha256 検証コマンドを選択
if command -v sha256sum >/dev/null 2>&1; then
  SHA256CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  SHA256CMD="shasum -a 256"
else
  echo "ERROR: sha256sum または shasum が必要です" >&2
  exit 1
fi

# ---- OS・arch 判定 ----
os_raw="$(uname -s)"
case "$os_raw" in
  Linux)  OS="Linux" ;;
  Darwin) OS="Darwin" ;;
  *)
    echo "ERROR: 未対応の OS: $os_raw (Linux / macOS のみサポート)" >&2
    exit 1
    ;;
esac

arch_raw="$(uname -m)"
case "$arch_raw" in
  x86_64 | amd64)   ARCH="x86_64" ;;
  aarch64 | arm64)  ARCH="arm64" ;;
  *)
    echo "ERROR: 未対応の arch: $arch_raw (x86_64 / arm64 のみサポート)" >&2
    exit 1
    ;;
esac

# ---- ダウンロードヘルパー ----
download() {
  local url="$1"
  local out="$2"
  if [[ "$DOWNLOADER" == "curl" ]]; then
    curl -fsSL -o "$out" "$url"
  else
    wget -q -O "$out" "$url"
  fi
}

download_stdout() {
  local url="$1"
  if [[ "$DOWNLOADER" == "curl" ]]; then
    curl -fsSL "$url"
  else
    wget -qO- "$url"
  fi
}

# ---- latest 解決 ----
if [[ -z "$VERSION" ]]; then
  echo "最新バージョンを GitHub API から解決しています..."
  VERSION="$(download_stdout "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
  if [[ -z "$VERSION" ]]; then
    echo "ERROR: latest バージョンの解決に失敗しました" >&2
    exit 1
  fi
  echo "最新バージョン: $VERSION"
fi

# VERSION から先頭 "v" を除いたもの（checksums ファイル名用）
VERSION_NUM="${VERSION#v}"

ASSET_NAME="vox-radio_${OS}_${ARCH}.tar.gz"
CHECKSUMS_NAME="vox-radio_${VERSION_NUM}_checksums.txt"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
TARBALL_URL="${BASE_URL}/${ASSET_NAME}"
CHECKSUMS_URL="${BASE_URL}/${CHECKSUMS_NAME}"

# ---- 一時ディレクトリ（EXIT 時に自動削除） ----
TMPDIR_WORK="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_WORK"' EXIT

# ---- ダウンロード ----
echo "ダウンロード中: $TARBALL_URL"
download "$TARBALL_URL" "$TMPDIR_WORK/$ASSET_NAME"

echo "ダウンロード中: $CHECKSUMS_URL"
download "$CHECKSUMS_URL" "$TMPDIR_WORK/$CHECKSUMS_NAME"

# ---- sha256 検証 ----
echo "チェックサムを検証しています..."
checksum_line="$(grep "$ASSET_NAME" "$TMPDIR_WORK/$CHECKSUMS_NAME" || true)"
if [[ -z "$checksum_line" ]]; then
  echo "ERROR: checksums.txt に $ASSET_NAME のエントリが見つかりません" >&2
  exit 1
fi
echo "$checksum_line" | (cd "$TMPDIR_WORK" && $SHA256CMD --check -)
echo "チェックサム OK"

# ---- 展開 ----
tar -xzf "$TMPDIR_WORK/$ASSET_NAME" -C "$TMPDIR_WORK" vox-radio

# ---- インストール ----
if [[ ! -d "$INSTALL_DIR" ]]; then
  echo "ERROR: INSTALL_DIR が存在しません: $INSTALL_DIR" >&2
  exit 1
fi

if [[ -w "$INSTALL_DIR" ]]; then
  install -m 0755 "$TMPDIR_WORK/vox-radio" "$INSTALL_DIR/vox-radio"
else
  echo "書き込み権限がないため sudo でインストールします..."
  sudo install -m 0755 "$TMPDIR_WORK/vox-radio" "$INSTALL_DIR/vox-radio"
fi

echo ""
echo "インストール完了: $INSTALL_DIR/vox-radio"
"$INSTALL_DIR/vox-radio" --version
