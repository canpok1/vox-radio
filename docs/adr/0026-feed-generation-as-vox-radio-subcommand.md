# 0026. feed 生成ツールを vox-radio の feedgen サブコマンドとして集約する（ADR-0012 の修正）

- ステータス: 採用
- 日付: 2026-06-02

## コンテキスト

ADR-0012 で配信機能を別リポジトリへ分離し、RSS `feed.xml` 生成は broadcast 側の `feedgen` が担う。だが `feedgen` の vox-radio 依存は実質「manifest スキーマ」のみで、broadcast は `Manifest` を複製しており、拡張のたびの手動同期コストが顕在化した。

ADR-0012 は「同一リポジトリ内の別バイナリ分離」案を「配信方式の独立進化ができず責務分離が中途半端」として却下したが、本 ADR はこの同期コストを踏まえ部分修正する。

## 決定

- `feed.xml` 生成（manifest → feed.xml 変換）を broadcast から vox-radio の**新サブコマンド `vox-radio feedgen`** へ移管する（移管元名を踏襲し `feed` 単独の曖昧さを避ける）。
- **`distribution.yaml`・`episodes.json`・ホスティング・CI は引き続き broadcast が所有**する。
- 形態は**別バイナリでなくサブコマンド**（goreleaser・install スクリプト変更不要、配信側 CI は同一バイナリを使い回せる）。
- CLI 入力は **`manifest.json` と `distribution.yaml` の2ファイル**に削減し、`episodes.json`・`public` パスは `distribution.yaml` に記述する。
- **feedgen はホスティング方式に依存しない**: 音声 URL は `distribution.yaml` のテンプレート（manifest 由来の値を置換）で組み立て、構築規約をコードに持たない。run の `episode.mp3` をリネームせず公開でき、id は manifest の `episode_number` を正とする。
- vox-radio 既存の `internal/model.Manifest`・`internal/mediainfo` を再利用し契約を単一情報源化する。

## 結果

- manifest 構造体の複製が解消し、スキーマ変更が生産者・消費者の同居でコンパイル時に検出される。
- 責務境界を「バイナリ」でなく「設定・状態・配信先」に引き直す。配信方式の本体は broadcast に残るため『配信方式の独立進化』(ADR-0012) は維持される。
- トレードオフ: RSS/iTunes 形式を持ち込む点は ADR-0012 と緊張するが、RSS は ADR-0005 で一本化済みで安定し、将来変われば `feedgen` を廃止できる。
- broadcast 側は旧 `feedgen` 廃止・CI の `vox-radio feedgen` 切替が追従作業として必要。

## 検討した代替案

- **契約だけ共有（Manifest 型を pkg 公開し broadcast が import）**: 複製は消えるがツールが2リポジトリにまたがり、Go モジュール依存（バージョンスキュー）が生じるため却下。
- **現状維持（二重定義の手動同期を継続）**: 顕在化した同期コストを解消できないため却下。
- **別バイナリ化**: リリース資産・DL が二重化し配信側の取得処理変更が必要で、分離効果も限定的なため却下。
