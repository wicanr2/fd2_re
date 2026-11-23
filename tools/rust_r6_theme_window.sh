#!/bin/bash
# 在正常 Bevy 視窗操作工具列，驗證設定冷啟動、缺包回退及序章對白。
# 用法：FD2_RUST_TARGET=target-r5i bash tools/rust_r6_theme_window.sh work/<新收據目錄> [先前收據或存檔目錄] [interface-cold|no-font|posters|poster-scenery|locales|locale-samples|gold-dialogue|loadslots|native-save] [原版城鎮參考圖]
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
native_reference=()
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
      if [[ "$MODE" == native-save ]]; then export FD2_NATIVE_SAVE=/out/profile/FD2.SAV; fi
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
    "${native_reference[@]}" \
    -e DISPLAY=:96 -e "MODE=$1" -e "CHECK_SHOP_PURCHASE=${FD2_R6_SHOP_PURCHASE:-0}" \
    -e "CHECK_SHOP_SELL=${FD2_R6_SHOP_SELL:-0}" -w /out fd2-promo-local:latest bash -euc '
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
        local rest_x=640 rest_y=400
        if [[ "$MODE" == loadslots* || "$MODE" == native-save ]]; then rest_x=30; rest_y=70; fi
        xdotool windowfocus --sync "$wid"
        xdotool mousemove --window "$wid" "$x" 2
        sleep 0.6
        xdotool mousemove --window "$wid" "$x" 24 mousedown 1
        sleep 0.35
        xdotool mouseup 1 mousemove --window "$wid" "$rest_x" "$rest_y"
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
        for _ in $(seq 1 "${2:-220}"); do
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
        local settings=profile/fd2_re/fd2_settings.json
        if [[ "$MODE" == loadslots-native ]]; then settings=empty/fd2_re/fd2_settings.json; fi
        for _ in $(seq 1 160); do
          if test -f "$settings" && grep -q "\"$field\": \"$value\"" "$settings"; then
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
      capture_locale_dialogue() {
        key Return
        wait_for_event "beat op=dialog source=0x32382" 400
        wait_for_event "locale page ready: $2 page=0"
        sleep 1
        capture "$1"
      }
      locale_text_region() { convert "$1" -crop 850x240+398+464 +repage "$2"; }
      check_dialogue_border() {
        convert "$1" -crop 1240x12+20+448 +repage /tmp/border-before.png
        convert "$2" -crop 1240x12+20+448 +repage /tmp/border-after.png
        different /tmp/border-before.png /tmp/border-after.png
        locale_text_region "$1" /tmp/text-before.png
        locale_text_region "$2" /tmp/text-after.png
        same /tmp/text-before.png /tmp/text-after.png
      }
      open_loadslots() {
        local previous
        previous=$(grep -Fc "[title] load slots opened" "$MODE.log" || true)
        for _ in $(seq 1 3); do
          xdotool mousemove --window "$wid" 640 705
          sleep 1
          xdotool mousedown 1
          sleep 0.45
          xdotool mouseup 1 mousemove --window "$wid" 30 70
          for _ in $(seq 1 60); do
            if test "$(grep -Fc "[title] load slots opened" "$MODE.log" || true)" -gt "$previous"; then
              sleep 1
              capture "$1"
              return
            fi
            sleep 0.5
          done
        done
        echo "未開啟標題讀檔槽：$MODE" >&2
        return 1
      }
      loadslots_key_until_event() {
        local event="$1" button
        shift
        for button in "$@"; do
          if grep -Fq "$event" "$MODE.log"; then return; fi
          key "$button"
          for _ in $(seq 1 12); do
            if grep -Fq "$event" "$MODE.log"; then return; fi
            sleep 0.5
          done
        done
        wait_for_event "$event" 12
      }
      hover_second_slot() {
        local row_y="$1" region="$2"
        convert slots-hd.png -crop "$region" +repage /tmp/slots-first-choice.png
        xdotool mousemove --window "$wid" 410 "$row_y"
        for _ in $(seq 1 35); do
          capture slots-second.png
          convert slots-second.png -crop "$region" +repage /tmp/slots-second-choice.png
          if different /tmp/slots-first-choice.png /tmp/slots-second-choice.png; then return; fi
          sleep 0.25
        done
        echo "滑鼠未選到第二槽：$MODE" >&2
        return 1
      }
      wait_for_restored_town() {
        local shot="${1:-slots-restored.png}"
        # 同狀態的原版畫格為 320×200。比對沒有游標、人物或標籤的右側營帳區。
        convert /native-town.png -filter point -resize 400% -crop 480x280+680+240 +repage /tmp/town-original.png
        for _ in $(seq 1 50); do
          capture "$shot"
          convert "$shot" -crop 480x280+680+240 +repage /tmp/town-rust.png
          if same /tmp/town-original.png /tmp/town-rust.png; then
            echo "原版城鎮營帳區逐像素相同"
            return
          fi
          sleep 0.4
        done
        compare -metric AE /tmp/town-original.png /tmp/town-rust.png null: || true
        echo "還原畫面未匹配原版城鎮" >&2
        return 1
      }
      click_chapter_toolbar() {
        xdotool mousemove --window "$wid" 1248 2
        sleep 1
        xdotool mousemove --window "$wid" 1248 24 mousedown 1
        sleep 0.45
        xdotool mouseup 1
      }
      verify_chapter_guide() {
        click_chapter_toolbar
        for _ in $(seq 1 25); do
          capture chapter-guide.png
          if test "$(convert chapter-guide.png -format "%[pixel:p{100,80}]" info:)" = "srgb(17,24,37)"; then
            xdotool mousemove --window "$wid" 940 110 mousedown 1
            sleep 0.45
            xdotool mouseup 1
            wait_for_change chapter-guide.png chapter-guide-details.png
            key Escape
            return
          fi
          sleep 0.25
        done
        echo "工具列未打開關卡事件" >&2
        return 1
      }
      verify_secret_shop_shortcut() {
        convert slots-restored.png -crop 480x280+680+240 +repage /tmp/loaded-town.png
        for _ in $(seq 1 3); do
          click_chapter_toolbar
          for _ in $(seq 1 15); do
            capture slots-secret-shop.png
            convert slots-secret-shop.png -crop 480x280+680+240 +repage /tmp/loaded-shop.png
            if different /tmp/loaded-town.png /tmp/loaded-shop.png; then
              sleep 1
              capture slots-secret-shop.png
              if [[ "$CHECK_SHOP_SELL" == 1 ]]; then
                verify_secret_shop_sell
              else
                verify_secret_shop_browse
              fi
              return
            fi
            sleep 0.25
          done
        done
        echo "工具列未進入祕密商店" >&2
        return 1
      }
      verify_secret_shop_browse() {
        convert slots-secret-shop.png -crop 340x190+60+550 +repage /tmp/shop-menu.png
        for _ in $(seq 1 3); do
          key Return
          for _ in $(seq 1 14); do
            capture slots-secret-stock.png
            convert slots-secret-stock.png -crop 340x190+60+550 +repage /tmp/shop-stock.png
            if different /tmp/shop-menu.png /tmp/shop-stock.png; then
              if [[ "$CHECK_SHOP_PURCHASE" == 1 ]]; then
                verify_secret_shop_purchase
              fi
              key Escape
              for _ in $(seq 1 25); do
                capture slots-secret-menu.png
                convert slots-secret-menu.png -crop 340x190+60+550 +repage /tmp/shop-return.png
                if same /tmp/shop-menu.png /tmp/shop-return.png; then
                  sleep 1
                  key Escape
                  wait_for_restored_town slots-town-returned.png
                  return
                fi
                sleep 0.35
              done
              break
            fi
            sleep 0.25
          done
        done
        echo "祕密商店商品瀏覽或返回未完成" >&2
        return 1
      }
      verify_secret_shop_purchase() {
        move_secret_shop_selection Right
        move_secret_shop_selection Down
        capture shop-purchase-selected.png
        loadslots_key_until_event "[shop] purchase confirmation index=3" Return space
        wait_for_change shop-purchase-selected.png shop-purchase-confirm.png
        loadslots_key_until_event "[shop] recipient selection item=192" Return space
        wait_for_change shop-purchase-confirm.png shop-purchase-recipient.png
        loadslots_key_until_event "[shop] purchase staged item=192" Return space Return
        wait_for_event "[shop] purchase committed item=192"
        wait_for_event "[shop] purchase list ready"
        grep -Eq "\[shop\] purchase committed item=192 recipient=[0-9]+ gold=990$" "$MODE.log"
        test "$(grep -Fc "[shop] purchase committed" "$MODE.log")" -eq 1
        capture shop-purchase-complete.png
      }
      verify_secret_shop_sell() {
        key Right
        loadslots_key_until_event "[shop] sell_roster ready" Return space Return
        wait_for_change slots-secret-shop.png shop-sell-roster.png
        loadslots_key_until_event "[shop] sell_items ready" Return space Return
        wait_for_change shop-sell-roster.png shop-sell-items.png
        sleep 1
        move_secret_shop_selection Down
        capture shop-sell-selected.png
        loadslots_key_until_event "[shop] sell_confirm ready" Return space Return
        wait_for_change shop-sell-selected.png shop-sell-confirm.png
        loadslots_key_until_event "[shop] sale staged item=192" Return space Return
        wait_for_event "[shop] sale committed item=192 recipient=0 gold=1007"
        test "$(grep -Fc "[shop] sale committed" "$MODE.log")" -eq 1
        loadslots_key_until_event "[shop] sell menu returned" Escape Escape
        sleep 1
        capture shop-sell-complete.png
        key Escape
        wait_for_restored_town slots-town-returned.png
      }
      move_secret_shop_selection() {
        local button="$1"
        capture /tmp/shop-cursor-before.png
        convert /tmp/shop-cursor-before.png -crop 1240x310+20+460 +repage /tmp/shop-items-before.png
        for _ in $(seq 1 3); do
          key "$button"
          for _ in $(seq 1 10); do
            capture /tmp/shop-cursor-after.png
            convert /tmp/shop-cursor-after.png -crop 1240x310+20+460 +repage /tmp/shop-items-after.png
            if different /tmp/shop-items-before.png /tmp/shop-items-after.png; then return; fi
            sleep 0.35
          done
        done
        echo "祕密商店商品選取未改變：$button" >&2
        return 1
      }
      wait_for_loaded_dialogue() {
        for _ in $(seq 1 60); do
          capture "$1"
          if test "$(convert "$1" -format "%[pixel:p{640,418}]" info:)" = "srgb(40,69,138)"; then
            return
          fi
          sleep 0.25
        done
        echo "首章讀檔對白未展開：$MODE" >&2
        return 1
      }
      advance_loaded_dialogue() {
        for _ in $(seq 1 3); do
          key Return
          for _ in $(seq 1 10); do
            capture slots-battle-next.png
            convert slots-battle-next.png -crop 1160x330+60+430 +repage /tmp/slots-battle-next-content.png
            if different /tmp/slots-battle-original-content.png /tmp/slots-battle-next-content.png; then return; fi
            sleep 0.35
          done
        done
        echo "首章讀檔對白未前進：$MODE" >&2
        return 1
      }
      check_loadslots_border() {
        for name in original hd; do
          convert "slots-$name.png" -crop "$1" +repage "/tmp/slots-$name-edge.png"
          convert "slots-$name.png" -crop "$2" +repage "/tmp/slots-$name-inside.png"
        done
        different /tmp/slots-original-edge.png /tmp/slots-hd-edge.png
        same /tmp/slots-original-inside.png /tmp/slots-hd-inside.png
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
        locale-cycle)
          wait_for_menu locale-title.png
          toolbar 300
          wait_for_preference locale_id zh-Hans
          toolbar 300
          wait_for_preference locale_id ja
          toolbar 300
          wait_for_preference locale_id en
          key F4
          grep -q "\"locale_id\": \"en\"" profile/fd2_re/fd2_settings.json
          capture_locale_dialogue locale-en.png en
          for _ in $(seq 1 30); do
            if test "$(grep -Fc "beat op=dialog source=" "$MODE.log")" -ge 2; then break; fi
            key Return
            sleep 0.6
          done
          test "$(grep -Fc "beat op=dialog source=" "$MODE.log")" -ge 2
          ;;
        locale-cold)
          wait_for_menu locale-cold-title.png
          capture_locale_dialogue locale-en-cold.png en
          locale_text_region locale-en.png /tmp/locale-en.png
          locale_text_region locale-en-cold.png /tmp/locale-en-cold.png
          same /tmp/locale-en.png /tmp/locale-en-cold.png
          ;;
        locale-ja|locale-hans)
          wait_for_menu "$MODE-title.png"
          locale=ja
          if [[ "$MODE" == locale-hans ]]; then locale=zh-Hans; fi
          capture_locale_dialogue "$MODE.png" "$locale"
          locale_text_region "$MODE.png" /tmp/locale-actual.png
          locale_text_region locale-en.png /tmp/locale-en.png
          different /tmp/locale-actual.png /tmp/locale-en.png
          ;;
        gold-dialogue)
          wait_for_menu gold-title.png
          toolbar 180
          wait_for_preference interface_mode hd
          capture_locale_dialogue gold-original-art.png en
          toolbar 180
          wait_for_preference interface_mode original
          capture original-original-art.png
          check_dialogue_border gold-original-art.png original-original-art.png
          toolbar 105
          wait_for_setting modern-handpainted-a
          capture original-hd-art.png
          toolbar 180
          wait_for_preference interface_mode hd
          capture gold-hd-art.png
          check_dialogue_border original-hd-art.png gold-hd-art.png
          convert gold-original-art.png -crop 1240x12+20+448 +repage /tmp/gold-first.png
          same /tmp/gold-first.png /tmp/border-after.png
          key Return
          wait_for_event "locale page ready: en page=1"
          sleep 1
          capture gold-hd-art-page2.png
          convert gold-hd-art-page2.png -crop 1240x12+20+448 +repage /tmp/gold-second.png
          same /tmp/gold-first.png /tmp/gold-second.png
          ;;
        loadslots|loadslots-native|native-save)
          wait_for_menu slots-title.png
          open_loadslots slots-original.png
          toolbar 180
          wait_for_preference interface_mode hd
          sleep 1
          capture slots-hd.png
          if [[ "$MODE" == loadslots ]]; then
            check_loadslots_border 720x2+280+140 680x440+296+170
            hover_second_slot 340 680x440+296+170
          else
            check_loadslots_border 1240x2+20+448 1200x300+40+466
            hover_second_slot 580 1200x300+40+466
          fi
          loadslots_key_until_event "[title] slot 2 loaded=false" Return space Return
          wait_for_picture slots-empty-rejected.png slots-second.png
          loadslots_key_until_event "[title] load slots closed" Escape Escape Escape
          wait_for_change slots-empty-rejected.png slots-cancelled.png
          if [[ "$MODE" == loadslots ]]; then
            open_loadslots slots-reopened.png
            loadslots_key_until_event "[title] slot 1 loaded=true" Return space Return
            wait_for_loaded_dialogue slots-battle.png
            toolbar 180
            wait_for_preference interface_mode original
            wait_for_loaded_dialogue slots-battle-original.png
            for name in battle battle-original; do
              convert "slots-$name.png" -crop 1240x6+20+396 +repage "/tmp/slots-$name-border.png"
              convert "slots-$name.png" -crop 1160x330+60+430 +repage "/tmp/slots-$name-content.png"
            done
            different /tmp/slots-battle-border.png /tmp/slots-battle-original-border.png
            same /tmp/slots-battle-content.png /tmp/slots-battle-original-content.png
            advance_loaded_dialogue
            verify_chapter_guide
            sha256sum -c slot-source.sha256
          elif [[ "$MODE" == native-save ]]; then
            open_loadslots slots-reopened.png
            loadslots_key_until_event "[title] slot 1 loaded=true" Return space Return
            wait_for_restored_town
            verify_secret_shop_shortcut
            sha256sum -c slot-source.sha256
          fi
          ;;
      esac
      if grep -E "panicked at|^Update：|^loadGame：" "$MODE.log"; then exit 1; fi
      echo "$MODE PASS"
    '
}

