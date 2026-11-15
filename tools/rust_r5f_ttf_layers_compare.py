#!/usr/bin/env python3
"""R5 階段 2 r5f（docs/goal/121）：TTF 局部黃金的比對。

用法：
  rust_r5f_ttf_layers_compare.py --verbatim <remake/cmd/fd2 目錄>
  rust_r5f_ttf_layers_compare.py --metrics-go <go ttf-metrics.json> --metrics-rust <rust ttf-metrics.json>
  rust_r5f_ttf_layers_compare.py --go <go ttf-layers.json> --rust <rust ttf-layers.json> [--stage0 <frames.json>]

- --verbatim：rust_golden_ttf_layers_test.go 的逐字複本（把 `r.fontDraw(g.font, ` 換回 `g.font.Draw(`）必須與
  main.go 對應行逐字相同。
- --metrics-*：Face.Metrics、每個字元的 GlyphIndex／GlyphBounds（八個尺寸）、非零 kern、每個字串的 BoundString
  逐筆比對；全部相同才算通過。
- --go／--rust：逐擷取點比對鍵（文字、座標、倍率、像素尺寸、ascent、顏色、BoundString 外框）；全部相同才算通過。
  每個不同的繪字結果（renders）另外比畫素：兩邊的裁切 RGBA 逐像素相減，印出不同像素數、最大通道差、覆蓋率與
  墨跡外框的差。畫素差異是 KNOWN_DIFFERENCES `ttf-raster-coverage`，不影響結束碼；但任一邊有墨跡落在
  BoundString 外框外、或同一個鍵兩次畫出不同像素（conflicts），結束碼為 1。
  --stage0：階段 0 chapter frames.json；只數帶 `+ttf-msg` 的擷取數，與本次 TTF 擷取數對照。

結束碼：全部通過為 0，否則 1。不遮蔽任何像素。
"""
import argparse
import base64
import json
import os
import re
import sys


def verbatim(src_dir):
    main = open(os.path.join(src_dir, 'main.go'), encoding='utf-8').read().split('\n')
    test = open(os.path.join(src_dir, 'rust_golden_ttf_layers_test.go'), encoding='utf-8').read().split('\n')
    starts = [i for i, l in enumerate(test) if '---- 逐字複本開始：main.go:' in l]
    if len(starts) != 1:
        print(f'逐字複本標記數 {len(starts)}，應為 1')
        return False
    i = starts[0]
    m = re.search(r'main\.go:(\d+)\.\.(\d+)', test[i])
    a, b = int(m.group(1)), int(m.group(2))
    end = next(j for j in range(i + 1, len(test)) if f'---- 逐字複本結束：main.go:{a}..{b} ----' in test[j])
    copy = [l.replace('r.fontDraw(g.font, ', 'g.font.Draw(') for l in test[i + 1:end]]
    orig = main[a - 1:b]
    same = copy == orig
    subst = sum(l.count('r.fontDraw(g.font, ') for l in test[i + 1:end])
    print(f'逐字複本 main.go:{a}..{b}（{len(orig)} 行，替換 {subst} 處 g.font.Draw）：'
          f'{"與 main.go 逐字相同" if same else "與 main.go 不同"}')
    if not same:
        for k, (x, y) in enumerate(zip(copy, orig)):
            if x != y:
                print(f'  第 {k} 行：複本 {x!r}；main.go {y!r}')
                break
    return same and subst == 3


