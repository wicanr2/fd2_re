#!/bin/bash
# fdtxt 的 Go→Rust 黃金差分（docs/goal/119）。Go 端用凍結版產生黃金 JSON，Rust 端逐欄比對。
#
# 用法：tools/rust_golden_fdtxt.sh
# 環境變數（皆有預設值）：
#   FD2_FLAME2_DIR     原版 FLAME2 目錄（讀 FDTXT.DAT、FDOTHER.DAT）
#   FD2_SEPARATED_DIR  分離素材根目錄（remake/generated-assets/fd2-original-b97caf22）
#   FD2_RUST_BUILD     CARGO_HOME、target 與黃金檔所在（預設 <repo>/work/rust-build）
#   FD2_CARGO_TARGET   target 子目錄名（預設 target）
# 原版檔與分離素材都唯讀掛載，黃金檔只記路徑與 SHA-256；黃金檔不進版控。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
flame2="${FD2_FLAME2_DIR:-$root/org_game/炎龍騎士團/FLAME2}"
separated="${FD2_SEPARATED_DIR:-$root/remake/generated-assets/fd2-original-b97caf22}"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
target="$build/${FD2_CARGO_TARGET:-target}"
golden="$build/golden"
for f in "$flame2/FDTXT.DAT" "$flame2/FDOTHER.DAT" "$separated/fonts/fdother_004/font.json"; do
  test -f "$f" || { echo "找不到 $f" >&2; exit 1; }
done
test -d "$separated/text" || { echo "找不到 $separated/text" >&2; exit 1; }
mkdir -p "$golden" "$build/cargo" "$target"
limits=(--rm --network none --memory 6g --cpus 3 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
inputs=(--mount "type=bind,source=$flame2,target=/flame2,readonly"
  --mount "type=bind,source=$separated,target=/separated,readonly")
timeout 600s docker run "${limits[@]}" "${inputs[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
  --mount "type=bind,source=$root,target=/repo,readonly" \
  --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
  fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-fdtxt -out /golden/fdtxt-v1.json -flame2 /flame2 -separated /separated
timeout 1200s docker run "${limits[@]}" "${inputs[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_FDTXT=/build/golden/fdtxt-v1.json -e FD2_REQUIRE_GOLDEN=1 \
  -e FD2_FLAME2_DIR=/flame2 -e FD2_SEPARATED_DIR=/separated -e FD2_REQUIRE_ORIGINAL=1 \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/build/golden,readonly" -w /src \
  fd2-rust:1.99.1-r3 cargo test --offline -p fd2-formats --test golden_fdtxt -- --nocapture
