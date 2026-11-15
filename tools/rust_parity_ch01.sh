#!/bin/bash
# R4 階段 3（docs/goal/120「驗證」第 2、3 層）：Rust 無畫面重播第一章受控流程，與凍結 Go 版黃金逐位元組比對，
# 再以 tools/verify_chapter_parity.py 對兩份輸出判定原版閘門。
#
# 用法：tools/rust_parity_ch01.sh <輸出目錄>
# 輸出：<輸出目錄>/rust/（checkpoints.jsonl、FD2.SAV、hotel-save.json、replay-settings.json、start-entry.json）、
#       replay.log、compare.txt；收據寫到 $FD2_RUST_BUILD/golden/r4i-<輸出目錄名>-{rust,go}.json。
#
# 輸入（唯讀掛載，同 tools/rust_r4_ch01_coverage.sh 與收據 docs/data/ui-traces/parity-ch01.json）：
#   建構槽   $FD2_WORK/ch01-flow-20261017/input/FD2.SAV（SHA-256 6d14f2c2…）
#   原版輸出 $FD2_WORK/ch01-flow-20261017/original-full-r4（原版實際 SAV 在 state-full-r4/FD2.SAV）
#   Go 黃金  $FD2_WORK/r4-stage0-20261114/parity-out（checkpoints.jsonl SHA-256 1c846f57…）
# 環境變數：
#   FD2_WORK        work/ 目錄（預設本工作區的 work；在工作樹副本中執行時指向主工作區的）
#   FD2_ASSET_PACK／FD2_REMAKE_ASSETS／FD2_RUST_BUILD／FD2_RUST_TARGET 同 tools/rust_game_test.sh
#   FD2_PYTHON_IMAGE 判定與比對用的映像檔（預設 python:3.13-slim；verify_chapter_parity.py 的 PIL 是選用）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
# R5 r5f：Go 黃金用的 TTF 字型（缺檔或雜湊不符即失敗），見 tools/rust_ttf_font.sh。
source "$root/tools/rust_ttf_font.sh"
out="$(mkdir -p "${1:?輸出目錄}" && cd "$1" && pwd -P)"
work="${FD2_WORK:-$root/work}"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
target="${FD2_RUST_TARGET:-target}"
packsrc="${FD2_ASSET_PACK:-$root/remake/generated-assets/fd2-original-b97caf22}"
assets="${FD2_REMAKE_ASSETS:-$root/remake/assets}"
pyimage="${FD2_PYTHON_IMAGE:-python:3.13-slim}"
slot="$work/ch01-flow-20261017/input"
oracle="$work/ch01-flow-20261017/original-full-r4"
oracle_state="$work/ch01-flow-20261017/state-full-r4"
golden="$work/r4-stage0-20261114/parity-out"
for d in "$slot" "$oracle" "$oracle_state" "$golden" "$packsrc" "$assets" "$build/cargo"; do
  test -d "$d" || { echo "找不到 $d" >&2; exit 2; }
done
for f in "$slot/FD2.SAV" "$oracle/actions.jsonl" "$oracle/runner.json" "$oracle_state/FD2.SAV" \
         "$golden/checkpoints.jsonl" "$golden/FD2.SAV" "$golden/hotel-save.json"; do
  test -f "$f" || { echo "缺少 $f" >&2; exit 2; }
done
test "$(sha256sum "$slot/FD2.SAV" | cut -c1-64)" = 6d14f2c22562cabca83725084f1a9d6539a1d4066da5c1debcdadb446812691f
test "$(sha256sum "$golden/checkpoints.jsonl" | cut -c1-64)" = 1c846f573434658cabb3cfa486edd5826b401673c3bff411e30cbdc3e18ac0cc
mkdir -p "$out/rust" "$build/$target" "$build/golden"
rm -f "$out/rust/"*

