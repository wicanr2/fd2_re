#!/usr/bin/env bash
set -euo pipefail

repo=$(cd "$(dirname "$0")/.." && pwd)
bundle=${1:-$repo/remake/packaging/dist/local-full/fd2-linux-x86_64-local-full}
out=${2:-$repo/dist/promo/fd2-ch01-march-remake.mp4}
duration=${FD2_CAPTURE_DURATION:-240}
presses=${FD2_CAPTURE_PRESSES:-2000}

test -x "$bundle/FD2-x86_64.AppImage" || { echo "缺少本機完整版 AppImage" >&2; exit 2; }
test -f "$bundle/assets/manifest.json" || { echo "缺少本機分離素材清單" >&2; exit 2; }
mkdir -p "$(dirname "$out")"

uid=$(id -u)
gid=$(id -g)
out_dir=$(cd "$(dirname "$out")" && pwd)

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  -u "$uid:$gid" -v "$bundle:/game:ro" -v "$out_dir:/out:rw" \
  --entrypoint bash game-video:latest -lc '
set -euo pipefail
export HOME=/tmp/home XDG_DATA_HOME=/tmp/data DISPLAY=:96
export LIBGL_ALWAYS_SOFTWARE=1 GALLIUM_DRIVER=llvmpipe
mkdir -p "$HOME" "$XDG_DATA_HOME/fd2_re"
ln -s /game/assets "$XDG_DATA_HOME/fd2_re/assets"
Xvfb :96 -screen 0 1280x800x24 -nolisten tcp >/tmp/xvfb.log 2>&1 & xvfb_pid=$!
cleanup() {
  kill "${game_pid:-}" "${record_pid:-}" "$xvfb_pid" 2>/dev/null || true
  wait "${game_pid:-}" "${record_pid:-}" "$xvfb_pid" 2>/dev/null || true
}
trap cleanup EXIT
for attempt in $(seq 1 50); do test -S /tmp/.X11-unix/X96 && break; sleep 0.1; done
ffmpeg -hide_banner -loglevel error -y -f x11grab -framerate 30 \
  -video_size 1280x800 -i :96 -t "$2" -c:v libx264 -preset medium -crf 18 \
  -pix_fmt yuv420p "/out/$1" >/out/fd2-ch01-march-ffmpeg.log 2>&1 & record_pid=$!
cd /tmp
env FD2_TITLE=1 FD2_NOCUT=1 FD2_MUTE=1 FD2_CUTSCENE_LOG=1 \
  FD2_CAMPAIGN=assets/scenarios/campaign_full.json \
  /game/FD2-x86_64.AppImage --appimage-extract-and-run >/out/fd2-ch01-march-remake.log 2>&1 & game_pid=$!
window=""
for attempt in $(seq 1 100); do
  window=$(xdotool search --name "炎龍騎士團" 2>/dev/null | head -1 || true)
  test -n "$window" && break
  sleep 0.1
done
test -n "$window"
sleep 2
xdotool key --window "$window" space
sleep 1
xdotool key --window "$window" space
sleep 22
# 先讓標題 reveal 完整結束，再送正常 Return 輸入，避免標題階段的重複輸入
# 跨狀態排隊。cutscene acting 是阻塞拍，輸入不會改寫其座標或跳過逐格移動。
for press in $(seq 1 "$3"); do
  if ! xdotool keydown --window "$window" Return 2>/dev/null; then
    break
  fi
  sleep 0.08
  xdotool keyup --window "$window" Return 2>/dev/null || break
  sleep 0.12
done
wait "$record_pid"
' _ "$(basename "$out")" "$duration" "$presses"

echo "$out"
