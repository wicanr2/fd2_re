#!/usr/bin/env python3
"""沿正式序章 handler／binding／roster 解析 DATO 需求，不改遊戲資料。"""
import argparse
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def audit():
    sources = {}

    def read(path):
        data = (ROOT / path).read_bytes()
        sources[path] = hashlib.sha256(data).hexdigest()
        return json.loads(data)

    def asset(path):
        assert path.startswith('assets/') and '..' not in Path(path).parts
        return 'remake/' + path

    campaign = read('remake/assets/scenarios/campaign_full.json')
    node = campaign['nodes'][campaign['start']]
    binding_path = asset(node['handler_binding'])
    binding = read(binding_path)
    handler_path = str((ROOT / Path(binding_path).parent / binding['handler_script']).resolve().relative_to(ROOT))
    handler = read(handler_path)
    catalog = read('remake/assets/themes/modern/catalog.json')
    portraits = {a['speaker_id']: a['asset_id'] for a in catalog['assets']
                 if a['role'] == 'story_portrait_frame'}
    units, rows = [], []
    for beat in handler['beats']:
        address = beat['source']['addr']
        if beat['op'] == 'loadch':
            roster = binding['overrides'][address]['loadch']['roster']
            units = read(asset(roster))['units']
        if beat['op'] != 'dialog':
            continue
        mappings = [(k, v) for k, v in binding['dialogue_overrides'].items()
                    if k.split('#')[0] == address]
        assert len(mappings) == 1, (address, len(mappings))
        binding_key, mapping = mappings[0]
        story_path = 'remake/assets/story/' + mapping['script']
        story = read(story_path)
        scene = mapping['scene_index']
        for entry in mapping['lines']:
            layout = entry['native_dialogue']
            control, operand = layout['control'], layout['operand']
            if control in ('FFED', 'FFEC'):
                selected = units[operand]
                speaker = selected['battle_fig']
                method = 'runtime slot → battle_fig（raw +7）'
            else:
                assert control in ('FFEF', 'FFEE')
                matches = [u for u in units if u['native_record_byte8'] == operand
                           and not (u.get('native_record_byte5', 0) & 1)]
                speaker = matches[0]['battle_fig'] if matches else operand
                method = 'active raw +8 → raw +7' if matches else 'direct DATO operand'
            line = story['scenes'][scene]['lines'][entry['line']]
            rows.append({'handler_address': address, 'binding_key': binding_key,
                         'story': story_path, 'scene_index': scene, 'line': entry['line'],
                         'declared_speaker': line['speaker'], 'control': control,
                         'operand': operand, 'resolved_dato': speaker, 'method': method,
                         'asset_id': portraits.get(speaker), 'registered': speaker in portraits})
    consumers = ['remake/cmd/fd2/main.go', 'remake/cmd/fd2/native_story_dialogue.go',
                 'remake/cmd/fd2/modern_story_portrait.go']
    for path in consumers:
        sources[path] = hashlib.sha256((ROOT / path).read_bytes()).hexdigest()
    return {'schema_version': 1, 'kind': 'fd2-modern-intro-consumer-audit',
            'scope': 'campaign_full 正式起點 ch00_pre 的對話及 LOADCH roster；不外推其他章、動態入隊或城鎮。',
            'runtime_replay': False, 'sources': sources, 'rows': rows,
            'required_dato': sorted({r['resolved_dato'] for r in rows}),
            'missing_dato': sorted({r['resolved_dato'] for r in rows if not r['registered']}),
            'limits': '此為現有已證實 consumer 的資料稽核，不是新增原版逆向證據或完整 PLAYER-E2。'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').exists()
    report = audit()
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({k: v for k, v in report.items() if k not in ('rows', 'sources')}, ensure_ascii=False))


if __name__ == '__main__':
    main()
