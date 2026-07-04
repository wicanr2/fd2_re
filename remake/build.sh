#!/usr/bin/env bash
# build.sh — docker 建置 fd2-linux,產物檔名帶 build 時間戳。
#
# 為什麼帶時間戳(2026-07-04 使用者要求):
#   直接覆寫 fd2-linux 會讓「我測的是哪一版」說不清——曾發生舊 binary 不認得新的
#   type:"cutscene" 節點、過場全被跳過直衝戰場,誤判成 code bug。帶時間戳後:
#   ① 產物 = fd2-linux-YYYYMMDD-HHMM(每次建置獨立、不互相覆蓋)
#   ② fd2-linux 軟連結 → 最新一版(play.sh 跑這個)
#   使用者回報時報檔名(play.sh 啟動會印),就能對到確切 commit/版本。
set -eu
cd "$(dirname "$(readlink -f "$0")")"   # 切到 remake/

STAMP=$(date +%Y%m%d-%H%M)
OUT="fd2-linux-${STAMP}"

echo "docker 建置 → ${OUT} …"
docker run --rm -v "$PWD":/src -w /src -e GOCACHE=/src/.gocache fd2-build \
  go build -o "$OUT" ./cmd/fd2

ln -sfn "$OUT" fd2-linux   # fd2-linux 永遠指向最新建置
echo "完成:$OUT(fd2-linux → $OUT)"
echo "git 短碼:$(git rev-parse --short HEAD 2>/dev/null || echo '?')"
