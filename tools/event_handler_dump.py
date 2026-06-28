#!/usr/bin/env python3
"""炎龍騎士團2 — 戰場事件 handler 反組譯 + 事件原語標註。

針對章節戰場事件跳表 0x51b19 的各章 handler(doc 25),遞迴反組譯單一 handler
(從入口跟隨 jcc/jmp 到 ret,不跟隨 call),並把已知事件原語標成可讀語意,
產出「條件→動作」序列,供 remake 改寫成資料驅動腳本(非 hardcoding)。

已知原語(doc 23/24/25):
  0x3453e(idx)         查單位 #idx 狀態 [0x53a45][idx+5]&1
  0x205be / 0x205da    handler prologue(載該章 FDTXT 文本、預設結果碼)
  0x15f84              繪事件畫面(全螢幕圖)
  0x1088d              載章節文本資源
  [0x53ecc]=N          設結果碼(1=中途事件 / 2=勝利 / 0=續打)
  [0x53ec8]            回合計數(clamp 99)
  [0x53a45]            戰場單位陣列基底(每單位 0x50B)
  [0x53c03]            目前章節

用法:
  python3 event_handler_dump.py <EXE> <handler_hex> [end_hex]    dump 單一 handler
  python3 event_handler_dump.py <EXE> table                       dump 0x51b19 全 30 章 handler
"""
import sys
sys.path.insert(0, __file__.rsplit('/', 1)[0])
from callgraph_le import CG, fixup_map

PRIM = {
    0x3453e: 'unit_alive?(idx)', 0x205be: 'prologue(載章文本/預設碼)', 0x205da: 'prologue2',
    0x15f84: '繪畫面', 0x1088d: '載章節文本', 0x111ba: '載資源', 0x25977: 'play_bgm/scene',
    0x25a96: 'play_sfx', 0x36cd7: '__STK', 0x2cad7: '結局判定?', 0x18890: '戰鬥行動',
}
VAR = {0x53ecc: '結果碼', 0x53ec8: '回合數', 0x53a45: '單位陣列', 0x53c03: '章節', 0x51a83: 'flagA'}


def annot(ins, fx):
    m, op = ins.mnemonic, ins.op_str
    note = ''
    # 相對 call/jmp:target 直接在 op_str(E8/E9 rel32 不經 fixup)
    if m == 'call' and op.startswith('0x'):
        t = int(op, 16)
        note = f'  ; ★{PRIM[t]}' if t in PRIM else f'  ; →動作 {hex(t)}'
    else:
        # 絕對資料引用走 fixup(disp32 被重定位)
        for o in range(ins.address, ins.address + ins.size):
            if o in fx:
                t = fx[o]
                if t in VAR:
                    note = f'  ; ⟨{VAR[t]}⟩'
                break
    return f'{ins.address:#08x}  {m:<6} {op}{note}'


def dump(cg, fx, start, end):
    """遞迴反組譯單一函式(跟隨 jcc/jmp 到 ret,不跟隨 call)。"""
    out = {}
    stack = [start]
    while stack:
        a = stack.pop()
        while a is not None and a not in out and start <= a < end:
            ins = cg._insn(a)
            if not ins:
                break
            out[a] = ins
            m, op = ins.mnemonic, ins.op_str
            nxt = a + ins.size
            if m in ('ret', 'retn', 'retf'):
                a = None
            elif m == 'jmp':
                a = int(op, 16) if op.startswith('0x') else None
            elif m.startswith('j'):
                if op.startswith('0x'):
                    t = int(op, 16)
                    if start <= t < end and t not in out:
                        stack.append(t)
                a = nxt
            else:
                a = nxt
    return [out[k] for k in sorted(out)]


def main(av):
    if len(av) < 3:
        print(__doc__); return 1
    cg = CG(av[1]); fx = fixup_map(cg.d, cg.meta)
    if av[2] == 'table':
        hs = []
        for i in range(30):
            t = fx.get(0x51b19 + i * 4)
            if t:
                hs.append((i, t))
        uniq = sorted(set(t for _, t in hs))
        by = {}
        for i, t in hs:
            by.setdefault(t, []).append(i)
        for t in uniq:
            end = min([u for u in uniq if u > t] + [t + 0x300])
            chs = by[t]
            tag = '(default 殲滅即勝)' if t == 0x205b4 else ''
            print(f'\n=== handler {hex(t)}  章節 {chs} {tag} ===')
            for ins in dump(cg, fx, t, end):
                m, op = ins.mnemonic, ins.op_str
                keep = m in ('call', 'cmp', 'test') \
                    or (m == 'mov' and ('0x3ec' in op or '0x3a45' in op)) \
                    or (m == 'push' and op.startswith('0x') and not op.startswith('0x5'))
                if keep:
                    print(' ', annot(ins, fx))
        return 0
    if av[2] == 'json':
        import json
        SKIP = {0x36cd7, 0x205be, 0x205da, 0x1088d, 0x111ba, 0x375c0, 0x37416, 0x37244}
        COND = {0x3453e: 'unit_dead', 0x33499: 'roster_has'}  # 條件查詢原語(非動作)
        hs = [(i, fx.get(0x51b19 + i * 4)) for i in range(30)]
        uniq = sorted(set(t for _, t in hs if t))
        cache = {}
        out = []
        for i, t in hs:
            if not t:
                continue
            if t not in cache:
                end = min([u for u in uniq if u > t] + [t + 0x300])
                units, codes, draw, acts, conds = [], [], False, [], []
                lastpush = None
                for ins in dump(cg, fx, t, end):
                    m, op = ins.mnemonic, ins.op_str
                    if m == 'push' and op.startswith('0x'):
                        lastpush = int(op, 16)
                    elif m == 'call' and op.startswith('0x'):
                        tt = int(op, 16)
                        if tt == 0x3453e and lastpush is not None and lastpush < 0x100:
                            units.append(lastpush)
                        elif tt in COND:
                            conds.append(COND[tt])
                        elif tt == 0x15f84:
                            draw = True
                        elif tt not in SKIP:
                            acts.append(hex(tt))
                    elif m == 'mov' and '[0x3ecc],' in op:
                        v = op.split(',')[-1].strip()
                        if v.lstrip('-').isdigit():
                            codes.append(int(v))
                cache[t] = {
                    'handler': hex(t),
                    'is_default': t == 0x205b4,
                    'trigger_units_dead': sorted(set(units)),  # 需陣亡才觸發的單位 idx(0x3453e)
                    'result_codes': sorted(set(codes)),         # 1=中途事件 2=特殊勝利
                    'draw_scene': draw,                          # 是否繪事件畫面(0x15f84)
                    'extra_conditions': sorted(set(conds)),      # 其他條件查詢原語
                    'action_fns': sorted(set(acts)),             # 真動作函式(經修正後多為空)
                }
            out.append({'chapter': i, **cache[t]})
        print(json.dumps(out, ensure_ascii=False, indent=1))
        return 0
    start = int(av[2], 16)
    end = int(av[3], 16) if len(av) > 3 else start + 0x300
    for ins in dump(cg, fx, start, end):
        print(annot(ins, fx))
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv))
