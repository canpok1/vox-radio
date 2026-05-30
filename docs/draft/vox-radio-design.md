# vox-radio 設計ドキュメント

> ⚠️ **ステータス: ドラフト（未実装の設計案）**
> このドキュメントは構想段階の設計案です。まだ実装されていません。実装が固まったら正式なドキュメントへ昇格させる想定です。

**VOICEVOX エンジン（HTTP API）**を活用して、約5分のラジオ番組を毎日1回自動生成し、**Podcast（RSS）として配信**する仕組み。
音声合成は VOICEVOX エンジンの HTTP API（`/audio_query` → `/synthesis`）を Go から直接呼び出す（外部 CLI ツールには依存しない）。

## 目的・要件

- インターネットの記事（RSS / URL）を元ネタに、約5分のラジオ番組を**毎日1回**自動生成・配信する
- 配信は **Podcast（RSS フィード）に一本化**する
  - feed.xml と音声ファイルを HTTP で公開するだけのシンプルな構成
  - Podcast アプリ（Apple Podcasts / Spotify / Pocket Casts / Overcast 等）で購読・自動DL・再生
  - **Slack でも聴ける**: Slack 標準の RSS アプリで `/feed subscribe <feed-url>` すると新着回がチャンネルに流れる（Slack 専用の API 実装は不要）
- 過去回は**直近1週間分**だけ残す（feed には直近7件を列挙し、7日より古い音声は削除）
- 台本生成 AI は **Gemini API（軽量モデル）を既定**とし、**OpenAI 互換仕様**に対応することで他プロバイダへ差し替え可能にする
- 軽量モデル前提で、台本生成は**複数の小さなステップに分割**する（1コール=1責務）
- 運用先は **GitHub Actions（cron）と自宅サーバー（docker compose）の両対応**
- BGM・効果音・ジングルを使った**本格的な放送演出**を行う（ダッキング含む）

## 役割分担の基本方針

- **VOICEVOX エンジン（HTTP API）**: 「しゃべり」の音声合成専用（テキスト → WAV）。BGM/SE は扱わない。Go から HTTP で直接呼び出す。
- **ffmpeg**: BGM・効果音・ジングルの重ね合わせ、ダッキング、ラウドネス正規化、エンコード。
- **LLM（多段）**: 元記事から段階的に「演出台本」を組み上げる。各ステップは責務を1つに絞る。
- **配信**: feed.xml（RSS）＋音声ファイルを公開ストレージに置くだけ。視聴側は Podcast アプリ or Slack の RSS アプリ。

## 全体アーキテクチャ

```
毎日1回トリガー（GitHub Actions cron / 自宅サーバーの cron・ofelia）
        │
  ① 情報収集    RSS / 記事URL（feeds.yaml）→ 本文抽出 → articles.json
        ▼
  ② 台本生成（多段LLMパイプライン。詳細は後述）
     [0] 記事要約(map) → [A] テーマ決定 → [B] 台本生成(コーナー単位fan-out) → [C] 演出(ルール中心+SEのみLLM)
        → 演出台本 JSON
        ▼
  ③ 音声合成    speech セグメントを VOICEVOX HTTP API でループ合成
                 → VOICEVOXエンジン(Docker)に /audio_query → /synthesis → clip_000.wav, clip_001.wav, ...
        ▼
  ④ 結合・整形  ffmpeg でタイムライン合成
                 - speech 連結（間に無音挿入）
                 - SE/ジングルを指定オフセットに overlay
                 - BGM をループ＋音量調整＋ダッキング(sidechaincompress)
                 - loudnorm でラウドネス正規化 → episode_YYYY-MM-DD.mp3
        ▼
  ⑤ 公開        1) mp3 を公開ストレージにアップロード
                 2) episodes.json（回の一覧メタdata）を更新
                 3) episodes.json から feed.xml を再生成してアップロード
        ▼
  ⑥ 保持管理    episodes.json を直近7件に絞る → feed には7件だけ列挙
                 → 7日より古い mp3 を削除
        │
        ▼
  視聴側  Podcast アプリで feed.xml を購読 / Slack の RSS アプリで /feed subscribe
```

## リポジトリ構成

