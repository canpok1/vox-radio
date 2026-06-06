# 0042. VOICEVOX URL に環境変数オーバーライドを導入する

- ステータス: 採用
- 日付: 2026-06-06

## コンテキスト

ADR 0011 で VOICEVOX URL を環境変数 `VOICEVOX_ENGINE_URL` から設定ファイル（`voicevox.url`）管理へ移行した。しかし devcontainer では VOICEVOX が別コンテナ（サービス名 `voicevox`）で動くため、`vox-radio init --sample` が生成するサンプル設定のデフォルト `http://localhost:50021` のままでは到達できず、テンプレートのコメントで手動変更を案内していた。サンプル設定そのままで音声生成できる体験を実現するには、設定ファイルを書き換えずに実行環境側で接続先を切り替える仕組みが必要になった（Issue #309）。

## 決定

環境変数 `VOX_RADIO_VOICEVOX_URL` による任意オーバーライドを導入する。優先順位は 環境変数 > `voicevox.url` > デフォルト `http://localhost:50021` とする。解決ロジックは既存の `Effective*()` パターンに合わせ `VoicevoxConfig.EffectiveURL()` に集約し、devcontainer の compose で `dev` サービスに `http://voicevox:50021` を注入する。ADR 0011 の「設定ファイル管理」は維持し、環境変数は実行環境差分を吸収する上書き手段と位置づける（部分改訂）。

## 結果

- devcontainer でサンプル設定そのまま音声生成でき、テンプレートでの手動変更案内が不要になる。
- synth / episodegen に重複していた URL フォールバックが 1 箇所に集約され、デフォルト値も定数化される。
- 環境変数が設定ファイルより優先されるため、devcontainer 内で `voicevox.url` を書き換えても効かない。テンプレートのコメントで優先順位を明示して混乱を防ぐ。

## 検討した代替案

- **socat サイドカーで localhost:50021 を voicevox へ中継**: コード変更ゼロだが、devcontainer 構成が複雑化し転送プロセスという暗黙の依存が増えるため不採用。
- **テンプレートのデフォルトを `http://voicevox:50021` に変更**: devcontainer では動くが、ホストで直接使うユーザーが逆に壊れるため不採用。
- **環境変数のみへ回帰（ADR 0011 以前の方式）**: 設定の一元管理が失われ ADR 0011 の課題が再発するため不採用。上書き手段としてのみ導入する。
