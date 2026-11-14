#!/bin/bash
# fd2-campaign present 組（localization、musiccatalog）的 Go→Rust 黃金差分（docs/goal/119，R3）。
# Go 端用凍結版工具產生黃金 JSON，Rust 端 tests/golden_present_loc.rs 逐欄比對。
# 用法：tools/rust_golden_present_loc.sh
#   FD2_RUST_BUILD     建置目錄（預設 work/rust-build；在工作樹副本中執行時指向主工作區的）
#   FD2_RUST_TARGET    建置目錄下的 cargo target 目錄名稱（預設 target）
#   FD2_REMAKE_ASSETS  remake/assets（預設本工作區那份）。music_fm／music_mt32 的 render 不入庫，工作樹副本
#                      沒有；在副本中執行時指向主工作區的 remake/assets（受版控的檔案兩邊相同）
# 真實輸入全部唯讀掛在兩個容器的同一個路徑 /assets，錯誤訊息裡的路徑兩邊相同；黃金檔記錄每個檔案的
# 大小與 SHA-256，Rust 端讀同一批檔案前先核對。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
golden="$build/golden"
target="${FD2_RUST_TARGET:-target}"
assets="${FD2_REMAKE_ASSETS:-$root/remake/assets}"
for dir in "$assets/locales" "$assets/music_fm" "$assets/music_mt32" "$root/remake/internal/localization"; do
  test -d "$dir" || { echo "找不到 $dir" >&2; exit 1; }
done
test -f "$assets/music_catalog.json" || { echo "找不到 $assets/music_catalog.json" >&2; exit 1; }
mkdir -p "$golden" "$build/cargo" "$build/$target"
limits=(--rm --network none --memory 6g --cpus 4 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
timeout 900s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
  --mount "type=bind,source=$root,target=/repo,readonly" \
  --mount "type=bind,source=$assets,target=/assets,readonly" \
  --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
  fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-present-loc -out /golden/present-loc-v1.json -assets /assets
sha256sum "$golden/present-loc-v1.json"
timeout 1500s docker run "${limits[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_PRESENT_LOC=/build/golden/present-loc-v1.json -e FD2_REQUIRE_GOLDEN=1 -e FD2_GOLDEN_ASSETS=/assets \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$assets,target=/assets,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/build/golden,readonly" -w /src \
  fd2-rust:1.99.1-r3 cargo test --offline --locked -p fd2-campaign --test golden_present_loc -- --nocapture
