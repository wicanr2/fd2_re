#!/bin/bash
# R5 階段 0（docs/goal/121）：以凍結 Go 版產生 640×400 黃金畫格、量測原版模式的繪圖函式，
# 並實測 Ebiten 的 alpha 與 ColorScale 取整。只執行新增的資料工具，不改任何既有 Go 檔。
#
# 用法：tools/rust_r5_golden_frames.sh <輸出目錄>
#
# 步驟（全部在 fd2-go-test-local 映像內、--network none、Xvfb）：
#   1. r1、r2 各跑一次：
#      - opening：TestRustGoldenFrames（FD2_GOLDEN_FRAMES_MODE=opening），標題開場 → START → 序章 → battle_ch01；
#      - chapter：TestRustGoldenFrames（FD2_GOLDEN_FRAMES_MODE=chapter），內部呼叫 TestChapterParityReplay，
#        輸入與 tools/rust_r4_ch01_coverage.sh 相同；
#      兩者都開 -covermode=count 的覆蓋率。
#   2. 比對：chapter 的 checkpoints.jsonl 必須是 R4 的 Go 黃金 SHA-256；r1／r2 的 frames.json 序列雜湊與
#      每張 PNG 必須相同。
#   3. cmd/fd2-r5-ebiten-rounding：半透明 Fill＋DrawImage 與 ColorScale 的 GPU 讀回對候選公式。
#   4. tools/rust_r5_draw_inventory：以 go/types 建 Game.Draw 的靜態呼叫樹，與覆蓋率交集，輸出
#      executed-draw-functions.json 與 draw-inventory.json。
#
# 輸入（唯讀）：本儲存庫的 remake/；資料根（FD2_DATA_ROOT，預設本儲存庫；在工作樹副本中執行時指向主工作區）
# 的 work/ch01-flow-20261017/input/FD2.SAV（SHA-256 6d14f2c2…）、work/ch01-flow-20261017/original-full-r4、
# remake/generated-assets/fd2-original-b97caf22。
#
# 環境變數：
#   FD2_DATA_ROOT   建構槽、原版輸出與原版素材包所在的儲存庫根（預設本儲存庫）
#   FD2_GOCACHE     Go 建置快取（預設 <資料根>/work/gocache）
#   FD2_R5_RUNS     要跑的輪次（預設 "r1 r2"）
#   FD2_R5_MODES    要跑的模式（預設 "opening chapter"；只供除錯，正式輸出兩者都要）
#   FD2_R5_SKIP_ANALYSIS=1  只跑畫格，略過步驟 3、4
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
data="$(cd "${FD2_DATA_ROOT:-$root}" && pwd -P)"
out="$(mkdir -p "${1:?輸出目錄}" && cd "$1" && pwd -P)"
gocache="${FD2_GOCACHE:-$data/work/gocache}"
runs="${FD2_R5_RUNS:-r1 r2}"
modes="${FD2_R5_MODES:-opening chapter}"
slot="$data/work/ch01-flow-20261017/input"
oracle="$data/work/ch01-flow-20261017/original-full-r4"
assets="$data/remake/generated-assets/fd2-original-b97caf22"
golden=1c846f573434658cabb3cfa486edd5826b401673c3bff411e30cbdc3e18ac0cc
for d in "$root/remake/cmd/fd2" "$root/remake/assets/locales" "$data/remake/assets" "$slot" "$oracle" "$assets" "$gocache"; do
  test -d "$d" || { echo "找不到目錄 $d" >&2; exit 2; }
done
for f in "$slot/FD2.SAV" "$oracle/actions.jsonl" "$oracle/runner.json"; do test -f "$f" || { echo "缺少 $f" >&2; exit 2; }; done
test "$(sha256sum "$slot/FD2.SAV" | cut -c1-64)" = 6d14f2c22562cabca83725084f1a9d6539a1d4066da5c1debcdadb446812691f

