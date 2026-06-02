# 0027. カタカナ化（読み変換）を write から direct へ移し、summary 入力を write 出力に変更する

- ステータス: 採用
- 日付: 2026-06-02

## コンテキスト

manifest.json の `summary` / `episode_title` / `conversation_notes` / コーナー `summary`・`points` が VOICEVOX 向けカタカナ（`AI`→`エーアイ` 等）になっていた。manifest は表示用メタデータのため元の表記（漢字・英字）を保持すべきである。

原因は 2 系統。（1）`summary.md` の `episode_title` セクションにカタカナ化指示があった。（2）`write.md` のカタカナ化ルールによりセリフが変換済みになり、`ProgramSummarizer` がその変換済みテキストを入力としていた。ADR-0003 で定義した多段パイプライン（write→direct→synth）では write と direct の責務が明確に分離されていなかった。

## 決定

1. **write はセリフを元表記で生成する** — `write.md` から誤読防止カタカナ化ルールを削除する。
2. **direct がカタカナ化（読み変換）を担う** — 既存の direct LLM 呼び出しを拡張し、SE/pause 挿入と同一呼び出しで各セリフの VOICEVOX 読み変換テキストも返させる（LLM 呼び出し回数は増やさない）。変換結果が欠落した行は元テキストにフォールバックする。
3. **ProgramSummarizer の入力を write 出力（03_lines.json）に切り替える** — direct 後の最終 Script（04_script.json）ではなく ScriptLines を入力にする。corner_summary は既に 03_lines.json 入力のため変更不要。
4. **`manifest` サブコマンドの `--script` フラグを削除する** — program summary が `--lines` 入力になるため不要になる。

## 結果

- manifest のメタデータが元表記で出力され、RSS/Show Notes として適切な文字列になる。
- 音声合成（synth）は direct 後の変換済みテキストを引き続き使用するため挙動維持。
- direct の LLM 出力が行数に比例して増えるが、呼び出し回数は変わらない。
- 変換欠落行をフォールバックで補うため、セリフの欠落・改変リスクを最小化できる。
- 過去エピソード文脈（cache）が元表記になり、write への文脈提供としても自然になる（副次的改善）。

## 検討した代替案

- **summary 生成時にカタカナを逆変換する**: 変換テーブルの整備が困難で網羅性を保証できない。却下。
- **direct の読み変換を別 LLM 呼び出しにする**: コスト・レイテンシが増加する。同一呼び出しで両方取得する現設計を採用。
- **write に元表記・VOICEVOX 表記の両フィールドを持たせる**: Line モデルの変更が広範囲に及ぶ。direct での変換の方が責務として自然。却下。