```
vox-radio/
├── cmd/vox-radio/main.go        # エントリポイント（CLIサブコマンド）
├── internal/
│   ├── collect/                 # ① 情報収集（RSS/記事本文抽出）
│   ├── script/                  # ② 演出台本の生成（多段パイプライン）
│   │   ├── llm/                 #    OpenAI互換クライアント（共通の呼び出し基盤）
│   │   ├── summarize/           #    [0] 記事要約
│   │   ├── plan/                #    [A] テーマ決定（rundown生成）
│   │   ├── write/               #    [B] 台本生成（コーナー単位fan-out）
│   │   └── direct/              #    [C] 演出（ルール中心＋SE位置のみLLM）
│   ├── synth/                   # ③ 音声合成（VOICEVOX HTTP APIクライアント）
│   ├── assemble/                # ④ ffmpeg合成（BGM/SE/ダッキング/正規化）
│   ├── publish/                 # ⑤ 公開（feed.xml生成・アップロード）＋ ⑥ 保持管理
│   │   └── hosting/             #    公開先抽象（ghpages / s3互換 / local）
│   └── pipeline/                # 全体オーケストレーション
├── assets/
│   ├── bgm/                     # BGM音源（差し替え可・ライセンス要確認）
│   ├── se/                      # 効果音
│   ├── jingle/                  # OP/EDジングル
│   └── cover.jpg                # Podcast のカバー画像（チャンネルアート）
├── config/
│   ├── feeds.yaml               # 収集元RSS/記事URL
│   ├── show.yaml                # 番組設定（キャラ、尺、コーナー構成、演出ルール）
│   ├── assets.yaml              # asset名→ファイル/音量/フェードのマッピング
│   ├── llm.yaml                 # LLM設定（base_url/model/各ステップのパラメータ）
│   └── podcast.yaml             # Podcastチャンネル情報（title/description/author等）
├── prompts/
│   ├── summarize.md             # [0] 記事要約プロンプト
│   ├── plan.md                  # [A] テーマ決定プロンプト
│   ├── write.md                 # [B] 台本生成プロンプト
│   └── direct.md                # [C] 演出（SE位置）プロンプト
├── docker-compose.yml           # voicevox-engine + runner（自宅サーバー/ローカル）
├── Dockerfile                   # runner（vox-radio バイナリ + ffmpeg 同梱）
├── .github/workflows/daily.yml  # GitHub Actions cron
└── README.md
```

## ② 台本生成（多段 LLM パイプライン）

軽量モデル（Gemini Flash 等）前提で、**1コール=1責務・小さい出力スキーマ**に分割する。
各ステップは「専用プロンプト＋専用JSONスキーマ＋検証＋リトライ」を個別に持ち、中間生成物をファイルとして残す。

```
① collect（記事収集）
   ▼
[0] 記事要約（map）  各記事 → 3〜5行の要点。軽量モデルは長文入力で精度が落ちるため前処理で圧縮
   ▼
[A] テーマ決定        要約群＋番組設定 → 番組構成(rundown): コーナー一覧・各トピック・要点・尺配分
   ▼
[B] 台本生成          コーナー単位で fan-out → 各コーナーのセリフ（話し言葉）を生成（並列化可）
   ▼
[C] 演出             ルール中心でコードが組み、SE挿入位置のみLLMに判定させる → 演出台本JSON
   ▼
③ synth → ④ assemble → ⑤ publish
```

### 各ステップの責務・入出力

| ステップ | 責務（1つだけ） | 入力 | 出力 |
|---|---|---|---|
| [0] 要約 | 記事を要点化 | 記事原文1件 | `{summary, points[]}` |
| [A] テーマ決定 | 何を/どの順で話すか決める | 要約群＋番組設定 | `rundown.json`（コーナー配列） |
| [B] 台本生成 | 1コーナー分の話し言葉を書く | 1コーナー＋該当要約＋キャラ設定 | セリフ列 `[{speaker_role, text}]` |
| [C] 演出 | SE挿入位置を決める（その他はコード固定） | 全セリフ＋利用可能SE一覧 | 演出台本JSON（既存フォーマット） |

### 決定事項（軽量モデル前提の割り切り）

