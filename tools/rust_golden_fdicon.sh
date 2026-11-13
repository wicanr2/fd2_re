#!/bin/bash
# fdicon 的 Go→Rust 黃金差分（docs/goal/119）。Go 端用凍結版產生黃金 JSON，Rust 端逐欄比對。
# 用法：tools/rust_golden_fdicon.sh
#   FD2_FLAME2_DIR  含 FDICON.B24 的原版目錄（預設 org_game/炎龍騎士團/FLAME2）
#   FD2_ASSET_PACK  分離素材包根目錄（預設 remake/generated-assets/fd2-original-b97caf22）
#   FD2_RUST_BUILD  建置目錄（預設 work/rust-build；在工作樹副本中執行時指向主工作區的）
#   FD2_RUST_TARGET 建置目錄下的 cargo target 目錄名稱（預設 target）
# 原版檔與素材包都唯讀掛載，黃金檔只記其路徑與 SHA-256，不嵌入內容。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
golden="$build/golden"
target="${FD2_RUST_TARGET:-target}"
flame2="${FD2_FLAME2_DIR:-$root/org_game/炎龍騎士團/FLAME2}"
assets="${FD2_ASSET_PACK:-$root/remake/generated-assets/fd2-original-b97caf22}"
test -f "$flame2/FDICON.B24" || { echo "找不到 $flame2/FDICON.B24" >&2; exit 1; }
test -f "$assets/sprites/fdicon/bank.json" || { echo "找不到 $assets/sprites/fdicon/bank.json" >&2; exit 1; }
mkdir -p "$golden" "$build/cargo" "$build/$target"
limits=(--rm --network none --memory 6g --cpus 4 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
real=(--mount "type=bind,source=$flame2,target=/real/flame2,readonly"
  --mount "type=bind,source=$assets,target=/real/assets,readonly")
timeout 900s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
  --mount "type=bind,source=$root,target=/repo,readonly" "${real[@]}" \
  --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
  fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-fdicon -out /golden/fdicon-v1.json \
  -flame2 /real/flame2 -assets /real/assets
timeout 1500s docker run "${limits[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_FDICON=/build/golden/fdicon-v1.json -e FD2_REQUIRE_GOLDEN=1 \
  -e FD2_FDICON_REAL_FLAME2=/real/flame2 -e FD2_FDICON_REAL_ASSETS=/real/assets "${real[@]}" \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/build/golden,readonly" -w /src \
  fd2-rust:1.99.1-r3 cargo test --offline -p fd2-formats --test golden_fdicon -- --nocapture
