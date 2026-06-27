#!/usr/bin/env python3
"""炎龍騎士團2 (Flame Dragon Knight 2) 通用 .DAT 容器解包工具。

第 1 輪逆向工程成果。漢堂國際 1995 年原版的所有 .DAT 資產檔共用同一個
簡單的歸檔(archive)容器格式:

    +0   6 bytes   magic = b"LLLLLL"  (0x4C x6)
    +6   uint32[N] little-endian offset 目錄
                   N = (offsets[0] - 6) // 4
                   每個 offset 指向一個 sub-resource 的起點(相對檔頭)
                   單調遞增;第 i 個資源的長度 = offsets[i+1] - offsets[i]
                   最後一個資源延伸到檔尾。

用法:
    python3 unpack_dat.py <檔案.DAT> [輸出目錄]      # 解包單一檔
    python3 unpack_dat.py --list <檔案.DAT>           # 只列目錄不解包
    python3 unpack_dat.py --all <FLAME2目錄> <輸出根目錄>  # 解包整個目錄

不依賴任何第三方套件,純標準函式庫。
"""
import os
import sys
import struct

MAGIC = b"LLLLLL"


class NotAContainer(Exception):
    pass


def parse_directory(data: bytes):
    """回傳 sub-resource 的 (offset, length) 清單;非本格式則丟 NotAContainer。"""
    if data[:6] != MAGIC:
        raise NotAContainer("缺少 LLLLLL magic")
    first = struct.unpack_from("<I", data, 6)[0]
    if first < 6 or first > len(data) or (first - 6) % 4 != 0:
        raise NotAContainer(f"目錄起點不合理: 0x{first:x}")
    n = (first - 6) // 4
    offs = [struct.unpack_from("<I", data, 6 + 4 * i)[0] for i in range(n)]
    # 健全性檢查:單調遞增且都在範圍內
    for i in range(n - 1):
        if offs[i] > offs[i + 1]:
            raise NotAContainer(f"目錄非單調遞增 @#{i}")
    if offs[-1] > len(data):
        raise NotAContainer("目錄 offset 超出檔尾")
    bounds = offs + [len(data)]
    return [(bounds[i], bounds[i + 1] - bounds[i]) for i in range(n)]


def list_container(path: str):
    data = open(path, "rb").read()
    entries = parse_directory(data)
    print(f"{os.path.basename(path)}: {len(data)} bytes, {len(entries)} 個資源")
    print(f"  {'idx':>4}  {'offset':>10}  {'length':>10}")
    for i, (off, ln) in enumerate(entries):
        print(f"  {i:>4}  0x{off:08x}  {ln:>10}")


def unpack(path: str, out_dir: str):
    data = open(path, "rb").read()
    entries = parse_directory(data)
    base = os.path.splitext(os.path.basename(path))[0]
    dst = os.path.join(out_dir, base)
    os.makedirs(dst, exist_ok=True)
    width = max(3, len(str(len(entries) - 1)))
    for i, (off, ln) in enumerate(entries):
        name = f"{base}_{i:0{width}d}.bin"
        with open(os.path.join(dst, name), "wb") as f:
            f.write(data[off:off + ln])
    print(f"{os.path.basename(path):14} -> {dst}  ({len(entries)} 個資源)")
    return len(entries)


def main(argv):
    if len(argv) < 2:
        print(__doc__)
        return 1
    if argv[1] == "--list":
        list_container(argv[2])
        return 0
    if argv[1] == "--all":
        src, out = argv[2], argv[3]
        total_files = total_res = 0
        skipped = []
        for fn in sorted(os.listdir(src)):
            p = os.path.join(src, fn)
            if not os.path.isfile(p):
                continue
            try:
                total_res += unpack(p, out)
                total_files += 1
            except NotAContainer as e:
                skipped.append((fn, str(e)))
        print(f"\n完成: {total_files} 個容器, 共 {total_res} 個資源 -> {out}")
        if skipped:
            print("略過(非容器格式):")
            for fn, why in skipped:
                print(f"  {fn}: {why}")
        return 0
    path = argv[1]
    out = argv[2] if len(argv) > 2 else "extracted"
    unpack(path, out)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
