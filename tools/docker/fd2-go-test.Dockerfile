# Reproducible FD2 remake test image. Ebiten's Linux backend compiles GLFW
# and Oto even when tests do not open a window or audio device.
FROM golang:1.22-bookworm

ENV PATH=/usr/local/go/bin:${PATH}

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        pkg-config libasound2-dev xorg-dev libgl1-mesa-dev xvfb xauth fonts-noto-cjk \
    && rm -rf /var/lib/apt/lists/*

# Ebiten 在 main 之前就初始化 X11，測試一定要有 DISPLAY。用受版控的 with-xvfb
# 明確擁有 Xvfb 的生命週期；映像裡雖然裝了 xvfb-run，但它與 Xvfb 的 SIGUSR1
# 交握會偶發卡死（詳見腳本開頭的註解），正式流程不要用它。
COPY tools/docker/with-xvfb.sh /usr/local/bin/with-xvfb
RUN chmod 0755 /usr/local/bin/with-xvfb

WORKDIR /src/remake

# Keep the test invocation hermetic after image build: dependencies are fetched
# while building this explicit development image, never from the host or during
# the verification command.
COPY remake/go.mod remake/go.sum ./
RUN go mod download
