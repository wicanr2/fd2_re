#!/usr/bin/env python3
"""炎龍騎士團2 — DOS4GW LE(32-bit flat)反組譯器(capstone)。

LE obj1 = code,base=0x10000(linear)。本工具把 linear 位址範圍反組譯成
帶位址的 x86-32 組語,並標出每條指令觸及的 fixup target(資料/字串/呼叫的絕對位址),
方便做控制流追蹤與 sink→caller 反向溯源(規則 62)。

linear ↔ file:obj1 first_page=1,page_size=0x1000,data_off=0x10e00。
  file = data_off + (linear - 0x10000)          # obj1 連續映射(page 1..63)

用法:
  python3 disasm_le.py <FD2.EXE> dis <linear_hex> [count]      反組譯 count 條指令
  python3 disasm_le.py <FD2.EXE> range <start_hex> <end_hex>   反組譯 linear 範圍
  python3 disasm_le.py <FD2.EXE> calls <target_hex>            找對 target 的相對 call/jmp 來源(linear)
  python3 disasm_le.py <FD2.EXE> refs <abs_hex>                找 code 中被 fixup 成 abs 的位置(資料 xref)
"""
import sys, struct
sys.path.insert(0, __file__.rsplit('/', 1)[0])
from le_xref import parse_le

try:
    from capstone import Cs, CS_ARCH_X86, CS_MODE_32
except ImportError:
    sys.exit("need capstone: run under docker uv (見 README)")

CODE_BASE = 0x10000


def lin2file(meta, lin):
    # obj1 連續:page 1 起,data_off 是 page0?實測 obj1 first_page=1 → file=data_off+(lin-base)
    return meta['data_off'] + (lin - CODE_BASE)


def load_code(d, meta):
    o = meta['objs'][0]
    start = lin2file(meta, o['base'])
    return d[start:start + o['vsize']], o['base']


def build_fixups(d, meta):
    """回傳 {code_linear: target_abs} —— 每個被 patch 的位置→其絕對 target。"""
    page_size = meta['page_size']
    fixpage = meta['fixpage']; fixrec = meta['fixrec']
    npages = meta['objs'][0]['pages']
    fx = {}
    for pg in range(npages):
        off0 = struct.unpack_from('<I', d, fixpage + pg * 4)[0]
        off1 = struct.unpack_from('<I', d, fixpage + (pg + 1) * 4)[0]
        p = fixrec + off0; end = fixrec + off1
        page_lin = CODE_BASE + pg * page_size
        while p < end:
            src_type = d[p]; flags = d[p + 1]; p += 2
            srcoff = struct.unpack_from('<h', d, p)[0]; p += 2
            # target
            if flags & 0x40:
                objn = d[p]; p += 1
            else:
                objn = d[p]; p += 1
            if flags & 0x10:
                trgoff = struct.unpack_from('<I', d, p)[0]; p += 4
            else:
                trgoff = struct.unpack_from('<H', d, p)[0]; p += 2
            base = meta['objs'][objn - 1]['base'] if 1 <= objn <= len(meta['objs']) else 0
            tgt = base + trgoff
            lin = page_lin + srcoff
            if (src_type & 0x0f) in (7, 5, 0):  # 32-bit offset / 16-bit etc
                fx[lin] = tgt
    return fx


def main(a):
    if len(a) < 3:
        print(__doc__); return 1
    d = open(a[1], 'rb').read()
    meta = parse_le(d)
    code, base = load_code(d, meta)
    md = Cs(CS_ARCH_X86, CS_MODE_32)
    md.detail = False
    cmd = a[2]

    if cmd in ('dis', 'range'):
        if cmd == 'dis':
            start = int(a[3], 16); cnt = int(a[4]) if len(a) > 4 else 40
            end = start + 0x400
        else:
            start = int(a[3], 16); end = int(a[4], 16); cnt = 10**9
        fx = build_fixups(d, meta)
        off = start - base
        n = 0
        for insn in md.disasm(code[off:off + (end - start)], start):
            tgt = ''
            for o2 in range(insn.address, insn.address + insn.size):
                if o2 in fx:
                    tgt = f'  ; ->{hex(fx[o2])}'
                    break
            print(f'{insn.address:#08x}  {insn.mnemonic:<7} {insn.op_str}{tgt}')
            n += 1
            if n >= cnt:
                break
        return 0

    if cmd == 'calls':
        tgt = int(a[3], 16)
        for insn in md.disasm(code, base):
            if insn.mnemonic in ('call', 'jmp') and insn.op_str.startswith('0x'):
                try:
                    if int(insn.op_str, 16) == tgt:
                        print(f'{insn.address:#08x}  {insn.mnemonic} {insn.op_str}')
                except ValueError:
                    pass
        return 0

    if cmd == 'refs':
        abs_ = int(a[3], 16)
        fx = build_fixups(d, meta)
        for lin, t in sorted(fx.items()):
            if t == abs_:
                print(f'{lin:#08x} -> {hex(t)}')
        return 0

    print(__doc__); return 1


if __name__ == '__main__':
    sys.exit(main(sys.argv))
