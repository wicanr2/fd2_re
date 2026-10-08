#!/usr/bin/env python3
"""以真實 compiler 物件驗證非連續函式布局及標準 REL32，所有檢查只在Docker。"""

import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import struct
import subprocess

from fd2_matching_game_restore import BINDINGS
from fd2_matching_pilot import linked_text
from fd2_matching_sparse import read_sparse_pe, split_coff


def run_link(coff, script, output, sections, entry='sub_2111A', defined=('sub_2111a', 'sub_21b18')):
    definitions = '\n'.join(f'{key} = {value:#x};' for key, value in BINDINGS.items()
                            if key.lower() not in defined)
    script.write_text(definitions + '\nSECTIONS { ' + sections + ' /DISCARD/ : { *(CONST) *(CONST2) *(_DATA) *(_BSS) *(.depend) *(.reloc) } }\n')
    subprocess.run(['ld', '-mi386pe', '--image-base', '0', '--section-alignment', '1', '--file-alignment', '1', '--no-insert-timestamp', '-T', str(script), '-e', entry, '-o', str(output), str(coff)], check=True, capture_output=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--linked', type=Path, required=True)
    parser.add_argument('--original', type=Path, required=True)
    parser.add_argument('--evidence', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--objects', type=Path)
    parser.add_argument('--additional-linked', type=Path)
    args = parser.parse_args()
    if not Path('/.dockerenv').exists():
        raise SystemExit('只在Docker執行')
    report = json.loads((args.linked / 'report.json').read_text())
    trial = next(t for t in report['trials'] if t.get('layout') == 'sparse_functions' and t['exact_interval_bytes'])
    source = args.linked / trial['stem']
    raw = (source / 'candidate.cof').read_bytes()
    dis = (source / 'candidate.dis').read_text()
    addresses = [int(a, 16) for a in trial['addresses']]
    converted, meta = split_coff(raw, dis, addresses)
    assert meta == trial['sparse_layout']
    assert converted == (source / 'sparse.cof').read_bytes()
    assert [p['size'] for p in meta['fragments']] == [138, 129]
    assert len(meta['lifted_branches']) == 1
    assert meta['lifted_branches'][0]['target_address'] == '0x21190'
    args.output.mkdir(parents=True, exist_ok=True)
    # 放回 compiler 原本的連續布局時，連結後全部267 bytes必須等於未分割物件。
    original_pe = args.output / 'identity-original.exe'
    run_link(source / 'candidate.cof', args.output / 'identity-original.ld', original_pe,
             '.text 0x2111a : SUBALIGN(1) { *(_TEXT) }')
    identity = copy.deepcopy(meta['fragments'])
    identity[1]['address'] = hex(addresses[0] + identity[1]['compiler_offset'])
    split_pe = args.output / 'identity-split.exe'
    sections = ' '.join(f".m{n} {p['address']} : SUBALIGN(1) {{ *({p['section']}) }}" for n, p in enumerate(identity))
    run_link(source / 'sparse.cof', args.output / 'identity-split.ld', split_pe, sections)
    original_code = linked_text(original_pe, addresses[0], 267)
    assert read_sparse_pe(split_pe, identity) == original_code
    # speed策略沒有共享分支，仍是有效compiler物件；轉換後不改其完整產碼。
    plain = next(t for t in report['trials'] if t.get('layout') == 'sparse_functions'
                 and not t['sparse_layout']['lifted_branches'])
    plain_source = args.linked / plain['stem']
    plain_original = args.output / 'plain-original.exe'
    run_link(plain_source / 'candidate.cof', args.output / 'plain-original.ld', plain_original,
             '.text 0x2111a : SUBALIGN(1) { *(_TEXT) }')
    plain_identity = copy.deepcopy(plain['sparse_layout']['fragments'])
    plain_identity[1]['address'] = hex(addresses[0] + plain_identity[1]['compiler_offset'])
    plain_split = args.output / 'plain-split.exe'
    sections = ' '.join(f".m{n} {p['address']} : SUBALIGN(1) {{ *({p['section']}) }}" for n, p in enumerate(plain_identity))
    run_link(plain_source / 'sparse.cof', args.output / 'plain-split.ld', plain_split, sections)
    assert read_sparse_pe(plain_split, plain_identity) == linked_text(plain_original, addresses[0], plain['candidate_size'])
    # 同一物件以實際非連續布局連結後，完整函式 bytes 逐一等於原版。
    assert read_sparse_pe(source / 'linked.exe', meta['fragments']) == (source / 'candidate.bin').read_bytes()
    rejected = []
    wrong_name = dis.replace('sub_21B18:', 'sub_21B19:')
    bad_code = bytearray(raw)
    text_at = struct.unpack_from('<I', raw, 20 + 20)[0]
    bad_code[text_at + 0x106] = 0xE8
    for name, code, listing, positions in (
        ('entry_name_changed', raw, wrong_name, addresses),
        ('branch_opcode_changed', bytes(bad_code), dis, addresses),
        ('layout_address_changed', raw, dis, [addresses[0], addresses[0] + 1]),
    ):
        try:
            split_coff(code, listing, positions)
        except ValueError:
            rejected.append(name)
        else:
            raise AssertionError(name + '沒有拒收')
    # 真實全檔組合正例及報告的負例，檢查輸出前拒收。
    cases = ('positive', 'layout_missing', 'fragments_changed', 'entry_unproven')
    for case in cases:
        directory = args.output / case
        directory.mkdir()
        candidate = directory / trial['stem']
        candidate.mkdir()
        (candidate / 'candidate.bin').write_bytes((source / 'candidate.bin').read_bytes())
        receipt = copy.deepcopy(report)
        receipt['trials'] = [copy.deepcopy(trial)]
        t = receipt['trials'][0]
        if case == 'layout_missing':
            t.pop('layout')
        elif case == 'fragments_changed':
            t['sparse_layout']['fragments'][1]['address'] = '0x21b19'
        elif case == 'entry_unproven':
            t['exact_entry_offsets'] = False
        (directory / 'report.json').write_text(json.dumps(receipt))
        output = directory / 'output'
        result = subprocess.run(['python', str(Path(__file__).with_name('fd2_matching_bootstrap.py')), '--original', str(args.original), '--evidence', str(args.evidence), '--restored', str(directory), '--output', str(output)], capture_output=True, text=True)
        if case == 'positive':
            assert result.returncode == 0, result.stderr
            rebuilt = json.loads((output / 'bootstrap-receipt.json').read_text())
            assert rebuilt['counts']['matched_c'] == 2 and not rebuilt['decompilation_complete']
            assert (output / 'FD2.EXE').read_bytes() == args.original.read_bytes()
        else:
            assert result.returncode != 0 and not (output / 'FD2.EXE').exists() and not (output / 'bootstrap-receipt.json').exists(), result.stderr
            rejected.append(case)
    extended = []
    for macro, size, targets in [('INDEXED_ALL', 385, {'0x21190', '0x21b0b'}),
                                 ('QUAKE_GROUP', 942, {'0x2153b'})]:
        found = [t for t in report['trials'] if t['exact_interval_bytes'] and '-d' + macro in t['flags']]
        if not found:
            continue
        t = found[0]
        assert len(t['addresses']) == 4 and t['candidate_size'] == size
        assert {b['target_address'] for b in t['sparse_layout']['lifted_branches']} == targets
        directory = args.output / macro
        directory.mkdir()
        source = args.linked / t['stem']
        entry = 'sub_' + format(int(t['addresses'][0], 16), 'X')
        defined = tuple('sub_' + a[2:] for a in t['addresses'])
        original_pe = directory / 'original.exe'
        run_link(source / 'candidate.cof', directory / 'original.ld', original_pe,
                 f".text {t['address']} : SUBALIGN(1) {{ *(_TEXT) }}", entry, defined)
        identity = copy.deepcopy(t['sparse_layout']['fragments'])
        for p in identity:
            p['address'] = hex(int(t['address'], 16) + p['compiler_offset'])
        split_pe = directory / 'split.exe'
        sections = ' '.join(f".m{n} {p['address']} : SUBALIGN(1) {{ *({p['section']}) }}" for n, p in enumerate(identity))
        run_link(source / 'sparse.cof', directory / 'split.ld', split_pe, sections, entry, defined)
        assert read_sparse_pe(split_pe, identity) == linked_text(original_pe, int(t['address'], 16), size)
        assert read_sparse_pe(source / 'linked.exe', t['sparse_layout']['fragments']) == (source / 'candidate.bin').read_bytes()
        received = copy.deepcopy(report)
        received['trials'] = [t]
        (directory / 'report.json').write_text(json.dumps(received))
        candidate = directory / t['stem']
        candidate.mkdir()
        (candidate / 'candidate.bin').write_bytes((source / 'candidate.bin').read_bytes())
        output = directory / 'output'
        check = subprocess.run(['python', str(Path(__file__).with_name('fd2_matching_bootstrap.py')), '--original', str(args.original), '--evidence', str(args.evidence), '--restored', str(directory), '--output', str(output)], capture_output=True, text=True)
        assert check.returncode == 0, check.stderr
        assert json.loads((output / 'bootstrap-receipt.json').read_text())['counts']['matched_c'] == 4
        assert (output / 'FD2.EXE').read_bytes() == args.original.read_bytes()
        extended.append({'macro': macro, 'complete_functions': 4, 'identity_layout_bytes': size, 'targets': sorted(targets)})
    include_rejected = []
    if args.objects is not None:
        compile_report = json.loads((args.objects / 'compile-report.json').read_text())
        assert len(compile_report['source_includes']) == 1
        for case in ('include_missing', 'include_bytes_changed', 'include_time_changed', 'include_hash_changed'):
            directory = args.output / case
            directory.mkdir()
            (directory / 'GAME.C').write_bytes((args.objects / 'GAME.C').read_bytes())
            include = directory / 'QUAKE.C'
            if case != 'include_missing':
                include.write_bytes((args.objects / 'QUAKE.C').read_bytes())
                os.utime(include, (315532800, 315532800))
            received = copy.deepcopy(compile_report)
            if case == 'include_bytes_changed':
                include.write_bytes(include.read_bytes() + b'\n')
                os.utime(include, (315532800, 315532800))
            elif case == 'include_time_changed':
                os.utime(include, (315532801, 315532801))
            elif case == 'include_hash_changed':
                received['source_includes'][0]['source_sha256'] = '0' * 64
            (directory / 'compile-report.json').write_text(json.dumps(received))
            output = directory / 'output'
            check = subprocess.run(['python', str(Path(__file__).with_name('fd2_matching_game_restore.py')), 'link', '--objects', str(directory), '--original', str(args.original), '--evidence', str(args.evidence), '--output', str(output)], capture_output=True, text=True)
            assert check.returncode != 0 and 'include' in check.stderr and not output.exists(), check.stderr
            include_rejected.append(case)
    indirect = None
    if args.additional_linked is not None:
        more = json.loads((args.additional_linked / 'report.json').read_text())
        t = next(t for t in more['trials'] if '-dRADIAL_GROUP' in t['flags'] and t.get('linkable', True))
        source = args.additional_linked / t['stem']
        converted, layout = split_coff((source / 'candidate.cof').read_bytes(),
                                       (source / 'candidate.dis').read_text(),
                                       [int(a, 16) for a in t['addresses']])
        assert converted == (source / 'sparse.cof').read_bytes() and layout == t['sparse_layout']
        assert 'funcs_1199C' in (source / 'candidate.dis').read_text()
        directory = args.output / 'indirect-call'
        directory.mkdir()
        defined = tuple('sub_' + a[2:] for a in t['addresses'])
        original_pe = directory / 'original.exe'
        run_link(source / 'candidate.cof', directory / 'original.ld', original_pe,
                 f".text {t['address']} : SUBALIGN(1) {{ *(_TEXT) }}", 'sub_190AC', defined)
        identity = copy.deepcopy(layout['fragments'])
        for p in identity:
            p['address'] = hex(int(t['address'], 16) + p['compiler_offset'])
        split_pe = directory / 'split.exe'
        sections = ' '.join(f".m{n} {p['address']} : SUBALIGN(1) {{ *({p['section']}) }}" for n, p in enumerate(identity))
        run_link(source / 'sparse.cof', directory / 'split.ld', split_pe, sections, 'sub_190AC', defined)
        assert read_sparse_pe(split_pe, identity) == linked_text(original_pe, int(t['address'], 16), t['candidate_size'])
        assert read_sparse_pe(source / 'linked.exe', layout['fragments']) == (source / 'candidate.bin').read_bytes()
        indirect = {'source_report_sha256': hashlib.sha256((args.additional_linked / 'report.json').read_bytes()).hexdigest(),
                    'identity_layout_bytes': t['candidate_size'], 'matched_original': t['exact_interval_bytes'],
                    'indirect_function_table': 'funcs_1199C', 'all_instruction_bytes_preserved': True}
    result = {'identity_layout_bytes': 267, 'unshared_identity_layout_bytes': plain['candidate_size'],
              'matched_original_functions': 2, 'lifted_rel32_branches': 1,
              'target_address': '0x21190', 'rejected': rejected,
              'source_report_sha256': hashlib.sha256((args.linked / 'report.json').read_bytes()).hexdigest(),
              'extended_groups': extended, 'include_rejected': include_rejected,
              'indirect_call_validation': indirect}
    (args.output / 'result.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(result, ensure_ascii=False))


if __name__ == '__main__':
    main()
