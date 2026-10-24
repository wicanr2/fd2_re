#!/usr/bin/env python3
"""在素材Docker編製已審查FDOTHER42母稿；先核固定原包，再建立候選。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil

from PIL import Image
from validate_modern_theme_catalog import ROOT, validate


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ['original-pack','master','base-catalog','out']:
        p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args()
    if (not Path('/.dockerenv').exists() or a.out.exists() or not a.out.parent.is_dir()
            or a.out.parent.stat().st_uid!=os.getuid()):
        raise ValueError('Docker內使用本人擁有的全新輸出')
    state=json.loads((ROOT/'docs/data/fd2-modern-hd-battlefield-20261019.json').read_text())['special_scene_hd_followup_20261025']['stage42']
    if state['status'] not in {'READY-ART-REVIEWED','CONFORMED-SCOPED-RUNTIME-E1'} or sha(a.master)!=state['master']['sha256'] or sha(a.base_catalog)!=state['base_catalog_sha256']:
        raise ValueError('未審原稿或基底SHA不同')
    pinned=json.loads((ROOT/'docs/data/fd2-battle-hd-source-inventory-20261019.json').read_text())
    if sha(a.original_pack/'manifest.json')!=pinned['pack_manifest_sha256']:
        raise ValueError('固定原包manifest不同')
    manifest=json.loads((a.original_pack/'manifest.json').read_text())
    source=next(x for x in manifest['source_resources'] if x['source_file']=='FDOTHER.DAT' and x['source_resource']==42)
    raw=next(x for x in manifest['assets'] if x['asset_id']==source['raw_asset_id'])
    if (sha(a.original_pack/raw['path'])!=state['source_metadata']['source']['raw_sha256']
            or (a.original_pack/raw['path']).stat().st_size!=59412):
        raise ValueError('原始FDOTHER42資源不同')
    for name,key in [('resource.json','metadata_sha256'),('frame.png','frame_png_sha256'),('mask.png','mask_png_sha256')]:
        if sha(a.original_pack/'surfaces/FDOTHER_042'/name)!=state[key]:
            raise ValueError('原圖來源不同：'+name)
    with Image.open(a.original_pack/'surfaces/FDOTHER_042/frame.png') as im:
        if im.mode!='P' or im.size!=(312,192) or hashlib.sha256(im.tobytes()).hexdigest()!=state['indexed_pixels_sha256']:
            raise ValueError('原始索引圖不同')
    with Image.open(a.original_pack/'surfaces/FDOTHER_042/mask.png') as im:
        if im.mode!='L' or im.size!=(312,192) or im.getextrema()!=(255,255):
            raise ValueError('固定原圖不是全不透明')
    with Image.open(a.master) as im:
        if im.mode!='RGB' or im.size!=(1599,984):
            raise ValueError('已審母稿形狀不同')
        picture=im.resize((1872,1152),Image.Resampling.LANCZOS).convert('RGBA')
    catalog=validate(True,a.base_catalog)
    if any(x['role']=='map_stage_hd' for x in catalog['assets']):
        raise ValueError('圖層已登錄')
    names=set()
    for e in catalog['assets']:
        names.update(e.get('files',[]));names.update(e.get('source_files',[]))
        names.update(e[k] for k in ['file','master_file'] if k in e)
    base=ROOT/catalog['private_root']
    master_name='map-stage-fdother042-master-oct25-r1.png';name='map-stage-fdother042-hd6-oct25-r1.png'
    if {master_name,name}&names or any(Path(n).name!=n or not (base/n).is_file() for n in names):
        raise ValueError('舊白名單缺件或名稱碰撞')
    a.out.mkdir()
    for n in sorted(names):
        (a.out/n).symlink_to(os.path.relpath(base/n,a.out))
    shutil.copyfile(a.master,a.out/master_name);picture.save(a.out/name)
    entry=dict(asset_id='modern.map.fdother042.stage.hd6',role='map_stage_hd',status='runtime_candidate',
               file=name,width=1872,height=1152,sha256=sha(a.out/name),master_file=master_name,
               consumer_contract='native_map_stage_rgba_hd6_v1',source_archive='FDOTHER.DAT',source_resource=42,
               source_metadata_sha256=state['metadata_sha256'],source_files=[master_name],source_sha256=[sha(a.master)],
               native_frames=[dict(x=0,y=0,width=312,height=192,delay_native=0,raw_byte_4=0,raw_byte_5=0,raw_byte_7=0)],
               alpha_contract='coverage',layout_policy='native_map_stage_hd6',
               source_refs=['asset:FDOTHER_042','docs/knowledge-base/61-modern-theme-runtime-spec.md'])
    catalog['assets'].append(entry);catalog['private_root']=str(a.out.resolve().relative_to(ROOT))
    out=a.out/'catalog.json';out.write_text(json.dumps(catalog,ensure_ascii=False,indent=2)+'\n');validate(True,out)
    report=dict(status='PASS-MAP-STAGE-CANDIDATE',base_catalog_sha256=sha(a.base_catalog),catalog_sha256=sha(out),
                tool_sha256=sha(Path(__file__)),source=state['source_metadata'],entry=entry,old_files_are_links=True,
                prior_files=len(names),recipe=state['compile'],production_changed=False)
    (a.out/'recipe.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':report['status'],'assets':len(catalog['assets']),'sha256':entry['sha256']}))


if __name__=='__main__':
    main()
