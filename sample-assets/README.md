# vox-radio サンプルアセット

vox-radio で番組にジングル・効果音（SE）・BGM を入れて試すためのサンプル音源パックです。自分で音源を用意しなくても、すぐに音入りの番組を作れます。

## 同梱物

```
assets.yaml      # 各音源を登録済みのアセット設定（音量・フェード等は設定済み）
jingle/          # オープニング/エンディング ジングル
se/              # 効果音（アクセント・シャキーン・シーン切り替え）
bgm/             # カフェ風BGM
README.md
CREDITS.md       # 各音源の提供元・ライセンス
```

各音源の用途は `assets.yaml` 内のコメントを参照してください。

## 使い方

vox-radio は番組設定（`assets/assets.yaml`）からアセットを読み込みます。このパックを番組の `assets/` ディレクトリに展開して使います。

```bash
unzip vox-radio-sample-assets.zip -d assets
vox-radio assets check assets/assets.yaml
```

次に、番組設定（`episode-spec.yaml`）でこのアセットを使うよう参照し、各コーナーに割り当てます。

- `assets_files` に `assets/assets.yaml` を登録する。
- 各コーナーの開始/終了に鳴らすジングル・効果音や、コーナー中に流す BGM を割り当てる。設定済みのIDは `assets.yaml` を参照（ジングル: `opening`/`ending`、効果音: `accent_low`/`onoma_syakiin`/`switch`、BGM: `coffee_break`）。

設定フィールドの詳細は vox-radio スキルのリファレンスを参照してください（アセット定義は `references/assets.md`、コーナーへの割り当ては `references/episode-spec.md`）。

## ライセンス

同梱音源の提供元・ライセンスは [CREDITS.md](CREDITS.md) を参照してください。利用の際は各提供元の利用規約に従ってください。
