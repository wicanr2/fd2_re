#!/bin/bash
# fd2-formats 完整檢查（docs/goal/119）：fmt、clippy、全部單元測試；原版檔與分離素材包都掛上，
# 並設 FD2_REQUIRE_ORIGINAL=1，缺檔即失敗，不會悄悄跳過。
# 用法：tools/rust_formats_test.sh [cargo test 的額外參數]
# 環境變數名稱見 remake-rs/crates/fd2-formats/src/test_support.rs，全 crate 統一。
# docs/ 掛在 /docs（remake-rs 的上一層），供讀受版控 IDA 證據的測試使用。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
flame="$root/org_game/炎龍騎士團/FLAME2"
pack="$root/remake/generated-assets/fd2-original-b97caf22"
for dir in "$flame" "$pack" "$root/extracted/raw" "$root/remake/assets"; do
  test -d "$dir" || { echo "找不到 $dir" >&2; exit 1; }
done
timeout 3000s docker run --rm --network none --memory 6g --cpus 4 --pids-limit 1024 \
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h \
  -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target -e FD2_REQUIRE_ORIGINAL=1 \
  -e FD2_FLAME2_DIR=/flame2 -e FD2_ASSET_PACK=/pack -e FD2_EXTRACTED_RAW_DIR=/raw -e FD2_REMAKE_ASSETS=/assets \
  --mount "type=bind,source=$flame,target=/flame2,readonly" \
  --mount "type=bind,source=$pack,target=/pack,readonly" \
  --mount "type=bind,source=$root/extracted/raw,target=/raw,readonly" \
  --mount "type=bind,source=$root/remake/assets,target=/assets,readonly" \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$root/docs,target=/docs,readonly" \
  --mount "type=bind,source=$root/work/rust-build,target=/build" -w /src \
  fd2-rust:1.99.1-r3 bash -euo pipefail -c '
    cargo fmt --check
    cargo clippy --offline --locked -p fd2-formats --all-targets -- -D warnings
    cargo test --offline --locked -p fd2-formats --lib "$@"
  ' _ "$@"
