#!/usr/bin/env python3
"""R5 階段 1（docs/goal/121）：把 tools/rust_r4_skeleton -r5 的 r5-plan.json 套進 fd2-game。

用法：rust_r5_skeleton_apply.py <r5-plan.json> <fd2-game crate 目錄>

規則（重跑結果相同，第二次不再改動任何檔）：
- PORTING.json 的狀態：已是 `ported` 的不動；HD 檔的 `r8` 改 `r6`（只改狀態，不產生骨架）；非 HD 的改 `r5`。
- 例外（更正錯誤的 `ported`）：crate 裡沒有該函式（R4 狀態工具以名稱比對，`Font.Draw` 被當成 `Game.Draw`）的，
  非 HD 改 `r5` 並補骨架；本體去掉註解後只有一個 `todo!("R5/R6/R8: …")` 的，依標記改狀態（R4 階段 2 的規則）。
  更正逐筆記在 PORTING.json 的 `r5_stage1.corrections`；本體只有部分分支是 `todo!("R5…")` 的仍算 ported，
  列在 `r5_stage1.ported_with_r5_todo_branch`。
- 非 HD、狀態是 `r5`、且對應 Rust 檔還沒有 `Go：\\`<檔>:<行>\\`` 標記的函式，把產生器給的簽名骨架
  （本體 `todo!("r5:<檔>.go:<函式名>")`）附加到該 Go 檔的 Rust 子模組末尾。
- PORTING.json 以 `json.dumps(indent=2, ensure_ascii=False)` 寫回；動手前先確認讀進來再寫出與原檔逐位元組相同，
  不同就停下（表示檔案不是這個格式，不能安全改寫）。
"""
import json
import os
import re
import sys
from collections import OrderedDict

SECTION = '// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----'
TODO_RE = re.compile(r'^todo!\(\s*"([Rr][568])[:：][^"]*"\s*,?\s*\)\s*;?$')


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


def code_lines(body):
    return [l.strip() for l in body.split('\n') if l.strip() and not l.strip().startswith('//')]


def has_r5_todo(body):
    return re.search(r'todo!\(\s*"[Rr]5', body) is not None


def whole_body_todo(body):
    """本體去掉註解與 `let _ = …;`（只為了消掉未使用參數）後只剩一個 todo!("R5/R6/R8: …")：
    回傳 'r5'／'r6'／'r8'，否則 None。"""
    lines = [l for l in code_lines(body) if not re.match(r'^let _ = [^;]*;$', l)]
    joined = ' '.join(lines).replace('( "', '("')
    m = TODO_RE.match(joined)
    return m.group(1).lower() if m else None


