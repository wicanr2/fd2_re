#!/bin/bash
# R5 階段 1（docs/goal/121）：重新產生 fd2-game CPU 畫布測試向量
# remake-rs/crates/fd2-game/src/game/canvas/ebiten_vectors.json。
#
# 用法：tools/rust_r5_canvas_vectors.sh
# 輸入（唯讀）：<資料根>/work/r5-stage0-20261115/rounding/ebiten-rounding.json（tools/rust_r5_golden_frames.sh）、
#              <資料根>/work/r5-stage1-20261115/sampling/r1/ebiten-sampling.json（tools/rust_r5_ebiten_sampling.sh）。
# 環境變數：FD2_DATA_ROOT 資料根（預設本儲存庫；在工作樹副本中執行時指向主工作區）。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
data="$(cd "${FD2_DATA_ROOT:-$root}" && pwd -P)"
r0="$data/work/r5-stage0-20261115/rounding"
r1="$data/work/r5-stage1-20261115/sampling/r1"
outdir="$root/remake-rs/crates/fd2-game/src/game/canvas"
for d in "$r0" "$r1" "$outdir"; do test -d "$d" || { echo "找不到目錄 $d" >&2; exit 2; }; done
for f in "$r0/ebiten-rounding.json" "$r1/ebiten-sampling.json" "$root/tools/rust_r5_canvas_vectors.py"; do
  test -f "$f" || { echo "缺少 $f" >&2; exit 2; }
done
timeout 300 docker run --rm --name "fd2-r5a-vectors-$$" --network none --memory 1g --cpus 1 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
  --mount "type=bind,source=$root/tools,target=/tools,readonly" \
  --mount "type=bind,source=$r0,target=/r0,readonly" \
  --mount "type=bind,source=$r1,target=/r1,readonly" \
  --mount "type=bind,source=$outdir,target=/out" \
  python:3.13-slim python3 -I /tools/rust_r5_canvas_vectors.py /r0/ebiten-rounding.json /r1/ebiten-sampling.json /out/ebiten_vectors.json
sha256sum "$outdir/ebiten_vectors.json"