def metrics(go_path, rs_path):
    go = json.load(open(go_path, encoding='utf-8'))
    rs = json.load(open(rs_path, encoding='utf-8'))
    ok = True
    print(f"字型：Go {go['font']}、Rust {rs['font']}；Rust 字型單位小數座標 {rs['fractional_points']} 個")
    for name in ('sizes', 'runes', 'strings', 'bound_sizes', 'kern_px'):
        if go[name] != rs[name]:
            print(f'輸入清單 {name} 不同')
            return False
    same = go['metrics'] == rs['metrics']
    ok &= same
    print(f"Face.Metrics（{len(go['sizes'])} 個尺寸 {go['sizes']}）：{'相同' if same else '不同'}")
    if not same:
        print(f"  Go {go['metrics']}\n  Rust {rs['metrics']}")
    bad = [r for r in go['glyphs'] if go['glyphs'][r] != rs['glyphs'].get(r)]
    ok &= not bad
    missing = sum(1 for r in go['glyphs'] if go['glyphs'][r][0] == 0)
    print(f"GlyphIndex＋GlyphBounds：{len(go['glyphs'])} 個字元 × {len(go['sizes'])} 尺寸，不同 {len(bad)} 個"
          f"（字型沒有的字 {missing} 個）")
    for r in bad[:5]:
        print(f"  U+{int(r):04X} Go {go['glyphs'][r]}\n          Rust {rs['glyphs'].get(r)}")
    same = go['kern_nonzero'] == rs['kern_nonzero'] and go['kern_pairs'] == rs['kern_pairs']
    ok &= same
    print(f"Kern：{go['kern_pairs']} 對，非零 Go {len(go['kern_nonzero'])}、Rust {len(rs['kern_nonzero'])}："
          f"{'相同' if same else '不同'}")
    if not same:
        keys = sorted(set(go['kern_nonzero']) | set(rs['kern_nonzero']))
        diff = [k for k in keys if go['kern_nonzero'].get(k) != rs['kern_nonzero'].get(k)]
        for k in diff[:5]:
            print(f"  {k}：Go {go['kern_nonzero'].get(k)}、Rust {rs['kern_nonzero'].get(k)}")
    bad = [i for i, (x, y) in enumerate(zip(go['bounds'], rs['bounds'])) if x != y]
    ok &= not bad and len(go['bounds']) == len(rs['bounds'])
    print(f"BoundString：{len(go['strings'])} 個字串 × 尺寸 {go['bound_sizes']}，不同 {len(bad)} 個")
    for i in bad[:5]:
        print(f"  {go['strings'][i]!r}：Go {go['bounds'][i]}、Rust {rs['bounds'][i]}")
    return ok


def expand(rows):
    out = []
    for row in rows:
        if row['capture'] != len(out):
            raise SystemExit(f"rows 的 capture 不連續：{row['capture']} != {len(out)}")
        out.extend([row['key']] * row['count'])
    return out


def pixel_diff(g, r):
    """兩邊裁切 RGBA 逐像素比較；裁切範圍由 BoundString 決定，鍵相同時兩邊相同。"""
    if g['crop'] != r['crop']:
        return None
    a = base64.b64decode(g['crop_rgba_base64'])
    b = base64.b64decode(r['crop_rgba_base64'])
    if len(a) != len(b):
        return None
    diff_px, max_ch, sum_abs = 0, 0, 0
    for i in range(0, len(a), 4):
        d = [abs(a[i + k] - b[i + k]) for k in range(4)]
        m = max(d)
        if m:
            diff_px += 1
            max_ch = max(max_ch, m)
            sum_abs += d[3]
    return diff_px, max_ch, sum_abs, len(a) // 4


