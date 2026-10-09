#!/usr/bin/env python3
"""核對物品評分完整負例、REGS完整正例及原始呼叫／位址契約。"""
import argparse
import hashlib
import json
from pathlib import Path
from fd2_matching_game_restore import BINDINGS, verify_original_bindings


def target(i):
    b = bytes.fromhex(i['loaded_bytes'])
    n = 2 if b[:1] == b'\x0f' else 1
    return int(i['ida_linear_address'], 16) + len(b) + int.from_bytes(b[n:], 'little', signed=True)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ('producer', 'helper', 'original', 'evidence', 'output'):
        p.add_argument('--' + name, type=Path, required=True)
    a = p.parse_args()
    assert Path('/.dockerenv').exists()
    sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
    raw = a.original.read_bytes()
    image = json.loads(a.evidence.read_text())
    assert hashlib.sha256(raw).hexdigest() == image['input']['sha256']
    by = {f['inventory']['start']: f for f in image['functions']}
    points = {int(i['ida_linear_address'], 16): i for f in image['functions']
              for c in f['chunks'] for i in c['instructions']}
    functions = [by[k] for k in ('0x1567e', '0x36255')]
    originals = {}
    for f, size in zip(functions, (514, 47)):
        assert f['inventory']['size'] == size and len(f['chunks']) == 1
        originals[f['inventory']['start']] = b''.join(
            bytes.fromhex(i['loaded_bytes']) for i in f['chunks'][0]['instructions'])
        for i in f['chunks'][0]['instructions']:
            at = int(i['file_offset'], 16)
            b = bytes.fromhex(i['file_bytes'])
            assert raw[at:at + len(b)] == b
    producer, helper = [json.loads((d / 'report.json').read_text()) for d in (a.producer, a.helper)]
    for r in (producer, helper):
        assert r['input'] == image['input'] and r['evidence_sha256'] == sha(a.evidence)
    assert len(producer['trials']) == 162 and producer['matched_addresses'] == []
    assert not any(t['exact_interval_bytes'] for t in producer['trials'])
    near = next(t for t in producer['trials'] if t['flags'] == ['-mf', '-3s', '-dAI_ITEM_SELECT_10'])
    near_code = (a.producer / near['stem'] / 'candidate.bin').read_bytes()
    assert len(near_code) == 514 and near_code != originals['0x1567e']
    differences = [n for n, (x, y) in enumerate(zip(near_code, originals['0x1567e'])) if x != y]
    assert differences == list(range(41, 48))
    assert len(helper['trials']) == 27 and helper['matched_addresses'] == ['0x36255']
    positive = [t for t in helper['trials'] if t['exact_interval_bytes']]
    assert len(positive) == 18
    for t in positive:
        assert (a.helper / t['stem'] / 'candidate.bin').read_bytes() == originals['0x36255']
    for n in range(3):
        assert any(t['flags'] == ['-mf', '-3s', '-dINT31_REGS_' + str(n)] for t in positive)
    assert points[0x36255]['loaded_bytes'] == '83ec38'
    assert points[0x36258]['loaded_bytes'] == 'c7042401010000'
    assert points[0x3625f]['loaded_bytes'] == '8b442444'
    assert points[0x36263]['loaded_bytes'] == '25ffff0000'
    assert points[0x36268]['loaded_bytes'] == '8944240c'
    assert points[0x3626c]['loaded_bytes'] == '8d44241c'
    assert points[0x36276]['loaded_bytes'] == '6a31' and target(points[0x36278]) == 0x36d98
    callers = (0x3f6e2, 0x3f8ad, 0x4095c, 0x40c8b, 0x40ce0, 0x40dae, 0x40e42, 0x40f06)
    for at in callers:
        assert points[at]['loaded_bytes'].startswith('e8') and target(points[at]) == 0x36255
    assert functions[1]['inventory']['direct_code_xref_count'] == 8
    assert points[0x1568c]['loaded_bytes'] == '83ec48'
    assert points[0x156cf]['loaded_bytes'] == '6890010000'
    assert target(points[0x156d4]) == BINDINGS['malloc']
    assert target(points[0x156e4]) == 0x1b8a6 and target(points[0x156f2]) == 0x15878
    assert target(points[0x1586e]) == BINDINGS['free']
    assert points[0x157dc]['loaded_bytes'] == '89c5' and points[0x157f0]['loaded_bytes'] == '89e0'
    assert points[0x157f3]['loaded_bytes'] == '55' and target(points[0x157f8]) == 0x15880
    assert points[0x15800]['loaded_bytes'] == '3b05333c0500' and target(points[0x15806]) == 0x15828
    for at, value in ((0x15808, 0x53c33), (0x15811, 0x53c37), (0x1581a, 0x53c3b), (0x15823, 0x53c3f)):
        assert bytes.fromhex(points[at]['loaded_bytes']) == b'\xa3' + value.to_bytes(4, 'little')
    assert target(points[0x1575a]) == target(points[0x157d4]) == 0x14818
    assert target(points[0x157b5]) == 0x149f8
    assert points[0x15833]['loaded_bytes'] == '8d043f'
    assert points[0x4e577]['loaded_bytes'] == 'f7e2' and points[0x4e581]['loaded_bytes'] == '5d'
    assert [points[at]['loaded_bytes'] for at in (0x4dc2f, 0x4dc30, 0x4dc31)] == ['5b', '5e', '5f']
    rejected = []
    for f, names in ((functions[0], ('sub_4E555', 'sub_4E56C', 'sub_14B16', 'sub_15880', 'dword_53C33')),
                     (functions[1], ('int386',))):
        for name in names:
            bindings = dict(BINDINGS)
            bindings[name] += 1
            try:
                verify_original_bindings([f], raw, bindings)
            except ValueError:
                rejected.append(name)
            else:
                raise AssertionError(name)
    result = {'schema_version': 1, 'producer_complete_bytes': 514, 'producer_candidates': 162,
              'producer_exact_candidates': 0, 'same_length_negative_rejected': True,
              'close_candidate_differences': differences, 'producer_local_bytes': 72,
              'allocation_precedes_count': True, 'zero_count_bypasses_free': True,
              'full_target_count_consumer': '0x157f8', 'strict_greater_winner_preserved': True,
              'helper_complete_bytes': 47, 'helper_candidates': 27, 'helper_exact_candidates': len(positive),
              'regs_local_bytes': 56, 'third_argument_bits': 32, 'third_argument_mask': 65535,
              'input_output_regs_offset': 28, 'helper_callers': [hex(at) for at in callers],
              'wrong_addresses_rejected': rejected, 'classification_changed': False,
              'author_declarations_and_original_compiler_version': 'unknown',
              'high_level_interrupt_service_inferred': False, 'pragma_allowance_is_actual_clobber': False,
              'source_report_sha256': {'producer': sha(a.producer / 'report.json'), 'helper': sha(a.helper / 'report.json')}}
    a.output.mkdir(parents=True, exist_ok=False)
    (a.output / 'result.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(result, ensure_ascii=False))


if __name__ == '__main__':
    main()
