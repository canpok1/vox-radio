# コントリビューションガイド

vox-radio への貢献に興味を持っていただきありがとうございます。バグ報告・機能提案・Pull Request を歓迎します。

## バグ報告・機能提案

- まず [Issues](https://github.com/canpok1/vox-radio/issues) で既存の報告がないか確認してください。
- なければ Issue テンプレートに沿って新規作成してください。
- セキュリティ上の脆弱性は **公開 Issue で報告せず**、[SECURITY.md](SECURITY.md) の手順に従ってください。

## 開発の進め方

開発環境の構築・ビルド・テストの詳細は **[開発ガイド（docs/development/）](docs/development/README.md)** にまとめています。要点は以下のとおりです。

```bash
make build   # ビルド
make test    # 単体テスト
make lint    # golangci-lint
make e2e     # e2e テスト（ffmpeg が必要、外部 API はモック）
```

- 開発環境は devcontainer での構築を推奨します（`make` 系コマンドは開発者向け。利用者はリリース版バイナリのみで完結します）。
- `make setup` を実行すると git フックが有効になります（pre-commit で gofmt・golangci-lint、pre-push で `go test ./...` が自動実行されます）。`--no-verify` でバイパス可能ですが、CI でも同等のチェックが走ります。
- `.go` ファイルを変更するときは [アーキテクチャの依存ルール](docs/development/architecture.md) に従ってください。
- CLI のコマンド・フラグ説明は cobra の `Short`/`Long` を編集し、`make docs` で `docs/cli/` を再生成します（手書きしない）。
- 設定ファイルのフィールド定義は `internal/cli/skills/vox-radio/references/` が正です。
- アーキテクチャ・技術選定など重要な判断は [ADR（docs/adr/）](docs/adr/) に記録しています。

## Pull Request

1. ブランチを切って変更を加える（`main` への直接コミットは禁止です。pre-commit フックでも拒否されます）。
2. `make test` / `make lint` がローカルで通ることを確認する（`make setup` 後は pre-push で自動実行されます）。
3. PR を作成する。CI（`build.yml`）が通ることを確認する。
4. レビューを受けてマージする。

## 補足

- このリポジトリ内部のタスク・バックログ管理は Todoist で行っており、GitHub Issue は外部からのバグ報告・機能提案・質問の窓口として使います。
- ライセンスは [MIT](LICENSE) です。コントリビュートされた変更も同ライセンスで配布されます。
- 利用上の注意・クレジット表記については [DISCLAIMER.md](DISCLAIMER.md) を参照してください。
