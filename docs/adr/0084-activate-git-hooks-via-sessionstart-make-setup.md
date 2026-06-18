# 0084. SessionStart フックで make setup を実行し git フックを有効化する

- ステータス: 採用
- 日付: 2026-06-18

## コンテキスト

ADR-0083 で lefthook による git フック（`main`/`master` への直接コミット拒否・fmt/lint/test の品質ゲート）を導入した。しかしフックの有効化（`lefthook install` による `.git/hooks/` への配置）は `make setup`（手動）と devcontainer の `post-create.sh` でしか行われていなかった。

`.git/hooks/` は clone ローカルでリポジトリ管理外のため、**Claude Code on the web のようにセッションごとにフレッシュなコンテナで動く環境では `make setup` が走らず、git フックが無効**になる（実測で lefthook 未導入・`.git/hooks/` が空であることを確認）。エージェントが誤って `main` へコミットする事故を防ぐという本来の目的が、肝心のエージェント環境で達成できていなかった。

一方、従来は Claude 層の PreToolUse フック `.claude/hooks/block-commit-on-main.sh` で `git commit` を拒否していたが、判定が `"git commit"*` 始まりのコマンドに限られ、`git add -A && git commit ...` のような複合コマンドをすり抜けるという穴があった（事故が再発していた一因と推測される）。

## 決定

- Claude 層の簡易ガード `.claude/hooks/block-commit-on-main.sh` と、それを呼ぶ `settings.json` の `PreToolUse` 設定を**廃止**する。
- 代わりに `.claude/settings.json` に **SessionStart フック**を追加し、新規スクリプト `.claude/hooks/session-setup.sh` から **`make setup` を無条件で実行**する。これにより、あらゆる Claude Code セッション開始時に lefthook の git フックが有効化され、git 層で `main` 直コミットを拒否できる（複合コマンドも確実にブロックされる）。
- SessionStart の `matcher` は省略し、全ソース（startup/resume/clear/compact）で実行する。`make setup` はインストール済みなら再実行が約 1 秒で済むため、毎回実行しても実害が小さい。
- スクリプトの `make setup` 出力は stderr に流して stdout を空に保ち、SessionStart の stdout が Claude のコンテキストへ取り込まれてノイズになるのを避ける。失敗時は非ゼロ終了させてエラーを可視化する。

「web のときだけ実行」「フックが既にあればスキップ」といった条件分岐は設けず、シンプルさと確実性を優先して**常に実行**する方針とした。

## 結果

- Claude Code on the web を含む全セッションで git フックが有効化され、`main`/`master` への直接コミットが git 層で拒否される。複合コマンド経由の事故も塞がる。
- Claude 層の独自ガード（すり抜けのある実装）を撤去し、ガードを lefthook に一本化できた。
- セッション開始時のコスト: 実測で導入済みなら約 1 秒、フレッシュコンテナでのコールドは当初約 3 分（go install でのビルド）だった。web は毎セッションがフレッシュなため毎回コールドになる点が課題だったが、ADR-0085 でビルド済みバイナリ取得に切り替え、コールドも約 2 秒に短縮した。
- トレードオフ:
  - Go/ネットワークが未整備のローカル環境で Claude Code を使う場合、毎セッション開始で `make setup` がコールド（最悪は失敗）する。
  - SessionStart はノンブロッキングのため、`make setup` が失敗するとフックが無効のままセッションが進む。失敗は stderr で可視化されるが、git 層ガードが効かない状態になりうる点は許容（`--no-verify` でバイパス可能なのと同様、事故防止策の位置づけ）。

## 検討した代替案

- **`CLAUDE_CODE_REMOTE=true` のとき（web）だけ実行**: ローカルへの影響を避けられるが、web のコールドコストは変わらず（web は毎回フレッシュ）、分岐で複雑になる。確実性とシンプルさを優先し却下。
- **フックが既に有効ならスキップするガードを入れる**: 同一コンテナでの再実行を 0 秒にできるが、go install の再実行も約 1 秒で十分軽いため、シンプルさを優先して採用せず。
- **Claude 層 PreToolUse ガードの判定を複合コマンド対応に改修して併存**: git 層で塞げば十分であり、二重管理を避けるため撤去を選択。
