#!/usr/bin/env bash
# Docker-internal recording owner. Inputs are ordinary X11 keys, sent separately.
set -euo pipefail
test -f /.dockerenv
# Bash reads a long script incrementally. Execute an immutable container-local
# copy so subsequent workspace edits cannot corrupt a running capture owner.
if [[ "${FD2_CAPTURE_RUNNER_COPY:-}" != "$0" ]]; then
  runner=$(mktemp /tmp/fd2-capture.XXXXXX.sh)
  cp -- "$0" "$runner"
  bash -n "$runner"
  export FD2_CAPTURE_RUNNER_COPY="$runner"
  exec bash "$runner" "$@"
fi
game=${1:?AppDir path}
out=${2:?capture directory}
duration=${3:-900}
size=${4:-640x400}
fps=${5:-60}
[[ "$duration" =~ ^[1-9][0-9]*$ ]] && (( duration <= 900 ))
[[ "$size" == 640x400 || "$size" == 1280x800 ]]
[[ "$fps" == 30 || "$fps" == 60 ]]
test -x "$game/AppRun"
test -f "$game/pack/manifest.json"
test ! -e "$out/live.mkv"
test ! -e "$out/video.mkv"
test ! -e "$out/audio.mka"
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
  kill -INT "${audio_pid:-}" 2>/dev/null || true
  wait "${record_pid:-}" 2>/dev/null || true
  wait "${audio_pid:-}" 2>/dev/null || true
  kill "${game_pid:-}" "${xvfb_pid:-}" "${pulse_pid:-}" 2>/dev/null || true
  kill "${memory_pid:-}" 2>/dev/null || true
  wait "${game_pid:-}" "${xvfb_pid:-}" "${pulse_pid:-}" 2>/dev/null || true
  if test -s "$out/video.mkv" && test -s "$out/audio.mka"; then
    ffmpeg -hide_banner -loglevel warning -y -i "$out/video.mkv" -i "$out/audio.mka" \
      -map 0:v:0 -map 1:a:0 -c copy -shortest "$out/live.mkv" >"$out/mux.log" 2>&1
  fi
}
trap cleanup EXIT
Xvfb :95 -screen 0 "${size}x24" -nolisten tcp >"$out/xvfb.log" 2>&1 & xvfb_pid=$!
pulseaudio --daemonize=no --exit-idle-time=-1 --disallow-exit --log-target=stderr >"$out/pulse.log" 2>&1 & pulse_pid=$!
for attempt in $(seq 1 100); do
  if test -S /tmp/.X11-unix/X95 && pactl info >/dev/null 2>&1; then break; fi
  sleep .1
done
pactl load-module module-null-sink sink_name=fd2_capture rate=44100 channels=2 >"$out/pulse-sink.txt"
pactl set-default-sink fd2_capture
printf 'pcm.!default { type pulse }\nctl.!default { type pulse }\n' >"$HOME/.asoundrc"
python3 - "$out" "$size" "$fps" "$duration" "$0" <<'PY'
import hashlib,json,os,pathlib,sys,time
pathlib.Path(sys.argv[1], 'start.json').write_text(json.dumps({'epoch': time.time(), 'monotonic': time.monotonic(), 'window_size':sys.argv[2], 'capture_fps':int(sys.argv[3]), 'duration_limit':int(sys.argv[4]), 'capture_mode':'independent X11 video and PulseAudio audio; stream-copy mux; initial device-start offset is not hardware-exact', 'capture_script_sha256':hashlib.sha256(pathlib.Path(sys.argv[5]).read_bytes()).hexdigest(), 'gameplay':os.getenv('FD2_CAPTURE_LABEL','unmodified release binary; ordinary keyboard input and window resize; edited continuity disclosed in final metadata')}))
PY
(
  while true; do
    date +%s
    cat /sys/fs/cgroup/memory.current /sys/fs/cgroup/memory.events
    sleep 2
  done
) >"$out/memory.log" & memory_pid=$!
ffmpeg -hide_banner -loglevel warning -y -thread_queue_size 8 \
  -f x11grab -framerate "$fps" -video_size "$size" -draw_mouse 0 -i :95 \
  -t "$duration" -filter_threads 1 -threads 2 -c:v libx264 -preset veryfast -crf 18 \
  -pix_fmt yuv420p "$out/video.mkv" >"$out/ffmpeg.log" 2>&1 & record_pid=$!
ffmpeg -hide_banner -loglevel warning -y -thread_queue_size 32 \
  -f pulse -fragment_size 4096 -i fd2_capture.monitor -t "$duration" \
  -threads 1 -c:a flac "$out/audio.mka" >"$out/ffmpeg-audio.log" 2>&1 & audio_pid=$!
cd /tmp
FD2_INPUT_AUDIT_LOG="$out/input-audit.jsonl" FD2_CUTSCENE_LOG=1 \
  "$game/AppRun" >"$out/game.log" 2>&1 & game_pid=$!
window=""
for attempt in $(seq 1 900); do
  window=$(xdotool search --name '炎龍騎士團' 2>/dev/null | head -1 || true)
  if test -n "$window"; then
    xdotool windowsize "$window" "${size%x*}" "${size#*x}"
    xdotool windowmove "$window" 0 0
    xdotool windowfocus "$window"
    break
  fi
  kill -0 "$game_pid" "$record_pid" "$audio_pid" 2>/dev/null || break
  sleep .1
done
test -n "$window"
for attempt in $(seq 1 "$duration"); do
  test ! -e "$out/STOP" || break
  kill -0 "$game_pid" "$record_pid" "$audio_pid" 2>/dev/null || break
  sleep 1
done
