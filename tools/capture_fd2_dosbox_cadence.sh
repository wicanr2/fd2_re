#!/usr/bin/env bash
# Docker-only auxiliary presentation diagnosis. Not a parity oracle.
set -euo pipefail
test -f /.dockerenv
original=${1:?read-only original directory}
out=${2:?output directory}
duration=${3:-120}
cycles=${4:-12000}
[[ "$duration" =~ ^[1-9][0-9]*$ ]] && (( duration <= 180 ))
[[ "$cycles" =~ ^[1-9][0-9]*$ ]] && (( cycles <= 100000 ))
test -f "$original/FD2.EXE"
test ! -e "$out/dosbox.mkv"
mkdir -p "$out"
export HOME=/tmp/fd2-dosbox-home DISPLAY=:96 SDL_AUDIODRIVER=dummy
mkdir -p "$HOME"
python3 - "$original" "$out" "$cycles" <<'PY'
import hashlib,json,pathlib,shutil,subprocess,sys
source,out=map(pathlib.Path,sys.argv[1:3])
sha=hashlib.sha256((source/'FD2.EXE').read_bytes()).hexdigest()
assert sha=='222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f'
shutil.copytree(source,'/tmp/fd2-dosbox-game')
version=subprocess.run(['dosbox-x','-version'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
(out/'dosbox-source.json').write_text(json.dumps({'original_exe_sha256':sha,'cycles':int(sys.argv[3]),'runner':'DOSBox-X auxiliary presentation diagnosis','audio':'disabled; no audio timing claim','state':'disposable copy; no state edits; no gameplay input','version':version.stdout,'version_exit_code':version.returncode},ensure_ascii=False,indent=2))
PY
cat >"$out/dosbox.conf" <<EOF
[sdl]
fullscreen=false
output=surface
autolock=false
windowresolution=640x400
waitonerror=false
showmenu=false
[dosbox]
machine=svga_s3
memsize=32
[render]
frameskip=0
aspect=false
scaler=none
[cpu]
core=normal
cputype=auto
cycles=fixed $cycles
[mixer]
nosound=true
[midi]
mpu401=none
[sblaster]
sbtype=none
[speaker]
pcspeaker=false
[joystick]
joysticktype=none
[dos]
xms=true
ems=true
umb=true
keyboardlayout=us
[autoexec]
mount c /tmp/fd2-dosbox-game
c:
FD2.EXE
exit
EOF
cleanup() {
  kill -INT "${record_pid:-}" 2>/dev/null || true
  wait "${record_pid:-}" 2>/dev/null || true
  kill "${dos_pid:-}" "${xvfb_pid:-}" 2>/dev/null || true
  wait "${dos_pid:-}" "${xvfb_pid:-}" 2>/dev/null || true
}
trap cleanup EXIT
Xvfb :96 -screen 0 640x400x24 -nolisten tcp >"$out/xvfb.log" 2>&1 & xvfb_pid=$!
for attempt in $(seq 1 50); do
  xdotool getdisplaygeometry >/dev/null 2>&1 && break
  sleep .1
done
ffmpeg -hide_banner -loglevel warning -y -thread_queue_size 8 -f x11grab \
  -framerate 60 -video_size 640x400 -draw_mouse 0 -i :96 \
  -t "$duration" -an -filter_threads 1 -threads 2 -c:v libx264 \
  -preset veryfast -crf 18 -pix_fmt yuv420p "$out/dosbox.mkv" >"$out/ffmpeg.log" 2>&1 & record_pid=$!
dosbox-x -conf "$out/dosbox.conf" -noconsole >"$out/dosbox.log" 2>&1 & dos_pid=$!
wait "$record_pid"
