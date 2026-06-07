---
name: assign-tasks
description: dev/vox-radioセクションのreadyタスクを優先度順に評価し、指定件数にassign-to-claudeラベルを付与するスキル
allowed-tools: Bash, Read, Grep, Glob, Agent, mcp__todoist__find-tasks, mcp__todoist__update-tasks
user-invocable: true
argument-hint: "[--count N]"
---

タスクの登録先・ラベル運用は `.claude/rules/task-management.md` に従う。

## 手順

1. `mcp__todoist__find-tasks`（`vox-radio` セクション、`labels: ["ready"]`）で未完了かつ `ready` ラベル付きのタスクを取得する。
2. 以下の条件に該当するタスクを除外する:
   - `assign-to-claude` または `in-progress` ラベルが付いているタスク
   - タイトルまたは本文に `.claude/` パスへの参照を含むタスク
   - `.claude/` ディレクトリの変更を主目的とするタスク（スキル、ルール、フック、CLAUDE.md、自動化関連）
3. 除外対象のタスクから `ready` ラベルを除去し（`mcp__todoist__update-tasks` で `labels` を更新）、除外理由をログに記録する。
4. 残りのタスクを `task-assigner` エージェントの優先度基準に従って優先順位付けする。
5. 上位N件（`$ARGUMENTS` の `--count` で指定、デフォルト: 2件）に `assign-to-claude` ラベルを付与する（`mcp__todoist__update-tasks` で既存 `labels` に追加）。

## 出力

- 除外したタスク: ID、タイトル、除外理由（ターミナルのみ）
- アサインしたタスク: ID、タイトル、判定根拠（ターミナルのみ）

## 制約

- タスクへのコメント投稿（`mcp__todoist__add-comments`）は禁止
- 使用する Todoist 操作: `mcp__todoist__find-tasks`, `mcp__todoist__update-tasks` のみ
- ラベル更新時は既存ラベルを保持したまま追加/除去すること（`update-tasks` の `labels` は全置換のため、取得済みの現ラベルをベースに編集する）
