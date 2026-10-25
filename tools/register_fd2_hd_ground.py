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
    bases=proof['status']=='PASS-GROUND-BASES-CANDIDATE'
    shore=proof['status']=='PASS-SHORE-CANDIDATE'
    edges=proof['status'] in ('PASS-GROUND-EDGES-CANDIDATE','PASS-GROUND-BASES-CANDIDATE','PASS-SHORE-CANDIDATE')
    maps,groups,slots,additions_count=(9,33,41,11) if shore else (18,5,52,18) if bases else (11,14,141,12)
    if proof['status'] not in ('PASS-GROUND-CANDIDATE','PASS-GROUND-EDGES-CANDIDATE','PASS-GROUND-BASES-CANDIDATE','PASS-SHORE-CANDIDATE') or sha(catalog_path)!=proof['base_catalog_sha256']:
        raise ValueError('正式清冊已變或候選未通過')
    if catalog['assets'][:len(base['assets'])]!=base['assets'] or len(catalog['assets'])!=len(base['assets'])+additions_count:
        raise ValueError('草地候選改動既有資產或新增範圍不符')
    expected_gpu='PASS-SHORE-GPU' if shore else 'PASS-GROUND-BASES-GPU' if bases else 'PASS-GROUND-EDGES-GPU' if edges else 'PASS-SHARED-GROUND-GPU'
    if (gpu.get('status')!=expected_gpu or gpu.get('catalog_sha256')!=sha(candidate/'catalog.json') or
        gpu.get('maps')!=maps or gpu.get('source_worlds')!=maps*6 or gpu.get('game_draws')!=1 or
        gpu.get('f2')!=1 or gpu.get('wrong_bank_masks_rejected')!=maps or len(gpu.get('cases',[]))!=maps*6+1):
        raise ValueError('缺少同清冊完整草地GPU收據')
    expected={(e['map'],e['frame']) for group in proof['frames'] for e in group['targets']} if edges else {(e['map'],e['frame']) for e in proof['targets']}
    if edges and (len(proof['frames'])!=groups or len(expected)!=slots or len({m for m,f in expected})!=maps):
        raise ValueError('候選實際來源組數或槽位不完整')
    rows=gpu['cases'];modes={'steady','up1','up6','lut','black','white'}
    if edges:
        if gpu.get('source_groups')!=groups or gpu.get('target_slots')!=slots or gpu.get('recipe_sha256')!=proof['recipe_sha256']:
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
    expected_guards='PASS-SHORE-GUARDS' if shore else 'PASS-GROUND-BASES-GUARDS' if bases else 'PASS-GROUND-EDGES-GUARDS' if edges else 'PASS-GROUND-GUARDS'
    if guards.get('status')!=expected_guards or guards.get('recipe_sha256')!=proof['recipe_sha256'] or len(guards.get('checks',[]))!=10:
        raise ValueError('缺少同配方完整草地反例檢查')
    if (bases or shore) and guards.get('invalid_source_ref_atomic_rejection') is not True:
        raise ValueError('缺少底材清冊來源引用原子拒收')
    files=[*proof['masters'],*proof['frames']] if shore else proof['frames'] if bases else [proof['earth_master'],*proof['frames']] if edges else [proof['master'],proof['tile']]
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
