#!/usr/bin/env python3
"""驗證完整ACTING／標題共享收尾、兩個入口及既有宣告組合負例。"""
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
    raw = a.original.read_bytes()
    image = json.loads(a.evidence.read_text())
    sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
    assert hashlib.sha256(raw).hexdigest() == image['input']['sha256']
    by = {f['inventory']['start']: f for f in image['functions']}
    points = {int(i['ida_linear_address'], 16): i for f in image['functions']
              for c in f['chunks'] for i in c['instructions']}
    originals = {}
    for at, size in (('0x1366a', 818), ('0x1f894', 1765), ('0x20c6f', 1043), ('0x1c75e', 193)):
        f = by[at]
        assert f['inventory']['size'] == size and len(f['chunks']) == 1
        originals[at] = b''.join(bytes.fromhex(i['loaded_bytes']) for i in f['chunks'][0]['instructions'])
        for i in f['chunks'][0]['instructions']:
            offset = int(i['file_offset'], 16)
            b = bytes.fromhex(i['file_bytes'])
            assert raw[offset:offset + len(b)] == b
    assert target(points[0x1ff74]) == 0x13994
    tail = b''.join(bytes.fromhex(points[at]['loaded_bytes']) for at in (0x13994, 0x13997, 0x13998, 0x13999, 0x1399a, 0x1399b))
    assert tail == bytes.fromhex('83c45c5d5f5e5bc3')
    assert points[0x13678]['loaded_bytes'] == points[0x1f8a2]['loaded_bytes'] == '83ec5c'
    assert points[0x1f8c8]['loaded_bytes'] == 'b90f000000' and points[0x1f8d4]['loaded_bytes'] == 'f3a5'
    assert points[0x1fa85]['loaded_bytes'] == 'be17020000'
    assert points[0x1fa52]['loaded_bytes'] == '83fe05'
    assert target(points[0x1fc60]) == 0x1fa8c and target(points[0x1fc59]) == 0x10620
    assert target(points[0x1fd8b]) == 0x36fcc and points[0x1fda2]['loaded_bytes'].startswith('e8')
    assert points[0x1fdf4]['loaded_bytes'] == 'c744245002000000'
    assert points[0x1fe07]['loaded_bytes'] == 'c744245003000000'
    assert points[0x1fe48]['loaded_bytes'] == 'c6058e3a050010'
    assert target(points[0x1fe5b]) == 0x36d98
    assert points[0x1ff2c]['loaded_bytes'] == '83fe04'
    assert points[0x1369d]['loaded_bytes'] == '88540420' and points[0x136a4]['loaded_bytes'] == '881404'
    assert points[0x136bc]['loaded_bytes'] == 'f644245480'
    assert points[0x136c7]['loaded_bytes'] == '806424547f'
    assert target(points[0x1391e]) == 0x2c9ec and points[0x138a2]['loaded_bytes'] == '88540304'
    matrices = []
    structural_examples = []
    expected_group = originals['0x1366a'] + originals['0x1f894']
    for label in ('10a', '95'):
        for name, count, expected in (('effect-mix-final', 54, originals['0x20c6f']),
                                      ('hit-mix-final', 9, originals['0x1c75e']),
                                      ('title-acting-final', 54, expected_group)):
            linked = a.work / (name + '-' + label + '-linked-r1')
            r = json.loads((linked / 'report.json').read_text())
            assert len(r['trials']) == count and r['input'] == image['input'] and r['evidence_sha256'] == sha(a.evidence)
            exact = 0
            rejected = 0
            for t in r['trials']:
                if not t.get('linkable', True):
                    assert t['comparison_performed'] is False and not t['exact_interval_bytes']
                    rejected += 1
                    continue
                code = (linked / t['stem'] / 'candidate.bin').read_bytes()
                equal = code == expected and t.get('exact_entry_offsets', True)
                assert equal == t['exact_interval_bytes']
                exact += equal
                if name == 'title-acting-final':
                    assert [(f['address'], f['size']) for f in t['function_intervals']] == [('0x1366a', 818), ('0x1f894', 1765)]
                    fragments = t['sparse_layout']['fragments']
                    assert [f['address'] for f in fragments] == ['0x1366a', '0x1f894']
                    assert sum(f['size'] for f in fragments) == len(code)
                    for branch in t['sparse_layout']['lifted_branches']:
                        if branch['source_address'] == '0x1f894' and int(branch['target_address'], 16) < 0x1f894:
                            assert branch['relocation_type'] == 'IMAGE_REL_I386_REL32'
                            assert bytes.fromhex(branch['instruction_bytes'])[0] == 0xe9
                            structural_examples.append({'compiler': label, 'flags': t['flags'], 'branch': branch,
                                                        'whole_group_equal': t['exact_interval_bytes']})
            matrices.append({'name': name, 'compiler': label, 'candidates': count, 'exact_candidates': exact, 'layout_rejections': rejected})
    assert structural_examples
    rejected = []
    for at, names in (('0x1366a', ('sub_4E7F8', 'dword_53AFB', 'sub_2C9EC')),
                      ('0x1f894', ('unk_5204E', 'aRb_2', 'aFd2Sav_3', 'sub_1F73F', 'sub_1F81E',
                                   'sub_1FF79', 'sub_286BD', 'word_53A8D', 'int386'))):
        for name in names:
            bindings = dict(BINDINGS)
            bindings[name] += 1
            try:
                verify_original_bindings([by[at]], raw, bindings)
            except ValueError:
                rejected.append(name)
            else:
                raise AssertionError(name)
    result = {'schema_version': 1, 'whole_group_original_bytes': 2583, 'original_function_bytes': [818, 1765],
              'original_local_bytes': [92, 92], 'shared_tail_address': '0x13994', 'shared_tail_bytes': 8,
              'original_tail_jump': '0x1ff74', 'caller_scope': '完整兩個原始函式，不裁切共享返回段',
              'scroll_initial_row': 535, 'copied_event_table_bytes': 60, 'scroll_image_count': 5,
              'constructed_scroll_buffer_bytes': 235200, 'save_read_bytes': 22987,
              'save_menu_counts': [1, 2, 3], 'confirm_flashes': 4,
              'compiler_shared_tail_structurally_generated': True, 'structural_examples': structural_examples,
              'structural_tail_implies_byte_match': False, 'matrices': matrices,
              'wrong_addresses_rejected': rejected, 'classification_changed': False,
              'author_declarations_and_original_compiler_version': 'unknown', 'player_parity_promoted': False}
    a.output.mkdir(parents=True, exist_ok=False)
    (a.output / 'result.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({k: v for k, v in result.items() if k != 'structural_examples'}, ensure_ascii=False))


if __name__ == '__main__':
    main()
