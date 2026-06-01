# 0020. BGM・ジングルの挿入をprofile.yamlのコーナー設定駆動へ移行し、program OP/ED指定を廃止する

- ステータス: 採用
- 日付: 2026-06-01

## コンテキスト

ADR 0014 では BGM と中間ジングルの挿入位置を LLM（`direct` ステップ）が判定し、OP/ED ジングルのみ `program.opening_jingle`/`ending_jingle` でコードが決定的に挿入していた。この方式には 3 つの問題があった。（1）LLM 応答により挿入位置・有無が変化し番組構成が不安定になる。（2）BGM を流すコーナーかどうかはコーナー定義時に決められるにもかかわらず実行時まで LLM に委ねていた。（3）OP/ED の program 指定とコーナー定義が分離し、コーナー追加・削除時に両者の同期が必要だった。SE と pause は会話の流れに依存するため LLM 判定を維持するが、BGM とジングルは番組構成として事前に決定できる。

## 決定

**BGM とジングルはコーナー毎に `profile.yaml` で決定的に指定する。**

- `CornerConfig` に `opening_jingle`/`ending_jingle`/`bgm`（任意・`assets` キー参照）を追加する。
- `ProgramConfig.OpeningJingle`/`EndingJingle` および `InjectProgramJingles` を廃止する。OP/ED は各コーナーの `opening_jingle`/`ending_jingle` で表現する。
  ※ 注記: 本ADR記載の `opening_jingle`/`ending_jingle` はその後 `start_jingle`/`end_jingle` にリネームした（Issue #154）。
- `direct` スキーマを SE 専用（`enum: [se]`）に変更し `AssetCatalog` も SE のみに縮小する。
- `CornerLines` にアセットフィールドを持たせ `03_lines.json` に永続化する。`buildScript` がコーナー毎に `opening_jingle → bgm開始 → 本編(SE/pause) → bgm停止 → ending_jingle` の順でセグメントを生成する。
- `ValidateProfileAssets` を追加し不正なキー参照をプロファイル読込時にエラーとして検出する。
- ADR 0014 の「BGM/中間ジングルの LLM 配置」と「OP/ED の program 指定」を本 ADR が部分的に上書きする。`filter.go` のセグメント駆動合成ロジックは変更しない。

## 結果

BGM・ジングルの挿入が決定的になり番組構成の再現性が向上する。`profile.yaml` 一箇所でコーナーの音楽演出が完結し、program/corners 間の二重管理がなくなる。LLM へのカタログ入力が SE のみに縮小しプロンプトが簡潔になる。BGM 停止セグメントをコーナー末尾に明示することで BGM が次コーナーへ漏れない（ADR 0014 の filter.go ルールに合致）。

## 検討した代替案

**LLM 温度を下げて安定化を図る**: 不安定性の根本は判断をモデルに委ねること自体にあり解決できない。却下。

**program OP/ED と corners 両方を残す**: 二重管理が続き移行メリットが薄れる。OP/ED はコーナーで完全に表現可能なため廃止を選択。

**script.json に BGM/ジングルセグメントを手書き**: 管理コストが高く LLM パイプラインとの統合が困難。却下。
