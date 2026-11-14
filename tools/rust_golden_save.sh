#!/bin/bash
# fd2-game F 組（存讀檔與設定，docs/goal/120 R4）的 Go→Rust 黃金差分。
# 1. Go 端：凍結版 remake/cmd/fd2 的 TestRustGoldenSave（只在 FD2_RUST_GOLDEN_SAVE_OUT 設定時執行、
#    只輸出資料）寫出黃金 JSON；連跑兩次，兩次的 SHA-256 必須相同。
# 2. Rust 端：fd2-game 的 game::save::golden_save_test 讀同一份黃金檔逐欄比對讀回結果，並比對
#    Rust 重新寫出的位元組。
# 用法：tools/rust_golden_save.sh
#   FD2_RUST_BUILD   建置目錄（預設 work/rust-build；在工作樹副本中執行時指向主工作區的）
#   FD2_RUST_TARGET  建置目錄下的 cargo target 目錄名稱（預設 target）
#   FD2_GOCACHE      Go 建置快取（預設 work/gocache；在工作樹副本中執行時指向主工作區的）
#   FD2_GOLDEN_TAG   黃金檔名的標記（預設 v1）
#   FD2_REMAKE_ASSETS remake/assets（含不入庫的 map.json；工作樹副本指向主工作區的）
#   FD2_ASSET_PACK_SRC 分離素材包（預設本工作區的 remake/generated-assets/fd2-original-b97caf22）
#   FD2_REAL_WORK    真實 Go 版存檔所在的 work 目錄（預設本工作區的 work；工作樹副本指向主工作區的）
# 真實輸入只讀不寫；檔名清單固定（已排序），黃金檔記下每個檔的大小與 SHA-256。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
golden="$build/golden"
target="${FD2_RUST_TARGET:-target}"
gocache="${FD2_GOCACHE:-$root/work/gocache}"
tag="${FD2_GOLDEN_TAG:-v1}"
realwork="${FD2_REAL_WORK:-$root/work}"
pack_src="${FD2_ASSET_PACK_SRC:-$root/remake/generated-assets/fd2-original-b97caf22}"
assets="${FD2_REMAKE_ASSETS:-$root/remake/assets}"
real_files=(
  ch01-flow-20261017/remake-full-r10/hotel-save.json
  ch02-flow-20261017/remake-full-r4/hotel-save.json
  ch03-flow-20261017/remake-full-r2/hotel-save.json
  hd1-feedback-20261104-r83/special-sources-r4/command30/fd2_save.json
  hd1-performance-60fps-20261105-r89/candidate-gui/user-data/fd2_re/fd2_save.json
  hd1-performance-60fps-20261105-r89/candidate-gui/user-data/fd2_re/fd2_settings.json
  hd1-performance-60fps-20261105-r89/quality-normal-gui/user-data/fd2_re/fd2_settings.json
  promo-v.1.0.0-20261017/r2/user-data/fd2_re/fd2_save.json
  r4-stage0-20261114/parity-out/hotel-save.json
)
for f in "${real_files[@]}"; do test -f "$realwork/$f" || { echo "缺少真實輸入 $realwork/$f" >&2; exit 2; }; done
test -d "$pack_src" || { echo "找不到 $pack_src" >&2; exit 2; }
test -f "$assets/map.json" || { echo "找不到 $assets/map.json（不入庫的素材；工作樹副本指向主工作區的 remake/assets）" >&2; exit 2; }
test -d "$gocache" || { echo "找不到 $gocache" >&2; exit 2; }
real_list="$(IFS=:; echo "${real_files[*]}")"
mkdir -p "$golden" "$build/cargo" "$build/$target"
pack=$(mktemp -d -t fd2-golden-save-pack-XXXXXX)
trap 'rm -rf "$pack"' EXIT
for e in "$pack_src"/*; do
  ln -s "/pack-src/$(basename "$e")" "$pack/$(basename "$e")"
done
ln -s /repo/remake/assets/locales "$pack/locales"
out="fd2-game-save-r4f-$tag.json"
run_go() {
  timeout 1800s docker run --rm --network none --memory 6g --cpus 4 --pids-limit 512 \
    --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" \
    -e HOME=/tmp/h -e GOCACHE=/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off -e FD2_ASSET_PACK=/pack \
    -e FD2_RUST_GOLDEN_SAVE_OUT="/golden/$1" -e FD2_RUST_GOLDEN_SAVE_REAL=/realwork \
    -e FD2_RUST_GOLDEN_SAVE_REAL_FILES="$real_list" \
    --mount "type=bind,source=$root,target=/repo,readonly" \
    --mount "type=bind,source=$assets,target=/repo/remake/assets,readonly" \
    --mount "type=bind,source=$pack_src,target=/pack-src,readonly" \
    --mount "type=bind,source=$pack,target=/pack,readonly" \
    --mount "type=bind,source=$realwork,target=/realwork,readonly" \
    --mount "type=bind,source=$gocache,target=/gocache" \
    --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
    fd2-go-test-local:latest with-xvfb go test ./cmd/fd2 -run '^TestRustGoldenSave$' -count=1 -timeout 25m -v
}
run_go "$out" | tail -5
run_go "$out.rerun" | tail -2
a=$(sha256sum "$golden/$out" | cut -c1-64)
b=$(sha256sum "$golden/$out.rerun" | cut -c1-64)
test "$a" = "$b" || { echo "Go 端兩次輸出不同：$a $b（保留 $golden/$out.rerun 供比對）" >&2; exit 1; }
rm -f "$golden/$out.rerun"
echo "黃金檔 $golden/$out SHA-256 $a（Go 端兩次相同）"
timeout 1800s docker run --rm --network none --memory 6g --cpus 4 --pids-limit 1024 \
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h \
  -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_SAVE="/build/golden/$out" -e FD2_REQUIRE_GOLDEN=1 \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/build/golden,readonly" -w /src \
  fd2-rust:1.99.1-r3 cargo test --offline --locked -p fd2-game --lib game::save::golden_save_test::golden_save_matches_go \
  -- --ignored --exact --nocapture 2>&1 | tee "$golden/$out.rust.log"
# 篩選字串沒有對到測試時 cargo 也回 0；確認黃金比對測試真的跑過。
grep -q 'golden_save_test::golden_save_matches_go ... ok' "$golden/$out.rust.log" ||
  { echo "Rust 端黃金比對測試沒有執行或沒有通過" >&2; exit 1; }
