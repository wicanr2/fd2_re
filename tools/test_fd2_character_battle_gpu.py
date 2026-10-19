#!/usr/bin/env python3
"""用實際GUI收據驗證人物HD核對器拒絕錯配，不發布負向報告。"""
import argparse
import copy
import json
from pathlib import Path
import subprocess
import sys

from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import ROOT


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--figure', type=int, required=True)
    p.add_argument('--other-figure', type=int, required=True)
    for name in ['receipt', 'catalog', 'source', 'baseline', 'negative-receipt', 'out']:
        p.add_argument('--'+name, type=Path, required=True)
    args = p.parse_args()
    verifier = Path(__file__).with_name('verify_fd2_character_battle_gpu.py')
    baseline = json.loads(args.baseline.read_text())
    if (baseline.get('status') != 'PASS-CHARACTER-NORMAL-ATTACK-GPU'
            or baseline.get('figure') != args.figure
            or baseline.get('receipt_sha256') != digest(args.receipt)
            or baseline.get('catalog_sha256') != digest(args.catalog)
            or baseline.get('verifier_sha256') != digest(verifier)):
        raise SystemExit('需要目前核對器的實檔正向報告')
    if args.out.exists() or not args.out.resolve().is_relative_to(ROOT):
        raise SystemExit('使用工作區新守衛目錄')
    args.out.mkdir()
    receipt = json.loads(args.receipt.read_text())
    catalog = json.loads(args.catalog.read_text())
    checks = []

    def check(name, path, expected, figure=None, source=None):
        report = args.out/(name+'-report.json')
        argv = [sys.executable, str(verifier), '--figure', str(args.figure if figure is None else figure),
                '--receipt', str(path), '--catalog', str(args.catalog),
                '--source', str(source or args.source), '--report', str(report)]
        result = subprocess.run(argv, capture_output=True, text=True, timeout=90)
        output = result.stdout+result.stderr
        if result.returncode == 0 or report.exists() or expected not in output:
            raise AssertionError((name, result.returncode, output))
        checks.append({'case': name, 'argv': argv, 'status': 'PASS-REJECTED',
                       'reason': output.strip().splitlines()[-1], 'output_published': False})

    wrong = copy.deepcopy(receipt); wrong['catalog_sha256'] = '0'*64
    path = args.out/'wrong-receipt.json'; path.write_text(json.dumps(wrong))
    check('wrong-receipt', path, '需要同份普通首章完整GUI收據及清冊')
    check('wrong-figure', args.receipt, '未在普通攻防GPU核對到所選人物的攻擊動作', figure=args.other_figure)
    shadow = args.out/'wrong-source'; shadow.mkdir()
    names = {name for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set' for name in a['files']}
    first = next(a for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set'
                 and a['source_resource'] == args.figure*3)['files'][0]
    other = next(name for name in sorted(names) if digest(args.source/name) != digest(args.source/first))
    for name in sorted(names):
        (shadow/name).symlink_to((args.source/(other if name == first else name)).resolve())
    check('wrong-source', args.receipt, '圖源雜湊不同', source=shadow)
    old = json.loads(args.negative_receipt.read_text())
    old['catalog_sha256'] = digest(args.catalog)
    # 明示偽造清冊指紋，以舊實際PNG反證不能只信hd_battle布林。
    folder = args.out/'old-capture-fixture'; folder.mkdir()
    path = folder/'receipt.json'; path.write_text(json.dumps(old))
    names = {s['file'] for s in old['battle_art_samples']}
    names.update(scene+'-physical-scene.png' for scene in ['world-hd', 'world-hd-replay'])
    for name in names:
        if Path(name).name != name: raise SystemExit('負向圖片路徑越界')
        (folder/name).symlink_to((args.negative_receipt.parent/name).resolve())
    check('old-capture', path, '未在普通攻防GPU核對到所選人物的攻擊動作')
    report = {'status': 'PASS-CHARACTER-GPU-GUARDS', 'baseline_sha256': digest(args.baseline),
              'tool_sha256': digest(Path(__file__)), 'checks': checks,
              'scope': '四種明示負向fixture，舊擷取只證拒收；不新增原版或普通玩家收據。'}
    (args.out/'guards.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'guards': len(checks)}))


if __name__ == '__main__': main()
