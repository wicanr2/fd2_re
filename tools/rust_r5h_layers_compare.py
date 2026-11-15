#!/usr/bin/env python3
"""R5 階段 2 r5h（docs/goal/121，HUD 與選單）：比對 Go 與 Rust 的 r5h-layers.json（tools/rust_r5h_layers.sh 產生）。

用法：
  rust_r5h_layers_compare.py --go <r5h-layers.json> --rust <r5h-layers.json> [--stage0 <frames.json>]
  rust_r5h_layers_compare.py --verbatim <remake/cmd/fd2 目錄>

- 逐擷取點展開 rows（count 連續相同），比對每一張的比對鍵；印出擷取數、序列雜湊、各圖層（recovery、cursor_hud、
  overlays、command）的張數、有畫的張數與相同數、第一個不同的擷取點與兩邊的鍵。不遮蔽任何欄位。
- --stage0：階段 0 chapter 黃金（整張 640×400 Draw）frames.json。recovery 分支在兩邊都呼叫整個 Draw，要求每一張
  都與同一擷取點的整張黃金相同（不同即失敗）；tail 的三層只是本組圖層，另外數有幾張剛好等於整張畫格，只是參考。
- --verbatim：核對 rust_golden_r5h_layers_test.go 裡三段「逐字複本」與 main.go 對應行逐字相同。

結束碼：全部相同為 0，否則 1。
"""
import argparse
import json
import os
import re
import sys

LAYERS = (
    ('recovery', ['frame', 'alive', 'drawn', 'rgba'], 'rgba'),
    ('cursor_hud', ['legacy', 'cursor', 'hud_unit', 'cursor_hud'], 'cursor_hud'),
    ('map', ['map'], None),
    ('overlays', ['move', 'banner_fallback', 'overlays'], 'overlays'),
    ('ttf_return', ['ttf_return'], None),
    ('command', ['transient', 'endturn_ui', 'inmap', 'info', 'status', 'command_ui', 'overlay_drawn', 'command'],
     'command'),
)
COPIES = 3


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
    test = open(os.path.join(src_dir, 'rust_golden_r5h_layers_test.go'), encoding='utf-8').read().split('\n')
    ok = True
    starts = [i for i, l in enumerate(test) if '---- 逐字複本開始：main.go:' in l]
    if len(starts) != COPIES:
        print(f'逐字複本標記數 {len(starts)}，應為 {COPIES}')
        return False
    for i in starts:
        m = re.search(r'main\.go:(\d+)\.\.(\d+)', test[i])
        a, b = int(m.group(1)), int(m.group(2))
        end = next(j for j in range(i + 1, len(test)) if f'---- 逐字複本結束：main.go:{a}..{b} ----' in test[j])
        copy, orig = test[i + 1:end], main[a - 1:b]
        same = copy == orig
        ok &= same
        print(f'逐字複本 main.go:{a}..{b}（{len(orig)} 行）：{"與 main.go 逐字相同" if same else "與 main.go 不同"}')
        if not same:
            for k, (x, y) in enumerate(zip(copy, orig)):
                if x != y:
                    print(f'  第 {k} 行不同：\n    複本：{x!r}\n    原檔：{y!r}')
                    break
            if len(copy) != len(orig):
                print(f'  行數不同：複本 {len(copy)}、原檔 {len(orig)}')
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
    print(f"擷取數：Go {go['captures']}（展開 {len(gk)}）、Rust {rs['captures']}（展開 {len(rk)}）")
    print(f"序列雜湊：Go {go['sequence_sha256']}、Rust {rs['sequence_sha256']}"
          f"（{'相同' if go['sequence_sha256'] == rs['sequence_sha256'] else '不同'}）")
    print(f"時鐘：Go {go['clock']}、Rust {rs['clock']}；virtual_frames Go {go['virtual_frames']}、Rust {rs['virtual_frames']}")
    print(f"分支：Go {go['branches']}；Rust {rs['branches']}")
    print(f"統計：Go {go['stats']}；Rust {rs['stats']}")
    n = min(len(gk), len(rk))
    layer = {}
    first = None
    distinct = {}
    for i in range(n):
        gf, rf = fields(gk[i]), fields(rk[i])
        c = layer.setdefault('branch', [0, 0, 0])
        c[0] += 1
        c[1] += gf['branch'] == rf['branch']
        for name, keys, pix in LAYERS:
            if not any(k in gf for k in keys):
                continue
            c = layer.setdefault(name, [0, 0, 0])
            c[0] += 1
            c[1] += all(gf.get(k) == rf.get(k) for k in keys)
            if pix and gf.get(pix) not in (None, '0'):
                c[2] += 1
                distinct.setdefault(name, set()).add(gf[pix])
        if gk[i] != rk[i] and first is None:
            first = i
    for name, (total, same, drawn) in layer.items():
        extra = f"（有畫 {drawn}、Go 相異 {len(distinct.get(name, ()))}）" if name in distinct or drawn else ''
        print(f"  {name}：{total} 張，相同 {same}{extra}")
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
        print(f"階段 0 整張黃金：{a.stage0}（{s0['mode']}，{len(sk)} 張，序列 {s0['sequence_sha256']}）")
        if len(sk) != len(gk):
            print(f"  擷取數與階段 0 不同（{len(sk)} 對 {len(gk)}）")
            ok = False
        stat = {'recovery': [0, 0], 'cursor_hud': [0, 0], 'overlays': [0, 0], 'command': [0, 0]}
        rec_first = None
        for i in range(min(len(gk), len(sk))):
            f = fields(gk[i])
            if f['branch'] == 'recovery':
                stat['recovery'][0] += 1
                same = f['rgba'] == sk[i]
                stat['recovery'][1] += same
                if not same and rec_first is None:
                    rec_first = i
            if f['branch'] == 'tail':
                for name in ('cursor_hud', 'overlays', 'command'):
                    if f.get(name) not in (None, '0'):
                        stat[name][0] += 1
                        stat[name][1] += f[name] == sk[i]
        print('  ' + '、'.join(f'{k} {v[0]} 張中 {v[1]} 張與整張畫格相同' for k, v in stat.items()))
        if stat['recovery'][0]:
            print(f"  recovery 是整個 Draw：{'每一張都與階段 0 相同' if rec_first is None else f'第一個不同在 capture {rec_first}'}")
            if rec_first is not None:
                print(f"    Go 鍵：{gk[rec_first]}；階段 0：{sk[rec_first]}")
            ok &= rec_first is None
        print('  tail 的三層只是本組圖層（畫在全透明底），與整張畫格相同的張數只是參考')
    print('結論：' + ('Go 與 Rust 每個擷取點相同' if ok else '不同'))
    return 0 if ok else 1


if __name__ == '__main__':
    sys.exit(main())
