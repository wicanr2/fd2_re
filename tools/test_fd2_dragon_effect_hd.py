#!/usr/bin/env python3
"""驗火龍36幀可重生矩陣、來源不變與有界編製拒收。"""
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
    parser=argparse.ArgumentParser(description=__doc__)
    for key in ['original-pack','reference','recipe','candidate','prototype','out']:
        parser.add_argument('--'+key,type=Path,required=True)
    args=parser.parse_args()
    if not Path('/.dockerenv').exists() or args.out.exists():raise SystemExit('在Docker使用新的驗證輸出目錄')
    args.out.mkdir();recipe=json.loads(args.recipe.read_text());reference=json.loads(args.reference.read_text())
    compiled=json.loads((args.candidate/'compiled.json').read_text());prototype=json.loads(args.prototype.read_text())
    assert compiled['recipe_sha256']==digest(args.recipe)
    assert compiled['compiler_sha256']==digest(Path(__file__).with_name('build_fd2_dragon_effect_hd.py'))
    expected={(v['resource'],v['frame']):v for v in prototype['frames']}
    assert len(expected)==36 and set(expected)=={(resource,index) for resource in [26,27] for index in range(18)}
    originals={key:digest(ROOT/v['file']) for key,v in recipe['sources'].items()}
    positives=[]
    for row in compiled['frames']:
        key=row['resource'],row['frame'];proof=expected[key];file=args.candidate/'pack'/row['file']
        assert row['sha256']==digest(file)==proof['sha256'],key
        with Image.open(file) as im:
            assert im.mode=='RGBA' and im.size==(row['native_descriptor']['width']*6,row['native_descriptor']['height']*6)
            assert im.getchannel('A').getextrema()==(0,255)
            core=sum(im.getchannel('A').histogram()[250:]);assert core>=1000
        assert row['size']==proof['fit_size'] and row['origin']==proof['origin']
        positives.append({'resource':key[0],'frame':key[1],'sha256':row['sha256'],'prototype_png_exact':True,'alpha250_pixels':core})
    assert len(positives)==36
    guards=[]
    for fault in ['resources','reference_sha','source_version','source_sha','source_anchor','source_alpha','missing_frame',
                  'mirror','size_limit','off_canvas','header','native_pixels','native_mask']:
        folder=args.out/fault;folder.mkdir();bad=copy.deepcopy(recipe);ref=copy.deepcopy(reference)
        refpath=args.reference;original=args.original_pack
        if fault=='resources':bad['resources']['28']=bad['resources']['26']
        elif fault=='reference_sha':bad['reference_sha256']='0'*64
        elif fault=='source_version':ref['source_file']['sha256']='0'*64
        elif fault=='source_sha':bad['sources']['closed']['sha256']='0'*64
        elif fault=='source_anchor':bad['sources']['closed']['anchor'][0]+=1
        elif fault=='source_alpha':
            image=folder/'opaque.png';Image.new('RGBA',(512,512),(210,40,20,255)).save(image)
            bad['sources']['closed'].update(file=str(image.relative_to(ROOT)),sha256=digest(image))
        elif fault=='missing_frame':bad['resources']['26'].pop()
        elif fault=='mirror':bad['resources']['26'][0]['flip']=True
        elif fault=='size_limit':bad['resources']['26'][0]['size'][0]=4097
        elif fault=='off_canvas':bad['resources']['26'][0]['origin']=[-4000,-4000]
        elif fault=='header':ref['resources'][next(i for i,v in enumerate(ref['resources']) if v['resource']==26)]['native_header']['byte_1']^=1
        else:
            original=folder/'original';original.mkdir();(original/'manifest.json').symlink_to(args.original_pack/'manifest.json')
            animations=original/'animations';animations.mkdir()
            for src in (args.original_pack/'animations').iterdir():
                dest=animations/src.name
                if src.name!='FDOTHER_026':dest.symlink_to(src);continue
                dest.mkdir();frame=json.loads((src/'animation.json').read_text())['frames'][-1]
                changed=frame['path' if fault=='native_pixels' else 'mask_path']
                for file in src.iterdir():
                    if file.name==changed:(dest/file.name).write_bytes(file.read_bytes()+b'\n')
                    else:(dest/file.name).symlink_to(file)
        if fault in ['source_version','header']:
            refpath=folder/'reference.json';refpath.write_text(json.dumps(ref));bad['reference_sha256']=digest(refpath)
        proposal=folder/'recipe.json';proposal.write_text(json.dumps(bad));output=folder/'must-not-exist'
        result=subprocess.run([sys.executable,str(Path(__file__).with_name('build_fd2_dragon_effect_hd.py')),
            '--original-pack',str(original),'--reference',str(refpath),'--recipe',str(proposal),'--out',str(output)],
            capture_output=True,text=True,timeout=90)
        assert result.returncode!=0 and not output.exists(),(fault,result.stdout,result.stderr)
        guards.append({'fault':fault,'returncode':result.returncode,'output_absent':True,'reason':(result.stdout+result.stderr).strip()})
    assert all(digest(ROOT/v['file'])==originals[key] for key,v in recipe['sources'].items())
    report={'status':'PASS-DRAGON-36-PROJECTION-AND-REJECTION','test_sha256':digest(Path(__file__)),
        'compiler_sha256':compiled['compiler_sha256'],'prototype_sha256':digest(args.prototype),
        'candidate_sha256':digest(args.candidate/'catalog.json'),'frames':positives,'guards':guards,'source_rgba_unchanged':True,
        'scope':'全36矩陣重生與先於正式編製的投影PNG完全相同，十三種錯件不發布輸出；非原版RGB／alpha parity或人工畫質驗收。'}
    (args.out/'report.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':report['status'],'frames':len(positives),'guards':len(guards)}))


if __name__=='__main__':
    main()
