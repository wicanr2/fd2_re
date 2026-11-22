#!/bin/bash
# 在正常 Bevy 視窗送 X11 鍵盤事件，驗證 F2、設定冷啟動及缺包回退。
# 用法：FD2_RUST_TARGET=target-r5i bash tools/rust_r6_theme_window.sh work/<新收據目錄>
# 先用 Rust 映像 release profile 建置 fd2。另一容器只經隔離的 X11 socket 操作視窗。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
out="$root/${1:?需要工作區相對輸出路徑}"
build="${FD2_RUST_BUILD:-$root/work/rust-build}/${FD2_RUST_TARGET:-target}"
orig="${FD2_ASSET_PACK:-$root/remake/generated-assets/fd2-original-b97caf22}"
hd="${FD2_MODERN_THEME_PACK:-$root/../fd2-assets-private/packs/fd2-modern-handpainted-a}"
ttf="${FD2_TTF_FONT:-/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc}"
test -f "$ttf" && test -f "$build/release/fd2" && test -d "$hd" && test -d "$orig"
test -d "$root/remake/assets" && test -d "$root/work"
[[ "$out" == "$root"/work/* && "$out" != *'/../'* && ! -e "$out" ]]
relative="${out#"$root"/work/}"
common=(--rm --network none --pids-limit 128 --user "$(id -u):$(id -g)")
timeout 20s docker run "${common[@]}" --memory 128m --cpus 1 \
  --mount "type=bind,source=$root/work,target=/work" fd2-rust:1.99.1-r3 \
  mkdir -p "/work/$relative/socket" "/work/$relative/profile" "/work/$relative/rejected"
hd="$(cd "$hd" && pwd -P)"
name="fd2-r6-theme-window-$$"
stop_window() {
  docker stop -t 3 "$name" >/dev/null 2>&1 || true
}
trap stop_window EXIT

window() {
  local mode="$1" profile="$2" pack="$3"
  docker run -d "${common[@]}" --init --name "$name" --memory 6g --cpus 4 \
    --mount "type=bind,source=$build/release,target=/bins,readonly" \
    --mount "type=bind,source=$root/remake/assets,target=/assets,readonly" \
    --mount "type=bind,source=$orig,target=/original,readonly" \
    --mount "type=bind,source=$hd,target=/hd,readonly" \
    --mount "type=bind,source=$ttf,target=/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc,readonly" \
    --mount "type=bind,source=$out,target=/out" \
    --mount "type=bind,source=$out/socket,target=/tmp/.X11-unix" \
    -e "PROFILE=$profile" -e "MODE=$mode" -e "FD2_MODERN_THEME_PACK=$pack" \
    -e DISPLAY=:96 -e FD2_CAMPAIGN=canonical -e FD2_REQUIRE_ORIGINAL=1 -e FD2_MUTE=1 -e FD2_SEED=4 \
    -e FD2_CUTSCENE_LOG=1 -e WGPU_BACKEND=vulkan -e LIBGL_ALWAYS_SOFTWARE=1 -e LP_NUM_THREADS=2 -e RUST_LOG=warn \
    -w / fd2-rust:1.99.1-r3 bash -euc '
      export XDG_DATA_HOME="/out/$PROFILE" XDG_CACHE_HOME=/tmp/fd2-cache FD2_ASSET_PACK=/tmp/pack
      mkdir -p "$FD2_ASSET_PACK"
      for path in /original/*; do ln -s "$path" "$FD2_ASSET_PACK/"; done
      ln -s /assets/locales "$FD2_ASSET_PACK/locales"
      rm -f /tmp/.X11-unix/X96
      Xvfb :96 -screen 0 1280x800x24 -nolisten tcp -ac >"/out/$MODE-xvfb.log" 2>&1 &
      xvfb=$!
      trap "kill $xvfb 2>/dev/null || true" EXIT
      for _ in $(seq 1 60); do test ! -S /tmp/.X11-unix/X96 || break; sleep 0.1; done
      test -S /tmp/.X11-unix/X96
      timeout 240s /bins/fd2 >"/out/$MODE.log" 2>&1
    ' >/dev/null
}

drive() {
  timeout 220s docker run "${common[@]}" --name "$name-driver" --memory 512m --cpus 1 \
    --mount "type=bind,source=$out,target=/out" \
    --mount "type=bind,source=$out/socket,target=/tmp/.X11-unix,readonly" \
    -e DISPLAY=:96 -e "MODE=$1" -w /out fd2-promo-local:latest bash -euc '
      for _ in $(seq 1 200); do
        wid="$(xdotool search --onlyvisible --name "fd2_re rust" 2>/dev/null | head -1)"
        test -z "$wid" || break
        sleep 0.1
      done
      test -n "$wid"
      xdotool windowfocus --sync "$wid"
      key() { xdotool keydown "$1"; sleep 0.45; xdotool keyup "$1"; }
      capture() { import -window "$wid" -depth 8 -define png:exclude-chunks=date,time "$1"; }
      same() { compare -metric AE "$1" "$2" null: 2>/dev/null; }
      different() {
        local changed
        changed="$(compare -metric AE "$1" "$2" null: 2>&1)" || test "$?" -eq 1
        test "$changed" -gt 0
      }
      first_visible_frame() {
        for _ in $(seq 1 60); do
          capture "$1"
          if test "$(identify -format "%k" "$1")" -gt 1; then return; fi
          sleep 0.3
        done
        return 1
      }
      wait_for_change() {
        for _ in $(seq 1 35); do
          sleep 0.6
          capture "$2"
          if different "$1" "$2"; then return; fi
        done
        echo "開場停在同一張圖：$1" >&2
        return 1
      }
      wait_for_event() {
        for _ in $(seq 1 220); do
          if grep -Fq "$1" "$MODE.log"; then return; fi
          sleep 0.5
        done
        echo "逾時等候 $1 ($MODE)" >&2
        return 1
      }
      wait_for_menu() {
        # 走正常開場的按鍵略過路徑。軟體 GPU 下完整播映可能大幅慢於 BIOS 節拍。
        for _ in $(seq 1 90); do
          if grep -Fq "[cutscene] title menu ready" "$MODE.log"; then break; fi
          key Escape
          sleep 0.3
        done
        grep -Fq "[cutscene] title menu ready" "$MODE.log"
        sleep 1
        capture "$1"
      }
      wait_for_setting() {
        for _ in $(seq 1 160); do
          if test -f profile/fd2_re/fd2_settings.json && grep -q "\"theme_id\": \"$1\"" profile/fd2_re/fd2_settings.json; then
            echo "設定已寫入 $1"
            return
          fi
          sleep 0.5
        done
        echo "逾時等候美術設定 $1" >&2
        return 1
      }
      wait_for_art() {
        for _ in $(seq 1 100); do
          sleep 0.6
          capture "$1"
          if same "$1" title-hd.png; then return; fi
        done
        return 1
      }
      case "$MODE" in
        f2)
          first_visible_frame opening-before.png
          key F2
          wait_for_change opening-before.png opening-loading.png
          wait_for_setting modern-handpainted-a
          grep -q "\"interface_mode\": \"original\"" profile/fd2_re/fd2_settings.json
          wait_for_menu title-hd.png
          ;;
        cold)
          first_visible_frame cold-open.png
          wait_for_change cold-open.png cold-loading.png
          wait_for_menu title-hd-cold.png
          wait_for_art title-hd-cold.png
          grep -Fq "[theme] 主題：現代手繪" cold.log
          ;;
        missing)
          first_visible_frame missing-opening.png
          key F2
          wait_for_event "[theme] 現代主題不可用"
          wait_for_menu title-missing-pack.png
          different title-hd.png title-missing-pack.png
          test ! -e rejected/fd2_re/fd2_settings.json
          ;;
      esac
      ! grep -E "panicked at|^Update：|^loadGame：" "$MODE.log"
      echo "$MODE PASS"
    '
}

timeout 15s docker run "${common[@]}" --memory 128m --cpus 1 \
  --mount "type=bind,source=$out,target=/out" fd2-rust:1.99.1-r3 bash -euc '
    mkdir -p /out/profile/fd2_re
    printf "{\"theme_id\":\"faithful\",\"interface_mode\":\"original\"}\n" > /out/profile/fd2_re/fd2_settings.json
  '
window f2 profile /hd
drive f2
stop_window
window cold profile /hd
drive cold
stop_window
window missing rejected /missing
drive missing
stop_window
