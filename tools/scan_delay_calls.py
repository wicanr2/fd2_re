#!/usr/bin/env python3
"""列出 FD2.EXE 內所有 `delay(ms)` 呼叫點、毫秒參數與所屬函式。

原版的動畫節奏有兩個來源：`sub_17AA9(n)` 等 n 個 BIOS tick（54.925 毫秒一個），
以及 Watcom 的 `delay(ms)`（`0x375B2` thunk → `0x3DCCD`）。後者把毫秒乘上開機
校準值 `[0x541B0]`，再用那麼多次 `int 21h AH=2Ch` 消耗時間，所以**毫秒數是寫死
在呼叫點的設計意圖**——比從模擬器量到的指令數可靠。回合橫幅的中段停留就是這樣
定出來的（見 docs/knowledge-base/105-phase-banner-timing-20260910.md）。

用法（容器內）：
  scan_delay_calls.py <FD2.EXE> <docs/data/ida/fd2_function_inventory.json>
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
# 0x375B2 是 `jmp 0x3DCCD` 的 thunk；兩個都算呼叫點。
DELAY_ENTRIES = {0x375B2, 0x3DCCD}


def owner_index(inventory_path):
    functions = json.load(open(inventory_path, encoding='utf-8'))['functions']
    return sorted((int(f['start'], 16), int(f['end'], 16), f['ida_analysis_name'])
                  for f in functions)


def owner_of(index, address):
    low, high = 0, len(index) - 1
    while low <= high:
        mid = (low + high) // 2
        start, end, name = index[mid]
        if address < start:
            high = mid - 1
        elif address >= end:
            low = mid + 1
        else:
            return name
    return '?'


def main(argv):
    if len(argv) != 3:
        sys.exit(__doc__)
    exe, inventory = argv[1], argv[2]
    data = open(exe, 'rb').read()
    meta = parse_le(data)
    obj = meta['objs'][0]
    start = meta['data_off'] + (obj['base'] - CODE_BASE)
    code = data[start:start + obj['vsize']]
    index = owner_index(inventory)
    md = Cs(CS_ARCH_X86, CS_MODE_32)
    md.detail = True
    instructions = list(md.disasm(code, obj['base']))
    print(f"# {exe} 的 delay(ms) 呼叫點（0x375B2／0x3DCCD）")
    for position, instruction in enumerate(instructions):
        if instruction.mnemonic != 'call' or not instruction.operands:
            continue
        operand = instruction.operands[0]
        if operand.type != 2 or (operand.imm & 0xFFFFFFFF) not in DELAY_ENTRIES:
            continue
        millis = None
        for back in range(position - 1, max(0, position - 4), -1):
            previous = instructions[back]
            if previous.mnemonic == 'push' and previous.operands and \
                    previous.operands[0].type == 2:
                millis = previous.operands[0].imm
                break
            if previous.mnemonic in ('call', 'jmp', 'ret'):
                break
        shown = millis if millis is not None else '?（參數不是就近的 push imm）'
        print(f"0x{instruction.address:06x}  delay({shown})  "
              f"{owner_of(index, instruction.address)}")
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv))
