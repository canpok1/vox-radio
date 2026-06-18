#!/usr/bin/env bash
# lefthook のビルド済みバイナリを GitHub Releases から取得して BIN_DIR に配置する。
#
# `make setup` から呼ばれる開発用スクリプト（go install のビルドを避けて高速化する）。
#
# 使い方:
#   scripts/install-lefthook.sh <version> <bin_dir>
#   例: scripts/install-lefthook.sh v2.1.8 "$(go env GOPATH)/bin"
set -euo pipefail

REPO="evilmartians/lefthook"

VERSION="${1:-}"
BIN_DIR="${2:-}"

if [[ -z "$VERSION" || -z "$BIN_DIR" ]]; then
  echo "Usage: $0 <version> <bin_dir>" >&2
  echo "  例: $0 v2.1.8 \"\$(go env GOPATH)/bin\"" >&2
  exit 1
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: $1 が必要です" >&2
    exit 1
  fi
}

require_cmd curl

# sha256 検証コマンドを選択
if command -v sha256sum >/dev/null 2>&1; then
  SHA256CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  SHA256CMD="shasum -a 256"
else
  echo "ERROR: sha256sum または shasum が必要です" >&2
  exit 1
fi

# ---- OS・arch 判定（lefthook のアセット命名規則に合わせる） ----
os_raw="$(uname -s)"
case "$os_raw" in
  Linux)  OS="Linux" ;;
  Darwin) OS="MacOS" ;;
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

# VERSION から先頭 "v" を除いたもの（アセット・checksums ファイル名用）
VERSION_NUM="${VERSION#v}"

ASSET_NAME="lefthook_${VERSION_NUM}_${OS}_${ARCH}"
CHECKSUMS_NAME="lefthook_checksums.txt"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

# ---- 一時ディレクトリ（EXIT 時に自動削除） ----
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

# ---- ダウンロード ----
echo "ダウンロード中: ${BASE_URL}/${ASSET_NAME}"
curl -fsSL -o "$WORK_DIR/$ASSET_NAME" "${BASE_URL}/${ASSET_NAME}"

echo "ダウンロード中: ${BASE_URL}/${CHECKSUMS_NAME}"
curl -fsSL -o "$WORK_DIR/$CHECKSUMS_NAME" "${BASE_URL}/${CHECKSUMS_NAME}"

# ---- sha256 検証 ----
echo "チェックサムを検証しています..."
checksum_line="$(grep " ${ASSET_NAME}\$" "$WORK_DIR/$CHECKSUMS_NAME" || true)"
if [[ -z "$checksum_line" ]]; then
  echo "ERROR: $CHECKSUMS_NAME に $ASSET_NAME のエントリが見つかりません" >&2
  exit 1
fi
echo "$checksum_line" | (cd "$WORK_DIR" && $SHA256CMD --check -)
echo "チェックサム OK"

# ---- インストール ----
if ! mkdir -p "$BIN_DIR" 2>/dev/null; then
  echo "ERROR: BIN_DIR を作成できません: $BIN_DIR" >&2
  exit 1
fi
install -m 0755 "$WORK_DIR/$ASSET_NAME" "$BIN_DIR/lefthook"

echo "インストール完了: $BIN_DIR/lefthook"
"$BIN_DIR/lefthook" version
