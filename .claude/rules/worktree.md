# worktree 作業ルール

## ファイル編集前の確認

worktree 環境でファイルを編集する前に、必ず以下を実行してカレントディレクトリとブランチを確認すること。

```bash
pwd
git status
```

- worktree と main リポジトリのパスを混同しないよう、編集先が意図したブランチであることを確認してから作業を始める。

## PR マージ方法

worktree 環境では `gh pr merge` が失敗するため、代わりに GitHub API を直接呼び出してマージすること。

```bash
gh api repos/{owner}/{repo}/pulls/{pr_number}/merge \
  --method PUT \
  --field merge_method=squash \
  --field commit_title="{タイトル}"
```

- `gh pr merge` は main チェックアウト済み worktree 環境では動作しない。
