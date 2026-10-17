#!/usr/bin/env bash
# Execute inside the project Go/X11 image or the existing Wine/X11 image.
set -euo pipefail
test -f /.dockerenv
platform=${1:?linux or windows}
release=${2:?version delivery directory}
out=${3:?writable smoke directory}
version=$(basename "$release")
test -d "$release/full-local"
test -d "$out"
export HOME=/tmp/fd2-full-home XDG_DATA_HOME=/tmp/fd2-full-xdg
export DISPLAY=:96 LIBGL_ALWAYS_SOFTWARE=1 GALLIUM_DRIVER=llvmpipe
export GOMAXPROCS=2 GOMEMLIMIT=2GiB LP_NUM_THREADS=2
mkdir -p "$HOME" "$XDG_DATA_HOME" /tmp/fd2-full-empty
Xvfb :96 -screen 0 1280x800x24 -nolisten tcp >"$out/full-$platform-xvfb.log" 2>&1 & xvfb_pid=$!
cleanup() {
  if test "$platform" = windows; then wineserver -k 2>/dev/null || true; fi
  kill "$xvfb_pid" 2>/dev/null || true
  wait "$xvfb_pid" 2>/dev/null || true
}
trap cleanup EXIT
for attempt in $(seq 1 100); do test ! -S /tmp/.X11-unix/X96 || break; sleep .1; done
test -S /tmp/.X11-unix/X96
cd /tmp/fd2-full-empty
if test "$platform" = linux; then
  image="$release/full-local/FD2-$version-linux-x86_64-full-local.AppImage"
  test -x "$image"
  FD2_PACKAGE_SELF_CHECK=1 timeout 120s "$image" --appimage-extract-and-run >"$out/full-linux-self-check.log" 2>&1
  FD2_SHOT="$out/full-linux-opening.png" FD2_SHOT_FRAME=120 \
    FD2_SHOT_STATE="$out/full-linux-opening.json" \
    timeout 180s "$image" --appimage-extract-and-run >"$out/full-linux-opening.log" 2>&1
  test -s "$out/full-linux-opening.png"
else
  test "$platform" = windows
  export WINEPREFIX=/tmp/fd2-full-wine WINEARCH=win64 WINEDEBUG=-all
  export LANG=C.UTF-8 LC_ALL=C.UTF-8
  mkdir -p "$WINEPREFIX"
  archive="$release/full-local/FD2-$version-windows-x86_64-full-local.zip"
  python3 - "$archive" <<'PY'
import pathlib,sys,zipfile
with zipfile.ZipFile(sys.argv[1]) as archive:
    for entry in archive.infolist():
        relative=pathlib.PurePosixPath(entry.filename)
        assert not relative.is_absolute() and '..' not in relative.parts
    archive.extractall('/tmp/fd2-full-unzip')
PY
  launcher="Z:\\tmp\\fd2-full-unzip\\FD2-$version-windows-x86_64-full-local\\開始遊戲.bat"
  FD2_PACKAGE_SELF_CHECK=1 timeout 120s /usr/lib/wine/wine64 cmd /c "$launcher" >"$out/full-windows-self-check.log" 2>&1
  FD2_SHOT="Z:\\out\\full-windows-opening.png" FD2_SHOT_FRAME=120 \
    FD2_SHOT_STATE="Z:\\out\\full-windows-opening.json" \
    timeout 180s /usr/lib/wine/wine64 cmd /c "$launcher" >"$out/full-windows-opening.log" 2>&1
  test -s "$out/full-windows-opening.png"
fi
cat "$out/full-$platform-self-check.log"
