#!/usr/bin/env python3
"""依首關四個事件的 IDA 呼叫與原始文字，重生可編輯援軍演出。僅限 Docker。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path

from generate_native_story_dialogue import decode_layouts, mapping_for, parse_fdtxt
from import_editor_legacy import import_legacy, write_canonical

ROOT = Path(__file__).resolve().parents[1]
SHA = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"


def main():
    assert Path('/.dockerenv').exists(), '只允許在 Docker 內執行'
    ap = argparse.ArgumentParser()
    ap.add_argument('--ida', type=Path, required=True)
    ap.add_argument('--write', action='store_true')
    args = ap.parse_args()
    raw_evidence = args.ida.read_bytes()
    evidence = json.loads(raw_evidence)
    assert evidence['input']['sha256'] == SHA
    assert evidence['tool'] == 'IDA Pro 9.4'
    ins = {r['address']: r for f in evidence['functions'] for r in f['instructions']}
    # 保留跳尾來源，不能把事件 2／3 誤認成沒有對話。
    for addr, expected in {'0x34248': 'sub_15F84', '0x342a6': 'sub_15F84',
                           '0x34313': 'sub_15F84', '0x34375': 'loc_3430D',
                           '0x343dd': 'loc_3430D'}.items():
        assert expected in ins[addr]['instruction']
    glyph_path = ROOT / 'docs/data/glyph_map.json'
    glyphs = {int(k): v for k, v in json.loads(glyph_path.read_text()).items() if k != '_comment'}
    raw_path = ROOT / 'extracted/raw/FDTXT/FDTXT_001.bin'
    strings = parse_fdtxt(raw_path)
    mapping_path = ROOT / 'remake/assets/cutscenes/dialogue-index/count-aligned.json'
    mapping = json.loads(mapping_path.read_text())
    scenario_path = ROOT / 'remake/assets/scenarios/ch01.json'
    scenario = json.loads(scenario_path.read_text())
    events = {e['id']: e for e in scenario['events']}
    spawns = {a['groups'][0]: copy.deepcopy(a) for e in scenario['events'] for a in e['do']
              if a['type'] == 'spawn_group'}
    roster_path = ROOT / 'remake/assets/maps/map0/map0_units.json'
    roster = json.loads(roster_path.read_text())['units']
    father = [u for u in roster if u['group'] == 7]
    assert len(father) == 1 and father[0]['native_record_byte6'] == 2 and father[0]['camp'] == 'own'
    spawns[7]['camp'] = 'own'

    def action(kind, source, **kw):
        return dict(type=kind, native_source=source, **kw)

    def dialog(index, source, event_id):
        layouts = decode_layouts('FDTXT_001', index, strings[index], glyphs)
        entry, targets = mapping_for(mapping, 'FDTXT_001', 'ch01.json', index)
        refs = [(t['scene_index'], line) for t in targets for line in t['lines']]
        assert len(layouts) == entry['utterance_count'] == len(refs)
        return [action('dialogue', source, native_event_id=event_id, native_text_index=index,
                       native_dialogue_ref=dict(script='assets/story/ch01.json', scene_index=scene,
                                                line=line, **layout))
                for (scene, line), layout in zip(refs, layouts)]

    def acting(resource, source):
        return action('native_acting', source, native_acting=dict(resource=resource, source=source))

    events['hano_hawat_join']['do'] = [
        action('join_party', '0x341e8', char_id=1), spawns[3],
        action('pan', '0x341fe', grid=[5, 8]), action('redraw', '0x34208'),
        action('delay', '0x34212', ms=100), acting(7, '0x3421c'),
        *dialog(11, '0x34248', 0), action('native_range_zero', '0x34250'), spawns[7],
        action('redraw', '0x34266'), action('delay', '0x34270', ms=100), acting(8, '0x3427a'),
        *dialog(3, '0x342a6', 0), action('reset_pose', '0x342ae')]
    # 既有原生登場 wrapper 已包含各自的重繪與 following_acting，不能重播兩次。
    events['enemy_reinforce']['do'] = [action('pan', '0x342c4', grid=[11, 16]), spawns[4],
                                     action('reset_pose', '0x342ef'), *dialog(4, '0x34313', 1)]
    events['pirate_boss']['do'] = [action('pan', '0x3432c', grid=[0, 16]), spawns[5],
                                 action('reset_pose', '0x34357'), *dialog(5, '0x34375', 2)]
    events['coast_guard']['do'] = [action('pan', '0x34386', grid=[11, 11]), spawns[6],
                                 action('redraw', '0x343a8'), acting(6, '0x343b2'),
                                 action('reset_pose', '0x343ba'), *dialog(6, '0x343dd', 3)]
    for event_id, name in enumerate(['hano_hawat_join', 'enemy_reinforce', 'pirate_boss', 'coast_guard']):
        for step in events[name]['do']:
            step.setdefault('native_event_id', event_id)
    evidence['raw_export_sha256'] = hashlib.sha256(raw_evidence).hexdigest()
    evidence['scope'] = '首關四個回合事件的直接呼叫順序；不宣稱執行期逐像素或時序一致'
    evidence['reviewed_calls'] = [{**ins[a], 'grade': '已證實',
        'semantic': '原生文字呼叫／共用呼叫尾端', 'source': '同檔保留的直接指令與參數堆疊'}
        for a in ['0x34248', '0x342a6', '0x34313', '0x34375', '0x343dd']]
    evidence['data_inputs'] = [{'path': str(p.relative_to(ROOT)),
        'sha256': hashlib.sha256(p.read_bytes()).hexdigest()} for p in [raw_path, glyph_path, mapping_path, roster_path]]
    encoded = json.dumps(scenario, ensure_ascii=False, indent=2) + '\n'
    if args.write:
        assert scenario_path.stat().st_uid == 1000
        scenario_path.write_text(encoded)
        canonical_root = ROOT / 'remake/assets/editor-canonical'
        canonical_path = canonical_root / 'scenarios/ch01.json'
        summary_path = canonical_root / 'bundle-summary.json'
        assert canonical_path.stat().st_uid == summary_path.stat().st_uid == 1000
        document, diagnostics = import_legacy(scenario, 'remake/assets/scenarios/ch01.json', 'scenario')
        write_canonical(document, canonical_path)
        summary = json.loads(summary_path.read_text())
        entries = [d for d in summary['documents'] if d['output'] == 'scenarios/ch01.json']
        assert len(entries) == 1 and entries[0]['document_id'] == document['document_id']
        entries[0]['sha256'] = hashlib.sha256(canonical_path.read_bytes()).hexdigest()
        entries[0]['diagnostics'] = len(diagnostics)
        for d in summary['diagnostics']:
            if d.get('source') == 'remake/assets/scenarios/ch01.json': d['items'] = diagnostics
        summary_path.write_text(json.dumps(summary, ensure_ascii=False, sort_keys=True, indent=2) + '\n')
        dest = ROOT / 'docs/data/ida/fd2_ch01_turn_events_20260908.json'
        assert dest.parent.stat().st_uid == 1000
        dest.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + '\n')
    else:
        assert json.loads(scenario_path.read_text()) == scenario
    print(json.dumps(dict(dialogue_utterances=32, event_count=4, written=args.write)))


if __name__ == '__main__':
    main()
