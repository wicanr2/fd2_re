#!/usr/bin/env python3
"""核對已完成START、但尾端未完成的序章產物；明示分段證據，不補造整次PASS。"""
import argparse
import hashlib
import json
from pathlib import Path
import re

from PIL import Image, ImageChops


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('run', 'binary', 'catalog', 'source', 'report'):
        parser.add_argument('--'+name, type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').exists() and not args.report.exists()
    assert not (args.run/'hd-gui.json').exists(), 'use the complete receipt when present'
    log = args.run/'game.log'
    beats = len(re.findall(r'beat op=dialog ', log.read_text()))
    assert beats == 97
    audit = args.run/'input-audit.jsonl'
    with audit.open('rb') as stream:
        stream.seek(max(0, audit.stat().st_size-131072))
        final = json.loads(stream.read().splitlines()[-1])
    assert final['node'] == 'battle_ch01' and not final['error']
    operations = [json.loads(row) for row in (args.run/'operations.jsonl').read_text().splitlines()]
    assert any(o['key'] == 'Return' and o['state'].get('node') == 'story_ch00_handler' for o in operations)
    assert any(o['key'] == 'F5' and o['state'].get('node') == 'battle_ch01' for o in operations)
    catalog = json.loads(args.catalog.read_text())
    assets = {a['speaker_id']: a for a in catalog['assets'] if a['role'] == 'story_portrait_hd_frame'}
    screenshots = {}
    for speaker in (0, 4, 9, 30, 48, 66, 96):
        path = args.run/f'speaker-{speaker:03d}.png'
        actual = Image.open(path).convert('RGB')
        source = args.source/assets[speaker]['file']
        reference = Image.open(source).convert('RGB')
        assert actual.size == (1920, 1200) and reference.size == (480, 480)
        matches = [rect for rect in ((1392,30,1872,510), (48,690,528,1170))
                   if ImageChops.difference(actual.crop(rect), reference).getbbox() is None]
        assert len(matches) == 1, speaker
        screenshots[str(speaker)] = {'file': path.name, 'sha256': digest(path),
            'anchor': list(matches[0]), 'mae_normalized': 0.0, 'source_sha256': digest(source)}
    result = {'status': 'START-PASS-TAIL-INCOMPLETE', 'start_completed': True,
        'catalog_sha256': digest(args.catalog), 'binary_sha256': digest(args.binary),
        'dialogue_beats': beats, 'screenshots': screenshots, 'final_state': final,
        'game_log_sha256': digest(log), 'input_audit_sha256': digest(audit),
        'operations_sha256': digest(args.run/'operations.jsonl'),
        'verifier_sha256': digest(Path(__file__)), 'recordings_created': False,
        'tail_failure': 'F9短按鍵未取得完成ACK；原執行器失敗，沒有完整hd-gui.json。',
        'scope': '只證明既有正常START至首章的97拍與七名靜態頭像GPU；尾端保存讀取不由本報告宣稱通過。另用獨立完整戰場保存冷讀收據，不冒稱同次長流程全通。'}
    args.report.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print('START scope verified; tail remains incomplete and disclosed')


if __name__ == '__main__':
    main()
