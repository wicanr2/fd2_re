#!/bin/bash
# R4 階段 0（docs/goal/120）：以凍結 Go 版重播第一章受控流程（ch01-start-flow-r4 對照原版
# original-full-r4），量測 cmd/fd2 實際執行到的函式，並輸出 Go 黃金軌跡 checkpoints.jsonl。
#
# 用法：tools/rust_r4_ch01_coverage.sh <輸出目錄>
# 輸出：<輸出目錄>/replay.log、cover.out、cover-func.txt、parity-out/checkpoints.jsonl
# 輸入與原版收據 docs/data/ui-traces/parity-ch01.json 相同：建構槽 work/ch01-flow-20261017/input/FD2.SAV
# （SHA-256 6d14f2c2…，即未修改的原版 FD2.SAV）、原版輸出 work/ch01-flow-20261017/original-full-r4。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
out="$(mkdir -p "${1:?輸出目錄}" && cd "$1" && pwd -P)"
slot="$root/work/ch01-flow-20261017/input"
oracle="$root/work/ch01-flow-20261017/original-full-r4"
for f in "$slot/FD2.SAV" "$oracle/actions.jsonl" "$oracle/runner.json"; do test -f "$f" || { echo "缺少 $f" >&2; exit 2; }; done
test "$(sha256sum "$slot/FD2.SAV" | cut -c1-64)" = 6d14f2c22562cabca83725084f1a9d6539a1d4066da5c1debcdadb446812691f
mkdir -p "$out/parity-out" "$root/work/gocache"
pack=$(mktemp -d -t fd2-r4cov-pack-XXXXXX)
trap 'rm -rf "$pack"' EXIT
for e in "$root"/remake/generated-assets/fd2-original-b97caf22/*; do
  ln -s "/src/remake/generated-assets/fd2-original-b97caf22/$(basename "$e")" "$pack/$(basename "$e")"
done
ln -s /src/remake/assets/locales "$pack/locales"
timeout 3000 docker run --rm --network none --memory 8g --cpus 4 --pids-limit 512 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
  -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod -e FD2_ASSET_PACK=/pack \
  -e FD2_PARITY_CHAPTER=1 -e FD2_PARITY_SLOT=/slot/FD2.SAV -e FD2_PARITY_ORACLE_RUN=/oracle \
  -e FD2_PARITY_OUT=/parity-out -e FD2_PARITY_ENTRY=start -e FD2_PARITY_CAPTURE_FRAMES=0 \
  -v "$root:/src" -v "$pack:/pack:ro" -v "$root/work/gocache:/gocache" \
  -v "$slot:/slot:ro" -v "$oracle:/oracle:ro" -v "$out/parity-out:/parity-out" -v "$out:/cov" \
  -w /src/remake fd2-go-test-local:latest with-xvfb bash -c '
    go test ./cmd/fd2 -run "^TestChapterParityReplay$" -count=1 -timeout 45m \
      -coverprofile=/cov/cover.out -coverpkg=./cmd/fd2 > /cov/replay.log 2>&1
    status=$?
    go tool cover -func=/cov/cover.out > /cov/cover-func.txt
    exit $status'
echo "完成：$out（checkpoints $(wc -l < "$out/parity-out/checkpoints.jsonl")）"
