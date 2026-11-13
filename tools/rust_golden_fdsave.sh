#!/bin/bash
# fdsave 的 Go→Rust 黃金差分（docs/goal/119）。Go 端用凍結版產生黃金 JSON，Rust 端逐欄比對。
# 用法：tools/rust_golden_fdsave.sh [真實 FD2.SAV ...]（路徑須在 repo 內）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="$root/work/rust-build"
golden="$build/golden"
mkdir -p "$golden" "$build/cargo" "$build/target"
save_args=()
for save in "$@"; do
  test -f "$root/$save" || { echo "找不到 $save" >&2; exit 1; }
  save_args+=(-save "/repo/$save")
done
limits=(--rm --network none --memory 6g --cpus 6 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
timeout 600s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
  --mount "type=bind,source=$root,target=/repo,readonly" \
  --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
  fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-fdsave -out /golden/fdsave-v1.json "${save_args[@]}"
timeout 1200s docker run "${limits[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target \
  -e FD2_GOLDEN_FDSAVE=/build/golden/fdsave-v1.json -e FD2_REQUIRE_GOLDEN=1 \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$build,target=/build" -w /src \
  fd2-rust:1.99.1-r1 cargo test --offline -p fd2-formats --test golden_fdsave -- --nocapture
