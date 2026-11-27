#!/usr/bin/env bash
# 量測 Rust 版 HD 主題載入的秒數與記憶體（player_theme_loader_measure_test.rs）。
# 用 release 組態；每項測試各起一個程序，VmHWM 才只反映那一項。
#
# 用法：tools/rust_hd_load_measure.sh <輸出目錄> [測試名…]
# HD 包預設是私人庫的 packs/fd2-modern-handpainted-a，可用 FD2_MODERN_THEME_PACK 改。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
source "$root/tools/rust_ttf_font.sh"
out="${1:?需要輸出目錄}"
shift
tests=("$@")
if [ ${#tests[@]} -eq 0 ]; then
  tests=(measure_theme_load_phases measure_theme_load_whole)
fi
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
target="${FD2_RUST_TARGET:-target-measure}"
pack="${FD2_ASSET_PACK:-$root/remake/generated-assets/fd2-original-b97caf22}"
assets="${FD2_REMAKE_ASSETS:-$root/remake/assets}"
hd="${FD2_MODERN_THEME_PACK:-$root/../fd2-assets-private/packs/fd2-modern-handpainted-a}"
for dir in "$pack" "$assets" "$hd" "$build/cargo"; do
  test -d "$dir" || { echo "找不到 $dir" >&2; exit 1; }
done
mkdir -p "$build/$target" "$out"
for name in "${tests[@]}"; do
  timeout 3000s docker run --rm --network none --memory 6g --cpus 4 --pids-limit 1024 \
    --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h \
    -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
    -e FD2_ASSET_PACK=/pack -e FD2_REMAKE_ASSETS=/assets -e FD2_MODERN_THEME_PACK=/hd \
    -e FD2_RUST_HD_LOAD_MEASURE=1 \
    --mount "type=bind,source=$pack,target=/pack,readonly" \
    --mount "type=bind,source=$assets,target=/assets,readonly" \
    --mount "type=bind,source=$hd,target=/hd,readonly" \
    "${ttf_mounts[@]}" \
    --mount "type=bind,source=$root/remake-rs,target=/src" \
    --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
    --mount "type=bind,source=$root/docs,target=/docs,readonly" \
    --mount "type=bind,source=$build/cargo,target=/build/cargo" \
    --mount "type=bind,source=$build/$target,target=/build/target-x" -w /src \
    fd2-rust:1.99.1-r3 \
    cargo test --offline --locked --release -p fd2-game --lib "$name" -- --nocapture --test-threads=1 \
    > "$out/$name.log" 2>&1 || { echo "$name 失敗，見 $out/$name.log" >&2; exit 1; }
  grep '^\[measure\]' "$out/$name.log" | sed "s/^/$name /"
done
