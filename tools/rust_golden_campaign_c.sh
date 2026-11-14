#!/bin/bash
# fd2-campaign C 組（存檔、CONTINUE、JOIN 與結局演出）的 Go→Rust 黃金差分（docs/goal/119）。
# Go 端用凍結版工具產生黃金 JSON，Rust 端 tests/golden_campaign_c.rs 逐欄比對。
# 用法：tools/rust_golden_campaign_c.sh
#   FD2_RUST_BUILD   建置目錄（預設 work/rust-build；在工作樹副本中執行時指向主工作區的）
#   FD2_RUST_TARGET  建置目錄下的 cargo target 目錄名稱（預設 target）
#   FD2_FLAME2_DIR／FD2_ASSET_PACK 原版 FLAME2 與分離素材包（預設在本工作區底下）
#   FD2_GOLDEN_CAMPAIGN_C_NAME 黃金檔名（預設 campaign-c-v1.json；竄改反向檢查時指向改過的複本）
#   FD2_GOLDEN_SKIP_GO=1 不重產黃金檔，只跑 Rust 端
# 真實輸入：受版控的 remake/assets 與 docs/data/glyph_map.json、原版 FD2.SAV／FDOTHER.DAT／ANI.DAT 與
# 分離素材包，全部唯讀掛載；黃金檔只記檔案標籤、大小與 SHA-256，Rust 端讀同一批檔案前先核對。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
golden="$build/golden"
target="${FD2_RUST_TARGET:-target}"
flame="${FD2_FLAME2_DIR:-$root/org_game/炎龍騎士團/FLAME2}"
pack="${FD2_ASSET_PACK:-$root/remake/generated-assets/fd2-original-b97caf22}"
out="${FD2_GOLDEN_CAMPAIGN_C_NAME:-campaign-c-v1.json}"
for dir in "$flame" "$pack" "$root/remake/assets" "$root/docs/data"; do
  test -d "$dir" || { echo "找不到 $dir" >&2; exit 1; }
done
for f in FD2.SAV FDOTHER.DAT ANI.DAT; do
  test -f "$flame/$f" || { echo "找不到 $flame/$f" >&2; exit 1; }
done
mkdir -p "$golden" "$build/cargo" "$build/$target"
limits=(--rm --network none --memory 6g --cpus 4 --pids-limit 1024
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h)
if [ "${FD2_GOLDEN_SKIP_GO:-0}" != 1 ]; then
  timeout 900s docker run "${limits[@]}" -e GOCACHE=/tmp/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off \
    --mount "type=bind,source=$root,target=/repo,readonly" \
    --mount "type=bind,source=$flame,target=/flame2,readonly" \
    --mount "type=bind,source=$pack,target=/pack,readonly" \
    --mount "type=bind,source=$golden,target=/golden" -w /repo/remake \
    fd2-build-mingw:1.22.12-r1 go run ./cmd/fd2-golden-campaign-c -out "/golden/$out" -flame2 /flame2 -pack /pack
fi
sha256sum "$golden/$out"
timeout 1500s docker run "${limits[@]}" -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_GOLDEN_CAMPAIGN_C="/build/golden/$out" -e FD2_REQUIRE_GOLDEN=1 -e FD2_GOLDEN_REPO=/repo \
  -e FD2_GOLDEN_FLAME2=/flame2 -e FD2_GOLDEN_PACK=/pack \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$root/remake/assets,target=/repo/remake/assets,readonly" \
  --mount "type=bind,source=$root/docs/data,target=/repo/docs/data,readonly" \
  --mount "type=bind,source=$flame,target=/flame2,readonly" \
  --mount "type=bind,source=$pack,target=/pack,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" \
  --mount "type=bind,source=$golden,target=/build/golden,readonly" -w /src \
  fd2-rust:1.99.1-r3 cargo test --offline --locked -p fd2-campaign --test golden_campaign_c -- --nocapture
