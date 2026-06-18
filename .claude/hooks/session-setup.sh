#!/bin/sh
# SessionStart hook: lefthook の git フックを有効化するため make setup を実行する。
#
# Claude Code on the web 等のフレッシュなコンテナでは git フック（.git/hooks/）が
# 未インストールのため、main への直接コミットを拒否する lefthook フックが効かない。
# セッション開始時に make setup（lefthook install を含む）を走らせて確実に有効化する。
# 設計判断は docs/adr/0084 を参照。
set -eu

# フックの実行カレントは不定なためリポジトリルートへ移動する。
# git リポジトリ外（取得失敗）なら何もせず正常終了する。
root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
cd "$root"

# make setup の出力は stderr に流し、stdout を空に保つ
# （SessionStart の stdout は Claude のコンテキストへ取り込まれ得るため）。
# 失敗時は set -e により非ゼロ終了し、エラーがログに表示される。
make setup 1>&2
