#!/bin/bash
# R0 可行性探針（docs/goal/119，#306）：在 Docker／Xvfb／Mesa lavapipe 執行 Bevy 探針，
# 輸出截圖與 JSON 收據；另外確認 fd2-app 能建 wasm32。
# 用法：tools/rust_r0_probe.sh <輸出目錄> <TTC 字型> <OGG>...（路徑須在 repo 內或為主機字型）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
out=${1:?輸出目錄}; font=${2:?字型}; shift 2
build="$root/work/rust-build"
image=fd2-rust:1.99.1-r3
mkdir -p "$out" "$build/cargo" "$build/target"
test -f "$font"
font_dir=$(cd "$(dirname "$font")" && pwd -P)
oggs=()
for ogg in "$@"; do test -f "$root/$ogg"; oggs+=("/repo/$ogg"); done
ogg_list=$(IFS=:; echo "${oggs[*]}")
limits=(--rm --init --network none --memory 10g --cpus 8 --pids-limit 2048
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h
  -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target
  --mount "type=bind,source=$root/remake-rs,target=/src"
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly"
  --mount "type=bind,source=$build,target=/build" -w /src)

timeout 3600s docker run "${limits[@]}" \
  --mount "type=bind,source=$root,target=/repo,readonly" \
  --mount "type=bind,source=$font_dir,target=/fonts,readonly" \
  --mount "type=bind,source=$(cd "$out" && pwd -P),target=/out" \
  -e FD2_PROBE_FONT="/fonts/$(basename "$font")" -e FD2_PROBE_OGG="$ogg_list" \
  -e FD2_PROBE_SHOT=/out/r0-probe.png -e FD2_PROBE_REPORT=/out/r0-probe.json \
  -e WGPU_BACKEND=vulkan -e LIBGL_ALWAYS_SOFTWARE=1 -e RUST_LOG=warn \
  "$image" bash -euo pipefail -c '
    cargo build --offline --release -p fd2-app --bin r0_probe
    Xvfb :95 -screen 0 1280x800x24 -nolisten tcp >/out/xvfb.log 2>&1 &
    xvfb=$!
    trap "kill $xvfb 2>/dev/null || true" EXIT
    for i in $(seq 1 100); do test ! -S /tmp/.X11-unix/X95 || break; sleep 0.1; done
    test -S /tmp/.X11-unix/X95
    rc=0
    DISPLAY=:95 timeout 300s /build/target/release/r0_probe >/out/r0-probe.log 2>&1 || rc=$?
    echo "probe exit $rc"
  ' || true

timeout 3600s docker run "${limits[@]}" "$image" bash -euo pipefail -c '
  cargo build --offline --release -p fd2-app --bin r0_probe --target wasm32-unknown-unknown 2>&1 | tail -3
  ls -la /build/target/wasm32-unknown-unknown/release/r0_probe.wasm
' | tee "$out/wasm-build.log"
