#!/bin/bash
# 開場與天空之鑰清冊、按需解碼和 Go 像素差分。HD 圖包與原版分離素材唯讀掛載。
# 用法：bash tools/rust_r6_opening.sh <輸出目錄>
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
out="$(mkdir -p "${1:?輸出目錄}" && cd "$1" && pwd -P)"
hd="${FD2_MODERN_THEME_PACK:-$root/../fd2-assets-private/packs/fd2-modern-handpainted-a}"
original="${FD2_ASSET_PACK:-$root/remake/generated-assets/fd2-original-b97caf22}"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
target="${FD2_RUST_TARGET:-target}"
gocache="${FD2_GOCACHE:-$root/work/gocache-r5i}"
for dir in "$hd" "$original" "$build/cargo" "$gocache" "$root/remake" "$root/remake-rs" "$out"; do
  test -d "$dir" || { echo "找不到 $dir" >&2; exit 2; }
done
test -f "$root/remake/assets/themes/modern/catalog.json"
test -f "$root/LICENSE"
mkdir -p "$build/$target"
hd="$(cd "$hd" && pwd -P)"
common=(--rm --network none --memory 6g --cpus 4 --pids-limit 512 --user "$(id -u):$(id -g)"
  --mount "type=bind,source=$hd,target=/hd,readonly"
  --mount "type=bind,source=$original,target=/original,readonly"
  --mount "type=bind,source=$root/remake/assets/themes/modern,target=/catalog,readonly"
  --mount "type=bind,source=$out,target=/out"
  -e FD2_TEST_OPENING_CATALOG=/catalog/catalog.json -e FD2_TEST_OPENING_PACK=/hd
  -e FD2_ASSET_PACK=/original -e FD2_R6_OPENING_OUT=/out)

timeout 900s docker run "${common[@]}" --name "fd2-r6-opening-go-$$" --init \
  --mount "type=bind,source=$root/remake,target=/src,readonly" \
  --mount "type=bind,source=$gocache,target=/gocache" -e GOCACHE=/gocache -w /src \
  fd2-go-test-local:latest with-xvfb bash -euo pipefail -c '
    test -z "$(gofmt -l cmd/fd2/rust_golden_r6_opening_test.go)"
    go test ./cmd/fd2 -run "^(TestModernHDOpeningAdmission|TestRustGoldenR6Opening|TestRustGoldenR6SkyKey)$" -count=1 -v
  ' 2>&1 | tee "$out/go.log"

timeout 900s docker run "${common[@]}" --name "fd2-r6-opening-rust-$$" \
  --mount "type=bind,source=$root/remake-rs,target=/src,readonly" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" \
  -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x -w /src \
  fd2-rust:1.99.1-r3 bash -euo pipefail -c '
    cargo test --offline --locked -p fd2-game --lib -- game::modern_opening::tests game::modern_hd_sky_key_assets::tests
    cargo test --offline --locked -p fd2-game --lib -- --ignored --nocapture real_opening_matches_go real_sky_key_matches_go
  ' 2>&1 | tee "$out/rust.log"