mode="${3:-toggle}"
baseline=()
if [[ -n "${2:-}" ]]; then
  if [[ "$mode" == "loadslots" ]]; then
    test -f "$2/fd2_save.json"
  elif [[ "$mode" == "native-save" ]]; then
    test -f "$2/FD2.SAV"
  elif [[ "$mode" == "locale-samples" ]]; then
    test -f "$2/locale-en.png" && test -f "$2/profile/fd2_re/fd2_settings.json"
  else
    test -f "$2/title-hd.png" && test -f "$2/title-missing-pack.png"
  fi
  baseline=(--mount "type=bind,source=$(cd "$2" && pwd -P),target=/baseline,readonly")
fi
if [[ "$mode" == "interface-cold" ]]; then
  test -f "${2:?需要標題基準}/title-hd-interface-hd-art.png"
elif [[ "$mode" == "no-font" ]]; then
  test -f "${2:?需要標題基準}/title-missing-pack.png"
elif [[ "$mode" == "locale-samples" ]]; then
  test -f "${2:?需要英文視窗基準}/locale-en.png"
elif [[ "$mode" == "loadslots" ]]; then
  test -f "${2:?需要章界存檔目錄}/fd2_save.json"
elif [[ "$mode" == "native-save" ]]; then
  test -f "${2:?需要原版章界存檔目錄}/FD2.SAV"
  reference="${4:?需要原版城鎮參考圖}"
  test -f "$reference"
  native_reference=(--mount "type=bind,source=$(cd "$(dirname "$reference")" && pwd -P)/$(basename "$reference"),target=/native-town.png,readonly")
