#!/bin/bash
# 在正常 Bevy 視窗操作工具列，驗證設定冷啟動、缺包回退及序章對白。
# 用法：FD2_RUST_TARGET=target-r5i bash tools/rust_r6_theme_window.sh work/<新收據目錄> [先前標題收據目錄] [interface-cold|no-font|posters|poster-scenery]
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
name=
stop_window() {
  test -n "$name" || return 0
  docker stop -t 3 "$name" >/dev/null 2>&1 || true
}
trap stop_window EXIT

window() {
  local mode="$1" profile="$2" pack="$3"
  local -a fonts=(--mount "type=bind,source=$ttf,target=/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc,readonly")
  if [[ "$mode" == "no-font" ]]; then fonts=(); fi
  name="fd2-r6-theme-window-$$-$mode"
  docker run -d "${common[@]}" --init --name "$name" --memory 6g --cpus 4 \
    --mount "type=bind,source=$build/release,target=/bins,readonly" \
    --mount "type=bind,source=$root/remake/assets,target=/assets,readonly" \
    --mount "type=bind,source=$orig,target=/original,readonly" \
    --mount "type=bind,source=$hd,target=/hd,readonly" \
    "${fonts[@]}" \
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
      width=1280
      if [[ "$MODE" == poster* ]]; then width=1808; fi
      Xvfb :96 -screen 0 "${width}x800x24" -nolisten tcp -ac >"/out/$MODE-xvfb.log" 2>&1 &
      xvfb=$!
      trap "kill $xvfb 2>/dev/null || true" EXIT
      for _ in $(seq 1 60); do test ! -S /tmp/.X11-unix/X96 || break; sleep 0.1; done
      test -S /tmp/.X11-unix/X96
      timeout 420s /bins/fd2 >"/out/$MODE.log" 2>&1
    ' >/dev/null
}

