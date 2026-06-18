# 0082. Dependabot による依存自動更新と patch/minor の自動マージ

- ステータス: 採用
- 日付: 2026-06-18

## コンテキスト

vox-radio は Go モジュール（`go.mod`）、GitHub Actions（`.github/workflows/`）、devcontainer features（`.devcontainer/devcontainer.json`）に依存している。これらの依存更新は従来手動で、追従漏れによりセキュリティ修正や互換性更新が遅れるリスクがあった。

依存更新を自動化したいが、次の要件がある。

- 更新頻度は週 1 回に抑え、通知ノイズを減らしたい。
- 同時に複数の更新が出る場合は PR を 1 つにまとめ、レビュー・CI 実行の手間を減らしたい。
- 安全な更新（patch / minor）はレビューなしで取り込み、メンテナンス負荷を下げたい。一方で破壊的変更を含みやすい major 更新は人手で確認したい。

## 決定

`.github/dependabot.yml` を追加し、上記 3 エコシステムに対して週次（`interval: weekly`）で更新を行う。各エコシステムで `groups` を用い、`update-types` に `minor` / `patch` を指定して、これらを 1 つの PR に集約する。major 更新はグループ対象外とし、Dependabot が個別 PR を作成する。

自動マージは `.github/dependabot.yml` 単体では実現できないため、`.github/workflows/dependabot-auto-merge.yml` を追加する。`pull_request` イベントで起動し、`github.actor == 'dependabot[bot]'` の PR に対して `dependabot/fetch-metadata` で更新種別を取得し、`update-type` が `semver-patch` / `semver-minor` の場合のみ `gh pr merge --auto --squash` で自動マージを有効化する。グループ PR の `update-type` はグループ内で最も大きい semver 変更が反映されるため、minor/patch グループは自動マージ対象になる。

前提として、リポジトリ設定で「Allow auto-merge」を有効化する必要がある（コードでは設定できないため運用で担保する）。`--auto` は必須ステータスチェックの完了を待ってからマージするため、ブランチ保護で `build` を必須チェックに設定することを推奨する。

## 結果

- 依存が週次で自動追従され、追従漏れによるセキュリティ・互換性リスクが下がる。
- 複数更新が 1 PR に集約され、レビュー・CI 実行回数が減る。
- patch / minor は CI 通過後に自動マージされ、メンテナンス負荷が下がる。
- major 更新は個別 PR・手動マージのままとなり、破壊的変更を人手で確認できる。
- 「Allow auto-merge」とブランチ保護の設定はリポジトリ管理画面での手動操作が必要で、ADR・README には記載するが自動化はされない。

## 検討した代替案

- **すべての更新を自動マージ**: メンテナンス負荷は最小だが、major 更新の破壊的変更が無確認で取り込まれるリスクがあるため却下。
- **全更新を 1 グループに集約（update-type で分けない）**: 単純だが、グループ内に major が 1 件でもあるとグループ全体が自動マージ対象外になり、安全な更新の取り込みまで滞るため却下。minor/patch と major をグループで分離する方式を採用。
- **自動マージを行わず PR 作成のみ**: 安全だが手動マージの手間が残り、自動化の主目的（メンテナンス負荷低減）を満たさないため却下。
