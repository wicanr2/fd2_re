#!/usr/bin/env python3
"""核七種普通法術owner的代表HD特效最終Game.Draw像素。"""
import argparse
import json
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate
from verify_fd2_ares_battle_gpu import matches
from verify_fd2_compound_caster_gpu import make_native_occlusion_test


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--receipt', type=Path, action='append', required=True)
    for key in ['catalog','source','original-pack','inventory','report']:
        parser.add_argument('--'+key,type=Path,required=True)
    args = parser.parse_args()
    if args.report.exists(): raise SystemExit('不覆寫既有收據')
    catalog = validate(False,args.catalog)
    effects = {a['source_resource']:a for a in catalog['assets'] if a['role']=='battle_effect_hd_set'}
    figures = {a['source_resource']:a for a in catalog['assets'] if a['role']=='battle_animation_hd_set'}
    occluded = make_native_occlusion_test(figures,args.source,args.original_pack,
                                         json.loads(args.inventory.read_text()),effects)
    expected = {0:[18,20],1:[19,21],3:[39,43],5:[24,25],6:[32,33],7:[37,38],8:[28,30]}
    seen, rows = set(), []
    for receipt in args.receipt:
        data = json.loads(receipt.read_text()); command = data.get('command_id')
        if (command not in expected or command in seen
                or data.get('status') != f'PASS-ORDINARY-COMMAND{command}-FIXTURE-RUNTIME-E1'
                or data.get('continuations') != 1 or data.get('mp_publications') != 1
                or not data.get('go_rng_unchanged')):
            raise SystemExit('需要七種不同owner的完整交易收據')
        seen.add(command); found = []
        for name, shot in data['source_shots'].items():
            if shot['subtract'] != 0 or shot['mix'] != 0: continue
            if Path(name).name != name: raise SystemExit('GPU擷取路徑越界')
            for layer_index, layer in enumerate(shot['layers']):
                if (layer['archive'] != 'FDOTHER.DAT' or layer['resource'] not in expected[command]
                        or layer.get('fill',-1) != -1 or layer.get('packed_shader',0) != 0): continue
                asset = effects[layer['resource']]; index = layer['frame']
                frame = asset['native_frames'][index]
                if (layer['width'],layer['height']) != (frame['width'],frame['height']):
                    raise SystemExit('特效來源描述子不同')
                source = args.source/asset['files'][index]
                if digest(source) != asset['frame_sha256'][index]: raise SystemExit('特效來源SHA不同')
                with Image.open(source) as im:
                    points = [(i%im.width,i//im.width,r,g,b,a) for i,(r,g,b,a) in enumerate(im.getdata()) if a==255]
                if len(points)<1000: continue
                selected = points[::max(1,len(points)//8000)]
                ox,oy = layer['x']*6,layer['y']*6
                visible = [point for point in selected
                           if 0<=point[0]+ox<1920 and 0<=point[1]+oy<1200
                           and not occluded(point[0]+ox,point[1]+oy,shot['layers'][layer_index+1:])]
                if len(visible)<1000: continue
                path = receipt.parent/name
                with Image.open(path) as im:
                    if im.size != (1920,1200): raise SystemExit('需要1920×1200最終GPU')
                    okay = matches(im.convert('RGB').load(),visible,ox,oy,1920,1200)
                if okay:
                    found.append({'file':name,'sha256':digest(path),'phase':shot['phase'],
                                  'resource':layer['resource'],'frame':index,'source_sha256':digest(source),
                                  'checked_opaque_pixels':len(visible)})
                    break
            if found: break
        if not found: raise SystemExit(f'普通法術{command}未核到實際HD特效核心')
        rows.append({'command':command,'receipt_sha256':digest(receipt),'matches':found})
    if seen != set(expected): raise SystemExit('七種普通owner不完整')
    report = {'status':'PASS-SEVEN-ORDINARY-OWNERS-EFFECT-GAME-DRAW','catalog_sha256':digest(args.catalog),
              'verifier_sha256':digest(Path(__file__)),'owners':rows,
              'scope':'七種owner各一張最終Game.Draw特效的至少1000個全不透明抽樣點，依前景來源遮罩排除遮擋。'
                      '4與5共用owner以5作代表；4的22/23來源另由304幀GPU驗證。非所有相位RGB、正常操作或人工美術驗收。'}
    args.report.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':report['status'],'owners':len(rows)}))


if __name__ == '__main__':
    main()
