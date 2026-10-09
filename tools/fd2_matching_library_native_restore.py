#!/usr/bin/env python3
"""從原廠SDK重新連結完整CODE後加入本機基準；函式庫覆蓋與C覆蓋分列。"""
import argparse
import contextlib
import copy
import hashlib
import io
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
from fd2_matching_library_link import INPUT_SHA, LIBRARIES, omf_sections, run as native_link, sha


def prepare(args):
    if sha(args.original) != INPUT_SHA or sha(args.baseline) != INPUT_SHA or sha(args.library) != LIBRARIES['9.5'][0]:
        raise ValueError('原版、基準或SDK來源不符')
    report_path = args.linked/'library-link-report.json'
    report = json.loads(report_path.read_text()); review = json.loads(args.review.read_text()); base = json.loads(args.receipt.read_text())
    if base['input']['sha256'] != INPUT_SHA or base['output_sha256'] != INPUT_SHA or not base['whole_file_equal'] or base['decompilation_complete']:
        raise ValueError('基準收據版本或完成層級不符')
    if report['driver_sha256'] != sha(Path(__file__).with_name('fd2_matching_library_link.py')):
        raise ValueError('原生連結器來源版本不符')
    if review['probe_sha256'] != sha(report_path) or review['source_kind'] != 'native_sdk_link' or review['tool']['version'] != '9.4':
        raise ValueError('IDA收據與原生SDK來源未綁定')
    if review['input'] != report['input'] or report['input'] != base['input'] or review['library'] != report['library']:
        raise ValueError('SDK／IDA／基準不是相同輸入')
    if review['classification_changed'] or report['classification_changed'] or len(review['matches']) != len(report['trials']):
        raise ValueError('SDK分類或完整函式數量矛盾')
    ledger = {f['ida_linear_address']:f['source_kind'] for f in base['functions']}
    for trial, checked in zip(report['trials'],review['matches']):
        if ledger.get(trial['ida_linear_address']) != 'retained_library':
            raise ValueError('SDK只能補既有保留函式庫，不替代C或未分類code')
        if any(checked.get(k) != v for k,v in trial.items()) or not trial['exact_interval_bytes']:
            raise ValueError('SDK與IDA完整函式審查不符')
        if not checked['ida_library_flag'] or not checked['caller_windows']:
            raise ValueError('缺少既有runtime標記或caller證據')
    image = json.loads(args.evidence.read_text()); functions = {f['inventory']['start']:f for f in image['functions']}
    raw = args.original.read_bytes(); library = args.library.read_bytes()
    existing = copy.deepcopy(base.get('sdk_library_spans',[])); native_addresses = {t['ida_linear_address'] for t in report['trials']}
    seen = set()
    for span in existing:
        address = span['ida_linear_address']
        if address in seen: raise ValueError('既有SDK台帳重複')
        seen.add(address); f = functions[address]
        if f['inventory']['classification']['value'] != 'runtime' or span['library_sha256'] != sha(args.library):
            raise ValueError('既有SDK分類或來源不符')
        if address in native_addresses: continue  # 由下方原生連結重新驗證。
        at = int(span['library_file_offset'],16); obj = library[at:at+span['object_size']]
        if hashlib.sha256(obj).hexdigest() != span['object_sha256']: raise ValueError('既有SDK物件雜湊不符')
        sections = omf_sections(obj)
        if len(sections) != 1 or sections[0]['relocation_count'] or len(sections[0]['data']) != span['size']:
            raise ValueError('既有SDK不是完整無重定位CODE')
        offset = int(f['chunks'][0]['instructions'][0]['file_offset'],16); code = sections[0]['data']
        if hashlib.sha256(code).hexdigest() != span['code_sha256'] or raw[offset:offset+len(code)] != code:
            raise ValueError('既有SDK完整區間不符')
    # 產物必須由原廠物件重新連結；不僅相信已存candidate.bin或收據中的hash。
    patches = []
    with tempfile.TemporaryDirectory(prefix='fd2-sdk-native-rebuild-') as temp:
        linked = Path(temp)/'linked'
        native_args = SimpleNamespace(objects=args.objects,library=args.library,evidence=args.evidence,original=args.original,
            output=linked,wlink=Path('/opt/watcom/binl64/wlink'),wdis=Path('/opt/watcom/binl64/wdis'),
            members=[Path(t['member']).stem for t in report['trials']])
        with contextlib.redirect_stdout(io.StringIO()): native_link(native_args)
        if (linked/'library-link-report.json').read_bytes() != report_path.read_bytes():
            raise ValueError('實際SDK重建與完整收據不符')
        for trial in report['trials']:
            address = trial['ida_linear_address']; f = functions[address]; inv = f['inventory']
            if inv['classification'] != trial['original_classification'] or inv['classification']['value'] != 'runtime':
                raise ValueError('SDK不能將未知code改列runtime')
            code = (linked/Path(trial['member']).stem/'candidate.bin').read_bytes()
            saved = args.linked/Path(trial['member']).stem/'candidate.bin'
            if saved.read_bytes() != code: raise ValueError('已存SDK CODE與實際重建不符')
            offset = int(f['chunks'][0]['instructions'][0]['file_offset'],16)
            if len(code) != inv['size'] or raw[offset:offset+len(code)] != code:
                raise ValueError('SDK不是完整原始檔區間，或含未明示解除的LE fixup')
            row = {k:trial[k] for k in ('ida_linear_address','original_name','size','member','vendor_symbols','object_sha256',
                                      'object_size','library_file_offset','code_sha256','original_classification','bindings')}
            row.update(file_offset=hex(offset),source_kind='retained_library_from_native_sdk',library_sha256=sha(args.library),
                       native_source_report_sha256=sha(report_path),native_tool_inputs=report['native_tools'])
            patches.append((offset,code,row))
    return base, existing, patches


