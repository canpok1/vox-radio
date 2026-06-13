# 0063. init の出力先を --output-dir に一本化し --sample の暗黙 sample/ 出力を廃止する（ADR-0041 改訂）

- ステータス: 採用
- 日付: 2026-06-13

## コンテキスト

`vox-radio init` の出力先が一貫していない。

- フラグなし `init` → カレントディレクトリに生成。
- `init --sample`（ADR-0041 で導入）→ 暗黙的に `sample/` ディレクトリへ生成。

`--sample` は本来「テンプレート内容（記入済みサンプル）の選択」を意図したフラグだが、同時に出力先まで暗黙で変える二重の役割を持っていた。利用者からは「どこに生成されるか」がフラグから読み取りづらく、出力先を明示的に指定する手段もなかった。

## 決定

出力先は `--output-dir` フラグ（デフォルト カレントディレクトリ）で一元的に決定し、`--sample` はテンプレート内容の選択のみに役割を絞る。

| コマンド | 出力先 |
|---|---|
| `init` | `./` |
| `init --output-dir foo` | `foo/` |
| `init --sample` | `./`（★ADR-0041 から変更） |
| `init --sample --output-dir sample` | `sample/`（旧 `init --sample` の再現） |

- `--output-dir` は string・shorthand なし（`--config`/`--log-dir`/`--spec` に倣う）。空文字はカレントへフォールバックする。
- 出力先ディレクトリは生成時に自動作成する（既存の `writeFile` の `os.MkdirAll` に委ねる）。
- これにより ADR-0041 の「`--sample` → `sample/`」という出力先挙動を改める（破壊的変更）。ADR-0041 の他の決定（バイナリ同梱・examples 廃止・音声非同梱）は維持する。

## 結果

- 出力先の決め方が「常に `--output-dir`（デフォルト カレント）」に統一され、`--sample` は内容選択のみの直交した役割になる。
- 破壊的変更: 旧 `init --sample`（→ `sample/`）に依存していた手順は `init --sample --output-dir sample` への書き換えが必要。`Makefile`（`run-sample` / `check-samples`）・README・`docs/development/`・各 `Long` 例を更新する。
- `--output-dir` で任意のディレクトリへ生成できるようになり、複数番組ディレクトリの初期化などが容易になる。

## 検討した代替案

- **override 方式（`--output-dir` 指定時のみ上書き、未指定時は `--sample`→`sample/` を維持）**: 後方互換は保てるが、`--sample` が出力先を暗黙で変える一貫性のなさが残るため却下。
- **親ディレクトリ結合方式（`init --sample --output-dir foo` → `foo/sample/`）**: `sample/` サブディレクトリが常に付き、出力先が直感的に読めないため却下。
