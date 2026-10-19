#!/usr/bin/env python3
"""依原始4／11幀描述子編製蓋亞HD候選；不改正式清冊或私人保存庫。"""

import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil

from PIL import Image, ImageDraw
from validate_modern_theme_catalog import ROOT, validate


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def names(catalog):
    found = set()
    for asset in catalog['assets']:
        for key in ['file', 'master_file']:
            if key in asset:
                found.add(asset[key])
        found.update(asset.get('files', []))
        found.update(asset.get('source_files', []))
    return found


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--inventory', required=True, type=Path)
    p.add_argument('--sources', required=True, type=Path, help='idle/charge/impact/flare/follow 五張母稿的JSON路徑表')
    p.add_argument('--out', required=True, type=Path)
    args = p.parse_args()
    out = args.out.resolve()
    if out.exists() or not out.is_relative_to(ROOT):
        raise SystemExit('使用儲存庫內新的候選輸出目錄')
    if out.parent.stat().st_uid != os.getuid():
        raise SystemExit('輸出目錄擁有權不符')
    inventory = json.loads(args.inventory.read_text())
    if inventory.get('status') != 'PASS-SOURCE-INVENTORY' or not inventory['archive_reconstruction']['matches_pinned_original']:
        raise SystemExit('原始圖源未通過固定容器核對')
    sources = json.loads(args.sources.read_text())
    if set(sources) != {'idle', 'charge', 'impact', 'flare', 'follow'}:
        raise SystemExit('需要完整五張母稿')
    art, fingerprints = {}, {}
    for key, name in sources.items():
        source = Path(name)
        with Image.open(source) as im:
            if im.mode != 'RGBA' or im.getchannel('A').getextrema() != (0,255):
                raise SystemExit(f'母稿缺少透明背景或不透明核心：{key}')
            art[key] = im.crop(im.getchannel('A').getbbox()).copy()
        fingerprints[key] = {'file': str(source.relative_to(ROOT)), 'sha256': digest(source)}
    catalog_path = ROOT/'remake/assets/themes/modern/catalog.json'
    base = validate(False, catalog_path)
    if any(x['role']=='battle_animation_hd_set' for x in base['assets']):
        raise SystemExit('已有動畫登錄；本工具只處理首批候選')
    out.mkdir()
    pack = out/'pack'
    pack.mkdir()
    private = ROOT/base['private_root']
    for name in sorted(names(base)):
        if Path(name).name != name or not (private/name).is_file():
            raise SystemExit(f'既有清冊來源缺件：{name}')
        (pack/name).symlink_to(os.path.relpath(private/name, pack))
    source_names = {}
    for key, name in sources.items():
        target = 'gaia-battle-'+key+'-master-v1.png'
        shutil.copyfile(name, pack/target)
        source_names[key] = target
    candidate = copy.deepcopy(base)
    candidate['private_root'] = str(pack.relative_to(ROOT))
    assignments = {90:['idle']*4, 91:['idle']+['charge']*5+['impact']+['flare']*2+['follow']*2}
    records, previews = [], []
    for resource in [90,91]:
        row = next(x for x in inventory['resources'] if x['resource']==resource)
        if len(row['frames'])!=len(assignments[resource]):
            raise SystemExit('蓋亞原始影格範圍改變')
        files, hashes, descriptors = [], [], []
        for frame, key in zip(row['frames'], assignments[resource]):
            source = art[key]
            w,h = frame['width']*6,frame['height']*6
            scale = min(w/source.width,h/source.height)
            if resource==90:
                scale *= frame['height']/row['frames'][0]['height']
            fitted = source.resize((max(1,round(source.width*scale)),max(1,round(source.height*scale))),Image.Resampling.LANCZOS)
            origin = ((w-fitted.width)//2,h-fitted.height)
            canvas = Image.new('RGBA',(w,h))
            canvas.paste(fitted,origin)
            name = f'figani-{resource:03d}-hd6-v1-f{frame["frame"]:02d}.png'
            canvas.save(pack/name)
            files.append(name)
            hashes.append(digest(pack/name))
            descriptor={k:frame[k] for k in ['x','y','width','height','delay_native','raw_byte_4','raw_byte_5','raw_byte_7']}
            descriptors.append(descriptor)
            records.append({'resource':resource,'frame':frame['frame'],'pose':key,'file':name,'sha256':hashes[-1],
                            'fit_size':list(fitted.size),'origin':list(origin),'native_descriptor':descriptor})
            stage=Image.new('RGBA',(1920,1200))
            stage.alpha_composite(canvas,(frame['x']*6,frame['y']*6))
            previews.append((resource,frame['frame'],key,stage))
        used=sorted(set(assignments[resource]))
        candidate['assets'].append({'asset_id':f'modern.figani.{resource:03d}.hd6.v1','role':'battle_animation_hd_set',
            'status':'runtime_candidate','files':files,'width':1920,'height':1200,'frame_count':len(files),
            'frame_sha256':hashes,'consumer_contract':'native_figani_rgba_hd6_v1','alpha_contract':'coverage',
            'source_refs':[f'asset:FIGANI_{resource:03d}','docs/data/fd2-reference-files.json'],
            'source_archive':'FIGANI.DAT','source_resource':resource,'source_metadata_sha256':row['metadata_sha256'],
            'native_header':[row['native_header'][k] for k in ['byte_1','byte_2','byte_4']],
            'native_frames':descriptors,'cycle_policy':'native_descriptor_key_poses',
            'layout_policy':'contain_native_descriptor_hd6_candidate','source_files':[source_names[k] for k in used],
            'source_sha256':[digest(pack/source_names[k]) for k in used]})
    path=out/'catalog.json'
    path.write_text(json.dumps(candidate,ensure_ascii=False,indent=2)+'\n')
    validate(True,path)
    sheet=Image.new('RGB',(1600,1000),(40,45,53));draw=ImageDraw.Draw(sheet)
    for i,(resource,frame,key,picture) in enumerate(previews):
        picture=picture.resize((384,240),Image.Resampling.LANCZOS)
        x,y=(i%4)*400,(i//4)*250
        sheet.paste(picture,(x,y),picture)
        draw.text((x+4,y+4),f'{resource:03d}/{frame:02d} {key}',fill='white')
    sheet.save(out/'review.png')
    report={'schema_version':1,'status':'PASS-COMPILED-ANIMATION-CANDIDATE','runtime_registered':False,
            'scope':'明示五母稿重定向的4／11幀；不是15張獨立重畫或原版逐幀相同；尚未通過GUI',
            'base_catalog_sha256':digest(catalog_path),'catalog_sha256':digest(path),'inventory_sha256':digest(args.inventory),
            'sources':fingerprints,'frames':records,'review_sha256':digest(out/'review.png'),
            'art_review_pending':['待機與攻擊之間的人物比例與位置','大小閃光的轉場與淡出','正常玩家呈現及效能'],
            'compiler_sha256':digest(Path(__file__))}
    (out/'compiled.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':report['status'],'resources':2,'frames':len(records),'catalog_assets':len(candidate['assets'])}))


if __name__=='__main__':
    main()
