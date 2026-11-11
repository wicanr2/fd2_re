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
startup_timeout=${FD2_CAPTURE_STARTUP_TIMEOUT:-90}
render_threads=${FD2_CAPTURE_RENDER_THREADS:-2}
[[ "$duration" =~ ^[1-9][0-9]*$ ]] && (( duration <= 3600 ))
[[ "$size" == 640x400 || "$size" == 1280x800 || "$size" == 1920x1200 ]]
[[ "$fps" == 30 || "$fps" == 60 ]]
[[ "$startup_timeout" == 90 || "$startup_timeout" == 180 || "$startup_timeout" == 360 || "$startup_timeout" == 900 ]]
[[ "$render_threads" == 2 || "$render_threads" == 4 || "$render_threads" == 8 ]]
test -x "$game/AppRun"
test -f "$game/pack/manifest.json"
test ! -e "$out/live.mkv"
test ! -e "$out/video.mkv"
test ! -e "$out/audio.mka"
mkdir -p "$out"
export HOME=/tmp/fd2-promo-home XDG_DATA_HOME="$out/user-data"
export XDG_RUNTIME_DIR=/tmp/fd2-pulse DISPLAY=:95
export LIBGL_ALWAYS_SOFTWARE=1 GALLIUM_DRIVER=llvmpipe
export GOMAXPROCS=2 GOMEMLIMIT="${FD2_CAPTURE_GO_MEMORY:-6GiB}" LP_NUM_THREADS="$render_threads"
mkdir -p "$HOME" "$XDG_RUNTIME_DIR" "$XDG_DATA_HOME"
chmod 700 "$XDG_RUNTIME_DIR"
python3 - "$game" "$out" <<'PY'
import hashlib,json,pathlib,sys
game,out=map(pathlib.Path,sys.argv[1:])
pack=game/'modern'
report={'status':'PASS-CAPTURE-SOURCE-PATHS','modern_present':pack.is_dir(),'scope':'只核對清冊來源路徑，內容與形狀仍由正式執行期驗證。'}
user_assets=out/'user-data/fd2_re/assets'
bootstrap=user_assets if user_assets.is_dir() else game/'assets'
if not (bootstrap/'map.json').is_file():
    raise SystemExit(f'錄影啟動素材目錄缺少map.json：{bootstrap}')
report['bootstrap_map_sha256']=hashlib.sha256((bootstrap/'map.json').read_bytes()).hexdigest()
if pack.is_dir():
    overlay=out/'user-data/fd2_re/assets/themes/modern/catalog.json'
    catalog=overlay if overlay.is_file() else game/'assets/themes/modern/catalog.json'
    raw=catalog.read_bytes();assets=json.loads(raw)['assets'];names=set()
    for asset in assets:
        if asset.get('status') not in {'runtime_candidate','runtime_ready'}:continue
        names.update(asset.get('files',[]))
        if asset.get('file'):names.add(asset['file'])
        if asset.get('role')=='opening_presentation_hd_set':names.update(asset.get('source_files',[]))
        elif asset.get('role')=='story_animation_hd_set':names.update(n for n in asset.get('source_files',[]) if n.endswith('.json'))
    for name in sorted(names):
        if pathlib.Path(name).name!=name or not (pack/name).is_file():
            raise SystemExit(f'錄影素材來源不完整：{name}')
    report.update(catalog_sha256=hashlib.sha256(raw).hexdigest(),runtime_paths=len(names))
(out/'capture-source-preflight.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
PY
export PULSE_SERVER=unix:$XDG_RUNTIME_DIR/pulse/native
# Polling is bounded even when an X11 read or GL swap does not return.
stop_children() {
  local signal=$1 limit=$2 pid attempt alive
  shift 2
  for pid in "$@"; do
    [[ "$pid" =~ ^[1-9][0-9]*$ ]] && kill -"$signal" "$pid" 2>/dev/null || true
  done
  for ((attempt=0; attempt<limit; attempt++)); do
    alive=0
    for pid in "$@"; do
      if [[ "$pid" =~ ^[1-9][0-9]*$ ]] && kill -0 "$pid" 2>/dev/null; then alive=1; fi
    done
    (( alive )) || return 0
    sleep .1
  done
}
cleanup() {
  trap - EXIT INT TERM
  stop_children INT 50 "${record_pid:-}" "${audio_pid:-}"
  stop_children TERM 20 "${record_pid:-}" "${audio_pid:-}" "${game_pid:-}" "${memory_pid:-}"
  stop_children KILL 10 "${record_pid:-}" "${audio_pid:-}" "${game_pid:-}" "${memory_pid:-}"
  stop_children TERM 20 "${xvfb_pid:-}" "${pulse_pid:-}"
  stop_children KILL 10 "${xvfb_pid:-}" "${pulse_pid:-}"
  if test -s "$out/video.mkv" && test -s "$out/audio.mka"; then
    timeout -k 2s 30s ffmpeg -hide_banner -loglevel warning -y -i "$out/video.mkv" -i "$out/audio.mka" \
      -map 0:v:0 -map 1:a:0 -c copy -shortest "$out/live.mkv" >"$out/mux.log" 2>&1 || true
  fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
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
FD2_INPUT_AUDIT_LOG="$out/input-audit.jsonl" FD2_INPUT_AUDIT_INTERVAL=1 FD2_CUTSCENE_LOG=1 \
  "$game/AppRun" >"$out/game.log" 2>&1 & game_pid=$!
window=""
for attempt in $(seq 1 "$((startup_timeout * 10))"); do
  test ! -e "$out/STOP" || break
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
if test -z "$window"; then
  printf '遊戲視窗未在 %s 秒啟動預算內出現。\n' "$startup_timeout" >&2
  exit 2
fi
for attempt in $(seq 1 "$duration"); do
  test ! -e "$out/STOP" || break
  kill -0 "$game_pid" "$record_pid" "$audio_pid" 2>/dev/null || break
  sleep 1
done
