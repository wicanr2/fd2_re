#!/bin/bash
# R5 階段 5（docs/goal/121，r5g）：GPU 加速模式與 CPU 合成逐像素相同的驗證，與兩種模式的每幀時間。
#
# 用法：tools/rust_r5g_gpu_frames.sh <輸出目錄>
# 步驟（FD2_R5G_STEPS 以逗號選，預設全部）：
#   vectors   Ebiten 實測向量（30 混色＋89 取樣＋9 語意）與 400 份亂數清單交給 GPU 執行器，讀回要與實測／直接 CPU 相同
#   opening   tools/rust_r5_opening_frames.sh 以 CPU、GPU 各跑一次（10372 次擷取），兩份 frames.json 逐列相同且都等於 Go 黃金
#   chapter   tools/rust_r5_chapter_frames.sh 以 CPU、GPU 各跑一次（13751 次擷取），兩份 frames.json 逐列相同
#   app       tools/rust_r5_app_frames.sh 以 GPU 模式外殼跑（無畫面驅動端仍是 CPU）：逐幀雜湊相同、截圖逐像素相同；
#             另以 CPU 模式外殼跑一次取時間
#   headless  fd2_frames 以 CPU、GPU 各跑 tools/rust_r5_app_scripts 的腳本：frames.jsonl 逐行相同，記每幀時間
# 最後把各份 timing.jsonl 整理成 <輸出目錄>/timing.json。GPU 是容器內 Mesa llvmpipe／lavapipe（軟體），時間只當參考。
#
# 環境變數同 tools/rust_r5_app_frames.sh（FD2_WORK／FD2_ASSET_PACK／FD2_REMAKE_ASSETS／FD2_RUST_BUILD／FD2_RUST_TARGET／
# FD2_PYTHON_IMAGE）；opening／chapter 的 Go 黃金同 tools/rust_r5_opening_frames.sh、tools/rust_r5_chapter_frames.sh。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
source "$root/tools/rust_ttf_font.sh"
out="$(mkdir -p "${1:?輸出目錄}" && cd "$1" && pwd -P)"
steps=",${FD2_R5G_STEPS:-vectors,opening,chapter,app,headless},"
work="${FD2_WORK:-$root/work}"
build="${FD2_RUST_BUILD:-$root/work/rust-build}"
target="${FD2_RUST_TARGET:-target}"
packsrc="${FD2_ASSET_PACK:-$root/remake/generated-assets/fd2-original-b97caf22}"
assets="${FD2_REMAKE_ASSETS:-$root/remake/assets}"
pyimage="${FD2_PYTHON_IMAGE:-python:3.13-slim}"
slot="$work/ch01-flow-20261017/input"
for d in "$slot" "$packsrc" "$assets" "$build/cargo"; do
  test -d "$d" || { echo "找不到 $d" >&2; exit 2; }
done
test -f "$slot/FD2.SAV" || { echo "缺少 $slot/FD2.SAV" >&2; exit 2; }
mkdir -p "$build/$target"
exec_path=/build/target-x/release/r5g_gpu_exec
has() { case "$steps" in *",$1,"*) return 0 ;; *) return 1 ;; esac; }

cargo_run=(--rm --network none --pids-limit 1024 --log-opt max-size=10m --log-opt max-file=3
  --user "$(id -u):$(id -g)" -e HOME=/tmp/h -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x
  --mount "type=bind,source=$root/remake-rs,target=/src"
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly"
  --mount "type=bind,source=$build/cargo,target=/build/cargo"
  --mount "type=bind,source=$build/$target,target=/build/target-x" -w /src)
py() {
  timeout 1200s docker run --rm --name "fd2-r5g-py-$$" --network none --memory 2g --cpus 1 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp \
    --mount "type=bind,source=$root/tools,target=/repo/tools,readonly" \
    --mount "type=bind,source=$out,target=/out" -w /repo "$pyimage" python3 -I tools/rust_r5g_frames_compare.py "$@"
}

status=0
fail() { echo "失敗：$*" >&2; status=1; }

echo "== 建置 fd2-app（release：fd2、fd2_frames、r5g_gpu_exec）"
timeout 3600s docker run "${cargo_run[@]}" --name "fd2-r5g-build-$$" --memory 8g --cpus 4 fd2-rust:1.99.1-r3 \
  cargo build --offline --locked --release -p fd2-app --bins

if has vectors; then
  echo "== 向量：GPU 執行器與 Ebiten 實測、直接 CPU"
  timeout 3600s docker run "${cargo_run[@]}" --name "fd2-r5g-vectors-$$" --memory 6g --cpus 4 \
    -e "FD2_R5G_GPU_EXECUTOR=$exec_path" -e WGPU_BACKEND=vulkan fd2-rust:1.99.1-r3 bash -euo pipefail -c '
      cargo test --offline --locked -p fd2-game --release --lib -- --ignored --exact --nocapture \
        game::canvas::display_list::display_list_test::gpu_executor_matches_replay
    ' > "$out/vectors.log" 2>&1 || fail "vectors（見 $out/vectors.log）"
  grep -E 'GPU 執行器|test result' "$out/vectors.log" || true
fi

