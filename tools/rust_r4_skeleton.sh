#!/bin/bash
# R4 階段 1（docs/goal/120）：重新產生 remake-rs/crates/fd2-game 的骨架。
#
# 用法：tools/rust_r4_skeleton.sh
# 1. Go 容器（fd2-go-test-local，--network none）：go list -export -deps -json ./cmd/fd2 取得匯出資料，
#    建置 tools/rust_r4_skeleton 並以 go/types 分析 remake/cmd/fd2 正式程式，輸出到暫存目錄。
# 2. Rust 容器（fd2-rust）：rustfmt 排版。
# 3. 取代 crate 的 Cargo.toml、PORTING.json 與 src/（PORTING.md 是人工維護，不動），印出每個輸出檔的
#    SHA-256 與總雜湊（重跑兩次應相同）。
# 輸入：remake/cmd/fd2（唯讀）、docs/data/rust-r4-ch01-coverage-20261114.json、remake-rs 既有 crate 的原始碼。
#
# 注意：階段 2 開始移植後，crate 內的檔案改為人工維護；再跑本腳本會蓋掉移植成果，只用來重現骨架提交。
#
# 環境變數：
#   FD2_GOCACHE     Go 建置快取（預設 work/gocache；在工作樹副本中執行時指向主工作區的）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
gocache="${FD2_GOCACHE:-$root/work/gocache}"
crate="$root/remake-rs/crates/fd2-game"
for d in "$root/remake/cmd/fd2" "$root/tools/rust_r4_skeleton" "$gocache"; do
  test -d "$d" || { echo "找不到 $d" >&2; exit 1; }
done
test -f "$root/docs/data/rust-r4-ch01-coverage-20261114.json"
stage=$(mktemp -d -t fd2-r4skel-XXXXXX)
trap 'rm -rf "$stage"' EXIT
mkdir -p "$stage/out"
timeout 900 docker run --rm --network none --memory 6g --cpus 4 --pids-limit 512 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
  -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod \
  --mount "type=bind,source=$root,target=/src,readonly" \
  --mount "type=bind,source=$gocache,target=/gocache" \
  --mount "type=bind,source=$stage/out,target=/out" \
  fd2-go-test-local:latest bash -euo pipefail -c '
    cd /src/remake && go list -export -deps -json ./cmd/fd2 > /tmp/deps.json
    cd /src/tools/rust_r4_skeleton && go vet . && go build -o /tmp/rust_r4_skeleton .
    /tmp/rust_r4_skeleton -deps /tmp/deps.json -repo /src -out /out' > "$stage/gen.log" 2>&1 || {
  cat "$stage/gen.log" >&2
  exit 1
}
grep -E '^(STAT|REVIEW|MISSING)' "$stage/gen.log" || true
timeout 600 docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp/h \
  --mount "type=bind,source=$stage/out,target=/out" \
  --mount "type=bind,source=$root/remake-rs/rustfmt.toml,target=/rustfmt.toml,readonly" -w /out \
  fd2-rust:1.99.1-r3 rustfmt --edition 2024 --config-path /rustfmt.toml src/lib.rs
mkdir -p "$crate"
rm -rf "$crate/src" "$crate/Cargo.toml" "$crate/PORTING.json"
cp -r "$stage/out/src" "$stage/out/Cargo.toml" "$stage/out/PORTING.json" "$crate/"
cd "$crate"
find Cargo.toml PORTING.json src -type f | LC_ALL=C sort | xargs sha256sum > "$stage/sums"
echo "輸出 $(wc -l < "$stage/sums") 個檔，總雜湊 $(sha256sum < "$stage/sums" | cut -c1-64)"
