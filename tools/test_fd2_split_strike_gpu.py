#!/usr/bin/env python3
"""以真實24／29畫面正對照核對生命週期、圖源與最終畫面拒收。"""
import argparse
import copy
import json
import os
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from verify_fd2_split_strike_gpu import verify


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('receipt', 'catalog', 'source', 'original-pack', 'inventory', 'out'):
        parser.add_argument('--'+name, type=Path, required=True)
    a = parser.parse_args()
    if not Path('/.dockerenv').exists() or a.out.exists() or a.out.parent.stat().st_uid != os.getuid():
        raise SystemExit('在Docker使用本人擁有的新輸出目錄')
    a.out.mkdir()
    call = [a.catalog, a.source, a.original_pack, a.inventory]
    positive = verify(a.receipt, *call)
    base = json.loads(a.receipt.read_text())
    (a.out/'positive.json').write_text(json.dumps(positive, ensure_ascii=False, indent=2)+'\n')
    results = []

    def reject(name, mutate, expected, corrupt=None):
        folder = a.out/name
        folder.mkdir()
        for shot in base['source_shots'].values():
            files = [shot['file'], shot['base_file']]+[l['native_file'] for l in shot['layers'] if 'native_file' in l]
            for filename in files:
                path = folder/filename
                if not path.exists():
                    path.symlink_to((a.receipt.parent/filename).resolve())
        doc = copy.deepcopy(base)
        mutate(doc)
        if corrupt:
            corrupt(doc, folder)
        path = folder/'receipt.json'
        path.write_text(json.dumps(doc))
        try:
            verify(path, *call)
        except ValueError as error:
            if expected not in str(error):
                raise AssertionError((name, str(error))) from error
            results.append({'case': name, 'status': 'PASS-REJECTED', 'reason': str(error)})
        else:
            raise AssertionError('反例被接受：'+name)

    for name, mutate in [
        ('wrong-command', lambda d: d.update(command_id=0)),
        ('missing-f2', lambda d: d['phase_f2'].update(actor=0)),
        ('missing-continuation', lambda d: d.update(continuations=0)),
        ('retained-owner', lambda d: d.update(owner_released=False)),
        ('changed-mp', lambda d: d.update(mp_after=d['mp_after']+1)),
        ('missing-hp', lambda d: d.update(hp_changes=[0])),
        ('extra-rng', lambda d: d.update(go_rng_planning_calls=7)),
        ('wrong-seed', lambda d: d.update(go_seed=999)),
    ]:
        reject(name, mutate, '需要24' if name == 'wrong-command' else '生命週期')
    first = next(iter(base['source_shots']))
    reject('wrong-screen-sha', lambda d: d['source_shots'][first].update(sha256='bad'), '圖片路徑或SHA')
    reject('changed-origin', lambda d: d['source_shots'][first]['layers'][0].update(dx=1), '目的矩形')
    reject('changed-descriptor', lambda d: d['source_shots'][first]['layers'][0].update(x=0, rect=[0, 0, 6, 6]), '目的矩形')
    reject('changed-hd-source', lambda d: d['source_shots'][first]['layers'][0].update(hd_sha256='bad'), 'HD來源SHA')
    reject('hidden-hd', lambda d: d['source_shots'][first]['layers'][0].update(clip=[0, 0, 0, 0]), '完整GPU')

    def missing_blank(doc):
        end = 14 if doc['command_id'] == 24 else 17
        doc['source_shots'] = {k: v for k, v in doc['source_shots'].items()
                               if not (v['phase'].startswith('target-') and v['cursor']['Effect'] == end)}
    reject('missing-native-blank', missing_blank, '來源幀或空拍缺少')
    reject('missing-transition', lambda d: d.update(source_shots={k: v for k, v in d['source_shots'].items()
           if not (v['phase'] == 'transition-in-0' and v['cursor']['Transition'] == 19)}), '來源幀或空拍缺少')

    def corrupt_screen(doc, folder):
        shot = doc['source_shots'][first]
        path = folder/shot['file']
        path.unlink()
        Image.new('RGB', (1920, 1200), (0, 255, 0)).save(path)
        shot['sha256'] = digest(path)
    reject('self-consistent-wrong-gpu', lambda d: None, '完整GPU', corrupt_screen)
    report = {'status': 'PASS-SPLIT-STRIKE-GPU-GUARDS', 'command_id': base['command_id'],
              'positive_sha256': digest(a.out/'positive.json'), 'checks': results,
              'scope': '真實最終GPU與固定原圖正對照的拒收，不提升正常玩家或原版parity。'}
    (a.out/'guards.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'cases': len(results)}))


if __name__ == '__main__':
    main()
