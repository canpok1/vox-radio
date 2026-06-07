# タスク管理ルール

タスク（フォローアップ・バックログ・相談結果・スコープ外指摘の追跡など）の管理は Todoist で行う。各ルール・スキルはツール非依存の「タスク作成／更新／参照」として記述し、具体的な登録先・ツールは本ルールにのみ集約する。

## 登録先

- プロジェクト: `dev`（projectId: `6gpvgxp9jVwCFrVV`）
- セクション: `vox-radio`（sectionId: `6gpvrxp3jVGwvp43`）

projectId / sectionId は名前から特定することもできる（`mcp__todoist__find-projects` / `mcp__todoist__find-sections`）。ID が変わっていた場合は名前で引き直すこと。

## タスクの作成・更新・参照

- **作成**: `mcp__todoist__add-tasks`（上記 projectId / sectionId を指定する）
- **参照・重複確認**: `mcp__todoist__find-tasks`（`projectId` / `sectionId` で絞り込む）
- **更新**: `mcp__todoist__update-tasks` ／ **完了**: `mcp__todoist__complete-tasks`

## ラベル運用

タスクの状態管理には以下のラベルを使う（GitHub Issue 運用から引き継いだもの）。

- `vox-radio`: 対象プロジェクトを示す
- `ready`: 着手してよい（対応可能）状態
- `assign-to-claude`: Claude が対応する対象
- `in-progress`: 対応中

## タスク内容の書き方

- `content`: 何をするかを簡潔・具体的に書く（例: 「`readJSON` / `loadProfile` の重複を共通ヘルパーへ抽出」）。
- `description`: 背景・根拠を Markdown で記載する。「どのメモ / PR のどの指摘か」を必ず明記し、後から再調査コストがかからないようにする。受け入れ条件があればチェックリストで含める。
- **重複登録を避ける** — 作成前に `find-tasks` で同種タスクの有無を確認し、あれば新規作成せず既存タスクへ追記する。

## ワークフロースクリプト（td CLI）

`workflow-scripts/` の自動化スクリプト（auto-assign / auto-solve / solve-task など）は、MCP ではなく Todoist CLI（`td` = `@doist/todoist-cli`）でタスクを参照・更新する。

- devcontainer で `npm install -g @doist/todoist-cli` により導入される。
- 認証は環境変数 `TODOIST_API_TOKEN` を使う。
- 対象タスクの絞り込みはフィルタクエリ `#dev & /vox-radio`（プロジェクト dev・セクション vox-radio）を使う。

## 注意

- Todoist タスクには `🤖 Generated with [Claude Code]...` フッターは不要（フッターは GitHub の PR・コメント向けの規約）。
- GitHub は PR・コードレビューにのみ使用し、タスク・バックログ管理には GitHub Issue を使わない。
