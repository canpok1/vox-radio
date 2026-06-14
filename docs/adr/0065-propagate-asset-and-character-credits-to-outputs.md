# 0065. 使用したアセット・キャラクターのクレジットを manifest 経由で feed/Slack へ転記する

- ステータス: 採用
- 日付: 2026-06-14

## コンテキスト

サンプルアセットに OtoLogic（CC BY 4.0・クレジット必須）等を同梱し、キャラクターの声（VOICEVOX 話者）にも利用規約がある。現状クレジットは `feed-spec.yaml` の `feed.credit` を手書きして `<itunes:author>` に出すだけで、利用者が規約を認識していないと表記が漏れ、規約違反になりうる。使用した素材・キャラから自動でクレジットを集めて出力したい。

実際に使った素材・キャラは中間成果物から特定できる（SE は `script.json` の `AssetName`、ジングル/BGM は `03_lines.json` の `CornerAudio`、キャラは `manifest.Casts` の `CharacterID`）。

## 決定

アセット各エントリ（jingle/se/bgm）と `CharacterConfig` に任意の `credit`（自由文字列）を追加し、**その回で実際に使われた**素材・キャラの credit を収集・重複排除して `manifest` の `credits`（文字列リスト）へ集約する。出力は次のとおり。

- **feed**: エピソード説明欄（`<description>`）末尾へ自動追記する。説明欄はテンプレートでなく自動生成のため、自動が唯一の差し込み手段。
- **Slack**: `{credit}` プレースホルダを追加し、テンプレートに置いた場合のみ出力する（任意）。
- **manifest**: `credits` を常に保持し、feed/Slack 双方の供給源とする。

既存の `feed.credit`（`<itunes:author>`）は配信者表記として残す。`credit` 未設定なら何も出力しない（後方互換）。

## 結果

- 素材・キャラを使うだけでクレジットが manifest と feed に乗り、規約遵守の取りこぼしを減らせる。feed は自動で担保される一方、Slack はテンプレートへの `{credit}` 配置が必要で取りこぼしの余地が残る。
- VOICEVOX 等の声のクレジットも `CharacterConfig.credit` で一元管理でき、手書き運用への依存が減る。
- 中間成果物→manifest→出力の配線にクレジット収集が一段増える。

## 検討した代替案

- **全出力で自動追記（Slack も保証）**: 取りこぼしは無くなるが Slack 文面の自由度が下がるため、feed=自動・Slack=任意とした。
- **手書き `feed.credit` のみ継続**: 利用者の認識に依存し漏れるため不採用。
- **アセットとキャラで credits を分離**: 出力は同じ列挙のため、重複排除しやすい単一リストに統合した。
