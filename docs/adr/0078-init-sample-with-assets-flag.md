# 0078. init --sample-with-assets で音入りサンプルを生成する

- ステータス: 採用
- 日付: 2026-06-16

## コンテキスト

ADR-0064 でサンプル音源パック（sample-assets）を GitHub Release で配布する方針を採った。しかし `init --sample` のサンプルは音声アセットを参照せず（assets.yaml は記入例、コーナー割り当てもコメントアウト）、パックを展開しても割り当てを手書きする必要があり「すぐ音入りで試す」導線が途切れていた。

## 決定

パック前提で音割り当て済みの設定を生成する `init --sample-with-assets` を追加する。

- 生成物は `vox-radio.yaml` と割り当て済み `episode-spec.yaml`（＋ feed/slack）。`assets/assets.yaml` は生成せずパック展開に委ねる。
- 共通ファイルは `templates-sample` を再利用し `episode-spec.yaml` のみオーバーレイする。`--sample` とは排他。

## 結果

- パック展開 → 本フラグ → `episodegen` で音入り番組をすぐ作れる。
- assets.yaml の二重管理が無く ADR-0064 と整合する。
- トレードオフ: パック未展開だと assets.yaml が無く `check` 失敗（想定挙動。ヘルプ/README で明示）。

## 検討した代替案

- **`--sample` にデフォルト導入**: パック未展開での `check` 失敗を全利用者に強いるため却下。
- **go:embed で assets.yaml 生成**: ADR-0064 の非埋め込み方針に反し二重管理のため却下。
- **補助フラグ `--with-assets` 構成**: 単一フラグの方が意図が明確なため却下。
