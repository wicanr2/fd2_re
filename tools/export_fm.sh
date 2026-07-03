#!/usr/bin/env bash
# 把炎龍騎士團2 的 15 首 XMI 音樂渲染成「道地 Sound Blaster / AdLib(OPL2 FM)」WAV/OGG。
# FD2 出廠預設驅動就是 Sound Blaster(見 org_game/.../FLAME2/MDI.INI:DRIVER=SBLASTER.MDI),
# MT-32 反而是選配;本管線用**遊戲自帶的 SAMPLE.AD 音色庫**(非通用 GM 音色),故是還原原意。
#
# 管線:
#   FDMUS XMI --xmi2mid.py--> 標準 MIDI
#   SAMPLE.AD(Miles AIL2 GTL 音色庫)--gtl2wopl.py--> WOPL v3 bank
#   MIDI + WOPL --adlmidiplay(libADLMIDI,docker)--> WAV --ffmpeg--> OGG
#
# 前置:
#   1. fd2-adlmidi Docker image(libADLMIDI + adlmidiplay CLI,WAVE_ONLY 模式)。
#      不存在時本腳本自動 build,Dockerfile 如下(debian:12 + cmake 編 libADLMIDI):
#        FROM debian:12
#        RUN apt-get update && apt-get install -y --no-install-recommends \
#            git cmake build-essential ca-certificates libasound2-dev \
#            && rm -rf /var/lib/apt/lists/*
#        RUN git clone --depth 1 https://github.com/Wohlstand/libADLMIDI.git /src
#        WORKDIR /src/build
#        RUN cmake -DCMAKE_BUILD_TYPE=Release -DWITH_MIDIPLAY=ON \
#                  -DMIDIPLAY_WAVE_ONLY=ON -DlibADLMIDI_STATIC=ON .. \
#            && make -j$(nproc)
#      (WAVE_ONLY 模式下 adlmidiplay 不吃 -w 旗標,直接固定輸出 <midi路徑>.wav)
#   2. 已解包的 FDMUS(unpack_dat.py → raw/FDMUS/*.bin)。
#   3. 遊戲自帶 SAMPLE.AD(玩家自備,漢堂版權,不散布)。
#
# 用法:  ./tools/export_fm.sh <raw/FDMUS目錄> <SAMPLE.AD路徑> <輸出目錄>
set -e
SRC="${1:?raw/FDMUS 目錄}"; SAMPLE_AD="${2:?SAMPLE.AD 或 SAMPLE.OPL 路徑}"; OUT="${3:?輸出目錄}"
cd "$(dirname "$0")/.."
TMP="$(mktemp -d)"; mkdir -p "$OUT"

if ! docker image inspect fd2-adlmidi >/dev/null 2>&1; then
  echo "[0/3] fd2-adlmidi image 不存在,build..."
  DFILE="$TMP/Dockerfile.adlmidi"
  cat > "$DFILE" <<'EOF'
FROM debian:12
RUN apt-get update && apt-get install -y --no-install-recommends \
    git cmake build-essential ca-certificates libasound2-dev \
    && rm -rf /var/lib/apt/lists/*
RUN git clone --depth 1 https://github.com/Wohlstand/libADLMIDI.git /src
WORKDIR /src/build
RUN cmake -DCMAKE_BUILD_TYPE=Release -DWITH_MIDIPLAY=ON -DMIDIPLAY_WAVE_ONLY=ON -DlibADLMIDI_STATIC=ON .. \
    && make -j$(nproc)
EOF
  docker build -t fd2-adlmidi -f "$DFILE" "$TMP"
fi

echo "[1/3] SAMPLE.AD(Miles AIL2 GTL)→ WOPL v3 bank…"
python3 tools/gtl2wopl.py "$SAMPLE_AD" "$TMP/fd2.wopl"

echo "[2/3] XMI → MIDI…"
for f in "$SRC"/*.bin; do
  base=$(basename "$f" .bin)
  python3 tools/xmi2mid.py "$f" "$TMP/$base.mid" >/dev/null 2>&1 || true
done

echo "[3/3] adlmidiplay(libADLMIDI,遊戲自帶音色)render → WAV → OGG…"
n=0
for mid in "$TMP"/*.mid; do
  [ -f "$mid" ] || continue
  base=$(basename "$mid" .mid)
  docker run --rm -v "$TMP":/m fd2-adlmidi \
        /src/build/adlmidiplay "/m/$base.mid" "/m/fd2.wopl" >/dev/null 2>&1 \
    && ffmpeg -y -i "$TMP/$base.mid.wav" -c:a libvorbis -q:a 6 "$OUT/$base.ogg" -loglevel error \
    && { n=$((n+1)); } || echo "  $base render 失敗"
done
rm -rf "$TMP"
echo "完成:$n 首 FM(Sound Blaster/AdLib)OGG → $OUT/"
echo "(換 OPL3 版:SAMPLE_AD 改傳 SAMPLE.OPL——本作 .AD/.OPL 位元組相同,無 4-op 樂器,結果應一致)"
