#!/usr/bin/env python3
"""炎龍騎士團2 (Flame Dragon Knight 2) SFX 導出工具。

FDOTHER.DAT 資源 #31 是巢狀 `LLLLLL` 容器(見 docs/knowledge-base/36-sfx-audio-data.md),
內含 14 個子樣本,格式為 8-bit unsigned mono raw PCM(無檔頭)。本工具解開巢狀容器,
逐個子樣本補上標準 44-byte RIFF/WAV 檔頭,輸出到 remake/assets/sfx/。

取樣率:反組譯未找到 AIL_set_sample_type/set_sample_playback_rate 立即數呼叫點
(見 docs/knowledge-base/36 待辦),沿用文件既有推定值 11025Hz ── 1995 年 AIL 遊戲常見預設值。

第 10 輪新增 `--battle` 模式:戰鬥音效走另一批動態載入的 FDOTHER.DAT 子資源
(`#48/#49/#50/#51/#52/#53/#64/#78/#88`,見 docs/knowledge-base/36 「戰鬥音效池」節),
用同一個巢狀 LLLLLL 解包邏輯,輸出到 remake/assets/sfx/battle_<資源號>_<子序>.wav。

用法:
    python3 tools/export_sfx.py                # UI 音效池(資源 #31)→ sfx_NN.wav
    python3 tools/export_sfx.py --battle        # 戰鬥音效候選池 → battle_NN_MM.wav
    python3 tools/export_sfx.py --res <idx>      # 導出任意 FDOTHER.DAT 資源號(需為巢狀容器)
"""
import os
import struct
import sys
import wave

sys.path.insert(0, os.path.dirname(__file__))
from unpack_dat import parse_directory, NotAContainer

FDOTHER_DAT = os.path.join(
    os.path.dirname(__file__), "..", "org_game", "炎龍騎士團", "FLAME2", "FDOTHER.DAT")
SRC = os.path.join(os.path.dirname(__file__), "..", "extracted", "FDOTHER", "FDOTHER_031.bin")
OUT_DIR = os.path.join(os.path.dirname(__file__), "..", "remake", "assets", "sfx")
SAMPLE_RATE = 11025  # 推定值,見 docs/knowledge-base/36-sfx-audio-data.md 待辦

# 戰鬥音效候選池:PCM 特徵(值集中 0x80 附近、std 窄)比對確認,見 doc36 第 10 輪。
# 精確「哪個 index 對應哪招」仍是動態值(攻擊資料決定),此處先把整個候選家族導出。
BATTLE_CANDIDATE_INDICES = [48, 49, 50, 51, 52, 53, 64, 78, 88]


def export_container(data: bytes, name_prefix: str, label: str):
    try:
        entries = parse_directory(data)
    except NotAContainer as e:
        print(f"  [{label}] 不是合法的 LLLLLL 容器,略過: {e}")
        return []

    os.makedirs(OUT_DIR, exist_ok=True)
    print(f"[{label}] {len(data)} bytes, {len(entries)} 個子樣本")

    written = []
    for i, (off, ln) in enumerate(entries):
        pcm = data[off:off + ln]
        if len(pcm) == 0:
            print(f"  {name_prefix}{i:02d}: 0 bytes(目錄結尾哨兵),略過")
            continue
        out_path = os.path.join(OUT_DIR, f"{name_prefix}{i:02d}.wav")
        with wave.open(out_path, "wb") as w:
            w.setnchannels(1)
            w.setsampwidth(1)  # 8-bit unsigned
            w.setframerate(SAMPLE_RATE)
            w.writeframes(pcm)
        dur_ms = len(pcm) / SAMPLE_RATE * 1000
        print(f"  {name_prefix}{i:02d}: {len(pcm):6d} bytes -> {dur_ms:7.1f} ms  ({out_path})")
        written.append(out_path)
    return written


def main():
    argv = sys.argv[1:]

    if "--battle" in argv or "--res" in argv:
        dat = open(FDOTHER_DAT, "rb").read()
        try:
            top_entries = parse_directory(dat)
        except NotAContainer as e:
            print(f"錯誤: {FDOTHER_DAT} 不是合法的 LLLLLL 容器: {e}")
            return 1

        if "--res" in argv:
            idx = int(argv[argv.index("--res") + 1])
            indices = [idx]
        else:
            indices = BATTLE_CANDIDATE_INDICES

        written = []
        for idx in indices:
            off, ln = top_entries[idx]
            chunk = dat[off:off + ln]
            written += export_container(chunk, f"battle_{idx:02d}_", f"FDOTHER.DAT #{idx}")

        print(f"\n完成: {len(written)} 個 WAV 已寫入 {OUT_DIR}")
        return 0

    data = open(SRC, "rb").read()
    written = export_container(data, "sfx_", os.path.basename(SRC))
    print(f"\n完成: {len(written)} 個 WAV 已寫入 {OUT_DIR}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
