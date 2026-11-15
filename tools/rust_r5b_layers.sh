#!/bin/bash
# R5 階段 2 r5b 組（docs/goal/121）：戰鬥演出（drawBattleScene、drawNativePhysicalPrelude／Body、
# drawNativeAIIdleRecovery、drawNativeCh28PostPresent）的局部黃金。Go 與 Rust 各跑第一章受控重播（chapter 模式，
# 每個 pump 幀），本組分支呼叫整個 Draw，其他分支照 Draw 留下狀態改動，再逐擷取點比對，並把本組分支與階段 0
# 整張黃金同一擷取點對照。做法見 remake/cmd/fd2/rust_golden_r5b_layers_test.go 開頭。
#
# 用法：tools/rust_r5b_layers.sh <輸出目錄>
# 輸出：<輸出目錄>/{go,rust}/（r5b-layers.json、log、parity-out/）、<輸出目錄>/compare.txt。
#
# 輸入（唯讀）：同 tools/rust_r5p_layers.sh。
# 環境變數：
#   FD2_DATA_ROOT   建構槽、原版輸出與原版素材包所在的儲存庫根（預設本儲存庫；工作樹副本指向主工作區）
#   FD2_GOCACHE     Go 建置快取（預設 <資料根>/work/gocache）
#   FD2_RUST_BUILD／FD2_RUST_TARGET 同 tools/rust_game_test.sh（預設 <資料根>/work/rust-build、target）
#   FD2_R5B_STAGE0  階段 0 黃金目錄（預設 <資料根>/work/r5-stage0-20261115/r1；用 chapter/frames.json）
#   FD2_R5B_SIDES   要跑的一側（預設 "go rust"；只跑一側時沿用輸出目錄裡另一側的結果）
#   FD2_PYTHON_IMAGE 比對用映像檔（預設 python:3.13-slim）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
# R5 r5f：Go 黃金用的 TTF 字型（缺檔或雜湊不符即失敗），見 tools/rust_ttf_font.sh。
source "$root/tools/rust_ttf_font.sh"
data="$(cd "${FD2_DATA_ROOT:-$root}" && pwd -P)"
out="$(mkdir -p "${1:?輸出目錄}" && cd "$1" && pwd -P)"
gocache="${FD2_GOCACHE:-$data/work/gocache}"
build="${FD2_RUST_BUILD:-$data/work/rust-build}"
target="${FD2_RUST_TARGET:-target}"
sides="${FD2_R5B_SIDES:-go rust}"
stage0="${FD2_R5B_STAGE0:-$data/work/r5-stage0-20261115/r1}"
pyimage="${FD2_PYTHON_IMAGE:-python:3.13-slim}"
slot="$data/work/ch01-flow-20261017/input"
oracle="$data/work/ch01-flow-20261017/original-full-r4"
assets="$data/remake/generated-assets/fd2-original-b97caf22"
golden=1c846f573434658cabb3cfa486edd5826b401673c3bff411e30cbdc3e18ac0cc
for d in "$root/remake/cmd/fd2" "$root/remake/assets/locales" "$data/remake/assets" "$slot" "$oracle" "$assets" \
         "$gocache" "$build/cargo" "$stage0/chapter"; do
  test -d "$d" || { echo "找不到目錄 $d" >&2; exit 2; }
done
for f in "$slot/FD2.SAV" "$oracle/actions.jsonl" "$oracle/runner.json" "$stage0/chapter/frames.json"; do
  test -f "$f" || { echo "缺少 $f" >&2; exit 2; }
done
test "$(sha256sum "$slot/FD2.SAV" | cut -c1-64)" = 6d14f2c22562cabca83725084f1a9d6539a1d4066da5c1debcdadb446812691f