pack=$(mktemp -d -t fd2-r5gf-pack-XXXXXX)
trap 'rm -rf "$pack"' EXIT
for e in "$assets"/*; do ln -s "/assets/$(basename "$e")" "$pack/$(basename "$e")"; done
ln -s /src/remake/assets/locales "$pack/locales"

# remake/assets 的產生檔（map.json、tileset.png、sprites 等）不進版控；在容器內把本儲存庫的 remake/ 複製到
# /tmp/src，再以 cp -n 從資料根的 remake/assets（唯讀）補上缺的檔（已有的受版控檔不覆蓋），原始碼本身不改。
prelude='set -e; mkdir -p /tmp/src; cp -a /src/remake /tmp/src/remake
cp -a -n /mainassets/. /tmp/src/remake/assets/
cd /tmp/src/remake; set +e'
docker_go() { # docker_go <名稱> <timeout 秒> <額外 docker 參數...> -- <bash 命令>
  local name="$1" limit="$2"; shift 2
  local extra=()
  while [ "$1" != "--" ]; do extra+=("$1"); shift; done
  shift
  timeout "$limit" docker run --rm --name "fd2-r5z-$name-$$" --network none --memory 8g --cpus 4 --pids-limit 512 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
    -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod \
    --mount "type=bind,source=$root,target=/src,readonly" \
    --mount "type=bind,source=$assets,target=/assets,readonly" \
    --mount "type=bind,source=$pack,target=/pack,readonly" \
    --mount "type=bind,source=$data/remake/assets,target=/mainassets,readonly" \
    --mount "type=bind,source=$gocache,target=/gocache" \
    "${extra[@]}" -w /src/remake fd2-go-test-local:latest with-xvfb bash -c "$prelude
$1"
}

for run in $runs; do
  rd="$out/$run"
  rm -rf "$rd"
  mkdir -p "$rd/opening" "$rd/chapter/parity-out"
  docker_go "$run" 3000 \
    --mount "type=bind,source=$slot,target=/slot,readonly" \
    --mount "type=bind,source=$oracle,target=/oracle,readonly" \
    --mount "type=bind,source=$rd,target=/run" \
    -e FD2_ASSET_PACK=/pack -e FD2_PARITY_SLOT=/slot/FD2.SAV -e "FD2_R5_MODES=$modes" -- '
      set -u
      status=0
      for m in $FD2_R5_MODES; do
        case $m in
          opening) env FD2_GOLDEN_FRAMES_MODE=opening FD2_GOLDEN_FRAMES_OUT=/run/opening \
            go test ./cmd/fd2 -run "^TestRustGoldenFrames$" -count=1 -timeout 30m -v \
              -covermode=count -coverprofile=/run/cover-opening.out -coverpkg=./cmd/fd2 > /run/opening.log 2>&1 || status=1 ;;
          chapter) env FD2_GOLDEN_FRAMES_MODE=chapter FD2_GOLDEN_FRAMES_OUT=/run/chapter \
            FD2_PARITY_CHAPTER=1 FD2_PARITY_ORACLE_RUN=/oracle FD2_PARITY_OUT=/run/chapter/parity-out \
            FD2_PARITY_ENTRY=start FD2_PARITY_CAPTURE_FRAMES=0 \
            go test ./cmd/fd2 -run "^TestRustGoldenFrames$" -count=1 -timeout 45m -v \
              -covermode=count -coverprofile=/run/cover-chapter.out -coverpkg=./cmd/fd2 > /run/chapter.log 2>&1 || status=1 ;;
        esac
      done
      dpkg-query -W -f "\${Package} \${Version}\n" libgl1-mesa-dri libglx-mesa0 xvfb > /run/mesa-packages.txt 2>/dev/null || true
      exit $status' || { echo "$run 失敗，見 $rd/*.log" >&2; exit 1; }
  if [ -f "$rd/chapter/parity-out/checkpoints.jsonl" ]; then
    sum=$(sha256sum "$rd/chapter/parity-out/checkpoints.jsonl" | cut -c1-64)
    echo "$run checkpoints.jsonl $sum $( [ "$sum" = "$golden" ] && echo 與R4黃金相同 || echo 與R4黃金不同)"
  fi
  (cd "$rd" && find opening/frames chapter/frames -name '*.png' -print0 | sort -z | xargs -0 sha256sum) > "$rd/png-sha256.txt"
done

set -- $runs
if [ $# -ge 2 ]; then
  a="$out/$1" b="$out/$2"
  for m in opening chapter; do
    sa=$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["sequence_sha256"])' "$a/$m/frames.json")
    sb=$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["sequence_sha256"])' "$b/$m/frames.json")
    echo "$m 序列雜湊 $1=$sa $2=$sb $( [ "$sa" = "$sb" ] && echo 相同 || echo 不同)"
  done
  if cmp -s "$a/png-sha256.txt" "$b/png-sha256.txt"; then echo "PNG 逐檔雜湊相同（$(wc -l < "$a/png-sha256.txt") 張）"; else echo "PNG 逐檔雜湊不同"; fi
fi

if [ "${FD2_R5_SKIP_ANALYSIS:-0}" = 1 ]; then echo "略過取整實測與盤點（FD2_R5_SKIP_ANALYSIS=1）：$out"; exit 0; fi
first="$out/$1"
mkdir -p "$out/rounding" "$out/inventory"
docker_go analysis 1800 \
  --mount "type=bind,source=$first,target=/run,readonly" \
  --mount "type=bind,source=$out/rounding,target=/rounding" \
  --mount "type=bind,source=$out/inventory,target=/inventory" -- '
    set -euo pipefail
    go run ./cmd/fd2-r5-ebiten-rounding -out /rounding/ebiten-rounding.json > /rounding/rounding.log 2>&1
    go list -export -deps -json ./cmd/fd2 > /tmp/deps.json
    cd /src/tools/rust_r5_draw_inventory && go vet . && go build -o /tmp/rust_r5_draw_inventory .
    /tmp/rust_r5_draw_inventory -deps /tmp/deps.json -repo /src \
      -cover opening=/run/cover-opening.out -cover chapter=/run/cover-chapter.out -out /inventory > /inventory/inventory.log 2>&1'
cat "$out/rounding/rounding.log"
grep -E '^(STAT|GROUP|LEGACY)' "$out/inventory/inventory.log" || true
echo "完成：$out"