drive() {
  timeout 400s docker run "${common[@]}" --name "$name-driver" --memory 512m --cpus 1 \
    --mount "type=bind,source=$out,target=/out" \
    --mount "type=bind,source=$out/socket,target=/tmp/.X11-unix,readonly" \
    -e DISPLAY=:96 -e "MODE=$1" -w /out fd2-promo-local:latest bash -euc '
      trap "echo \"視窗測試 $MODE 失敗，腳本第 \$LINENO 行\" >&2" ERR
      for _ in $(seq 1 200); do
        wid="$(xdotool search --onlyvisible --name "fd2_re rust" 2>/dev/null | head -1)"
        test -z "$wid" || break
        sleep 0.1
      done
      test -n "$wid"
      xdotool windowfocus --sync "$wid"
      key() {
        xdotool windowfocus --sync "$wid"
        xdotool keydown "$1"
        sleep 0.45
        xdotool keyup "$1"
      }
      toolbar() {
        local x=$((2 * $1))
        xdotool windowfocus --sync "$wid"
        xdotool mousemove --window "$wid" "$x" 2
        sleep 0.6
        xdotool mousemove --window "$wid" "$x" 24 mousedown 1
        sleep 0.35
        xdotool mouseup 1 mousemove --window "$wid" 640 400
        for _ in $(seq 1 60); do
          capture /tmp/fd2-toolbar.png
          if test "$(convert /tmp/fd2-toolbar.png -format "%[pixel:p{0,0}]" info:)" != "srgb(17,24,37)"; then
            return
          fi
          sleep 0.2
        done
      }
      capture() { import -window "$wid" -depth 8 -define png:exclude-chunks=date,time "$1"; }
      same() { compare -metric AE "$1" "$2" null: 2>/dev/null; }
      different() {
        local status=0
        compare -metric AE "$1" "$2" null: 2>/dev/null || status=$?
        test "$status" -eq 1
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
        echo "畫面未變更：$1" >&2
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
        grep -Fq "[cutscene] title menu ready" "$MODE.log" || { echo "未抵達標題選單：$MODE" >&2; return 1; }
        sleep 1
        capture "$1"
      }
      wait_for_preference() {
        local field="$1" value="$2"
        for _ in $(seq 1 160); do
          if test -f profile/fd2_re/fd2_settings.json && grep -q "\"$field\": \"$value\"" profile/fd2_re/fd2_settings.json; then
            echo "設定已寫入 $field=$value"
            return
          fi
          sleep 0.5
        done
        echo "逾時等候 $field=$value" >&2
        return 1
      }
      wait_for_setting() { wait_for_preference theme_id "$1"; }
      check_title_art() {
        local actual="$1" expected="$2"
        convert "$actual" -crop 1280x656+0+0 +repage /tmp/title-actual.png
        convert "$expected" -crop 1280x656+0+0 +repage /tmp/title-expected.png
        same /tmp/title-actual.png /tmp/title-expected.png
      }
      wait_for_picture() {
        for _ in $(seq 1 100); do
          sleep 0.6
          capture "$1"
          if same "$1" "$2"; then return; fi
        done
        echo "逾時等候相同畫面 $1 與 $2" >&2
        return 1
      }
      poster_point() {
        xdotool mousemove --window "$wid" "$((264 + 2 * $1))" "$((2 * $2))"
        sleep 0.55
      }
      poster_click() {
        poster_point "$1" "$2"
        xdotool mousedown 1
        sleep 0.35
        xdotool mouseup 1
        sleep 0.5
      }
      poster_menu() {
        poster_point 564 1
        poster_click 564 12
      }
      poster_capture() {
        poster_point 320 200
        for _ in $(seq 1 60); do
          sleep 0.4
          capture "$1"
          if test "$(convert "$1" -format "%[pixel:p{264,0}]" info:)" != "srgb(17,24,37)"; then
            return
          fi
        done
        echo "海報工具列未收合：$1" >&2
        return 1
      }
      compare_poster_center() {
        convert "$1" -crop 1280x656+264+0 +repage /tmp/poster-center.png
        convert poster-off.png -crop 1280x656+264+0 +repage /tmp/poster-baseline.png
        same /tmp/poster-center.png /tmp/poster-baseline.png
      }
      case "$MODE" in
        f2)
          first_visible_frame opening-before.png
          toolbar 105
          wait_for_change opening-before.png opening-loading.png
          wait_for_setting modern-handpainted-a
          grep -q "\"interface_mode\": \"original\"" profile/fd2_re/fd2_settings.json
          wait_for_menu title-hd.png
          ;;
        cold)
          first_visible_frame cold-open.png
          wait_for_change cold-open.png cold-loading.png
          wait_for_menu title-hd-cold.png
          wait_for_picture title-hd-cold.png title-hd.png
          grep -Fq "[theme] 主題：現代手繪" cold.log
          ;;
        missing)
          first_visible_frame missing-opening.png
          toolbar 105
          wait_for_event "[theme] 現代主題不可用"
          wait_for_menu title-missing-pack.png
          different title-hd.png title-missing-pack.png
          test ! -e rejected/fd2_re/fd2_settings.json
          ;;
        toggle)
          first_visible_frame toggle-opening.png
          wait_for_menu title-before-toggle.png
          wait_for_picture title-before-toggle.png title-hd.png
          key Down
          wait_for_change title-before-toggle.png title-load-selected.png
          convert title-before-toggle.png -crop 1280x656+0+0 +repage title-background.png
          convert title-load-selected.png -crop 1280x656+0+0 +repage title-load-background.png
          same title-background.png title-load-background.png
          key Up
          wait_for_picture title-start-selected.png title-before-toggle.png
          key F2
          key F11
          sleep 1
          wait_for_picture title-shortcuts-retired.png title-start-selected.png
          toolbar 180
          wait_for_preference interface_mode hd
          wait_for_change title-start-selected.png title-hd-interface-hd-art.png
          check_title_art title-hd-interface-hd-art.png title-hd.png
          toolbar 45
          wait_for_setting faithful
          wait_for_change title-hd-interface-hd-art.png title-hd-interface-original-art.png
          check_title_art title-hd-interface-original-art.png title-missing-pack.png
          different title-hd-interface-original-art.png title-missing-pack.png
          toolbar 180
          wait_for_preference interface_mode original
          wait_for_picture title-original-toggle.png title-missing-pack.png
          toolbar 105
          wait_for_setting modern-handpainted-a
          wait_for_picture title-hd-toggle.png title-hd.png
          toolbar 180
          wait_for_preference interface_mode hd
          wait_for_picture title-both-restored.png title-hd-interface-hd-art.png
          key Return
          wait_for_event "beat op=dialog "
          sleep 1
          first_visible_frame story-hd.png
          different title-hd-toggle.png story-hd.png
          toolbar 45
          wait_for_setting faithful
          sleep 1
          first_visible_frame story-original.png
          different story-hd.png story-original.png
          toolbar 105
          wait_for_setting modern-handpainted-a
          sleep 1
          first_visible_frame story-hd-restored.png
          different story-original.png story-hd-restored.png
          test "$(grep -c "beat op=dialog " toggle.log)" -eq 1
          wait_for_preference interface_mode hd
          ;;
        interface-cold)
          first_visible_frame interface-opening.png
          wait_for_menu title-interface-cold.png
          wait_for_picture title-interface-cold.png title-hd-interface-hd-art.png
          toolbar 590
          wait_for_change title-interface-cold.png controls-from-toolbar.png
          key Escape
          wait_for_picture title-controls-closed.png title-interface-cold.png
          toolbar 270
          wait_for_preference bgm_source mt32
          toolbar 345
          grep -q "\"font_scale\": 1.1" profile/fd2_re/fd2_settings.json
          toolbar 400
          grep -q "\"game_speed\": 2" profile/fd2_re/fd2_settings.json
          ;;
        no-font)
          first_visible_frame no-font-opening.png
          wait_for_menu title-no-font.png
          wait_for_picture title-no-font.png title-missing-pack.png
          grep -q "\"interface_mode\":\"hd\"" no-font/fd2_re/fd2_settings.json
          ;;
        posters)
          xdotool windowsize "$wid" 1808 800
          wait_for_menu poster-off.png
          poster_menu
          capture poster-menu.png
          poster_click 495 57
          wait_for_preference set hd
          key Escape
          poster_capture poster-hd.png
          compare_poster_center poster-hd.png
          different poster-hd.png poster-off.png
          poster_menu
          poster_click 559 90
          poster_click 466 118
          poster_click 578 57
          wait_for_preference set classic
          grep -q "\"edge\": 72" profile/fd2_re/fd2_settings.json
          grep -q "\"fade_ms\": 300" profile/fd2_re/fd2_settings.json
          key Escape
          poster_capture poster-classic.png
          compare_poster_center poster-classic.png
          different poster-hd.png poster-classic.png
          poster_click -1 200
          poster_click 641 200
          poster_capture poster-after-side-click.png
          compare_poster_center poster-after-side-click.png
          ! grep -Fq "beat op=act source=0x32343" "$MODE.log"
          ;;
        poster-cold)
          wait_for_menu poster-cold.png
          compare_poster_center poster-cold.png
          convert poster-cold.png -crop 264x800+0+0 +repage /tmp/poster-cold.png
          convert poster-classic.png -crop 264x800+0+0 +repage /tmp/poster-classic.png
          same /tmp/poster-cold.png /tmp/poster-classic.png
          poster_menu
          poster_click 413 57
          wait_for_preference set off
          key Escape
          poster_capture poster-off-after.png
          compare_poster_center poster-off-after.png
          ;;
        poster-missing)
          xdotool windowsize "$wid" 1808 800
          wait_for_menu poster-missing.png
          compare_poster_center poster-missing.png
          grep -q "海報不可用" "$MODE.log"
          grep -q "\"set\":\"classic\"" rejected/fd2_re/fd2_settings.json
          ;;
        poster-scenery)
          wait_for_menu poster-neutral.png
          test "$(convert poster-neutral.png -crop 1x400+263+0 -format "%k" info:)" -eq 1
          key Return
          wait_for_event "beat op=dialog source=0x32382"
          poster_capture poster-native-scenery.png
          test "$(convert poster-native-scenery.png -crop 1x400+263+0 -format "%k" info:)" -gt 4
          poster_point 105 1
          poster_click 105 12
          wait_for_event "[theme] 主題：現代手繪"
          poster_capture poster-hd-scenery.png
          test "$(convert poster-hd-scenery.png -crop 1x400+263+0 -format "%k" info:)" -gt 4
          different poster-native-scenery.png poster-hd-scenery.png
          ;;
      esac
      if grep -E "panicked at|^Update：|^loadGame：" "$MODE.log"; then exit 1; fi
      echo "$MODE PASS"
    '
}

