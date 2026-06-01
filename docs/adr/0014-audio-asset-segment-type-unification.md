# 0014. 音声アセット（SE/BGM/Jingle）を script.json のセグメント型に統一し、ジングルをラン単位の serial 再生に変更する

- ステータス: 採用
- 日付: 2026-05-31

## コンテキスト

OP/ED ジングルを `filter.go` の専用 `amix` ブロックとして実装し、設定キーをコードへハードコードしていた（`Jingle["op"]`/`["ed"]`）。YAML では `opening`/`ending` キーを使っているため両者が食い違い**サイレント無効化**が発生していた。BGM も設定レベルで常時適用され台本から ON/OFF 不可。SE は `se_name` フィールドを使い他アセットと統一性がなかった。加えて、任意位置でのジングル挿入（中間アイキャッチ等）にも対応できていなかった。

## 決定

- `ScriptSegment.SEName` を廃止し `AssetName` に統一。SE/BGM/Jingle すべて `asset_name` で参照する。
- `SegmentType` に `bgm`/`jingle` を追加。`type=bgm` + `asset_name` 空 = BGM 停止。
- `filter.go` をラン単位合成へ全面再構成。ジングルセグメントをラン境界とし `[ラン][pause][jingle][pause][ラン]` を concat で直列結合後、`loudnorm` を全体に 1 回適用。
- OP/ED ジングルはコードが `program.opening_jingle`/`ending_jingle` に基づき script 生成ステップで `script.json` の先頭/末尾へ埋め込む。中間アイキャッチは LLM が `insertions` の `type=jingle` で配置。
  ※ 注記: 本ADR記載の `opening_jingle`/`ending_jingle` はその後 `start_jingle`/`end_jingle` にリネームした（Issue #154）。
- `Director.Direct` を `SECatalog` → `AssetCatalog`（SE/BGM/Jingle キー一覧）に変更し LLM への入力も拡張。

## 結果

サイレント無効化バグを根本解消。ジングルの任意位置挿入が位置非依存で処理される。BGM の台本レベル制御（開始・停止・切替）が可能になる。`loudnorm` が全体 1 回となり BGM ポンピングも解消。

breaking change あり：既存 `script.json` の `se_name` キーを `asset_name` に移行要。プロファイルの jingle/bgm 設定構造は変わらないが、`program.opening_jingle`/`ending_jingle` によるキー指定は引き続き使用可能。なお Issue #131 により OP/ED ジングルの注入タイミングが assemble ステップから script 生成ステップへ移動し、生成済み `04_script.json` に OP/ED ジングルが含まれるようになった。

## 検討した代替案

**A. キー名の一致のみ修正**（`op`→`opening` 等）: 最小修正だが任意位置ジングル要件を満たせないため却下。

**B. OP/ED を BuildContext フィールドとして維持し BGM/Jingle のみ Script 制御**: 部分統一にとどまりアーキテクチャが複雑化するため却下。

## 後続の変更

ADR 0020 にて BGM/中間ジングルの LLM 配置と program OP/ED 指定を廃止し、コーナー毎 profile.yaml 設定駆動へ移行した（本 ADR の「決定事項3: provenance」を部分的に上書き）。
