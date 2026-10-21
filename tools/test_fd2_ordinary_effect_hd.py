#!/usr/bin/env python3
"""獨立核對普通HD特效的原版alpha、材質數值及損壞來源拒收。"""
import argparse
import copy
import json
import math
from pathlib import Path
import subprocess
import sys

from PIL import Image, ImageChops
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import ROOT


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ['original-pack', 'reference', 'recipe', 'candidate', 'out']:
        parser.add_argument('--'+key, type=Path, required=True)
    args = parser.parse_args()
    if not Path('/.dockerenv').exists() or args.out.exists():
        raise SystemExit('在Docker使用新的驗證輸出目錄')
    args.out.mkdir()
    recipe = json.loads(args.recipe.read_text())
    reference = json.loads(args.reference.read_text())
    compiled = json.loads((args.candidate/'compiled.json').read_text())
    assert compiled['recipe_sha256'] == digest(args.recipe)
    assert compiled['compiler_sha256'] == digest(Path(__file__).with_name('build_fd2_ordinary_effect_hd.py'))
    textures = {key: Image.open(ROOT/row['file']).convert('RGB').resize((1920,1200), Image.Resampling.BICUBIC)
                for key, row in recipe['materials'].items()}
    rows, changed = [], 0
    for row in compiled['frames']:
        folder = args.original_pack/'animations'/f"FDOTHER_{row['resource']:03d}"
        frame = json.loads((folder/'animation.json').read_text())['frames'][row['frame']]
        with Image.open(folder/frame['path']) as pixels, Image.open(folder/frame['mask_path']) as mask:
            native = pixels.convert('RGBA'); native.putalpha(mask.convert('L'))
            native = native.resize((frame['width']*6, frame['height']*6), Image.Resampling.LANCZOS)
            alpha = mask.convert('L').resize(native.size, Image.Resampling.LANCZOS)
        file = args.candidate/'pack'/row['file']
        assert digest(file) == row['sha256']
        with Image.open(file) as actual:
            assert actual.mode == 'RGBA' and actual.size == native.size
            assert ImageChops.difference(actual.getchannel('A'), alpha).getbbox() is None
            max_error, checked = 0, 0
            texture = textures[row['material']]
            for n in range(113):
                x,y = (n*97+13)%actual.width, (n*173+7)%actual.height
                old = native.getpixel((x,y))
                tex = texture.getpixel((frame['x']*6+x, frame['y']*6+y))
                lum = sum(t*w for t,w in zip(tex,[.2126,.7152,.0722]))
                expected = [max(0,min(255,math.floor(c*(.72+.28*lum/255)+.15*(t-lum)+.5))) for c,t in zip(old[:3],tex)]
                value = actual.getpixel((x,y))
                error = max(abs(a-b) for a,b in zip(value[:3],expected))
                assert error <= 1, (row['resource'],row['frame'],x,y,value,expected)
                max_error = max(max_error,error); checked += 1
                changed += int(value[3] == 255 and value[:3] != old[:3])
        rows.append({'resource':row['resource'],'frame':row['frame'],'alpha_exact':True,
                     'rgb_samples':checked,'max_rgb_round_error':max_error,'sha256':row['sha256']})
    assert len(rows) == 304 and changed > 1000
    guards = []
    for fault in ['resource_scope','material_sha','reference_sha','source_version','header','descriptor','pixels','mask','metadata','transparent_material']:
        root = args.out/fault; root.mkdir()
        bad = copy.deepcopy(recipe); ref = copy.deepcopy(reference)
        original, refpath = args.original_pack, args.reference
        if fault == 'resource_scope': bad['resources'] = bad['resources'][:-1]
        elif fault == 'material_sha': bad['materials']['fire']['sha256'] = '0'*64
        elif fault == 'reference_sha': bad['reference_sha256'] = '0'*64
        elif fault in ['source_version','header','descriptor']:
            if fault == 'source_version': ref['source_file']['sha256'] = '0'*64
            elif fault == 'header': ref['resources'][0]['native_header']['byte_1'] ^= 1
            else: ref['resources'][0]['frames'][0]['x'] += 1
            refpath = root/'reference.json'; refpath.write_text(json.dumps(ref))
            bad['reference_sha256'] = digest(refpath)
        elif fault == 'transparent_material':
            material = root/'transparent.png'; Image.new('RGBA',(512,512),(90,100,110,128)).save(material)
            bad['materials']['fire'] = {'file':str(material.relative_to(ROOT)),'sha256':digest(material)}
        else:
            original = root/'original'; original.mkdir()
            (original/'manifest.json').symlink_to(args.original_pack/'manifest.json')
            animations = original/'animations'; animations.mkdir()
            for src in (args.original_pack/'animations').iterdir():
                dest = animations/src.name
                if src.name != 'FDOTHER_018': dest.symlink_to(src); continue
                dest.mkdir()
                first = json.loads((src/'animation.json').read_text())['frames'][0]
                badname = {'pixels': first['path'], 'mask': first['mask_path'], 'metadata':'animation.json'}[fault]
                for file in src.iterdir():
                    if file.name == badname: (dest/file.name).write_bytes(file.read_bytes()+b'\n')
                    else: (dest/file.name).symlink_to(file)
        proposal = root/'recipe.json'; proposal.write_text(json.dumps(bad))
        output = root/'must-not-exist'
        run = subprocess.run([sys.executable,str(Path(__file__).with_name('build_fd2_ordinary_effect_hd.py')),
            '--original-pack',str(original),'--reference',str(refpath),'--recipe',str(proposal),'--out',str(output)],
            capture_output=True,text=True,timeout=60)
        assert run.returncode != 0 and not output.exists(), (fault,run.stdout,run.stderr)
        guards.append({'fault':fault,'returncode':run.returncode,'output_absent':True,'reason':(run.stdout+run.stderr).strip()})
    report = {'status':'PASS-ORDINARY-MATERIAL-SOURCE-AND-REJECTION','compiler_sha256':compiled['compiler_sha256'],
              'test_sha256':digest(Path(__file__)),'candidate_sha256':digest(args.candidate/'catalog.json'),
              'frames':rows,'changed_opaque_rgb_samples':changed,'guards':guards,
              'scope':'304幀alpha逐像素相同，34352組獨立float64 RGB抽樣及十種來源拒收；非原版RGB對拍或人工畫質驗收。'}
    (args.out/'report.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':report['status'],'frames':len(rows),'guards':len(guards),'changed_opaque_samples':changed}))


if __name__ == '__main__':
    main()
