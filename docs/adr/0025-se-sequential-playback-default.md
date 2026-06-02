# 0025. SE の既定再生方式を順次再生（serial）に変更し per-SE overlay フラグを追加する

- ステータス: 採用
- 日付: 2026-06-02

## コンテキスト

ADR 0014 でSE は "overlay"（セリフに重ねて再生）として設計された。`script.json` に `se` セグメントが出現すると `adelay + amix` でセリフトラックに重ねられ、SE は直後のセリフと並行再生される。

しかしトランジション音・句読点的な効果音として使う場合、「SE が鳴り終わってから次のセリフが始まる」方が自然なケースが多い。現状ではその表現ができず、SE の後ろに明示的な `pause` セグメントを挿入して擬似的に対応するしかなかった。また SE ファイルの実長さに合わせて pause の秒数を手動で合わせる必要があり、`trim_silence` の影響を受けて端数が生じる問題もあった。

## 決定

SE の既定再生方式を **順次再生（serial）** に変更し、`SEEntry` に `overlay *bool` フラグを追加する。

- `overlay` を省略または `false` → SE を concat チェーンの 1 パーツとして差し込み（順次）。
- `overlay: true` → 従来の `adelay + amix`（重ね再生）。

順次 SE の実長は `Assembler.Run` が ffprobe で取得し `BuildContext.SEDurations` に格納する。`collectRuns` は順次 SE の分だけ `durationMs` を進め、後続の overlay SE や BGM の offset を正しく算出する。

## 結果

- 既定で「SE が鳴り終わってから次のセリフ」という自然な挙動が実現し、LLM が pause を手動調整しなくて済む。
- `trim_silence` 有効時は ffprobe の生ファイル長 > 実再生長のため、同一ラン内の後続 BGM/overlay SE の offset がわずかに後ろへずれる（許容値として設計上コメントで明記）。
- `overlay: true` を明示することで従来の重ね再生も引き続き利用可能。
- 既存プロファイルの SE は `overlay` 省略のため全て順次再生に変わる（破壊的変更）。既存の SE をオーバーレイで使いたい場合は `overlay: true` を明示する必要がある。

## 検討した代替案

**無音ギャップ挿入方式**: SE の後に `anullsrc`（無音）を挿入して擬似的に待機する。SE の実長を静的に算出しなければならず、`trim_silence` 適用後のずれが残るため不採用。

**concat 連結方式（採用）**: SE を通常の concat パーツとして差し込む。`silenceremove` 後の実長そのままで連結されるため、余分な空白が出ず正確な順次再生が実現できる。
