# shellcheck shell=bash
# R5 r5f（docs/goal/121，使用者 2026-11-15 定案）：跑 fd2-game 的驅動腳本以 `source` 載入。
#
# Go 黃金（fd2-go-test-local 的 fonts-noto-cjk）的 loadFont 讀系統的 NotoSansCJK-Regular.ttc；Rust 測試映像
# fd2-rust 沒有這個字型，所以把主機同一檔唯讀掛到容器內同一路徑（Go fontPaths 的第三項）。執行前核對 SHA-256，
# 缺檔或不符即失敗；容器內設 FD2_REQUIRE_ORIGINAL=1，Rust 端看不到字型時同樣失敗，不悄悄跳過
# （教訓 rust-test-env-unified-fail-closed）。
#
# 提供：ttf_font（主機路徑，可用 FD2_TTF_FONT 指定）、ttf_font_path（容器內路徑）、ttf_font_sha、
#       ttf_mounts（放進 Rust 的 docker run 參數）。
ttf_font_path=/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc
ttf_font_sha=b76b0433203017ca80401b2ee0dd69350349871c4b19d504c34dbdd80541690a
ttf_font="${FD2_TTF_FONT:-$ttf_font_path}"
test -f "$ttf_font" || { echo "缺少 TTF 字型 $ttf_font（Go 黃金用的 NotoSansCJK-Regular.ttc）" >&2; exit 2; }
test "$(sha256sum "$ttf_font" | cut -c1-64)" = "$ttf_font_sha" \
  || { echo "$ttf_font 的 SHA-256 不是 $ttf_font_sha（不是 Go 黃金用的那一版）" >&2; exit 2; }
ttf_mounts=(--mount "type=bind,source=$ttf_font,target=$ttf_font_path,readonly" -e FD2_REQUIRE_ORIGINAL=1)
