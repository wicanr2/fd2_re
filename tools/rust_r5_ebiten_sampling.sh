#!/bin/bash
# R5 階段 1（docs/goal/121）：實測 Ebiten 在 Xvfb／llvmpipe 下的最近鄰取樣（非整數倍縮放、小數平移）與
# Fill、SubImage、Set、NewImageFromImage 語意。跑兩次並比對輸出雜湊，結果供 fd2-game 的 CPU 畫布測試。
#
# 用法：tools/rust_r5_ebiten_sampling.sh <輸出目錄>
# 產出：<輸出目錄>/r1/ebiten-sampling.json、r2/ebiten-sampling.json、各自的 sampling.log。
#
# 只執行 remake/cmd/fd2-r5-ebiten-rounding（R5 新增的黃金工具）；既有 Go 檔不改。環境與階段 0 相同：
# fd2-go-test-local（Go 1.22.12、Ebiten v2.6.6）、with-xvfb、--network none。
#
# 環境變數：
#   FD2_GOCACHE  Go 建置快取（預設 <本儲存庫>/work/gocache；在工作樹副本中執行時指向主工作區的）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
out="$(mkdir -p "${1:?輸出目錄}" && cd "$1" && pwd -P)"
gocache="${FD2_GOCACHE:-$root/work/gocache}"
for d in "$root/remake/cmd/fd2-r5-ebiten-rounding" "$gocache"; do
  test -d "$d" || { echo "找不到目錄 $d" >&2; exit 2; }
done
for run in r1 r2; do
  rd="$out/$run"
  rm -rf "$rd"
  mkdir -p "$rd"
  timeout 1200 docker run --rm --name "fd2-r5a-sampling-$run-$$" --network none --memory 4g --cpus 4 --pids-limit 512 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
    -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod \
    --mount "type=bind,source=$root/remake,target=/src/remake,readonly" \
    --mount "type=bind,source=$gocache,target=/gocache" \
    --mount "type=bind,source=$rd,target=/run" \
    -w /src/remake fd2-go-test-local:latest with-xvfb bash -c '
      set -euo pipefail
      test -z "$(gofmt -l cmd/fd2-r5-ebiten-rounding)" || { gofmt -l cmd/fd2-r5-ebiten-rounding; exit 1; }
      go vet ./cmd/fd2-r5-ebiten-rounding
      go run ./cmd/fd2-r5-ebiten-rounding -sampling-out /run/ebiten-sampling.json > /run/sampling.log 2>&1
      dpkg-query -W -f "\${Package} \${Version}\n" libgl1-mesa-dri libglx-mesa0 xvfb > /run/mesa-packages.txt 2>/dev/null || true' \
    || { echo "$run 失敗，見 $rd/sampling.log" >&2; cat "$rd/sampling.log" >&2 || true; exit 1; }
  echo "$run ebiten-sampling.json $(sha256sum "$rd/ebiten-sampling.json" | cut -c1-64)"
done
if cmp -s "$out/r1/ebiten-sampling.json" "$out/r2/ebiten-sampling.json"; then echo "兩次輸出相同"; else echo "兩次輸出不同" >&2; exit 1; fi
grep -E '^(SAMPLING|SEMANTICS) ' "$out/r1/sampling.log" | grep -E '全部|與預期' || true