- **[0] 記事要約ステップを入れる**: 原文をそのまま渡さず、記事ごとに要約してから [A] に渡す（map-reduce）。
- **[B] はコーナー単位の fan-out**: 番組全体を1コールで書かず、コーナーごとに別コールで書く。1コールあたりの入出力を小さく保ち、並列実行も可能。
- **[C] 演出はルール中心＋SEのみLLM**:
  - **決定的な演出はコードで固定**（OPジングルは必ず先頭・EDは末尾・全体に talk_bgm を敷きダッキング）。
  - **創造的な判断（トピック転換のSE挿入位置）だけLLMに任せる**。LLMの責務を最小化して軽量モデルでも安定させる。
  - 話者ID・話速/音高/抑揚の割り当ても、基本はキャラ設定（show.yaml）からコードで決定する。
- **通し調整パスは入れない**: コーナー間のトーンは、各 [B] に共通のキャラ設定（show.yaml）と rundown を渡すことで揃える。軽量モデルでの追加調整パスは逆効果になりうるため設けない。

### 中間生成物（デバッグ・再実行用）

各ステップの出力はファイルに残し、ステップ単位で再実行できる。

```
work/
├── articles.json     # ① の出力
├── summaries.json    # [0] の出力
├── rundown.json      # [A] の出力
├── lines.json        # [B] の出力（全コーナー結合後のセリフ列）
└── script.json       # [C] の出力（= 演出台本。③ synth の入力）
```

## ②-LLM プロバイダ抽象（OpenAI 互換 1 実装＋設定切替）

ドメイン層のインターフェースは多段パイプラインを表現しつつ、**ワイヤープロトコルは OpenAI 互換に統一**する。

```go
// ドメイン層（pipeline はこの ScriptGenerator しか知らない）
type ScriptGenerator interface {
    Generate(ctx context.Context, articles []Article, show ShowConfig) (Script, error)
}

// 内部のサブステップ（各々が小さな LLM コール。同一クライアントを共有）
type Summarizer interface { Summarize(ctx context.Context, a Article) (Summary, error) }
type Planner    interface { Plan(ctx context.Context, s []Summary, show ShowConfig) (Rundown, error) }
type Writer     interface { Write(ctx context.Context, c Corner, s Summary, show ShowConfig) ([]Line, error) }
type Director   interface { Direct(ctx context.Context, lines []Line, se SECatalog) (Script, error) }

// 共通の LLM 呼び出し基盤（OpenAI 互換）。全サブステップが利用する
type Client interface {
    // JSON Schema を指定して構造化出力を得る（検証＋リトライは実装側）
    Complete(ctx context.Context, req CompletionRequest) (json.RawMessage, error)
}
```

- 具象 `Client` は **OpenAI 互換エンドポイント1実装**。`base_url`・`api_key`・`model` を設定注入。
- プロバイダ切替は列挙（`LLM_PROVIDER`）ではなく、**`base_url`＋`model` の設定差し替え**で行う。

| プロバイダ | base_url | 備考 |
|---|---|---|
| **Gemini（既定）** | `https://generativelanguage.googleapis.com/v1beta/openai/` | 軽量モデル（例 `gemini-2.5-flash`）。APIキーは `GEMINI_API_KEY` |
| OpenAI | `https://api.openai.com/v1` | |
| OpenRouter | `https://openrouter.ai/api/v1` | 1キーで多数モデルを束ねるゲートウェイ |
| Groq / Together / DeepSeek / Mistral | 各社のURL | いずれも OpenAI 互換 |
| ローカル(Ollama / vLLM / LM Studio) | `http://localhost:11434/v1` 等 | オフライン/自宅サーバー向き |

- **退避ルート（注意点）**: Gemini の OpenAI 互換レイヤは beta 扱いで、ネイティブ専用機能（Google検索グラウンディング等）は使えない。
  ただし本システムは ①collect で自前に記事を集めて渡すため、モデル側グラウンディングは不要 → 互換APIで問題ない。
  将来ネイティブ機能が必要になった場合は、`Client` 実装を `geminiNativeGenerator` に差し替えればよい（ドメインの `ScriptGenerator` 境界で吸収）。
- **構造化出力**: `response_format` の `json_schema` / `json_object` を使うが、対応度に各社差があるため
  **「スキーマ検証＋リトライ（自己修復プロンプト）」を必ず併用**する。Go クライアントは `openai/openai-go`（base_url変更可）等。

### 尺（約5分）のコントロール

