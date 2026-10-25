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
    edges=proof['status']=='PASS-GROUND-EDGES-CANDIDATE'
    if proof['status'] not in ('PASS-GROUND-CANDIDATE','PASS-GROUND-EDGES-CANDIDATE') or sha(catalog_path)!=proof['base_catalog_sha256']:
        raise ValueError('正式清冊已變或候選未通過')
    if catalog['assets'][:len(base['assets'])]!=base['assets'] or len(catalog['assets'])!=len(base['assets'])+12:
        raise ValueError('草地候選改動既有資產或新增範圍不符')
    expected_gpu='PASS-GROUND-EDGES-GPU' if edges else 'PASS-SHARED-GROUND-GPU'
    if (gpu.get('status')!=expected_gpu or gpu.get('catalog_sha256')!=sha(candidate/'catalog.json') or
        gpu.get('maps')!=11 or gpu.get('source_worlds')!=66 or gpu.get('game_draws')!=1 or
        gpu.get('f2')!=1 or gpu.get('wrong_bank_masks_rejected')!=11 or len(gpu.get('cases',[]))!=67):
        raise ValueError('缺少同清冊完整草地GPU收據')
    expected={(e['map'],e['frame']) for group in proof['frames'] for e in group['targets']} if edges else {(e['map'],e['frame']) for e in proof['targets']}
    rows=gpu['cases'];modes={'steady','up1','up6','lut','black','white'}
    if edges:
        if gpu.get('source_groups')!=14 or gpu.get('target_slots')!=141 or gpu.get('recipe_sha256')!=proof['recipe_sha256']:
            raise ValueError('交界GPU來源或配方不同')
        for m in {m for m,f in expected}:
            frames={f for map_id,f in expected if map_id==m}
            for mode in modes:
                found=[r for r in rows if r['case']==f'map{m:02d}-{mode}' and r['map']==m and not r['game_draw']]
                if len(found)!=1 or set(found[0]['source_frames'])!=frames:raise ValueError('GPU實際圖源範圍不完整')
    else:
        for m,f in expected:
            names={r['case'] for r in rows if r['map']==m and r['source_frame']==f and not r['game_draw']}
            if names!={f'map{m:02d}-{mode}' for mode in modes}:raise ValueError('GPU實際圖源範圍不完整')
    expected_guards='PASS-GROUND-EDGES-GUARDS' if edges else 'PASS-GROUND-GUARDS'
    if guards.get('status')!=expected_guards or guards.get('recipe_sha256')!=proof['recipe_sha256'] or len(guards.get('checks',[]))!=10:
        raise ValueError('缺少同配方完整草地反例檢查')
    files=[proof['earth_master'],*proof['frames']] if edges else [proof['master'],proof['tile']]
    for entry in files:
        if Path(entry['file']).name!=entry['file'] or sha(candidate/'pack'/entry['file'])!=entry['sha256']:
            raise ValueError('候選圖源不同')
    if edges:
        for frame in proof['frames']:
            rgba=hashlib.sha256(Image.open(candidate/'pack'/frame['file']).convert('RGBA').tobytes()).hexdigest()
            if gpu.get('painted_frames_rgba',{}).get(frame['file'])!=rgba or guards['frame_sha256'].get(frame['file'])!=frame['sha256']:
                raise ValueError('GPU／反例不是同一草地圖源')
    else:
        tile=candidate/'pack'/proof['tile']['file']
        rgba=hashlib.sha256(Image.open(tile).convert('RGBA').tobytes()).hexdigest()
        if gpu.get('painted_rgba_sha256')!=rgba or guards['tile_sha256']!=sha(tile):
            raise ValueError('GPU／反例不是同一草地圖源')
    additions=catalog['assets'][len(base['assets']):]
    targets={(e['map_id'],f) for e in additions if e['role']=='native_terrain_hd_set' for f in e['frame_ids']}
    if targets!=expected:raise ValueError('候選新增槽與來源不同')
    for entry in files:
        dest=asset_root/entry['file']
        if dest.exists() and sha(dest)!=entry['sha256']:raise ValueError('拒絕覆寫不同素材')
    for entry in files:
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
