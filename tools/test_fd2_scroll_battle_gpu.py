#!/usr/bin/env python3
"""以實際飛矢PNG反證圖源核對器拒絕錯幀、錯位置及原生替身。"""
import argparse
import copy
import json
from pathlib import Path
import subprocess
import sys

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import ROOT


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['receipt', 'catalog', 'source', 'baseline', 'original-pack', 'out']:
        p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--target-figure',type=int,choices=[76,96],default=96)
    p.add_argument('--target-frames',action='store_true')
    args = p.parse_args()
    verifier = Path(__file__).with_name('verify_fd2_scroll_battle_gpu.py')
    baseline = json.loads(args.baseline.read_text())
    if (baseline.get('status') != 'PASS-SCROLL-FIXTURE-REGISTERED-GPU'
            or baseline.get('four_projectile_frames_checked') is not True
            or baseline.get('target_figure',96) != args.target_figure
            or args.target_frames and baseline.get('target_attack_frames_checked') is not True
            or baseline.get('receipt_sha256') != digest(args.receipt)
            or baseline.get('catalog_sha256') != digest(args.catalog)
            or baseline.get('verifier_sha256') != digest(verifier)):
        raise SystemExit('需要目前工具的四飛矢實檔正向報告')
    if args.out.exists() or not args.out.resolve().is_relative_to(ROOT):
        raise SystemExit('使用工作區新守衛目錄')
    args.out.mkdir()
    receipt = json.loads(args.receipt.read_text())
    checks = []

    def check(name, doc, expected, replace_bolts=False):
        folder = args.out/name; folder.mkdir()
        for shot in doc['source_shots'].values():
            filename = shot['file']
            if Path(filename).name != filename: raise SystemExit('圖片路徑越界')
            target = folder/filename
            if not target.exists(): target.symlink_to((args.receipt.parent/filename).resolve())
        if replace_bolts:
            # 只修改輸出陰影層，原始GPU與來源包保持唯讀。
            for phase, shot in doc['source_shots'].items():
                if not phase.startswith('body-bolt-'): continue
                path = folder/shot['file']; path.unlink()
                with Image.open(args.receipt.parent/shot['file']) as source:
                    canvas = source.convert('RGBA')
                layer = next(s for s in shot['layers'] if s['resource']==25 and 15<=s['frame']<=18)
                index = layer['frame']; native = args.original_pack/'animations/FIGANI_025'
                with Image.open(native/f'FIGANI_025_f{index:02d}.png') as image, Image.open(native/f'FIGANI_025_f{index:02d}_mask.png') as mask:
                    raw = image.convert('RGBA'); raw.putalpha(mask.convert('L'))
                    raw = raw.resize((raw.width*6,raw.height*6),Image.Resampling.NEAREST)
                x,y=layer['rect']['Min']['X']*6,layer['rect']['Min']['Y']*6
                canvas.paste((0,0,0,255),(x,y,x+raw.width,y+raw.height))
                canvas.alpha_composite(raw,(x,y)); canvas.convert('RGB').save(path)
        path = folder/'receipt.json'; path.write_text(json.dumps(doc))
        report = folder/'report.json'
        argv=[sys.executable,str(verifier),'--receipt',str(path),
            '--catalog',str(args.catalog),'--source',str(args.source),'--projectiles','--report',str(report),
            '--target-figure',str(args.target_figure)]
        if args.target_frames:argv.append('--target-frames')
        result = subprocess.run(argv,
            capture_output=True,text=True,timeout=120)
        output=result.stdout+result.stderr
        if result.returncode==0 or report.exists() or expected not in output:
            raise AssertionError((name,result.returncode,output))
        checks.append({'case':name,'status':'PASS-REJECTED','reason':output.strip().splitlines()[-1],
                       'output_published':False})

    missing=copy.deepcopy(receipt); del missing['source_shots']['body-bolt-15']
    check('missing-bolt',missing,'未核對到實際飛矢GPU圖源：15')
    wrong=copy.deepcopy(receipt)
    layer=next(s for s in wrong['source_shots']['body-bolt-15']['layers'] if s['resource']==25)
    layer['frame']=16
    check('wrong-frame',wrong,'原生描述子尺寸或垂直原點不同')
    wrong=copy.deepcopy(receipt)
    layer=next(s for s in wrong['source_shots']['body-bolt-15']['layers'] if s['resource']==25)
    layer['rect']['Min']['Y']+=1; layer['rect']['Max']['Y']+=1
    check('wrong-position',wrong,'原生描述子尺寸或垂直原點不同')
    check('native-pixel-substitute',copy.deepcopy(receipt),'未核對到實際飛矢GPU圖源：15',True)
    wrong=copy.deepcopy(receipt);wrong['target_figure']=76 if args.target_figure==96 else 96
    check('wrong-target',wrong,'測試收據的敵方人物不同')
    if args.target_frames:
        missing=copy.deepcopy(receipt);del missing['source_shots']['counter-target-01']
        check('missing-target-attack',missing,'未核對到實際敵方攻擊GPU圖源：1')
    report={'status':'PASS-SCROLL-PROJECTILE-GUARDS','baseline_sha256':digest(args.baseline),
            'tool_sha256':digest(Path(__file__)),'checks':checks,
            'scope':'實際GPU收據的缺幀、錯幀、錯位置、原生替身、錯人物及適用的敵方缺攻擊幀負向陰影層；不新增普通玩家或原版對拍證據。'}
    (args.out/'guards.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':report['status'],'checks':len(checks)}))


if __name__ == '__main__': main()
