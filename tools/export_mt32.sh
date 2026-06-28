#!/usr/bin/env bash
# 把炎龍騎士團2 的 15 首 XMI 音樂渲染成「道地 Roland MT-32」WAV。
# FD2 原生支援 MT-32(隨遊戲附 MT32MPU.MDI 驅動),故 MT-32 版是還原原意,非腦補。
#
# 管線:FDMUS XMI → tools/xmi2mid.py → 標準 MIDI → munt(MT-32 模擬器)+ 真 MT-32 ROM → WAV
#
# 前置:
#   1. munt-smf2wav Docker image(見 ~/dq3/docs/59-munt-mt32-build.md 的 Dockerfile)
#   2. MT-32 ROM(玩家自備,從自有實機 dump;Roland 版權,不散布):
#        $ROMDIR/MT32_CONTROL.ROM(64KB) + MT32_PCM.ROM(512KB)
#   3. 已解包的 FDMUS(unpack_dat.py → raw/FDMUS/*.bin)
#
# 用法:  ./tools/export_mt32.sh <raw/FDMUS目錄> <ROM目錄> <輸出目錄>
set -e
SRC="${1:?raw/FDMUS 目錄}"; ROM="${2:?MT-32 ROM 目錄}"; OUT="${3:?輸出目錄}"
cd "$(dirname "$0")/.."
TMP="$(mktemp -d)"; mkdir -p "$OUT"
docker image inspect munt-smf2wav >/dev/null 2>&1 || { echo "缺 munt-smf2wav image(見 dq3 docs/59)"; exit 1; }

echo "[1/2] XMI → MIDI…"
for f in "$SRC"/*.bin; do
  base=$(basename "$f" .bin)
  python3 tools/xmi2mid.py "$f" "$TMP/$base.mid" >/dev/null 2>&1 || true
done

echo "[2/2] munt MT-32 render → WAV…"
n=0
for mid in "$TMP"/*.mid; do
  [ -f "$mid" ] || continue
  base=$(basename "$mid" .mid)
  docker run --rm -v "$TMP":/m -v "$ROM":/rom munt-smf2wav \
        -m /rom -i mt32 -f -o "/m/$base.wav" "/m/$base.mid" >/dev/null 2>&1 \
    && { cp "$TMP/$base.wav" "$OUT/"; n=$((n+1)); } || echo "  $base render 失敗"
done
rm -rf "$TMP"
echo "完成:$n 首 MT-32 WAV → $OUT/"
echo "(換 CM-32L:ROM 改用 CM32L_CONTROL/PCM,munt -i cm32l)"
