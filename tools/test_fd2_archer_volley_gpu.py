#!/usr/bin/env python3
"""以固定原圖及實際Draw正對照核對飛矢驗證器的拒收條件。"""
import argparse
import copy
import json
import os
from pathlib import Path
import subprocess
import sys

from PIL import Image
from build_fd2_gaia_battle_hd import digest


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['receipt', 'catalog', 'source', 'original-pack', 'inventory', 'out']:
        p.add_argument('--'+name, type=Path, required=True)
    args = p.parse_args()
    if not Path('/.dockerenv').exists() or args.out.exists() or args.out.parent.stat().st_uid != os.getuid():
        raise SystemExit('在Docker使用本人擁有的新輸出目錄')
    args.out.mkdir()
    verifier = Path(__file__).with_name('verify_fd2_archer_volley_gpu.py')
    base = json.loads(args.receipt.read_text())
    checks = []

    def run(receipt, report, source=None, catalog=None, inventory=None):
        return subprocess.run([sys.executable, str(verifier), '--receipt', str(receipt), '--report', str(report),
            '--source', str(source or args.source), '--catalog', str(catalog or args.catalog),
            '--inventory', str(inventory or args.inventory), '--original-pack', str(args.original_pack)],
            capture_output=True, text=True, timeout=90)

    positive = args.out/'positive-control.json'
    result = run(args.receipt, positive)
    if result.returncode != 0 or not positive.is_file():
        raise AssertionError('有效正對照未通過：'+result.stdout+result.stderr)
    actual = json.loads(positive.read_text())
    if actual['status'] != 'PASS-PINNED-ARCHER-VOLLEY-GPU' or len(actual['matches']) != 9:
        raise AssertionError('正對照範圍不完整')

    def reject(name, doc, expected, folder=None, **kwargs):
        folder = folder or args.out
        receipt, report = folder/(name+'.json'), folder/(name+'-report.json')
        receipt.write_text(json.dumps(doc))
        result = run(receipt, report, **kwargs)
        if result.returncode == 0 or report.exists() or expected not in result.stdout+result.stderr:
            raise AssertionError((name, result.returncode, result.stdout, result.stderr))
        checks.append({'case': name, 'status': 'PASS-REJECTED', 'reason': (result.stdout+result.stderr).strip().splitlines()[-1]})

    for name, mutate in [('wrong-figure', lambda d:d.update(figure=7)),
                         ('missing-departure-f2', lambda d:d['phase_f2'].update(departure=0)),
                         ('missing-scroll-in-f2', lambda d:d['phase_f2'].update({'scroll-in': 0})),
                         ('missing-scroll-out-f2', lambda d:d['phase_f2'].update({'scroll-out': 0})),
                         ('missing-return', lambda d:d.update(map_dac_steps=64))]:
        d = copy.deepcopy(base); mutate(d)
        reject(name, d, '需要完整弓手')

    def contains(sample, index):
        return any(layer['resource'] == 70 and layer['frame'] == index
                   for key in ('before', 'after') for layer in sample.get(key, {}).get('battle_source_layers') or [])

    # Valid receipt-relative images are required even in the missing-frame case.
    def links(folder):
        folder.mkdir()
        for sample in base['source_shots'].values():
            name = sample['file']; target = folder/name
            if not target.exists():target.symlink_to((args.receipt.parent/name).resolve())

    for index in (18, 24):
        d = copy.deepcopy(base); d['source_shots'] = {k:v for k,v in d['source_shots'].items() if not contains(v, index)}
        folder = args.out/f'missing-{index}'; links(folder)
        reject(f'missing-frame-{index}', d, '最終Draw不完整', folder)

    folder = args.out/'wrong-pixels'; links(folder)
    d = copy.deepcopy(base); sample = next(s for s in d['source_shots'].values() if contains(s, 17))
    path = folder/sample['file']; path.unlink(); Image.new('RGB', (1920, 1200), (0, 255, 0)).save(path); sample['sha256'] = digest(path)
    reject('wrong-pixels', d, '最終Draw不完整', folder)

    wrong_inventory = args.out/'wrong-inventory.json'; inv = json.loads(args.inventory.read_text()); inv['status'] = 'forged'; wrong_inventory.write_text(json.dumps(inv))
    reject('wrong-inventory', copy.deepcopy(base), '飛矢固定盤點不同', inventory=wrong_inventory)

    wrong_source = args.out/'wrong-source'; wrong_source.mkdir()
    for path in args.source.iterdir():
        if path.is_file():(wrong_source/path.name).symlink_to(path.resolve())
    catalog = json.loads(args.catalog.read_text()); asset = next(a for a in catalog['assets'] if a.get('role') == 'battle_animation_hd_set' and a['source_resource'] == 70)
    path = wrong_source/asset['files'][16]; path.unlink(); Image.new('RGBA', (6, 6), (255, 255, 255, 255)).save(path)
    reject('wrong-source-sha', copy.deepcopy(base), '飛矢成品SHA不同', source=wrong_source)
    asset['frame_sha256'][16] = digest(path); wrong_catalog = args.out/'wrong-catalog.json'; wrong_catalog.write_text(json.dumps(catalog)); d = copy.deepcopy(base); d['catalog_sha256'] = digest(wrong_catalog)
    reject('self-consistent-wrong-dimensions', d, '飛矢成品形狀不同', source=wrong_source, catalog=wrong_catalog)
    report = {'status': 'PASS-ARCHER-VOLLEY-GPU-GUARDS', 'positive_control': digest(positive),
              'checks': checks, 'scope': '固定飛矢／空拍及真實Draw正對照的拒收；不代表普通玩家或原版parity。'}
    (args.out/'guards.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'cases': len(checks)}))


if __name__ == '__main__':
    main()
