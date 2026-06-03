# 0026. feed 生成ツールを vox-radio の feedgen サブコマンドとして集約する（ADR-0012 の修正）

- ステータス: 採用
- 日付: 2026-06-02

## コンテキスト

ADR-0012 で配信機能を別リポジトリへ分離し、RSS `feed.xml` 生成は broadcast 側の `feedgen` が担う。だが依存は実質「manifest スキーマ」のみで、broadcast は `Manifest` 複製・`episodes.json`・ffprobe を維持しており、同期・運用コストが顕在化した。

ADR-0012 は「別バイナリ分離」を責務分離が中途半端として却下したが、本 ADR はこのコストを踏まえ部分修正する。

## 決定

- `feed.xml` 生成を vox-radio の**新サブコマンド `vox-radio feedgen`** へ移管する（移管元名を踏襲、別バイナリでなくサブコマンド）。
- **状態は run の cache（`internal/cache`）を正とし `episodes.json` は持たない**。feedgen は **cache + `distribution.yaml` → feed.xml** の純粋レンダラで、manifest・mp3・ffprobe に依存しない。
- **`bytes`/`duration` は run 時に算出して cache に保存**し、配信側の ffprobe 依存を無くす（ADR-0012 を更新）。
- **cache は全件保持し、detailed 窓外の回は重いフィールド（`corners`/`conversation_notes`）を compact** する（feed 全件と肥大抑制の両立）。
- **feedgen は配信プラットフォーム非依存**（RSS 形式は前提）。音声 URL は `distribution.yaml` のテンプレート（cache 由来の値を置換）で組み立て、`episode.mp3` をリネーム不要で公開、id は `episode_number`。
- ホスティング・配信メタ・CI は broadcast 所有。既存の `model.Manifest`・`mediainfo`・`cache` を再利用し契約を単一情報源化する。

## 結果

- Manifest 複製が解消しスキーマ変更がコンパイル時に検出される。配信側から `episodes.json` と ffprobe が消え、維持物が減る。
- 責務境界を「設定・状態・配信先」に引き直す。ホスティングは broadcast に残り『配信プラットフォームの独立進化』(ADR-0012) は維持される。
- トレードオフ: feed 長・dedup 窓が cache 設定に連動し、`bytes`/`duration` を vox-radio が供給する点で ADR-0012 を更新する。
- broadcast 側は旧 `feedgen`・`episodes.json` 廃止、`distribution.yaml` 改訂、CI 切替が追従作業。

## 検討した代替案

- **`episodes.json` を別持ち**: feed の retention を独立制御できるが配信側の維持ファイルが増えるため却下。
- **契約だけ共有（Manifest 型を pkg 公開）**: 複製は消えるが2リポジトリにまたがり Go モジュール依存（スキュー）が生じるため却下。
- **別バイナリ化 / 現状維持**: 前者はリリース・DL の二重化、後者は同期コスト未解消で却下。

## 補足

補足（#213）: 設定ファイル `distribution.yaml` は `feedgen.yaml` にリネームした。Go シンボルも `DistributionConfig` → `FeedgenConfig`・`LoadDistribution` → `LoadFeedgen` に改名。YAML キー構造は変更なし。
