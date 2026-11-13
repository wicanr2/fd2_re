#!/bin/bash
# afm 的 Go→Rust 黃金差分（docs/goal/119）。Go 端用凍結版產生黃金 JSON（連跑兩次確認逐位元組相同），
# Rust 端逐欄比對，並以同一份原版檔跑 afm 單元測試。
# 用法：tools/rust_golden_afm.sh [含 ANI.DAT 的目錄] [分離動畫目錄]
#   預設為 repo 內的 org_game/炎龍騎士團/FLAME2 與 remake/generated-assets/fd2-original-b97caf22/animations。
#   FD2_RUST_BUILD 可改 work/rust-build 位置（例如在 worktree 內執行時指向主 checkout）。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
golden="$build/golden"
target="${FD2_RUST_TARGET:-$build/target-afm}"
org="${1:-$root/org_game/炎龍騎士團/FLAME2}"
anim="${2:-$root/remake/generated-assets/fd2-original-b97caf22/animations}"
test -f "$org/ANI.DAT" || { echo "找不到 $org/ANI.DAT" >&2; exit 1; }
test -f "$anim/ANI_000/animation.json" || { echo "找不到 $anim/ANI_000/animation.json" >&2; exit 1; }
test -f "$root/LICENSE" || { echo "找不到 $root/LICENSE" >&2; exit 1; }
mkdir -p "$golden" "$build/cargo" "$target"
limits=(--rm --network none --memory 6g --cpus 4 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
real=(--mount "type=bind,source=$org,target=/real/flame2,readonly"
  --mount "type=bind,source=$anim,target=/real/animations,readonly")
timeout 900s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
  --mount "type=bind,source=$root,target=/repo,readonly" "${real[@]}" \
  --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
  fd2-build-mingw:1.22.12-r1 sh -c '
    set -e
    go run ./cmd/fd2-golden-afm -out /golden/afm-v1.json -org /real/flame2 -animations /real/animations
    go run ./cmd/fd2-golden-afm -out /tmp/afm-again.json -org /real/flame2 -animations /real/animations
    cmp /golden/afm-v1.json /tmp/afm-again.json && echo "fd2-golden-afm: 兩次輸出相同"'
timeout 1500s docker run "${limits[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_AFM=/golden/afm-v1.json -e FD2_REQUIRE_GOLDEN=1 \
  -e FD2_AFM_ORG_DIR=/real/flame2 -e FD2_AFM_ANIMATIONS_DIR=/real/animations -e FD2_ANI_DAT=/real/flame2/ANI.DAT -e FD2_SEPARATED_ANIMATIONS=/real/animations \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/golden,readonly" "${real[@]}" -w /src \
  fd2-rust:1.99.1-r3 sh -c '
    set -e
    cargo test --offline -p fd2-formats --lib afm
    cargo test --offline -p fd2-formats --test golden_afm -- --nocapture'
