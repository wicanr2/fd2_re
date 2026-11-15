#!/bin/bash
# R5 階段 3（docs/goal/121，代號 r5s）：開場與第一章整張畫格整合的核對。
#
# 三段（FD2_STAGE3_PARTS，預設 "ttf chapter facility"）：
#   ttf       tools/rust_r5f_ttf_layers.sh：同一批 pump 擷取點的 TTF 繪字鍵、墨跡外框與兩邊的圖層裁切（chapter 段的輸入）。
#   chapter   tools/rust_r5_chapter_frames.sh（FD2_R5_DUMP_DIFF=1）：Rust chapter 模式整張擷取，與階段 0 Go 黃金雜湊不同的
#             畫格寫出 RGBA；再以 tools/rust_r5_stage3_compare.py chapter 逐張核對差異只在 TTF 墨跡外框內。
#   facility  城鎮與設施的整張畫格：remake/cmd/fd2/rust_golden_r5s_frames_test.go（Go）與
#             parity_replay/rust_golden_r5s_frames_test.rs（Rust）在 r5c 同一批 pump＋gap 擷取點呼叫整個 Draw；先以
#             tools/rust_r5c_layers_compare.py --profile r5s --verbatim 核對 Go 逐字複本，兩邊 checkpoints.jsonl 必須是
#             R4 黃金，再以 rust_r5_stage3_compare.py facility 逐張核對。
#
# 用法：tools/rust_r5_stage3_verify.sh <輸出目錄>
# 輸出：<輸出目錄>/{ttf-layers,chapter,facility}/、<輸出目錄>/compare/（chapter-*、facility-*）。
#
# 環境變數：
#   FD2_DATA_ROOT   建構槽、原版輸出、原版素材包與階段 0 黃金所在的儲存庫根（預設本儲存庫；工作樹副本指向主工作區）
#   FD2_GOCACHE     Go 建置快取（預設 <資料根>/work/gocache）
#   FD2_RUST_BUILD／FD2_RUST_TARGET 同 tools/rust_game_test.sh（預設 <資料根>/work/rust-build、target）
#   FD2_STAGE3_TTF_LAYERS 已有的 tools/rust_r5f_ttf_layers.sh 輸出目錄（設了就不跑 ttf 段，直接用它）
#   FD2_STAGE3_PARTS 要跑的段（預設 "ttf chapter facility"）
#   FD2_PYTHON_IMAGE 比對用映像檔（預設 python:3.13-slim）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
source "$root/tools/rust_ttf_font.sh"
data="$(cd "${FD2_DATA_ROOT:-$root}" && pwd -P)"
out="$(mkdir -p "${1:?輸出目錄}" && cd "$1" && pwd -P)"
gocache="${FD2_GOCACHE:-$data/work/gocache}"
build="${FD2_RUST_BUILD:-$data/work/rust-build}"
target="${FD2_RUST_TARGET:-target}"
parts="${FD2_STAGE3_PARTS:-ttf chapter facility}"
pyimage="${FD2_PYTHON_IMAGE:-python:3.13-slim}"
stage0="$data/work/r5-stage0-20261115/r1/chapter"
slot="$data/work/ch01-flow-20261017/input"
oracle="$data/work/ch01-flow-20261017/original-full-r4"
assets="$data/remake/generated-assets/fd2-original-b97caf22"
golden=1c846f573434658cabb3cfa486edd5826b401673c3bff411e30cbdc3e18ac0cc
stage0_sha=d71aad6a20e1258459f99b8d95e32ae773dd067b3c720149b66980e6428ac3de
for d in "$root/remake/cmd/fd2" "$root/remake/assets/locales" "$data/remake/assets" "$slot" "$oracle" "$assets" \
         "$gocache" "$build/cargo" "$stage0/frames"; do
  test -d "$d" || { echo "找不到目錄 $d" >&2; exit 2; }
done
for f in "$slot/FD2.SAV" "$oracle/actions.jsonl" "$oracle/runner.json" "$stage0/frames.json"; do
  test -f "$f" || { echo "缺少 $f" >&2; exit 2; }
done
test "$(sha256sum "$slot/FD2.SAV" | cut -c1-64)" = 6d14f2c22562cabca83725084f1a9d6539a1d4066da5c1debcdadb446812691f
test "$(sha256sum "$stage0/frames.json" | cut -c1-64)" = "$stage0_sha" || { echo "階段 0 chapter 黃金的雜湊不同" >&2; exit 2; }
mkdir -p "$out/compare"

