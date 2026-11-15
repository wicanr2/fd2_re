#!/usr/bin/env python3
"""R5 階段 2 r5p（docs/goal/121）：比對 Go 與 Rust 的 r5p-layers.json（tools/rust_r5p_layers.sh 產生）。

用法：
  rust_r5p_layers_compare.py --go <r5p-layers.json> --rust <r5p-layers.json> [--stage0 <frames.json>]
  rust_r5p_layers_compare.py --verbatim <remake/cmd/fd2 目錄>

- 逐擷取點展開 rows（count 連續相同），比對每一張的比對鍵；印出擷取數、序列雜湊、各圖層（ramp、world、fade）的
  張數與相同數、第一個不同的擷取點與兩邊的鍵。不遮蔽任何欄位。
- --stage0：同模式的階段 0 黃金（整張 640×400 Draw）frames.json。ramp 分支在 Go 與 Rust 兩邊都呼叫整個 Draw，
  要求每一張都與同一擷取點的整張黃金相同（不同即失敗）；world、fade 只是本組圖層，另外數有幾張剛好等於整張
  畫格（整張畫格上沒有別組圖層的那些），只是參考。
- --verbatim：核對 rust_golden_r5p_layers_test.go 裡兩段「逐字複本」與 main.go 對應行逐字相同。

結束碼：全部相同為 0，否則 1。
"""
import argparse
import json
import os
import re
import sys

LAYERS = (
    ('ramp', ['step', 'drawn', 'hd_sources', 'dac_white', 'dac_black', 'help', 'rgba']),
    ('world', ['legacy', 'sel', 'reach', 'cast', 'attack', 'world']),
    ('map_ok', ['map_ok']),
    ('fade', ['fade_out', 'fade_t', 'fade_alpha', 'fade_rgba']),
)


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


def verbatim(src_dir):
    main = open(os.path.join(src_dir, 'main.go'), encoding='utf-8').read().split('\n')
    test = open(os.path.join(src_dir, 'rust_golden_r5p_layers_test.go'), encoding='utf-8').read().split('\n')
    ok = True
    starts = [i for i, l in enumerate(test) if '---- 逐字複本開始：main.go:' in l]
    if len(starts) != 2:
        print(f'逐字複本標記數 {len(starts)}，應為 2')
        return False
    for i in starts:
        m = re.search(r'main\.go:(\d+)\.\.(\d+)', test[i])
        a, b = int(m.group(1)), int(m.group(2))
        end = next(j for j in range(i + 1, len(test)) if f'---- 逐字複本結束：main.go:{a}..{b} ----' in test[j])
        copy, orig = test[i + 1:end], main[a - 1:b]
        same = copy == orig
        ok &= same
        print(f'逐字複本 main.go:{a}..{b}（{len(orig)} 行）：{"與 main.go 逐字相同" if same else "與 main.go 不同"}')
    return ok


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--go')
    ap.add_argument('--rust')
    ap.add_argument('--stage0')
    ap.add_argument('--verbatim')
    a = ap.parse_args()
    if a.verbatim:
        return 0 if verbatim(a.verbatim) else 1
    go = json.load(open(a.go, encoding='utf-8'))
    rs = json.load(open(a.rust, encoding='utf-8'))
    gk, rk = expand(go['rows'], 'key'), expand(rs['rows'], 'key')
    print(f"模式：Go {go['mode']}、Rust {rs['mode']}")
    print(f"擷取數：Go {go['captures']}（展開 {len(gk)}）、Rust {rs['captures']}（展開 {len(rk)}）")
    print(f"序列雜湊：Go {go['sequence_sha256']}、Rust {rs['sequence_sha256']}"
          f"（{'相同' if go['sequence_sha256'] == rs['sequence_sha256'] else '不同'}）")
    print(f"時鐘：Go {go['clock']}、Rust {rs['clock']}；virtual_frames Go {go['virtual_frames']}、Rust {rs['virtual_frames']}")
    print(f"分支：Go {go['branches']}；Rust {rs['branches']}")
    print(f"統計：Go {go['stats']}；Rust {rs['stats']}")
    n = min(len(gk), len(rk))
    layer = {}
    first = None
    for i in range(n):
        gf, rf = fields(gk[i]), fields(rk[i])
        c = layer.setdefault('branch', [0, 0])
        c[0] += 1
        c[1] += gf['branch'] == rf['branch']
        for name, keys in LAYERS:
            if name == 'fade' and gf.get('fade') != 'true':
                continue
            if not any(k in gf for k in keys):
                continue
            c = layer.setdefault(name, [0, 0])
            c[0] += 1
            c[1] += all(gf.get(k) == rf.get(k) for k in keys)
        if gk[i] != rk[i] and first is None:
            first = i
    for name, (total, same) in layer.items():
        print(f"  {name}：{total} 張，相同 {same}")
    distinct = {'ramp': set(), 'world': set(), 'fade': set()}
    for k in gk:
        f = fields(k)
        for name, key in (('ramp', 'rgba'), ('world', 'world'), ('fade', 'fade_rgba')):
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
        sf = expand(s0['rows'], 'update_frame')
        print(f"階段 0 整張黃金：{a.stage0}（{s0['mode']}，{len(sk)} 張，序列 {s0['sequence_sha256']}）")
        gu = expand(go['rows'], 'update_frame')
        if len(sk) != len(gk):
            print(f"  擷取數與階段 0 不同（{len(sk)} 對 {len(gk)}）")
        stat = {'ramp': [0, 0], 'world': [0, 0], 'fade': [0, 0]}
        ramp_first = None
        for i in range(min(len(gk), len(sk))):
            f = fields(gk[i])
            if f['branch'] == 'ramp':
                stat['ramp'][0] += 1
                same = f['rgba'] == sk[i]
                stat['ramp'][1] += same
                if not same and ramp_first is None:
                    ramp_first = i
            if f['branch'] == 'tail':
                stat['world'][0] += 1
                stat['world'][1] += f['world'] == sk[i]
                if f.get('fade') == 'true':
                    stat['fade'][0] += 1
                    stat['fade'][1] += f['fade_rgba'] == sk[i]
        print('  ' + '、'.join(f'{k} {v[0]} 張中 {v[1]} 張與整張畫格相同' for k, v in stat.items()))
        if stat['ramp'][0]:
            frames_ok = ramp_first is None and len(sk) == len(gk)
            print(f"  ramp 是整個 Draw：{'每一張都與階段 0 相同' if ramp_first is None else f'第一個不同在 capture {ramp_first}'}")
            if ramp_first is not None:
                print(f"    Go 鍵：{gk[ramp_first]}；階段 0：{sk[ramp_first]}（update_frame Go {gu[ramp_first]}、階段 0 {sf[ramp_first]}）")
            ok &= frames_ok
        print('  world／fade 只是本組圖層，與整張畫格相同的張數只是參考')
    print('結論：' + ('Go 與 Rust 每個擷取點相同' if ok else '不同'))
    return 0 if ok else 1


if __name__ == '__main__':
    sys.exit(main())
