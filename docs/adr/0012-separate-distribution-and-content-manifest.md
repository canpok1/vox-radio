# 0012. 配信機能を別リポジトリへ分離し、vox-radio はコンテンツ manifest を出力する

- ステータス: 採用
- 日付: 2026-05-31

## コンテキスト

vox-radio は `collect → script → synth → assemble → publish → prune` を担っていた。後半の `publish`（ホスティングへのコピー・`episodes.json` 更新・RSS `feed.xml` 生成・gh-pages への `git push`）と `prune`（古いエピソード削除）は、コンテンツ生成ではなく「配信」の関心事であり、ツールの責務を曖昧にしていた。また `program`（旧 `podcast`）設定に配信専用フィールド（`cover_image_url`・`site_url`・`max_items` 等）が混在していた。

配信方式（RSS 一本化＝ADR-0005、ghpages＋GitHub Actions＝ADR-0006）は今後変わりうるが、それらが本体へ密結合していると差し替えや独立した進化が難しい。

## 決定

- vox-radio を **「コンテンツ → mp3 生成 + コンテンツ manifest 出力」** までに絞る。`publish`/`prune` と `internal/publish`（`feed`/`hosting` 含む）を削除し、**配信は別リポジトリへ分離**する。
- mp3 と同時に、番組内容を記した **コンテンツ manifest（JSON, `manifest.json`）** をサイドカー出力する。配信側リポジトリはこれを入力に feed 生成・配信する。manifest がツール間の契約となる。
- manifest は**番組内容**（`title` / `description` / `datetime` / `audio_file` / `corners[].articles`）を記し、**技術的事実（bytes/duration）は含めない**（配信側が必要なら ffprobe で取得）。`datetime` は manifest 出力時点の RFC3339・UTC。
- `program` 設定から配信専用フィールド（`language`/`author`/`category`/`explicit`/`cover_image_url`/`site_url`/`max_items`）を削除し、生成関連（`title`/`description`/`segment_pause_sec`、将来 `target_duration_sec`）のみ残す。
- 台本ベースの番組要約（`manifest.summary`）は将来の LLM ステップとして追加できるよう、manifest を拡張可能な形にしておく。

## 結果

- vox-radio の責務が「番組コンテンツ生成」に明確化し、配信方式（RSS/ghpages 等）の変更が本体へ波及しなくなる。
- 配信ロジックの実装・テスト・運用は別リポジトリへ移る。**ADR-0005（RSS 一本化）/ ADR-0006（ghpages＋Actions）の配信判断は、その別リポジトリ側で引き継ぐ**（vox-radio 本体からは該当実装を削除）。
- 成果物は mp3 と manifest の 2 つになり、配信側は manifest を読んで `episodes.json` 蓄積・`feed.xml` 生成・prune・公開を行う。
- 段階移行のため、publish/prune 削除の前に manifest 出力を先に追加（非破壊）し、4 つの Issue（manifest 出力 / publish・prune 削除 / program 整理 / 要約 follow-up）に分割して依存順に進める。

## 検討した代替案

- **publish/prune を専用シェルスクリプトで実現**: RSS XML・`episodes.json` の構造管理がシェルだと壊れやすくテストしづらいため却下。
- **同一リポジトリ内の別バイナリに分離**: 依存重複は避けられるが、配信方式の独立した進化・リリースができず責務分離が中途半端なため却下。
- **manifest に配信メタ（cover_image_url 等）も含めて自己完結化**: vox-radio に配信の関心事が残り責務が曖昧になるため却下。配信メタは配信側リポジトリの設定が持つ。
- **manifest に bytes/duration を含める**: 配信側が ffprobe で取得でき、vox-radio を技術的事実の供給源にする必要がないため含めない。
