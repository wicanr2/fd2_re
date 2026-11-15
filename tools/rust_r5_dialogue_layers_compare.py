#!/usr/bin/env python3
"""R5 階段 2 r5d（docs/goal/121）：比對 Go 與 Rust 的 dialogue-layers.json（tools/rust_r5_dialogue_layers.sh 產生）。

用法：rust_r5_dialogue_layers_compare.py --go <dialogue-layers.json> --rust <dialogue-layers.json> [--stage0 <frames.json>]

- 逐擷取點展開 rows（count 連續相同），比對每一張的比對鍵；印出擷取數、序列雜湊、嘴型狀態（每次擷取都有）與
  對白畫格（drawNativeStoryDialogue 回 true 的擷取）各自的張數與相同數、各階段（收框、開框、穩定頁）與穩定頁
  合成條件（說話嘴型、等鍵張嘴、等鍵箭頭）的張數、第一個不同的擷取點與兩邊的鍵。不遮蔽任何欄位。
- --stage0：階段 0 chapter 黃金（整張 640×400 Draw）的 frames.json。對 Go 鍵裡畫成的對白畫格，另外數有幾張與
  同一擷取點的整張黃金逐位元組相同（presentNativeClassFrame 蓋滿全畫面、之後沒有別組圖層的那些）；只是參考，
  不影響結束碼。

結束碼：兩邊所有擷取點相同為 0，否則 1。
"""
import argparse
import json
import sys


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


MOUTH = ('mouth', 'confirm')
DIALOGUE = ('stage', 'closing_t', 'dlg_t', 'page', 'progress', 'speaking', 'mouth_open', 'arrow', 'err', 'rgba')


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
    print(f"全域亂數：Go global_rand_seed={go.get('global_rand_seed')}；Rust {rs.get('global_rand')}")
    print(f"分支：Go {go['branches']}；Rust {rs['branches']}")
    print(f"統計：Go {go['stats']}；Rust {rs['stats']}")
    n = min(len(gk), len(rk))
    count = {'mouth': [0, 0], 'dialogue': [0, 0], 'branch': [0, 0]}
    first = None
    for i in range(n):
        gf, rf = fields(gk[i]), fields(rk[i])
        count['branch'][0] += 1
        count['branch'][1] += gf['branch'] == rf['branch'] and gf.get('dialogue') == rf.get('dialogue')
        count['mouth'][0] += 1
        count['mouth'][1] += all(gf.get(k) == rf.get(k) for k in MOUTH)
        if gf.get('dialogue') == 'true':
            count['dialogue'][0] += 1
            count['dialogue'][1] += all(gf.get(k) == rf.get(k) for k in DIALOGUE)
        if gk[i] != rk[i] and first is None:
            first = i
    for name, label in (('branch', '分支與是否畫對白'), ('mouth', '嘴型狀態'), ('dialogue', '對白畫格（狀態與 RGBA）')):
        total, same = count[name]
        print(f"  {label}：{total} 張，相同 {same}")
    distinct = set()
    breakdown = {}
    for k in gk:
        f = fields(k)
        if f.get('dialogue') != 'true':
            continue
        distinct.add(f['rgba'])
        breakdown[f['stage']] = breakdown.get(f['stage'], 0) + 1
        for cond in ('speaking', 'mouth_open', 'arrow'):
            if f[cond].split(':')[-1 if cond == 'speaking' else 0] == 'true':
                breakdown[cond] = breakdown.get(cond, 0) + 1
    print(f"Go 對白相異畫格數：{len(distinct)}；分類：{breakdown}")
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
        total = same = 0
        for i in range(min(len(gk), len(sk))):
            f = fields(gk[i])
            if f.get('dialogue') == 'true':
                total += 1
                same += f['rgba'] == sk[i]
        print(f"階段 0 整張黃金（{a.stage0}，{len(sk)} 張）：對白 {total} 張中 {same} 張與整張畫格相同（參考）")
    print('結論：' + ('Go 與 Rust 每個擷取點相同' if ok else 'Go 與 Rust 不同'))
    return 0 if ok else 1


if __name__ == '__main__':
    sys.exit(main())
