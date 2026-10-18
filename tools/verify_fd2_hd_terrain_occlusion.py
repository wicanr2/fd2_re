#!/usr/bin/env python3
"""核對正常屋後GPU的前景opaque覆寫與透明處人物核心，不作原版對拍。"""
import argparse
import hashlib
import json
from pathlib import Path
from PIL import Image


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def load_frame(pack, name, expected):
    path=pack/name
    if Path(name).name!=name or sha(path.read_bytes())!=expected:
        raise ValueError('來源檔案SHA不符')
    return Image.open(path).convert('RGBA')


def rgba_hash(picture):
    # Go image.RGBA stores binary-alpha transparent pixels as premultiplied zero.
    return sha(bytes(c for p in picture.getdata() for c in (p if p[3] else (0,0,0,0))))


def main():
    p=argparse.ArgumentParser(description=__doc__)
    group=p.add_mutually_exclusive_group(required=True)
    group.add_argument('--receipt',type=Path)
    group.add_argument('--observation',type=Path,help='保留整輪失敗，獨立核對已完成屋後階段的原始觀測')
    p.add_argument('--route',type=Path)
    for name in ['catalog','pack','report']:
        p.add_argument('--'+name,type=Path,required=True)
    args=p.parse_args()
    if args.report.exists(): raise ValueError('不得覆寫收據')
    origin=args.receipt or args.observation
    receipt=json.loads(origin.read_text())
    if args.receipt:
        if receipt['status']!='PASS': raise ValueError('正常流程未通過')
    else:
        if not receipt['tool_source_unchanged'] or not args.route: raise ValueError('觀測缺少來源及普通路徑')
        route=json.loads(args.route.read_text())
        if route['state_injection'] or not route['tool_source_unchanged']:
            raise ValueError('路徑有注入或來源被改寫')
        if not route['steps'] or route['steps'][-1]['to']!=route['goal']:
            raise ValueError('普通路徑未抵達屋後')
        first=receipt['screenshots']['terrain-hd-occlusion.png']['state']
        faithful=receipt['screenshots']['terrain-faithful-occlusion.png']['state']
        last=receipt['screenshots']['terrain-hd-occlusion-restored.png']['state']
        if first['units']!=faithful['units'] or first['units']!=last['units'] or first['view']!=faithful['view'] or first['view']!=last['view']:
            raise ValueError('屋後F2狀態不同')
    catalog=json.loads(args.catalog.read_text())
    terrain={}
    units={}
    for a in catalog['assets']:
        if a['role']=='native_terrain_hd_set' and a['map_id']==0:
            for frame,name,h,role in zip(a['frame_ids'],a['files'],a['frame_sha256'],a['semantics']):
                if role in ['foreground','both']: terrain[frame]=load_frame(args.pack,name,h)
        if a['role']=='map_sprite_hd_set':
            for i,(name,h) in enumerate(zip(a['files'],a['frame_sha256'])):
                units[a['source_group'],i]=(name,h)
    results=[]
    unit_proof=False
    for name in ['terrain-hd-occlusion.png','terrain-hd-occlusion-restored.png']:
        shot=receipt['screenshots'][name]
        path=origin.parent/name
        if sha(path.read_bytes())!=shot['sha256']: raise ValueError('實際PNG被改寫')
        actual=Image.open(path).convert('RGBA')
        if actual.size!=(1920,1200): raise ValueError('視窗尺寸不符')
        attempts=[]
        for state in [shot['gpu_before'],shot['state']]:
            for fg in state.get('hd_world_terrain_foreground_draws',[]):
                source=terrain[fg['frame']]
                if rgba_hash(source)!=fg['rgba_sha256']:
                    attempts.append({'status':'NOT-COMPARABLE','reason':'published shade differs from identity source'})
                    continue
                x,y,x1,y1=fg['rect']
                opaque=[]
                for sy in range(144):
                    for sx in range(144):
                        px,py=x+sx,y+sy
                        if source.getpixel((sx,sy))[3]==255 and 24<=px<1896 and 24<=py<1176:
                            opaque.append((px,py,source.getpixel((sx,sy))))
                wrong=sum(actual.getpixel((px,py))!=value for px,py,value in opaque)
                covered,transparent,visible_wrong=0,0,0
                for unit in state.get('hd_world_units',[]):
                    ux,uy,ux1,uy1=unit['rect']
                    if ux>=x1 or ux1<=x or uy>=y1 or uy1<=y: continue
                    uname,uhash=units[unit['group'],unit['frame']]
                    person=load_frame(args.pack,uname,uhash)
                    for py in range(max(24,y,uy),min(1176,y1,uy1)):
                        for px in range(max(24,x,ux),min(1896,x1,ux1)):
                            value=person.getpixel((px-ux,py-uy))
                            if value[3]!=255: continue
                            if source.getpixel((px-x,py-y))[3]:
                                covered+=1
                            else:
                                transparent+=1
                                expected=value[:3]
                                if unit.get('palette_band'):
                                    expected=tuple(round(a*b) for a,b in zip(expected,[.55,.55,.6]))
                                got=actual.getpixel((px,py))[:3]
                                visible_wrong+=any(abs(a-b)>1 for a,b in zip(expected,got))
                foreground_ok=wrong==0 and covered>100
                unit_ok=visible_wrong==0 and transparent>100
                attempts.append({'status':'PASS' if foreground_ok else 'FAIL',
                                 'unit_status':'PASS' if unit_ok else 'NOT-COMPARABLE',
                                 'frame':fg['frame'],'rect':fg['rect'],'opaque_pixels':len(opaque),
                                 'opaque_differences':wrong,'covered_unit_core_pixels':covered,
                                 'visible_unit_core_pixels':transparent,'visible_unit_differences_over_1':visible_wrong})
        passing=[a for a in attempts if a['status']=='PASS']
        if not passing: raise ValueError((name,attempts))
        comparable=[a for a in passing if a['unit_status']=='PASS']
        unit_proof=unit_proof or bool(comparable)
        results.append({'file':name,'verified':(comparable or passing)[0],'other_observations':attempts,
                        'limits':'前景opaque必須全相符；人物只用與實際PNG相符的before／after來源相位，其他相位保留不可比。'})
    if not unit_proof: raise ValueError('沒有來源相位可比的透明處人物核心證據')
    result={'status':'PASS','samples':results,'receipt_sha256':sha(origin.read_bytes()),
            'catalog_sha256':sha(args.catalog.read_bytes()),'tool_sha256':sha(Path(__file__).read_bytes()),
            'scope':'正常移動／END抵達屋後；兩張前景opaque逐像素，至少一張來源相位可比的透明處人物核心容許GPU色階1；其餘人物相位不當成通過。非原版oracle。'}
    if args.observation:
        result['scope']+=' 限已完成屋後階段；整輪後續F9腳本失敗保留，不宣稱整輪通過。'
        result['route_sha256']=sha(args.route.read_bytes())
    args.report.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