- 日本語の読み上げは概ね 300〜350 字/分 → 5分なら **約1,500〜1,800字**。
- [A] テーマ決定で各コーナーに目標文字数を配分し、[B] 台本生成でコーナーごとに目標字数を指定。
- 全コーナー結合後に総文字数を測定し、超過/不足なら該当コーナーのみ再生成（多段ゆえ局所修正が容易）。

## ③ 音声合成（synth）

- 演出台本の `speech` セグメントを順に走査し、各セリフを **VOICEVOX エンジンの HTTP API** で
  個別の WAV に合成する（ループ）。外部 CLI ツールは使わず、Go の HTTP クライアントで直接呼び出す。
- 合成は 2 ステップ:
  1. **`POST /audio_query?speaker=<id>&text=<text>`** → 音声合成用クエリ（`AudioQuery` JSON）を取得。
  2. 取得した `AudioQuery` に話速/音高/抑揚を反映してから
     **`POST /synthesis?speaker=<id>`**（body = `AudioQuery` JSON）→ WAV バイナリを取得し `clip_NNN.wav` に保存。
- 話者・話し方のパラメータは `AudioQuery` のフィールドで指定する（show.yaml からコードでマッピング）:

  | 概念 | `AudioQuery` フィールド | 備考 |
  |---|---|---|
  | 話者 | （`speaker` クエリパラメータ） | VOICEVOX の style ID |
  | 話速 | `speedScale` | 1.0 が標準 |
  | 音高 | `pitchScale` | 0.0 が標準 |
  | 抑揚 | `intonationScale` | 1.0 が標準 |
  | 音量 | `volumeScale` | 1.0 が標準 |
  | 前後の無音 | `prePhonemeLength` / `postPhonemeLength` | セリフ間ポーズに利用可 |

- VOICEVOX エンジン URL は `VOICEVOX_ENGINE_URL`（既定 `http://localhost:50021`）。
- **ALSA / libasound2 への依存は不要**（再生はせず WAV をファイル化するだけ。HTTP 経由なのでネイティブ音声ライブラリを要求しない）。
- 合成後、各 clip の尺を `ffprobe` で取得してタイムライン化に渡す。

## ④ 結合・整形（assemble / ffmpeg）

処理順:

1. **各 speech を合成** → `ffprobe` で尺取得 → 各セグメントの開始オフセットを確定（タイムライン化）。
2. **speech 連結**: セリフ間に短い無音（例 0.3s, `anullsrc`）を挟んで `concat`。
3. **SE / ジングル配置**: `adelay=<offset_ms>` で頭出しし、`amix` で重畳。
4. **BGM 区間**: 区間長にループ（`aloop`）→ `volume` で下げ →
   ナレーションを sidechain にして `sidechaincompress` でダッキング。
5. **OP/ED ジングル**: 先頭・末尾に `concat`、`afade` でフェードイン/アウト。
6. **ラウドネス正規化**: `loudnorm`（EBU R128, 例 `I=-16 LUFS, TP=-1.5, LRA=11`）でスマホ視聴に最適化。
7. **エンコード**: mp3（128kbps 程度）で `episode_YYYY-MM-DD.mp3` を出力。
8. **尺を計測**: 最終 mp3 の長さ（`itunes:duration` 用）とバイト数（`enclosure length` 用）を取得。

実装方針: タイムラインを Go 側で組み立て、最終的に 1 本の `filter_complex` を生成して
ffmpeg を 1 回呼ぶ方式にすると堅牢（中間ファイルの取り回しが減る）。

### ダッキング（しゃべり中だけ BGM を自動で下げる）の例

```bash
ffmpeg -i narration.wav -i bgm.wav -filter_complex \
 "[1:a]volume=0.3[bg]; \
  [bg][0:a]sidechaincompress=threshold=0.05:ratio=8:attack=20:release=400[ducked]; \
  [ducked][0:a]amix=inputs=2:duration=first[out]" \
 -map "[out]" mixed.wav
```

## ⑤ 公開（Podcast RSS 配信）

配信は「**音声ファイル**」と「**feed.xml**」を HTTP で公開するだけ。視聴側はそのfeed URLを購読する。

### episodes.json（番組の状態台帳）

feed.xml を毎回組み直すための小さな台帳。公開先（または repo）に保存し、毎回読み書きする。