def restore(args):
    if not Path('/.dockerenv').exists(): raise ValueError('本工具只在Docker執行')
    base, existing, patches = prepare(args)
    old = {s['ida_linear_address']:s for s in existing}; merged = dict(old); new = []
    binary = bytearray(args.baseline.read_bytes())
    for offset,code,row in patches:
        binary[offset:offset+len(code)] = code
        address = row['ida_linear_address']
        if address in old:
            if old[address]['code_sha256'] != row['code_sha256'] or old[address]['size'] != row['size']:
                raise ValueError('重複SDK函式與既有出處矛盾')
        else:
            merged[address]=row;new.append(row)
    if hashlib.sha256(binary).hexdigest() != INPUT_SHA: raise ValueError('原生SDK重建後全檔不符')
    args.output.mkdir(parents=True,exist_ok=False)
    (args.output/'FD2.EXE').write_bytes(binary)
    result = {**base,'sdk_library_spans':list(merged.values()),'sdk_library_function_count':len(merged),
        'sdk_library_code_bytes':sum(s['size'] for s in merged.values()),'sdk_library_is_subset_of_retained_library':True,
        'sdk_native_rebuilt_spans':[r for _,_,r in patches],'sdk_native_new_functions':len(new),'sdk_native_new_code_bytes':sum(s['size'] for s in new),
        'sdk_native_inputs':{'library_sha256':sha(args.library),'source_report_sha256':sha(args.linked/'library-link-report.json'),
                             'review_sha256':sha(args.review),'baseline_receipt_sha256':sha(args.receipt),'driver_sha256':sha(Path(__file__))},
        'classification_changed':False,'decompilation_complete':False,
        'reason':'SDK重建只補已保留函式庫來源；C及原始分類不變，其餘遊戲code仍未還原'}
    (args.output/'native-library-bootstrap-receipt.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'sdk_functions':len(merged),'sdk_bytes':result['sdk_library_code_bytes'],'new_sdk_functions':len(new),
                      'new_sdk_bytes':result['sdk_native_new_code_bytes'],'counts':result['counts'],'whole_file_equal':True,'decompilation_complete':False},ensure_ascii=False))


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('objects','library','evidence','original','linked','review','baseline','receipt','output'):p.add_argument('--'+name,type=Path,required=True)
    restore(p.parse_args())


if __name__=='__main__':main()
