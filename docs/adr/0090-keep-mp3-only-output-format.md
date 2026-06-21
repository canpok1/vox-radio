# 0090. 出力音声形式を mp3 固定とし他形式（AAC/Opus 等）対応を見送る

- ステータス: 採用
- 日付: 2026-06-21

## コンテキスト

エピソードの最終出力は mp3 に固定されている。番組設定から出力音声形式を選べるようにしたい要望が出たため、対応可否を調査した。

当初は「ffmpeg に出力形式パラメータを渡すフィールドを1つ追加するだけ」で済むと見込んだが、出力形式は単一パラメータでは閉じず、パイプライン横断で複数箇所の連動が必要だと判明した。mp3 を前提にハードコードされている主な箇所は次のとおり。

| 箇所 | mp3 前提の内容 |
|---|---|
| `internal/mix/filter.go` | コーデック `-c:a libmp3lame` / 品質 `-q:a`（libmp3lame の VBR 専用、AAC・Opus は `-b:a` 等で流儀が異なる）/ `-id3v2_version 3` ＋ ID3 タグ（mp3 専用。M4A・Opus は別系統で非 mp3 では無視/エラー。[ADR-0066](0066-id3-tags-for-generated-mp3.md)） |
| `internal/fileio/paths.go` | 出力ファイル名の拡張子 `.mp3`（[ADR-0067](0067-descriptive-episode-mp3-filename.md)） |
| `internal/feed/feed.go` | RSS enclosure `type="audio/mpeg"`（配信 MIME） |
| `internal/mix/preview.go`・Slack 投稿 | プレビューが別途 mp3 固定／投稿は拡張子依存 |

加えて配信は Podcast(RSS) に一本化しており（[ADR-0005](0005-podcast-rss-only-distribution.md)）、podcast クライアントで広く再生できるのは実質 mp3 と AAC で、Opus 等は対応が限定的という制約もある。

## 決定

当面、出力音声形式は mp3 固定を維持し、他形式（AAC/Opus 等）への対応は見送る。現時点で要望は小さく、上記の横断的な実装・テスト・ドキュメント改修に見合わないと判断した。

将来対応する場合も、生の ffmpeg パラメータを渡すのではなく、拡張子・コーデック・品質マッピング・メタデータ方式・配信 MIME を一括決定するキュレート列挙型（例: `output_format: mp3 | aac`）で設計する。`audio_quality: high/standard/low` の抽象は維持し、形式ごとに品質値を割り当てる。本判断は [ADR-0070](0070-keep-ffmpeg-over-mp3-alternatives.md) と整合する、出力形式側の現状維持である。

## 結果

- 実装・テスト・ドキュメントを mp3 前提のまま単純に保てる。
- 他形式を求めるユースケースには応えられないが、現時点の需要は小さい。
- 将来要望が増えたら本 ADR を改訂し、上記方針で `output_format` を設計する。波及箇所を記録済みのため再調査コストは小さい。

## 検討した代替案

- **生 ffmpeg パラメータの passthrough フィールド**: 柔軟だが、出力拡張子・RSS の MIME 型・メタデータ方式（ID3/MP4/Vorbis）が自動連動せず、不整合な組み合わせで実行時に破綻しやすい。利用者に ffmpeg 内部を晒す点でも扱いづらい。却下。
- **キュレート列挙型 `output_format` を今すぐ導入**: 安全な方向だが、形式ごとの品質マッピング・メタデータ分岐・MIME 連動の実装が必要で、現時点の需要に見合わない。将来案として保留。