```jsonc
{
  "episodes": [
    {
      "guid": "episode-2026-05-29",
      "title": "2026-05-29 今日のニュース",
      "description": "本日の話題は新型AIチップとオープンソースの動向です",
      "pub_date": "2026-05-29T21:00:00Z",
      "audio_url": "https://<host>/audio/episode_2026-05-29.mp3",
      "bytes": 5242880,
      "duration": "00:05:12"
    }
    // 直近7件のみ保持（古いものは ⑥ で削除）
  ]
}
```

### feed.xml（RSS 2.0 / iTunes 拡張）

channel 情報は `config/podcast.yaml` から、item は `episodes.json` から生成する。

```xml
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
  <channel>
    <title>今日のテックニュース</title>
    <link>https://&lt;host&gt;/</link>
    <description>毎日5分のニュースラジオ</description>
    <language>ja</language>
    <itunes:author>vox-radio</itunes:author>
    <itunes:image href="https://&lt;host&gt;/cover.jpg"/>
    <itunes:category text="News"/>
    <itunes:explicit>false</itunes:explicit>

    <item>
      <title>2026-05-29 今日のニュース</title>
      <description>本日の話題は新型AIチップとオープンソースの動向です</description>
      <pubDate>Thu, 29 May 2026 21:00:00 GMT</pubDate>
      <guid isPermaLink="false">episode-2026-05-29</guid>
      <enclosure url="https://&lt;host&gt;/audio/episode_2026-05-29.mp3"
                 type="audio/mpeg" length="5242880"/>
      <itunes:duration>00:05:12</itunes:duration>
    </item>
    <!-- 直近7回分の item を並べる -->
  </channel>
</rss>
```

主要フィールド:

| 要素 | 値 | 取得元 |
|---|---|---|
| `enclosure url` | 音声本体の公開URL | アップロード先のURL |
| `enclosure length` | 音声ファイルのバイト数 | ④で計測 |
| `enclosure type` | `audio/mpeg`（mp3） | 固定 |
| `itunes:duration` | 尺（HH:MM:SS） | ④で計測（ffprobe） |
| `guid` | 回の一意キー（重複DL防止） | `episode-<date>` |
| `pubDate` | RFC1123 形式の公開日時 | 実行日時 |

### ホスティング（公開先）— `hosting` 抽象で差し替え可能

```go
type Hosting interface {
    PutAudio(ctx context.Context, name string, r io.Reader) (url string, err error)
    PutFeed(ctx context.Context, feedXML []byte) (url string, err error)
    LoadEpisodes(ctx context.Context) (Episodes, error)
    SaveEpisodes(ctx context.Context, e Episodes) error
    DeleteAudio(ctx context.Context, name string) error
}
```

| 実装 | 公開先 | 向き | 補足 |
|---|---|---|---|
| **ghpages** | `gh-pages` ブランチに audio/・feed.xml・episodes.json | GitHub Actions運用と相性◎・無料 | 容量/帯域十分（1週間で数十MB）。音声で履歴が肥大しないよう orphan ブランチを毎回作り直す運用が安全 |
| **release** | mp3 を Release アセット、feed は Pages | 無料・履歴肥大しない | アセットURLが安定 |
| **s3互換** | Cloudflare R2 / S3 / B2 | 自宅サーバー/Actions両対応 | 無料枠あり。署名URLで限定公開も可 |
| **local** | 自宅 nginx の `/audio` `/feed.xml` | 自宅運用なら自然 | HTTPS公開は Cloudflare Tunnel / Tailscale Funnel が楽（ポート開放不要） |

GitHub Actions 主体なら **ghpages** が最も手離れが良く追加課金ゼロ。

### Slack で聴く

Slack の標準 RSS アプリで購読するだけ（専用実装不要）:

```
/feed subscribe https://<host>/feed.xml
```

新着回が指定チャンネルに自動投稿され、リンクから再生できる。Podcast アプリと併用可能。

### 限定公開（私的Podcast）にしたい場合

- feed.xml と音声の URL のパスに**推測困難なトークン**を含める（プライベートPodcast方式）。
- Pocket Casts / Overcast / Apple Podcasts / Slack RSS は任意の feed URL を登録できるため、
  トークン付き URL を自分だけが知っていれば実質非公開運用になる。

## ⑥ 保持管理（直近1週間）

- `episodes.json` を**直近7件**に絞る → feed.xml には7件だけ列挙される。
- feed から外れた回の音声ファイルを公開先から削除（`DeleteAudio`）。
- 既にダウンロード済みの回は購読者の端末に残るため、聴き逃しは起きにくい。

