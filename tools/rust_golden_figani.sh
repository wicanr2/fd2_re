#!/bin/bash
# figani 的 Go→Rust 黃金差分（docs/goal/119）。Go 端用凍結版產生黃金 JSON，Rust 端逐欄比對。
# 用法：tools/rust_golden_figani.sh [--go-only|--rust-only]
# 原版檔案與分離素材包不在版控內，從主 checkout 唯讀掛載；可用環境變數覆寫：
#   FD2_FLAME2_HOST   原版 FLAME2 目錄（預設 <主 checkout>/org_game/炎龍騎士團/FLAME2）
#   FD2_PACK_HOST     分離素材包（預設 <主 checkout>/remake/generated-assets/fd2-original-b97caf22）
#   FD2_GOLDEN_FILE   黃金檔名（預設 figani-v1.json，位於 <主 checkout>/work/rust-build/golden/）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
# worktree 與主 checkout 共用 git common dir；建置產物一律放主 checkout 的 work/rust-build。
main="$(dirname "$(git -C "$root" rev-parse --path-format=absolute --git-common-dir)")"
build="$main/work/rust-build"
golden="$build/golden"
flame2="${FD2_FLAME2_HOST:-$main/org_game/炎龍騎士團/FLAME2}"
pack="${FD2_PACK_HOST:-$main/remake/generated-assets/fd2-original-b97caf22}"
name="${FD2_GOLDEN_FILE:-figani-v1.json}"
mode="${1:-all}"
for p in "$flame2/FIGANI.DAT" "$flame2/FDOTHER.DAT" "$root/remake/assets/data/native_command_indexed_entries.json"; do
  test -f "$p" || { echo "找不到 $p" >&2; exit 1; }
done
test -d "$pack/animations" || { echo "找不到 $pack/animations" >&2; exit 1; }
mkdir -p "$golden" "$build/cargo" "$build/target-figani"
limits=(--rm --network none --memory 6g --cpus 3 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
# 兩邊容器用同一組掛載路徑，錯誤字串中的路徑才能直接比較。
inputs=(--mount "type=bind,source=$flame2,target=/orig/FLAME2,readonly"
  --mount "type=bind,source=$pack,target=/orig/pack,readonly"
  --mount "type=bind,source=$root/remake/assets,target=/orig/remake-assets,readonly")
if [ "$mode" != "--rust-only" ]; then
  timeout 900s docker run "${limits[@]}" "${inputs[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
    --mount "type=bind,source=$root,target=/repo,readonly" \
    --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
    fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-figani -out "/golden/$name" \
    -flame2 /orig/FLAME2 -pack /orig/pack -entries /orig/remake-assets/data/native_command_indexed_entries.json \
    -work /tmp/figani-work
  sha256sum "$golden/$name"
fi
if [ "$mode" != "--go-only" ]; then
  timeout 1500s docker run "${limits[@]}" "${inputs[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
    -e FD2_GOLDEN_FIGANI="/golden/$name" -e FD2_REQUIRE_GOLDEN=1 \
    -e FD2_FLAME2_DIR=/orig/FLAME2 -e FD2_TEST_ORIGINAL_PACK=/orig/pack -e FD2_REMAKE_ASSETS=/orig/remake-assets \
    --mount "type=bind,source=$root/remake-rs,target=/src" \
    --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
    --mount "type=bind,source=$build/cargo,target=/build/cargo" \
    --mount "type=bind,source=$build/target-figani,target=/build/target-x" \
    --mount "type=bind,source=$golden,target=/golden,readonly" -w /src \
    fd2-rust:1.99.1-r3 cargo test --offline -p fd2-formats --test golden_figani -- --nocapture
fi
