#!/bin/bash
# fdother B 組（原版流程邏輯）的 Go→Rust 黃金差分（docs/goal/119）。Go 端用凍結版產生黃金 JSON，
# Rust 端逐欄比對。本組不讀原版檔，FDOTHER #80 的原版資源檢查留在單元測試（FD2_FLAME2_DIR）。
# 用法：tools/rust_golden_fdother_native.sh
#   FD2_RUST_BUILD  建置目錄（預設 work/rust-build；在工作樹副本中執行時指向主工作區的）
#   FD2_RUST_TARGET 建置目錄下的 cargo target 目錄名稱（預設 target）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
golden="$build/golden"
target="${FD2_RUST_TARGET:-target}"
mkdir -p "$golden" "$build/cargo" "$build/$target"
limits=(--rm --network none --memory 6g --cpus 3 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
timeout 900s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
  --mount "type=bind,source=$root,target=/repo,readonly" \
  --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
  fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-fdother-native -out /golden/fdother-native-v1.json
timeout 1500s docker run "${limits[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_FDOTHER_NATIVE=/build/golden/fdother-native-v1.json -e FD2_REQUIRE_GOLDEN=1 \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/build/golden,readonly" -w /src \
  fd2-rust:1.99.1-r3 cargo test --offline --locked -p fd2-formats --test golden_fdother_native -- --nocapture
