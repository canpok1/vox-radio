# 0089. vox-radio を Homebrew Cask で配布し ffmpeg を依存に含める

- ステータス: 採用
- 日付: 2026-06-19

## コンテキスト

現状の配布は `install.sh`（GitHub Releases）とエージェントスキル（ADR-0039）で、必須依存の ffmpeg（ADR-0007・ADR-0070）は利用者が別途導入する必要がある。とくに ADR-0070 は「Apple Silicon での ffmpeg 導入の手間が未解決」と記録していた。macOS 利用者が vox-radio 本体と ffmpeg をワンステップで導入できる経路が欲しい。

別リポジトリ vox-actor は既に Homebrew Cask（tap: `canpok1/homebrew-tap`）で配布しており、同じ仕組みを踏襲できる。vox-radio は既に GoReleaser とタグ起動のリリースワークフローを持つため、設定追加だけで対応できる。

## 決定

GoReleaser に `homebrew_casks` を追加し、vox-actor と共用の tap `canpok1/homebrew-tap` へ Cask を publish する。

- **ffmpeg を Cask の依存に含める**（GoReleaser の `dependencies` で `depends_on formula: "ffmpeg"` を生成）。`brew install --cask` 時に ffmpeg を自動導入する。
- vox-actor と同様に post-install で macOS の quarantine 属性を解除する。
- tap への push 認証は `HOMEBREW_TAP_GITHUB_TOKEN`（vox-actor 用と同じ PAT を流用）を使う。
- 配布は Cask 一本とし、Linux 向けは既存 `install.sh`／apt を継続する（Linuxbrew は Cask 非対応のため）。

## 結果

- macOS 利用者は `brew install --cask` 一発で vox-radio と ffmpeg を導入でき、ADR-0070 の Apple Silicon 課題が解消する。
- vox-actor と tap・配布方式を共有し保守が一貫する。
- トレードオフ: Linux は Homebrew 経路を持たず既存手段を継続する。tap publish に PAT の事前設定が要る。Cask は概念上 GUI 向けだが GoReleaser が `binary` stanza を生成し CLI も symlink される。

## 検討した代替案

- **Homebrew Formula（`brews`）**: macOS/Linux 両対応だが vox-actor と方式が分かれ、`brews` は Cask より将来の保守優先度が低い。Linux は `install.sh` で足りるため不採用。
- **Cask と Formula の両方**: 双方をカバーできるが設定・tap 管理が増え、利点に対し保守負担が大きく不採用。
- **ffmpeg を依存にせず手動導入を案内**: 現状維持で導入の手間が残り ADR-0070 の課題が解消しないため不採用。
- **専用 tap を新設**: 管理対象が増えるだけで利点が薄く、既存 tap を再利用した。
