#!/usr/bin/env bash
# play.sh — 用「正式 campaign 流程」啟動炎龍騎士團2 remake 供驗證。
#
# 為什麼要用這個而不是直接 ./fd2-linux：
#   ./fd2-linux 直接跑 = 開發用「快速單場戰鬥」(用 map0 roster 佔位資料,主角隊會堆在同一格)。
#   本腳本設 FD2_CAMPAIGN → 走正式流程:標題畫面 → START → 序章劇情 → 第一章戰鬥(ch01 散開部署)。
#
# 用法:
#   cd remake && ./play.sh              # 正式遊玩(需 X 桌面;或在有畫面的環境)
#   ./play.sh --shot <輸出.png> <幀>    # 截一張圖驗證(headless 也可,需 DISPLAY 指向 Xvfb)
#
# 前置:remake/assets/ 要有解好的素材(玩家自備原版跑 tools/export_*.py 解出);
#       開場過場需 ../org_game/炎龍騎士團/FLAME2/ANI.DAT(玩家自備原版)。
set -eu
cd "$(dirname "$(readlink -f "$0")")"   # 切到 remake/

BIN=./fd2-linux
if [ ! -x "$BIN" ]; then
  echo "找不到 $BIN,先 build:"
  echo "  docker run --rm -v \$PWD:/src -w /src -e GOCACHE=/src/.gocache fd2-build go build -o fd2-linux ./cmd/fd2"
  exit 1
fi

export FD2_CAMPAIGN="assets/scenarios/campaign_full.json"

if [ "${1:-}" = "--shot" ]; then
  # 截圖驗證模式:./play.sh --shot out.png 120
  export FD2_MUTE=1
  export FD2_SHOT="${2:?輸出檔}"
  export FD2_SHOT_FRAME="${3:-120}"
  echo "截圖:campaign 流程 @frame ${FD2_SHOT_FRAME} → ${FD2_SHOT}"
  exec "$BIN"
fi

echo "啟動正式 campaign 流程(標題→START→序章→第一章戰鬥)。"
echo "驗證重點:①開場過場動畫 ②主角隊在第一章是否散開站位(非堆疊) ③戰鬥/移動/指令環"
exec "$BIN"
