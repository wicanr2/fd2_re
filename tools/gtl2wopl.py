#!/usr/bin/env python3
"""炎龍騎士團2 — Miles AIL2 Global Timbre Library(.AD/.OPL)→ WOPL v3 bank 轉換器。

FD2 原生音效卡驅動預設是 **Sound Blaster**(見 `MDI.INI`:DRIVER=SBLASTER.MDI,IO 220h),
SBLASTER.MDI/ADLIB.MDI 走 OPL2、OPL3.MDI 走 OPL3;全域音色庫是 `SAMPLE.AD`(OPL2)/
`SAMPLE.OPL`(OPL3,本作與 .AD 位元組相同,無 4-op 樂器)。本工具把該音色庫轉成
libADLMIDI(adlmidiplay/ADLMIDI)可讀的 WOPL 格式,讓 XMI 音樂能用**遊戲自帶音色**
渲染成 FM(AdLib/Sound Blaster)版 BGM,而非套用通用 GM 音色包(會失真)。

格式依據(逐位元組核實,非猜測):
  - AIL GTL 來源格式(.AD/.OPL):
    OPL3BankEditor Specifications/AILBANK.TXT
    https://github.com/Wohlstand/OPL3BankEditor/blob/master/Specifications/AILBANK.TXT
  - WOPL v3 目的格式:libADLMIDI src/wopl/wopl_file.c 的
    WOPL_parseInstrument / WOPL_writeInstrument(實際消費端,非規格書猜測)
    https://github.com/Wohlstand/libADLMIDI/blob/master/src/wopl/wopl_file.c
  - Operator 順序(WOPL operators[0]=Carrier1 / [1]=Modulator1)對照
    OPL3BankEditor src/bank.h 的 #define CARRIER1 0 / MODULATOR1 1,
    與 src/FileFormats/format_ail2_gtl.cpp 的
    ins.setAVEKM(MODULATOR1, idata[1]) / ins.setAVEKM(CARRIER1, idata[7]) 交叉驗證
    (AIL op0=register 0x20 群組=傳統 OPL Modulator;AIL op1=register 0x23 群組=Carrier)。

AIL GTL 2-op 樂器 body(去掉 size+transpose 後,11 bytes):
  [0]op0_misc(0x20) [1]op0_scale(0x40) [2]op0_AD(0x60) [3]op0_SR(0x80) [4]op0_WF(0xE0)
  [5]feedbackConnection(0xC0)
  [6]op1_misc(0x23) [7]op1_scale(0x43) [8]op1_AD(0x63) [9]op1_SR(0x83) [10]op1_WF(0xE3)

FD2 的 SAMPLE.AD 全為 2-op(162 筆:bank0 melodic 128 筆全滿 + bank127 percussion 34 筆),
4-op 未在本作出現過,故本工具只實作 2-op 轉換(遇到 4-op 樂器會丟例外,不悄悄跳過)。

用法:
    python3 gtl2wopl.py <SAMPLE.AD 或 .OPL> <out.wopl>
"""
import struct
import sys

VOLUME_AIL = 7          # OPL3BankEditor src/bank.h enum VolumesScale::VOLUME_AIL
WOPL_INS_FIXED_NOTE = 0x40
WOPL_INS_IS_BLANK = 0x04


def parse_gtl(path):
    """解析 GTL 索引(programNumber, bankNumber, fileOffset)三元組,直到 0xFF/0xFF 結束。"""
    with open(path, 'rb') as f:
        data = f.read()
    heads = []
    off = 0
    while off + 6 <= len(data):
        patch, bank = data[off], data[off + 1]
        foff = struct.unpack_from('<I', data, off + 2)[0]
        off += 6
        if patch == 0xFF and bank == 0xFF:
            break
        heads.append((patch, bank, foff))

    insts = {}
    for patch, bank, foff in heads:
        size = struct.unpack_from('<H', data, foff)[0]
        if size != 0x0E:
            raise NotImplementedError(
                f"4-op 樂器(size=0x{size:02x}, patch={patch} bank={bank})不在 FD2 SAMPLE.AD "
                "中出現過,未實作轉換 —— 別悄悄跳過,先確認這是不是新素材再擴充")
        body = data[foff + 2:foff + size]
        transpose = body[0]
        insts[(bank, patch)] = dict(transpose=transpose, raw=body[1:])  # raw: 11 bytes
    return insts


