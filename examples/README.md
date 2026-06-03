# examples/

エピソード仕様（episode-spec.yaml）のサンプルを格納するディレクトリです。

仕様ファイルはジャンルごとに `<genre>.yaml` として直下に配置し、音声素材（`assets/`）はジャンル間で共通利用できるよう直下に1つだけ置く構成です。

## ディレクトリ構成

```
examples/
  tech.yaml             # 技術ニュース用エピソード仕様
  assets/               # 全ジャンル共通の音声素材
    bgm/                # BGM ファイル
    se/                 # SE ファイル
    jingle/             # ジングルファイル
```

## 仕様ファイルの切り替え

コマンドの `--spec` フラグで仕様ファイルのパスを指定します（必須）。

```bash
# 技術ニュース用仕様で実行
vox-radio episodegen collect --out work/articles.json --spec examples/tech.yaml
```

## 新しいジャンルの追加

1. `examples/<genre>.yaml` を作成する（既存仕様をコピーして編集）
2. 必要に応じて共通の `assets/` に音声素材を追加する（既存素材を流用する場合は不要）

## episode-spec.yaml のスキーマ

```yaml
program:
  title: "番組タイトル"
  description: "番組の説明"
  segment_pause_sec: 0.3   # セリフ間の無音（秒）
  length_sec: 240 # 番組全体の目標再生時間（秒）

corners:                  # 固定コーナーのリスト
  - title: "オープニング"
    content: "番組の挨拶と本日のトピック紹介"
    direction: "冒頭でオープニングジングルを流す。"  # 演出説明（省略可）
    cast:
      zundamon: "元気に挨拶する進行役"  # キャラID: そのコーナーの役割指示
    length_sec: 30  # コーナーの目標再生時間（秒）
  - title: "ニュースコーナー"
    content: "テック記事を紹介"
    cast:
      zundamon: "司会"
      metan: "解説役"
    length_sec: 180
    source:                # このコーナーのデータソース（省略可）
      feeds:
        - url: https://example.com/rss.xml
          max_items: 5
      articles:
        - https://example.com/articles/123

assets_files:
  - assets/assets.yaml
```

## ダミー素材について

`assets/` に含まれる素材（mp3/wav）はダミーの無音ファイルです。
本番運用では各自の素材に差し替えてください。

> **注意:** 音声素材の著作権・ライセンスに従って素材を利用してください。