# 完整分離素材根（同 tools/rust_r4_ch01_coverage.sh）：符號連結根，目標寫成容器內路徑。
pack="$out/pack"
rm -rf "$pack"
mkdir -p "$pack"
trap 'rm -rf "$pack"' EXIT
for e in "$packsrc"/*; do
  ln -s "/packsrc/$(basename "$e")" "$pack/$(basename "$e")"
done
ln -s /assets/locales "$pack/locales"

echo "== Rust 章重播（第一章 START）"
status=0
timeout 3000s docker run --rm --network none --memory 6g --cpus 4 --pids-limit 1024 \
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h \
  -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_ASSET_PACK=/pack -e FD2_REMAKE_ASSETS=/assets \
  -e FD2_PARITY_CHAPTER=1 -e FD2_PARITY_SLOT=/slot/FD2.SAV -e FD2_PARITY_ORACLE_RUN=/oracle \
  -e FD2_PARITY_OUT=/parity-out -e FD2_PARITY_ENTRY=start -e FD2_PARITY_CAPTURE_FRAMES=0 \
  --mount "type=bind,source=$slot,target=/slot,readonly" \
  --mount "type=bind,source=$oracle,target=/oracle,readonly" \
  --mount "type=bind,source=$out/rust,target=/parity-out" \
  --mount "type=bind,source=$pack,target=/pack,readonly" \
  --mount "type=bind,source=$packsrc,target=/packsrc,readonly" \
  --mount "type=bind,source=$assets,target=/assets,readonly" \
  "${ttf_mounts[@]}" \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$root/docs,target=/docs,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" -w /src \
  fd2-rust:1.99.1-r3 bash -euo pipefail -c '
    cargo test --offline --locked -p fd2-game --release --lib -- --ignored --exact --nocapture \
      game::parity_replay::chapter_parity_replay_test::test_chapter_parity_replay
  ' > "$out/replay.log" 2>&1 || status=$?
tail -5 "$out/replay.log"
echo "重播結束（status=$status）"

tag="$(basename "$out")"
echo "== verify_chapter_parity.py（Rust 與 Go 各一份）"
for side in rust go; do
  if [ "$side" = rust ]; then remake="$out/rust"; else remake="$golden"; fi
  timeout 600s docker run --rm --network none --memory 2g --cpus 1 --pids-limit 128 \
    --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp \
    --mount "type=bind,source=$root/tools,target=/repo/tools,readonly" \
    --mount "type=bind,source=$root/docs,target=/repo/docs,readonly" \
    --mount "type=bind,source=$oracle,target=/oracle,readonly" \
    --mount "type=bind,source=$oracle_state,target=/oracle-state,readonly" \
    --mount "type=bind,source=$remake,target=/parity-out,readonly" \
    --mount "type=bind,source=$build/golden,target=/receipts" -w /repo "$pyimage" \
    python3 tools/verify_chapter_parity.py --chapter 1 --oracle /oracle --remake /parity-out \
      --slot-manifest docs/data/parity-slots/ch01-start-manifest.json \
      --plan docs/data/parity-plans/ch01-start-flow-r4.jsonl \
      --out "/receipts/r4i-$tag-$side.json" --frame-policy advisory \
      --save-policy ch01-start-residual-20261017 --oracle-save /oracle-state/FD2.SAV \
      > "$out/verify-$side.log" 2>&1 || echo "verify（$side）結束碼 $?（收據仍可能寫出）"
  tail -3 "$out/verify-$side.log"
done

echo "== 比對"
cmp=0
timeout 600s docker run --rm --network none --memory 1g --cpus 1 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp \
  --mount "type=bind,source=$root/tools,target=/repo/tools,readonly" \
  --mount "type=bind,source=$out/rust,target=/rust,readonly" \
  --mount "type=bind,source=$golden,target=/go,readonly" \
  --mount "type=bind,source=$build/golden,target=/receipts,readonly" -w /repo "$pyimage" \
  python3 -I tools/rust_parity_compare.py --rust /rust --go /go \
    --receipt-rust "/receipts/r4i-$tag-rust.json" --receipt-go "/receipts/r4i-$tag-go.json" \
  > "$out/compare.txt" 2>&1 || cmp=$?
cat "$out/compare.txt"
if [ "$status" -ne 0 ]; then exit "$status"; fi
exit "$cmp"
