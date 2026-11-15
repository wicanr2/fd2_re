#!/usr/bin/env python3
"""R5 階段 2（docs/goal/121）：把移植完成的 r5 函式在 fd2-game 的 PORTING.json 改成 ported。

用法：rust_r5_mark_ported.py [--stage4] <fd2-game crate 目錄> <組名> <Go 檔:行> [<Go 檔:行> ...]

例：rust_r5_mark_ported.py remake-rs/crates/fd2-game r5t title.go:45 title.go:456 title.go:486 title.go:716

規則（重跑結果相同，第二次不改任何檔）：
- 每個 `<Go 檔:行>` 必須是 PORTING.json 裡狀態 `r5`（或已是 `ported`）的函式；對應 Rust 檔要有
  `Go：\\`<檔>:<行>\\`` 標記，標記後第一個函式的本體不可再有 `todo!("r5`（`todo!("r6…")`／`todo!("r8…")` 的
  未執行分支可以留，逐筆列在紀錄的 `todo_branches`）。條件不成立就停下，不改任何檔。
- 狀態改 `ported`，重算 `totals.functions_by_status`；組別紀錄寫在頂層 `r5_stage2_<組名>`（各組分開，
  主代理合併時不互相覆蓋）。
- PORTING.json 以 `json.dumps(indent=2, ensure_ascii=False)` 寫回；動手前先確認讀進來再寫出與原檔逐位元組相同。
- `--stage4`（R5 階段 4 r5w 加）：也接受狀態 `r8` 的函式（說明畫面等外殼函式在階段 0 盤點時沒有執行到）；Go 檔還沒有
  Rust 子模組（`rust` 是 null）時寫成 `remake-rs/crates/fd2-game/src/game/<Go 檔名>.rs`，該檔要存在；紀錄寫在
  `r5_stage4_<組名>`。不加旗標時行為不變。
"""
import json
import os
import re
import sys
from collections import OrderedDict


def rust_body(text, marker):
    """`Go：`<檔>:<行>`` 標記之後第一個函式的本體（大括號內文字）；沒有標記回 None。"""
    i = text.find(marker)
    if i < 0:
        return None
    k = text.find('{', text.find(' fn ', i))
    depth = 0
    for p in range(k, len(text)):
        if text[p] == '{':
            depth += 1
        elif text[p] == '}':
            depth -= 1
            if depth == 0:
                return text[k + 1:p]
    raise SystemExit(f'{marker} 的函式本體大括號不成對')


def code_only(body):
    return '\n'.join(l for l in body.split('\n') if not l.strip().startswith('//'))


def main():
    args = sys.argv[1:]
    stage4 = bool(args) and args[0] == '--stage4'
    if stage4:
        args = args[1:]
    if len(args) < 3:
        print(__doc__, file=sys.stderr)
        return 2
    crate, group, targets = args[0], args[1], args[2:]
    allowed = ('r5', 'r8', 'ported') if stage4 else ('r5', 'ported')
    if not re.fullmatch(r'[a-z0-9]+', group):
        raise SystemExit(f'組名 {group!r} 只能是小寫英數')
    porting_path = os.path.join(crate, 'PORTING.json')
    raw = open(porting_path, encoding='utf-8').read()
    porting = json.loads(raw, object_pairs_hook=OrderedDict)
    if json.dumps(porting, ensure_ascii=False, indent=2) + '\n' != raw:
        raise SystemExit('PORTING.json 不是 indent=2、ensure_ascii=False 的格式，停止改寫')
    by_line = {}
    for f in porting['files']:
        base = f['go'].rsplit('/', 1)[-1]
        for fn in f['functions']:
            by_line[(base, fn['go_line'])] = (f, fn)
    records = []
    for target in targets:
        base, _, line = target.partition(':')
        hit = by_line.get((base, int(line)))
        if hit is None:
            raise SystemExit(f'PORTING.json 找不到 {target}')
        f, fn = hit
        if fn['status'] not in allowed:
            raise SystemExit(f"{target} {fn['name']} 狀態是 {fn['status']}，不是 {'／'.join(allowed[:-1])}")
        if f['rust'] is None and stage4:
            f['rust'] = 'remake-rs/crates/fd2-game/src/game/' + base.removesuffix('.go') + '.rs'
        path = os.path.join(crate, f['rust'].removeprefix('remake-rs/crates/fd2-game/'))
        if not os.path.isfile(path):
            raise SystemExit(f'{target} 的 Rust 檔 {path} 不存在')
        body = rust_body(open(path, encoding='utf-8').read(), f'Go：`{target}`')
        if body is None:
            raise SystemExit(f'{path} 沒有 Go：`{target}` 標記')
        code = code_only(body)
        if re.search(r'todo!\(\s*"[Rr]5', code):
            raise SystemExit(f"{target} {fn['name']} 的本體還有 todo!(\"r5…\")")
        branches = re.findall(r'todo!\(\s*"([Rr][68][:：][^"]*)"', code)
        fn['status'] = 'ported'
        records.append(OrderedDict([('go', target), ('func', fn['name']), ('rust', fn['rust']),
                                    ('todo_branches', branches)]))
    status = OrderedDict()
    for f in porting['files']:
        for fn in f['functions']:
            status[fn['status']] = status.get(fn['status'], 0) + 1
    totals = porting['totals']
    totals['functions_by_status'] = OrderedDict(sorted(status.items()))
    note = 'R5 階段 2：移植完成的 r5 函式改 ported（tools/rust_r5_mark_ported.py）'
    if stage4:
        note = 'R5 階段 4：外殼移植的函式改 ported（tools/rust_r5_mark_ported.py --stage4）'
    if note not in totals.get('status_updated', ''):
        totals['status_updated'] = totals.get('status_updated', '') + '；' + note
    key = f"r5_stage{4 if stage4 else 2}_{group}"
    prev = porting.get(key, OrderedDict())
    merged = OrderedDict((r['go'], r) for r in prev.get('functions', []))
    for r in records:
        merged[r['go']] = r
    porting[key] = OrderedDict([
        ('rule', '本體不再有 todo!("r5…") 才改 ported；未執行分支的 todo!("r6…")／todo!("r8…") 列在 todo_branches'
                 + ('；--stage4 也接受 r8' if stage4 else '')),
        ('functions', [merged[k] for k in sorted(merged, key=lambda s: (s.split(':')[0], int(s.split(':')[1])))]),
    ])
    with open(porting_path, 'w', encoding='utf-8') as out:
        out.write(json.dumps(porting, ensure_ascii=False, indent=2) + '\n')
    print(f'{key}：' + '、'.join(f"{r['go']} {r['func']}" for r in records))
    print('狀態：' + '、'.join(f'{k}={v}' for k, v in status.items()))
    return 0


if __name__ == '__main__':
    sys.exit(main())
