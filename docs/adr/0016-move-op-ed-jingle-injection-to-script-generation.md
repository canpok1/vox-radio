# 0016. OP/EDジングルの挿入タイミングをassembleステップからscript生成ステップへ移行する

- ステータス: 採用
- 日付: 2026-05-31

## コンテキスト

OP/EDジングルは `injectProgramJingles()`（`internal/assemble/filter.go`）により ffmpeg 実行直前のメモリ上でスクリプトへ差し込まれていた。このため `04_script.json` にはジングルセグメントが含まれず、最終的な音声構成をファイルから把握できない非対称な状態だった。中間アイキャッチ（LLM が配置）は `script.json` に現れるのに、OP/ED だけ assemble 内部で暗黙挿入されていた。また `--step direct` でスクリプトを単独再生成した場合もジングルが含まれず、再アセンブル時にジングルが消えるという問題があった。

## 決定

OP/EDジングルの埋め込みを `LLMScriptGenerator.Generate`（`internal/script/script.go`）内、`director.Direct()` 後に移した。`InjectProgramJingles` をエクスポート関数として定義し、`--step direct` 単体実行の CLI パス（`runScriptDirect`）からも同じ関数を呼ぶことで対称性を確保した。assemble 側の `injectProgramJingles` と呼び出しは削除した。

## 結果

生成された `04_script.json` に OP/ED ジングルセグメントが含まれ、音声構成の可視性が向上する。中間アイキャッチとの非対称性が解消され、assemble は `script.json` をそのまま処理するシンプルな責務になる。`--step direct` も同じ関数を使うため単体再生成の結果が一貫する。トレードオフとして、`ScriptGenerator` の実装が `ProgramConfig` からジングルを読んで注入する責務を持つため、将来別の Generator 実装を追加した場合は同じ呼び出しが必要になる。

## 検討した代替案

**A. assemble 維持**: 変更最小だが `script.json` に可視化できないため却下。

**B. Director.Direct 内で注入**: Director はアセット配置の LLM 判断層であり設定値ベースの確定挿入と責務が混在するため却下。

**C. pipeline.Runner で注入**: Generate と assemble の中間で挿入する案。`ScriptGenerator` インターフェースの変更が不要だが、パイプライン層がジングル注入を知る必要があり結合度が上がるため却下。
