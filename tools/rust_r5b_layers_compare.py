#!/usr/bin/env python3
"""R5 階段 2 r5b（docs/goal/121）：比對 Go 與 Rust 的 r5b-layers.json（tools/rust_r5b_layers.sh 產生）。

用法：
  rust_r5b_layers_compare.py --go <r5b-layers.json> --rust <r5b-layers.json> [--stage0 <frames.json>]

- 逐擷取點展開 rows（count 連續相同），比對每一張的比對鍵（分支、Draw 前後的工作狀態、整張雜湊）；印出擷取數、
  序列雜湊、各分支的張數與相同數、相異畫格數、第一個不同的擷取點與兩邊的鍵。不遮蔽任何欄位。
- --stage0：階段 0 chapter 黃金（整張 640×400 Draw）frames.json。ch28、idle、atk（本組）與 ramp、spawn 分支在兩邊都
  呼叫整個 Draw，要求每一張的雜湊都與同一擷取點的整張黃金相同（不同即失敗）；兩邊都有列開頭的擷取點另比 update_frame。

結束碼：全部相同為 0，否則 1。
"""
import argparse
import json
import sys

FULL = ('ch28', 'idle', 'atk', 'ramp', 'spawn')
OURS = ('ch28', 'idle', 'atk')


def expand(rows, field):
    out = []
    for row in rows:
        if row['capture'] != len(out):
            raise SystemExit(f"rows 的 capture 不連續：{row['capture']} != {len(out)}")
        out.extend([row[field]] * row['count'])
    return out


def fields(key):
    parts = key.split('|')
    out = {'branch': parts[0]}
    for p in parts[1:]:
        k, _, v = p.partition('=')
        out[k] = v
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--go', required=True)
    ap.add_argument('--rust', required=True)
    ap.add_argument('--stage0')
    a = ap.parse_args()
    go = json.load(open(a.go, encoding='utf-8'))
    rs = json.load(open(a.rust, encoding='utf-8'))
    gk, rk = expand(go['rows'], 'key'), expand(rs['rows'], 'key')
    print(f"擷取數：Go {go['captures']}（展開 {len(gk)}）、Rust {rs['captures']}（展開 {len(rk)}）")
    print(f"序列雜湊：Go {go['sequence_sha256']}、Rust {rs['sequence_sha256']}"
          f"（{'相同' if go['sequence_sha256'] == rs['sequence_sha256'] else '不同'}）")
    print(f"時鐘：Go {go['clock']}、Rust {rs['clock']}；virtual_frames Go {go['virtual_frames']}、Rust {rs['virtual_frames']}")
    print(f"分支：Go {go['branches']}；Rust {rs['branches']}")
    print(f"統計：Go {go['stats']}；Rust {rs['stats']}")
    n = min(len(gk), len(rk))
    per = {}
    first = None
    for i in range(n):
        gf = fields(gk[i])
        c = per.setdefault(gf['branch'], [0, 0])
        c[0] += 1
        c[1] += gk[i] == rk[i]
        if gk[i] != rk[i] and first is None:
            first = i
    for name in sorted(per):
        total, same = per[name]
        print(f"  {name}：{total} 張，相同 {same}")
    distinct = {}
    for k in gk:
        f = fields(k)
        if 'rgba' in f:
            distinct.setdefault(f['branch'], set()).add(f['rgba'])
    print('Go 相異畫格數：' + '、'.join(f'{k} {len(v)}' for k, v in sorted(distinct.items())))
    ok = len(gk) == len(rk) and first is None and go['sequence_sha256'] == rs['sequence_sha256']
    if first is not None:
        print(f"第一個不同：capture {first}")
        print(f"  Go  ：{gk[first]}")
        print(f"  Rust：{rk[first]}")
    elif len(gk) != len(rk):
        print(f"擷取數不同，前 {n} 張相同")
    if a.stage0:
        s0 = json.load(open(a.stage0, encoding='utf-8'))
        sk = expand(s0['rows'], 'rgba_sha256')
        sp = expand(s0['rows'], 'draw_path')
        print(f"階段 0 整張黃金：{a.stage0}（{s0['mode']}，{len(sk)} 張，序列 {s0['sequence_sha256']}）")
        if len(sk) != len(gk):
            print(f"  擷取數與階段 0 不同（{len(sk)} 對 {len(gk)}）")
            ok = False
        # rows 依鍵（階段 0 依雜湊）去重，update_frame 只在每列開頭那一次擷取有意義；只比兩邊都有列開頭的擷取點。
        gs = {r['capture']: r['update_frame'] for r in go['rows']}
        ss = {r['capture']: r['update_frame'] for r in s0['rows']}
        common = sorted(set(gs) & set(ss))
        frame_diff = [c for c in common if gs[c] != ss[c]]
        print(f"  update_frame：兩邊都有列開頭的 {len(common)} 個擷取點中 {len(frame_diff)} 個不同")
        ok &= not frame_diff
        stat = {b: [0, 0, None] for b in FULL}
        paths = {}
        for i in range(min(len(gk), len(sk))):
            f = fields(gk[i])
            b = f['branch']
            if b not in stat:
                continue
            stat[b][0] += 1
            same = f['rgba'] == sk[i]
            stat[b][1] += same
            if not same and stat[b][2] is None:
                stat[b][2] = i
            paths.setdefault(b, {}).setdefault(sp[i], 0)
            paths[b][sp[i]] += 1
        for b in FULL:
            total, same, bad = stat[b]
            tag = '本組' if b in OURS else '別組（只留狀態）'
            line = f"  {b}（{tag}，整個 Draw）：{total} 張中 {same} 張與階段 0 同一擷取點相同"
            if bad is not None:
                line += f"；第一個不同在 capture {bad}（Go 鍵 {gk[bad]}；階段 0 {sk[bad]}）"
                ok = False
            print(line)
        print(f"  階段 0 draw_path：{paths}")
    print('結論：' + ('全部相同' if ok else '有不同'))
    return 0 if ok else 1


if __name__ == '__main__':
    sys.exit(main())
