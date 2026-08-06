# 0105. プロンプト品質評価フレームワーク（internal/eval, ADR-0049）を廃止する

- ステータス: 採用
- 日付: 2026-08-06

## コンテキスト

ADR-0049で導入した`internal/eval`は、組み込みプロンプト8種（proofread/summarize/corner_summary/summary/select/flow/write/direct）をGeminiによるLLM-as-judge方式で週次採点し、内容品質の回帰を検知する仕組みである。

ADR-0103の実装（PR #571）でdirect/proofreadの読み変換ルールを変更した結果、`internal/eval/testdata`の回帰ケース・judge採点基準が旧仕様（漢字のかな化前提）のまま陳腐化し、追随修正のタスクを起票した。その過程で調査したところ、evalの実行結果は`.github/workflows/prompt-eval.yml`実行時のGitHub Actionsログにしか出力されず、artifact保存・Slack通知・Issue化・バッジ表示などの可視化/通知手段が一切存在しないことが判明した。結果として、実質「見に行かなければ気づけない」運用になっており、ADR-0049が意図した継続的な回帰検知という価値を回収できていなかった。ADR-0098は「evalは開発者向けCI機構であり本番エピソードの振り返り（retro）には流用しない」と明記しており、代替となる仕組みも存在しない。

## 決定

`internal/eval`パッケージ一式（Goコード・`testdata`）、`.github/workflows/prompt-eval.yml`、`Makefile`の`eval`ターゲット、`.golangci.yml`の`domain-eval`ルール、`docs/development/architecture.md`・`docs/development/README.md`のeval関連記述を削除する。ADR-0049は「廃止」ステータスへ変更し本ADRを参照させる（ADR本文自体は改変しない）。プロンプト内容品質の自動監視機構は持たず、人手レビュー・実運用での気づきに委ねる。

## 結果

- 良い影響: 通知が届かず活用できていなかった週次ジョブの保守コストが消える。ADR-0103のような読み変換仕様の変更のたびに発生していた回帰データ・judge基準の追随作業が今後不要になる。
- 悪い影響: プロンプト内容品質の劣化を自動検知する唯一の手段が失われる。将来的に自動監視が必要になった場合、通知の仕組みを含めて再構築するコストが発生する。

## 検討した代替案

- 失敗時にSlack通知を追加して存続させる案: 既存のslackpost基盤（ADR-0035/0071）の流用を検討したが、通知先設計・実装スコープが新たに必要になる。「有効活用できていない」現状を踏まえ、まず廃止を優先し、品質監視の必要性が再燃した際に通知付きで作り直す方針とした。
