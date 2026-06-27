#!/usr/bin/env python3
"""炎龍騎士團2 — LE(Linear Executable)解析 + 重定位 xref 工具。

FD2.EXE 是 DOS4GW LE。資料(字串/表)的絕對位址在檔內走 **fixup(重定位)**,
raw bytes 不是最終 linear 位址 → 不能直接搜字串 xref。本工具:
  1. 解析 LE object table(file ↔ linear 對映)。
  2. 解析 fixup 表,建立「code 中哪個位址被 patch 成哪個 target linear」。
  3. 由此找「某字串 / 資料位址」被 code 參照的位置(xref)。

第 3 輪用它解出:play_bgm 場景→曲號、FDMUS 載入點等(見 docs/12,13)。

用法:
    python3 le_xref.py <FD2.EXE> str <字串>      # 找字串被 code 參照處(file offset)
    python3 le_xref.py <FD2.EXE> calls <hexaddr>  # 找某函式(file offset)的相對呼叫端
"""
import sys
import struct
import re


def parse_le(d):
    le = d.find(b"LE\x00\x00", 0x2000)
    g = lambda o: struct.unpack_from("<I", d, le + o)[0]
    page_size = g(0x28)
    data_off = g(0x80)
    obj_cnt = g(0x44)
    obj_tab = le + g(0x40)
    page_cnt = g(0x14)
    objs = []
    for i in range(obj_cnt):
        o = obj_tab + i * 24
        vsize, base, flags, pmidx, pcnt = struct.unpack_from("<IIIII", d, o)[:5]
        objs.append({"base": base, "vsize": vsize, "first": pmidx, "pages": pcnt})
    return {"le": le, "page_size": page_size, "data_off": data_off,
            "objs": objs, "page_cnt": page_cnt,
            "fixpage": le + g(0x68), "fixrec": le + g(0x6c)}


def page_file(meta, pg):
    return meta["data_off"] + pg * meta["page_size"]


def page_linear(meta, pg):
    # page pg(0-based) 屬於哪個 object
    acc = 0
    for ob in meta["objs"]:
        if pg < acc + ob["pages"]:
            return ob["base"] + (pg - acc) * meta["page_size"]
        acc += ob["pages"]
    return None


def file_to_linear(meta, f):
    for ob in meta["objs"]:
        fstart = page_file(meta, ob["first"] - 1)
        fend = fstart + ob["pages"] * meta["page_size"]
        if fstart <= f < fend:
            return ob["base"] + (f - fstart)
    return None


def parse_fixups(d, meta):
    """回傳 dict: src_file_offset -> target_linear(僅 internal 1..N obj)。"""
    n = meta["page_cnt"]
    fpt = [struct.unpack_from("<I", d, meta["fixpage"] + 4 * i)[0] for i in range(n + 1)]
    out = {}
    for pg in range(n):
        i = meta["fixrec"] + fpt[pg]
        end = meta["fixrec"] + fpt[pg + 1]
        fbase = page_file(meta, pg)
        while i < end:
            src = d[i]; flags = d[i + 1]; i += 2
            srcoff = struct.unpack_from("<h", d, i)[0]; i += 2
            if flags & 0x40:
                obj = struct.unpack_from("<H", d, i)[0]; i += 2
            else:
                obj = d[i]; i += 1
            if flags & 0x10:
                toff = struct.unpack_from("<I", d, i)[0]; i += 4
            else:
                toff = struct.unpack_from("<H", d, i)[0]; i += 2
            if src & 0x20:  # source list 少見,略過該頁其餘
                break
            if 1 <= obj <= len(meta["objs"]):
                out[fbase + srcoff] = meta["objs"][obj - 1]["base"] + toff
    return out


def find_str_xref(d, meta, s):
    fo = d.find(s.encode() if isinstance(s, str) else s)
    if fo < 0:
        return None, []
    lin = file_to_linear(meta, fo)
    fixups = parse_fixups(d, meta)
    srcs = [k for k, v in fixups.items() if v == lin]
    return lin, sorted(srcs)


def find_callers(d, target):
    out = []
    for m in re.finditer(b"\xe8", d[0x10000:0x4ec00]):
        p = 0x10000 + m.start()
        rel = struct.unpack_from("<i", d, p + 1)[0]
        if p + 5 + rel == target:
            out.append(p)
    return out


def main(argv):
    if len(argv) < 3:
        print(__doc__); return 1
    d = open(argv[1], "rb").read()
    meta = parse_le(d)
    if argv[2] == "str":
        lin, srcs = find_str_xref(d, meta, argv[3])
        print(f"'{argv[3]}' linear=0x{lin:x};被參照 {len(srcs)} 處(file):")
        for s in srcs:
            print(f"  0x{s:x}")
    elif argv[2] == "calls":
        t = int(argv[3], 16)
        cs = find_callers(d, t)
        print(f"0x{t:x} 相對呼叫端 {len(cs)} 處:")
        for c in cs:
            print(f"  0x{c:x}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
