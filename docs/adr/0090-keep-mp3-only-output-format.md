# 0090. 出力音声形式を mp3 固定とし他形式（AAC/Opus 等）対応を見送る

- ステータス: 採用
- 日付: 2026-06-21

## コンテキスト

エピソードの最終出力は現在 mp3 に固定されている。番組設定から出力音声形式をユーザーが選べるようにしたい、という要望が出たため対応可否を調査した。

当初は「ffmpeg に出力形式パラメータを渡すフィールドを番組設定に1つ追加するだけ」で済むと見込んだが、調査の結果、出力形式は単一パラメータでは閉じず、パイプライン横断で複数箇所の連動が必要だと判明した。mp3 を前提にハードコードされている主な箇所は次のとおり。

- コーデック: `internal/mix/filter.go` の `-c:a libmp3lame`
- 品質指定: `internal/mix/filter.go` の `audioQualityArgs`（`-q:a 0/2/5`）は libmp3lame の VBR 専用。AAC/Opus は品質指定の流儀（`-b:a` 等）が異なる
- メタデータ: `internal/mix/filter.go` の `-id3v2_version 3` ＋ ID3 タグは mp3 専用。M4A(MP4) や Opus はメタデータ方式が別系統で、非 mp3 では無視/エラーになる（ID3 タグは [ADR-0066](0066-id3-tags-for-generated-mp3.md)）
- 出力ファイル名: `internal/fileio/paths.go` の拡張子 `.mp3`（命名は [ADR-0067](0067-descriptive-episode-mp3-filename.md)）
- 配信 MIME: `internal/feed/feed.go` の RSS enclosure `type="audio/mpeg"`
- Slack 投稿（拡張子依存）・`internal/mix/preview.go` のプレビュー（別途 mp3 固定）

加えて、配信は Podcast(RSS) に一本化しており（[ADR-0005](0005-podcast-rss-only-distribution.md)）、podcast クライアントで広く再生できるのは実質 mp3 と AAC で、Opus 等は対応が限定的という制約もある。

## 決定

当面、出力音声形式は **mp3 固定を維持し、他形式（AAC/Opus 等）への対応は見送る**。現時点で他形式への要望は小さく、上記の横断的な実装・テスト・ドキュメント改修に見合わないと判断した。

将来対応する場合の方針も併せて記録しておく。

- 「生の ffmpeg パラメータを passthrough するフィールド」は採用しない。拡張子・MIME・メタデータ方式が自動連動せず破綻しやすいため。
- 代わりに、拡張子・コーデック・品質マッピング・メタデータ方式・配信 MIME を一括で決定するキュレートされた列挙型（例: `output_format: mp3 | aac`）として設計する。既存の `audio_quality: high/standard/low` の抽象は維持し、形式ごとに品質値マッピングを足す。

これは [ADR-0070](0070-keep-ffmpeg-over-mp3-alternatives.md)（mp3 エンコード代替を見送り ffmpeg 依存を維持）と整合する、出力形式側の現状維持判断である。

## 結果

- 実装・テスト・ドキュメントを mp3 前提のまま単純に保てる。
- 他形式を求めるユースケースには応えられないが、現時点の需要は小さい。
- 将来要望が増えた場合は本 ADR を改訂し、上記方針に沿って `output_format` を設計する。本 ADR に波及箇所を記録済みのため、再調査コストは小さい。

## 検討した代替案

- **生 ffmpeg パラメータの passthrough フィールド**: 柔軟だが、出力拡張子・RSS の MIME 型・メタデータ方式（ID3/MP4/Vorbis）が自動で連動せず、不整合な組み合わせで実行時に破綻しやすい。設定としても利用者に ffmpeg 内部を晒すことになり扱いづらい。却下。
- **キュレート列挙型 `output_format`（mp3/aac 等）を今すぐ導入**: 安全な方向だが、形式ごとの品質マッピング・メタデータ分岐・MIME 連動の実装が必要で、現時点の需要に見合わない。将来案として保留。
