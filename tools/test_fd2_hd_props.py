#!/usr/bin/env python3
"""核對透明物件編製、開箱來源與正式登錄的拒收邊界。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import shutil
import tempfile

from PIL import Image
from build_fd2_hd_ground import make_tile
from build_fd2_hd_props import build, validate_sources, painted_crop
from register_fd2_hd_ground import register
from verify_fd2_terrain_sources import sha

ROOT=Path(__file__).resolve().parents[1]


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ['original-pack','sources','recipe','candidate','base-pack','base-catalog','gpu','report']:
        p.add_argument('--'+n,type=Path,required=True)
    a=p.parse_args()
    if a.report.exists():raise ValueError('不得覆寫物件收據')
    maps=ROOT/'remake/assets/maps';recipe=json.loads(a.recipe.read_text());manifest=json.loads((a.sources/'manifest.json').read_text())
    proof=json.loads((a.candidate/'manifest.json').read_text())
    checked=validate_sources(a.original_pack,maps,manifest,recipe)
    grass=make_tile(a.base_pack/recipe['grass_master_file'],144,8)[0]
    pixels=high_pixels=0
    for item in checked:
        g=item['group'];tile=Image.open(a.candidate/'pack'/g['file']).convert('RGBA')
        assert tile.size==(144,144) and tile.getextrema()[3]==(255,255)
        master=Image.open(a.candidate/'pack'/recipe['masters'][g['master']]['file'])
        x0,y0,x1,y1=[v*6 for v in g['native_bbox']]
        art=painted_crop(master,g).resize((x1-x0,y1-y0),Image.Resampling.LANCZOS)
        for y in range(144):
            for x in range(144):
                actual=tile.getpixel((x,y));base=grass.getpixel((x,y))
                inside=x0<=x<x1 and y0<=y<y1
                painted=art.getpixel((x-x0,y-y0)) if inside else (0,0,0,0)
                if not inside or painted[3]==0:
                    assert all(b*153//255<=v<=b for v,b in zip(actual[:3],base[:3]))
                if painted[3]>=250:
                    assert all(abs(v-b)<=6 for v,b in zip(actual[:3],painted[:3]))
                    high_pixels+=1
                if (x in (0,143) or y in (0,143)) and not inside:
                    factor=153 if item['pixels'][y//6*24+x//6] in recipe['shadow_indices'] else 255
                    assert actual==(*[v*factor//255 for v in base[:3]],255)
                pixels+=1
    assert pixels==124416 and high_pixels>10000
    checks=[]
    def reject(name,fn,text):
        try:fn()
        except ValueError as error:
            assert text in str(error),(name,str(error));checks.append({'case':name,'error':str(error)})
        else:raise AssertionError('物件反例未拒收：'+name)
    for name,change,text in [
        ('missing-target',lambda r:r['groups'][0]['targets'].pop(),'consumer'),
        ('missing-group',lambda r:r['groups'].pop(),'身分不完整'),
        ('wrong-shadow',lambda r:r.__setitem__('shadow_level',0),'陰影契約'),
        ('wrong-bbox',lambda r:r['groups'][0]['native_bbox'].__setitem__(0,0),'bbox'),
        ('wrong-master',lambda r:r['groups'][0].__setitem__('master','rocks'),'配對'),
        ('unsafe-file',lambda r:r['groups'][0].__setitem__('file','../bad.png'),'越界'),
    ]:
        r=copy.deepcopy(recipe);change(r);reject(name,lambda r=r:validate_sources(a.original_pack,maps,manifest,r),text)
    bad=copy.deepcopy(manifest);bad['maps'][0]['frames'][248]['input_layers'][0]['sha256']='0'*64
    reject('wrong-binding',lambda:validate_sources(a.original_pack,maps,bad,recipe),'雜湊不符')
    with tempfile.TemporaryDirectory(dir=a.report.parent) as tmp:
        tmp=Path(tmp)
        for name,layer in [('wrong-opaque',1),('wrong-remap',2)]:
            shadow=tmp/name;shadow.mkdir()
            # Only flat file links are used; no directory link may redirect a write to the original pack.
            entries=[mp['bank'] for mp in manifest['maps']]
            entries += [e for item in checked for target in item['bindings'] for e in target['input_layers']]
            for e in entries:
                target=shadow/e['file'];target.parent.mkdir(parents=True,exist_ok=True)
                if not target.is_symlink():target.symlink_to((a.original_pack/e['file']).resolve())
            bad=copy.deepcopy(manifest);r=copy.deepcopy(recipe);fr=bad['maps'][0]['frames'][248]
            path=shadow/fr['input_layers'][layer]['file'];im=Image.open(path).copy();path.unlink();im.putpixel((0,0),0 if layer==1 else 255);im.save(path)
            fr['input_layers'][layer]['sha256']=sha(path)
            raw,opaque,remap=[Image.open(shadow/e['file']).tobytes() for e in fr['input_layers']]
            content=hashlib.sha256(raw+bytes(bool(v) for v in opaque)+bytes(bool(v) for v in remap)).hexdigest()
            fr['native_content_sha256']=content;r['groups'][0]['native_content_sha256']=content
            reject(name,lambda:validate_sources(shadow,maps,bad,r),'原生遮罩')
        g=copy.deepcopy(recipe['groups'][0]);g['painted_crop'][2]-=100
        reject('clipped-high-alpha',lambda:painted_crop(Image.open(a.candidate/'pack'/recipe['masters']['chests']['file']),g),'主體裁切')
        assert len(checks)==10
        rebuilt=tmp/'rebuilt'
        build(a.original_pack,maps,a.sources,a.base_catalog,a.base_pack,a.recipe,a.candidate/'pack',rebuilt)
        assert (rebuilt/'catalog.json').read_bytes()==(a.candidate/'catalog.json').read_bytes()
        assert (rebuilt/'preview.png').read_bytes()==(a.candidate/'preview.png').read_bytes()
        for e in [*proof['masters'],*proof['frames']]:assert sha(rebuilt/'pack'/e['file'])==e['sha256']
        bad=copy.deepcopy(recipe);bad['source_refs']=['docs/data/fd2-modern-hd-battlefield-20261019.json#invalid']
        bad_recipe=tmp/'bad-source.json';bad_recipe.write_text(json.dumps(bad));output=tmp/'rejected';before=sha(a.base_catalog)
        try:build(a.original_pack,maps,a.sources,a.base_catalog,a.base_pack,bad_recipe,a.candidate/'pack',output)
        except ValueError as error:assert 'invalid source_ref' in str(error)
        else:raise AssertionError('錯來源引用未拒收')
        assert not output.exists() and sha(a.base_catalog)==before
        report={'status':'PASS-PROPS-GUARDS','recipe_sha256':sha(a.recipe),'tool_sha256':sha(Path(__file__)),
                'source_groups':6,'target_slots':10,'checked_composite_pixels':pixels,'high_covered_source_pixels':high_pixels,
                'checks':checks,'frame_sha256':{e['file']:sha(a.candidate/'pack'/e['file']) for e in proof['frames']},
                'rebuild_matches_reference':True,'invalid_source_ref_atomic_rejection':True}
        original=a.base_catalog.read_bytes();catalog=tmp/'catalog.json';catalog.write_bytes(original);assets=tmp/'assets';assets.mkdir();guards=tmp/'guards.json';guards.write_text(json.dumps(report))
        positive=register(a.candidate,a.gpu,guards,catalog,assets);assert positive['slots']==10 and len(list(assets.iterdir()))==8
        gpu=json.loads(a.gpu.read_text());rejected=[]
        for name in ['catalog','rgba','missing-case','missing-frame','guard-frame','wrong-writer','missing-opened']:
            catalog.write_bytes(original);bad=copy.deepcopy(gpu);bg=copy.deepcopy(report);first=proof['frames'][0]['file']
            if name=='catalog':bad['catalog_sha256']='0'*64
            if name=='rgba':bad['painted_frames_rgba'][first]='0'*64
            if name=='missing-case':bad['cases'].pop()
            if name=='missing-frame':bad['cases'][0]['source_frames'].pop()
            if name=='guard-frame':bg['frame_sha256'][first]='0'*64
            if name=='wrong-writer':bad['opened_selector_writer']='invented'
            if name=='missing-opened':next(row for row in bad['cases'] if row['case']=='game-map00-opened')['source_frames'].remove(249)
            gp=tmp/('gpu-'+name+'.json');gp.write_text(json.dumps(bad));guards.write_text(json.dumps(bg));empty=tmp/name;empty.mkdir()
            try:register(a.candidate,gp,guards,catalog,empty)
            except ValueError as error:rejected.append({'case':name,'error':str(error)})
            else:raise AssertionError('錯物件收據未拒收：'+name)
            assert catalog.read_bytes()==original and not list(empty.iterdir())
        report['registration_positive']=positive;report['registration_rejections']=rejected
    a.report.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print('PASS',len(checks),'source guards',len(rejected),'registration guards',pixels,'pixels')


if __name__=='__main__':main()