# 分離素材根：符號連結根，目標寫成容器內路徑（同 tools/rust_r5_golden_frames.sh）。
pack=$(mktemp -d -t fd2-r5b-pack-XXXXXX)
trap 'rm -rf "$pack"' EXIT
for e in "$assets"/*; do ln -s "/packsrc/$(basename "$e")" "$pack/$(basename "$e")"; done
ln -s /locales "$pack/locales"

for side in $sides; do
  sd="$out/$side"
  rm -rf "$sd"
  mkdir -p "$sd/parity-out"
  echo "== chapter／$side"
  case $side in
    go)
      timeout 3000 docker run --rm --name "fd2-r5b-go-$$" --network none --memory 8g --cpus 4 --pids-limit 512 \
        --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
        -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod \
        -e FD2_ASSET_PACK=/pack -e FD2_PARITY_SLOT=/slot/FD2.SAV \
        -e FD2_GOLDEN_R5B_LAYERS_OUT=/run \
        -e FD2_PARITY_CHAPTER=1 -e FD2_PARITY_ORACLE_RUN=/oracle \
        -e FD2_PARITY_OUT=/run/parity-out -e FD2_PARITY_ENTRY=start -e FD2_PARITY_CAPTURE_FRAMES=0 \
        --mount "type=bind,source=$root,target=/src,readonly" \
        --mount "type=bind,source=$assets,target=/packsrc,readonly" \
        --mount "type=bind,source=$root/remake/assets/locales,target=/locales,readonly" \
        --mount "type=bind,source=$pack,target=/pack,readonly" \
        --mount "type=bind,source=$data/remake/assets,target=/mainassets,readonly" \
        --mount "type=bind,source=$gocache,target=/gocache" \
        --mount "type=bind,source=$slot,target=/slot,readonly" \
        --mount "type=bind,source=$oracle,target=/oracle,readonly" \
        --mount "type=bind,source=$sd,target=/run" \
        -w /src/remake fd2-go-test-local:latest with-xvfb bash -c '
          set -e; mkdir -p /tmp/src; cp -a /src/remake /tmp/src/remake
          cp -a -n /mainassets/. /tmp/src/remake/assets/
          cd /tmp/src/remake
          go test ./cmd/fd2 -run "^TestRustGoldenR5bLayers$" -count=1 -timeout 45m -v > /run/go.log 2>&1' \
        || { echo "Go 端失敗，見 $sd/go.log" >&2; tail -20 "$sd/go.log" >&2; exit 1; }
      ;;
    rust)
      mkdir -p "$build/$target"
      timeout 3000s docker run --rm --name "fd2-r5b-rust-$$" --network none --memory 6g --cpus 4 --pids-limit 1024 \
        --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h \
        -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
        -e FD2_ASSET_PACK=/pack -e FD2_REMAKE_ASSETS=/assets \
        -e FD2_GOLDEN_R5B_LAYERS_OUT=/run \
        -e FD2_PARITY_CHAPTER=1 -e FD2_PARITY_SLOT=/slot/FD2.SAV \
        -e FD2_PARITY_ORACLE_RUN=/oracle -e FD2_PARITY_OUT=/run/parity-out -e FD2_PARITY_ENTRY=start \
        -e FD2_PARITY_CAPTURE_FRAMES=0 \
        --mount "type=bind,source=$slot,target=/slot,readonly" \
        --mount "type=bind,source=$oracle,target=/oracle,readonly" \
        --mount "type=bind,source=$sd,target=/run" \
        --mount "type=bind,source=$pack,target=/pack,readonly" \
        --mount "type=bind,source=$assets,target=/packsrc,readonly" \
        --mount "type=bind,source=$root/remake/assets/locales,target=/locales,readonly" \
        --mount "type=bind,source=$data/remake/assets,target=/assets,readonly" \
        "${ttf_mounts[@]}" \
        --mount "type=bind,source=$root/remake-rs,target=/src" \
        --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
        --mount "type=bind,source=$root/docs,target=/docs,readonly" \
        --mount "type=bind,source=$build/cargo,target=/build/cargo" \
        --mount "type=bind,source=$build/$target,target=/build/target-x" -w /src \
        fd2-rust:1.99.1-r3 bash -euo pipefail -c '
          cargo test --offline --locked -p fd2-game --release --lib -- --ignored --exact --nocapture \
            game::parity_replay::chapter_parity_replay_test::test_chapter_parity_replay
        ' > "$sd/replay.log" 2>&1 || { echo "Rust 端失敗，見 $sd/replay.log" >&2; tail -20 "$sd/replay.log" >&2; exit 1; }
      ;;
    *) echo "未知的一側 $side" >&2; exit 2 ;;
  esac
  test -f "$sd/r5b-layers.json" || { echo "$side 沒有輸出 r5b-layers.json" >&2; exit 1; }
  sum=$(sha256sum "$sd/parity-out/checkpoints.jsonl" | cut -c1-64)
  echo "$side checkpoints.jsonl $sum $( [ "$sum" = "$golden" ] && echo 與R4黃金相同 || echo 與R4黃金不同)"
done

status=0
timeout 600s docker run --rm --name "fd2-r5b-cmp-$$" --network none --memory 2g --cpus 1 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp \
  --mount "type=bind,source=$root/tools,target=/repo/tools,readonly" \
  --mount "type=bind,source=$out,target=/out,readonly" \
  --mount "type=bind,source=$stage0/chapter,target=/stage0,readonly" -w /repo "$pyimage" \
  python3 -I tools/rust_r5b_layers_compare.py --go /out/go/r5b-layers.json --rust /out/rust/r5b-layers.json \
    --stage0 /stage0/frames.json > "$out/compare.txt" 2>&1 || status=$?
cat "$out/compare.txt"
exit "$status"
