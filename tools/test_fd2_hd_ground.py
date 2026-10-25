#!/usr/bin/env python3
"""以實際原生來源與候選驗證草地共用邊界，不重製原版規則。"""
import argparse
import copy
import json
from pathlib import Path
import tempfile

from PIL import Image
from build_fd2_hd_ground import build, ground_sources, make_tile, sha


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ['original-pack','sources','recipe','candidate','master','report']:
        p.add_argument('--'+n,type=Path,required=True)
    p.add_argument('--gpu',type=Path)
    p.add_argument('--base-catalog',type=Path,help='GPU登錄反例使用的769項清冊；正式登錄後可明示已保存基底')
    a=p.parse_args()
    if a.report.exists():raise ValueError('不得覆寫檢查收據')
    recipe=json.loads(a.recipe.read_text());manifest=json.loads((a.sources/'manifest.json').read_text())
    bindings=ground_sources(a.original_pack,manifest,recipe)
    candidate=json.loads((a.candidate/'manifest.json').read_text())
    assert candidate['targets']==bindings and candidate['status']=='PASS-GROUND-CANDIDATE'
    tile=Image.open(a.candidate/'pack'/recipe['tile_file']).convert('RGBA')
    assert tile.size==(144,144) and tile.getextrema()[3]==(255,255)
    assert all(tile.getpixel((0,i))==tile.getpixel((143,i)) and tile.getpixel((i,0))==tile.getpixel((i,143)) for i in range(144))
    checks=[]
    def reject(name,call,text):
        try:call()
        except (ValueError,FileNotFoundError) as e:
            assert text in str(e),(name,str(e));checks.append({'case':name,'error':str(e)})
        else:raise AssertionError('反例未拒收：'+name)
    for name,change in [
        ('missing-map',lambda r:r['targets'].pop()),
        ('unused-frame',lambda r:r['targets'].append([0,0])),
        ('duplicate',lambda r:r['targets'].append(r['targets'][0])),
        ('boolean-map',lambda r:r['targets'][0].__setitem__(0,False)),
        ('different-content',lambda r:r.__setitem__('native_content_sha256','0'*64)),
    ]:
        r=copy.deepcopy(recipe);change(r)
        reject(name,lambda r=r:ground_sources(a.original_pack,manifest,r),'草地')
    bad=copy.deepcopy(manifest);bad['maps'][0]['frames'][144]['input_layers'][0]['sha256']='0'*64
    reject('wrong-source-binding',lambda:ground_sources(a.original_pack,bad,recipe),'雜湊不符')
    with tempfile.TemporaryDirectory(dir=a.report.parent) as tmp:
        tmp=Path(tmp); bad_master=tmp/'transparent.png';Image.new('RGBA',(144,144),(1,2,3,0)).save(bad_master)
        reject('transparent-master',lambda:make_tile(bad_master,144,8),'不透明')
        reject('edge-budget',lambda:make_tile(a.master,144,30),'越界')
        before=sha(a.candidate/'manifest.json')
        reject('existing-output',lambda:build(None,None,None,None,None,None,None,a.candidate),'新目錄')
        assert sha(a.candidate/'manifest.json')==before
        wrong=tmp/'catalog.json';wrong.write_text('{}')
        reject('stale-base-catalog',lambda:build(a.original_pack,None,a.sources,wrong,None,a.master,a.recipe,tmp/'out'),'綁定不同')
        assert not (tmp/'out').exists()
    result={'status':'PASS-GROUND-GUARDS','source_slots':len(bindings),'tile_sha256':sha(a.candidate/'pack'/recipe['tile_file']),'recipe_sha256':sha(a.recipe),'tool_sha256':sha(Path(__file__)),'checks':checks,'scope':'實際來源綁定、錯槽／缺槽、透明度、接縫及既有輸出保護；非GPU或玩家對拍。'}
    if a.gpu:
        from register_fd2_hd_ground import register
        original_catalog=a.base_catalog or Path(__file__).resolve().parents[1]/'remake/assets/themes/modern/catalog.json'
        original=original_catalog.read_bytes();gpu=json.loads(a.gpu.read_text())
        registration=[]
        with tempfile.TemporaryDirectory(dir=a.report.parent) as tmp:
            tmp=Path(tmp);catalog=tmp/'catalog.json';catalog.write_bytes(original)
            assets=tmp/'assets';assets.mkdir();guards=tmp/'guards.json';guards.write_text(json.dumps(result))
            positive=register(a.candidate,a.gpu,guards,catalog,assets)
            assert positive['status']=='PASS-GROUND-REGISTERED' and len(list(assets.iterdir()))==2
            actual=json.loads(catalog.read_text());base=json.loads(original)
            assert actual['assets'][:len(base['assets'])]==base['assets']
            for name in ['catalog','painted-rgba','missing-case','wrong-slot','guard-tile']:
                catalog.write_bytes(original);bad=copy.deepcopy(gpu);badguards=copy.deepcopy(result)
                if name=='catalog':bad['catalog_sha256']='0'*64
                if name=='painted-rgba':bad['painted_rgba_sha256']='0'*64
                if name=='missing-case':bad['cases'].pop()
                if name=='wrong-slot':bad['cases'][0]['source_frame']+=1
                if name=='guard-tile':badguards['tile_sha256']='0'*64
                path=tmp/('gpu-'+name+'.json');path.write_text(json.dumps(bad));guards.write_text(json.dumps(badguards))
                empty=tmp/name;empty.mkdir()
                try:register(a.candidate,path,guards,catalog,empty)
                except ValueError as e:registration.append({'case':name,'error':str(e)})
                else:raise AssertionError('錯誤登錄收據未拒收：'+name)
                assert catalog.read_bytes()==original and not list(empty.iterdir()),name
            result['registration_positive']=positive
            result['registration_rejections']=registration
    a.report.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print('PASS',len(bindings),'slots',len(checks),'guards')


if __name__=='__main__':main()