elif [[ "$mode" != "posters" && "$mode" != "poster-scenery" && "$mode" != "locales" && "$mode" != "gold-dialogue" ]]; then
  test "$mode" = toggle
fi
timeout 15s docker run "${common[@]}" --memory 128m --cpus 1 "${baseline[@]}" \
  -e "CHECK_MODE=$mode" \
  --mount "type=bind,source=$out,target=/out" fd2-rust:1.99.1-r3 bash -euc '
    mkdir -p /out/profile/fd2_re
    theme=faithful
    interface=original
    if test -d /baseline && [[ "$CHECK_MODE" != locale-samples && "$CHECK_MODE" != loadslots && "$CHECK_MODE" != native-save ]]; then
      cp /baseline/title-hd.png /baseline/title-missing-pack.png /out/
      theme=modern-handpainted-a
    fi
    if test "$CHECK_MODE" = interface-cold; then
      cp /baseline/title-hd-interface-hd-art.png /out/
      interface=hd
    fi
    printf "{\"theme_id\":\"%s\",\"interface_mode\":\"%s\"}\n" "$theme" "$interface" > /out/profile/fd2_re/fd2_settings.json
    if [[ "$CHECK_MODE" == gold-dialogue ]]; then
      printf "{\"theme_id\":\"faithful\",\"locale_id\":\"en\",\"game_speed\":4}\n" > /out/profile/fd2_re/fd2_settings.json
    fi
    if [[ "$CHECK_MODE" == loadslots ]]; then
      cp /baseline/fd2_save.json /out/profile/fd2_re/fd2_save.json
      mkdir -p /out/empty/fd2_re
      for profile in profile empty; do
        printf "{\"theme_id\":\"faithful\",\"game_speed\":4}\n" > "/out/$profile/fd2_re/fd2_settings.json"
      done
      cd /out
      sha256sum profile/fd2_re/fd2_save.json > slot-source.sha256
    fi
    if [[ "$CHECK_MODE" == native-save ]]; then
      cp /baseline/FD2.SAV /out/profile/FD2.SAV
      printf "{\"theme_id\":\"faithful\",\"game_speed\":4}\n" > /out/profile/fd2_re/fd2_settings.json
      cd /out
      sha256sum profile/FD2.SAV > slot-source.sha256
    fi
    mkdir -p /out/no-font/fd2_re
    printf "{\"theme_id\":\"faithful\",\"interface_mode\":\"hd\"}\n" > /out/no-font/fd2_re/fd2_settings.json
    if [[ "$CHECK_MODE" == poster* ]]; then
      mkdir -p /out/rejected/fd2_re
      printf "{\"poster\":{\"set\":\"classic\",\"edge\":72,\"fade_ms\":300}}\n" > /out/rejected/fd2_re/fd2_settings.json
      mkdir -p /out/scenery/fd2_re
      cp /out/rejected/fd2_re/fd2_settings.json /out/scenery/fd2_re/fd2_settings.json
    fi
    if [[ "$CHECK_MODE" == locale* ]]; then
      mkdir -p /out/locale-ja/fd2_re /out/locale-hans/fd2_re
      printf "{\"locale_id\":\"ja\",\"game_speed\":4}\n" > /out/locale-ja/fd2_re/fd2_settings.json
      printf "{\"locale_id\":\"zh-Hans\",\"game_speed\":4}\n" > /out/locale-hans/fd2_re/fd2_settings.json
      if [[ "$CHECK_MODE" == locale-samples ]]; then
        cp /baseline/locale-en.png /out/
        sed -E "s/\"game_speed\":[[:space:]]*[0-9]+/\"game_speed\": 4/" \
          /baseline/profile/fd2_re/fd2_settings.json > /out/profile/fd2_re/fd2_settings.json
      fi
    fi
  '
