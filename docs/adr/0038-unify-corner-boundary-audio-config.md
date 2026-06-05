# 0038. コーナー境界音声を start_jingle/end_jingle から type 付き start_audio/end_audio へ再構成する

- ステータス: 採用
- 日付: 2026-06-05

## コンテキスト

ADR 0020 でジングルはコーナー設定駆動（`start_jingle`/`end_jingle`）になったが、SE は LLM 挿入のみでコーナー境界へ確定配置する手段がない。jingle と SE の本質差は「BGM と共存できるか」の 1 点に集約される。jingle は run 境界として BGM を停止し、SE は run 内で BGM が下に流れ続ける。その他はアセット定義側のオプション差にすぎない。

## 決定

**`start_jingle`/`end_jingle` を廃止し、`type`（`jingle` | `se`）と `id`（assets キー）を持つ `start_audio`/`end_audio` に統一する。**

```yaml
corners:
  - title: "今日のテックニュース"
    start_audio:
      type: jingle
      id: opening
    end_audio:
      type: se
      id: page_turn
    bgm: coffee_break
```

- 形式はスカラー（境界ごとに 1 音声）。`type` → `SegmentType`、`id` → `AssetName` にマップする。
- `type: jingle` は従来どおり挿入後に BGM 開始。`type: se` は BGM 開始後に挿入し BGM の下で再生する（activeBGM を維持）。終了側は対称。
- `bgm` フィールドは連続背景音として独立維持する。
- 旧フィールドは即時廃止（alias なし）。strict 検証では未知フィールドとして検出される。
- `ValidateEpisodeSpecAssets` で `type` の enum と `id` の存在を検証する。
- ADR 0020 の `start_jingle`/`end_jingle` 部分を本 ADR が置換する。`filter.go` は変更しない（SE 処理は ADR 0025 で実装済み）。

## 結果

境界音声の表現が 1 フィールドに統一され、jingle/SE の選択が `type` で明示的になる。assemble 層は変更不要で、実装は config → CornerLines → buildScript → バリデーションの配線に収まる。破壊的変更のため既存 YAML・テンプレート・examples の修正が必要で、旧フォーマットの `03_lines.json` キャッシュは境界音声が無言で脱落するため再生成が必要。リスト形式へはカスタム UnmarshalYAML で非破壊拡張できる。

## 検討した代替案

**`start_se`/`end_se` を追加**: 本質差が 1 点しかないのに API が型別に分裂し、同時指定時の順序仕様も必要。却下。

**最初からリスト形式**: BGM 開始位置を「最後の jingle の直後」とするルールが必要で複雑化。スカラーから非破壊拡張できるため見送り。

**prefix 付き文字列（`"jingle:opening"`）**: 独自記法のパースが必要で YAML として自己記述的でない。却下。

**旧フィールドを alias 維持**: コードパスが 2 系統になり廃止の 2 段階目も必要。利用者が作者のみのため即時廃止。
