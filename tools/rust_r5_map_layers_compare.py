#!/usr/bin/env python3
"""R5 階段 2 r5m（docs/goal/121）：比對 Go 與 Rust 的 map-layers.json（tools/rust_r5_map_layers.sh 產生）。

用法：rust_r5_map_layers_compare.py --go <map-layers.json> --rust <map-layers.json> [--stage0 <frames.json>]

- 逐擷取點展開 rows（count 連續相同），比對每一張的比對鍵；印出擷取數、序列雜湊、各圖層的張數與相同數、
  第一個不同的擷取點與兩邊的鍵。不遮蔽任何欄位。
- --stage0：階段 0 chapter 黃金（整張 640×400 Draw）的 frames.json。對 Go 鍵裡畫成的 spawn 與 map 畫格，
  另外數有幾張與同一擷取點的整張黃金逐位元組相同（整張畫格上沒有別組圖層的那些）；只是參考，不影響結束碼。

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
    layer = {}  # 圖層 → [張數, 相同數]
    first = None
    for i in range(n):
        gf, rf = fields(gk[i]), fields(rk[i])
        for name, keys in (('branch', ['branch']), ('spawn', ['ok', 'rgba']), ('world', ['legacy', 'world', 'units', 'acted', 'fallback']),
                           ('admission', ['admitted']), ('map', ['map_ok', 'map', 'hud']), ('index', ['index', 'palette'])):
            if not any(k in gf for k in keys) and name != 'branch':
                continue
            c = layer.setdefault(name, [0, 0])
            c[0] += 1
            c[1] += all(gf.get(k) == rf.get(k) for k in keys)
        if gk[i] != rk[i] and first is None:
            first = i
    for name, (total, same) in layer.items():
        print(f"  {name}：{total} 張，相同 {same}")
    distinct = {name: set() for name in ('spawn', 'world', 'map', 'index')}
    for k in gk:
        f = fields(k)
        for name, key in (('spawn', 'rgba'), ('world', 'world'), ('map', 'map'), ('index', 'index')):
            if key in f:
                distinct[name].add(f[key])
    print('Go 相異畫格數：' + '、'.join(f'{k} {len(v)}' for k, v in distinct.items()))
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
        stat = {'spawn': [0, 0], 'map': [0, 0]}
        for i in range(min(len(gk), len(sk))):
            f = fields(gk[i])
            if f['branch'] == 'spawn' and f.get('ok') == 'true':
                stat['spawn'][0] += 1
                stat['spawn'][1] += f['rgba'] == sk[i]
            if f.get('map_ok') == 'true':
                stat['map'][0] += 1
                stat['map'][1] += f['map'] == sk[i]
        print(f"階段 0 整張黃金（{a.stage0}，{len(sk)} 張）：" + '、'.join(
            f'{k} {v[0]} 張中 {v[1]} 張與整張畫格相同' for k, v in stat.items()) + '（參考）')
    print('結論：' + ('Go 與 Rust 每個擷取點相同' if ok else 'Go 與 Rust 不同'))
    return 0 if ok else 1


if __name__ == '__main__':
    sys.exit(main())
