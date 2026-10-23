#!/usr/bin/env python3
"""以已通過的實檔候選驗證錯配方／錯來源不發布部分圖源。"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys


ROOT = Path(__file__).resolve().parent.parent


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['pack', 'inventory', 'recipe', 'baseline', 'out']:
        p.add_argument('--'+name, required=True, type=Path)
    p.add_argument('--base-catalog',type=Path,help='省略時沿基底收據旁的base-catalog.json，不拿後續正式登錄干擾舊配方')
    args = p.parse_args(); out = args.out.resolve(); compiler = Path(__file__).with_name('build_fd2_character_battle_hd.py')
    if not Path('/.dockerenv').exists() or out.exists() or not out.is_relative_to(ROOT) or out.parent.stat().st_uid != os.getuid():
        raise SystemExit('在Docker使用本人擁有的工作區新測試目錄')
    baseline = json.loads(args.baseline.read_text()); recipe = json.loads(args.recipe.read_text())
    if baseline['status'] != 'PASS-COMPILED-CHARACTER-CANDIDATE' or baseline['compiler_sha256'] != hashlib.sha256(compiler.read_bytes()).hexdigest():
        raise SystemExit('需要目前編製器已通過的實檔候選')
    if baseline['recipe_sha256'] != hashlib.sha256(args.recipe.read_bytes()).hexdigest():
        raise SystemExit('基底與配方不同')
    out.mkdir(); resources = list(recipe['resources']); first = resources[0]; attack = resources[-1]; source = next(iter(recipe['sources']))
    idle_only=len(resources)==1
    with_cast=baseline.get('with_cast') is True
    layered=baseline.get('layered') is True
    base_catalog=args.base_catalog or args.baseline.parent/'base-catalog.json'
    if not base_catalog.is_file():raise SystemExit('需要同次編製的基底清冊快照')
    checks = []

    def check(name, mutate, expected, pack=None, base=None, inventory=None, cast_flag=None, layer_flag=None):
        doc = copy.deepcopy(recipe); mutate(doc); path = out/(name+'.json'); path.write_text(json.dumps(doc))
        target = out/(name+'-candidate')
        command = [sys.executable, str(compiler), '--pack', str(pack or args.pack), '--inventory', str(inventory or args.inventory),
                   '--recipe', str(path), '--out', str(target)]
        command += ['--base-catalog', str(base or base_catalog)]
        if (with_cast if cast_flag is None else cast_flag):
            command.append('--with-cast')
        if (layered if layer_flag is None else layer_flag):
            command.append('--layered')
        result = subprocess.run(command, capture_output=True, text=True, timeout=60)
        output = result.stdout+result.stderr
        if result.returncode == 0 or target.exists() or expected not in output:
            raise AssertionError((name, result.returncode, output))
        checks.append({'case': name, 'status': 'PASS-REJECTED', 'reason': output.strip().splitlines()[-1], 'output_published': False})

    check('schema', lambda d:d.update(schema_version=2), '未知配方版本')
    check('inventory-sha', lambda d:d.update(inventory_sha256='0'*64), '盤點、配方或原版包指紋不符')
    check('source-sha', lambda d:d['sources'][source].update(sha256='0'*64), '母稿路徑、擁有權或SHA不符')
    check('source-anchor', lambda d:d['sources'][source].update(anchor=[0,0,1000000,1000000]), '錨點須是畫布內')
    def layer(d, resource, index=0):
        value = d['resources'][resource][index]
        return value[0] if isinstance(value, list) else value

    check('native-anchor', lambda d:layer(d,first).update(anchor=[0,0,1000000,1000000]), '錨點須是畫布內')
    check('missing-frame', lambda d:d['resources'][attack].pop(), '配方影格不完整')
    check('missing-resource', lambda d:d['resources'].pop(attack), '配方須涵蓋完整待機及攻擊資源')
    check('unknown-pose', lambda d:layer(d,attack).update(pose='unknown'), '未知母稿、群組或配方欄位')
    check('unknown-field', lambda d:layer(d,attack).update(mystery=True), '未知母稿、群組或配方欄位')
    check('already-registered', lambda d:None, '所選圖源已登錄', base=args.baseline.parent/'catalog.json')
    if with_cast:
        check('empty-layers',lambda d:d['resources'][first].__setitem__(0,[]),'分層配方須明示三組模式並含非空圖層')
        check('invalid-layer',lambda d:d['resources'][first].__setitem__(0,[None]),'未知母稿、群組或配方欄位')
        check('missing-attack', lambda d:d['resources'].pop(resources[1]), '配方須涵蓋完整待機及攻擊資源')
        check('missing-with-cast-flag', lambda d:None, '配方須涵蓋完整待機及攻擊資源', cast_flag=False)
        proof=json.loads(args.inventory.read_text())
        cast_id=int(resources[-1])
        next(r for r in proof['resources'] if r['resource']==cast_id)['status']='empty_header_zero'
        path=out/'nondecoded-cast-inventory.json';path.write_text(json.dumps(proof))
        check('nondecoded-cast',lambda d:d.update(inventory_sha256=hashlib.sha256(path.read_bytes()).hexdigest()),
              '三組配方必須全部已解碼',inventory=path)
    if layered and not with_cast:
        check('empty-layers',lambda d:d['resources'][first].__setitem__(0,[]),'分層配方須明示三組模式並含非空圖層')
        check('invalid-layer',lambda d:d['resources'][first].__setitem__(0,[None]),'未知母稿、群組或配方欄位')
        def require_layered_flag(d):
            # The flag permits layers; a plain assignment remains valid without it.
            value = d['resources'][first][0]
            if not isinstance(value, list):
                d['resources'][first][0] = [value]
        check('missing-layered-flag',require_layered_flag,'分層配方須明示三組模式並含非空圖層',layer_flag=False)
        check('two-banks-not-cast',lambda d:None,'配方須涵蓋完整待機及攻擊資源',cast_flag=True)
        # 中間層可沿既有母稿契約接受250；完整圖仍必須含真正不透明核心。
        from PIL import Image
        import math
        soft = Image.new('RGBA', (128, 128), (255, 255, 128, 0))
        alpha = Image.new('L', soft.size)
        alpha.putdata([round(125*(1-math.cos(math.pi*min(x,y,127-x,127-y,16)/16)))
                       for y in range(128) for x in range(128)])
        soft.putalpha(alpha); soft_path = out/'soft-only.png'; soft.save(soft_path)
        assert alpha.getextrema() == (0, 250)
        def missing_final_core(d):
            # Use the full idle subject, not a tiny FX fragment whose downsample
            # can overshoot the soft source to 255 through normal LANCZOS.
            # 純FX人物的idle可為固定原空白；空白本來就不要求核心。
            # 此反例必須改到已編製的非空幀，才是在核對非空255門檻。
            frame = next(row for row in baseline['frames'] if not row.get('native_empty_source'))
            resource,index = str(frame['resource']),frame['frame']
            target = layer(d,resource,index)['anchor']
            d['resources'][resource][index] = [{'pose':'soft_only','anchor':target,'scale_group':'soft_only'}]
            d['sources']['soft_only'] = {'file':str(soft_path),'sha256':hashlib.sha256(soft_path.read_bytes()).hexdigest(),
                                        'anchor':[0,0,128,128]}
            used = {part['pose'] for frames in d['resources'].values() for value in frames
                    for part in (value if isinstance(value,list) else [value])}
            d['sources'] = {key:value for key,value in d['sources'].items() if key in used}
        check('missing-final-opaque-core',missing_final_core,'分層成品缺少透明背景或不透明核心')
    # 只在複本陰影層改metadata末尾換行，不改唯讀原版檔案。
    shadow = out/'wrong-original'; (shadow/'animations').mkdir(parents=True)
    (shadow/'manifest.json').symlink_to(args.pack.resolve()/'manifest.json')
    for resource in resources:
        original = args.pack.resolve()/'animations'/f'FIGANI_{int(resource):03d}'
        folder = shadow/'animations'/original.name
        if resource != first:
            folder.symlink_to(original, target_is_directory=True); continue
        folder.mkdir()
        for path in original.iterdir():
            if path.name == 'animation.json': (folder/path.name).write_bytes(path.read_bytes()+b'\n')
            else: (folder/path.name).symlink_to(path)
    if idle_only:
        for resource in [int(first)+1,int(first)+2]:
            (shadow/'animations'/f'FIGANI_{resource:03d}').symlink_to(
                args.pack.resolve()/'animations'/f'FIGANI_{resource:03d}',target_is_directory=True)
    check('wrong-original-metadata', lambda d:None, '原始動畫未解碼或metadata不同', pack=shadow)
    if idle_only:
        empty=int(first)+1
        missing=out/'missing-empty-proof';(missing/'animations').mkdir(parents=True)
        (missing/'manifest.json').symlink_to(args.pack.resolve()/'manifest.json')
        (missing/'animations'/f'FIGANI_{int(first):03d}').symlink_to(
            args.pack.resolve()/'animations'/f'FIGANI_{int(first):03d}',target_is_directory=True)
        (missing/'animations'/f'FIGANI_{int(first)+2:03d}').symlink_to(
            args.pack.resolve()/'animations'/f'FIGANI_{int(first)+2:03d}',target_is_directory=True)
        check('missing-empty-proof',lambda d:None,'idle-only須有兩項固定零header證據',pack=missing)
        changed=out/'changed-empty-proof';(changed/'animations').mkdir(parents=True)
        (changed/'manifest.json').symlink_to(args.pack.resolve()/'manifest.json')
        for resource in [int(first),empty,empty+1]:
            folder=changed/'animations'/f'FIGANI_{resource:03d}'
            original=args.pack.resolve()/'animations'/folder.name
            if resource!=empty:folder.symlink_to(original,target_is_directory=True)
            else:
                folder.mkdir();(folder/'resource.json').write_bytes((original/'resource.json').read_bytes()+b'\n')
        check('changed-empty-proof',lambda d:None,'idle-only須有兩項固定零header證據',pack=changed)
        proof=json.loads(args.inventory.read_text())
        next(r for r in proof['resources'] if r['resource']==empty)['status']='decoded'
        path=out/'nonempty-source-inventory.json';path.write_text(json.dumps(proof))
        check('nonempty-attack',lambda d:d.update(inventory_sha256=hashlib.sha256(path.read_bytes()).hexdigest()),
              'idle-only須有兩項固定零header證據',inventory=path)
        check('invented-empty-attack',lambda d:d['resources'].update({str(empty):[]}),
              '原始動畫未解碼或metadata不同')
    # 獨立核對空白圖源的稽核門檻；alpha=1也不能混入墨跡。
    from PIL import Image
    from audit_fd2_battle_hd_coverage import check_pngs
    blank_cases = [('native-empty', 0, [True], True), ('missing-proof', 0, None, False),
                   ('nonempty-source', 0, [False], False), ('soft-ink', 1, [True], False),
                   ('opaque-ink', 255, [True], False)]
    for name, alpha, masks, accepted in blank_cases:
        path = out/(name+'.png'); Image.new('RGBA', (6,6), (20,30,40,alpha)).save(path)
        asset = {'files':[path.name], 'frame_sha256':[hashlib.sha256(path.read_bytes()).hexdigest()],
                 'native_frames':[{'width':1,'height':1}], 'source_archive':'FIGANI.DAT'}
        error = None
        try: check_pngs(asset, out, empty_masks=masks)
        except ValueError as exc: error = str(exc)
        if (error is None) != accepted: raise AssertionError((name, error))
        checks.append({'case':'empty-audit-'+name, 'status':'PASS-ACCEPTED' if accepted else 'PASS-REJECTED',
                       'reason':error or '已證實空白來源只接受全透明成品', 'output_published':False})
    report = {'status': 'PASS-CHARACTER-COMPILER-GUARDS', 'compiler_sha256': hashlib.sha256(compiler.read_bytes()).hexdigest(),
        'baseline_sha256': hashlib.sha256(args.baseline.read_bytes()).hexdigest(), 'checks': checks,
        'scope': '實檔錯配方／來源、適用的idle-only零header證據與五種空白幀稽核條件；不代表遊戲玩家驗證。'}
    (out/'guards.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'guards': len(checks)}))


if __name__ == '__main__': main()
