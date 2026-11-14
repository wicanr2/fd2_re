#!/bin/bash
# fd2-campaign 完整檢查（docs/goal/119）：fmt、clippy、全部單元測試；原版檔與分離素材包都掛上，
# 並設 FD2_REQUIRE_ORIGINAL=1，缺檔即失敗，不會悄悄跳過。
# 用法：tools/rust_rules_test.sh [cargo test 的額外參數]
# 環境變數名稱見 remake-rs/crates/fd2-campaign/src/test_support.rs，與 fd2-formats 相同。
#   FD2_FLAME2_DIR／FD2_ASSET_PACK／FD2_EXTRACTED_RAW_DIR 指定原版輸入（預設在本工作區底下）
#   FD2_REMAKE_ASSETS remake/assets（預設本工作區那份）。部分檔案（例如 `map0_units.json`）由
#                   tools/ 從原版產生、不入庫，工作樹副本沒有；在副本中執行時指向主工作區的
#                   remake/assets（受版控的檔案兩邊相同）
#   FD2_RUST_BUILD  建置目錄（預設 work/rust-build；在工作樹副本中執行時指向主工作區的）
#   FD2_RUST_TARGET 建置目錄下的 cargo target 目錄名稱（預設 target）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
target="${FD2_RUST_TARGET:-target}"
flame="${FD2_FLAME2_DIR:-$root/org_game/炎龍騎士團/FLAME2}"
pack="${FD2_ASSET_PACK:-$root/remake/generated-assets/fd2-original-b97caf22}"
raw="${FD2_EXTRACTED_RAW_DIR:-$root/extracted/raw}"
assets="${FD2_REMAKE_ASSETS:-$root/remake/assets}"
for dir in "$flame" "$pack" "$raw" "$assets" "$build/cargo"; do
  test -d "$dir" || { echo "找不到 $dir" >&2; exit 1; }
done
mkdir -p "$build/$target"
timeout 3000s docker run --rm --network none --memory 6g --cpus 4 --pids-limit 1024 \
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h \
  -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x -e FD2_REQUIRE_ORIGINAL=1 \
  -e FD2_FLAME2_DIR=/flame2 -e FD2_ASSET_PACK=/pack -e FD2_EXTRACTED_RAW_DIR=/raw -e FD2_REMAKE_ASSETS=/assets \
  --mount "type=bind,source=$flame,target=/flame2,readonly" \
  --mount "type=bind,source=$pack,target=/pack,readonly" \
  --mount "type=bind,source=$raw,target=/raw,readonly" \
  --mount "type=bind,source=$assets,target=/assets,readonly" \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$root/docs,target=/docs,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" -w /src \
  fd2-rust:1.99.1-r3 bash -euo pipefail -c '
    cargo fmt --check
    cargo clippy --offline --locked -p fd2-campaign --all-targets -- -D warnings
    cargo test --offline --locked -p fd2-campaign --lib "$@"
  ' _ "$@"
