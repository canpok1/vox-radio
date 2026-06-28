# vox-radio 公式イメージ。
# GoReleaser が各アーキ向けにビルド済みの vox-radio バイナリをビルドコンテキストへ
# コピーして利用するため、ここでは Go ビルドは行わずランタイムのみを構成する。
FROM debian:bookworm-slim

# 音声整形に ffmpeg（ffprobe を含む）、HTTPS 通信に CA 証明書が必要。
RUN apt-get update \
    && apt-get install -y --no-install-recommends ffmpeg ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# GoReleaser がビルドした vox-radio バイナリをコピーする。
COPY vox-radio /usr/local/bin/vox-radio

WORKDIR /work
ENTRYPOINT ["vox-radio"]
