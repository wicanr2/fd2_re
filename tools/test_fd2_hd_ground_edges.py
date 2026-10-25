#!/usr/bin/env python3
"""以實際來源驗交界遮罩、圖源、登錄原子性與私人包同檔同步。"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

from PIL import Image
from build_fd2_hd_ground import make_tile
from build_fd2_hd_ground_edges import validate_sources, composite_materials, build, ground_profile
from verify_fd2_terrain_sources import sha


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ['original-pack','sources','recipe','candidate','base-pack','base-catalog','report']:p.add_argument('--'+n,type=Path,required=True)
    p.add_argument('--gpu',type=Path)
    p.add_argument('--rebuild-reference',action='store_true',help='完整重編並逐byte核對既有候選，驗共用編製器相容性')
    a=p.parse_args()
    if a.report.exists():raise ValueError('不得覆寫交界收據')
    recipe=json.loads(a.recipe.read_text());manifest=json.loads((a.sources/'manifest.json').read_text());proof=json.loads((a.candidate/'manifest.json').read_text())
    count,slots,maps,bases=ground_profile(recipe)
    shore=recipe['kind']=='fd2_shore_hd_v1'
    relief=recipe['kind']=='fd2_relief_hd_v1'
    checked=validate_sources(a.original_pack,manifest,recipe);assert len(checked)==count and proof['target_slots']==slots
    anchor=recipe['groups'][0]['anchor'][1]
    keys=['grass','earth','stone'] if relief else ['grass','sand','water'] if shore else ['grass','earth']
    materials={k:make_tile((a.base_pack if k=='grass' or relief and k=='earth' else a.candidate/'pack')/recipe[k+'_master_file'],144,8)[0] for k in keys}
    pixels=0
    for item in checked:
        tile=Image.open(a.candidate/'pack'/item['group']['file']).convert('RGBA');assert tile.size==(144,144) and tile.getextrema()[3]==(255,255)
        pair=item['group'].get('materials',['grass','earth'])
        if len(pair)==1 and not relief:assert tile.tobytes()==materials[pair[0]].tobytes(),'純底材必須完整等於對應的新手繪材質'
        if bases and item['group']['anchor'][1] in (52,53):
            expected=materials['grass'] if item['group']['anchor'][1]==52 else materials['earth']
            assert tile.tobytes()==expected.tobytes(),'純底材必須完整等於對應的新手繪材質'
        for y in range(144):
            for x in range(144):
                actual=tile.getpixel((x,y));u=materials[pair[0]].getpixel((x,y));v=materials[pair[-1]].getpixel((x,y))
                if relief:
                    assert all(round(min(im.getpixel((x,y))[c] for im in materials.values())*115/255)<=actual[c]<=max(im.getpixel((x,y))[c] for im in materials.values()) for c in range(3))
                else:assert all(min(s,t)<=c<=max(s,t) for c,s,t in zip(actual[:3],u[:3],v[:3]))
                if x in (0,143) or y in (0,143):
                    label=next(k for k in pair if item['pixels'][y//6*24+x//6] in recipe['index_groups'][k])
                    expected=materials[label].getpixel((x,y))
                    if relief:expected=(*[round(c*recipe['index_shades'][str(item['pixels'][y//6*24+x//6])]/255) for c in expected[:3]],255)
                    assert actual==expected
                pixels+=1
    checks=[]
    def reject(name,call,text):
        try:call()
        except ValueError as e:
            assert text in str(e),(name,str(e));checks.append({'case':name,'error':str(e)})
        else:raise AssertionError('反例未拒收：'+name)
    for name,change,text in [
        ('overlap-indices',lambda r:r['index_groups']['sand' if shore else 'earth'].append(70),'分組'),
        ('unknown-indices',lambda r:r['index_groups']['grass'].remove(70),'藝術索引'),
        ('missing-target',lambda r:r['groups'][0]['targets'].pop(),'consumer'),
        ('duplicate-group',lambda r:r['groups'][0].__setitem__('anchor',r['groups'][1]['anchor']),'組數'),
        ('missing-group',lambda r:r['groups'].pop(),'組數'),
        ('unsafe-file',lambda r:r['groups'][0].__setitem__('file','../bad.png'),'越界'),
    ]:
        r=copy.deepcopy(recipe);change(r);reject(name,lambda r=r:validate_sources(a.original_pack,manifest,r),text)
    bad=copy.deepcopy(manifest);bad['maps'][0]['frames'][anchor]['input_layers'][0]['sha256']='0'*64
    reject('wrong-binding',lambda:validate_sources(a.original_pack,bad,recipe),'雜湊不符')
    with tempfile.TemporaryDirectory(dir=a.report.parent) as tmp:
        tmp=Path(tmp);shadow=tmp/'shadow';bad=copy.deepcopy(manifest);r=copy.deepcopy(recipe);fr=bad['maps'][0]['frames'][anchor]
        for entry in fr['input_layers']:
            target=shadow/entry['file'];target.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(a.original_pack/entry['file'],target)
        mask=shadow/fr['input_layers'][1]['file'];im=Image.open(mask).copy();im.putpixel((0,0),0);im.save(mask);fr['input_layers'][1]['sha256']=sha(mask)
        raw,opaque,remap=[Image.open(shadow/e['file']).tobytes() for e in fr['input_layers']]
        changed=hashlib.sha256(raw+bytes(bool(v) for v in opaque)+bytes(bool(v) for v in remap)).hexdigest();fr['native_content_sha256']=changed;r['groups'][0]['native_content_sha256']=changed
        reject('wrong-native-mask',lambda:validate_sources(shadow,bad,r),'原生圖層')
        r=copy.deepcopy(recipe);r['mask_blur_hd_pixels']=10
        reject('blur-budget',lambda:composite_materials(checked[0]['pixels'],materials[keys[0]],materials[keys[1]],r),'範圍')
        reject('existing-output',lambda:build(None,None,None,None,None,None,None,a.candidate),'新目錄')
        if a.rebuild_reference:
            rebuilt=tmp/'rebuilt'
            build(a.original_pack,Path(__file__).resolve().parents[1]/'remake/assets/maps',a.sources,a.base_catalog,
                  a.base_pack,a.base_pack/recipe['earth_master_file'] if relief else None if shore else a.candidate/'pack'/recipe['earth_master_file'],a.recipe,rebuilt,
                  {k:a.candidate/'pack'/recipe[k+'_master_file'] for k in (['stone'] if relief else ['sand','water'])} if shore or relief else None)
            assert (rebuilt/'catalog.json').read_bytes()==(a.candidate/'catalog.json').read_bytes()
            assert (rebuilt/'preview.png').read_bytes()==(a.candidate/'preview.png').read_bytes()
            for frame in proof['frames']:assert sha(rebuilt/'pack'/frame['file'])==frame['sha256']
        if bases or shore or relief:
            bad_recipe=copy.deepcopy(recipe);bad_recipe['source_refs']=['docs/data/fd2-modern-hd-battlefield-20261019.json#invalid']
            recipe_path=tmp/'bad-source-ref.json';recipe_path.write_text(json.dumps(bad_recipe));output=tmp/'rejected-candidate'
            before=sha(a.base_catalog)
            try:
                build(a.original_pack,Path(__file__).resolve().parents[1]/'remake/assets/maps',a.sources,
                      a.base_catalog,a.base_pack,a.base_pack/recipe['earth_master_file'] if relief else None if shore else a.candidate/'pack'/recipe['earth_master_file'],recipe_path,output,
                      {k:a.candidate/'pack'/recipe[k+'_master_file'] for k in (['stone'] if relief else ['sand','water'])} if shore or relief else None)
            except ValueError as error:assert 'invalid source_ref' in str(error),str(error)
            else:raise AssertionError('來源引用錯誤未阻擋候選發布')
            assert not output.exists() and sha(a.base_catalog)==before
        sync_source=tmp/'sync-source';sync_dest=tmp/'sync-dest';sync_source.mkdir();sync_dest.mkdir()
        Image.new('RGBA',(2,2),(1,2,3,255)).save(sync_dest/'same.png');(sync_source/'same.png').symlink_to((sync_dest/'same.png').resolve())
        Image.new('RGBA',(2,2),(7,8,9,255)).save(sync_source/'equal.png');shutil.copyfile(sync_source/'equal.png',sync_dest/'equal.png')
        os.utime(sync_dest/'equal.png',ns=(1000000000,1000000000));equal_before=(sync_dest/'equal.png').stat()
        Image.new('RGBA',(2,2),(4,5,6,255)).save(sync_source/'new.png')
        catalog=tmp/'sync-catalog.json';meta={'theme_id':'modern-handpainted-a','assets':[{'file':n,'sha256':sha(sync_source/n),'status':'runtime_candidate'} for n in ['same.png','new.png','equal.png']]};catalog.write_text(json.dumps(meta))
        argv=[sys.executable,str(Path(__file__).with_name('sync_modern_theme_private.py')),'--catalog',str(catalog),'--source',str(sync_source),'--destination',str(sync_dest)]
        before=(sync_dest/'same.png').stat()
        for i in range(2):
            result=subprocess.run(argv,capture_output=True,text=True,timeout=20);assert result.returncode==0,(result.stdout,result.stderr)
            if i==0:first=(sync_dest/'private-manifest.json').read_bytes()
            else:assert first==(sync_dest/'private-manifest.json').read_bytes()
        after=(sync_dest/'same.png').stat();assert before.st_ino==after.st_ino and before.st_mtime_ns==after.st_mtime_ns
        equal_after=(sync_dest/'equal.png').stat();assert equal_before.st_ino==equal_after.st_ino and equal_before.st_mtime_ns==equal_after.st_mtime_ns
        meta['assets'][1]['sha256']='0'*64;catalog.write_text(json.dumps(meta));result=subprocess.run(argv,capture_output=True,text=True,timeout=20);assert result.returncode!=0 and 'digest mismatch: new.png' in result.stderr
    report={'status':'PASS-RELIEF-GUARDS' if relief else 'PASS-SHORE-GUARDS' if shore else 'PASS-GROUND-BASES-GUARDS' if bases else 'PASS-GROUND-EDGES-GUARDS','recipe_sha256':sha(a.recipe),'tool_sha256':sha(Path(__file__)),'source_groups':count,'target_slots':slots,'checked_composite_pixels':pixels,'checks':checks,'frame_sha256':{f['file']:sha(a.candidate/'pack'/f['file']) for f in proof['frames']},'private_alias_sync':{'status':'PASS','same_inode_mtime_preserved':True,'equal_regular_file_inode_mtime_preserved':True,'idempotent_manifest_bytes':True,'wrong_digest_rejected':True},'scope':'來源及藝術遮罩／材質／同步反例；非原版RGB或正常玩家。'}
    assert len(checks)==10
    if a.rebuild_reference:report['rebuild_matches_reference']=True
    if bases:
        report['pure_material_frames']={'52':'exact grass material','53':'exact earth material'}
    if shore:report['pure_material_frames']={'49':'exact water material','50':'exact sand material'}
    if relief:
        extra=[]
        for name,change,text in [
            ('wrong-relief-shade',lambda r:r['index_shades'].__setitem__('111',0),'暗部契約'),
            ('missing-relief-stone',lambda r:r['groups'][3]['materials'].remove('stone'),'原生圖層'),
        ]:
            r=copy.deepcopy(recipe);change(r)
            try:validate_sources(a.original_pack,manifest,r)
            except ValueError as error:
                assert text in str(error),(name,str(error));extra.append({'case':name,'error':str(error)})
            else:raise AssertionError('起伏地形反例未拒收：'+name)
        report['relief_rejections']=extra
    if bases or shore or relief:
        report['invalid_source_ref_atomic_rejection']=True
    if a.gpu:
        from register_fd2_hd_ground import register
        gpu=json.loads(a.gpu.read_text());original=a.base_catalog.read_bytes();rejected=[]
        with tempfile.TemporaryDirectory(dir=a.report.parent) as tmp:
            tmp=Path(tmp);catalog=tmp/'catalog.json';catalog.write_bytes(original);assets=tmp/'assets';assets.mkdir();guards=tmp/'guards.json';guards.write_text(json.dumps(report))
            positive=register(a.candidate,a.gpu,guards,catalog,assets);assert positive['slots']==slots and len(list(assets.iterdir()))==count+(2 if shore else not bases)
            for name in ['catalog','rgba','missing-case','missing-frame','guard-frame']:
                catalog.write_bytes(original);bad=copy.deepcopy(gpu);badguards=copy.deepcopy(report)
                first=proof['frames'][0]['file']
                if name=='catalog':bad['catalog_sha256']='0'*64
                if name=='rgba':bad['painted_frames_rgba'][first]='0'*64
                if name=='missing-case':bad['cases'].pop()
                if name=='missing-frame':bad['cases'][0]['source_frames'].pop()
                if name=='guard-frame':badguards['frame_sha256'][first]='0'*64
                path=tmp/('gpu-'+name+'.json');path.write_text(json.dumps(bad));guards.write_text(json.dumps(badguards));empty=tmp/name;empty.mkdir()
                try:register(a.candidate,path,guards,catalog,empty)
                except ValueError as e:rejected.append({'case':name,'error':str(e)})
                else:raise AssertionError('錯交界收據未拒收：'+name)
                assert catalog.read_bytes()==original and not list(empty.iterdir())
            report['registration_positive']=positive;report['registration_rejections']=rejected
    a.report.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n');print('PASS',len(checks),'guards',pixels,'pixels')


if __name__=='__main__':main()
