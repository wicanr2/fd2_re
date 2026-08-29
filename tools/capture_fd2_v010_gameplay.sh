#!/usr/bin/env bash
set -euo pipefail

repo=$(cd "$(dirname "$0")/.." && pwd)
bundle=${1:-$repo/remake/packaging/dist/local-full/fd2-linux-x86_64-local-full}
out=${2:-$repo/dist/promo/fd2-v0.1.0-live-gameplay.mp4}

test -x "$bundle/FD2-x86_64.AppImage" || {
  echo "缺少 Linux 完整版 AppImage：$bundle/FD2-x86_64.AppImage" >&2
  exit 2
}
test -f "$bundle/assets/manifest.json" || {
  echo "缺少本機分離素材清單：$bundle/assets/manifest.json" >&2
  exit 2
}

mkdir -p "$(dirname "$out")"
uid=$(id -u)
gid=$(id -g)
out_dir=$(cd "$(dirname "$out")" && pwd)
out_name=$(basename "$out")

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  -u "$uid:$gid" \
  -v "$bundle:/game:ro" -v "$out_dir:/out:rw" \
  --entrypoint bash game-video:latest -lc '
set -uo pipefail
export HOME=/tmp/home XDG_DATA_HOME=/tmp/data DISPLAY=:95
export LIBGL_ALWAYS_SOFTWARE=1 GALLIUM_DRIVER=llvmpipe
mkdir -p "$HOME" "$XDG_DATA_HOME/fd2_re"
ln -s /game/assets "$XDG_DATA_HOME/fd2_re/assets"

Xvfb :95 -screen 0 1280x800x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
xvfb_pid=$!
cleanup() {
  kill "${game_pid:-}" "${record_pid:-}" "$xvfb_pid" 2>/dev/null || true
  wait "${game_pid:-}" "${record_pid:-}" "$xvfb_pid" 2>/dev/null || true
}
trap cleanup EXIT

for attempt in $(seq 1 50); do
  test -S /tmp/.X11-unix/X95 && break
  sleep 0.1
done

ffmpeg -hide_banner -loglevel error -y \
  -f x11grab -framerate 30 -video_size 1280x800 -i :95 -t 55 \
  -c:v libx264 -preset medium -crf 18 -pix_fmt yuv420p "/out/$1" &
record_pid=$!

cd /tmp
env FD2_TITLE=1 FD2_NOCUT=1 FD2_MUTE=1 \
  /game/FD2-x86_64.AppImage --appimage-extract-and-run \
  >/tmp/fd2.log 2>&1 &
game_pid=$!

window=""
for attempt in $(seq 1 100); do
  window=$(xdotool search --name "炎龍騎士團" 2>/dev/null | head -1 || true)
  test -n "$window" && break
  sleep 0.1
done
test -n "$window"

sleep 2
xdotool key --window "$window" space || true
sleep 1
xdotool key --window "$window" space || true
sleep 2
xdotool key --window "$window" Return || true
sleep 2
for press in $(seq 1 24); do
  xdotool key --window "$window" Return || true
  sleep 0.4
done
sleep 2
for language in 1 2 3 4; do
  xdotool key --window "$window" F4 || true
  sleep 2
done
xdotool key --window "$window" Right || true
sleep 1
xdotool key --window "$window" Down || true
sleep 1
xdotool key --window "$window" Return || true

wait "$record_pid"
' _ "$out_name"

echo "$out"
