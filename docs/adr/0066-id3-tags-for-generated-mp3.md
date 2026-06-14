# 0066. 生成 MP3 に ID3 タグを設定し、タイトル取得のため pipeline を並べ替える

- ステータス: 採用
- 日付: 2026-06-14

## コンテキスト

生成する MP3 には ID3 タグ（メタ情報）が一切設定されておらず、ffmpeg 出力は `-c:a libmp3lame -q:a 2` のみだった。ポッドキャストプレイヤーでの表示・整理を改善するため、番組名・回番号・サブタイトル等のエピソードメタ情報をタグとして埋め込みたい。

ここで配線上の制約がある。MP3 を生成する `assemble` ステップは pipeline 上、サマリー生成より**前**に実行される。タイトルに使いたいサブタイトル（`EpisodeTitle`）はサマリー生成の成果物のため、assemble 時点では未確定である（回番号 `EpisodeNumber` と収録日 `GeneratedAt` は `opts` から事前取得できる）。

## 決定

ffmpeg の汎用 `-metadata` キー（ID3v2 フレームへ自動マッピング）で以下を設定する。空値はタグごと省略し、互換性のため `-id3v2_version 3` を付ける。

| タグ | 値 |
|---|---|
| アルバム(TALB) | `program.title` |
| タイトル(TIT2) | 第N回 ＋ サブタイトル（下記） |
| アーティスト(TPE1) | `program.author`（新設フィールド） |
| トラック(TRCK) | `EpisodeNumber` |
| 日付(TDRC) | `GeneratedAt` を `program.timezone` で `YYYY-MM-DD` 整形 |

コメント・ジャンルは今回対象外とする。タイトル組み立ては RSS の item title と一致させるため、`feed.itemTitle` と共通のヘルパー（`model.EpisodeDisplayTitle`）へ抽出して両者から使う。

サブタイトル取得のため **pipeline を並べ替え**、サマリー生成（と `generatedAt` 確定）を `assemble` の前へ移す。サマリーは確定済みの `scriptLines` のみに依存し assemble 出力に依存しないため、並べ替えは安全である。

## 結果

- タグはアセンブル時の単一 ffmpeg 呼び出しで付与でき、追加パスや再 mux が不要。
- タイトル表記が RSS とコードレベルで一致し、表示の食い違いを防げる。
- 副作用として、サマリー生成失敗時は MP3 生成より前に失敗する（fail-fast）。無駄な音声生成を避けられる一方、失敗の発生位置が前倒しになる。
- 単体 `episodegen assemble` コマンドはエピソード情報を持たないため album/artist のみ設定され、エピソード固有タグ（title/track/date）は省略される。

## 検討した代替案

- **assemble 後にタグ付け専用ステップを追加（ffmpeg 再 mux）**: pipeline 並べ替えは不要だが、ffmpeg が 2 パスになり MP3 生成者が 2 か所に分かれる。実行コストと責務分散の点で却下。
- **タイトルにサブタイトルを含めない（回番号のみ）**: 並べ替え不要だが、プレイヤー表示の情報量が下がるため却下。
