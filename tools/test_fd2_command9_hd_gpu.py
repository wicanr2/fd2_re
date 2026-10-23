#!/usr/bin/env python3
"""以成功實檔正對照確認指令9 GPU核對器拒收缺幀、錯圖及偽造交易。"""
import argparse
import copy
import json
import os
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from verify_fd2_command9_hd_gpu import verify


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for key in ['receipt', 'catalog', 'source', 'original-pack', 'inventory', 'out']:
        p.add_argument('--'+key, type=Path, required=True)
    args = p.parse_args()
    if not Path('/.dockerenv').exists() or args.out.exists() or args.out.parent.stat().st_uid != os.getuid():
        raise SystemExit('在Docker使用本人擁有的新測試目錄')
    args.out.mkdir()
    positive = verify(args.receipt, args.catalog, args.source, args.original_pack, args.inventory)
    (args.out/'positive.json').write_text(json.dumps(positive, ensure_ascii=False, indent=2)+'\n')
    original = json.loads(args.receipt.read_text()); rejected = []
    faults = ['status', 'command', 'hp', 'mp', 'rng', 'callback', 'f2', 'missing-handler', 'renamed-handler', 'wrong-handler-phase', 'wrong-gpu', 'wrong-descriptor', 'unsafe-png', 'unbound-resource', 'blank-source-dropped']
    for fault in faults:
        d = copy.deepcopy(original); folder = args.out/fault; folder.mkdir()
        for name in d['source_shots']:
            (folder/name).symlink_to(args.receipt.parent/name)
        if fault == 'status': d['status'] = 'PASS'
        elif fault == 'command': d['command_id'] = 8
        elif fault == 'hp': d['hp_publications'] = 19
        elif fault == 'mp': d['mp_publications'] = 2
        elif fault == 'rng': d['go_rng_unchanged'] = False
        elif fault == 'callback': d['continuations'] = 2
        elif fault == 'f2': d['phase_f2_switches']['command9-target'] = 0
        elif fault == 'missing-handler': del d['source_shots']['command9-target-090.png']
        elif fault == 'renamed-handler': d['source_shots']['duplicate.png'] = d['source_shots'].pop('command9-target-090.png')
        elif fault == 'wrong-handler-phase': d['source_shots']['command9-target-090.png']['phase'] = 'tail'
        else:
            first = next(iter(d['source_shots'])); sample = d['source_shots'][first]
            if fault == 'wrong-gpu':
                path = folder/first; path.unlink(); Image.new('RGB', (1920,1200), 'black').save(path)
            elif fault == 'wrong-descriptor': sample['layers'][0]['width'] += 1
            elif fault == 'unsafe-png': d['source_shots']['../escape.png'] = d['source_shots'].pop(first)
            elif fault == 'unbound-resource': sample['layers'][0]['resource'] = 408
            elif fault == 'blank-source-dropped': d['source_shots']['command9-target-090.png']['layers'].pop()
        path = folder/'receipt.json'; path.write_text(json.dumps(d))
        try:
            verify(path, args.catalog, args.source, args.original_pack, args.inventory)
        except ValueError as e:
            rejected.append({'fault': fault, 'reason': str(e)})
        else:
            raise ValueError('錯誤收據未拒收：'+fault)
    report = {'status': 'PASS-COMMAND9-GPU-VERIFIER-REFUSAL', 'positive': digest(args.out/'positive.json'), 'negative_controls': rejected,
              'verifier_sha256': digest(Path(__file__).with_name('verify_fd2_command9_hd_gpu.py')), 'test_sha256': digest(Path(__file__))}
    (args.out/'receipt.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'negative_controls': len(rejected)}))


if __name__ == '__main__': main()