## 設定ファイル

### config/feeds.yaml（収集元）

```yaml
feeds:
  - url: https://example.com/rss.xml
    max_items: 5
  - url: https://another.example.com/feed
    max_items: 3
articles:               # 個別URL指定も可
  - https://example.com/articles/123
```

### config/show.yaml（番組設定）

```yaml
title_format: "今日のテックニュース {date}"
target_chars: 1700            # 約5分相当（[A]が各コーナーへ配分）
corners: 3                    # コーナー数の目安（[A]への指示）
default_speaker: 3
speakers:                     # 登場キャラ（VOICEVOX の style ID）と役割
  host: 3                     # ずんだもん 等
  guest: 2
persona: |                    # 全[B]コールに渡す共通のキャラ設定（トーン統一用）
  hostは元気で親しみやすい進行役。guestは落ち着いた解説役。
segment_pause_sec: 0.3        # セリフ間の無音
```

### config/assets.yaml（asset 名 → 実体のマッピング）

```yaml
jingle:
  opening:    { file: assets/jingle/opening.mp3, fade_in: 0.5, fade_out: 0.5 }
  ending:     { file: assets/jingle/ending.mp3,  fade_in: 0.5, fade_out: 1.0 }
se:
  chime:      { file: assets/se/chime.wav,        volume: 0.8 }
  transition: { file: assets/se/transition.wav,   volume: 0.8 }
bgm:
  talk_bgm:   { file: assets/bgm/talk.mp3, volume: 0.3, duck_ratio: 8, loop: true }
```

- [C] 演出ステップの LLM には `se`（SE）の抽象名一覧だけを渡し、挿入位置を判定させる。
- `jingle` / `bgm` はコードが決定的に配置するため LLM には渡さない。

### config/llm.yaml（LLM 設定）

```yaml
base_url: https://generativelanguage.googleapis.com/v1beta/openai/
api_key_env: GEMINI_API_KEY   # 環境変数名
model: gemini-2.5-flash       # 軽量モデル既定
temperature: 0.7
max_retries: 3                # スキーマ検証失敗時のリトライ回数
steps:                        # ステップ個別に上書き可能
  summarize: { temperature: 0.2 }
  plan:      { temperature: 0.4 }
  write:     { temperature: 0.8 }
  direct:    { temperature: 0.2 }
```

### config/podcast.yaml（Podcast チャンネル情報）

```yaml
title: "今日のテックニュース"
description: "毎日5分のニュースラジオ"
language: ja
author: vox-radio
category: News
explicit: false
cover_image_url: https://<host>/cover.jpg
site_url: https://<host>/
max_items: 7                  # feed に載せる最大件数（=保持件数）
```

## 実行基盤（両対応）

### docker-compose.yml（自宅サーバー / ローカル）

```yaml
services:
  voicevox:
    image: voicevox/voicevox_engine:cpu-latest
    ports: ["50021:50021"]
  runner:
    build: .                       # vox-radio バイナリ + ffmpeg 同梱イメージ
    environment:
      VOICEVOX_ENGINE_URL: http://voicevox:50021
      GEMINI_API_KEY: ${GEMINI_API_KEY}        # llm.yaml の api_key_env に対応
      LLM_BASE_URL: ${LLM_BASE_URL:-}          # 省略時は llm.yaml を使用
      LLM_MODEL: ${LLM_MODEL:-}                # 省略時は llm.yaml を使用
      HOSTING: s3                              # ghpages / release / s3 / local
      S3_ENDPOINT: ${S3_ENDPOINT}
      S3_BUCKET: ${S3_BUCKET}
      S3_ACCESS_KEY: ${S3_ACCESS_KEY}
      S3_SECRET_KEY: ${S3_SECRET_KEY}
      PUBLIC_BASE_URL: ${PUBLIC_BASE_URL}      # feed/enclosure に使う公開URLの基点
    depends_on: [voicevox]
    # 毎日のキックは cron / ofelia から `vox-radio run` を実行
```

### .github/workflows/daily.yml（GitHub Actions cron / ghpages 公開）

