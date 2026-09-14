#!/usr/bin/env python3
"""炎龍騎士團2 — 全域 event_id handler 與 FDFIELD group 對應擷取。

`0x51b91` 是 90-entry、event_id 0..89 的全域跳表。FDFIELD
turn_events 目前只使用 0..57；58..89 仍可由玩家操作、格子互動、
單位行動與其他 dispatcher 選中，不能因 FDFIELD 分布較窄就截斷跳表。
它也是**真正消費 turn_events.event_id 的跳表**
(不是 doc25 §6 早先猜測的 `0x22e5c`——該函式由 0x25de5 唯一呼叫、
固定載入 FDOTHER #79 且不讀 FDFIELD；章節專屬語意仍未知)。

真實鏈路(0x1a813 的三個 raw camp 1/0/2 呼叫點；其中 camp0 已證實在敵軍 AI 前，
不可把 raw byte 直接改名成 ally/enemy/special 陣營或一概稱為回合結束後):
  迴圈 FDFIELD 控制段 turn_events[16](base=[0x53a55],3B/筆:turn,event_id,camp)
  → turn==[0x53bef](回合數) && camp==filter → call [event_id*4 + 0x51b91]
  → 該 event_id 專屬 handler(硬編碼 C 函式,非資料驅動)
  → handler 內 `push <group_id>; call 0x10b4e`(spawn_group,依 FDFIELD b21 啟用該
    group 的單位)或 `call 0x32999`(spawn_group_with_intro,內部一樣呼叫 0x10b4e,
    另做 FDOTHER #9 的12-pass indexed transition)。全域 event 1/2 在 wrapper
    返回後另呼叫 `0x1366a(3/4)`；acting 不在 `0x32999` 內。
    group_id 通常是 handler 裡的字面常數,少數(如 event27/54/57)
    是動態值 `[0x53bef]`(=用當下回合數當 group 編號,對應同一 event_id 逐回合重觸發)。

本工具:從每個 event_id handler(0x51b91 jump table)擷取其自身函式體內的
`push <imm>; call 0x10b4e` / `push <imm>; call 0x32999`，以及三個來源
`PUSH (group,y,x)` 後呼叫 `0x35822` 的 staging 增援；後者保留座標與 helper
provenance，不降成一般 spawn_group，
只走 handler 自己的 basic-block 鏈(call 視為過站不進入被呼叫者本體,
遇 ret 停止),避免線性 sweep 漂移進共用子函式或下一個 handler。

用法(docker fd2-cap):
    python3 tools/extract_event_id_groups.py [out.json]
"""
import hashlib
import sys, struct, json
sys.path.insert(0, "/work/tools")
from le_xref import parse_le
from capstone import Cs, CS_ARCH_X86, CS_MODE_32

EXE = "org_game/炎龍騎士團/FLAME2/FD2.EXE"
CODE_BASE = 0x10000
EXPECTED_SIZE = 357074
EXPECTED_MD5 = "b97caf2239a27a896069d03549d96e1e"
EXPECTED_SHA256 = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"


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
if (len(d) != EXPECTED_SIZE or hashlib.md5(d).hexdigest() != EXPECTED_MD5 or
        hashlib.sha256(d).hexdigest() != EXPECTED_SHA256):
    raise RuntimeError("FD2.EXE 與固定參考版本不符；禁止沿用 event handler 位址")
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
ACTING_FN = 0x1366a
STAGING_HELPER = 0x35822
STAGING_SHARED_TAIL = 0x35318

# Complete [0x53AFA] writer set for global event handlers.  Official IDA Pro
# 9.4 finds the single reader in 0x10C50 and all paired 1/0 writers; Docker
# Capstone independently verifies each direct call sequence.  Calls not in
# this set read zero, including the 0x32999 wrapper's internal 0x10B4E call.
RAW_PLACEMENT_GATE_ONE_CALLS = {
    0x34397, 0x3444C, 0x3464B, 0x34945,
    0x34C95, 0x34D12, 0x34D45, 0x34D91,
}

# event47／49 的 EAX 來源已由固定雜湊 IDA Pro 9.4 主證據閉合。這兩個
# call-site 都使用 `mov eax,[0x53bef]; mov edx,eax; sar edx,31;
# sub eax,edx; sar eax,1; push eax`，亦即有號除以二、向零截斷。
# 保留 symbolic formula，不能在 per-handler 清冊中假造單一常數 group。
DYNAMIC_GROUP_CALLS = {
    0x3512B: {
        'group': '$round_counter_div2[0x53bef]',
        'group_formula': 'signed_trunc_toward_zero(round_counter/2)',
        'evidence': 'docs/data/ida/fd2_reinforcement_eax_sources.json#handlers.47',
    },
    0x35202: {
        'group': '$round_counter_div2[0x53bef]',
        'group_formula': 'signed_trunc_toward_zero(round_counter/2)',
        'evidence': 'docs/data/ida/fd2_reinforcement_eax_sources.json#handlers.49',
    },
}


