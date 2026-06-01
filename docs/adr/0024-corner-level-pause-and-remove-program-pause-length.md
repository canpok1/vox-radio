# 0024. コーナーごとの開始/終了無音設定と program.segment_pause_sec / program.length_sec の削除

- ステータス: 採用
- 日付: 2026-06-01

## コンテキスト

`program.segment_pause_sec` はセリフ間・ラン間すべてに一律に適用される無音時間として ADR-0011 で導入された。しかし実際の要件はコーナーごとに異なる「コーナー開始前/終了後の無音」であり、一律設定では表現できない。加えて `program.length_sec`（番組全体の目標収録時間）は本番コードで参照されておらず、コーナーごとの `length_sec` と二重管理になっていた。ADR-0020 の方針「番組構成として事前決定できる要素はコーナーごとに決定的に持たせる」に沿って整理する必要があった。

## 決定

- `program.segment_pause_sec` を削除し、セリフ間・ラン間の無音は固定値 `defaultPauseSec = 0.3` を常に使用する。
- `program.length_sec` を削除する（コーナーごとの `corners[].length_sec` のみ残す）。
- `corners[]` に `start_pause_sec` / `end_pause_sec` を追加する（`omitempty`、デフォルト 0 = 挿入しない）。
- `buildScript` でコーナーの最外側に pause セグメントを注入する（`start_pause → start_jingle → ... → end_jingle → end_pause` の順）。
- `model.CornerLines` にも同フィールドを追加し `03_lines.json` へ永続化する（jingle/bgm と同じ決定的配線）。

## 結果

**良い点:** コーナーごとに開始/終了の無音を細かく制御できる。設定が一元化され整合性チェック不要になった。`defaultPauseSec` 固定でセリフ間挙動が単純化された。  
**トレードオフ:** `segment_pause_sec` を設定していた既存プロファイルは strict モードでエラーになるため手動マイグレーションが必要。

## 検討した代替案

**`program.segment_pause_sec` を残しコーナー設定と共存させる案:** `program` 側が優先か `corners` 側が優先かの競合ロジックが発生し、テストが複雑になる。ADR-0011 の一部見直しとして削除を選択した。  
**`start_pause_sec` をLLM判断にする案:** pause は番組構成として事前決定できる要素であり、ADR-0020 の方針（決定的設定駆動）に反するため却下。
