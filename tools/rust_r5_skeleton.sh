#!/bin/bash
# R5 階段 1（docs/goal/121）：替開場與第一章執行到的非 HD 繪圖函式補 R4 規則的簽名骨架，並更新 PORTING.json 狀態。
#
# 用法：tools/rust_r5_skeleton.sh
# 1. Go 容器（fd2-go-test-local，--network none）：go list 匯出資料，建置 tools/rust_r4_skeleton，以 -r5 讀
#    docs/data/rust-r5-executed-draw-functions-20261115.json（階段 0 盤點的受版控複本，SHA-256 60e0a421…），
#    輸出 r5-plan.json（每個函式的狀態依據與骨架原始碼）。R4 的型別閉包不擴大，簽名用到閉包外的型別即失敗。
# 2. Python 容器：tools/rust_r5_skeleton_apply.py 依 crate 現況套用（已 ported 的不動，見該檔說明）。
# 3. Rust 容器：cargo fmt。
# 重跑時第二次不改任何檔；最後印出 crate 的 PORTING.json 與 src/ 總雜湊。
#
# 環境變數：
#   FD2_GOCACHE     Go 建置快取（預設 work/gocache；在工作樹副本中執行時指向主工作區的）
#   FD2_RUST_BUILD  Rust 建置目錄（預設 work/rust-build；cargo fmt 只用其中的 cargo 目錄）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
gocache="${FD2_GOCACHE:-$root/work/gocache}"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
crate="$root/remake-rs/crates/fd2-game"
inv="$root/docs/data/rust-r5-executed-draw-functions-20261115.json"
for d in "$root/remake/cmd/fd2" "$root/tools/rust_r4_skeleton" "$gocache" "$crate/src" "$build/cargo"; do
  test -d "$d" || { echo "找不到 $d" >&2; exit 1; }
done
test -f "$inv" && test -f "$root/tools/rust_r5_skeleton_apply.py"
test "$(sha256sum "$inv" | cut -c1-64)" = 60e0a421b3a2f0906f3c69fe282bc534ec491dc131d5e77486dd3c6e3ada1f2a
stage=$(mktemp -d -t fd2-r5skel-XXXXXX)
trap 'rm -rf "$stage"' EXIT
mkdir -p "$stage/out"
timeout 900 docker run --rm --name "fd2-r5a-skel-go-$$" --network none --memory 6g --cpus 4 --pids-limit 512 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
  -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod \
  --mount "type=bind,source=$root,target=/src,readonly" \
  --mount "type=bind,source=$gocache,target=/gocache" \
  --mount "type=bind,source=$stage/out,target=/out" \
  fd2-go-test-local:latest bash -euo pipefail -c '
    cd /src/remake && go list -export -deps -json ./cmd/fd2 > /tmp/deps.json
    cd /src/tools/rust_r4_skeleton && test -z "$(gofmt -l .)" && go vet . && go build -o /tmp/rust_r4_skeleton .
    /tmp/rust_r4_skeleton -deps /tmp/deps.json -repo /src -out /out -r5 /src/docs/data/rust-r5-executed-draw-functions-20261115.json' \
  > "$stage/gen.log" 2>&1 || { cat "$stage/gen.log" >&2; exit 1; }
grep -E '^STAT' "$stage/gen.log" || true
timeout 300 docker run --rm --name "fd2-r5a-skel-py-$$" --network none --memory 1g --cpus 1 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
  --mount "type=bind,source=$root/tools,target=/tools,readonly" \
  --mount "type=bind,source=$stage/out,target=/out,readonly" \
  --mount "type=bind,source=$crate,target=/crate" \
  python:3.13-slim python3 -I /tools/rust_r5_skeleton_apply.py /out/r5-plan.json /crate
timeout 600 docker run --rm --name "fd2-r5a-skel-fmt-$$" --network none --memory 2g --cpus 2 --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp/h -e CARGO_HOME=/build/cargo \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" -w /src \
  fd2-rust:1.99.1-r3 cargo fmt -p fd2-game
cd "$crate"
find PORTING.json src -type f | LC_ALL=C sort | xargs sha256sum > "$stage/sums"
echo "crate $(wc -l < "$stage/sums") 個檔，總雜湊 $(sha256sum < "$stage/sums" | cut -c1-64)"
