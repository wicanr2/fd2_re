#!/usr/bin/env bash
# 把炎龍騎士團2 的 15 首音樂預錄成 OGG(MT-32 音源),供重製版串流播放。
# 全管線:FDMUS XMI → xmi2mid.py → MIDI → munt(MT-32 模擬器)+ 真 MT-32 ROM → WAV → OGG(libvorbis)
#
# 為何 OGG:MT-32 要靠 Roland 版權 ROM 即時模擬,不便內建;改「預先用 ROM 錄成 OGG」隨重製散布,
#         玩家不需 ROM 即可聽到道地 MT-32 音色。OGG 壓縮率高(113MB WAV → ~14MB)、跨平台、可迴圈。
#
# 前置:munt-smf2wav image、MT-32 ROM 目錄(MT32_CONTROL.ROM+MT32_PCM.ROM)、ffmpeg(libvorbis)、
#       已解包的 FDMUS(unpack_dat.py)。
# 用法: ./tools/export_music_ogg.sh <raw/FDMUS> <ROM目錄> <out目錄> [vorbis品質0-10,預設4]
set -e
SRC="${1:?raw/FDMUS}"; ROM="${2:?ROM目錄}"; OUT="${3:?out目錄}"; Q="${4:-4}"
cd "$(dirname "$0")/.."
TMP="$(mktemp -d)"; mkdir -p "$OUT"
docker image inspect munt-smf2wav >/dev/null 2>&1 || { echo "缺 munt-smf2wav(見 dq3 docs/59)"; exit 1; }
command -v ffmpeg >/dev/null || { echo "缺 ffmpeg"; exit 1; }
n=0
for f in "$SRC"/*.bin; do
  base=$(basename "$f" .bin)
  python3 tools/xmi2mid.py "$f" "$TMP/$base.mid" >/dev/null 2>&1 || continue
  [ -f "$TMP/$base.mid" ] || continue
  docker run --rm -v "$TMP":/m -v "$ROM":/rom munt-smf2wav \
        -m /rom -i mt32 -f -o "/m/$base.wav" "/m/$base.mid" >/dev/null 2>&1 || { echo "  $base munt失敗"; continue; }
  ffmpeg -y -loglevel error -i "$TMP/$base.wav" -c:a libvorbis -q:a "$Q" -ar 44100 "$OUT/$base.ogg" \
    && n=$((n+1))
done
rm -rf "$TMP"
echo "完成:$n 首 MT-32 OGG → $OUT/  ($(du -sh "$OUT" 2>/dev/null|cut -f1))"
echo "→ 重製版把 $OUT/ 放進 assets;BGM 迴圈由引擎處理(整曲 loop,或之後補 loop 點 metadata)。"
