# アセット設定 YAML リファレンス

> **検証の正**: 設定が正しいかは下記の検証コマンドの結果で判断してください。本ドキュメントと実際の挙動が食い違う場合は、スキルとバイナリの版ずれが原因のことがあります。SKILL.md の「バージョン整合チェック」に従ってスキル / バイナリを揃えてください。

> **検証コマンド**: `vox-radio assets check <パス>`

アセット設定ファイルはジングル・SE・BGM を定義した YAML ファイルです。`episode-spec.yaml` の `assets_files` フィールドで参照します。

アセット設定ファイルのトップレベルには `jingle:` / `se:` / `bgm:` を記述します。

音声アセットは `script.json` のセグメント型として統一的に表現されます。各セグメントは `type` フィールドで種別を指定し、`asset_name` フィールドで対応するマップのキーを参照します。

| セグメント種別 | `type` 値 | 再生方式 | 説明 |
|---|---|---|---|
| 音声（ナレーション） | `speech` | serial | メイン音声。複数同時不可 |
| 効果音 | `se` | serial（既定）/ overlay（`overlay: true` 指定時） | 既定は SE が鳴り終わってから次のセリフを再生（順次）。`overlay: true` を設定すると音声に重ねて再生 |
| BGM | `bgm` | overlay | 音声の裏で再生。排他（停止→切替）。`asset_name` 空 = 停止 |
| ジングル | `jingle` | serial | 単独再生（音声・BGMと重ならない）。前後に pause が入る |

ジングルはラン境界として機能します: 台本がジングルで区切られ、各ラン内の SE/BGM はそのランにのみ適用されます（ジングルをまたいで継続しません）。

## `jingle` マップ値

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `file` | string | 必須 | 音声ファイルパス |
| `fade_in` | float64 | 任意 | フェードイン時間（秒）。デフォルト: 0 |
| `fade_out` | float64 | 任意 | フェードアウト時間（秒）。デフォルト: 0 |
| `trim_silence` | bool | 任意 | 前後の無音を自動除去するかどうか。デフォルト: true |
| `trim_silence_threshold` | float64 | 任意 | 無音判定の振幅閾値（dB、負値のみ）。デフォルト: -50。素材のノイズフロアに合わせて調整 |
| `description` | string | 任意 | アセットの説明（「何の音か・いつ使うか」）。LLM が挿入タイミングを判断する際の手がかりになる |
| `credit` | string | 任意 | 素材のクレジット表記（例: `OtoLogic https://otologic.jp / CC BY 4.0`）。設定すると manifest の `credits` へ自動収集され、feed の `<description>` と Slack の `{credit}` プレースホルダへ転記される |

ジングルおよびコーナー境界 SE はコーナー毎に `corners[].start_audio` / `corners[].end_audio`（`type: jingle` または `type: se`）で設定します。script 生成ステップでコードがコーナー本編の前後へ確定的に挿入するため、生成された `04_script.json` にジングル/SEセグメントが含まれます。`type: jingle` は BGM を停止してから再生し、`type: se` は BGM を継続したまま BGM の下で再生します。BGM も `corners[].bgm` で同様にコーナー単位で管理します。

## `se` マップ値

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `file` | string | 必須 | 音声ファイルパス |
| `volume` | float64 | 任意 | 音量倍率。デフォルト: 0（Go ゼロ値） |
| `trim_silence` | bool | 任意 | 前後の無音を自動除去するかどうか。デフォルト: true |
| `trim_silence_threshold` | float64 | 任意 | 無音判定の振幅閾値（dB、負値のみ）。デフォルト: -50。素材のノイズフロアに合わせて調整 |
| `overlay` | bool | 任意 | `true` = 音声に重ねて再生（従来の overlay 動作）。`false` または省略 = SE が鳴り終わってから次のセリフを再生（順次）。デフォルト: false |
| `description` | string | 任意 | アセットの説明（「何の音か・いつ使うか」）。LLM が挿入タイミングを判断する際の手がかりになる |
| `credit` | string | 任意 | 素材のクレジット表記。jingle と同様に manifest へ自動収集される |

## `bgm` マップ値

| フィールド | 型 | 必須/任意 | 説明 |
|---|---|---|---|
| `file` | string | 必須 | 音声ファイルパス |
| `volume` | float64 | 任意 | 音量倍率。デフォルト: 0（Go ゼロ値） |
| `duck_ratio` | float64 | 任意 | セリフ再生中の音量低減比率（サイドチェインコンプ）。デフォルト: 0 |
| `loop` | bool | 任意 | ループ再生するかどうか。デフォルト: false |
| `loop_gap_sec` | float64 | 任意 | ループの各繰り返しの間に挿入する無音秒数。デフォルト: 0（無音なし）。`loop: true` のときのみ有効 |
| `fade_in` | float64 | 任意 | BGM 開始時のフェードイン秒数。省略時は 1.0 秒。`0` を指定するとフェードなし |
| `fade_out` | float64 | 任意 | BGM 終了時のフェードアウト秒数。省略時は 1.0 秒。`0` を指定するとフェードなし |
| `trim_silence` | bool | 任意 | 前後の無音を自動除去するかどうか。デフォルト: true。ループの継ぎ目に挟まる無音を取り除き、ループを自然に繋ぐ |
| `trim_silence_threshold` | float64 | 任意 | 無音判定の振幅閾値（dB、負値のみ）。デフォルト: -50。素材のノイズフロアに合わせて調整 |
| `description` | string | 任意 | アセットの説明（「何の音か・いつ使うか」）。LLM が挿入タイミングを判断する際の手がかりになる |
| `credit` | string | 任意 | 素材のクレジット表記。jingle と同様に manifest へ自動収集される |

BGM の開始・停止は台本の `bgm` セグメントで制御します。`asset_name` にキー名を指定するとその BGM を開始し、空文字列を指定すると停止します。

同一ラン内で BGM が別の BGM に切り替わる場合、前の BGM がフェードアウトしつつ次の BGM がフェードインするクロスフェードが自動で適用されます（重なり幅 = `min(prevFadeOut, nextFadeIn)`）。ジングル境界または BGM 明示停止時も `fade_out` 秒でフェードアウトします。

ループ再生（`loop: true`）では、デフォルトで前後の無音が `trim_silence` により除去され、各繰り返しが隙間なく繋がります。意図的に繰り返しの間へ「間」を入れたい場合は `loop_gap_sec` に秒数を指定します。`trim_silence: false` かつ `loop_gap_sec: 0` のときは素材ファイルをそのまま繰り返す従来動作になります。
