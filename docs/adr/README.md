# Architecture Decision Records (ADR)

重要度の高い判断を記録する。フォーマットは MADR 軽量版（日本語本文）。

新しい ADR は `create-adr` スキルで作成する（手動実行のほか、重要な判断時に LLM 判断でも起動する）。

| 番号 | タイトル | ステータス | 日付 |
|------|----------|------------|------|
| [0001](0001-remove-vox-actor-dependency.md) | vox-actor 依存の除去 | 採用 | 2026-05-30 |
| [0002](0002-openai-compatible-llm-abstraction.md) | LLM プロバイダ抽象と OpenAI 互換ワイヤープロトコル | 採用 | 2026-05-30 |
| [0003](0003-multi-stage-llm-script-pipeline.md) | 台本生成を多段 LLM パイプラインに分割する | 採用 | 2026-05-30 |
| [0004](0004-file-based-intermediate-artifacts.md) | 中間生成物をファイルで疎結合に繋ぎ、入出力規約を定める | 採用 | 2026-05-30 |
| [0005](0005-podcast-rss-only-distribution.md) | 配信を Podcast(RSS) に一本化する | 採用 | 2026-05-30 |
| [0006](0006-ghpages-hosting-github-actions-runtime.md) | ホスティングを ghpages、実行基盤を GitHub Actions に一本化する | 採用 | 2026-05-30 |
| [0007](0007-ffmpeg-audio-assembly.md) | 音声整形に ffmpeg を採用する | 採用 | 2026-05-30 |
| [0008](0008-collect-rss-html-parser-libraries.md) | collect パッケージの RSS/HTML パースライブラリ選定 | 採用 | 2026-05-30 |
| [0009](0009-cli-framework-cobra.md) | CLIフレームワークにcobraを採用 | 採用 | 2026-05-30 |
| [0010](0010-split-config-into-common-and-profile.md) | 設定を共通設定(Config)とジャンル別設定(Profile)に二分割する | 採用 | 2026-05-30 |
| [0011](0011-restructure-config-schema-characters-program-corners.md) | 設定スキーマを全面再編し、キャラカタログ・番組(program)・コーナー(corners)を導入する | 採用 | 2026-05-31 |
| [0012](0012-separate-distribution-and-content-manifest.md) | 配信機能を別リポジトリへ分離し、vox-radio はコンテンツ manifest を出力する | 採用 | 2026-05-31 |
| [0013](0013-migrate-feed-parser-to-gofeed.md) | フィードパーサを自作XMLパーサからgofeedへ移行する | 採用 | 2026-05-31 |
| [0014](0014-audio-asset-segment-type-unification.md) | 音声アセット（SE/BGM/Jingle）を script.json のセグメント型に統一し、ジングルをラン単位の serial 再生に変更する | 採用 | 2026-05-31 |
| [0015](0015-rundown-step-as-single-source-of-truth.md) | 番組構成(rundown)ステップを新設し、扱う記事の正とする | 採用 | 2026-05-31 |
| [0016](0016-move-op-ed-jingle-injection-to-script-generation.md) | OP/EDジングルの挿入タイミングをassembleステップからscript生成ステップへ移行する | 採用 | 2026-05-31 |
| [0017](0017-separate-direction-field-and-corner-nested-lines.md) | 演出説明フィールドを台本生成から分離し、セリフをコーナー単位のネスト構造で保持する | 採用 | 2026-05-31 |
| [0018](0018-manifest-corner-summary.md) | コーナー単位サマリーを manifest に追加し履歴キャッシュへ集約する | 採用 | 2026-06-01 |
