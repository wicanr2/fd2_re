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
    p.add_argument('--sample-group',choices=['attack','end','fixture','idle-end','idle-fixture'],default='attack')
    for name in ['receipt', 'catalog', 'source', 'baseline', 'negative-receipt', 'out']:
        p.add_argument('--'+name, type=Path, required=True)
    args = p.parse_args()
    idle_only=args.sample_group in ('idle-end','idle-fixture')
    fixture_group=args.sample_group in ('fixture','idle-fixture')
    end_group=args.sample_group in ('end','idle-end')
    verifier = Path(__file__).with_name('verify_fd2_character_battle_gpu.py')
    baseline = json.loads(args.baseline.read_text())
    expected_status=('PASS-IDLE-TARGET-FIXTURE-GPU' if fixture_group else 'PASS-CHARACTER-NORMAL-IDLE-GPU') if idle_only else (
        'PASS-CHARACTER-FIXTURE-GPU' if fixture_group else 'PASS-CHARACTER-NORMAL-ATTACK-GPU')
    if (baseline.get('status') != expected_status
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

    def check(name, path, expected, figure=None, source=None, inventory=None):
        report = args.out/(name+'-report.json')
        argv = [sys.executable, str(verifier), '--figure', str(args.figure if figure is None else figure),
                '--core-profile', args.core_profile if figure is None else args.other_core_profile,
                '--receipt', str(path), '--catalog', str(args.catalog),
                '--source', str(source or args.source), '--report', str(report)]
        argv += ['--sample-group',args.sample_group]
        if inventory is not None:argv += ['--inventory',str(inventory)]
        if args.body_frames and figure is None: argv += ['--body-frames', args.body_frames]
        result = subprocess.run(argv, capture_output=True, text=True, timeout=90)
        output = result.stdout+result.stderr
        if result.returncode == 0 or report.exists() or expected not in output:
            raise AssertionError((name, result.returncode, output))
        checks.append({'case': name, 'argv': argv, 'status': 'PASS-REJECTED',
                       'reason': output.strip().splitlines()[-1], 'output_published': False})

    wrong = copy.deepcopy(receipt); wrong['catalog_sha256'] = '0'*64
    path = args.out/'wrong-receipt.json'; path.write_text(json.dumps(wrong))
    fixture_error='需要完整目標待機Game.Draw測試收據' if idle_only else '需要完整人物Game.Draw測試收據'
    check('wrong-receipt', path, fixture_error if fixture_group else '需要同份普通攻防完整GUI收據及清冊')
    if args.sample_group in ('attack', 'end', 'idle-end') and receipt.get('battle_repetitions') == 1:
        for name, updates in [
                ('missing-natural-return', {'natural_attack_return_checked': False}),
                ('missing-single-save', {'post_battle_save_checked': False}),
                ('extra-single-scene', {'battle_hd_scenes': receipt['battle_hd_scenes']+
                                       [{'name': 'world-hd-replay'}]}),
                ('invalid-repetitions', {'battle_repetitions': 3})]:
            wrong = copy.deepcopy(receipt); wrong.update(updates)
            path = args.out/(name+'.json'); path.write_text(json.dumps(wrong))
            check(name, path, '單場須有完整自然返回與保存收據')
    if receipt.get('node_boundary_save_checked'):
        wrong = copy.deepcopy(receipt)
        wrong['source_save_bytes_equal'] = False
        wrong['cold_units'] = []
        path = args.out/'wrong-boundary-save.json'; path.write_text(json.dumps(wrong))
        check('wrong-boundary-save', path, '需要同份普通攻防完整GUI收據及清冊')
    attack_error='未在普通END GPU核對到所選目標的待機' if idle_only else (
        '未在普通END GPU核對到所選人物的攻擊動作' if end_group else '未在普通攻防GPU核對到所選人物的攻擊動作')
    check('wrong-figure', args.receipt, fixture_error if fixture_group else (
        '需要明示待機目標的普通END擷取範圍' if idle_only else attack_error), figure=args.other_figure)
    shadow = args.out/'wrong-source'; shadow.mkdir()
    names = {name for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set' for name in a['files']}
    first = next(a for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set'
                 and a['source_resource'] == args.figure*3)['files'][0]
    other = next(name for name in sorted(names) if digest(args.source/name) != digest(args.source/first))
    for name in sorted(names):
        (shadow/name).symlink_to((args.source/(other if name == first else name)).resolve())
    check('wrong-source', args.receipt, '圖源雜湊不同', source=shadow)
    if idle_only:
        inventory_path=ROOT/'docs/data/fd2-battle-hd-source-inventory-20261019.json'
        inventory=json.loads(inventory_path.read_text())
        next(r for r in inventory['resources'] if r['resource']==args.figure*3+1)['status']='decoded'
        path=args.out/'wrong-empty-proof.json';path.write_text(json.dumps(inventory))
        check('wrong-empty-proof',args.receipt,'需要原版僅待機及兩項固定零header證據',inventory=path)
        banks={a['source_resource'] for a in catalog['assets'] if a['role']=='battle_animation_hd_set'}
        paired=next(r for r in sorted(banks) if r%3==0 and r+1 in banks)//3
        wrong=copy.deepcopy(receipt)
        if fixture_group:
            wrong.update(figure=paired,selected_resources=[paired*3])
        else:
            wrong['battle_end_figures']=sorted(set(wrong['battle_end_figures']+[paired]))
        path=args.out/'paired-bank-as-idle.json';path.write_text(json.dumps(wrong))
        check('paired-bank-as-idle',path,'需要原版僅待機及兩項固定零header證據',figure=paired)
        if fixture_group:
            for name,values in [('wrong-idle-role',{'role':'attack'}),
                                ('invented-counter',{'counter_present':True}),
                                ('invented-attack-bank',{'selected_resources':[args.figure*3,args.figure*3+1]})]:
                wrong=copy.deepcopy(receipt);wrong.update(values)
                path=args.out/(name+'.json');path.write_text(json.dumps(wrong))
                check(name,path,'idle-target不得冒用攻擊／反擊測試範圍')
            wrong=copy.deepcopy(receipt);wrong['phase_hd_draws']['counter']=1
            path=args.out/'invented-counter-phase.json';path.write_text(json.dumps(wrong))
            check('invented-counter-phase',path,'idle-target不得冒用攻擊／反擊測試範圍')
            wrong=copy.deepcopy(receipt)
            shot=next(iter(wrong['source_shots'].values()))
            shot['before']['battle_source_layers'].append({'resource':args.figure*3+1,'frame':0,'fill':-1,'archive':'FIGANI.DAT'})
            path=args.out/'invented-empty-layer.json';path.write_text(json.dumps(wrong))
            check('invented-empty-layer',path,'idle-target收據包含原版空圖庫的呈現圖層')
        else:
            wrong=copy.deepcopy(receipt);wrong['battle_end_art_samples']=[]
            path=args.out/'missing-end-pixels.json';path.write_text(json.dumps(wrong))
            check('missing-end-pixels',path,attack_error)
            wrong=copy.deepcopy(receipt);wrong['battle_end_figures']=[]
            path=args.out/'missing-idle-range.json';path.write_text(json.dumps(wrong))
            check('missing-idle-range',path,'需要明示待機目標的普通END擷取範圍')
    if end_group:
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
    if fixture_group:
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
        wrong=copy.deepcopy(receipt); wrong['phase_f2']['body' if idle_only else 'counter']=0
        path=args.out/'missing-f2.json'; path.write_text(json.dumps(wrong))
        check('missing-f2',path,fixture_error)
        report={'status':'PASS-CHARACTER-GPU-GUARDS','sample_group':args.sample_group,
                'baseline_sha256':digest(args.baseline),'tool_sha256':digest(Path(__file__)),
                'checks':checks,'scope':'修改fixture的錯收據、人物、圖源、實際PNG及F2反例；非普通玩家驗收。'}
        (args.out/'guards.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
        print(json.dumps({'status':report['status'],'guards':len(checks)}))
        return
    old = json.loads(args.negative_receipt.read_text())
    old['catalog_sha256'] = digest(args.catalog)
    if end_group:
        old['battle_end_art_checked']=True
        if idle_only:old['battle_end_figures']=[args.figure]
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
