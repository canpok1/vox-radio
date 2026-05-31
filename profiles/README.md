# profiles/

ジャンル別設定（プロファイル）を格納するディレクトリです。

## ディレクトリ構成

```
profiles/
  tech/
    profile.yaml        # 技術ニュース用プロファイル
    assets/
      bgm/              # BGM ファイル
      se/               # SE ファイル
      jingle/           # ジングルファイル
  test/
    profile.yaml        # 動作確認用の最小プロファイル
    assets/
      ...               # ダミー素材（tech/ と同じファイルを使用）
```

## プロファイルの切り替え

コマンドの `--profile` フラグでプロファイルファイルのパスを指定します。

```bash
# 技術ニュース用プロファイルで実行
vox-radio collect --out work/articles.json --profile profiles/tech/profile.yaml

# 動作確認用プロファイルで実行（デフォルト）
vox-radio collect --out work/articles.json
```

## 新しいジャンルの追加

1. `profiles/<genre>/` ディレクトリを作成する
2. `profile.yaml` を作成する（既存プロファイルをコピーして編集）
3. `assets/` に音声素材を配置する

## profile.yaml のスキーマ

```yaml
program:
  title: "番組タイトル"
  description: "番組の説明"
  language: ja
  author: vox-radio
  category: News
  explicit: false
  cover_image_url: https://example.com/cover.jpg
  site_url: https://example.com/
  max_items: 7            # フィードに載せる最大件数
  segment_pause_sec: 0.3  # セリフ間の無音（秒）

corners:                  # 固定コーナーのリスト
  - title: "オープニング"
    content: "番組の挨拶と本日のトピック紹介"
    cast:
      zundamon: "元気に挨拶する進行役"  # キャラID: そのコーナーの役割指示
    target_chars: 200      # 目標文字数（暫定）
    # source なし → 収集スキップ（挨拶のみのコーナーに使用）
  - title: "ニュースコーナー"
    content: "テック記事を紹介"
    cast:
      zundamon: "司会"
      metan: "解説役"
    target_chars: 1300
    source:                # このコーナーのデータソース（省略可）
      feeds:
        - url: https://example.com/rss.xml
          max_items: 5
      articles:
        - https://example.com/articles/123

assets:
  jingle:
    opening: { file: assets/jingle/opening.mp3, fade_in: 0.5, fade_out: 0.5 }
    ending:  { file: assets/jingle/ending.mp3,  fade_in: 0.5, fade_out: 1.0 }
  se:
    chime:      { file: assets/se/chime.wav,      volume: 0.8 }
    transition: { file: assets/se/transition.wav, volume: 0.8 }
  bgm:
    talk_bgm: { file: assets/bgm/talk.mp3, volume: 0.3, duck_ratio: 8, loop: true }
```

### フィールド説明

- `program`: 番組全体の設定（旧 `podcast` + `show.segment_pause_sec`）
- `corners`: 固定コーナーのリスト（旧 `show` を再設計）
  - `cast`: キャラID→役割指示のマップ（`vox-radio.yaml` の `characters` のキーを参照）
  - `target_chars`: 暫定の目標文字数（#4 で再生時間化予定）
  - `source`（省略可）: コーナーのデータソース。省略したコーナーは収集をスキップ
    - `feeds`: RSS フィードのリスト（`url` / `max_items`）
    - `articles`: 個別記事 URL のリスト
- キャラIDは `vox-radio.yaml` の `characters` セクションで定義したIDと一致させること

`assets.*/file` のパスは、このプロファイルファイルが置かれているディレクトリからの相対パスで解決されます。

## ダミー素材について

`assets/` に含まれる素材（mp3/wav）はダミーの無音ファイルです。
本番運用では各自の素材に差し替えてください。

> **注意:** 音声素材の著作権・ライセンスに従って素材を利用してください。
