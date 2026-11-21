#!/bin/bash
# R5 階段 4 r5i（docs/goal/121）：滑鼠、手把與按鍵設定面板的局部黃金。
# Go 端（remake/cmd/fd2/rust_golden_r5i_controls_test.go，Xvfb）輸出鍵名、設定檔、輸入序列、指向換算與面板疊層；
# Rust 端（fd2-game 的 rust_golden_r5i_test.rs，#[ignore]）讀同一份輸出重跑並比對，報告寫到 <輸出目錄>/rust/report.json。
#
# 用法：tools/rust_r5i_controls.sh <輸出目錄>
# 輸出：<輸出目錄>/go/（keys.json、settings.json、input.json、pointer.json、panel.json、panel-*.rgba、manifest.txt、go.log）、
#       <輸出目錄>/rust/（report.json、rust.log）。
#
# 輸入（唯讀）：remake/（Go 測試與受版控素材）、主工作區的 remake/assets（FD2_DATA_ROOT）、tools/rust_ttf_font.sh 的 TTF 字型。
# 環境變數：
#   FD2_DATA_ROOT   主工作區（預設本儲存庫；工作樹副本指向主工作區）
#   FD2_GOCACHE     Go 建置快取（預設 <資料根>/work/gocache）
#   FD2_RUST_BUILD／FD2_RUST_TARGET 同 tools/rust_game_test.sh
#   FD2_R5I_SIDES   要跑的一側（預設 "go rust"）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
data="$(cd "${FD2_DATA_ROOT:-$root}" && pwd -P)"
out="$(mkdir -p "${1:?輸出目錄}" && cd "$1" && pwd -P)"
gocache="${FD2_GOCACHE:-$data/work/gocache}"
build="${FD2_RUST_BUILD:-$data/work/rust-build}"
target="${FD2_RUST_TARGET:-target}"
sides="${FD2_R5I_SIDES:-go rust}"
source "$root/tools/rust_ttf_font.sh"
for d in "$root/remake/cmd/fd2" "$root/remake/assets/locales" "$data/remake/assets" "$gocache" "$build/cargo"; do
  test -d "$d" || { echo "找不到目錄 $d" >&2; exit 2; }
done
for side in $sides; do
  sd="$out/$side"
  rm -rf "$sd"
  mkdir -p "$sd"
  echo "== $side"
  case $side in
    go)
      timeout 2400 docker run --rm --name "fd2-r5i-go-$$" --network none --memory 8g --cpus 4 --pids-limit 512 \
        --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
        -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod -e FD2_GOLDEN_R5I_OUT=/run \
        --mount "type=bind,source=$root,target=/src,readonly" \
        --mount "type=bind,source=$data/remake/assets,target=/mainassets,readonly" \
        --mount "type=bind,source=$gocache,target=/gocache" \
        --mount "type=bind,source=$sd,target=/run" \
        -w /src/remake fd2-go-test-local:latest with-xvfb bash -c '
          set -e; sha256sum '"$ttf_font_path"' > /run/font-sha256.txt
          mkdir -p /tmp/src; cp -a /src/remake /tmp/src/remake
          cp -a -n /mainassets/. /tmp/src/remake/assets/
          cd /tmp/src/remake
          go test ./cmd/fd2 -run "^TestRustGoldenR5iControls$" -count=1 -timeout 30m -v > /run/go.log 2>&1' \
        || { echo "Go 端失敗，見 $sd/go.log" >&2; tail -20 "$sd/go.log" >&2; exit 1; }
      test "$(cut -c1-64 "$sd/font-sha256.txt")" = "$ttf_font_sha" || { echo "Go 映像內的字型雜湊不同" >&2; exit 1; }
      for f in keys.json settings.json input.json pointer.json panel.json manifest.txt; do
        test -f "$sd/$f" || { echo "Go 沒有輸出 $f" >&2; exit 1; }
      done
      ;;
    rust)
      test -f "$out/go/manifest.txt" || { echo "先跑 Go 端" >&2; exit 2; }
      mkdir -p "$build/$target"
      timeout 3000s docker run --rm --name "fd2-r5i-rust-$$" --network none --memory 6g --cpus 4 --pids-limit 1024 \
        --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" -e HOME=/tmp/h \
        -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x -e FD2_REMAKE_ASSETS=/assets \
        -e FD2_R5I_GOLDEN_IN=/go -e FD2_R5I_GOLDEN_OUT=/run \
        --mount "type=bind,source=$sd,target=/run" \
        --mount "type=bind,source=$out/go,target=/go,readonly" \
        --mount "type=bind,source=$root/remake/assets,target=/assets,readonly" \
        "${ttf_mounts[@]}" \
        --mount "type=bind,source=$root/remake-rs,target=/src" \
        --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
        --mount "type=bind,source=$root/docs,target=/docs,readonly" \
        --mount "type=bind,source=$build/cargo,target=/build/cargo" \
        --mount "type=bind,source=$build/$target,target=/build/target-x" -w /src \
        fd2-rust:1.99.1-r3 bash -euo pipefail -c '
          cargo test --offline --locked -p fd2-game --release --lib -- --ignored --exact --nocapture \
            game::player_controls_panel::rust_golden_r5i_test::test_rust_golden_r5i_controls
        ' > "$sd/rust.log" 2>&1 || { echo "Rust 端失敗，見 $sd/rust.log" >&2; tail -30 "$sd/rust.log" >&2; exit 1; }
      test -f "$sd/report.json" || { echo "Rust 沒有輸出 report.json" >&2; exit 1; }
      cat "$sd/report.json"
      ;;
    *) echo "未知的一側 $side" >&2; exit 2 ;;
  esac
done
