#!/usr/bin/env python3
"""炎龍騎士團2 — turn_events.event_id → FDFIELD group 對應機制擷取。

`0x51b91`(58 entry,event_id 0-57 全域)是**真正消費 turn_events.event_id 的跳表**
(不是 doc25 §6 早先猜測的 `0x22e5c`——那支只是章1專屬的單次過場演出,
call graph 顯示唯一 caller 是 0x25de5,固定寫死,不讀 FDFIELD)。

真實鏈路(0x1a813,3 處呼叫點:camp=1/0/2 分別在 ally/enemy/special 回合結束後):
  迴圈 FDFIELD 控制段 turn_events[16](base=[0x53a55],3B/筆:turn,event_id,camp)
  → turn==[0x53bef](回合數) && camp==filter → call [event_id*4 + 0x51b91]
  → 該 event_id 專屬 handler(硬編碼 C 函式,非資料驅動)
  → handler 內 `push <group_id>; call 0x10b4e`(spawn_group,依 FDFIELD b21 啟用該
    group 的單位)或 `call 0x32999`(spawn_group_with_intro,內部一樣呼叫 0x10b4e,
    多繪 portrait+對話)。group_id 通常是 handler 裡的字面常數,少數(如 event27/54/57)
    是動態值 `[0x53bef]`(=用當下回合數當 group 編號,對應同一 event_id 逐回合重觸發)。

本工具:從每個 event_id handler(0x51b91 jump table)擷取其自身函式體內的
`push <imm>; call 0x10b4e` / `push <imm>; call 0x32999`(spawn_group 呼叫),
只走 handler 自己的 basic-block 鏈(call 視為過站不進入被呼叫者本體,
遇 ret 停止),避免線性 sweep 漂移進共用子函式或下一個 handler。

用法(docker fd2-cap):
    python3 tools/extract_event_id_groups.py [out.json]
"""
import sys, struct, json
sys.path.insert(0, "/work/tools")
from le_xref import parse_le
from capstone import Cs, CS_ARCH_X86, CS_MODE_32

EXE = "org_game/炎龍騎士團/FLAME2/FD2.EXE"
CODE_BASE = 0x10000


def load_code(d, meta):
    o = meta['objs'][0]
    start = meta['data_off'] + (o['base'] - CODE_BASE)
    return d[start:start + o['vsize']], o['base'], o['vsize']


def page_base_linear(meta, pg):
    psize = meta['page_size']
    for o in meta['objs']:
        first = o['first'] - 1
        if first <= pg < first + o['pages']:
            return o['base'] + (pg - first) * psize
    return None


def fixup_map(d, meta):
    fixpage, fixrec = meta['fixpage'], meta['fixrec']
    psize = meta['page_size']
    npages = sum(o['pages'] for o in meta['objs'])
    fx = {}
    for pg in range(npages):
        o0 = struct.unpack_from('<I', d, fixpage + pg * 4)[0]
        o1 = struct.unpack_from('<I', d, fixpage + (pg + 1) * 4)[0]
        p, end = fixrec + o0, fixrec + o1
        page_lin = page_base_linear(meta, pg)
        if page_lin is None:
            continue
        while p < end:
            st, fl = d[p], d[p + 1]; p += 2
            srcoff = struct.unpack_from('<h', d, p)[0]; p += 2
            objn = d[p]; p += 1
            if fl & 0x10:
                trg = struct.unpack_from('<I', d, p)[0]; p += 4
            else:
                trg = struct.unpack_from('<H', d, p)[0]; p += 2
            base = meta['objs'][objn - 1]['base'] if 1 <= objn <= len(meta['objs']) else 0
            fx[page_lin + srcoff] = base + trg
    return fx


d = open(EXE, 'rb').read()
meta = parse_le(d)
code, base, vsize = load_code(d, meta)
md = Cs(CS_ARCH_X86, CS_MODE_32)
end = base + vsize


def insn_at(addr):
    off = addr - base
    if off < 0 or off >= len(code):
        return None
    for ins in md.disasm(code[off:off + 15], addr):
        return ins
    return None


SPAWN_FNS = {0x10b4e: 'spawn_group', 0x32999: 'spawn_group_with_intro'}


def walk_handler(start, max_insns=4000):
    """只走這個 handler 自己的鏈:call 過站不進入,遇 ret 停止,
    條件跳兩路都走(BFS),無條件跳跟隨,限制在 [start, start+0x1000) 內
    (handler 本體不該超出這範圍;超出視為離開本函式,不繼續)。"""
    spawns = []
    visited = set()
    stack = [start]
    n = 0
    while stack and n < max_insns:
        a = stack.pop()
        pending_push = None
        while a is not None and a not in visited:
            if not (start <= a < start + 0x1000):
                break
            ins = insn_at(a)
            if ins is None:
                break
            visited.add(a)
            n += 1
            m, op = ins.mnemonic, ins.op_str
            nxt = a + ins.size
            if m == 'push':
                if op.startswith('0x') or op.lstrip('-').isdigit():
                    try:
                        pending_push = int(op, 0)
                    except ValueError:
                        pending_push = None
                elif '0x3bef' in op:
                    pending_push = '$turn_counter[0x53bef]'
                elif '0x3ae1' in op:
                    pending_push = '$0x53ae1'
                else:
                    pending_push = f'$reg_or_mem({op})'
            elif m == 'call':
                if op.startswith('0x'):
                    t = int(op, 16)
                    if t in SPAWN_FNS and pending_push is not None:
                        spawns.append({'group': pending_push, 'via': SPAWN_FNS[t]})
                pending_push = None
                a = nxt
                continue
            elif m == 'jmp':
                if op.startswith('0x'):
                    t = int(op, 16)
                    a = t
                    continue
                a = None
                break
            elif m.startswith('j'):
                if op.startswith('0x'):
                    t = int(op, 16)
                    if t not in visited:
                        stack.append(t)
                a = nxt
                pending_push = None
                continue
            elif m in ('ret', 'retn'):
                a = None
                break
            else:
                pending_push = None
            a = nxt
    return spawns


def jtab(tab, count):
    fx = fixup_map(d, meta)
    out = []
    for i in range(count):
        site = tab + i * 4
        t = fx.get(site)
        out.append(t)
    return out


if __name__ == '__main__':
    handlers = jtab(0x51b91, 58)
    results = {}
    for eid, h in enumerate(handlers):
        spawns = walk_handler(h)
        results[eid] = {'handler': hex(h), 'spawns': spawns}
    out = sys.argv[1] if len(sys.argv) > 1 else 'docs/data/event_id_groups.json'
    json.dump(results, open(out, 'w'), indent=1, ensure_ascii=False)
    print(json.dumps(results, indent=1))
