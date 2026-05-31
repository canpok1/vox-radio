#!/bin/bash
# mainブランチにバージョンタグを付与するスクリプト
# 最新リリースのパッチバージョンを1つ進めたタグを作成・push する
set -euo pipefail

DRY_RUN=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run) DRY_RUN=true; shift ;;
        *) echo "不明なオプション: $1" >&2; exit 1 ;;
    esac
done

ALL_TAGS=$(git tag -l "v[0-9]*.[0-9]*.[0-9]*" --sort=-v:refname | { grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' || true; })
LATEST_TAG=$(echo "$ALL_TAGS" | head -n 1)

if [[ -z "$LATEST_TAG" ]]; then
    NEW_VERSION="v0.0.1"
else
    IFS='.' read -r MAJOR MINOR PATCH <<< "${LATEST_TAG#v}"
    NEW_VERSION="v${MAJOR}.${MINOR}.$((PATCH + 1))"
fi
echo "新しいバージョン: $NEW_VERSION"

if [[ "$DRY_RUN" == "true" ]]; then
    echo "[ドライラン] タグの作成とプッシュをスキップします。"
else
    git tag "$NEW_VERSION"
    git push origin "$NEW_VERSION"
fi
