#!/usr/bin/env python3
"""只登錄來源、接縫、守衛與完整GPU已通過的共用純地形候選。"""
import argparse
import json
import os
from pathlib import Path
import shutil
import tempfile
import hashlib

from PIL import Image
from verify_fd2_terrain_sources import sha


def register(candidate, gpu_path, guards_path, catalog_path, asset_root):
    proof=json.loads((candidate/'manifest.json').read_text())
    catalog=json.loads((candidate/'catalog.json').read_text())
    base=json.loads(catalog_path.read_text())
    gpu=json.loads(gpu_path.read_text());guards=json.loads(guards_path.read_text())
    if proof['status']!='PASS-GROUND-CANDIDATE' or sha(catalog_path)!=proof['base_catalog_sha256']:
        raise ValueError('正式清冊已變或候選未通過')
    if catalog['assets'][:len(base['assets'])]!=base['assets'] or len(catalog['assets'])!=len(base['assets'])+12:
        raise ValueError('草地候選改動既有資產或新增範圍不符')
    if (gpu.get('status')!='PASS-SHARED-GROUND-GPU' or gpu.get('catalog_sha256')!=sha(candidate/'catalog.json') or
        gpu.get('maps')!=11 or gpu.get('source_worlds')!=66 or gpu.get('game_draws')!=1 or
        gpu.get('f2')!=1 or gpu.get('wrong_bank_masks_rejected')!=11 or len(gpu.get('cases',[]))!=67):
        raise ValueError('缺少同清冊完整草地GPU收據')
    expected={(e['map'],e['frame']) for e in proof['targets']}
    rows=gpu['cases'];modes={'steady','up1','up6','lut','black','white'}
    for m,f in expected:
        names={r['case'] for r in rows if r['map']==m and r['source_frame']==f and not r['game_draw']}
        if names!={f'map{m:02d}-{mode}' for mode in modes}:raise ValueError('GPU實際圖源範圍不完整')
    if guards.get('status')!='PASS-GROUND-GUARDS' or guards.get('recipe_sha256')!=proof['recipe_sha256'] or len(guards.get('checks',[]))!=10:
        raise ValueError('缺少同配方完整草地反例檢查')
    for entry in [proof['master'],proof['tile']]:
        if Path(entry['file']).name!=entry['file'] or sha(candidate/'pack'/entry['file'])!=entry['sha256']:
            raise ValueError('候選圖源不同')
    tile=candidate/'pack'/proof['tile']['file']
    rgba=hashlib.sha256(Image.open(tile).convert('RGBA').tobytes()).hexdigest()
    if gpu.get('painted_rgba_sha256')!=rgba or guards['tile_sha256']!=sha(tile):
        raise ValueError('GPU／反例不是同一草地圖源')
    additions=catalog['assets'][len(base['assets']):]
    targets={(e['map_id'],f) for e in additions if e['role']=='native_terrain_hd_set' for f in e['frame_ids']}
    if targets!=expected:raise ValueError('候選新增槽與來源不同')
    for entry in [proof['master'],proof['tile']]:
        dest=asset_root/entry['file']
        if dest.exists() and sha(dest)!=entry['sha256']:raise ValueError('拒絕覆寫不同素材')
    for entry in [proof['master'],proof['tile']]:
        dest=asset_root/entry['file']
        if not dest.exists():shutil.copyfile(candidate/'pack'/entry['file'],dest)
    with tempfile.NamedTemporaryFile('w',dir=catalog_path.parent,delete=False,encoding='utf-8') as out:
        out.write(json.dumps(catalog,ensure_ascii=False,indent=2)+'\n');tmp=Path(out.name)
    os.chmod(tmp,catalog_path.stat().st_mode&0o777);tmp.replace(catalog_path)
    return {'status':'PASS-GROUND-REGISTERED','slots':len(expected),'assets':len(catalog['assets']),
            'catalog_sha256':sha(catalog_path),'gpu_sha256':sha(gpu_path),'guards_sha256':sha(guards_path)}


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ['candidate','gpu','guards','catalog','asset-root','report']:p.add_argument('--'+n,type=Path,required=True)
    a=p.parse_args()
    if a.report.exists():raise ValueError('不得覆寫登錄收據')
    report=register(a.candidate,a.gpu,a.guards,a.catalog,a.asset_root)
    a.report.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n');print(json.dumps(report))


if __name__=='__main__':main()