def _blank_instrument():
    d = bytearray(66)
    d[39] = WOPL_INS_IS_BLANK
    return bytes(d)


def _build_instrument(entry, is_perc):
    b = entry['raw']
    transpose = entry['transpose']
    d = bytearray(66)
    if is_perc:
        # AILBANK.TXT:bank 127 的 transpose 語意是「絕對音高」(打擊樂器固定音高)
        # -> 對應 WOPL 的 percussion_key_number + Fixed-note flag。
        d[38] = transpose & 0xFF
        d[39] = WOPL_INS_FIXED_NOTE
    else:
        # 旋律樂器的 transpose 是相對音高位移 -> note_offset1(signed BE16)。
        note_offset1 = transpose if transpose < 128 else transpose - 256
        struct.pack_into('>h', d, 32, note_offset1)
    d[40] = b[5]   # fb_conn1_C0 <- AIL feedbackConnection(register 0xC0)
    d[41] = 0      # fb_conn2_C0:2-op 無第二聲部
    # operators[0]=Carrier1  <- AIL op1(register 0x23 群組, raw[6:11])
    d[42 + 0 * 5:42 + 0 * 5 + 5] = bytes(b[6:11])
    # operators[1]=Modulator1 <- AIL op0(register 0x20 群組, raw[0:5])
    d[42 + 1 * 5:42 + 1 * 5 + 5] = bytes(b[0:5])
    # operators[2]/[3](Carrier2/Modulator2):2-op 樂器不使用,留 0
    # delay_on_ms/delay_off_ms 留 0,交給 player 自動判斷
    return bytes(d)


def build_wopl(ad_path, out_path, bank_label="FD2 SAMPLE.AD (SB/AdLib OPL2)"):
    insts = parse_gtl(ad_path)

    melodic = [_blank_instrument() for _ in range(128)]
    percussion = [_blank_instrument() for _ in range(128)]
    n_mel = n_perc = 0
    for (bank, patch), entry in insts.items():
        if bank == 127:
            if 0 <= patch < 128:
                percussion[patch] = _build_instrument(entry, is_perc=True)
                n_perc += 1
        elif bank == 0:
            if 0 <= patch < 128:
                melodic[patch] = _build_instrument(entry, is_perc=False)
                n_mel += 1
        else:
            print(f"[警告] 忽略非 bank0/127 的樂器 bank={bank} patch={patch}"
                  "(FD2 未見過此 bank,若真出現需擴充多 bank 支援)", file=sys.stderr)

    out = bytearray()
    out += b"WOPL3-BANK\x00"
    out += struct.pack('<H', 3)        # version = 3
    out += struct.pack('>H', 1)        # melodic bank 數
    out += struct.pack('>H', 1)        # percussion bank 數
    out += bytes([0x03])               # opl_flags:deep tremolo + deep vibrato(AIL 音色庫慣例)
    out += bytes([VOLUME_AIL])         # volume_model = AIL

    def bank_meta(name):
        nb = name.encode('utf-8')[:31]
        return nb + b'\x00' * (32 - len(nb)) + bytes([0, 0])  # lsb=0, msb=0

    out += bank_meta(bank_label)
    out += bank_meta(bank_label + " -Percussion")
    for ins in melodic:
        out += ins
    for ins in percussion:
        out += ins

    with open(out_path, 'wb') as f:
        f.write(out)

    print(f"寫出 {out_path}:{len(out)} bytes,melodic 樂器 {n_mel}/128,percussion 樂器 {n_perc}/128")
    return n_mel, n_perc


if __name__ == '__main__':
    if len(sys.argv) < 3:
        print(__doc__)
        sys.exit(1)
    build_wopl(sys.argv[1], sys.argv[2])
