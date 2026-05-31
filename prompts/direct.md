# [C] 演出プロンプト（アセット挿入位置の判定）

以下のセリフ列と使用可能なアセット一覧を元に、SE（効果音）・BGM・中間ジングル（アイキャッチ）を挿入する位置を判定してください。

OP/EDジングルはコードが自動挿入するため、ここでは中間アイキャッチ用ジングル・SEの挿入位置とBGMの開始/停止のみを判定してください。

## セリフ列

```json
{{lines}}
```

## 使用可能なアセット一覧

```json
{{asset_catalog}}
```

アセット一覧のフィールド:
- `se`: 効果音のキー名リスト（overlay再生・複数同時可）
- `bgm`: BGMのキー名リスト（overlay再生・排他・ループ可）
- `jingle`: ジングルのキー名リスト（serial再生・単独・前後にポーズ）

## 出力形式

以下のJSON形式で回答してください。

```json
{
  "insertions": [
    {
      "after_line_index": 0,
      "type": "se",
      "asset_name": "chime",
      "reason": "コーナー開始のため"
    },
    {
      "after_line_index": 0,
      "type": "bgm",
      "asset_name": "talk_bgm",
      "reason": "BGM開始"
    },
    {
      "after_line_index": 3,
      "type": "bgm",
      "asset_name": "",
      "reason": "BGM停止"
    },
    {
      "after_line_index": 5,
      "type": "jingle",
      "asset_name": "eyecatch",
      "reason": "コーナー区切り"
    }
  ]
}
```

## 各アセットタイプの挿入ルール

### SE（効果音）
- トピックの転換点など、効果的な場所にのみ挿入する
- 使いすぎると煩わしくなるので、厳選して挿入する（全体で0〜3回程度）
- `after_line_index` のセリフの直後に再生される（overlayのため本編と同時）

### BGM
- 番組の雰囲気に合わせて開始・停止を制御する
- BGMを開始したい場合: `asset_name` にBGMキー名を指定
- BGMを停止したい場合: `asset_name` を空文字列 `""` で指定
- 同時に複数のBGMは再生できない（切替時は自動停止）

### Jingle（中間アイキャッチ）
- コーナーの区切りなど、明確な転換点に挿入する
- ジングルは単独再生（音声・BGMと重ならない）
- ジングルの前後に自動的にポーズ（無音）が挿入される
- OP/EDのジングルはコードが挿入するのでここには含めない

## 注意事項

- `after_line_index` は0始まりのインデックスです（0 = 最初のセリフの後に挿入）
- `asset_name` は対応するアセット一覧のキー名を使用してください（BGM停止は空文字列）
- 挿入が不要な場合は `insertions` を空配列にしてください
