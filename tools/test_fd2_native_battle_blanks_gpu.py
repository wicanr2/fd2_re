#!/usr/bin/env python3
"""以真實空拍Draw正對照驗證缺拍、錯來源與不完整生命週期均拒收。"""
import argparse
import copy
import json
from pathlib import Path

from build_fd2_gaia_battle_hd import digest
from verify_fd2_native_battle_blanks_gpu import verify


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--resource', type=int, required=True)
    for name in ['receipt', 'catalog', 'source', 'original-pack', 'inventory', 'out']:
        p.add_argument('--'+name, type=Path, required=True)
    a = p.parse_args()
    if not Path('/.dockerenv').exists() or a.out.exists():
        raise SystemExit('在Docker使用新輸出目錄')
    a.out.mkdir()
    args = [a.catalog, a.source, a.original_pack, a.inventory, a.resource]
    positive = verify(a.receipt, *args)
    base = json.loads(a.receipt.read_text())
    checks = []

    def reject(name, doc, expected, inventory=None):
        folder = a.out/name
        folder.mkdir()
        for shot in base['source_shots'].values():
            target = folder/shot['file']
            if not target.exists():
                target.symlink_to((a.receipt.parent/shot['file']).resolve())
        receipt = folder/'receipt.json'
        receipt.write_text(json.dumps(doc))
        call = list(args)
        if inventory:
            call[3] = inventory
        try:
            verify(receipt, *call)
        except ValueError as error:
            if expected not in str(error):
                raise AssertionError((name, str(error))) from error
            checks.append({'case': name, 'status': 'PASS-REJECTED', 'reason': str(error)})
        else:
            raise AssertionError(name+' accepted')

    for name, mutate, expected in [
        ('wrong-figure', lambda d: d.update(figure=-1), '完整Game.Draw'),
        ('missing-return', lambda d: d.update(map_dac_steps=64), '完整Game.Draw'),
        ('missing-body-f2', lambda d: d['phase_f2'].update(body=0), '完整Game.Draw'),
        ('wrong-gpu-sha', lambda d: next(s for s in d['source_shots'].values()
                                       if s.get('phase') == 'body').update(sha256='bad'), 'GPU圖片'),
    ]:
        doc = copy.deepcopy(base)
        mutate(doc)
        reject(name, doc, expected)
    for match in positive['matches']:
        index = match['frame']
        doc = copy.deepcopy(base)
        doc['source_shots'] = {k: v for k, v in doc['source_shots'].items() if not any(
            layer.get('resource') == a.resource and layer.get('frame') == index
            for key in ('before', 'after') for layer in v.get(key, {}).get('battle_source_layers') or [])}
        reject(f'missing-{index}', doc, '同次Draw不完整')
    doc = copy.deepcopy(base)
    for shot in doc['source_shots'].values():
        if shot.get('phase') == 'body':
            shot['before']['hd_battle'] = False
    reject('wrong-hd-mode', doc, '需要HD戰鬥')
    wrong_inventory = a.out/'wrong-inventory.json'
    wrong_inventory.write_text('{}')
    reject('wrong-inventory', copy.deepcopy(base), '固定原始盤點不同', wrong_inventory)
    (a.out/'guards.json').write_text(json.dumps({'status': 'PASS-NATIVE-BLANK-DRAW-GUARDS',
        'positive': positive, 'checks': checks, 'test_sha256': digest(Path(__file__))}, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': 'PASS-NATIVE-BLANK-DRAW-GUARDS', 'cases': len(checks)}))


if __name__ == '__main__':
    main()
