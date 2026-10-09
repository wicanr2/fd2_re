#!/usr/bin/env python3
"""在唯讀原版的 IDA 9.4 資料庫複本核對 SDK 完整函式及直接呼叫者。"""
import hashlib
import json
import os
from pathlib import Path
import ida_auto
import ida_bytes
import ida_funcs
import ida_kernwin
import ida_loader
import ida_nalt
import ida_pro
import idautils
import idc

INPUT_SHA = '222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f'


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def instruction(ea, original):
    size = ida_bytes.get_item_size(ea)
    offset = ida_loader.get_fileregion_offset(ea)
    loaded = ida_bytes.get_bytes(ea, size)
    if offset < 0 or offset + size > len(original) or loaded is None:
        raise ValueError('指令無法對應完整原始檔位元組：' + hex(ea))
    return {'ida_linear_address': hex(ea), 'file_offset': hex(offset),
            'loaded_bytes': loaded.hex(), 'file_bytes': original[offset:offset+size].hex(),
            'loaded_equals_file': loaded == original[offset:offset+size],
            'instruction': idc.generate_disasm_line(ea, 0)}


def main():
    if not Path('/.dockerenv').exists() or ida_kernwin.get_kernel_version() != '9.4':
        raise ValueError('只允許 Docker 的 IDA 9.4')
    ida_auto.auto_wait()
    original_path = Path('/input/FD2.EXE')
    if sha(original_path) != INPUT_SHA or sha(Path(ida_nalt.get_input_file_path())) != INPUT_SHA:
        raise ValueError('原版或資料庫輸入不符')
    original = original_path.read_bytes()
    probe_path = Path('/source/library-probe.json')
    probe = json.loads(probe_path.read_text(encoding='utf-8'))
    if probe['input']['sha256'] != INPUT_SHA or probe['representation'] != 'omf':
        raise ValueError('來源比對輸入不符')
    rows = []
    for match in probe['matches']:
        start = int(match['ida_linear_address'], 16)
        function = ida_funcs.get_func(start)
        if not function or function.start_ea != start or list(idautils.Chunks(start)) != [(start, function.end_ea)]:
            raise ValueError('非完整連續的 IDA 函式')
        code = ida_bytes.get_bytes(start, function.end_ea - start)
        if len(code) != match['size'] or hashlib.sha256(code).hexdigest() != match['code_sha256']:
            raise ValueError('SDK 與資料庫完整函式不符')
        if idc.get_func_name(start) != match['original_name']:
            raise ValueError('原始定位名稱漂移')
        offset = ida_loader.get_fileregion_offset(start)
        if offset < 0 or original[offset:offset + len(code)] != code:
            raise ValueError('函式包含未處理的載入重定位或不相同原始位元組')
        calls = []
        for reference in idautils.CodeRefsTo(start, False):
            caller = ida_funcs.get_func(reference)
            if caller is None or idc.print_insn_mnem(reference).lower() != 'call':
                continue
            insn = instruction(reference, original)
            raw = bytes.fromhex(insn['file_bytes'])
            if not insn['loaded_equals_file'] or len(raw) != 5 or raw[0] != 0xe8 or reference + 5 + int.from_bytes(raw[1:], 'little', signed=True) != start:
                raise ValueError('直接 call 目標與原始 E8 位元組不符')
            calls.append((caller.start_ea, reference))
        if sorted({hex(c) for c, _ in calls}) != sorted(match['caller_functions']):
            raise ValueError('原始 caller 清冊與目前 IDA 直接 call 不符')
        windows = []
        seen = set()
        for caller, call in sorted(calls):
            if caller in seen or len(seen) >= 3:
                continue
            seen.add(caller)
            items = list(idautils.FuncItems(caller))
            n = items.index(call)
            windows.append({'caller_start': hex(caller), 'original_caller_name': idc.get_func_name(caller),
                            'call_address': hex(call), 'instructions': [instruction(ea, original) for ea in items[max(0, n-3):n+4]],
                            'argument_and_return_semantics': 'unknown'})
        rows.append({**match, 'file_offset': hex(offset), 'ida_function_end': hex(function.end_ea),
                     'ida_library_flag': bool(function.flags & ida_funcs.FUNC_LIB),
                     'direct_call_count': len(calls), 'caller_windows': windows,
                     'module_classification': '已證實：固定 SDK 的完整無重定位 code 相同；既有 runtime 分類保留'})
    output = Path('/out/library-ida-review.json')
    if output.parent.stat().st_uid != os.getuid():
        raise ValueError('輸出擁有權不符')
    result = {'schema_version': 1, 'input': probe['input'], 'tool': {'name': 'IDA Pro', 'version': '9.4', 'address_space': 'IDA database linear address'},
              'probe_sha256': sha(probe_path), 'library': probe['library'], 'driver_sha256': sha(Path(__file__)),
              'matches': rows, 'classification_changed': False, 'function_semantics': 'unknown',
              'uid': os.getuid(), 'gid': os.getgid()}
    output.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')


try:
    main()
except Exception as error:
    Path('/out/library-ida-error.txt').write_text(type(error).__name__ + ': ' + str(error)[:500] + '\n', encoding='utf-8')
    ida_pro.qexit(1)
else:
    ida_pro.qexit(0)
