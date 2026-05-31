# [C] 演出プロンプト（アセット挿入位置の判定）

以下のセリフ列と使用可能なアセット一覧を元に、SE（効果音）・BGM・中間ジングル（アイキャッチ）を挿入する位置を判定してください。

OP/EDジングルはscript生成時にコードが台本へ埋め込み済みのため、ここでは中間アイキャッチ用ジングル・SEの挿入位置とBGMの開始/停止のみを判定してください。

## セリフ列

```json
{{lines}}
```

## 使用可能なアセット一覧

```json
{{asset_catalog}}
```

アセット一覧のフィールド:
- `se`: 効果音エントリのリスト（overlay再生・複数同時可）
- `bgm`: BGMエントリのリスト（overlay再生・排他・ループ可）
- `jingle`: ジングルエントリのリスト（serial再生・単独・前後にポーズ）

各エントリのフィールド:
- `name`: アセットキー名（挿入時に `asset_name` へ指定する値）
- `description`: アセットの説明（省略可）。何の音か・どんな場面で使うかを記述。**挿入タイミングを判断する際はこの説明を参考にすること**

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
  ],
  "pause_insertions": [
    {
      "after_line_index": 4,
      "duration_sec": 1.0,
      "reason": "オチの前の溜め"
    }
  ]
}
```

## 各アセットタイプの挿入ルール

### 間（pause）
- オチの前の溜め、しんみりした余韻など、**意図的な空白**が演出として必要な場合のみ使用する
- `after_line_index` のセリフの直後に指定秒数の無音が入る（BGMは途切れず継続）
- `duration_sec` は **0.3〜2.0秒** を目安とし、最大5.0秒まで指定可能
- 過剰な「間」は dead air になるため、**全体で0〜2回まで**を目安に控えめに使う
- 通常のトーク進行・セリフの切れ目には使わない（デフォルトのポーズで十分）

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
- OP/EDのジングルはscript生成時にコードが台本へ埋め込み済みなのでここには含めない

## 注意事項

- `after_line_index` は0始まりのインデックスです（0 = 最初のセリフの後に挿入）
- `asset_name` は対応するアセット一覧のキー名を使用してください（BGM停止は空文字列）
- 挿入が不要な場合は `insertions` / `pause_insertions` をそれぞれ空配列にしてください
- 同一 `after_line_index` に SE と pause が両方ある場合、SE → pause の順で再生されます
