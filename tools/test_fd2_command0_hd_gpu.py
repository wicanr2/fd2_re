#!/usr/bin/env python3
"""以同次實檔收據確認普通指令0核對器拒收偽造範圍與錯圖。"""
import argparse
import copy
import json
import os
from pathlib import Path
import subprocess
import sys

from PIL import Image


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['receipt', 'catalog', 'source', 'original-pack', 'inventory', 'out']:
        p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--normal', action='store_true')
    args = p.parse_args()
    if not Path('/.dockerenv').exists() or args.out.exists() or args.out.parent.stat().st_uid != os.getuid():
        raise SystemExit('在Docker使用本人擁有的新測試目錄')
    args.out.mkdir()
    receipt = json.loads(args.receipt.read_text())
    verifier = Path(__file__).with_name('verify_fd2_compound_caster_gpu.py')
    checks = []

    def command_for(receipt_path, report_path):
        return [sys.executable, str(verifier), '--normal-command0' if args.normal else '--command0',
                '--figure', '9', '--core-profile', 'gold', '--receipt', str(receipt_path),
                '--catalog', str(args.catalog), '--source', str(args.source),
                '--original-pack', str(args.original_pack), '--inventory', str(args.inventory),
                '--report', str(report_path)]

    # A broken path or environment must not make every negative case look green.
    positive_path = args.out/'positive-control.json'
    positive = subprocess.run(command_for(args.receipt, positive_path), capture_output=True, text=True, timeout=90)
    if positive.returncode != 0 or not positive_path.is_file():
        raise AssertionError('有效基底未通過，停止反例：'+positive.stdout+positive.stderr)

    def reject(name, mutate, blank=False):
        folder = args.out/name
        folder.mkdir()
        doc = copy.deepcopy(receipt)
        mutate(doc)
        for path in args.receipt.parent.glob('*.png'):
            if blank:
                Image.new('RGB', (1920, 1200), (0, 0, 0)).save(folder/path.name)
            else:
                (folder/path.name).symlink_to(path.resolve())
        target = folder/'receipt.json'
        target.write_text(json.dumps(doc))
        command = command_for(target, folder/'report.json')
        result = subprocess.run(command, capture_output=True, text=True, timeout=90)
        if result.returncode == 0 or (folder/'report.json').exists():
            raise AssertionError('錯誤收據被接受：'+name)
        reason = (result.stdout+result.stderr).strip().splitlines()[-1]
        expected = '需要完整普通指令0及邊界保存收據' if args.normal else '需要完整普通指令0交易／F2收據'
        if name == 'wrong-status' and not args.normal:
            expected = '需要完整RunGame交易收據'
        elif name == 'blank-gpu':
            expected = '普通GPU圖片路徑或SHA不同' if args.normal else '缺少必要施術者階段的實際HD核心或指定F2'
        elif name in ['no-samples', 'missing-actor']:
            expected = '缺少必要施術者階段的實際HD核心或指定F2'
        if expected not in reason:
            raise AssertionError((name, '非預期拒收原因', reason))
        checks.append({'case': name, 'status': 'PASS-REJECTED', 'reason': reason})

    reject('wrong-command', lambda d:d.update(battle_command_id=6) if args.normal else d.update(command_id=6))
    reject('wrong-status', lambda d:d.update(status='OBSERVATION-ONLY'))
    reject('incomplete-transaction', lambda d:d.update(cold_loaded_units_equal=False) if args.normal else d.update(hp_publications=6))
    reject('incomplete-f2-or-return', lambda d:d.update(natural_attack_return_checked=False) if args.normal else d['phase_f2_switches'].update(target=0))
    reject('blank-gpu', lambda d:None, blank=True)
    if args.normal:
        reject('no-samples', lambda d:d.update(battle_art_samples=[]))
    else:
        reject('missing-actor', lambda d:d.update(source_shots={k:v for k,v in d['source_shots'].items() if v['phase']!='actor'}))
    report = {'status': 'PASS-COMMAND0-HD-GPU-GUARDS', 'normal_player_receipt': args.normal,
              'positive_control_passed': True,
              'checks': checks, 'scope': '保留原RGB／alpha／至少1000點門檻；拒收錯範圍、缺交易／F2或自然返回、錯圖與缺圖。'}
    (args.out/'report.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'cases': len(checks)}))


if __name__ == '__main__':
    main()
