#!/usr/bin/env bash
# Docker-internal recording owner. Inputs are ordinary X11 keys, sent separately.
set -euo pipefail
test -f /.dockerenv
game=${1:?AppDir path}
out=${2:?capture directory}
test -x "$game/AppRun"
test -f "$game/pack/manifest.json"
test ! -e "$out/live.mkv"
mkdir -p "$out"
export HOME=/tmp/fd2-promo-home XDG_DATA_HOME="$out/user-data"
export XDG_RUNTIME_DIR=/tmp/fd2-pulse DISPLAY=:95
export LIBGL_ALWAYS_SOFTWARE=1 GALLIUM_DRIVER=llvmpipe
export GOMAXPROCS=2 GOMEMLIMIT=2GiB LP_NUM_THREADS=2
mkdir -p "$HOME" "$XDG_RUNTIME_DIR" "$XDG_DATA_HOME"
chmod 700 "$XDG_RUNTIME_DIR"
export PULSE_SERVER=unix:$XDG_RUNTIME_DIR/pulse/native
cleanup() {
  kill -INT "${record_pid:-}" 2>/dev/null || true
  wait "${record_pid:-}" 2>/dev/null || true
  kill "${game_pid:-}" "${xvfb_pid:-}" "${pulse_pid:-}" 2>/dev/null || true
  kill "${memory_pid:-}" 2>/dev/null || true
  wait "${game_pid:-}" "${xvfb_pid:-}" "${pulse_pid:-}" 2>/dev/null || true
}
trap cleanup EXIT
Xvfb :95 -screen 0 1280x800x24 -nolisten tcp >"$out/xvfb.log" 2>&1 & xvfb_pid=$!
pulseaudio --daemonize=no --exit-idle-time=-1 --disallow-exit --log-target=stderr >"$out/pulse.log" 2>&1 & pulse_pid=$!
for attempt in $(seq 1 100); do
  if test -S /tmp/.X11-unix/X95 && pactl info >/dev/null 2>&1; then break; fi
  sleep .1
done
pactl load-module module-null-sink sink_name=fd2_capture rate=44100 channels=2 >"$out/pulse-sink.txt"
pactl set-default-sink fd2_capture
printf 'pcm.!default { type pulse }\nctl.!default { type pulse }\n' >"$HOME/.asoundrc"
python3 - "$out" <<'PY'
import json,pathlib,sys,time
pathlib.Path(sys.argv[1], 'start.json').write_text(json.dumps({'epoch': time.time(), 'monotonic': time.monotonic(), 'gameplay': 'unmodified release binary; ordinary keyboard input; edited continuity disclosed in final metadata'}))
PY
(
  while true; do
    date +%s
    cat /sys/fs/cgroup/memory.current /sys/fs/cgroup/memory.events
    sleep 2
  done
) >"$out/memory.log" & memory_pid=$!
ffmpeg -hide_banner -loglevel warning -y -thread_queue_size 8 \
  -f x11grab -framerate 30 -video_size 1280x800 -draw_mouse 0 -i :95 \
  -thread_queue_size 32 -f pulse -i fd2_capture.monitor \
  -t 900 -filter_threads 1 -threads 2 -c:v libx264 -preset veryfast -crf 18 \
  -pix_fmt yuv420p -c:a flac "$out/live.mkv" >"$out/ffmpeg.log" 2>&1 & record_pid=$!
cd /tmp
FD2_INPUT_AUDIT_LOG="$out/input-audit.jsonl" FD2_CUTSCENE_LOG=1 \
  "$game/AppRun" >"$out/game.log" 2>&1 & game_pid=$!
for attempt in $(seq 1 900); do
  test ! -e "$out/STOP" || break
  kill -0 "$game_pid" "$record_pid" 2>/dev/null || break
  sleep 1
done
