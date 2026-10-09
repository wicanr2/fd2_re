#!/usr/bin/env python3
"""用真實 SDK 收據驗證完整物件來源、未分類拒收與 OMF 格式邊界。"""
import argparse
import copy
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
from fd2_matching_library_probe import omf_sections
from fd2_matching_library_restore import restore


def record(kind, data):
    raw = bytes([kind]) + (len(data)+1).to_bytes(2, 'little') + data
    return raw + bytes([-sum(raw) & 255])


def minimal_omf(easy=False, block_offset=0, fixups=False):
    wide = 4 if easy else 2
    parts = [record(0x80, b'\x01m'), record(0x96, b'\x05_TEXT'), record(0x96, b'\x04CODE')]
    if easy:
        parts.append(record(0x88, b'\x80\xaa80386'))
    parts += [record(0x98, bytes([0x68 if easy else 0x69]) + (32).to_bytes(wide, 'little') + b'\x01\x02\x00'),
              record(0x90, b'\x00\x01\x01f' + bytes(wide) + b'\x00'),
              record(0xa0, b'\x01' + block_offset.to_bytes(wide, 'little') + b'\x90'*32)]
    if fixups:
        parts.append(record(0x9c, b'\x00'))
    parts.append(record(0x8a, b'\x00'))
    return b''.join(parts)


def reject_parser(raw):
    try:
        omf_sections(raw)
    except (ValueError, UnicodeError):
        return
    raise AssertionError('不完整／損壞 OMF 未被拒收')


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ('library', 'probe', 'review', 'evidence', 'original', 'baseline', 'receipt', 'output'):
        p.add_argument('--'+name, type=Path, required=True)
    a = p.parse_args()
    assert Path('/.dockerenv').exists()
    standard = omf_sections(minimal_omf())[0]
    easy = omf_sections(minimal_omf(True))[0]
    assert standard['data'] == easy['data'] == b'\x90'*32
    assert standard['publics'][0]['name'] == easy['publics'][0]['name'] == 'f'
    assert easy['omf_encoding'] == 'easy_omf32'
    assert omf_sections(minimal_omf(fixups=True))[0]['relocation_count'] == -1
    reject_parser(minimal_omf()[:-1])
    reject_parser(minimal_omf(block_offset=1))
    corrupted = bytearray(minimal_omf()); corrupted[5] ^= 1
    reject_parser(corrupted)
    restore(a)
    result = json.loads((a.output/'library-bootstrap-receipt.json').read_text())
    baseline = json.loads(a.receipt.read_text())
    assert result['restored_spans'] == baseline['restored_spans']
    assert result['functions'] == baseline['functions']
    assert result['counts'] == baseline['counts']
    assert result['sdk_library_function_count'] == 11 and result['sdk_library_code_bytes'] == 805
    assert result['decompilation_complete'] is False
    rejected = []
    with tempfile.TemporaryDirectory(prefix='fd2-sdk-test-') as scratch:
        root = Path(scratch)
        for case in ('symbol', 'object_hash', 'object_offset', 'unknown_classification', 'input_hash', 'probe_binding', 'library_hash'):
            args = SimpleNamespace(**vars(a))
            args.output = root/(case+'-output')
            probe = json.loads(a.probe.read_text())
            review = json.loads(a.review.read_text())
            if case == 'symbol': probe['matches'][0]['vendor_symbols'] = ['unsupported_alias']
            elif case == 'object_hash': probe['matches'][0]['object_sha256'] = '0'*64
            elif case == 'object_offset': probe['matches'][0]['library_file_offset'] = '0x0'
            elif case == 'input_hash': probe['input']['sha256'] = '0'*64
            elif case == 'probe_binding': review['probe_sha256'] = '0'*64
            elif case == 'library_hash':
                args.library = root/'bad.lib'
                args.library.write_bytes(a.library.read_bytes()[:-1])
            elif case == 'unknown_classification':
                evidence = json.loads(a.evidence.read_text())
                for f in evidence['functions']:
                    if f['inventory']['start'] == probe['matches'][0]['ida_linear_address']:
                        f['inventory']['classification']['value'] = 'unknown'
                        probe['matches'][0]['original_classification'] = copy.deepcopy(f['inventory']['classification'])
                args.evidence = root/'evidence.json'
                args.evidence.write_text(json.dumps(evidence))
                from fd2_matching_library_probe import sha
                probe['evidence_sha256'] = sha(args.evidence)
            args.probe = root/(case+'-probe.json')
            args.probe.write_text(json.dumps(probe))
            from fd2_matching_library_probe import sha
            if case != 'probe_binding': review['probe_sha256'] = sha(args.probe)
            if case in ('symbol', 'object_hash', 'object_offset', 'unknown_classification'):
                for key, value in probe['matches'][0].items():
                    if key != 'module_classification': review['matches'][0][key] = copy.deepcopy(value)
            args.review = root/(case+'-review.json')
            args.review.write_text(json.dumps(review))
            try:
                restore(args)
            except ValueError as error:
                assert not args.output.exists(), case
                rejected.append({'case': case, 'error': str(error), 'output_created': False})
            else:
                raise AssertionError(case+' 未拒收')
    report = {'schema_version': 1, 'positive_sdk_functions': 11, 'positive_sdk_bytes': 805,
              'baseline_c_ledger_unchanged': True, 'omf_cases': ['multiple_lnames', 'easy_omf32', 'fixupp_ineligible', 'truncated', 'gap', 'checksum'],
              'rejections': rejected}
    (a.output/'library-tests.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(report, ensure_ascii=False))


if __name__ == '__main__':
    main()
