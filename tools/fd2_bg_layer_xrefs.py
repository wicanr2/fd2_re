#!/usr/bin/env python3
"""列出 FD2.EXE 中參照物理／演出背景全域 dword_54103..54113 的 LE fixup（#321）。

每筆印出 IDA LE 線性位址、fixup 前 3 byte 與後 4 byte，用來辨識 mov [mem],imm（C7 05）、
mov [mem],eax（A3）、mov [mem],ebp（89 2D）、push [mem]（FF 35）與 push [reg*4+mem]（FF 34 95／AD）。
另印出三組 BG 載入序列的原始 bytes，以及其中 push imm32 指向的字串。

用法（Docker 內）：python3 -I tools/fd2_bg_layer_xrefs.py <FD2.EXE>
"""
import hashlib
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import le_xref as L  # noqa: E402

NAMES = {0x54103: "dword_54103", 0x54107: "dword_54107", 0x5410B: "dword_5410B",
         0x5410F: "dword_5410F", 0x54113: "dword_54113"}
LOAD_RANGES = [(0x278C0, 0x27950), (0x28D60, 0x28E00), (0x2A2E0, 0x2A360)]


def main(path):
    d = open(path, "rb").read()
    print("size", len(d), "sha256", hashlib.sha256(d).hexdigest())
    meta = L.parse_le(d)
    fx = L.parse_fixups(d, meta)
    fxlin = {L.file_to_linear(meta, s): (s, t) for s, t in fx.items()}

    def lin_to_file(lin):
        for pg in range(meta["page_cnt"]):
            base = L.page_linear(meta, pg)
            if base is not None and base <= lin < base + meta["page_size"]:
                return L.page_file(meta, pg) + lin - base
        return None

    for lin in sorted(fxlin):
        src, tgt = fxlin[lin]
        if tgt in NAMES:
            print(f"{NAMES[tgt]} fixup_linear={lin:#x} bytes={d[src - 3:src].hex()}|{d[src:src + 4].hex()}")
    for start, end in LOAD_RANGES:
        f = lin_to_file(start)
        print(f"== {start:#x}..{end:#x} {d[f:f + end - start].hex()}")
        for lin in range(start, end):
            if lin in fxlin and 0x50000 <= fxlin[lin][1] < 0x60000 and fxlin[lin][1] not in NAMES:
                t = fxlin[lin][1]
                tf = lin_to_file(t)
                text = d[tf:d.index(b"\0", tf)].decode("latin1")
                print(f"  push_imm32 at {lin - 1:#x} -> {t:#x} {text!r}")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(__doc__)
        sys.exit(1)
    main(sys.argv[1])
