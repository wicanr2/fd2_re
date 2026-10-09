#!/usr/bin/env python3
"""由固定 SDK 物件重建已分類 runtime 的完整函式，另列函式庫出處。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
from fd2_matching_library_probe import INPUT_SHA, LIBRARIES, omf_sections, sha


def plan(library, probe, review, evidence, original, baseline, receipt):
    library_bytes = library.read_bytes()
    p = json.loads(probe.read_text())
    r = json.loads(review.read_text())
    e = json.loads(evidence.read_text())
    b = json.loads(receipt.read_text())
    library_sha, _ = LIBRARIES[p['library']['source_package']]
    if sha(library) != library_sha or p['library']['sha256'] != library_sha or r['library'] != p['library']:
        raise ValueError('SDK 版本或來源雜湊不符')
    if p['representation'] != 'omf' or p['converter_used'] or p['classification_changed'] or r['classification_changed']:
        raise ValueError('只接受未改分類的原生 OMF 來源')
    if r['probe_sha256'] != sha(probe) or p['evidence_sha256'] != sha(evidence):
        raise ValueError('來源與 IDA 收據綁定不符')
    if r['tool']['version'] != '9.4' or e['tool']['version'] != '9.4':
        raise ValueError('IDA 版本不符')
    if any(x['input']['sha256'] != INPUT_SHA for x in (p, r, e, b)):
        raise ValueError('收據原版輸入不符')
    if sha(original) != INPUT_SHA or sha(baseline) != INPUT_SHA or b['output_sha256'] != INPUT_SHA:
        raise ValueError('原版或 C 基準檔不符')
    if not b['whole_file_equal'] or b['decompilation_complete']:
        raise ValueError('基準檔完成層級不符')
    if len(r['matches']) != len(p['matches']) or not p['matches']:
        raise ValueError('SDK 來源數量不符')
    functions = {f['inventory']['start']: f for f in e['functions']}
    originals = original.read_bytes()
    patches = []
    seen = set()
    for m, reviewed in zip(p['matches'], r['matches']):
        if any(reviewed.get(k) != v for k, v in m.items() if k != 'module_classification'):
            raise ValueError('IDA 審查與 SDK 來源不符')
        address = m['ida_linear_address']
        if address in seen:
            raise ValueError('SDK 函式區間重複')
        seen.add(address)
        f = functions[address]
        inv = f['inventory']
        if inv['classification']['value'] != 'runtime' or inv['classification'] != m['original_classification']:
            raise ValueError('只重建已分類 runtime，不將未知 code 自動歸類')
        if inv['ida_analysis_name'] != m['original_name'] or inv['size'] != m['size']:
            raise ValueError('原始名稱或函式邊界漂移')
        at = int(m['library_file_offset'], 16)
        size = m['object_size']
        if at < 0 or size <= 0 or at + size > len(library_bytes):
            raise ValueError('SDK 完整物件超界')
        obj = library_bytes[at:at+size]
        if hashlib.sha256(obj).hexdigest() != m['object_sha256']:
            raise ValueError('SDK 完整物件雜湊不符')
        sections = omf_sections(obj)
        if len(sections) != 1:
            raise ValueError('SDK 不是唯一完整 code 區段')
        section = sections[0]
        code = section['data']
        if section['relocation_count'] or len(code) < 32 or section['size'] != m['size']:
            raise ValueError('SDK 含未處理重定位或不完整 code')
        if {s['offset'] for s in section['publics']} != {0} or [s['name'] for s in section['publics']] != m['vendor_symbols']:
            raise ValueError('SDK 公開入口或原始符號不符')
        if hashlib.sha256(code).hexdigest() != m['code_sha256']:
            raise ValueError('SDK code 雜湊不符')
        instructions = [i for c in f['chunks'] for i in c['instructions']]
        linear = int(address, 16)
        file_at = int(instructions[0]['file_offset'], 16)
        cursor = 0
        for insn in instructions:
            raw = bytes.fromhex(insn['loaded_bytes'])
            if int(insn['ida_linear_address'], 16) != linear + cursor or int(insn['file_offset'], 16) != file_at + cursor:
                raise ValueError('IDA 函式有缺口或非連續檔案位置')
            if raw != bytes.fromhex(insn['file_bytes']) or originals[file_at+cursor:file_at+cursor+len(raw)] != raw:
                raise ValueError('原版有未處理重定位或不相同指令')
            cursor += len(raw)
        if cursor != len(code) or linear + cursor != int(inv['end'], 16) or originals[file_at:file_at+cursor] != code:
            raise ValueError('SDK 與完整原始函式不符')
        if reviewed['file_offset'] != hex(file_at) or not reviewed['ida_library_flag'] or not reviewed['caller_windows']:
            raise ValueError('缺少 IDA 原始位置、runtime 標記或 caller 證據')
        patches.append((file_at, code, {k: m[k] for k in ('ida_linear_address', 'original_name', 'size', 'member', 'vendor_symbols',
                                                         'object_sha256', 'code_sha256', 'library_file_offset', 'object_size', 'original_classification')}))
    for previous, current in zip(sorted(patches), sorted(patches)[1:]):
        if previous[0] + len(previous[1]) > current[0]:
            raise ValueError('SDK 函式檔案區間重疊')
    return patches, b


def restore(args):
    if not Path('/.dockerenv').exists():
        raise ValueError('本工具只在 Docker 執行')
    patches, base_receipt = plan(args.library, args.probe, args.review, args.evidence, args.original, args.baseline, args.receipt)
    result = bytearray(args.baseline.read_bytes())
    spans = []
    for offset, code, row in patches:
        result[offset:offset+len(code)] = code
        spans.append({**row, 'file_offset': hex(offset), 'source_kind': 'retained_library_from_sdk', 'library_sha256': sha(args.library)})
    if hashlib.sha256(result).hexdigest() != INPUT_SHA:
        raise ValueError('SDK 重建後全檔不符')
    args.output.mkdir(parents=True, exist_ok=False)
    if args.output.stat().st_uid != os.getuid():
        raise ValueError('輸出擁有權不符')
    (args.output/'FD2.EXE').write_bytes(result)
    receipt = {**base_receipt, 'sdk_library_spans': spans, 'sdk_library_function_count': len(spans),
               'sdk_library_code_bytes': sum(s['size'] for s in spans),
               'sdk_library_is_subset_of_retained_library': True, 'classification_changed': False,
               'sdk_inputs': {'library_sha256': sha(args.library), 'probe_sha256': sha(args.probe), 'review_sha256': sha(args.review),
                              'baseline_receipt_sha256': sha(args.receipt), 'driver_sha256': sha(Path(__file__)),
                              'omf_parser_sha256': sha(Path(__file__).with_name('fd2_matching_library_probe.py'))},
               'reason': 'SDK 來源只補既有 runtime 子集；其餘未還原 code 仍來自原版，整體反編譯尚未完成'}
    (args.output/'library-bootstrap-receipt.json').write_text(json.dumps(receipt, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'sdk_functions': len(spans), 'sdk_bytes': receipt['sdk_library_code_bytes'], 'counts': receipt['counts'],
                      'whole_file_equal': True, 'decompilation_complete': False}, ensure_ascii=False))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('library', 'probe', 'review', 'evidence', 'original', 'baseline', 'receipt', 'output'):
        parser.add_argument('--'+name, type=Path, required=True)
    restore(parser.parse_args())


if __name__ == '__main__':
    main()
