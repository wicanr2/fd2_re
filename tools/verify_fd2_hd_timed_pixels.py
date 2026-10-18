#!/usr/bin/env python3
"""以正式格網邊角與144圖源核對實際GPU，容許擷取跨相鄰呈現幀。"""
import argparse
import hashlib
import json
from pathlib import Path
from PIL import Image, ImageChops


def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ('observation','original-pack','catalog','modern-pack','report'):
        p.add_argument('--'+n,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').is_file() and not a.report.exists()
    obs=json.loads(a.observation.read_text());root=a.observation.parent
    resource=a.original_pack/'ui/fdother_005_item_panel/resource.json'
    data=json.loads(resource.read_text())
    original=json.loads((a.original_pack/'manifest.json').read_text())
    source=next(s for s in original['source_set'] if s['file']=='FDOTHER.DAT')
    assert data['source']['sha256']==source['sha256']
    palette_path=a.original_pack/'palette/fdother_000.json'
    palette=json.loads(palette_path.read_text());assert palette['source']['sha256']==source['sha256']
    dac=palette['dac_6bit_components'];assert len(dac)==768 and all(0<=v<=63 for v in dac)
    rgb=[tuple((v<<2)|(v>>4) for v in dac[i:i+3]) for i in range(0,768,3)]
    cells={}
    for index in (2,4):
        entry=next(e for e in data['entries'] if e['index']==index)
        path=resource.parent/entry['frame'];im=Image.open(path)
        assert im.mode=='P' and im.size==(3,3)
        cells[index]=(list(im.getdata()),path)
    assets=json.loads(a.catalog.read_text())['assets']
    sol=next(s for s in assets if s['role']=='map_sprite_hd_set' and s['source_group']==0)
    samples=obs['samples']+obs['rejected'];results=[];phases=set();checked_owners={}
    sizes={0:((4,2),(8,3),(12,4),(16,5),(19,5)),2:((16,5),(12,4),(8,3),(4,2))}
    for sample in samples:
        phase=sample['before'].get('hd_timed_phase');owner=sample['before'].get('timed_dialogue_owner')
        if phase not in sizes:continue
        refs=[s for s in obs['samples'] if s['before'].get('timed_dialogue_owner')==owner and s['before'].get('hd_timed_phase')==1]
        if not refs:continue
        ref=refs[0];path=root/sample['file'];assert sha(path)==sample['sha256']
        actual=Image.open(path).convert('RGB');assert actual.size==(1920,1200)
        matching=[]
        for columns,rows in sizes[phase]:
            good=True
            for index,y in ((2,112),(4,115+16*rows)):
                pixels,_=cells[index];x=8+16*columns
                for dy in range(3):
                    for dx in range(3):
                        if actual.getpixel(((x+dx)*6+3,(y+dy)*6+3))!=rgb[pixels[dy*3+dx]]:good=False
            if good:matching.append([columns,rows])
        result={'file':sample['file'],'sha256':sample['sha256'],'phase':phase,'actual_grid_sizes':matching,
                'audit_indices':[sample['before'].get('hd_timed_index'),sample['after'].get('hd_timed_index')]}
        if not matching:
            result.update(status='NOT-COMPARABLE',reason='actual pixels do not show a grid from the declared phase')
            results.append(result);continue
        reference=Image.open(root/ref['file']).convert('RGB');box=(24,24,1896,666)
        difference=sum(p!=(0,0,0) for p in ImageChops.difference(actual.crop(box),reference.crop(box)).getdata())
        assert difference==0,(sample['file'],difference)
        if owner not in checked_owners:
            unit=next(u for u in ref['before']['hd_world_units'] if u['group']==0)
            assert unit==next(u for u in ref['after']['hd_world_units'] if u['group']==0)
            source=a.modern_pack/sol['files'][unit['frame']];assert sha(source)==sol['frame_sha256'][unit['frame']]
            sprite=Image.open(source).convert('RGBA');left,top,_,_=unit['rect'];checked=wrong=0
            for y in range(144):
                for x in range(144):
                    pixel=sprite.getpixel((x,y))
                    if pixel[3]!=255 or not(24<=left+x<1896 and 24<=top+y<666):continue
                    checked+=1;wrong+=reference.getpixel((left+x,top+y))!=pixel[:3]
            assert checked>3000 and wrong==0,(checked,wrong)
            checked_owners[owner]={'source':source.name,'source_sha256':sha(source),'opaque_core_pixels':checked,'different_pixels':wrong}
        result.update(status='PASS',outside_different_pixels=difference,reference=ref['file'])
        results.append(result);phases.add(phase)
    assert {0,2}<=phases,'missing actual opening/closing grid pixels'
    a.report.write_text(json.dumps({'status':'PASS','observation_sha256':sha(a.observation),'verifier_sha256':sha(Path(__file__)),
        'cell_sha256':{str(i):sha(cells[i][1]) for i in cells},'palette_sha256':sha(palette_path),
        'catalog_sha256':sha(a.catalog),'samples':results,'hd_source_checks':checked_owners,
        'scope':'Actual GPU, native corner geometry and registered HD source; not original oracle or exact-frame parity.'},ensure_ascii=False,indent=2)+'\n')
    print('PASS: actual opening and closing grids, frozen HD world, registered 144 source')


if __name__=='__main__':main()
