#!/usr/bin/env bash
# 把炎龍騎士團2 的 15 首音樂預錄成 OGG(MT-32/CM-32L 音源),供重製版串流播放。
# 全管線:FDMUS XMI → xmi2mid.py → MIDI → munt(MT-32 模擬器)+ 真 ROM → WAV → 峰值正規化 → OGG(libvorbis)
#
# 為何 OGG:MT-32 要靠 Roland 版權 ROM 即時模擬,不便內建;改「預先用 ROM 錄成 OGG」隨重製散布,
#         玩家不需 ROM 即可聽到道地 MT-32 音色。OGG 壓縮率高(113MB WAV → ~14MB)、跨平台、可迴圈。
#
# 「氣勢偏弱」排查結論(第15輪,詳見 doc12):
#   - Reverb:munt 預設輸出就是「乾+濕」混音(MT-32 開機預設 Reverb=ON),
#     用 `-w 4 -w 5` dump 純 wet stream 實測非靜音,證實不需額外 flag——不是 reverb 沒開的問題。
#   - 真正原因是**跨曲峰值正規化缺失**:15 首裡有 3 首(FDMUS_011/013/014)munt 原始渲染
#     只到 -2.1~-3.5dBFS 就見頂(其餘 12 首本來就頂到 0dBFS),量起來明顯比同輯其他曲小聲、
#     顯得「氣勢弱」。故本輪加入**線性峰值正規化**(非動態壓縮,不動態值,只補回被浪費的空間)。
#
# 前置:munt-smf2wav image、ROM 目錄(MT32_CONTROL.ROM+MT32_PCM.ROM,或 CM32L_CONTROL.ROM+CM32L_PCM.ROM)、
#       ffmpeg(libvorbis)、已解包的 FDMUS(unpack_dat.py)。
# 用法: ./tools/export_music_ogg.sh <raw/FDMUS> <ROM目錄> <out目錄> [vorbis品質0-10,預設4] [峰值目標dBFS,預設-1.0] [machine-id,預設mt32]
set -e
SRC="${1:?raw/FDMUS}"; ROM="${2:?ROM目錄}"; OUT="${3:?out目錄}"; Q="${4:-4}"; PEAK="${5:--1.0}"; MACHINE="${6:-mt32}"
cd "$(dirname "$0")/.."
TMP="$(mktemp -d)"; mkdir -p "$OUT"
docker image inspect munt-smf2wav >/dev/null 2>&1 || { echo "缺 munt-smf2wav(見 dq3 docs/59)"; exit 1; }
command -v ffmpeg >/dev/null || { echo "缺 ffmpeg"; exit 1; }
n=0
for f in "$SRC"/*.bin; do
  base=$(basename "$f" .bin)
  python3 tools/xmi2mid.py "$f" "$TMP/$base.mid" >/dev/null 2>&1 || continue
  [ -f "$TMP/$base.mid" ] || continue
  # -a 0(analog filter DISABLED,munt 預設)刻意不開類比低通模擬:開了會悶化高頻、更弱氣勢,
  # 與「還原真實硬體」無關(那顆濾波器是類比電路的雜訊,非音樂內容)。reverb 隨預設值混入,不需額外 flag。
  docker run --rm -v "$TMP":/m -v "$ROM":/rom munt-smf2wav \
        -m /rom -i "$MACHINE" -f -o "/m/$base.wav" "/m/$base.mid" >/dev/null 2>&1 || { echo "  $base munt失敗"; continue; }
  # 峰值正規化:量測該曲實際 max_volume,線性增益補到 $PEAK dBFS(不動態壓縮,保留原始動態範圍)。
  # 注意:volumedetect 那行 log 開頭有 `@ 0x...` hex 位址,不能用裸 grep -oE 數字(會誤抓位址裡的數字),
  # 必須用 sed 錨定 "max_volume:" 後面那段(踩過一次的坑)。
  maxvol=$(ffmpeg -i "$TMP/$base.wav" -af volumedetect -f null - 2>&1 \
    | grep max_volume | sed -E 's/.*max_volume:[[:space:]]*(-?[0-9.]+) dB.*/\1/')
  gain=$(python3 -c "print(${PEAK} - (${maxvol:-0}))")
  ffmpeg -y -loglevel error -i "$TMP/$base.wav" -af "volume=${gain}dB" -c:a libvorbis -q:a "$Q" -ar 44100 "$OUT/$base.ogg" \
    && n=$((n+1))
done
rm -rf "$TMP"
echo "完成:$n 首 $MACHINE OGG(峰值正規化至 ${PEAK}dBFS)→ $OUT/  ($(du -sh "$OUT" 2>/dev/null|cut -f1))"
echo "→ 重製版把 $OUT/ 放進 assets;BGM 迴圈由引擎處理(整曲 loop,或之後補 loop 點 metadata)。"
