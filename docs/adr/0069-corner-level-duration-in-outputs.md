# 0069. コーナー単位の尺（実測）を成果物に追加し corner_id をパイプライン全体に伝播させる

- ステータス: 採用
- 日付: 2026-06-14

## コンテキスト

台本のコーナーごとの分量・実尺を調査して `length_sec`（目標尺）や `chars_per_minute` をチューニングする際、コーナー単位の尺データが成果物に無い。`clips.json` は clip ごとに `duration_sec`（実測発話秒）を持つがどのコーナーの clip かが不明で、`manifest.json` の `corners[]` は尺・文字数を持たない。`04_script.json` の segments はコーナー境界（jingle / scene_change SE）でしか判別できず、コーナー対応が線形化で失われている。このため利用者は中間ファイルを手で突き合わせてコーナー別尺を集計している。

`config.CornerConfig.ID` 由来のコーナーIDは存在するが（ADR-0052）、`model.CornerLines` 以降の成果物には伝播していない。

## 決定

コーナー単位の尺を episodegen の成果物に標準出力する。

- **corner_id をパイプライン全体に伝播させる**: `CornerConfig.ID` → `CornerLines.ID` → `ScriptSegment.CornerID` → `ClipMeta.CornerID`。既存の StartAudio 等の transfer パターンに準拠し、各中間ファイル（03_lines.json / 04_script.json / clips.json）に永続化する。
- **manifest の `corners[]` に4フィールドを追加する**（すべて additive・後方互換）:
  - `target_sec`（目標尺 = `LengthSec`）
  - `speech_sec`（発話実測秒 = コーナーの clip `duration_sec` 合計）
  - `duration_sec`（コーナー全体再生尺 = SE/BGM/pause/jingle/crossfade 込み）
  - `char_count`（元表記セリフの文字数合計）
- **コーナー全体尺は assemble のタイムライン計算を再利用する**: assemble が既に持つ `durationMs` 累積（`collectRuns`）と同じ計算で各 segment の尺寄与を `corner_id` 別に合計し、独立に再実装して数値が乖離しないようにする。
- **コーナー別尺を新規中間生成物 `06_timeline.json` に永続化する**（ADR-0004 のファイルベース中間生成物方針に準拠）。assemble は corner 別尺を戻り値で返し、呼び出し側（pipeline / `cli/assemble`）が他の中間ファイルと同様に書き出す。`manifest.Build` は clips（speech_sec）・06_timeline（duration_sec）・03_lines（char_count）・spec（target_sec）から各フィールドを `corner_id` キーで集計する。

## 結果

- **良い面**: コーナー別の目標尺・発話実測秒・全体尺・文字数が manifest 一発で取得でき、`length_sec` / `chars_per_minute` チューニングの突き合わせ作業が不要になる。発話実測秒は目標尺と同じ「発話予算」軸で比較でき、全体尺は実際の再生尺を表す。
- **悪い面**: corner_id 用フィールドを複数 model・中間ファイルに追加する。新規中間生成物 `06_timeline.json` が増え、`Assembler` インターフェースの戻り値が変わる。
- **トレードオフ**: 全体尺は assemble の既存タイムライン計算を再利用するため、独自集計による数値乖離リスクを避けつつ実装コストを抑える。

## 検討した代替案

- **発話実測秒のみ追加（全体尺なし）**: clips.json 集計だけで済み軽量だが、SE/BGM/pause を含む実再生尺が分からない。両方を別フィールドで持つ方針を採用。
- **コーナー全体尺を script + アセット尺から独立に再計算**: assemble を経由せず算出する案。crossfade/overlay/ducking/loudnorm を正確に再現できず assemble の実尺と乖離するため却下。
- **corner index（整数）で対応付け**: ID ではなく順序で突き合わせる案。順序依存で脆く、manifest（ID キー）との整合性も悪いため却下し、`corner_id`（文字列）を採用。
- **集計サマリを別ファイル/CLI で出力**: manifest を汚さない案。manifest corners に項目追加する方が一発で取得でき新規成果物も増えないため、サマリ専用出力は今回スコープ外（必要時に別タスク）。
