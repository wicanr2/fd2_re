#!/usr/bin/env python3
"""炎龍騎士團2 — DOS4GW LE 遞迴反組譯 / call-graph 工具。

動機:線性 sweep(disasm_le.py 的 range/calls)會在資料區、跳表、字串池「漂移」,
產生偽指令與偽 call,害人把資料當 caller。本工具改做**遞迴可達性反組譯**:
從種子函式出發,只跟隨真正會執行到的 call/jcc/jmp(立即數)目標,標記「可達指令集」。
於是「誰呼叫 X」只回報落在可達指令集內的 call/jmp —— 排除資料偽命中。

間接呼叫 `call [reg*4 + table]`(章節跳表)無法自動跟隨 → 用 --seed 把跳表各 entry
手動注入種子,或先用 `jtab` 子命令把跳表解出再餵回。

linear↔file:obj1 code,file = linear − 0xe00(base 0x10000)。

用法:
  python3 callgraph_le.py <EXE> reach <seed_hex>[,<seed_hex>...]      建可達集,印統計
  python3 callgraph_le.py <EXE> callers <target_hex> [seeds_csv]      可達 caller(預設種子=main 0x25bf4)
  python3 callgraph_le.py <EXE> edges <func_hex> [seeds_csv]          某函式內所有 call 目標(可達)
  python3 callgraph_le.py <EXE> jtab <table_hex> <count>              用 fixup 解跳表 N 個 entry
  python3 callgraph_le.py <EXE> path <from_hex> <to_hex> [seeds_csv]  call 路徑(BFS 最短)
"""
import sys, struct
sys.path.insert(0, __file__.rsplit('/', 1)[0])
from le_xref import parse_le

try:
    from capstone import Cs, CS_ARCH_X86, CS_MODE_32
except ImportError:
    sys.exit("need capstone: docker fd2-cap")

CODE_BASE = 0x10000
DEFAULT_SEEDS = [0x25bf4]  # 已驗證的真 main


def load_code(d, meta):
    o = meta['objs'][0]
    start = meta['data_off'] + (o['base'] - CODE_BASE)
    return d[start:start + o['vsize']], o['base'], o['vsize']


def page_base_linear(meta, pg):
    """全域 page index(0-based)→ 該 page 的 linear 起點(跨 object)。"""
    psize = meta['page_size']
    for o in meta['objs']:
        first = o['first'] - 1  # first 是 1-based page map index
        if first <= pg < first + o['pages']:
            return o['base'] + (pg - first) * psize
    return None


def fixup_map(d, meta):
    """{linear: target_abs}(32-bit offset fixups);涵蓋全部 object(code+data)。"""
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


class CG:
    def __init__(self, exe):
        self.d = open(exe, 'rb').read()
        self.meta = parse_le(self.d)
        self.code, self.base, self.vsize = load_code(self.d, self.meta)
        self.md = Cs(CS_ARCH_X86, CS_MODE_32)
        self.end = self.base + self.vsize
        self.calls = {}   # call_site_addr -> target  (direct call,可達)
        self.jmps = {}    # jmp_site_addr -> target
        self.reached = set()  # 可達指令位址

    def _insn(self, addr):
        off = addr - self.base
        if off < 0 or off >= len(self.code):
            return None
        for ins in self.md.disasm(self.code[off:off + 15], addr):
            return ins
        return None

    def build(self, seeds):
        """遞迴可達反組譯。"""
        stack = list(seeds)
        seen_starts = set()
        while stack:
            a = stack.pop()
            # 線性走一條 basic-block 鏈,遇 call 繼續(會返回),遇 ret/jmp 終止
            while a is not None and a not in self.reached and self.base <= a < self.end:
                ins = self._insn(a)
                if ins is None:
                    break
                self.reached.add(a)
                m = ins.mnemonic
                op = ins.op_str
                nxt = a + ins.size
                if m == 'call':
                    if op.startswith('0x'):
                        t = int(op, 16)
                        self.calls[a] = t
                        if self.base <= t < self.end and t not in seen_starts:
                            seen_starts.add(t); stack.append(t)
                    a = nxt  # call 之後繼續
                elif m == 'jmp':
                    if op.startswith('0x'):
                        t = int(op, 16)
                        self.jmps[a] = t
                        if self.base <= t < self.end:
                            a = t  # 跟隨無條件跳
                            continue
                    a = None  # 間接 jmp / 出界 → 終止此鏈
                elif m.startswith('j'):  # 條件跳:兩路都走
                    if op.startswith('0x'):
                        t = int(op, 16)
                        self.jmps[a] = t
                        if self.base <= t < self.end and t not in self.reached:
                            stack.append(t)
                    a = nxt
                elif m in ('ret', 'retn', 'retf', 'iret'):
                    a = None
                else:
                    a = nxt

    def callers(self, target):
        return ([a for a, t in self.calls.items() if t == target],
                [a for a, t in self.jmps.items() if t == target])

    def entries(self, seeds):
        """所有函式入口 = 被 direct call 的目標 + 種子。"""
        return sorted(set(self.calls.values()) | set(seeds))

    def funcof(self, addr, ents):
        """包含 addr 的函式入口 = 最大的 entry <= addr。"""
        lo = [e for e in ents if e <= addr]
        return max(lo) if lo else None

    def rpath(self, to, seeds, maxdepth=12):
        """反向 BFS:從 to 找到 main(種子)的 call 路徑(函式層級)。"""
        ents = self.entries(seeds)
        # 反向邊:func_entry -> {它呼叫的 target}  反過來查
        callers_of = {}
        for a, t in self.calls.items():
            f = self.funcof(a, ents)
            if f is None:
                continue
            callers_of.setdefault(t, []).append((f, a))
        seedset = set(seeds)
        # BFS
        from collections import deque
        q = deque([(to, [to])])
        seen = {to}
        results = []
        while q:
            node, path = q.popleft()
            if node in seedset:
                results.append(path); continue
            if len(path) > maxdepth:
                continue
            for f, site in sorted(set(callers_of.get(node, []))):
                if f not in seen:
                    seen.add(f)
                    q.append((f, path + [f]))
        return results


