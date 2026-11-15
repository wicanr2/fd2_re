#!/bin/bash
# R5 階段 4（docs/goal/121，r5w）：fd2-app 外殼驗證。
#
# 每份腳本（預設 tools/rust_r5_app_scripts/*.json）跑兩次：
#   1. 無畫面驅動端 fd2_frames（不經 Bevy）→ <輸出>/<腳本名>/headless.jsonl
#   2. Bevy 外殼 fd2（Xvfb＋Mesa llvmpipe，按鍵與觸控以 Bevy 輸入訊息注入）→ <輸出>/<腳本名>/app/
# 再以 tools/rust_r5_app_frames_compare.py 核對：兩邊每個顯示幀的 CPU 畫格雜湊序列相同；外殼每張截圖等於同一幀
# CPU 畫格的整數倍最近鄰放大（逐像素，不遮蔽）。
#
# 用法：tools/rust_r5_app_frames.sh <輸出目錄> [腳本.json ...]
# 輸入（唯讀掛載）：建構槽 $FD2_WORK/ch01-flow-20261017/input/FD2.SAV（SHA-256 6d14f2c2…，與 R5 opening 黃金同一個）、
#   分離素材包、remake/assets、TTF 字型（tools/rust_ttf_font.sh）。
# 環境變數：FD2_WORK／FD2_ASSET_PACK／FD2_REMAKE_ASSETS／FD2_RUST_BUILD／FD2_RUST_TARGET 同 tools/rust_parity_ch01.sh；
#   FD2_PYTHON_IMAGE 比對用映像（預設 python:3.13-slim）。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
source "$root/tools/rust_ttf_font.sh"
out="$(mkdir -p "${1:?輸出目錄}" && cd "$1" && pwd -P)"
shift
scripts=("$@")
if [ ${#scripts[@]} -eq 0 ]; then
  scripts=("$root"/tools/rust_r5_app_scripts/*.json)
fi
work="${FD2_WORK:-$root/work}"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
target="${FD2_RUST_TARGET:-target}"
packsrc="${FD2_ASSET_PACK:-$root/remake/generated-assets/fd2-original-b97caf22}"
assets="${FD2_REMAKE_ASSETS:-$root/remake/assets}"
pyimage="${FD2_PYTHON_IMAGE:-python:3.13-slim}"
slot="$work/ch01-flow-20261017/input"
for d in "$slot" "$packsrc" "$assets" "$build/cargo"; do
  test -d "$d" || { echo "找不到 $d" >&2; exit 2; }
done
test -f "$slot/FD2.SAV" || { echo "缺少 $slot/FD2.SAV" >&2; exit 2; }
test "$(sha256sum "$slot/FD2.SAV" | cut -c1-64)" = 6d14f2c22562cabca83725084f1a9d6539a1d4066da5c1debcdadb446812691f
for s in "${scripts[@]}"; do test -f "$s" || { echo "缺少腳本 $s" >&2; exit 2; }; done
mkdir -p "$build/$target"

# 完整分離素材根（同 tools/rust_parity_ch01.sh）：符號連結根，目標寫成容器內路徑。
pack="$out/pack"
rm -rf "$pack"
mkdir -p "$pack"
trap 'rm -rf "$pack"' EXIT
for e in "$packsrc"/*; do
  ln -s "/packsrc/$(basename "$e")" "$pack/$(basename "$e")"
done
ln -s /assets/locales "$pack/locales"

common=(--rm --network none --pids-limit 1024 --log-opt max-size=10m --log-opt max-file=3
  --user "$(id -u):$(id -g)" -e HOME=/tmp/h -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x
  --mount "type=bind,source=$root/remake-rs,target=/src"
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly"
  --mount "type=bind,source=$build/cargo,target=/build/cargo"
  --mount "type=bind,source=$build/$target,target=/build/target-x" -w /src)

echo "== 建置 fd2-app（release）"
timeout 3600s docker run "${common[@]}" --name "fd2-r5w-build-$$" --memory 8g --cpus 4 fd2-rust:1.99.1-r3 \
  bash -euo pipefail -c 'cargo build --offline --locked --release -p fd2-app --bins 2>&1 | tail -3'

status=0
for s in "${scripts[@]}"; do
  name="$(basename "$s" .json)"
  dir="$out/$name"
  rm -rf "$dir"
  mkdir -p "$dir/app" "$dir/state-headless" "$dir/state-app" "$dir/script"
  cp "$s" "$dir/script/script.json"
  cp "$slot/FD2.SAV" "$dir/state-headless/FD2.SAV"
  cp "$slot/FD2.SAV" "$dir/state-app/FD2.SAV"
  game_env=(-e FD2_ASSET_PACK=/pack -e FD2_REMAKE_ASSETS=/assets -e FD2_CAMPAIGN=canonical -e FD2_SEED=4
    -e FD2_MUTE=1 -e FD2_NATIVE_SAVE=/state/FD2.SAV -e XDG_DATA_HOME=/state/xdg
    --mount "type=bind,source=$pack,target=/pack,readonly"
    --mount "type=bind,source=$packsrc,target=/packsrc,readonly"
    --mount "type=bind,source=$assets,target=/assets,readonly"
    --mount "type=bind,source=$dir/script,target=/script,readonly"
    "${ttf_mounts[@]}")
  echo "== $name：無畫面驅動端"
  timeout 3600s docker run "${common[@]}" --name "fd2-r5w-headless-$$" --memory 4g --cpus 2 "${game_env[@]}" \
    --mount "type=bind,source=$dir/state-headless,target=/state" \
    --mount "type=bind,source=$dir,target=/out" fd2-rust:1.99.1-r3 \
    bash -euo pipefail -c '/build/target-x/release/fd2_frames /script/script.json /out/headless.jsonl' \
    > "$dir/headless.log" 2>&1 || { status=$?; tail -5 "$dir/headless.log"; continue; }
  echo "== $name：Bevy 外殼（Xvfb）"
  timeout 3600s docker run "${common[@]}" --init --name "fd2-r5w-app-$$" --memory 6g --cpus 4 "${game_env[@]}" \
    --mount "type=bind,source=$dir/state-app,target=/state" \
    --mount "type=bind,source=$dir/app,target=/app-out" \
    -e FD2_APP_SCRIPT=/script/script.json -e FD2_APP_OUT=/app-out \
    -e WGPU_BACKEND=vulkan -e LIBGL_ALWAYS_SOFTWARE=1 -e RUST_LOG=warn fd2-rust:1.99.1-r3 \
    bash -euo pipefail -c '
      Xvfb :95 -screen 0 1920x1200x24 -nolisten tcp >/app-out/xvfb.log 2>&1 &
      xvfb=$!
      trap "kill $xvfb 2>/dev/null || true" EXIT
      for i in $(seq 1 100); do test ! -S /tmp/.X11-unix/X95 || break; sleep 0.1; done
      test -S /tmp/.X11-unix/X95
      DISPLAY=:95 timeout 3000s /build/target-x/release/fd2
    ' > "$dir/app.log" 2>&1 || { status=$?; tail -5 "$dir/app.log"; continue; }
  echo "== $name：比對"
  timeout 1200s docker run --rm --name "fd2-r5w-cmp-$$" --network none --memory 2g --cpus 1 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp \
    --mount "type=bind,source=$root/tools,target=/repo/tools,readonly" \
    --mount "type=bind,source=$dir,target=/d" -w /repo "$pyimage" \
    python3 -I tools/rust_r5_app_frames_compare.py --headless /d/headless.jsonl --app /d/app --out /d/compare.json \
    > "$dir/compare.txt" 2>&1 || status=1
  cat "$dir/compare.txt"
done
exit "$status"
