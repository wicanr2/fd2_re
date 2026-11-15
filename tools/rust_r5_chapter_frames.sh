#!/bin/bash
# R5 階段 2（docs/goal/121，r5b 的階段 3 預備）：Rust 的 chapter 模式整張畫格擷取（第一章受控重播的每個 pump 幀，
# rust_golden_frames_test.rs），與 R5 階段 0 的 Go 黃金（chapter 模式）逐次比對。
#
# 用法：tools/rust_r5_chapter_frames.sh <輸出目錄>
# 輸出：<輸出目錄>/rust/frames.json（欄位同 Go 黃金；Game::draw 停在未移植的 todo! 時另有 stopped，章重播在那裡停下，
#       這時結束碼不是 0 屬預期）、replay.log、compare.json、compare.txt。
#
# 輸入（唯讀掛載）：
#   建構槽與原版輸出  $FD2_WORK/ch01-flow-20261017/{input,original-full-r4}（同 tools/rust_parity_ch01.sh）
#   Go 黃金          $FD2_GOLDEN_CHAPTER（預設 $FD2_WORK/r5-stage0-20261115/r1/chapter/frames.json，SHA-256 d71aad6a…）
# 環境變數：
#   FD2_WORK／FD2_ASSET_PACK／FD2_REMAKE_ASSETS／FD2_RUST_BUILD／FD2_RUST_TARGET 同 tools/rust_parity_ch01.sh
#   FD2_GOLDEN_FRAMES_DUMP 逗號分隔的擷取序號，Rust 端另把這些擷取的 RGBA 寫到 rust/frames/<序號>.rgba
#   FD2_PYTHON_IMAGE 比對用的映像檔（預設 python:3.13-slim）
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
golden="${FD2_GOLDEN_CHAPTER:-$work/r5-stage0-20261115/r1/chapter/frames.json}"
for d in "$slot" "$oracle" "$packsrc" "$assets" "$build/cargo"; do
  test -d "$d" || { echo "找不到 $d" >&2; exit 2; }
done
for f in "$slot/FD2.SAV" "$oracle/actions.jsonl" "$oracle/runner.json" "$golden"; do
  test -f "$f" || { echo "缺少 $f" >&2; exit 2; }
done
test "$(sha256sum "$slot/FD2.SAV" | cut -c1-64)" = 6d14f2c22562cabca83725084f1a9d6539a1d4066da5c1debcdadb446812691f
golden_dir="$(cd "$(dirname "$golden")" && pwd -P)"
mkdir -p "$out/rust/parity-out" "$build/$target"
rm -rf "$out/rust/frames.json" "$out/rust/frames" "$out/rust/parity-out/"*

# 完整分離素材根（同 tools/rust_parity_ch01.sh）：符號連結根，目標寫成容器內路徑。
pack="$out/pack"
rm -rf "$pack"
mkdir -p "$pack"
trap 'rm -rf "$pack"' EXIT
for e in "$packsrc"/*; do
  ln -s "/packsrc/$(basename "$e")" "$pack/$(basename "$e")"
done
ln -s /assets/locales "$pack/locales"

echo "== Rust chapter 模式擷取"
status=0
timeout 3000s docker run --rm --name "fd2-r5ch-$$" --network none --memory 6g --cpus 4 --pids-limit 1024 \
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h \
  -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  -e FD2_ASSET_PACK=/pack -e FD2_REMAKE_ASSETS=/assets \
  -e FD2_GOLDEN_FRAMES_MODE=chapter -e FD2_GOLDEN_FRAMES_OUT=/frames-out \
  -e "FD2_GOLDEN_FRAMES_DUMP=${FD2_GOLDEN_FRAMES_DUMP:-}" \
  -e FD2_PARITY_CHAPTER=1 -e FD2_PARITY_SLOT=/slot/FD2.SAV -e FD2_PARITY_ORACLE_RUN=/oracle \
  -e FD2_PARITY_OUT=/frames-out/parity-out -e FD2_PARITY_ENTRY=start -e FD2_PARITY_CAPTURE_FRAMES=0 \
  --mount "type=bind,source=$slot,target=/slot,readonly" \
  --mount "type=bind,source=$oracle,target=/oracle,readonly" \
  --mount "type=bind,source=$out/rust,target=/frames-out" \
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
echo "擷取結束（status=$status）"
test -f "$out/rust/frames.json" || { echo "沒有輸出 frames.json，見 $out/replay.log" >&2; exit 1; }

echo "== 與 Go 黃金比對"
cmp=0
timeout 600s docker run --rm --name "fd2-r5ch-cmp-$$" --network none --memory 1g --cpus 1 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp \
  --mount "type=bind,source=$root/tools,target=/repo/tools,readonly" \
  --mount "type=bind,source=$out/rust,target=/rust,readonly" \
  --mount "type=bind,source=$golden_dir,target=/go,readonly" \
  --mount "type=bind,source=$out,target=/out" -w /repo "$pyimage" \
  python3 -I tools/rust_r5_frames_compare.py --rust /rust/frames.json --go "/go/$(basename "$golden")" \
    --out /out/compare.json > "$out/compare.txt" 2>&1 || cmp=$?
cat "$out/compare.txt"
exit "$cmp"
