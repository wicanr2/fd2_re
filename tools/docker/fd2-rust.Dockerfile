# fd2-rust — Rust 改寫的建置與測試映像（docs/goal/119）。
# Rust 版本與 dosgolem-rust 相同（1.99.1、edition 2024）。Bevy 的 Linux 相依與
# Mesa lavapipe／Xvfb 一併裝好，讓視窗與截圖測試能在無 GPU 的容器內執行。
# 建置：docker build -f tools/docker/fd2-rust.Dockerfile -t fd2-rust:1.99.1-r3 tools/docker
# r3：補 libxkbcommon-x11-0（winit 在 X11 動態載入）。r2：加入 wasm32-unknown-unknown 目標與 ALSA null 裝置（容器內沒有音效卡，cpal 需要可開啟的 PCM）。
FROM rust:1.99.1-trixie
RUN apt-get update -qq && apt-get install -y -qq --no-install-recommends \
        pkg-config clang lld libasound2-dev libudev-dev libwayland-dev libxkbcommon-dev \
        libx11-dev libxi-dev libxcursor-dev libxrandr-dev libxkbcommon-x11-0 \
        mesa-vulkan-drivers libvulkan1 xvfb xauth \
    && rm -rf /var/lib/apt/lists/*
RUN rustup component add clippy rustfmt && rustup target add wasm32-unknown-unknown \
    && rustc --version && cargo --version
RUN printf 'pcm.!default { type null }\n' > /etc/asound.conf
ENV CARGO_TERM_COLOR=never
WORKDIR /src
