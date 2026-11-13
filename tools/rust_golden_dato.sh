#!/bin/bash
# dato 的 Go→Rust 黃金差分（docs/goal/119）。Go 端用凍結版產生黃金 JSON，Rust 端逐欄比對。
# 用法：tools/rust_golden_dato.sh [FLAME2 目錄] [分離頭像目錄]
#   預設為 repo 內的 org_game/炎龍騎士團/FLAME2 與
#   remake/generated-assets/fd2-original-b97caf22/portraits；兩者都在 .gitignore 內，
#   在 worktree 執行時請傳主 checkout 的路徑。兩者都以唯讀掛載。
# 環境變數：FD2_RUST_BUILD 指定建置目錄（預設 work/rust-build，內含 cargo 與 golden）；
#   FD2_RUST_TARGET 指定 cargo target 目錄（預設 $FD2_RUST_BUILD/target）。
# 黃金檔寫到 work/rust-build/golden/dato-v1.json，不進版控。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
flame="${1:-$root/org_game/炎龍騎士團/FLAME2}"
portraits="${2:-$root/remake/generated-assets/fd2-original-b97caf22/portraits}"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
golden="$build/golden"
target="${FD2_RUST_TARGET:-$build/target}"
test -f "$flame/DATO.DAT" || { echo "找不到 $flame/DATO.DAT" >&2; exit 1; }
test -f "$portraits/DATO_026_m3.png" || { echo "找不到 $portraits/DATO_026_m3.png" >&2; exit 1; }
mkdir -p "$golden" "$build/cargo" "$target"
limits=(--rm --network none --memory 6g --cpus 3 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
originals=(--mount "type=bind,source=$flame,target=/flame2,readonly"
  --mount "type=bind,source=$portraits,target=/portraits,readonly")
timeout 600s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
  --mount "type=bind,source=$root,target=/repo,readonly" "${originals[@]}" \
  --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
  fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-dato -out /golden/dato-v1.json \
  -dat /flame2/DATO.DAT -portraits /portraits -tmp /tmp/fd2-golden-dato
timeout 1200s docker run "${limits[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_DATO=/golden/dato-v1.json -e FD2_REQUIRE_GOLDEN=1 \
  -e FD2_FLAME2_DIR=/flame2 -e FD2_PORTRAITS_DIR=/portraits \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" "${originals[@]}" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/golden,readonly" -w /src \
  fd2-rust:1.99.1-r3 cargo test --offline -p fd2-formats --test golden_dato -- --nocapture
