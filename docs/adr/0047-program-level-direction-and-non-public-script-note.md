# 0047. 番組全体の演出指示(direct専用)と台本指示 script_note(write専用・非公開)を追加する

- ステータス: 採用
- 日付: 2026-06-06

## コンテキスト

番組全体・コーナーに効かせたい指示の要望が出た。演出方針(SE/BGM)は direct へ、台本のやり取りの指示(例「記事タイトルを正確に」)は write へ届けたい。

`program.description` は write に渡るが manifest 経由で RSS 説明や Slack にも露出する公開メタデータのため、非公開の台本指示を相乗りできない。コーナー `direction` は direct 専用(ADR 0017)で番組全体向けは無かった。`script direct` は `03_lines.json` のみを読むため、番組演出指示の伝達には中間ファイルへの永続化が要る。

## 決定

- `program.direction`(演出): direct のみに渡す。`03_lines.json`(`model.ScriptLines` ルート直下)へ永続化し `Director.Direct` に渡す(ADR 0017 を番組レベルへ拡張)。
- `script_note`(台本指示): write のみに渡し manifest/feed/Slack には流さない非公開フィールド。番組全体(`program.script_note`)とコーナー個別(`corners[].script_note`)の両レベルに置き、公開される `description` と責務を分離する。

## 結果

- 演出意図は direct に、台本指示は外部公開なしで番組・コーナー両粒度から write に届く。
- `description` と分離したため、将来「統合」と誤って単純化されても非公開要件が壊れにくい(本 ADR が根拠)。
- 破壊的変更: `Director.Direct` シグネチャと `model.ScriptLines` スキーマ。

## 検討した代替案

- **description に相乗り**: RSS/Slack に露出し非公開要件を満たせず却下。
- **direction を ProgramConfig から direct で読む**: file-only 経路と相性が悪く ADR 0017 に反するため却下。
- **演出と台本指示を1フィールドに統合**: 渡る段階と公開可否が異なり混在問題が残るため却下。
