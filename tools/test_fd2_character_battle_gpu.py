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
    p.add_argument('--core-profile', choices=['blue', 'gold'], default='blue')
    p.add_argument('--other-core-profile', choices=['blue', 'gold'], default='blue')
    p.add_argument('--body-frames')
    p.add_argument('--sample-group',choices=['attack','end','fixture'],default='attack')
    for name in ['receipt', 'catalog', 'source', 'baseline', 'negative-receipt', 'out']:
        p.add_argument('--'+name, type=Path, required=True)
    args = p.parse_args()
    verifier = Path(__file__).with_name('verify_fd2_character_battle_gpu.py')
    baseline = json.loads(args.baseline.read_text())
    if (baseline.get('status') != ('PASS-CHARACTER-FIXTURE-GPU' if args.sample_group=='fixture' else 'PASS-CHARACTER-NORMAL-ATTACK-GPU')
            or baseline.get('figure') != args.figure
            or baseline.get('core_profile', 'blue') != args.core_profile
            or baseline.get('sample_group','attack') != args.sample_group
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
                '--core-profile', args.core_profile if figure is None else args.other_core_profile,
                '--receipt', str(path), '--catalog', str(args.catalog),
                '--source', str(source or args.source), '--report', str(report)]
        argv += ['--sample-group',args.sample_group]
        if args.body_frames and figure is None: argv += ['--body-frames', args.body_frames]
        result = subprocess.run(argv, capture_output=True, text=True, timeout=90)
        output = result.stdout+result.stderr
        if result.returncode == 0 or report.exists() or expected not in output:
            raise AssertionError((name, result.returncode, output))
        checks.append({'case': name, 'argv': argv, 'status': 'PASS-REJECTED',
                       'reason': output.strip().splitlines()[-1], 'output_published': False})

    wrong = copy.deepcopy(receipt); wrong['catalog_sha256'] = '0'*64
    path = args.out/'wrong-receipt.json'; path.write_text(json.dumps(wrong))
    check('wrong-receipt', path, '需要完整人物Game.Draw測試收據' if args.sample_group=='fixture' else '需要同份普通攻防完整GUI收據及清冊')
    if receipt.get('node_boundary_save_checked'):
        wrong = copy.deepcopy(receipt)
        wrong['source_save_bytes_equal'] = False
        wrong['cold_units'] = []
        path = args.out/'wrong-boundary-save.json'; path.write_text(json.dumps(wrong))
        check('wrong-boundary-save', path, '需要同份普通攻防完整GUI收據及清冊')
    attack_error='未在普通END GPU核對到所選人物的攻擊動作' if args.sample_group=='end' else '未在普通攻防GPU核對到所選人物的攻擊動作'
    check('wrong-figure', args.receipt, '需要完整人物Game.Draw測試收據' if args.sample_group=='fixture' else attack_error, figure=args.other_figure)
    shadow = args.out/'wrong-source'; shadow.mkdir()
    names = {name for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set' for name in a['files']}
    first = next(a for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set'
                 and a['source_resource'] == args.figure*3)['files'][0]
    other = next(name for name in sorted(names) if digest(args.source/name) != digest(args.source/first))
    for name in sorted(names):
        (shadow/name).symlink_to((args.source/(other if name == first else name)).resolve())
    check('wrong-source', args.receipt, '圖源雜湊不同', source=shadow)
    if args.sample_group=='end':
        folder=args.out/'wrong-end-translation'; folder.mkdir()
        wrong=copy.deepcopy(receipt)
        for sample in wrong['battle_end_art_samples']:
            for state in [sample['before'],sample['after']]:
                for layer in state.get('battle_source_layers') or []:
                    if layer['resource'] in [args.figure*3,args.figure*3+1]:layer['x']+=37
        for name in {s['file'] for s in wrong['battle_end_art_samples']}:
            if Path(name).name!=name:raise SystemExit('負向圖片路徑越界')
            (folder/name).symlink_to((args.receipt.parent/name).resolve())
        path=folder/'receipt.json'; path.write_text(json.dumps(wrong))
        check('wrong-end-translation',path,attack_error)
    if args.sample_group=='fixture':
        # 保留完整觀測圖層，但把實際PNG全部改成黑圖。
        # 回讀像素必須拒絕，不能只相信來源欄位與HD布林。
        from PIL import Image
        folder=args.out/'wrong-pixels'; folder.mkdir()
        wrong=copy.deepcopy(receipt)
        for shot in wrong['source_shots'].values():
            path=folder/shot['file']
            Image.new('RGBA',(1920,1200),(0,0,0,255)).save(path)
            shot['sha256']=digest(path)
        path=folder/'receipt.json'; path.write_text(json.dumps(wrong))
        check('wrong-pixels',path,'未核對到實際人物GPU圖源')
        wrong=copy.deepcopy(receipt); wrong['phase_f2']['counter']=0
        path=args.out/'missing-f2.json'; path.write_text(json.dumps(wrong))
        check('missing-f2',path,'需要完整人物Game.Draw測試收據')
        report={'status':'PASS-CHARACTER-GPU-GUARDS','sample_group':'fixture',
                'baseline_sha256':digest(args.baseline),'tool_sha256':digest(Path(__file__)),
                'checks':checks,'scope':'修改fixture的錯收據、人物、圖源、實際PNG及F2反例；非普通玩家驗收。'}
        (args.out/'guards.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
        print(json.dumps({'status':report['status'],'guards':len(checks)}))
        return
    old = json.loads(args.negative_receipt.read_text())
    old['catalog_sha256'] = digest(args.catalog)
    if args.sample_group=='end':
        old['battle_end_art_checked']=True
        old['battle_end_art_samples']=copy.deepcopy(old['battle_art_samples'])
        for sample in old['battle_end_art_samples']:
            for state in [sample['before'],sample['after']]:
                state['battle_source_layers']=[{'resource':a['source_resource'],'frame':i,'fill':-1,'archive':'FIGANI.DAT',
                    'x':f['x'],'y':f['y'],'width':f['width'],'height':f['height'],'clip':None}
                    for a in catalog['assets'] if a['role']=='battle_animation_hd_set' and a['source_resource'] in [args.figure*3,args.figure*3+1]
                    for i,f in enumerate(a['native_frames'])]
    # 明示偽造清冊指紋，以舊實際PNG反證不能只信hd_battle布林。
    folder = args.out/'old-capture-fixture'; folder.mkdir()
    path = folder/'receipt.json'; path.write_text(json.dumps(old))
    names = {s['file'] for s in old['battle_art_samples']}
    names.update(scene+'-physical-scene.png' for scene in ['world-hd', 'world-hd-replay'])
    for name in names:
        if Path(name).name != name: raise SystemExit('負向圖片路徑越界')
        (folder/name).symlink_to((args.negative_receipt.parent/name).resolve())
    check('old-capture', path, attack_error)
    report = {'status': 'PASS-CHARACTER-GPU-GUARDS', 'core_profile': args.core_profile,
              'other_core_profile': args.other_core_profile, 'baseline_sha256': digest(args.baseline),
              'tool_sha256': digest(Path(__file__)), 'checks': checks,
              'scope': '錯收據、人物、圖源、舊擷取及適用的章節邊界保存負向fixture；不新增原版或普通玩家收據。'}
    (args.out/'guards.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'guards': len(checks)}))


if __name__ == '__main__': main()
