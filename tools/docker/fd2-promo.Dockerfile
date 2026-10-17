# FD2 promo capture revision: replaces the unavailable historical game-video image.
# The existing Go/X11 image supplies the same runtime as the release smoke test.
FROM fd2-go-test-local:latest
USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg imagemagick xdotool pulseaudio-utils pulseaudio libxtst-dev \
    && rm -rf /var/lib/apt/lists/*
USER 1000:1000
WORKDIR /src
