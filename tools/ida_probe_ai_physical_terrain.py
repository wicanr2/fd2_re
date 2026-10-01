"""以唯讀原版與既有 IDA 資料庫匯出物理候選地形及低分比較分支。"""
import hashlib
import json
import os

import ida_auto
import ida_bytes
import ida_funcs
import ida_xref
import idaapi
import idautils
import idc

ida_auto.auto_wait()
raw = open(os.environ.get('FD2_IDA_INPUT', '/input/FD2.EXE'), 'rb').read()
assert len(raw) == 357074
assert hashlib.sha256(raw).hexdigest() == '222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f'
result = {'input': 'FD2.EXE', 'size': len(raw), 'md5': hashlib.md5(raw).hexdigest(),
          'sha256': hashlib.sha256(raw).hexdigest(), 'tool': 'IDA Pro ' + idaapi.get_kernel_version(),
          'address_space': 'IDA LE linear', 'functions': []}
canonical_path = '/src/docs/data/ida/fd2_ai_physical_target_terrain_20261001.json'
with open(canonical_path, encoding='utf-8') as source:
    canonical = json.load(source)
assert canonical['sha256'] == result['sha256']


def reviewed_annotations(ea):
    annotations = []
    for claim in canonical['claims']:
        assert claim['level'] in ('已證實', '強推論', '假說', '未知')
        for location in claim['original']:
            bounds = [int(part, 16) for part in location.split('..')]
            if bounds[0] <= ea <= bounds[-1]:
                annotations.append({'semantic': claim['semantic'], 'level': claim['level'],
                                    'source': [canonical_path.replace('/src/', '')] + claim['source']})
                break
    return annotations


for address in (0x14237, 0x1F183):
    fn = ida_funcs.get_func(address)
    assert fn is not None and fn.start_ea == address
    rows = []
    for ea in idautils.FuncItems(address):
        if not ida_bytes.is_code(ida_bytes.get_full_flags(ea)):
            continue
        annotations = reviewed_annotations(ea)
        level = '已證實' if annotations and all(a['level'] == '已證實' for a in annotations) else '未知'
        rows.append({'address': hex(ea), 'bytes': ida_bytes.get_bytes(ea, idc.get_item_size(ea)).hex(),
                     'original': idc.generate_disasm_line(ea, 0), 'semantic': '',
                     'level': level, 'warning': '' if level == '已證實' else '未審查語意；原始指令僅作定位',
                     'source': ['tools/ida_probe_ai_physical_terrain.py'], 'semantic_annotations': annotations})
    result['functions'].append({'address': hex(address), 'end': hex(fn.end_ea),
                                'original_name': idc.get_func_name(address), 'instructions': rows,
                                'semantic': '', 'level': '未知', 'warning': '本匯出只閉合列出的 caller-specific claims',
                                'source': ['tools/ida_probe_ai_physical_terrain.py'],
                                'callers': [{'address': hex(x.frm), 'original': idc.generate_disasm_line(x.frm, 0)}
                                            for x in idautils.XrefsTo(address) if x.iscode]})
with open(os.environ['FD2_IDA_OUTPUT'], 'w', encoding='utf-8') as out:
    json.dump(result, out, ensure_ascii=False, indent=2)
    out.write('\n')
idc.qexit(0)
