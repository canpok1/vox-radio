# 0011. 設定スキーマを全面再編し、キャラカタログ・番組(program)・コーナー(corners)を導入する

- ステータス: 採用
- 日付: 2026-05-31

## コンテキスト

ADR-0010 で設定を `vox-radio.yaml`（共通=LLM）と `profile`（ジャンル別）に二分割したが、次の課題が残った。

- VOICEVOX URL が環境変数 `VOICEVOX_ENGINE_URL` にハードコードされ設定で管理できない。
- キャラ設定が `show.speakers`（role→speaker_id）のみで、特徴や複数スタイル（ノーマル/なみだめ等）を表現できない。
- `vox-radio.yaml` が `--config` 必須で煩雑。
- `show` の責務が曖昧で、コーナーを LLM が毎回動的生成しており構成を宣言的に固定できない。
- 尺が文字数（`target_chars`）指定で直感的でない。
- `feeds`/`articles` の役割が曖昧で、コーナーごとの使い分けができない。

## 決定

設定スキーマを全面再編する。

```yaml
# vox-radio.yaml（config側）
voicevox: { url: http://localhost:50021 }     # 環境変数を置き換え
characters:                                   # キャラカタログ（キー=キャラID）
  zundamon: { name: ずんだもん, pronoun: ボク, personality: [...],
              default_style: ノーマル, styles: { ノーマル: 3, なみだめ: 76 } }

# profile.yaml（profile側）
program: { title, ..., target_duration_sec: 300, segment_pause_sec: 0.3 }  # 旧 podcast
corners:                                       # 旧 show を再設計（固定構造）
  - { title: オープニング, content: ..., target_duration_sec: 30,
      cast: { zundamon: 進行役 } }              # source は任意
  - { title: 今日のニュース, ..., cast: {...},
      source: { feeds: [...], articles: [...] } }  # データソースはコーナーが持つ
```

- `--config` を廃止し固定パス `vox-radio.yaml` を自動読込（`script`/`synth`/`run`）。
- キャラカタログを config 側に集約（特徴＋スタイル名→speaker_id＋`default_style`）。
- `podcast`→`program` にリネームし、番組全体の `target_duration_sec`/`segment_pause_sec` を集約。
- `show`→`corners`（固定コーナーのリスト）。役割は `corners[].cast`（キャラID→指示）に集約し、`default_speaker`/`persona`/旧`speakers` を廃止。
- 尺は再生時間ベース。台本生成時は文字数係数（≈7文字/秒）で概算換算する。
- `source` はコーナーが持ち、`collect` はコーナー単位で収集・帰属させる。
- コーナー固定化に伴い `plan` ステップを廃止。`speaker_role` の値は host/guest からキャラIDへ変更。

## 結果

キャラの特徴・複数スタイルを表現でき、番組構成を宣言的に固定でき、尺を直感的に指定でき、コーナー別データソースを扱える。`--config` 不要で簡素になる。一方、`model.ShowConfig` 廃止・パイプライン改修・テスト/ドキュメント刷新を伴う破壊的変更が広範に及ぶ。コーナー固定化で日替わりの構成自由度は下がり、尺は文字数概算のため精度は粗い（実測補正は将来課題）。影響が大きいため、領域ごとに 5 件の Issue へ分割し依存順に段階移行する。

## 検討した代替案

- **キャラ設定を profile 側に置く**: ジャンル非依存のカタログは config が自然なため却下。
- **コーナーを LLM 動的生成のまま維持**: 構成を宣言的に固定・明示できず却下。
- **尺を文字数のまま維持**: 設定が直感的でないため却下（内部換算で既存の文字数ロジックは再利用）。
- **URL を環境変数で管理継続**: 設定の一元化方針に反するため却下。
- **一括 1 Issue で実装**: 巨大でレビュー困難・中間状態が壊れやすいため、依存順の 5 分割を採用。