def staging_spawn(pushes, invoker):
    """把 0x35822 的來源 PUSH (group,y,x) 保留成不可直接降階的 staging call。"""
    if len(pushes) < 3:
        return None
    group, y, x = pushes[-3:]
    if not all(isinstance(value, int) and value >= 0 for value in (group, y, x)):
        return None
    return {
        'group': group,
        'via': 'staging_helper_0x35822',
        'source': hex(invoker),
        'raw_placement_gate': 0,
        'staging': {
            'helper': '0x35822',
            'spawn_source': '0x35842',
            'x': x,
            'y': y,
        },
    }


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
        pending_pushes = []
        awaiting_intro = None
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
                        pending_pushes.append(int(op, 0))
                    except ValueError:
                        pending_pushes = []
                elif '0x3bef' in op:
                    pending_pushes.append('$turn_counter[0x53bef]')
                elif '0x3ae1' in op:
                    pending_pushes.append('$0x53ae1')
                else:
                    pending_pushes.append(f'$reg_or_mem({op})')
            elif m == 'call':
                if op.startswith('0x'):
                    t = int(op, 16)
                    pending_push = pending_pushes[-1] if pending_pushes else None
                    if t in SPAWN_FNS and pending_push is not None:
                        dynamic_group = DYNAMIC_GROUP_CALLS.get(ins.address)
                        spawn = {
                            'group': (
                                dynamic_group['group']
                                if dynamic_group is not None
                                else pending_push
                            ),
                            'via': SPAWN_FNS[t],
                            'source': hex(ins.address),
                            'raw_placement_gate': (
                                1 if ins.address in RAW_PLACEMENT_GATE_ONE_CALLS else 0
                            ),
                        }
                        if dynamic_group is not None:
                            spawn['group_formula'] = dynamic_group['group_formula']
                            spawn['evidence'] = dynamic_group['evidence']
                        spawns.append(spawn)
                        awaiting_intro = spawn if t == 0x32999 else None
                    elif t == STAGING_HELPER:
                        spawn = staging_spawn(pending_pushes, ins.address)
                        if spawn is not None:
                            spawns.append(spawn)
                    elif t == ACTING_FN and pending_push is not None and awaiting_intro is not None:
                        awaiting_intro['following_acting'] = {
                            'resource': pending_push,
                            'source': hex(ins.address),
                        }
                        awaiting_intro = None
                pending_pushes = []
                a = nxt
                continue
            elif m == 'jmp':
                awaiting_intro = None
                if op.startswith('0x'):
                    t = int(op, 16)
                    if t == STAGING_SHARED_TAIL:
                        target = insn_at(t)
                        if target is not None and target.mnemonic == 'call' and target.op_str == hex(STAGING_HELPER):
                            spawn = staging_spawn(pending_pushes, ins.address)
                            if spawn is not None:
                                spawns.append(spawn)
                        a = None
                        continue
                    a = t
                    continue
                a = None
                break
            elif m.startswith('j'):
                awaiting_intro = None
                if op.startswith('0x'):
                    t = int(op, 16)
                    if t not in visited:
                        stack.append(t)
                a = nxt
                pending_pushes = []
                continue
            elif m in ('ret', 'retn'):
                awaiting_intro = None
                a = None
                break
            else:
                pending_pushes = []
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
    handlers = jtab(0x51b91, 90)
    results = {
        '_source': {
            'file': 'FD2.EXE',
            'size': EXPECTED_SIZE,
            'md5': EXPECTED_MD5,
            'sha256': EXPECTED_SHA256,
            'tool': 'Capstone 5.0.3',
            'address_space': 'LE linear address',
        },
    }
    for eid, h in enumerate(handlers):
        spawns = walk_handler(h)
        results[eid] = {'handler': hex(h), 'spawns': spawns}
    out = sys.argv[1] if len(sys.argv) > 1 else 'docs/data/event_id_groups.json'
    with open(out, 'w', encoding='utf-8') as output:
        json.dump(results, output, indent=1, ensure_ascii=False)
        output.write('\n')
    staging_calls = sum(
        call.get('via') == 'staging_helper_0x35822'
        for metadata in results.values()
        if isinstance(metadata, dict)
        for call in metadata.get('spawns', [])
    )
    print(f"90 個 event handlers -> {out}；staging calls={staging_calls}")
