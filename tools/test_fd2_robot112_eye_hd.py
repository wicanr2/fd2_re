#!/usr/bin/env python3
"""驗證112光束候選的正對照與有完整重綁SHA的錯位反例。"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path

from PIL import Image

from validate_modern_theme_catalog import ROOT
from verify_fd2_robot112_eye_hd import verify


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for key in ['candidate','inventory','original','out']:
        p.add_argument('--'+key,type=Path,required=True)
    a=p.parse_args()
    assert Path('/.dockerenv').exists() and not a.out.exists()
    assert a.out.is_relative_to(ROOT) and a.out.parent.stat().st_uid==os.getuid()
    positive=verify(a.candidate/'compiled.json',a.candidate/'pack',a.inventory,a.original)
    a.out.mkdir()
    digest=lambda path:hashlib.sha256(path.read_bytes()).hexdigest()
    base=json.loads((a.candidate/'compiled.json').read_text())
    basecat=json.loads((a.candidate/'compiler-catalog.json').read_text())
    checks=[dict(case='positive',status=positive['status'])]

    def check(name,mutate):
        folder=a.out/name
        folder.mkdir()
        pack=folder/'pack'
        pack.mkdir()
        for source in (a.candidate/'pack').iterdir():
            (pack/source.name).symlink_to(os.path.relpath(source,pack))
        report=copy.deepcopy(base)
        catalog=copy.deepcopy(basecat)
        catalog['private_root']=str(pack.relative_to(ROOT))
        mutate(folder,pack,report,catalog)
        catalog_path=folder/'compiler-catalog.json'
        catalog_path.write_text(json.dumps(catalog,ensure_ascii=False,indent=2)+'\n')
        report['catalog_sha256']=digest(catalog_path)
        path=folder/'compiled.json'
        path.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
        try:
            verify(path,pack,a.inventory,a.original)
        except (AssertionError,ValueError,KeyError,StopIteration,FileNotFoundError) as error:
            checks.append(dict(case=name,status='PASS-REJECTED',exception=type(error).__name__,reason=str(error)))
        else:
            raise AssertionError('錯誤光束候選未拒收：'+name)

    check('wrong-opaque-eye',lambda f,p,r,c:r['ray_source'].update(eye=[528,300]))
    check('wrong-opaque-axis',lambda f,p,r,c:r['ray_source'].update(axis=[[538,19],[538,1456]]))
    check('raw-sha',lambda f,p,r,c:r['ray_source'].update(sha256='0'*64))
    check('parent-sha',lambda f,p,r,c:r['parent_compiled'].update(sha256='0'*64))
    check('missing-last-frame',lambda f,p,r,c:r['frames'].pop())
    def wrong_descriptor(f,p,r,c):
        r['frames'][-1]['native_descriptor']['delay_native']+=1
    check('wrong-last-descriptor',wrong_descriptor)
    def wrong_phase(f,p,r,c):
        next(v for v in r['frames'] if v['frame']==16 and v['resource']==337)['ray_projection']['attached_to_eye']=True
    check('tail-attached',wrong_phase)
    def wrong_unchanged(f,p,r,c):
        next(v for v in r['frames'] if v['frame']==19 and v['resource']==337)['sha256']='0'*64
    check('last-ember-sha',wrong_unchanged)

    def altered_layer(folder,pack,report,catalog,mode,index):
        record=next(v for v in report['frames'] if v['resource']==337 and v['frame']==index)
        projection=record['ray_projection']
        with Image.open(ROOT/projection['layer_file']) as image:
            original=image.copy()
        layer=Image.new('RGBA',original.size)
        if mode=='shift':
            layer.alpha_composite(original,(16,0))
        layer_path=folder/'changed-ray.png'
        layer.save(layer_path)
        projection.update(layer_file=str(layer_path.relative_to(ROOT)),layer_sha256=digest(layer_path))
        parent_path=ROOT/report['parent_compiled']['file']
        with Image.open(parent_path.parent/'pack'/record['file']) as image:
            composite=image.copy()
        composite.alpha_composite(layer)
        (pack/record['file']).unlink()
        composite.save(pack/record['file'])
        record['sha256']=digest(pack/record['file'])
        catalog['assets'][-1]['frame_sha256'][index]=record['sha256']
    check('shifted-opaque-ray-rebound-shas',lambda f,p,r,c:altered_layer(f,p,r,c,'shift',5))
    check('empty-ray-rebound-shas',lambda f,p,r,c:altered_layer(f,p,r,c,'empty',5))
    check('shifted-detached-tail-rebound-shas',lambda f,p,r,c:altered_layer(f,p,r,c,'shift',16))
    assert len(checks)==12
    result=dict(status='PASS-ROBOT112-EYE-GUARDS',checks=checks,
        candidate_sha256=digest(a.candidate/'compiled.json'),verifier_sha256=digest(Path(__file__).with_name('verify_fd2_robot112_eye_hd.py')),
        scope='錯位與空白光束反例重新合成PNG、逐層及清冊SHA均重綁，仍須由原生走廊拒收。')
    (a.out/'guards.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(dict(status=result['status'],checks=len(checks))))


if __name__=='__main__':
    main()