for mode in opening chapter; do
  has "$mode" || continue
  echo "== $mode：CPU"
  bash "$root/tools/rust_r5_${mode}_frames.sh" "$out/$mode-cpu" > "$out/$mode-cpu.log" 2>&1 || echo "（$mode CPU 與 Go 黃金比對的結束碼不是 0，見 $out/$mode-cpu.log）"
  tail -4 "$out/$mode-cpu.log"
  echo "== $mode：GPU"
  FD2_R5G_GPU_EXECUTOR="$exec_path" bash "$root/tools/rust_r5_${mode}_frames.sh" "$out/$mode-gpu" > "$out/$mode-gpu.log" 2>&1 \
    || echo "（$mode GPU 與 Go 黃金比對的結束碼不是 0，見 $out/$mode-gpu.log）"
  tail -4 "$out/$mode-gpu.log"
  echo "== $mode：GPU 與 CPU 逐列比對"
  expected=(--expected-captures 10372)
  if [ "$mode" = chapter ]; then
    # 階段 3 已確認這個 Rust 序列和 Go 的差異全在 TTF 墨跡外框內。
    expected=(--expected-captures 13751
      --expected-sequence 4c3effd317d72cb10c7cea90fef55cf8a527df9e6974c6cd5355e6aed2b9f811)
  fi
  py frames --cpu "/out/$mode-cpu/rust/frames.json" --gpu "/out/$mode-gpu/rust/frames.json" \
    --out "/out/$mode-compare.json" "${expected[@]}" || fail "$mode GPU 與 CPU 未通過畫格驗證"
done
if has opening; then
  # opening 兩種模式都要整段等於 Go 黃金（10372／10372；chapter 有 TTF 的已知差異，只比 GPU 與 CPU）。
  for m in cpu gpu; do
    py gocheck --compare "/out/opening-$m/compare.json" || fail "opening $m 與 Go 黃金不同"
  done
fi

if has app; then
  echo "== app：GPU 模式外殼"
  FD2_R5G_RENDERER=gpu bash "$root/tools/rust_r5_app_frames.sh" "$out/app-gpu" > "$out/app-gpu.log" 2>&1 || fail "app GPU（見 $out/app-gpu.log）"
  grep -E '驅動一致|截圖|相同|不同' "$out/app-gpu.log" | tail -8 || true
  for d in "$out"/app-gpu/*/; do
    [ -f "$d/app/summary.json" ] || { fail "缺少 $d/app/summary.json"; continue; }
    py summary --app-summary "/out/app-gpu/$(basename "$d")/app/summary.json" --want gpu || fail "$(basename "$d") 外殼沒有用 GPU"
  done
  echo "== app：CPU 模式外殼（取時間）"
  bash "$root/tools/rust_r5_app_frames.sh" "$out/app-cpu" > "$out/app-cpu.log" 2>&1 || fail "app CPU（見 $out/app-cpu.log）"
fi

if has headless; then
  pack="$out/pack"
  rm -rf "$pack"
  mkdir -p "$pack"
  trap 'rm -rf "$pack"' EXIT
  for e in "$packsrc"/*; do
    ln -s "/packsrc/$(basename "$e")" "$pack/$(basename "$e")"
  done
  ln -s /assets/locales "$pack/locales"
  for s in "$root"/tools/rust_r5_app_scripts/*.json; do
    name="$(basename "$s" .json)"
    for r in cpu gpu; do
      dir="$out/headless-$r/$name"
      rm -rf "$dir"
      mkdir -p "$dir/state" "$dir/script"
      cp "$s" "$dir/script/script.json"
      cp "$slot/FD2.SAV" "$dir/state/FD2.SAV"
      echo "== headless：$name（$r）"
      timeout 3600s docker run "${cargo_run[@]}" --name "fd2-r5g-headless-$$" --memory 4g --cpus 2 \
        -e FD2_ASSET_PACK=/pack -e FD2_REMAKE_ASSETS=/assets -e FD2_CAMPAIGN=canonical -e FD2_SEED=4 \
        -e FD2_MUTE=1 -e FD2_NATIVE_SAVE=/state/FD2.SAV -e XDG_DATA_HOME=/state/xdg \
        -e "FD2_RENDERER=$r" -e WGPU_BACKEND=vulkan \
        --mount "type=bind,source=$pack,target=/pack,readonly" \
        --mount "type=bind,source=$packsrc,target=/packsrc,readonly" \
        --mount "type=bind,source=$assets,target=/assets,readonly" \
        --mount "type=bind,source=$dir/script,target=/script,readonly" \
        --mount "type=bind,source=$dir/state,target=/state" \
        --mount "type=bind,source=$dir,target=/out" \
        "${ttf_mounts[@]}" fd2-rust:1.99.1-r3 \
        bash -euo pipefail -c '/build/target-x/release/fd2_frames /script/script.json /out/frames.jsonl /out/timing.jsonl' \
        > "$dir/run.log" 2>&1 || { fail "headless $name $r（見 $dir/run.log）"; continue; }
    done
    py jsonl --cpu "/out/headless-cpu/$name/frames.jsonl" --gpu "/out/headless-gpu/$name/frames.jsonl" \
      --out "/out/headless-$name-compare.json" || fail "headless $name GPU 與 CPU 不同"
  done
fi

echo "== 時間"
specs=()
for f in "$out"/headless-*/*/timing.jsonl "$out"/app-*/*/app/timing.jsonl; do
  [ -f "$f" ] || continue
  rel="${f#"$out"/}"
  specs+=("$(echo "$rel" | sed 's#/timing.jsonl##; s#/app$##; s#/#:#g')=/out/$rel")
done
if [ ${#specs[@]} -gt 0 ]; then
  py timing --out /out/timing.json "${specs[@]}" || true
fi
echo "結束（status=$status）"
exit "$status"
