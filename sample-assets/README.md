# vox-radio サンプルアセット

vox-radio で番組にジングル・効果音（SE）・BGM を入れて試すためのサンプル音源パックです。自分で音源を用意しなくても、すぐに音入りの番組を作れます。

## 同梱物

```
assets.yaml      # 各音源を登録済みのアセット設定（音量・フェード等は設定済み）
jingle/          # 番組テーマジングル
se/              # 効果音（アクセント・シャキーン・シーン切り替え）
bgm/             # カフェ風BGM
README.md
CREDITS.md       # 各音源の提供元・ライセンス
```

各音源の用途は `assets.yaml` 内のコメントを参照してください。

## 使い方

このパックを、番組の設定ファイルがあるディレクトリで `assets/` に展開して使います。

```bash
unzip vox-radio-sample-assets.zip -d assets
vox-radio assets check assets/assets.yaml
```

次に、エピソード設定（`episode-spec.yaml`）でこのアセット設定を参照し（`assets_files` に `assets/assets.yaml` を登録）、各コーナーに割り当てます。設定済みのIDは次のとおりです（詳細は `assets.yaml` のコメントを参照）。

- ジングル: `theme`
- 効果音: `accent` / `syakiin` / `switch`
- BGM: `coffee_break`

設定フィールドの詳細は vox-radio スキルのリファレンスを参照してください（アセット定義は `references/assets.md`、コーナーへの割り当ては `references/episode-spec.md`）。

## ライセンス

同梱音源の提供元・ライセンスは [CREDITS.md](CREDITS.md) を参照してください。利用の際は各提供元の利用規約に従ってください。
