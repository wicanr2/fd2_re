#!/bin/bash
# fd2-rules 跨組整合的 Go→Rust 黃金差分（docs/goal/119 的 R2 完成條件：以相同種子跑完整的 AI 戰鬥，
# Rust 與 Go 逐回合狀態相同）。Go 端用凍結版工具重播 33 張地圖、每張 3 個種子的 AI 戰鬥並產生黃金
# JSON，Rust 端 tests/golden_rules_int.rs 以同一份無頭驅動逐步比對。
# 用法：tools/rust_golden_rules_int.sh
#   FD2_RUST_BUILD  建置目錄（預設 work/rust-build；在工作樹副本中執行時指向主工作區的）
#   FD2_RUST_TARGET 建置目錄下的 cargo target 目錄名稱（預設 target）
# 真實輸入只用受版控的 remake/assets（唯讀掛載）；黃金檔記下每個真實檔的路徑、大小與 SHA-256，
# Rust 端讀同一份檔案前先核對雜湊。重播的步數多，Rust 端以 --release 建置。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
golden="$build/golden"
target="${FD2_RUST_TARGET:-target}"
test -d "$root/remake/assets" || { echo "找不到 $root/remake/assets" >&2; exit 1; }
mkdir -p "$golden" "$build/cargo" "$build/$target"
limits=(--rm --network none --memory 6g --cpus 4 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
timeout 1500s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
  --mount "type=bind,source=$root,target=/repo,readonly" \
  --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
  fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-rules-int -out /golden/rules-int-v1.json
sha256sum "$golden/rules-int-v1.json"
timeout 2400s docker run "${limits[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_RULES_INT=/build/golden/rules-int-v1.json -e FD2_REQUIRE_GOLDEN=1 -e FD2_GOLDEN_REPO=/repo \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$root/remake/assets,target=/repo/remake/assets,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/build/golden,readonly" -w /src \
  fd2-rust:1.99.1-r3 cargo test --release --offline --locked -p fd2-rules --test golden_rules_int -- --nocapture
