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
| [0019](0019-conversation-notes-in-manifest.md) | 番組構成外の会話情報を会話メモとして manifest に残し履歴へ集約する | 採用 | 2026-06-01 |
| [0020](0020-corner-level-bgm-jingle-provenance.md) | BGM・ジングルの挿入をprofile.yamlのコーナー設定駆動へ移行し、program OP/ED指定を廃止する | 採用 | 2026-06-01 |
| [0021](0021-intra-episode-corner-context.md) | コーナー台本生成に同一回の生成済みセリフを文脈として渡す | 採用 | 2026-06-01 |
| [0022](0022-dify-api-llm-provider.md) | LLM プロバイダに Dify API 経由の利用を追加する | 採用 | 2026-06-01 |
| [0023](0023-embed-prompts-in-binary.md) | プロンプトをバイナリに同梱し --prompts フラグを廃止する | 採用 | 2026-06-01 |
| [0024](0024-corner-level-pause-and-remove-program-pause-length.md) | コーナーごとの開始/終了無音設定と program.segment_pause_sec / program.length_sec の削除 | 採用 | 2026-06-01 |
| [0025](0025-se-sequential-playback-default.md) | SE の既定再生方式を順次再生（serial）に変更し per-SE overlay フラグを追加する | 採用 | 2026-06-02 |
| [0026](0026-feed-generation-as-vox-radio-subcommand.md) | feed 生成ツールを vox-radio の feedgen サブコマンドとして集約する（ADR-0012 の修正） | 採用 | 2026-06-02 |
| [0027](0027-move-katakana-conversion-from-write-to-direct.md) | カタカナ化（読み変換）を write から direct へ移し、summary 入力を write 出力に変更する | 採用 | 2026-06-02 |
| [0028](0028-nest-pipeline-steps-under-episodegen.md) | パイプライン各ステップを episodegen サブコマンド配下へ再編する（run のリネーム） | 採用 | 2026-06-03 |
| [0029](0029-separate-asset-config-from-profile.md) | アセット設定のプロファイルからの分離 | 採用 | 2026-06-03 |
| [0030](0030-rename-profile-and-feedgen-config-to-spec.md) | 設定ファイルを episode-spec / feed-spec に統一し「プロファイル」概念を廃止する | 採用 | 2026-06-03 |
| [0031](0031-guest-appearance-by-episode-number-condition.md) | ゲスト出演を回番号条件で制御し、出演確定結果を rundown に永続化する | 採用 | 2026-06-04 |
| [0032](0032-corner-rotation-by-episode-number-condition.md) | コーナーの回ごと組み替えを回番号条件で制御し、採用結果を rundown に乗せる | 採用 | 2026-06-04 |
| [0033](0033-negation-condition-for-episode-condition.md) | 出現条件 EpisodeCondition に否定（補集合）条件 not を追加する | 採用 | 2026-06-04 |
| [0034](0034-rotation-offset-for-episode-condition.md) | 出現条件 EpisodeCondition に剰余（offset）を追加しN者ローテーションを可能にする | 採用 | 2026-06-04 |
| [0035](0035-slack-episode-posting-subcommand.md) | Slack へエピソードを投稿する slackpost サブコマンドを追加する（ADR-0005 の方針転換） | 採用 | 2026-06-04 |
| [0036](0036-unify-cast-roster-with-regular-and-guest-types.md) | 出演者名簿を casts に一本化し、レギュラーと出演条件（お休み）を導入する | 採用 | 2026-06-05 |
| [0037](0037-record-cast-appearances-in-cache-and-pass-counts-to-llm.md) | 出演キャスト実績をキャッシュに記録し、参加回数を rundown 経由で生成 LLM に渡す | 採用 | 2026-06-05 |
| [0038](0038-unify-corner-boundary-audio-config.md) | コーナー境界音声を start_jingle/end_jingle から type 付き start_audio/end_audio へ再構成する | 採用 | 2026-06-05 |
| [0039](0039-distribute-vox-radio-as-installable-agent-skill.md) | vox-radio をインストール可能なエージェントスキルとして配布する | 採用 | 2026-06-06 |
| [0040](0040-appearance-count-include-current-episode-with-boundary-conversion.md) | 出演回数を「今回を含めた回数（初登場=1）」で永続化し、LLM 入力は境界で逆変換して維持する（ADR-0037 の回数定義を置換） | 採用 | 2026-06-06 |
| [0041](0041-sample-config-generation-command.md) | 番組生成お試し用のサンプル設定一式を init --sample で生成し、リポジトリ同梱の examples/ サンプルを廃止する | 採用 | 2026-06-06 |
| [0042](0042-voicevox-url-env-override.md) | VOICEVOX URL に環境変数オーバーライドを導入する | 採用 | 2026-06-06 |
| [0043](0043-deprecate-cache-disable-and-require-program-id.md) | キャッシュ無効化機能（cache.enabled）を廃止し program.id を必須化する | 採用 | 2026-06-06 |
| [0044](0044-switch-sample-feeds-to-jma-weather.md) | サンプル設定のデータソースを気象庁防災情報XMLフィードへ変更する | 採用 | 2026-06-06 |
| [0045](0045-add-pronunciation-proofread-pass-to-direct.md) | direct に発音校正パスを追加し VOICEVOX のかな化取りこぼしによる誤読を抑制する | 採用 | 2026-06-06 |
| [0046](0046-article-source-attribution-to-llm.md) | 記事の出典（サイト名・著者名）を rundown 経由で生成 LLM に渡す | 採用 | 2026-06-06 |
| [0047](0047-program-timezone-and-temporal-context-to-llm.md) | 番組タイムゾーン設定の導入と時間文脈（記事配信日時・収録日時）の生成 LLM への伝達 | 採用 | 2026-06-06 |
| [0048](0048-program-level-direction-and-non-public-script-note.md) | 番組全体の演出指示（direct専用）と台本指示 script_note（write専用・非公開、番組/コーナー両レベル）を追加する | 採用 | 2026-06-06 |
| [0049](0049-prompt-eval-llm-as-judge-framework.md) | 組み込みプロンプトの品質を LLM-as-judge で評価するフレームワークを導入する | 採用 | 2026-06-07 |
| [0050](0050-fail-on-cache-corruption-instead-of-degraded-mode.md) | キャッシュ破損時は degraded mode ではなくエラー停止する | 採用 | 2026-06-07 |
| [0051](0051-http-retry-with-exponential-backoff.md) | 外部 HTTP API 呼び出しに指数バックオフのリトライを導入する | 採用 | 2026-06-08 |
| [0052](0052-corner-id-and-appearance-context-to-llm.md) | コーナーに ID を導入し扱い回数・前回出演回番号を生成 LLM に渡す | 採用 | 2026-06-08 |
| [0053](0053-slackpost-idempotent-resume-via-state-file.md) | slackpost に状態ファイルによる冪等な再投稿を導入する（ADR-0035 の二重投稿方針を改訂） | 採用 | 2026-06-08 |
| [0054](0054-godog-bdd-e2e-tests.md) | godog による BDD e2e テストの導入 | 採用 | 2026-06-10 |
| [0055](0055-slack-api-url-env-override.md) | Slack API URL の環境変数オーバーライド | 採用 | 2026-06-10 |
| [0056](0056-layered-architecture-dependency-rules.md) | レイヤードアーキテクチャと依存方向ルールの明文化 | 採用 | 2026-06-11 |
| [0057](0057-feed-prompt-injection-defense.md) | 外部情報源のプロンプトインジェクション多層防御 | 採用 | 2026-06-11 |
| [0058](0058-decouple-article-dedup-key-from-url.md) | 記事の重複判定をURLから内容ベースの識別キー（DedupKey）へ分離する | 採用 | 2026-06-12 |
| [0059](0059-drop-article-summary-pass-body-to-write.md) | rundown の記事要約を廃止し原文を write に引き渡す | 採用 | 2026-06-12 |
| [0060](0060-drop-whole-article-on-injection-detection.md) | 注入検出記事をフィールド破棄でなく丸ごと除外する | 採用 | 2026-06-12 |
| [0061](0061-drop-rundown-full-text-fetch-pass-feed-text.md) | rundown の全文取得を廃止しフィード由来テキストを description として write へ渡す（ADR-0059 改訂） | 採用 | 2026-06-12 |
| [0062](0062-split-user-and-developer-docs.md) | ドキュメントを利用者向け(README)と開発者向け(docs/development)に分離する | 採用 | 2026-06-13 |
| [0063](0063-unify-init-output-dir-with-flag.md) | init の出力先を --output-dir に一本化し --sample の暗黙 sample/ 出力を廃止する（ADR-0041 改訂） | 採用 | 2026-06-13 |
| [0064](0064-distribute-sample-assets-via-github-release.md) | サンプルアセット音源をリポジトリに同梱し GitHub Release へ自動添付する（ADR-0041 改訂） | 採用 | 2026-06-14 |
| [0065](0065-propagate-asset-and-character-credits-to-outputs.md) | 使用したアセット・キャラクターのクレジットを manifest 経由で feed/Slack へ転記する | 置換済み | 2026-06-14 |
| [0066](0066-id3-tags-for-generated-mp3.md) | 生成 MP3 に ID3 タグを設定し、タイトル取得のため pipeline を並べ替える | 採用 | 2026-06-14 |
| [0067](0067-descriptive-episode-mp3-filename.md) | 生成 MP3 のファイル名を {program.id}_ep{NNN}.mp3 にする（ADR-0026 改訂） | 採用 | 2026-06-14 |
| [0068](0068-manifest-build-owns-credit-collection.md) | manifest.Build がクレジット収集を内包する | 採用 | 2026-06-14 |
| [0069](0069-corner-level-duration-in-outputs.md) | コーナー単位の尺（実測）を成果物に追加し corner_id をパイプライン全体に伝播させる | 採用 | 2026-06-14 |
| [0070](0070-keep-ffmpeg-over-mp3-alternatives.md) | mp3 エンコード代替を見送り ffmpeg 依存を維持する | 採用 | 2026-06-15 |
| [0071](0071-slackpost-text-template-message-spec.md) | slackpost のメッセージ整形を text/template ベースの2テンプレ構成へ刷新する | 採用 | 2026-06-15 |
| [0072](0072-render-command-and-template-file-reference.md) | 汎用 render コマンドを追加し、テンプレをファイルパス参照に統一する | 採用 | 2026-06-15 |
| [0073](0073-corner-defaults-for-bgm-jingle-pause.md) | コーナーの BGM・ジングル・pause に番組共通デフォルト（corner_defaults）を導入する | 採用 | 2026-06-16 |
| [0074](0074-unify-data-source-attribution-into-description-credits.md) | データソース帰属を description クレジット節に統合し feed.credit を廃止する | 採用 | 2026-06-16 |
| [0075](0075-unify-author-source-to-program-config.md) | 番組の著者名の出典を番組設定に一本化する | 採用 | 2026-06-16 |
| [0076](0076-pipeline-progress-log-convention.md) | パイプライン進捗ログの出力規約を定める | 採用 | 2026-06-16 |
| [0077](0077-per-episode-output-layout.md) | episodegen の出力を回数別名・回ごとディレクトリにする | 採用 | 2026-06-16 |
| [0078](0078-init-sample-with-assets-flag.md) | init --sample-with-assets で音入りサンプルを生成する | 採用 | 2026-06-16 |
| [0079](0079-templatize-sample-episode-spec.md) | サンプル episode-spec を text/template で単一ソース化する | 採用 | 2026-06-16 |
| [0080](0080-slackpost-channel-via-env-var-name.md) | slackpost の投稿先チャンネルを環境変数名の間接指定にする | 採用 | 2026-06-16 |
| [0081](0081-load-env-file-on-startup.md) | 起動時に .env ファイルから環境変数を読み込む | 採用 | 2026-06-16 |
| [0082](0082-dependabot-auto-update-with-auto-merge.md) | Dependabot による依存自動更新と patch/minor の自動マージ | 採用 | 2026-06-18 |
| [0083](0083-lefthook-for-git-hooks.md) | lefthook による git フック管理（保護ブランチへの直接コミット禁止・品質ゲート） | 採用 | 2026-06-18 |
| [0084](0084-activate-git-hooks-via-sessionstart-make-setup.md) | SessionStart フックで make setup を実行し git フックを有効化する | 採用 | 2026-06-18 |
