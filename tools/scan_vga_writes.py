#!/usr/bin/env python3
"""列出 FD2.EXE 內所有指向 VGA 線性位址（0xA0000..0xB0000）的立即數。

回答「這一段畫面是誰寫的」時，`-eip-watch` 需要候選位址，而候選要有來源。原版
的繪圖呼叫幾乎都把目的地當立即數傳進去（例如地圖窗格是 `0xA0504`＝(4,4)），
所以掃立即數就能把候選收斂到幾十個函式。回合橫幅就是這樣找到 `sub_1F1CC`
家族的（見 docs/knowledge-base/105-phase-banner-timing-20260910.md）。

注意它掃不到「從全域變數取得 VGA 基底」的寫入端；那類要另外追指標。

用法（容器內）：
  scan_vga_writes.py <FD2.EXE> <docs/data/ida/fd2_function_inventory.json>
"""
import json
import sys

sys.path.insert(0, __file__.rsplit('/', 1)[0])
from le_xref import parse_le

try:
    from capstone import Cs, CS_ARCH_X86, CS_MODE_32
except ImportError:
    sys.exit("need capstone: 只在 fd2-cap-local 容器內執行")

CODE_BASE = 0x10000
VGA_BASE, VGA_END = 0xA0000, 0xB0000


def main(argv):
    if len(argv) != 3:
        sys.exit(__doc__)
    exe, inventory = argv[1], argv[2]
    data = open(exe, 'rb').read()
    meta = parse_le(data)
    obj = meta['objs'][0]
    start = meta['data_off'] + (obj['base'] - CODE_BASE)
    code = data[start:start + obj['vsize']]
    functions = json.load(open(inventory, encoding='utf-8'))['functions']
    index = sorted((int(f['start'], 16), int(f['end'], 16), f['ida_analysis_name'])
                   for f in functions)

    def owner_of(address):
        low, high = 0, len(index) - 1
        while low <= high:
            mid = (low + high) // 2
            first, last, name = index[mid]
            if address < first:
                high = mid - 1
            elif address >= last:
                low = mid + 1
            else:
                return name, first
        return '?', None

    md = Cs(CS_ARCH_X86, CS_MODE_32)
    md.detail = True
    hits = 0
    print(f"# {exe} 的 VGA 立即數（0xA0000..0xB0000）")
    for instruction in md.disasm(code, obj['base']):
        for operand in instruction.operands:
            if operand.type != 2:
                continue
            value = operand.imm & 0xFFFFFFFF
            if not VGA_BASE <= value < VGA_END:
                continue
            offset = value - VGA_BASE
            y, x = divmod(offset, 320)
            name, first = owner_of(instruction.address)
            text = f"{instruction.mnemonic} {instruction.op_str}"
            print(f"0x{instruction.address:06x}  {text:<34} → 0xA0000+0x{offset:05x} "
                  f"(x={x:3d},y={y:3d})  {name}@{hex(first) if first else '?'}")
            hits += 1
            break
    print(f"# 共 {hits} 筆")
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv))