def parse_seeds(s):
    return [int(x, 16) for x in s.split(',') if x.strip()]


def main(av):
    if len(av) < 3:
        print(__doc__); return 1
    cg = CG(av[1]); cmd = av[2]

    if cmd == 'jtab':
        tab = int(av[3], 16); n = int(av[4])
        fx = fixup_map(cg.d, cg.meta)
        for i in range(n):
            site = tab + i * 4
            t = fx.get(site)
            print(f'  [{i:2d}] {site:#08x} -> {hex(t) if t else "(no fixup)"}')
        return 0

    seeds = parse_seeds(av[4]) if len(av) > 4 and cmd in ('callers', 'edges', 'path') else \
            (parse_seeds(av[3]) if cmd == 'reach' else DEFAULT_SEEDS)
    if cmd == 'reach':
        cg.build(seeds)
        print(f'種子 {[hex(s) for s in seeds]}:可達指令 {len(cg.reached)},direct call 點 {len(cg.calls)}')
        return 0

    cg.build(seeds)
    if cmd == 'callers':
        tgt = int(av[3], 16)
        c, j = cg.callers(tgt)
        print(f'可達 caller of {hex(tgt)}(種子 {[hex(s) for s in seeds]}):')
        for a in sorted(c): print(f'  call @ {a:#08x}')
        for a in sorted(j): print(f'  jmp  @ {a:#08x}')
        if not c and not j: print('  (可達集內無直接 call/jmp;可能經間接呼叫/跳表,或不可達)')
        return 0
    if cmd == 'edges':
        f = int(av[3], 16)
        # 只列該函式入口起、ret 前的 call(近似:用建好的全域 calls,過濾範圍)
        outs = sorted([(a, t) for a, t in cg.calls.items() if f <= a < f + 0x600])
        for a, t in outs: print(f'  {a:#08x} call {hex(t)}')
        return 0
    if cmd == 'path':
        frm, to = int(av[3], 16), int(av[4], 16)
        c, j = cg.callers(to)
        print(f'{hex(to)} 的直接可達 caller:{[hex(x) for x in sorted(c+j)]}')
        return 0
    if cmd == 'funcof':
        a = int(av[3], 16)
        print(hex(cg.funcof(a, cg.entries(seeds))))
        return 0
    if cmd == 'rpath':
        to = int(av[3], 16)
        res = cg.rpath(to, seeds)
        if not res:
            print(f'{hex(to)} → main 無 direct-call 路徑(可能經跳表/間接)')
        for p in res[:6]:
            print(' → '.join(hex(x) for x in reversed(p)))
        return 0
    print(__doc__); return 1


if __name__ == '__main__':
    sys.exit(main(sys.argv))
