#!/bin/bash
# fd2-rules C 組（物品效果與面板繪製）的 Go→Rust 黃金差分（docs/goal/119）。
# Go 端用凍結版工具產生黃金 JSON，Rust 端 tests/golden_rules_c.rs 逐欄比對。
# 用法：tools/rust_golden_rules_c.sh
#   FD2_RUST_BUILD   建置目錄（預設 work/rust-build；在工作樹副本中執行時指向主工作區的）
#   FD2_RUST_TARGET  建置目錄下的 cargo target 目錄名稱（預設 target）
#   FD2_FLAME2_DIR／FD2_ASSET_PACK 原版 FLAME2 與分離素材包（預設在本工作區底下）
# 真實輸入：受版控的 remake/assets、原版 FDOTHER.DAT／FDTXT.DAT 與分離素材包，全部唯讀掛載；黃金檔
# 只記檔案標籤、大小與 SHA-256，Rust 端讀同一批檔案前先核對。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
golden="$build/golden"
target="${FD2_RUST_TARGET:-target}"
flame="${FD2_FLAME2_DIR:-$root/org_game/炎龍騎士團/FLAME2}"
pack="${FD2_ASSET_PACK:-$root/remake/generated-assets/fd2-original-b97caf22}"
for dir in "$flame" "$pack" "$root/remake/assets"; do
  test -d "$dir" || { echo "找不到 $dir" >&2; exit 1; }
done
test -f "$flame/FDOTHER.DAT" || { echo "找不到 $flame/FDOTHER.DAT" >&2; exit 1; }
mkdir -p "$golden" "$build/cargo" "$build/$target"
limits=(--rm --network none --memory 6g --cpus 4 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
timeout 900s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
  --mount "type=bind,source=$root,target=/repo,readonly" \
  --mount "type=bind,source=$flame,target=/flame2,readonly" \
  --mount "type=bind,source=$pack,target=/pack,readonly" \
  --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
  fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-rules-c -out /golden/rules-c-v1.json -flame2 /flame2 -pack /pack
sha256sum "$golden/rules-c-v1.json"
timeout 1500s docker run "${limits[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_RULES_C=/build/golden/rules-c-v1.json -e FD2_REQUIRE_GOLDEN=1 -e FD2_GOLDEN_REPO=/repo \
  -e FD2_GOLDEN_FLAME2=/flame2 -e FD2_GOLDEN_PACK=/pack \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$root/remake/assets,target=/repo/remake/assets,readonly" \
  --mount "type=bind,source=$flame,target=/flame2,readonly" \
  --mount "type=bind,source=$pack,target=/pack,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/build/golden,readonly" -w /src \
  fd2-rust:1.99.1-r3 cargo test --offline --locked -p fd2-rules --test golden_rules_c -- --nocapture