def layers(go_path, rs_path, stage0):
    go = json.load(open(go_path, encoding='utf-8'))
    rs = json.load(open(rs_path, encoding='utf-8'))
    gk, rk = expand(go['rows']), expand(rs['rows'])
    print(f"字型：Go {go['font']}、Rust {rs['font']}")
    print(f"擷取數：Go {go['captures']}（展開 {len(gk)}）、Rust {rs['captures']}（展開 {len(rk)}）")
    same_seq = go['sequence_sha256'] == rs['sequence_sha256']
    print(f"序列雜湊：Go {go['sequence_sha256']}、Rust {rs['sequence_sha256']}（{'相同' if same_seq else '不同'}）")
    print(f"時鐘：Go {go['clock']}、Rust {rs['clock']}；virtual_frames Go {go['virtual_frames']}、Rust {rs['virtual_frames']}")
    print(f"分支：Go {go['branches']}；Rust {rs['branches']}")
    print(f"統計：Go {go['stats']}；Rust {rs['stats']}")
    n = min(len(gk), len(rk))
    ttf = [i for i in range(n) if gk[i].startswith('tail|ttf=') and gk[i] != 'tail|ttf=0']
    same_ttf = sum(1 for i in ttf if gk[i] == rk[i])
    first = next((i for i in range(n) if gk[i] != rk[i]), None)
    print(f"TTF 擷取（文字與外框）：Go {len(ttf)} 張，Rust 同一擷取點相同 {same_ttf} 張")
    ok = same_seq and len(gk) == len(rk) and first is None
    if first is not None:
        print(f"第一個不同：第 {first} 次擷取\n  Go   {gk[first]}\n  Rust {rk[first]}")
    if stage0:
        s0 = json.load(open(stage0, encoding='utf-8'))
        count = sum(row['count'] for row in s0['rows'] if '+ttf-msg' in row.get('draw_path', ''))
        print(f"階段 0 chapter 整張黃金帶 +ttf-msg 的擷取：{count} 張（draw_path 依 Draw 後狀態推定，含 Draw 尾段以外的分支；"
              f"本次 Go 尾段實際以 TTF 畫 g.msg {go['stats'].get('ttf_msg', 0)} 張）")
    # 畫素（KNOWN_DIFFERENCES，只量化）
    keys = sorted(go['renders'])
    print(f"不同的繪字結果：Go {len(go['renders'])} 個、Rust {len(rs['renders'])} 個")
    tot = {'diff_px': 0, 'px': 0, 'max_ch': 0, 'sum_abs': 0, 'exact': 0}
    worst = []
    for k in keys:
        g, r = go['renders'][k], rs['renders'].get(k)
        if r is None:
            print(f'  Rust 沒有這個繪字結果：{k[:120]}')
            ok = False
            continue
        for side, rd in (('Go', g), ('Rust', r)):
            if rd['outside_bounds'] or rd['conflicts']:
                print(f"  {side} 墨跡在外框外 {rd['outside_bounds']} 個、同鍵不同像素 {rd['conflicts']} 次：{k[:120]}")
                ok = False
        d = pixel_diff(g, r)
        if d is None:
            print(f'  裁切範圍不同：{k[:120]}')
            ok = False
            continue
        diff_px, max_ch, sum_abs, px = d
        tot['diff_px'] += diff_px
        tot['px'] += px
        tot['sum_abs'] += sum_abs
        tot['max_ch'] = max(tot['max_ch'], max_ch)
        tot['exact'] += diff_px == 0
        ink = [r['ink_box'][i] - g['ink_box'][i] for i in range(4)]
        worst.append((diff_px, max_ch, g['nonzero'], r['nonzero'], g['alpha_sum'], r['alpha_sum'], ink, g['captures'], k))
    print(f"畫素：裁切範圍共 {tot['px']} 個像素，不同 {tot['diff_px']} 個（{100 * tot['diff_px'] / max(tot['px'], 1):.2f}%），"
          f"最大通道差 {tot['max_ch']}，alpha 絕對差總和 {tot['sum_abs']}；完全相同的繪字結果 {tot['exact']}／{len(keys)}")
    ink_shift = [w[6] for w in worst]
    if ink_shift:
        span = [max(abs(s[i]) for s in ink_shift) for i in range(4)]
        print(f"墨跡外框差（Rust−Go，最小 x、最小 y、最大 x、最大 y）的最大絕對值：{span}")
    worst.sort(key=lambda w: -w[0])
    for w in worst[:5]:
        diff_px, max_ch, gn, rn, ga, ra, ink, caps, k = w
        text = re.search(r'text=("(?:[^"\\]|\\.)*")', k)
        print(f"  不同 {diff_px} 像素、最大差 {max_ch}；覆蓋像素 Go {gn}／Rust {rn}；alpha 和 Go {ga}／Rust {ra}；"
              f"墨跡外框差 {ink}；{caps} 次擷取；{text.group(1) if text else k[:60]}")
    return ok


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--verbatim')
    ap.add_argument('--metrics-go')
    ap.add_argument('--metrics-rust')
    ap.add_argument('--go')
    ap.add_argument('--rust')
    ap.add_argument('--stage0')
    a = ap.parse_args()
    if a.verbatim:
        return 0 if verbatim(a.verbatim) else 1
    if a.metrics_go:
        return 0 if metrics(a.metrics_go, a.metrics_rust) else 1
    return 0 if layers(a.go, a.rust, a.stage0) else 1


if __name__ == '__main__':
    sys.exit(main())
