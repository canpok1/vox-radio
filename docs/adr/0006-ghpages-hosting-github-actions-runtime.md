# 0006. ホスティングを ghpages、実行基盤を GitHub Actions に一本化する

- ステータス: 採用
- 日付: 2026-05-30
- 補足: [ADR-0012](0012-separate-distribution-and-content-manifest.md) により配信機能（hosting・publish/prune）は vox-radio 本体から別リポジトリへ分離する。本 ADR のホスティング/実行基盤の判断は分離先の配信リポジトリで引き継ぐ。

## コンテキスト

公開先（ghpages / release / s3 互換 / local）と実行基盤（GitHub Actions cron / 自宅サーバー docker compose）は当初いずれも両対応の設計だった。だが対応形態が増えるほど設定・テスト・運用手順が分岐し、個人運用には過剰になる。配信は RSS 一本化（ADR 0005）で軽量であり、追加課金なしで手離れよく回したい。

## 決定

公開先は **ghpages に一本化**する（gh-pages ブランチに audio/・feed.xml・episodes.json を置く。音声で履歴が肥大しないよう orphan ブランチを毎回作り直す運用）。s3 互換・release・local は採用しない。実行基盤は **GitHub Actions cron に一本化**し、自宅サーバー（docker compose）両対応はやめる。運用用 docker-compose / Dockerfile は作らない。ただし `Hosting` インターフェース自体は残し、将来の差し替え余地を確保する。

## 結果

- GitHub Actions ＋ ghpages の組み合わせで、追加課金ゼロ・手離れよく運用できる。設定と CI が単純になる。
- 1 週間分（数十 MB）なら ghpages の容量・帯域で十分。
- 自宅サーバー運用やオフライン LLM は対象外になる。必要になれば `Hosting` 実装と実行基盤を追加して対応する。
- VOICEVOX エンジンは Actions のサービスコンテナで起動する。

## 検討した代替案

- **s3 互換(R2/B2 等)**: 署名 URL での限定公開や大容量に強いが、認証情報管理と外部依存が増え、現状の規模に過剰なため却下。
- **自宅サーバー両対応の維持**: 柔軟だが ghpages 一本化と相性が悪く、運用手順が二重化するため却下。
