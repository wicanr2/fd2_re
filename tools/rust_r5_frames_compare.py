#!/usr/bin/env python3
"""R5 階段 2（docs/goal/121）：逐次比對 Rust 與 Go 黃金的 frames.json（rows 以 capture／count 去重）。

用法：rust_r5_frames_compare.py --rust <frames.json> --go <frames.json> [--path <draw_path>] [--out <compare.json>]

- 兩邊各自展開成每次擷取的 RGBA SHA-256，比對 Rust 實際擷取到的範圍（Rust 停在未移植函式時 `stopped`
  記下停點，之後不比）。比的是 SHA-256，相同即 640×400×4 個位元組都相同；不遮蔽任何像素。
- `--path`：另外要求 Go 黃金裡 `draw_path` 等於該值的每一次擷取都在 Rust 範圍內且相同（局部黃金的通過條件）。
- 前綴序列雜湊：前 N 次擷取的 SHA-256 串接後再取 SHA-256（與 frames.json 的 sequence_sha256 同算法），兩邊各算一次。
- 全部相同且 `--path` 的擷取都涵蓋時結束碼 0，否則 1。
"""
import argparse
import hashlib
import json
import sys


def expand(doc):
    hashes, rows = [], []
    for row in doc['rows']:
        if row['capture'] != len(hashes):
            raise SystemExit(f"rows 不連續：capture={row['capture']}，預期 {len(hashes)}")
        for _ in range(row['count']):
            hashes.append(row['rgba_sha256'])
            rows.append(row)
    if len(hashes) != doc['captures']:
        raise SystemExit(f"rows 展開 {len(hashes)} 次，captures={doc['captures']}")
    return hashes, rows


def prefix_sha(hashes, n):
    h = hashlib.sha256()
    for x in hashes[:n]:
        h.update(bytes.fromhex(x))
    return h.hexdigest()


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--rust', required=True)
    ap.add_argument('--go', required=True)
    ap.add_argument('--path')
    ap.add_argument('--out')
    args = ap.parse_args()
    raw_go = open(args.go, 'rb').read()
    rust, go = json.load(open(args.rust, encoding='utf-8')), json.loads(raw_go)
    if rust.get('mode') != go.get('mode'):
        raise SystemExit(f"模式不同：rust={rust.get('mode')} go={go.get('mode')}")
    rh, _ = expand(rust)
    gh, grows = expand(go)
    n = min(len(rh), len(gh))
    equal = sum(1 for i in range(n) if rh[i] == gh[i])
    first = next((i for i in range(n) if rh[i] != gh[i]), None)
    report = {
        'mode': go['mode'],
        'go_frames_json_sha256': hashlib.sha256(raw_go).hexdigest(),
        'go_sequence_sha256': go['sequence_sha256'],
        'go_captures': len(gh),
        'rust_captures': len(rh),
        'rust_stopped': rust.get('stopped'),
        'compared': n,
        'equal': equal,
        'prefix_sequence_sha256': {'rust': prefix_sha(rh, n), 'go': prefix_sha(gh, n)},
        'first_mismatch': None,
    }
    if first is not None:
        row = grows[first]
        report['first_mismatch'] = {'capture': first, 'rust': rh[first], 'go': gh[first], 'go_row_file': row.get('file'),
                                    'go_draw_path': row.get('draw_path'), 'go_title_phase': row.get('title_phase', '')}
    ok = first is None
    if args.path:
        idx = [i for i, r in enumerate(grows) if r.get('draw_path') == args.path]
        covered = [i for i in idx if i < n]
        same = [i for i in covered if rh[i] == gh[i]]
        report['path'] = {'draw_path': args.path, 'go_captures': len(idx), 'covered': len(covered), 'equal': len(same),
                          'range': [idx[0], idx[-1]] if idx else None,
                          'contiguous': bool(idx) and idx == list(range(idx[0], idx[-1] + 1))}
        ok = ok and len(same) == len(idx)
    report['ok'] = ok
    text = json.dumps(report, ensure_ascii=False, indent=1)
    if args.out:
        with open(args.out, 'w', encoding='utf-8') as f:
            f.write(text + '\n')
    print(text)
    return 0 if ok else 1


if __name__ == '__main__':
    sys.exit(main())
