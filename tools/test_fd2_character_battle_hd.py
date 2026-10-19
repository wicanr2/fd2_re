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
    args = p.parse_args(); out = args.out.resolve(); compiler = Path(__file__).with_name('build_fd2_character_battle_hd.py')
    if not Path('/.dockerenv').exists() or out.exists() or not out.is_relative_to(ROOT) or out.parent.stat().st_uid != os.getuid():
        raise SystemExit('在Docker使用本人擁有的工作區新測試目錄')
    baseline = json.loads(args.baseline.read_text()); recipe = json.loads(args.recipe.read_text())
    if baseline['status'] != 'PASS-COMPILED-CHARACTER-CANDIDATE' or baseline['compiler_sha256'] != hashlib.sha256(compiler.read_bytes()).hexdigest():
        raise SystemExit('需要目前編製器已通過的實檔候選')
    if baseline['recipe_sha256'] != hashlib.sha256(args.recipe.read_bytes()).hexdigest():
        raise SystemExit('基底與配方不同')
    out.mkdir(); resources = list(recipe['resources']); first = resources[0]; attack = resources[1]; source = next(iter(recipe['sources']))
    checks = []

    def check(name, mutate, expected, pack=None, base=None):
        doc = copy.deepcopy(recipe); mutate(doc); path = out/(name+'.json'); path.write_text(json.dumps(doc))
        target = out/(name+'-candidate')
        command = [sys.executable, str(compiler), '--pack', str(pack or args.pack), '--inventory', str(args.inventory),
                   '--recipe', str(path), '--out', str(target)]
        if base is not None: command += ['--base-catalog', str(base)]
        result = subprocess.run(command, capture_output=True, text=True, timeout=60)
        output = result.stdout+result.stderr
        if result.returncode == 0 or target.exists() or expected not in output:
            raise AssertionError((name, result.returncode, output))
        checks.append({'case': name, 'status': 'PASS-REJECTED', 'reason': output.strip().splitlines()[-1], 'output_published': False})

    check('schema', lambda d:d.update(schema_version=2), '未知配方版本')
    check('inventory-sha', lambda d:d.update(inventory_sha256='0'*64), '盤點、配方或原版包指紋不符')
    check('source-sha', lambda d:d['sources'][source].update(sha256='0'*64), '母稿路徑、擁有權或SHA不符')
    check('source-anchor', lambda d:d['sources'][source].update(anchor=[0,0,1000000,1000000]), '錨點須是畫布內')
    check('native-anchor', lambda d:d['resources'][first][0].update(anchor=[0,0,1000000,1000000]), '錨點須是畫布內')
    check('missing-frame', lambda d:d['resources'][attack].pop(), '配方影格不完整')
    check('missing-resource', lambda d:d['resources'].pop(attack), '配方須涵蓋完整待機及攻擊資源')
    check('unknown-pose', lambda d:d['resources'][attack][0].update(pose='unknown'), '未知母稿、群組或配方欄位')
    check('unknown-field', lambda d:d['resources'][attack][0].update(mystery=True), '未知母稿、群組或配方欄位')
    check('already-registered', lambda d:None, '所選圖源已登錄', base=args.baseline.parent/'catalog.json')
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
    check('wrong-original-metadata', lambda d:None, '原始動畫未解碼或metadata不同', pack=shadow)
    report = {'status': 'PASS-CHARACTER-COMPILER-GUARDS', 'compiler_sha256': hashlib.sha256(compiler.read_bytes()).hexdigest(),
        'baseline_sha256': hashlib.sha256(args.baseline.read_bytes()).hexdigest(), 'checks': checks,
        'scope': '十一種實檔錯配方／來源fixture全部拒收且不建立候選；不代表遊戲玩家驗證。'}
    (out/'guards.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'guards': len(checks)}))


if __name__ == '__main__': main()
