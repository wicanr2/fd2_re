#!/bin/bash
# fd2-campaign present 組 battlepresent 的 Go→Rust 黃金差分（docs/goal/119）。
# Go 端用凍結版工具產生黃金 JSON，Rust 端 tests/golden_present_bp.rs 逐欄比對。
# 用法：tools/rust_golden_present_bp.sh
#   FD2_RUST_BUILD   建置目錄（預設 work/rust-build；在工作樹副本中執行時指向主工作區的）
#   FD2_RUST_TARGET  建置目錄下的 cargo target 目錄名稱（預設 target）
#   FD2_ASSET_PACK   分離素材包（預設在本工作區底下）
# 真實輸入：分離素材包 animations/ 底下用到的 FIGANI 與 FDOTHER 動畫，唯讀掛載；黃金檔只記檔案標籤、
# 大小與 SHA-256，Rust 端讀同一批檔案前先核對。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
golden="$build/golden"
target="${FD2_RUST_TARGET:-target}"
pack="${FD2_ASSET_PACK:-$root/remake/generated-assets/fd2-original-b97caf22}"
test -d "$pack/animations" || { echo "找不到 $pack/animations" >&2; exit 1; }
mkdir -p "$golden" "$build/cargo" "$build/$target"
limits=(--rm --network none --memory 6g --cpus 4 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
timeout 900s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
  --mount "type=bind,source=$root,target=/repo,readonly" \
  --mount "type=bind,source=$pack,target=/pack,readonly" \
  --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
  fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-present-bp -out /golden/present-bp-v1.json -pack /pack
sha256sum "$golden/present-bp-v1.json"
timeout 1500s docker run "${limits[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_PRESENT_BP=/build/golden/present-bp-v1.json -e FD2_REQUIRE_GOLDEN=1 -e FD2_GOLDEN_PACK=/pack \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$pack,target=/pack,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/build/golden,readonly" -w /src \
  fd2-rust:1.99.1-r3 cargo test --offline --locked -p fd2-campaign --test golden_present_bp -- --nocapture
