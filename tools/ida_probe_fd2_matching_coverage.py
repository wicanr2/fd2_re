"""非破壞性核對匹配清冊、IDA 函式及指定區間的原始指令。"""

import hashlib
import json
import os
from pathlib import Path

import ida_auto
import ida_bytes
import ida_funcs
import ida_loader
import ida_segment
import idaapi
import idautils
import idc


def main():
    repo = Path('/repo')
    raw = Path('/input/FD2.EXE').read_bytes()
    identity = {'file': 'FD2.EXE', 'size': len(raw), 'md5': hashlib.md5(raw).hexdigest(),
                'sha256': hashlib.sha256(raw).hexdigest()}
    reference = next(x for x in json.loads((repo / 'docs/data/fd2-reference-files.json').read_text(encoding='utf-8'))['files'] if x['file'] == 'FD2.EXE')
    if identity != reference:
        raise ValueError('原始輸入雜湊不符')
    inventory = json.loads((repo / 'docs/data/ida/fd2_function_inventory.json').read_text(encoding='utf-8'))
    if inventory['input'] != identity:
        raise ValueError('清冊版本不符')
    ida_auto.auto_wait()
    known = {int(f['start'], 16) for f in inventory['functions']}
    actual = list(idautils.Functions())
    missing = []
    for address in actual:
        if address not in known:
            fn = ida_funcs.get_func(address)
            missing.append({'ida_linear_start': hex(address), 'ida_linear_end': hex(fn.end_ea),
                            'original_ida_name': idc.get_func_name(address), 'confidence': '未知'})
    unowned = []
    code_bytes = 0
    for segment_start in idautils.Segments():
        segment = ida_segment.getseg(segment_start)
        if segment.type != ida_segment.SEG_CODE:
            continue
        for ea in idautils.Heads(segment.start_ea, segment.end_ea):
            if not ida_bytes.is_code(ida_bytes.get_full_flags(ea)):
                continue
            size = idc.get_item_size(ea)
            code_bytes += size
            if ida_funcs.get_func(ea) is not None:
                continue
            if unowned and unowned[-1]['end'] == ea:
                unowned[-1]['end'] += size
                unowned[-1]['instruction_count'] += 1
            else:
                unowned.append({'start': ea, 'end': ea + size, 'instruction_count': 1})
    unowned_ranges = [{'ida_linear_start': hex(r['start']), 'ida_linear_end': hex(r['end']),
                       'size': r['end'] - r['start'], 'instruction_count': r['instruction_count'],
                       'semantic_confidence': '未知'} for r in unowned]
    start = int(os.environ.get('FD2_MATCHING_AUDIT_START', '0x212b9'), 0)
    end = int(os.environ.get('FD2_MATCHING_AUDIT_END', '0x2134b'), 0)
    instructions = []
    for ea in idautils.Heads(start, end):
        size = idc.get_item_size(ea)
        offset = ida_loader.get_fileregion_offset(ea)
        if offset < 0 or offset + size > len(raw):
            raise ValueError('區間沒有原始檔映射')
        fn = ida_funcs.get_func(ea)
        instructions.append({'ida_linear_address': hex(ea), 'file_offset': hex(offset),
            'loaded_bytes': ida_bytes.get_bytes(ea, size).hex(), 'file_bytes': raw[offset:offset + size].hex(),
            'is_code': ida_bytes.is_code(ida_bytes.get_full_flags(ea)),
            'instruction': idc.generate_disasm_line(ea, 0),
            'ida_owner_start': hex(fn.start_ea) if fn else None,
            'original_ida_name': idc.get_name(ea),
            'incoming_xrefs': [{'ida_linear_from': hex(x.frm), 'type': x.type, 'is_code': bool(x.iscode)} for x in idautils.XrefsTo(ea)]})
    result = {'schema_version': 1, 'input': identity,
        'tool': {'name': 'IDA Pro', 'version': idaapi.get_kernel_version(),
                 'address_space': 'IDA LE loader linear address; file offsets separate'},
        'script_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        'input_database_sha256': hashlib.sha256(Path('/database/ida-full-image.i64').read_bytes()).hexdigest(),
        'inventory_sha256': hashlib.sha256((repo / 'docs/data/ida/fd2_function_inventory.json').read_bytes()).hexdigest(),
        'inventory_function_count': len(known), 'ida_function_count': len(actual),
        'inventory_missing_ida_functions': missing,
        'ida_code_bytes': code_bytes, 'unowned_code_ranges': unowned_ranges,
        'unowned_code_bytes': sum(r['size'] for r in unowned_ranges),
        'interval': {'ida_linear_start': hex(start), 'ida_linear_end': hex(end), 'instructions': instructions},
        'policy': '不改名、不改型別、不修函式邊界；語意與來源分類維持未知'}
    Path(os.environ['FD2_MATCHING_AUDIT_OUTPUT']).write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    idaapi.qexit(0)


main()
