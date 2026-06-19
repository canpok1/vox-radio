# 0089. vox-radio を Homebrew Cask で配布し ffmpeg を依存に含める

- ステータス: 採用
- 日付: 2026-06-19

## コンテキスト

現状の配布は `install.sh`（GitHub Releases）とエージェントスキル（ADR-0039）で、必須依存の ffmpeg（ADR-0007・ADR-0070）は利用者が別途導入する必要がある。とくに ADR-0070 は「Apple Silicon での ffmpeg 導入の手間が未解決」と記録していた。利用者が vox-radio 本体と ffmpeg をワンステップで導入できる経路が欲しい。

別リポジトリ vox-actor は既に Homebrew Cask（tap: `canpok1/homebrew-tap`）で配布しており、同じ仕組みを踏襲できる。vox-radio は既に GoReleaser とタグ起動のリリースワークフローを持つため、設定追加だけで対応できる。なお Homebrew は brew#19121（4.5.0, 2025）以降 Linux の Cask に対応し、GoReleaser の `homebrew_casks` はビルド済みバイナリ全対象（macOS/Linux）の Cask を生成できる。実際に vox-actor の Cask も Darwin/Linux × amd64/arm64 を含み、devcontainer（Ubuntu）でも `brew install --cask` で導入している。

## 決定

GoReleaser に `homebrew_casks` を追加し、vox-actor と共用の tap `canpok1/homebrew-tap` へ Cask を publish する。

- **ffmpeg を Cask の依存に含める**（GoReleaser の `dependencies` で `depends_on formula: "ffmpeg"` を生成）。`brew install --cask` 時に ffmpeg を自動導入する。formula は Linux でも動くため両 OS で効く。
- 生成される Cask は vox-radio の既存ビルド対象（macOS/Linux × amd64/arm64）をカバーし、macOS・Linux 双方の Homebrew で導入できる。
- post-install で macOS の quarantine 属性を解除する（`OS.mac?` ガード付き。Linux では no-op）。
- tap への push 認証は `HOMEBREW_TAP_GITHUB_TOKEN`（vox-actor 用と同じ PAT を流用）を使う。
- `install.sh` は Homebrew を使わない利用者向け・Windows 向けの経路として継続する。

## 結果

- macOS・Linux の Homebrew 利用者は `brew install --cask` 一発で vox-radio と ffmpeg を導入でき、ADR-0070 の Apple Silicon 課題（および Linux での導入の手間）が解消する。
- vox-actor と tap・配布方式を共有し保守が一貫する。
- トレードオフ: tap publish に PAT の事前設定が要る。Homebrew 非利用者・Windows 向けに `install.sh` を併存させる（チャネルは増えるが既存資産の継続）。Cask は概念上 GUI 向けだが GoReleaser が `binary` stanza を生成し CLI も symlink される。

## 検討した代替案

- **Homebrew Formula（`brews`）**: かつては Linux 対応が Cask に対する利点だったが、Homebrew が Linux Cask に対応した現在その差は消え、GoReleaser も `brews` を非推奨化して `homebrew_casks` を推奨している。利点が無く不採用。
- **ffmpeg を依存にせず手動導入を案内**: 現状維持で導入の手間が残り、ADR-0070 の課題が解消しないため不採用。
- **専用 tap を新設**: 管理対象が増えるだけで利点が薄く、既存 tap を再利用した。
