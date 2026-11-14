#!/bin/bash
# fd2-campaign A 組（劇本資料與處理器編譯）與 editorcanonical 的 Go→Rust 黃金差分（docs/goal/119）。
# Go 端用凍結版工具讀全部 remake/assets 劇本產生黃金 JSON，Rust 端 tests/golden_campaign_a.rs 逐欄比對。
# 用法：tools/rust_golden_campaign_a.sh
#   FD2_RUST_BUILD    建置目錄（預設 work/rust-build；在工作樹副本中執行時指向主工作區的）
#   FD2_RUST_TARGET   建置目錄下的 cargo target 目錄名稱（預設 target）
#   FD2_REMAKE_ASSETS remake/assets（預設本工作區那份）。bindings/generated-* 等產生檔不入庫，工作樹副本
#                     沒有；在副本中執行時指向主工作區的 remake/assets（受版控的檔案兩邊相同）
# 兩端都把素材唯讀掛在 /assets：錯誤訊息裡的路徑字樣因此相同。黃金檔記每個讀到的檔案的相對路徑、
# 大小與 SHA-256，Rust 端讀同一份檔案前先核對。Go 端跑兩次，確認輸出位元組相同。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
golden="$build/golden"
target="${FD2_RUST_TARGET:-target}"
assets="${FD2_REMAKE_ASSETS:-$root/remake/assets}"
test -d "$assets/cutscenes" || { echo "找不到 $assets/cutscenes" >&2; exit 1; }
mkdir -p "$golden" "$build/cargo" "$build/$target"
limits=(--rm --network none --memory 6g --cpus 4 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
for run in 1 2; do
  timeout 900s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
    --mount "type=bind,source=$root,target=/repo,readonly" \
    --mount "type=bind,source=$assets,target=/assets,readonly" \
    --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
    fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-campaign-a -assets /assets -out "/golden/campaign-a-v1.run$run.json"
done
sha256sum "$golden/campaign-a-v1.run1.json" "$golden/campaign-a-v1.run2.json"
cmp "$golden/campaign-a-v1.run1.json" "$golden/campaign-a-v1.run2.json" || { echo "Go 端兩次輸出不同" >&2; exit 1; }
mv "$golden/campaign-a-v1.run1.json" "$golden/campaign-a-v1.json"
rm "$golden/campaign-a-v1.run2.json"
timeout 1500s docker run "${limits[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_CAMPAIGN_A=/build/golden/campaign-a-v1.json -e FD2_REQUIRE_GOLDEN=1 -e FD2_GOLDEN_ASSETS=/assets \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$assets,target=/assets,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/build/golden,readonly" -w /src \
  fd2-rust:1.99.1-r3 cargo test --offline --locked -p fd2-campaign --test golden_campaign_a -- --nocapture
