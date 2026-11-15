#!/bin/bash
# R5 階段 2 r5d 組（docs/goal/121）：對話與文字組的局部黃金。Go 與 Rust 各跑一次第一章受控重播，
# 在同一個 pump 擷取點記嘴型狀態（全域亂數的消費者），走到 Draw 尾段時只呼叫 drawNativeStoryDialogue，
# 再逐擷取點比對。做法見 remake/cmd/fd2/rust_golden_dialogue_layers_test.go 開頭；驅動方式同
# tools/rust_r5_map_layers.sh。
#
# 用法：tools/rust_r5_dialogue_layers.sh <輸出目錄>
# 輸出：<輸出目錄>/go/（dialogue-layers.json、parity-out/、go.log）、<輸出目錄>/rust/（同）、compare.txt。
#
# 輸入（唯讀）：同 tools/rust_r5_golden_frames.sh 的 chapter 模式與 tools/rust_parity_ch01.sh。
# 環境變數：
#   FD2_DATA_ROOT   建構槽、原版輸出與原版素材包所在的儲存庫根（預設本儲存庫；工作樹副本指向主工作區）
#   FD2_GOCACHE     Go 建置快取（預設 <資料根>/work/gocache）
#   FD2_RUST_BUILD／FD2_RUST_TARGET 同 tools/rust_game_test.sh（預設 <資料根>/work/rust-build、target）
#   FD2_R5D_STAGE0  階段 0 chapter 黃金的 frames.json（預設 <資料根>/work/r5-stage0-20261115/r1/chapter/frames.json；
#                   有這個檔時另外以整張畫格對照，只是參考）
#   FD2_R5D_SIDES   要跑的一側（預設 "go rust"；只供除錯）
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
sides="${FD2_R5D_SIDES:-go rust}"
stage0="${FD2_R5D_STAGE0:-$data/work/r5-stage0-20261115/r1/chapter/frames.json}"
pyimage="${FD2_PYTHON_IMAGE:-python:3.13-slim}"
slot="$data/work/ch01-flow-20261017/input"
oracle="$data/work/ch01-flow-20261017/original-full-r4"
assets="$data/remake/generated-assets/fd2-original-b97caf22"
golden=1c846f573434658cabb3cfa486edd5826b401673c3bff411e30cbdc3e18ac0cc
for d in "$root/remake/cmd/fd2" "$root/remake/assets/locales" "$data/remake/assets" "$slot" "$oracle" "$assets" \
         "$gocache" "$build/cargo"; do
  test -d "$d" || { echo "找不到目錄 $d" >&2; exit 2; }
done
for f in "$slot/FD2.SAV" "$oracle/actions.jsonl" "$oracle/runner.json"; do test -f "$f" || { echo "缺少 $f" >&2; exit 2; }; done
test "$(sha256sum "$slot/FD2.SAV" | cut -c1-64)" = 6d14f2c22562cabca83725084f1a9d6539a1d4066da5c1debcdadb446812691f

# 分離素材根：符號連結根，目標寫成容器內路徑（同 tools/rust_r5_golden_frames.sh）。
pack=$(mktemp -d -t fd2-r5d-pack-XXXXXX)
trap 'rm -rf "$pack"' EXIT
for e in "$assets"/*; do ln -s "/packsrc/$(basename "$e")" "$pack/$(basename "$e")"; done
ln -s /locales "$pack/locales"

for side in $sides; do
  sd="$out/$side"
  rm -rf "$sd"
  mkdir -p "$sd/parity-out"
  case $side in
    go)
      # remake/assets 的產生檔不進版控：容器內複製 remake/ 到 /tmp/src，再從資料根補缺檔（同階段 0 工具）。
      timeout 3000 docker run --rm --name "fd2-r5d-go-$$" --network none --memory 8g --cpus 4 --pids-limit 512 \
        --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
        -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod \
        -e FD2_ASSET_PACK=/pack -e FD2_PARITY_SLOT=/slot/FD2.SAV \
        -e FD2_GOLDEN_DIALOGUE_LAYERS_OUT=/run -e FD2_PARITY_CHAPTER=1 -e FD2_PARITY_ORACLE_RUN=/oracle \
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
          go test ./cmd/fd2 -run "^TestRustGoldenDialogueLayers$" -count=1 -timeout 45m -v > /run/go.log 2>&1' \
        || { echo "Go 端失敗，見 $sd/go.log" >&2; tail -20 "$sd/go.log" >&2; exit 1; }
      ;;
    rust)
      mkdir -p "$build/$target"
      timeout 3000s docker run --rm --name "fd2-r5d-rust-$$" --network none --memory 6g --cpus 4 --pids-limit 1024 \
        --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h \
        -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
        -e FD2_ASSET_PACK=/pack -e FD2_REMAKE_ASSETS=/assets \
        -e FD2_GOLDEN_DIALOGUE_LAYERS_OUT=/run -e FD2_PARITY_CHAPTER=1 -e FD2_PARITY_SLOT=/slot/FD2.SAV \
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
  sum=$(sha256sum "$sd/parity-out/checkpoints.jsonl" | cut -c1-64)
  echo "$side checkpoints.jsonl $sum $( [ "$sum" = "$golden" ] && echo 與R4黃金相同 || echo 與R4黃金不同)"
done

stage0_mount=()
stage0_arg=()
if [ -f "$stage0" ]; then
  stage0_mount=(--mount "type=bind,source=$(dirname "$stage0"),target=/stage0,readonly")
  stage0_arg=(--stage0 "/stage0/$(basename "$stage0")")
fi
timeout 600s docker run --rm --name "fd2-r5d-cmp-$$" --network none --memory 2g --cpus 1 --pids-limit 64 \
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp \
  --mount "type=bind,source=$root/tools,target=/repo/tools,readonly" \
  --mount "type=bind,source=$out,target=/out,readonly" "${stage0_mount[@]}" -w /repo "$pyimage" \
  python3 -I tools/rust_r5_dialogue_layers_compare.py --go /out/go/dialogue-layers.json --rust /out/rust/dialogue-layers.json \
    "${stage0_arg[@]}" > "$out/compare.txt" 2>&1 || cmp=$?
cat "$out/compare.txt"
exit "${cmp:-0}"