```yaml
name: daily-radio
on:
  schedule:
    - cron: "0 21 * * *"   # UTC。JST 06:00 配信なら 21:00 UTC
  workflow_dispatch: {}
permissions:
  contents: write          # gh-pages への push 用
jobs:
  broadcast:
    runs-on: ubuntu-latest
    services:
      voicevox:
        image: voicevox/voicevox_engine:cpu-latest
        ports: ["50021:50021"]
    steps:
      - uses: actions/checkout@v4
      - run: sudo apt-get update && sudo apt-get install -y ffmpeg
      # 音声合成は VOICEVOX エンジンの HTTP API を直接叩くため、
      # 外部 CLI ツールや ALSA/libasound2 のインストールは不要。
      - uses: actions/setup-go@v5
        with: { go-version: "1.x" }
      - run: go build -o vox-radio ./cmd/vox-radio
      - run: ./vox-radio run
        env:
          VOICEVOX_ENGINE_URL: http://localhost:50021
          GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
          HOSTING: ghpages
          PUBLIC_BASE_URL: https://<user>.github.io/vox-radio
      # ghpages 実装が gh-pages ブランチを更新（audio/・feed.xml・episodes.json）
```

### 必要な Secrets / 環境変数

| 変数                | 用途                                                                      |
|---------------------|---------------------------------------------------------------------------|
| `GEMINI_API_KEY`    | 既定 LLM（Gemini）の API キー。`llm.yaml` の `api_key_env` で名前指定      |
| `LLM_BASE_URL`      | （任意）OpenAI 互換エンドポイント上書き。プロバイダ切替に使用              |
| `LLM_MODEL`         | （任意）モデル名上書き                                                     |
| `VOICEVOX_ENGINE_URL` | VOICEVOX エンジン URL（既定 `http://localhost:50021`）                   |
| `HOSTING`           | `ghpages` / `release` / `s3` / `local`                                    |
| `PUBLIC_BASE_URL`   | feed.xml / enclosure URL の基点となる公開URL                              |
| `S3_ENDPOINT` ほか  | `HOSTING=s3` 時の接続情報（R2/S3/B2）                                      |

## 音源のライセンスに関する注意

- Podcast は原則公開配信なので、BGM/SE/ジングルは**ライセンス確認が必須**。
- フリー音源候補: DOVA-SYNDROME / 甘茶の音楽工房 / 効果音ラボ など
  （商用可否・クレジット要否・**リポジトリ同梱の可否**を必ず規約で確認）。
- 規約上リポジトリ同梱が不可の音源は、`assets/` を差し替え式にして
  各自で配置する前提とし、調達手順を README に記載する。
- 番組概要（show notes）でクレジット表記が必要な音源もあるため、
  必要に応じて feed の `<description>` 末尾にクレジットを自動付与する。

## サブコマンド案（CLI）

```
vox-radio run                  # ①〜⑥ を一括実行（本番）
vox-radio collect              # ① のみ（記事収集の確認）
vox-radio script               # ② 全体（[0]→[A]→[B]→[C]）。中間生成物を work/ に出力
vox-radio script --step=summarize|plan|write|direct   # ② の特定ステップのみ再実行
vox-radio synth <script.json>  # ③ のみ（合成）
vox-radio assemble <...>       # ④ のみ（ffmpeg 合成）
vox-radio publish <mp3>        # ⑤ のみ（アップロード＋feed.xml生成・更新）
vox-radio prune                # ⑥ のみ（古い回の削除＋feed更新）
```

ステップ単位で再実行できるため、軽量モデルの出力が不調なステップだけを作り直せる。

## 実装ステップ（推奨順）

1. 演出台本フォーマット＋各中間スキーマ（summaries/rundown/lines）の JSON Schema 確定 + Go 型定義
2. `synth`（VOICEVOX HTTP APIクライアント）と `assemble`（ffmpeg）を先に作り、固定の演出台本で 1 本生成できることを確認
3. `publish`（feed.xml 生成 + `hosting=local` で動作確認）＋ `prune`（保持管理）
4. `collect`（RSS/記事抽出）
5. `script/llm`（OpenAI 互換クライアント＋スキーマ検証＋リトライ）を実装
6. 多段ステップ [0]→[A]→[B]→[C] を順に実装（[C] はコード固定演出＋SE位置のみLLM）
7. `hosting` の本番実装（ghpages / s3）
8. `pipeline`（run で一括）→ docker-compose / GitHub Actions 整備
9. Podcast アプリ・Slack の RSS アプリで feed 購読を確認
```
