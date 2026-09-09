"""整理本機 IDA 原始匯出為可審查的最小 EXP 證據，不匯入未分級推論。"""
import hashlib
import json
from pathlib import Path
import sys

assert Path('/.dockerenv').exists(), '只能在 Docker 中執行'
root, destination = map(Path, sys.argv[1:3])
selections = [
    ('physical-exp.json', 0x29F72, 0x2A205, 0x2A27A, '普通物理攻擊 EXP 來源與兩階段整數除法'),
    ('physical-exp.json', 0x117E7, 0x11959, 0x11972, '玩家交易的 99 上限及 JOIN 以外 EXP 消費端'),
    ('exp-functions.json', 0x1E292, 0x1E309, 0x1E318, 'unit +0x3C byte 與整數交易累加器相加'),
    ('exp-functions.json', 0x1E292, 0x1E513, 0x1E524, 'EXP 寫回 byte 並清除交易累加器'),
]
out = {'schema': 1, 'date': '2026-09-08', 'tool': 'IDA Pro 9.4',
       'address_space': 'IDA DOS LE loader linear', 'input': {
           'name': 'FD2.EXE', 'size': 357074,
           'md5': 'b97caf2239a27a896069d03549d96e1e',
           'sha256': '222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f'},
       'scope': '只閉合普通物理 EXP；不外推傷害、成長、法術或整場原版一致性',
       'sources': {}, 'slices': []}
for name, address, start, end, semantic in selections:
    p = root / name
    data = json.loads(p.read_text())
    assert data['input']['sha256'] == out['input']['sha256']
    fn = next(f for f in data['functions'] if int(f['address'], 16) == address)
    rows = [dict(r) for r in fn['instructions'] if start <= int(r['address'], 16) <= end]
    assert rows
    for r in rows:
        r['grade'] = '已證實'
        r['semantic_scope'] = semantic
        r['source'] = '94-ch01-town-parity-20260908.md 的直接指令與來源／消費端審查；' + name
    out['sources'][name] = {'sha256': hashlib.sha256(p.read_bytes()).hexdigest(),
                            'local_path': 'work/ch01-town-parity-20260908/ida/' + name}
    out['slices'].append({'original_name': fn['original_name'], 'address': fn['address'],
                          'grade': '已證實', 'semantic_scope': semantic,
                          'xrefs': fn['xrefs'], 'instructions': rows})
assert destination.parent.stat().st_uid == __import__('os').getuid()
destination.write_text(json.dumps(out, ensure_ascii=False, indent=2) + '\n')
