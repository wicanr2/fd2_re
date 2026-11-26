#!/bin/bash
# R8 逐章受控重播（docs/goal/119「階段」R8）：Rust 無畫面以標題 LOAD 讀入章界建構槽，依 dosgolem 原版
# 收據的 actions.jsonl／control-history.jsonl 重播，逐筆核對原版檢查點。第一章 START 走 rust_parity_ch01.sh。
#
# 用法：tools/rust_parity_chapter.sh <章號 2..30> <建構槽 FD2.SAV> <原版執行目錄> <輸出目錄>
# 輸出：<輸出目錄>/rust/（checkpoints.jsonl、FD2.SAV 複本與重播設定）、<輸出目錄>/replay.log。
# 建構槽與原版執行目錄唯讀掛載；建構槽在輸出目錄另存可寫複本（測試本身負責）。
# 環境變數：FD2_ASSET_PACK／FD2_REMAKE_ASSETS／FD2_RUST_BUILD／FD2_RUST_TARGET 同 tools/rust_game_test.sh；
#   FD2_PARITY_AI_DIAGNOSTICS 原樣傳給測試。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
source "$root/tools/rust_ttf_font.sh"
chapter="${1:?章號}"
[[ "$chapter" =~ ^([2-9]|[12][0-9]|30)$ ]] || { echo "章號須為 2..30（第一章用 rust_parity_ch01.sh）" >&2; exit 2; }
slot_file="$(cd "$(dirname "${2:?建構槽 FD2.SAV}")" && pwd -P)/$(basename "$2")"
oracle="$(cd "${3:?原版執行目錄}" && pwd -P)"
out="$(mkdir -p "${4:?輸出目錄}" && cd "$4" && pwd -P)"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
target="${FD2_RUST_TARGET:-target}"
packsrc="${FD2_ASSET_PACK:-$root/remake/generated-assets/fd2-original-b97caf22}"
assets="${FD2_REMAKE_ASSETS:-$root/remake/assets}"
test -f "$slot_file" || { echo "缺少 $slot_file" >&2; exit 2; }
for f in actions.jsonl control-history.jsonl runner.json; do
  test -f "$oracle/$f" || { echo "缺少 $oracle/$f" >&2; exit 2; }
done
# 重播在每個 AI 計畫與升級擲骰前抄原版 0x13A9F／0x1E54A 的 rng_word；缺追蹤時測試會改用重製端自己的亂數，
# 照樣通過卻不再是對拍。原版要以 FD2_ORACLE_EIP_TRACE=0x13A9F,0x1E54A 執行。
trace="$oracle/eip-trace.jsonl"
if [ ! -f "$trace" ] && [ -f "$trace.gz" ]; then
  echo "原版追蹤只有 $trace.gz，請先解到暫存原版樣本副本" >&2; exit 2
fi
test -f "$trace" && grep -q '"0x13A9F"' "$trace" \
  || { echo "缺少含 0x13A9F 的 $trace：原版要以 FD2_ORACLE_EIP_TRACE=0x13A9F,0x1E54A 執行" >&2; exit 2; }
echo "原版追蹤：0x13A9F $(grep -c '"0x13A9F"' "$trace") 筆、0x1E54A $(grep -c '"0x1E54A"' "$trace" || true) 筆"
for d in "$packsrc" "$assets" "$build/cargo"; do
  test -d "$d" || { echo "找不到 $d" >&2; exit 2; }
done
slot_dir="$out/slot"
mkdir -p "$out/rust" "$slot_dir" "$build/$target"
rm -f "$out/rust/"*
cp "$slot_file" "$slot_dir/FD2.SAV"
echo "建構槽 SHA-256 $(sha256sum "$slot_dir/FD2.SAV" | cut -c1-64)"

# 完整分離素材根（同 tools/rust_parity_ch01.sh）：符號連結根，目標寫成容器內路徑。
pack="$out/pack"
rm -rf "$pack"
mkdir -p "$pack"
trap 'rm -rf "$pack"' EXIT
for e in "$packsrc"/*; do
  ln -s "/packsrc/$(basename "$e")" "$pack/$(basename "$e")"
done
ln -s /assets/locales "$pack/locales"

status=0
timeout 3600s docker run --rm --network none --memory 6g --cpus 4 --pids-limit 1024 \
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h \
  -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_ASSET_PACK=/pack -e FD2_REMAKE_ASSETS=/assets \
  -e FD2_PARITY_CHAPTER="$chapter" -e FD2_PARITY_SLOT=/slot/FD2.SAV -e FD2_PARITY_ORACLE_RUN=/oracle \
  -e FD2_PARITY_OUT=/parity-out -e FD2_PARITY_ENTRY=load -e FD2_PARITY_CAPTURE_FRAMES=0 \
  -e FD2_PARITY_AI_DIAGNOSTICS="${FD2_PARITY_AI_DIAGNOSTICS:-}" \
  --mount "type=bind,source=$slot_dir,target=/slot,readonly" \
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
tail -8 "$out/replay.log"
echo "重播結束（status=$status）"
exit "$status"
