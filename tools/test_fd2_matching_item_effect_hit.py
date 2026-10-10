#!/usr/bin/env python3
"""核對物品效果完整範圍、命中表／signed算式與錯址拒收。"""
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
    for name in ('work', 'original', 'evidence', 'output'):
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
    originals = {}
    for at, size in (('0x20c6f', 1043), ('0x1c75e', 193)):
        f = by[at]
        assert f['inventory']['size'] == size and len(f['chunks']) == 1
        originals[at] = b''.join(bytes.fromhex(i['loaded_bytes']) for i in f['chunks'][0]['instructions'])
        for i in f['chunks'][0]['instructions']:
            offset = int(i['file_offset'], 16)
            b = bytes.fromhex(i['file_bytes'])
            assert raw[offset:offset + len(b)] == b
    matrices = []
    for label in ('10a', '95'):
        for name, at, count in (('item-effect', '0x20c6f', 108), ('command-hit', '0x1c75e', 90)):
            linked = a.work / (name + '-final-' + label + '-linked-r1')
            r = json.loads((linked / 'report.json').read_text())
            assert r['input'] == image['input'] and r['evidence_sha256'] == sha(a.evidence)
            assert len(r['trials']) == count
            exact = []
            for t in r['trials']:
                equal = (linked / t['stem'] / 'candidate.bin').read_bytes() == originals[at]
                assert equal == t['exact_interval_bytes']
                if equal:
                    exact.append(t['flags'])
            assert r['matched_addresses'] == ([at] if exact else [])
            matrices.append({'name': name, 'compiler': label, 'candidates': count, 'exact_profiles': exact})
    linked = a.work / 'command-hit-final-10a-linked-r1'
    r = json.loads((linked / 'report.json').read_text())
    near = next(t for t in r['trials'] if t['flags'] == ['-mf', '-3s', '-dCOMMAND_HIT_4'])
    code = (linked / near['stem'] / 'candidate.bin').read_bytes()
    assert len(code) == 193 and code != originals['0x1c75e'] and not near['exact_interval_bytes']
    differences = [n for n, (x, y) in enumerate(zip(code, originals['0x1c75e'])) if x != y]
    assert points[0x20c7d]['loaded_bytes'] == '83ec6c'
    assert points[0x20cb5]['loaded_bytes'] == '0fb7500e'
    assert points[0x20cbd]['loaded_bytes'] == '8a580d'
    assert points[0x20cb9]['loaded_bytes'] == '89542468'
    assert target(points[0x20ce0]) == 0x211a4 and target(points[0x20d32]) == 0x1b8e7
    assert points[0x20df0]['loaded_bytes'] == 'eb1d' and target(points[0x20e16]) == 0x20e42
    assert target(points[0x20df7]) == 0x1c9dd and target(points[0x20e38]) == 0x1e1dc
    assert points[0x20f29]['loaded_bytes'] == '89442464' and points[0x20f57]['loaded_bytes'] == '88433c'
    assert target(points[0x20f80]) == 0x1c4cc and target(points[0x20f9b]) == 0x1cd17
    assert target(points[0x20fcb]) == 0x1c75e and target(points[0x20fef]) == 0x1df58
    assert points[0x21055]['loaded_bytes'] == points[0x21067]['loaded_bytes'] == '89e0'
    assert target(points[0x21058]) == 0x1b6b7 and target(points[0x21072]) == 0x1aa1d
    assert points[0x1c76b]['loaded_bytes'] == '83ec74'
    assert points[0x1c76e]['loaded_bytes'] == 'b91c000000'
    assert points[0x1c775]['loaded_bytes'] == 'be961f0500' and points[0x1c77a]['loaded_bytes'] == 'f3a5'
    assert points[0x1c792]['loaded_bytes'] == '0fb6740320'
    assert points[0x1c7a8]['loaded_bytes'] == '0fbf10' and points[0x1c7ab]['loaded_bytes'] == '0faf54b4fc'
    assert points[0x1c7b0]['loaded_bytes'] == 'be0a000000' and points[0x1c7ba]['loaded_bytes'] == 'f7fe'
    assert points[0x1c7c2]['loaded_bytes'] == '89442470'
    assert target(points[0x1c7ce]) == target(points[0x1c7d8]) == 0x1c7ed
    assert target(points[0x1c7eb]) == 0x1c816
    assert target(points[0x1c7ed]) == 0x4e893 and points[0x1c7fc]['loaded_bytes'] == 'f7fb'
    assert target(points[0x1c802]) == 0x1c816 and target(points[0x1c80c]) == 0x1c81f
    callers = (0x20fcb, 0x21176, 0x2129f, 0x2142f, 0x214ed, 0x2182e, 0x2b114)
    for at in callers:
        assert points[at]['loaded_bytes'].startswith('e8') and target(points[at]) == 0x1c75e
    assert points[0x1c768]['loaded_bytes'] == '53' and points[0x1c81d]['loaded_bytes'] == '5b'
    assert target(points[0x1ca6d]) == 0x1c9d5 and target(points[0x1ca84]) == 0x1c9c7
    assert target(points[0x1c9d8]) == 0x22bbe
    rejected = []
    for at, names in (('0x20c6f', ('sub_211A4', 'sub_21082', 'sub_1C9DD', 'sub_2111A', 'sub_2218A',
                                  'dword_53EC4', 'dword_53EC8', 'dword_53A45')),
                      ('0x1c75e', ('unk_51F96', 'sub_4E516', 'sub_4E893', 'sub_1F183', 'sub_1C81F'))):
        for name in names:
            bindings = dict(BINDINGS)
            bindings[name] += 1
            try:
                verify_original_bindings([by[at]], raw, bindings)
            except ValueError:
                rejected.append(name)
            else:
                raise AssertionError(name)
    result = {'schema_version': 1, 'matrices': matrices, 'complete_function_bytes': {'0x20c6f': 1043, '0x1c75e': 193},
              'effect_local_bytes': 108, 'saved_byte_offset': 100, 'power_word_to_dword_offset': 104,
              'full_target_list_and_count_preserved': True, 'common_finish_buffer_bytes': 100,
              'hit_local_bytes': 116, 'copied_table_bytes': 112, 'raw_class_offset': 32, 'class_table_bias': -1,
              'signed_input_word_and_divisions': True, 'rng_execution_parity': False,
              'original_near_callers': [hex(at) for at in callers], 'inventory_code_xref_count': 8,
              'near_calls_are_all_ida_code_xrefs': False, 'shared_tail_not_cropped_or_counted': True,
              'same_length_negative_not_accepted': True, 'close_candidate_differences': differences,
              'wrong_addresses_rejected': rejected, 'classification_changed': False,
              'author_declarations_and_original_compiler_version': 'unknown', 'pragma_allowance_is_actual_clobber': False}
    a.output.mkdir(parents=True, exist_ok=False)
    (a.output / 'result.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(result, ensure_ascii=False))


if __name__ == '__main__':
    main()
