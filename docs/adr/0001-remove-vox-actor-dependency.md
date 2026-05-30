# 0001. vox-actor 依存の除去

- ステータス: 採用
- 日付: 2026-05-30

## コンテキスト

vox-radio の音声合成（③ synth）は当初、VOICEVOX エンジンの CLI ラッパーである vox-actor バイナリ（`vox-actor say --save-wav`）を呼び出す設計だった。しかし vox-actor の Linux バイナリは ALSA に動的リンクしており、WAV 保存のみでも `libasound2`（`libasound2t64`）が必要になる。このため runner イメージや GitHub Actions に音声ライブラリの導入が必要となり、外部 CLI ツールへの依存とセットアップの複雑さを抱えていた。VOICEVOX エンジン自体は今後も使い続ける方針である。

## 決定

音声合成を、vox-actor バイナリ経由ではなく **VOICEVOX エンジンの HTTP API を Go から直接呼び出す**方式に変更する。各セリフは `POST /audio_query` で `AudioQuery` を取得し、話速・音高・抑揚などを反映してから `POST /synthesis` で WAV を取得する。話者・話し方は show.yaml からコードでマッピングする。VOICEVOX エンジン（Docker）は維持する。

## 結果

- 外部 CLI ツール（vox-actor）と ALSA/libasound2 への依存が消え、コンテナ・CI 構成が簡素化された。
- HTTP クライアントのみで完結し、再生用ネイティブライブラリが不要になった。
- 声質・話者 ID・話し方パラメータは従来どおり扱える。
- 一方、`AudioQuery` の組み立てやリトライなど、これまで vox-actor が吸収していた処理を自前実装する必要がある。

## 検討した代替案

- **汎用 TTS プロバイダ抽象に置換**: VOICEVOX をやめ Synthesizer インターフェースで複数 TTS を差し替え可能にする案。柔軟だが設計変更が大きく、現状 VOICEVOX を使い続けるため過剰と判断し却下。
- **vox-actor を別コンテナ化して維持**: 依存は隔離できるが、CLI ツールへの依存自体は残り目的に反するため却下。