baseline=()
if [[ -n "${2:-}" ]]; then
  test -f "$2/title-hd.png" && test -f "$2/title-missing-pack.png"
  baseline=(--mount "type=bind,source=$(cd "$2" && pwd -P),target=/baseline,readonly")
fi
mode="${3:-toggle}"
if [[ "$mode" == "interface-cold" ]]; then
  test -f "${2:?需要標題基準}/title-hd-interface-hd-art.png"
elif [[ "$mode" == "no-font" ]]; then
  test -f "${2:?需要標題基準}/title-missing-pack.png"
elif [[ "$mode" != "posters" && "$mode" != "poster-scenery" ]]; then
  test "$mode" = toggle
fi
timeout 15s docker run "${common[@]}" --memory 128m --cpus 1 "${baseline[@]}" \
  -e "CHECK_MODE=$mode" \
  --mount "type=bind,source=$out,target=/out" fd2-rust:1.99.1-r3 bash -euc '
    mkdir -p /out/profile/fd2_re
    theme=faithful
    interface=original
    if test -d /baseline; then
      cp /baseline/title-hd.png /baseline/title-missing-pack.png /out/
      theme=modern-handpainted-a
    fi
    if test "$CHECK_MODE" = interface-cold; then
      cp /baseline/title-hd-interface-hd-art.png /out/
      interface=hd
    fi
    printf "{\"theme_id\":\"%s\",\"interface_mode\":\"%s\"}\n" "$theme" "$interface" > /out/profile/fd2_re/fd2_settings.json
    mkdir -p /out/no-font/fd2_re
    printf "{\"theme_id\":\"faithful\",\"interface_mode\":\"hd\"}\n" > /out/no-font/fd2_re/fd2_settings.json
    if [[ "$CHECK_MODE" == poster* ]]; then
      mkdir -p /out/rejected/fd2_re
      printf "{\"poster\":{\"set\":\"classic\",\"edge\":72,\"fade_ms\":300}}\n" > /out/rejected/fd2_re/fd2_settings.json
      mkdir -p /out/scenery/fd2_re
      cp /out/rejected/fd2_re/fd2_settings.json /out/scenery/fd2_re/fd2_settings.json
    fi
  '
if [[ "$mode" == "poster-scenery" ]]; then
  window poster-scenery scenery /hd
  drive poster-scenery
  stop_window
  exit 0
fi
if [[ "$mode" == "posters" ]]; then
  window posters profile /hd
  drive posters
  stop_window
  window poster-cold profile /hd
  drive poster-cold
  stop_window
  window poster-missing rejected /missing
  drive poster-missing
  stop_window
  window poster-scenery scenery /hd
  drive poster-scenery
  stop_window
  exit 0
fi
if [[ "$mode" == "interface-cold" ]]; then
  window interface-cold profile /hd
  drive interface-cold
  stop_window
  exit 0
fi
if [[ "$mode" == "no-font" ]]; then
  window no-font no-font /hd
  drive no-font
  stop_window
  exit 0
fi
if [[ ${#baseline[@]} -eq 0 ]]; then
  window f2 profile /hd
  drive f2
  stop_window
  window cold profile /hd
  drive cold
  stop_window
  window missing rejected /missing
  drive missing
  stop_window
fi
window toggle profile /hd
drive toggle
stop_window
window interface-cold profile /hd
drive interface-cold
stop_window
window no-font no-font /hd
drive no-font
stop_window
