# dosbox-x(含 heavy debugger)build/執行環境。
# 目的:對《炎龍騎士團2》FD2.EXE 做執行期記憶體 dump(見 docs/knowledge-base/48-dosbox-x-debugger-build.md)。
#
#   docker build -t fd2-dosbox-x docker/dosbox-x
#   docker run --rm -v "$PWD/org_game/炎龍騎士團/FLAME2:/game:ro" -it fd2-dosbox-x
#
# 來源固定 commit(可重現):joncampbell123/dosbox-x master @ 0d7b272b690351a92405ee1d672152ee134da35b
FROM debian:bookworm-slim

# 套件清單依 dosbox-x 官方 BUILD.md「To compile DOSBox-X in Ubuntu」段落(bookworm 上驗證過皆存在),
# 另加 libsdl2-dev(讓 build-debug-sdl2 偵測到系統 sdl2-config,不必內建重編 SDL2,加速 build)、
# xvfb/imagemagick(headless 開圖 + 截圖佐證)、tmux(驅動 ncurses debugger TUI 用)。
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates curl \
        automake autoconf libtool pkg-config \
        gcc g++ make nasm \
        libncurses-dev \
        libsdl-net1.2-dev libsdl2-net-dev libsdl2-dev \
        libpcap-dev libslirp-dev \
        fluidsynth libfluidsynth-dev \
        libavdevice-dev libavformat-dev libavcodec-dev libswscale-dev \
        libfreetype-dev libxkbfile-dev libxrandr-dev \
        libpng-dev zlib1g-dev \
        xvfb x11-apps imagemagick tmux \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
RUN curl -sL -o dosbox-x.tar.gz \
        https://github.com/joncampbell123/dosbox-x/archive/0d7b272b690351a92405ee1d672152ee134da35b.tar.gz \
    && tar xzf dosbox-x.tar.gz \
    && mv dosbox-x-0d7b272b690351a92405ee1d672152ee134da35b dosbox-x \
    && rm dosbox-x.tar.gz

WORKDIR /src/dosbox-x
# build-debug-sdl2 內部會跑 autogen.sh + configure --enable-debug=heavy --enable-sdl2 + make。
# --enable-debug=heavy 定義 C_DEBUG + C_HEAVY_DEBUG(見 configure.ac,需要 curses)= 啟用 ncurses debugger。
RUN ./build-debug-sdl2 && make install

# 追加一層裝 xdotool(headless 自動化送鍵用,沿用 dq3 專案 tools/dosbox_run.sh 已驗證模式)。
# 獨立成最後一層,不動前面的編譯層,重跑 docker build 時吃得到既有 cache 不必重編。
RUN apt-get update && apt-get install -y --no-install-recommends xdotool \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /game
ENV SDL_VIDEODRIVER=x11 \
    DISPLAY=:70

CMD ["dosbox-x"]