if [[ "$mode" == "native-save" ]]; then
  window native-save profile /hd
  drive native-save
  stop_window
  exit 0
fi
if [[ "$mode" == "loadslots" ]]; then
  window loadslots profile /hd
  drive loadslots
  stop_window
  timeout 20s docker run "${common[@]}" --memory 128m --cpus 1 \
    --mount "type=bind,source=$out,target=/out" fd2-rust:1.99.1-r3 bash -euc '
      mkdir /out/json
      mv /out/slots-*.png /out/json/
    '
  window loadslots-native empty /hd
  drive loadslots-native
  stop_window
  exit 0
fi
if [[ "$mode" == "gold-dialogue" ]]; then
  window gold-dialogue profile /hd
  drive gold-dialogue
  stop_window
  exit 0
fi
if [[ "$mode" == "locales" || "$mode" == "locale-samples" ]]; then
  phases=(locale-cold locale-ja locale-hans)
  if [[ "$mode" == "locales" ]]; then phases=(locale-cycle "${phases[@]}"); fi
  for phase in "${phases[@]}"; do
    profile=profile
    if [[ "$phase" == locale-ja || "$phase" == locale-hans ]]; then profile="$phase"; fi
    window "$phase" "$profile" /hd
    drive "$phase"
    stop_window
  done
  exit 0
fi
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
