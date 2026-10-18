#!/usr/bin/env python3
"""以實際屋後PNG做驗證器拒收測試；不改原始收據、圖源或遊戲。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
from PIL import Image


def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ['observation','route','catalog','pack','report']:
        p.add_argument('--'+name,type=Path,required=True)
    args=p.parse_args()
    if args.report.exists():raise ValueError('不得覆寫收據')
    original=json.loads(args.observation.read_text())
    catalog=json.loads(args.catalog.read_text())
    root=args.observation.parent
    name='terrain-hd-occlusion.png'
    state=original['screenshots'][name]['gpu_before']
    fg=state['hd_world_terrain_foreground_draws'][0]
    terrain=next(a for a in catalog['assets'] if a['role']=='native_terrain_hd_set' and fg['frame'] in a['frame_ids'])
    mask=Image.open(args.pack/terrain['files'][terrain['frame_ids'].index(fg['frame'])]).convert('RGBA')
    unit=next(u for u in state['hd_world_units'] if u['group']==0)
    person_set=next(a for a in catalog['assets'] if a['role']=='map_sprite_hd_set' and a['source_group']==0)
    person=Image.open(args.pack/person_set['files'][unit['frame']]).convert('RGBA')
    x,y,_,_=fg['rect']
    ux,uy,ux1,uy1=unit['rect']
    opaque=next((x+sx,y+sy) for sy in range(144) for sx in range(144) if mask.getpixel((sx,sy))[3]==255)
    visible=next((px,py) for py in range(max(y,uy),min(y+144,uy1))
                 for px in range(max(x,ux),min(x+144,ux1))
                 if mask.getpixel((px-x,py-y))[3]==0 and person.getpixel((px-ux,py-uy))[3]==255)
    outcomes=[]
    with tempfile.TemporaryDirectory(prefix='.occlusion-negative-',dir=args.report.parent) as tmp:
        tmp=Path(tmp)
        for case in ['baseline','hash-mismatch','opaque-write-wrong','transparent-unit-wrong','missing-foreground']:
            folder=tmp/case;folder.mkdir()
            data=copy.deepcopy(original)
            for filename in ['terrain-hd-occlusion.png','terrain-hd-occlusion-restored.png']:
                shutil.copyfile(root/filename,folder/filename)
            if case in ['hash-mismatch','opaque-write-wrong','transparent-unit-wrong']:
                image=Image.open(folder/name).convert('RGBA')
                position=visible if case=='transparent-unit-wrong' else opaque
                v=image.getpixel(position)
                image.putpixel(position,(255-v[0],255-v[1],255-v[2],255));image.save(folder/name)
                if case!='hash-mismatch':data['screenshots'][name]['sha256']=sha(folder/name)
            if case=='missing-foreground':
                for key in ['gpu_before','state']:data['screenshots'][name][key]['hd_world_terrain_foreground_draws']=[]
            observation=folder/'terrain-observation.json'
            observation.write_text(json.dumps(data,ensure_ascii=False))
            result=subprocess.run([sys.executable,str(Path(__file__).with_name('verify_fd2_hd_terrain_occlusion.py')),
                                   '--observation',str(observation),'--route',str(args.route),
                                   '--catalog',str(args.catalog),'--pack',str(args.pack),'--report',str(folder/'result.json')],
                                  text=True,capture_output=True,timeout=60)
            expected=case=='baseline'
            if (result.returncode==0)!=expected:
                raise AssertionError((case,result.stdout,result.stderr))
            outcomes.append({'case':case,'status':'PASS','accepted':result.returncode==0,
                             'rejection':result.stderr.strip().splitlines()[-1:] if not expected else []})
    args.report.write_text(json.dumps({'status':'PASS','cases':outcomes,'temporary_fixtures_removed':True,
                                      'source_observation_sha256':sha(args.observation),'tool_sha256':sha(Path(__file__))},
                                     ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(outcomes,ensure_ascii=False))


if __name__=='__main__':main()