def main():
    if len(sys.argv) != 3:
        print(__doc__, file=sys.stderr)
        return 2
    plan_path, crate = sys.argv[1:]
    plan = json.load(open(plan_path, encoding='utf-8'))
    if plan.get('schema') != 'fd2_rust_r5_skeleton_plan/1':
        raise SystemExit('r5-plan.json 的 schema 不對')
    porting_path = os.path.join(crate, 'PORTING.json')
    raw = open(porting_path, encoding='utf-8').read()
    porting = json.loads(raw, object_pairs_hook=OrderedDict)
    if json.dumps(porting, ensure_ascii=False, indent=2) + '\n' != raw:
        raise SystemExit('PORTING.json 不是 indent=2、ensure_ascii=False 的格式，停止改寫')

    by_line = {}
    rust_of = {}
    for f in porting['files']:
        base = f['go'].rsplit('/', 1)[-1]
        rust_of[base] = f['rust']
        for fn in f['functions']:
            by_line[(base, fn['go_line'])] = fn

    changes = OrderedDict([('r8_to_r5', 0), ('hd_r8_to_r6', 0), ('ported_corrected', 0), ('stubs_added', 0)])
    prev = porting.get('r5_stage1', {})
    corrections = OrderedDict((c['go'], c) for c in prev.get('corrections', []))
    stubs = OrderedDict()  # Rust 檔 → [(impl, code)]
    entries = []
    for e in plan['functions']:
        fn = by_line.get((e['file'], e['go_line']))
        if fn is None or fn['name'] != e['name']:
            raise SystemExit(f"PORTING.json 找不到 {e['file']}:{e['go_line']} {e['name']}")
        rel = rust_of.get(e['file'])
        path = os.path.join(crate, rel.removeprefix('remake-rs/crates/fd2-game/')) if rel else None
        entries.append((e, fn, path))
        marker = f"Go：`{e['file']}:{e['go_line']}`"
        text = open(path, encoding='utf-8').read() if path else ''
        if fn['status'] == 'ported':
            body = rust_body(text, marker)
            want = why = None
            if body is None:
                want, why = ('r6' if e['hd'] else 'r5'), 'PORTING.json 寫 ported，crate 裡卻沒有這個函式'
            else:
                kind = whole_body_todo(body)
                if kind:
                    want, why = kind, f'本體只有 todo!("{kind.upper()}: …")（R4 階段 2 的狀態工具把註解算成本體）'
            if want is None:
                continue
            key = f"{e['file']}:{e['go_line']}"
            corrections[key] = OrderedDict([('go', key), ('func', e['func']), ('from', 'ported'), ('to', want), ('reason', why)])
            fn['status'] = want
            changes['ported_corrected'] += 1
            if body is not None or e['hd']:
                continue
        elif e['hd']:
            if fn['status'] == 'r8':
                fn['status'] = 'r6'
                changes['hd_r8_to_r6'] += 1
            continue
        elif fn['status'] != 'r5':
            fn['status'] = 'r5'
            changes['r8_to_r5'] += 1
        if path is None:
            raise SystemExit(f"{e['file']} 沒有 Rust 子模組，需要先建立")
        if marker in text:
            continue
        if not e.get('code'):
            raise SystemExit(f"{e['file']}:{e['go_line']} 沒有骨架原始碼")
        stubs.setdefault(path, []).append((e.get('impl', ''), e['code']))
        changes['stubs_added'] += 1

    for path, items in stubs.items():
        text = open(path, encoding='utf-8').read()
        parts = ['', SECTION, '']
        impls = OrderedDict()
        frees = []
        for impl, code in items:
            if impl:
                impls.setdefault(impl, []).append(code)
            else:
                frees.append(code)
        for impl, codes in impls.items():
            parts.append(f'impl {impl} {{')
            parts.append('\n\n'.join(c.rstrip('\n') for c in codes))
            parts.append('}')
            parts.append('')
        for code in frees:
            parts.append(code.rstrip('\n'))
            parts.append('')
        if SECTION in text:
            raise SystemExit(f'{path} 已有 R5 骨架區段卻缺函式，請人工檢查')
        with open(path, 'w', encoding='utf-8') as f:
            f.write(text.rstrip('\n') + '\n' + '\n'.join(parts).rstrip('\n') + '\n')

    status = OrderedDict()
    for f in porting['files']:
        for fn in f['functions']:
            status[fn['status']] = status.get(fn['status'], 0) + 1
    totals = porting['totals']
    totals['functions_by_status'] = OrderedDict(sorted(status.items()))
    note = 'R5 階段 1：開場與第一章執行到的非 HD 繪圖函式標 r5、HD 檔的 r8 改 r6（tools/rust_r5_skeleton.sh）'
    if note not in totals.get('status_updated', ''):
        totals['status_updated'] = totals.get('status_updated', '') + '；' + note
    # 結果統計只看套用後的狀態，重跑時相同。
    result = OrderedDict([('inventory', 0), ('non_hd', 0), ('non_hd_ported', 0), ('non_hd_r5', 0),
                          ('r5_stubs', 0), ('hd', 0), ('hd_ported', 0), ('hd_r6', 0)])
    partial = []
    for e, fn, path in entries:
        result['inventory'] += 1
        text = open(path, encoding='utf-8').read() if path else ''
        if fn['status'] == 'ported':
            body = rust_body(text, f"Go：`{e['file']}:{e['go_line']}`") or ''
            if has_r5_todo('\n'.join(code_lines(body))):
                partial.append(f"{e['file']}:{e['go_line']} {e['func']}")
        if e['hd']:
            result['hd'] += 1
            result['hd_ported'] += fn['status'] == 'ported'
            result['hd_r6'] += fn['status'] == 'r6'
            continue
        result['non_hd'] += 1
        result['non_hd_ported'] += fn['status'] == 'ported'
        result['non_hd_r5'] += fn['status'] == 'r5'
        code = e.get('code', '')
        at = code.find('todo!("r5:')
        if fn['status'] == 'r5' and at >= 0:
            todo = code[at:code.index(')', at) + 1]
            result['r5_stubs'] += todo in text
    porting['r5_stage1'] = OrderedDict([
        ('inventory', plan['inventory']),
        ('rule', '已 ported 的不動；非 HD 標 r5，沒有 Rust 對應的補 todo!("r5:<檔>.go:<函式名>") 骨架；HD 檔 r8 改 r6。'
                 'ported 但 crate 沒有該函式、或本體只有 todo!("R5/R6/R8: …") 的，依 R4 階段 2 的規則更正（列在 corrections）'),
        ('result', result),
        ('corrections', [corrections[k] for k in sorted(corrections)]),
        ('ported_with_r5_todo_branch', partial),
    ])
    with open(porting_path, 'w', encoding='utf-8') as f:
        f.write(json.dumps(porting, ensure_ascii=False, indent=2) + '\n')
    print('R5 本次改動：' + '、'.join(f'{k}={v}' for k, v in changes.items()))
    print('R5 結果：' + '、'.join(f'{k}={v}' for k, v in result.items()))
    print('R5 更正：' + '；'.join(f"{c['go']} {c['from']}→{c['to']}" for c in corrections.values()))
    print('狀態：' + '、'.join(f'{k}={v}' for k, v in status.items()))
    return 0


if __name__ == '__main__':
    sys.exit(main())
