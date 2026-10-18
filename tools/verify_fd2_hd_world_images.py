#!/usr/bin/env python3
"""核對普通GUI的144圖源細節與雙主題取景；不是原版oracle或逐幀parity。"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ('receipt','catalog','source','report'):
        parser.add_argument('--'+name,type=Path,required=True)
    args=parser.parse_args()
    assert Path('/.dockerenv').is_file()
    assert not args.report.exists() and args.report.parent.stat().st_uid==1000
    receipt=json.loads(args.receipt.read_text())
    assert receipt['status']=='PASS' and receipt['world_runtime_checked'] is True
    catalog=json.loads(args.catalog.read_text())
    assets={a['source_group']:a for a in catalog['assets'] if a['role']=='map_sprite_hd_set'}
    assert len(assets)==125
    shots=receipt['screenshots']
    samples=[]
    # Fixed samples selected before observing pixel results. Changing phase
    # during capture is disclosed, never repaired by trying another source.
    for name in ('battle-hd-focused.png','battle-hd-focused-restored.png','world-hd-source-reloaded.png'):
        record=shots[name]
        path=args.receipt.parent/name
        assert digest(path)==record['sha256']
        before,after=record['gpu_before'],record['state']
        first=[u for u in before.get('hd_world_units',[]) if u['group']==0]
        last=[u for u in after.get('hd_world_units',[]) if u['group']==0]
        if not before.get('hd_world') or not after.get('hd_world') or len(first)!=1 or first!=last:
            samples.append({'file':name,'status':'NOT-COMPARABLE','reason':'source phase/position changed during GPU capture'})
            continue
        unit=first[0]
        left,top,right,bottom=unit['rect']
        assert right-left==bottom-top==144
        offset=unit['work_offset']
        assert left==(offset%456-0x8088%456+4)*6
        assert top==(offset//456-0x8088//456+4)*6
        asset=assets[0]
        source_name=asset['files'][unit['frame']]
        assert Path(source_name).name==source_name
        source_path=args.source/source_name
        assert digest(source_path)==asset['frame_sha256'][unit['frame']]
        source=Image.open(source_path).convert('RGBA')
        actual=Image.open(path).convert('RGB')
        assert source.size==(144,144) and actual.size==(1920,1200)
        checked,different=0,0
        for y in range(144):
            for x in range(144):
                pixel=source.getpixel((x,y))
                if pixel[3]!=255 or not (24<=left+x<1896 and 24<=top+y<1176):continue
                checked+=1
                different+=actual.getpixel((left+x,top+y))!=pixel[:3]
        assert checked>3000
        samples.append({'file':name,'status':'PASS' if not different else 'FAIL','source':source_name,
                        'source_sha256':digest(source_path),'rect':unit['rect'],'source_frame':unit['frame'],
                        'opaque_core_pixels':checked,'different_pixels':different})
        assert not different, samples[-1]
    assert sum(s['status']=='PASS' for s in samples)>=2,'insufficient stable GPU source samples'
    hd_name='world-hd-source-reloaded.png'
    hd,faithful=shots[hd_name],shots['battle-faithful-focused.png']
    assert hd['state']['view']==faithful['state']['view']
    h=Image.open(args.receipt.parent/hd_name).convert('RGB')
    f=Image.open(args.receipt.parent/'battle-faithful-focused.png').convert('RGB')
    assert h.size==(1920,1200) and f.size==(1280,800)
    terrain_difference=sum(h.getpixel((x*6+3,y*6+3))!=f.getpixel((x*4+2,y*4+2))
                           for y in range(4,9) for x in range(4,196))
    result={'status':'PASS','scope':'actual remake GPU versus registered 144 source; faithful remake geometry only, no original oracle',
            'receipt_sha256':digest(args.receipt),'catalog_sha256':digest(args.catalog),
            'verifier_sha256':digest(Path(__file__)),'source_samples':samples,
            'same_view':True,'fixed_static_terrain_samples':960,'terrain_different_pixels':terrain_difference,
            'normal_walk_samples':len(receipt['world_walk_gpu']),
            'physical_scene_and_return':all(n in shots for n in ('world-hd-physical-scene.png','world-hd-physical-return.png'))}
    assert result['normal_walk_samples']>0 and result['physical_scene_and_return']
    assert terrain_difference==0,'static terrain coordinates/color differ across themes'
    args.report.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