# 比對容器：tools 唯讀、輸入唯讀、只寫 compare/。
cmp() {
  local name=$1; shift
  timeout 3000s docker run --rm --name "fd2-r5s-$name-$$" --network none --memory 4g --cpus 4 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp \
    --mount "type=bind,source=$root/tools,target=/repo/tools,readonly" \
    --mount "type=bind,source=$root/remake/cmd/fd2,target=/gosrc,readonly" \
    --mount "type=bind,source=$stage0,target=/stage0,readonly" \
    --mount "type=bind,source=$out,target=/in,readonly" \
    --mount "type=bind,source=$out/compare,target=/out" -w /repo "$pyimage" "$@"
}

ttfdir="${FD2_STAGE3_TTF_LAYERS:-$out/ttf-layers}"
status=0
for part in $parts; do
  case $part in
    ttf)
      if [ -n "${FD2_STAGE3_TTF_LAYERS:-}" ]; then
        echo "== ttf：沿用 $FD2_STAGE3_TTF_LAYERS"
        continue
      fi
      echo "== ttf（tools/rust_r5f_ttf_layers.sh）"
      FD2_DATA_ROOT="$data" FD2_GOCACHE="$gocache" FD2_RUST_BUILD="$build" FD2_RUST_TARGET="$target" \
        "$root/tools/rust_r5f_ttf_layers.sh" "$ttfdir" > "$out/ttf-layers.log" 2>&1 \
        || { echo "TTF 圖層黃金失敗，見 $out/ttf-layers.log" >&2; tail -20 "$out/ttf-layers.log" >&2; exit 1; }
      tail -6 "$out/ttf-layers.log"
      ;;
    chapter)
      for f in "$ttfdir/go/ttf-layers.json" "$ttfdir/rust/ttf-layers.json"; do
        test -f "$f" || { echo "缺少 $f（先跑 ttf 段）" >&2; exit 2; }
      done
      echo "== chapter（tools/rust_r5_chapter_frames.sh，FD2_R5_DUMP_DIFF=1）"
      # 比對有差異時這支會回非 0；這裡只要 frames.json 寫出且沒有停點，判定交給下面的逐像素核對。
      FD2_R5_DUMP_DIFF=1 FD2_WORK="$data/work" FD2_ASSET_PACK="$assets" FD2_REMAKE_ASSETS="$data/remake/assets" \
        FD2_RUST_BUILD="$build" FD2_RUST_TARGET="$target" FD2_GOLDEN_CHAPTER="$stage0/frames.json" \
        "$root/tools/rust_r5_chapter_frames.sh" "$out/chapter" > "$out/chapter.log" 2>&1 || true
      test -f "$out/chapter/rust/frames.json" || { echo "chapter 沒有輸出 frames.json，見 $out/chapter.log" >&2; exit 1; }
      sum=$(sha256sum "$out/chapter/rust/parity-out/checkpoints.jsonl" | cut -c1-64)
      echo "chapter checkpoints.jsonl $sum $( [ "$sum" = "$golden" ] && echo 與R4黃金相同 || echo 與R4黃金不同)"
      [ "$sum" = "$golden" ] || status=1
      ttf_in="$ttfdir"
      ttf_mount=(--mount "type=bind,source=$ttf_in,target=/ttf,readonly")
      timeout 3000s docker run --rm --name "fd2-r5s-cmpch-$$" --network none --memory 4g --cpus 4 --pids-limit 64 \
        --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp \
        --mount "type=bind,source=$root/tools,target=/repo/tools,readonly" \
        --mount "type=bind,source=$stage0,target=/stage0,readonly" \
        --mount "type=bind,source=$out/chapter/rust,target=/rust,readonly" "${ttf_mount[@]}" \
        --mount "type=bind,source=$out/compare,target=/out" -w /repo "$pyimage" \
        python3 -I tools/rust_r5_stage3_compare.py chapter --go /stage0/frames.json --rust /rust/frames.json \
          --ttf-go /ttf/go/ttf-layers.json --ttf-rust /ttf/rust/ttf-layers.json --out /out --jobs 4 || status=1
      ;;
    facility)
      echo "== facility：Go 測試裡的逐字複本"
      cmp verbatim python3 -I tools/rust_r5c_layers_compare.py --profile r5s --verbatim /gosrc | tee "$out/facility-verbatim.txt" \
        || { echo "逐字複本與原檔不同" >&2; exit 1; }
      pack=$(mktemp -d -t fd2-r5s-pack-XXXXXX)
      trap 'rm -rf "$pack"' EXIT
      for e in "$assets"/*; do ln -s "/packsrc/$(basename "$e")" "$pack/$(basename "$e")"; done
      ln -s /locales "$pack/locales"
      fd="$out/facility"
      rm -rf "$fd"
      mkdir -p "$fd/go/parity-out" "$fd/rust/parity-out"
      echo "== facility：go"
      timeout 3000 docker run --rm --name "fd2-r5s-go-$$" --network none --memory 8g --cpus 4 --pids-limit 512 \
        --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
        -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod \
        -e FD2_ASSET_PACK=/pack -e FD2_PARITY_SLOT=/slot/FD2.SAV -e FD2_GOLDEN_R5S_FRAMES_OUT=/run \
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
        --mount "type=bind,source=$fd/go,target=/run" \
        -w /src/remake fd2-go-test-local:latest with-xvfb bash -c '
          set -e; sha256sum '"$ttf_font_path"' > /run/font-sha256.txt
          mkdir -p /tmp/src; cp -a /src/remake /tmp/src/remake
          cp -a -n /mainassets/. /tmp/src/remake/assets/
          cd /tmp/src/remake
          go test ./cmd/fd2 -run "^TestRustGoldenR5sFrames$" -count=1 -timeout 45m -v > /run/go.log 2>&1' \
        || { echo "Go 端失敗，見 $fd/go/go.log" >&2; tail -20 "$fd/go/go.log" >&2; exit 1; }
      test "$(cut -c1-64 "$fd/go/font-sha256.txt")" = "$ttf_font_sha" || { echo "Go 映像內的字型雜湊不同" >&2; exit 1; }
      echo "== facility：rust"
      mkdir -p "$build/$target"
      timeout 3000s docker run --rm --name "fd2-r5s-rust-$$" --network none --memory 6g --cpus 4 --pids-limit 1024 \
        --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h \
        -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
        -e FD2_ASSET_PACK=/pack -e FD2_REMAKE_ASSETS=/assets -e FD2_GOLDEN_R5S_FRAMES_OUT=/run \
        -e FD2_GOLDEN_R5S_FRAMES_DIFF_REF=/go/r5s-frames.json \
        -e FD2_PARITY_CHAPTER=1 -e FD2_PARITY_SLOT=/slot/FD2.SAV \
        -e FD2_PARITY_ORACLE_RUN=/oracle -e FD2_PARITY_OUT=/run/parity-out -e FD2_PARITY_ENTRY=start \
        -e FD2_PARITY_CAPTURE_FRAMES=0 \
        --mount "type=bind,source=$slot,target=/slot,readonly" \
        --mount "type=bind,source=$oracle,target=/oracle,readonly" \
        --mount "type=bind,source=$fd/rust,target=/run" \
        --mount "type=bind,source=$fd/go,target=/go,readonly" \
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
        ' > "$fd/rust/replay.log" 2>&1 || { echo "Rust 端失敗，見 $fd/rust/replay.log" >&2; tail -20 "$fd/rust/replay.log" >&2; exit 1; }
      for side in go rust; do
        test -f "$fd/$side/r5s-frames.json" || { echo "$side 沒有輸出 r5s-frames.json" >&2; exit 1; }
        sum=$(sha256sum "$fd/$side/parity-out/checkpoints.jsonl" | cut -c1-64)
        echo "facility $side checkpoints.jsonl $sum $( [ "$sum" = "$golden" ] && echo 與R4黃金相同 || echo 與R4黃金不同)"
        [ "$sum" = "$golden" ] || status=1
      done
      cmp cmpfac python3 -I tools/rust_r5_stage3_compare.py facility --go /in/facility/go/r5s-frames.json \
        --rust /in/facility/rust/r5s-frames.json --stage0 /stage0/frames.json --out /out --jobs 4 || status=1
      ;;
    *) echo "未知的段 $part" >&2; exit 2 ;;
  esac
done
exit "$status"
