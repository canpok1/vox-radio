# 0026. feed 生成ツールを vox-radio の feedgen サブコマンドとして集約する（ADR-0012 の修正）

- ステータス: 採用
- 日付: 2026-06-02

## コンテキスト

ADR-0012 で配信機能を別リポジトリ（vox-radio-broadcast）へ分離し、RSS `feed.xml` 生成は broadcast 側の `feedgen` が担っている。だが `feedgen` の vox-radio 依存は実質「manifest スキーマ」のみで、broadcast は `Manifest` 構造体を複製しており、拡張のたびに手動同期するコストが顕在化した。

ADR-0012 は「同一リポジトリ内の別バイナリ分離」案を「配信方式の独立進化ができず責務分離が中途半端」として却下していたが、本 ADR は当時予見しなかったこの同期コストを踏まえ部分的に修正する。

## 決定

- `feed.xml` 生成（「manifest → feed.xml」変換）を broadcast から vox-radio の**新サブコマンド `vox-radio feedgen`** へ移管する（移管元ツール名を踏襲し `feed` 単独の曖昧さを避ける）。
- **`distribution.yaml`・`episodes.json`・ホスティング・CI は引き続き broadcast が所有**する。
- 形態は**別バイナリでなくサブコマンド**（goreleaser・install スクリプト変更不要、配信側 CI は同一バイナリを使い回せる）。
- CLI 入力は **`manifest.json` と `distribution.yaml` の2ファイル**に削減する。audio は manifest から導出、`episodes.json`・`public` パスは `distribution.yaml` に記述する。
- vox-radio 既存の `internal/model.Manifest`・`internal/mediainfo` を再利用し契約を単一情報源化する。

## 結果

- manifest 構造体の複製が解消し、スキーマ変更が生産者・消費者の同居でコンパイル時に検出される。
- 責務境界を「バイナリ」でなく「設定・状態・配信先」に引き直す。配信方式の本体は broadcast に残るため ADR-0012 が守った「配信方式の独立進化」は維持される。
- トレードオフ: RSS/iTunes という**フィード形式**を持ち込む点は ADR-0012 と若干緊張するが、RSS は ADR-0005 で一本化済みで形式が安定し、将来変われば `feedgen` を廃止できる。
- broadcast 側は旧 `feedgen` 廃止・`distribution.yaml` 拡張・CI の `vox-radio feedgen` 呼び出し化が追従作業として必要。

## 検討した代替案

- **契約だけ共有（Manifest 型を pkg 公開し broadcast が import）**: 複製は消えるが、ツールが2リポジトリにまたがり Go モジュール依存（バージョンスキュー）も生じるため却下。
- **現状維持（二重定義の手動同期を継続）**: 顕在化した同期コストを解消できないため却下。
- **別バイナリ化**: リリース資産・DL が二重化し配信側の取得処理に変更が必要。同一モジュールでは分離効果も限定的なため却下。
