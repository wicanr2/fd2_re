#!/bin/bash
# fd2-rules F 組（劇本事件、回合與快照）的 Go→Rust 黃金差分（docs/goal/119）。
# 1. Go 端用凍結版工具產生黃金 JSON；
# 2. Rust 端 tests/golden_rules_f.rs 逐步重做同一串操作並逐欄比對，另把自己寫出的戰場快照存檔；
# 3. Go 端 -verify-rust 讀回 Rust 寫的快照、RestoreSnapshot 後重新 Snapshot，與黃金比對（快照雙向）。
# 用法：tools/rust_golden_rules_f.sh
#   FD2_RUST_BUILD  建置目錄（預設 work/rust-build；在工作樹副本中執行時指向主工作區的）
#   FD2_RUST_TARGET 建置目錄下的 cargo target 目錄名稱（預設 target）
# 真實輸入只用受版控的 remake/assets（唯讀掛載）；黃金檔記下每個真實檔的路徑、大小與 SHA-256，
# Rust 端讀同一份檔案前先核對雜湊。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
golden="$build/golden"
target="${FD2_RUST_TARGET:-target}"
test -d "$root/remake/assets" || { echo "找不到 $root/remake/assets" >&2; exit 1; }
mkdir -p "$golden" "$build/cargo" "$build/$target"
limits=(--rm --network none --memory 6g --cpus 4 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
timeout 900s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
  --mount "type=bind,source=$root,target=/repo,readonly" \
  --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
  fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-rules-f -out /golden/rules-f-v1.json
sha256sum "$golden/rules-f-v1.json"
rm -f "$golden/rules-f-rust-v1.json"
timeout 1500s docker run "${limits[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_RULES_F=/build/golden/rules-f-v1.json -e FD2_REQUIRE_GOLDEN=1 -e FD2_GOLDEN_REPO=/repo \
  -e FD2_GOLDEN_RULES_F_RUST_OUT=/build/golden/rules-f-rust-v1.json \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$root/remake/assets,target=/repo/remake/assets,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/build/golden" -w /src \
  fd2-rust:1.99.1-r3 cargo test --offline --locked -p fd2-rules --test golden_rules_f -- --nocapture
test -f "$golden/rules-f-rust-v1.json" || { echo "Rust 端沒有寫出快照檔" >&2; exit 1; }
timeout 900s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
  --mount "type=bind,source=$root,target=/repo,readonly" \
  --mount "type=bind,source=$golden,target=/golden,readonly" -w /repo/remake \
  fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-rules-f \
  -verify-rust /golden/rules-f-rust-v1.json -golden /golden/rules-f-v1.json
