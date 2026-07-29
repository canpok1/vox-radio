# [D] 番組分析プロンプト

以下のラジオ番組の1回分の台本・機械集計指標を読み、この回の課題（findings）と会話の型（patterns）を抽出してください。

## 番組情報

```json
{{program}}
```

## コーナー（採用順）

```json
{{corners}}
```

## セリフ

各要素は `{"corner_id": "<コーナーID>", "speaker": "<speaker_role>", "text": "<セリフ本文>"}` の形式です。

```json
{{lines}}
```

## 機械集計指標

コーナーごとの目標文字数（`target_chars`）・実際に書かれた文字数（`actual_chars`）・セリフ数（`line_count`）、
実測尺の内訳（`actual_length_sec` = `speech_length_sec`（セリフのみ）+ `non_speech_length_sec`（ジングル・SE・ポーズ））、
実効発話レート（`chars_per_sec` = `actual_chars` ÷ `speech_length_sec`。Go側で計算済み）、
話者別のセリフ数・文字数、校正での修正件数（`proofread_corrections`）です。

尺の乖離を指摘する際は、`target_chars` と `actual_chars` の差（指示追従の問題）と、`chars_per_sec`
（発話速度の見積もりの問題）を区別してください。演出音（ジングル・SE・ポーズ）による超過は
`non_speech_length_sec` に現れるため、セリフの書きすぎだと誤って指摘しないでください。

```json
{{metrics}}
```

## 出力形式

以下のJSON形式で回答してください。

```json
{
  "findings": [
    {"aspect": "掛け合い", "severity": "high", "detail": "記事の要点を並べる説明に終始している", "evidence": "ずんだもん「〜という記事なのだ」"}
  ],
  "patterns": [
    {"aspect": "つかみ", "detail": "天気の雑談から入った"}
  ]
}
```

## findings の注意事項

- 「もっと面白く」のような一般論を禁止する。**具体的にどのセリフの何が問題か**を書く
- `evidence` に該当セリフを引用する（創作しない）
- `severity` は `high` / `medium` / `low` のいずれかにする
- 該当が無ければ空配列にする。最大5件まで（優先度の高いものから）

## patterns の注意事項

- 良し悪しを判定せず、**今回実際に使った構造を事実として**短く記録する
- 例: aspect「つかみ」/ detail「天気の雑談から入った」、aspect「掛け合い」/ detail「片方が記事を読み上げもう片方が驚く形を3コーナーとも使った」
- 他の回との比較はしない（この回の入力しか与えられていないため）
- 該当が無ければ空配列にする。最大5件まで
