#!/usr/bin/env python3
"""炎龍騎士團2 (Flame Dragon Knight 2) SFX 導出工具。

FDOTHER.DAT 資源 #31 是巢狀 `LLLLLL` 容器(見 docs/knowledge-base/36-sfx-audio-data.md),
內含 14 個子樣本,格式為 8-bit unsigned mono raw PCM(無檔頭)。本工具解開巢狀容器,
逐個子樣本補上標準 44-byte RIFF/WAV 檔頭,輸出到 remake/assets/sfx/。

取樣率:反組譯未找到 AIL_set_sample_type/set_sample_playback_rate 立即數呼叫點
(見 docs/knowledge-base/36 待辦),沿用文件既有推定值 11025Hz ── 1995 年 AIL 遊戲常見預設值。

用法:
    python3 tools/export_sfx.py
"""
import os
import struct
import sys
import wave

sys.path.insert(0, os.path.dirname(__file__))
from unpack_dat import parse_directory, NotAContainer

SRC = os.path.join(os.path.dirname(__file__), "..", "extracted", "FDOTHER", "FDOTHER_031.bin")
OUT_DIR = os.path.join(os.path.dirname(__file__), "..", "remake", "assets", "sfx")
SAMPLE_RATE = 11025  # 推定值,見 docs/knowledge-base/36-sfx-audio-data.md 待辦


def main():
    data = open(SRC, "rb").read()
    try:
        entries = parse_directory(data)
    except NotAContainer as e:
        print(f"錯誤: {SRC} 不是合法的 LLLLLL 容器: {e}")
        return 1

    os.makedirs(OUT_DIR, exist_ok=True)
    print(f"{os.path.basename(SRC)}: {len(data)} bytes, {len(entries)} 個子樣本\n")

    written = []
    for i, (off, ln) in enumerate(entries):
        pcm = data[off:off + ln]
        if len(pcm) == 0:
            print(f"  sfx_{i:02d}: 0 bytes(目錄結尾哨兵),略過")
            continue
        out_path = os.path.join(OUT_DIR, f"sfx_{i:02d}.wav")
        with wave.open(out_path, "wb") as w:
            w.setnchannels(1)
            w.setsampwidth(1)  # 8-bit unsigned
            w.setframerate(SAMPLE_RATE)
            w.writeframes(pcm)
        dur_ms = len(pcm) / SAMPLE_RATE * 1000
        print(f"  sfx_{i:02d}: {len(pcm):6d} bytes -> {dur_ms:7.1f} ms  ({out_path})")
        written.append(out_path)

    print(f"\n完成: {len(written)} 個 WAV 已寫入 {OUT_DIR}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
