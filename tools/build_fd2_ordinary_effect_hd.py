#!/usr/bin/env python3
"""依固定原版遮罩及手繪材質編製十六組普通效果候選，不登錄正式清冊。"""
import argparse
import copy
import json
import os
from pathlib import Path
import shutil

from PIL import Image, ImageMath
from build_fd2_gaia_battle_hd import digest, names
from validate_modern_theme_catalog import ROOT, validate

RESOURCES = [18,19,20,21,22,23,24,25,28,30,32,33,37,38,39,43]
FIELDS = ['x','y','width','height','delay_native','raw_byte_4','raw_byte_5','raw_byte_7']


def colour_frame(original, material):
    """F通道float32；alpha保持，僅RGB按READY公式重定向。"""
    if original.mode != 'RGBA' or material.mode != 'RGB' or original.size != material.size:
        raise ValueError('材質與原圖形狀不同')
    tr,tg,tb = (channel.convert('F') for channel in material.split())
    luminance = ImageMath.lambda_eval(lambda a:a['r']*.2126+a['g']*.7152+a['b']*.0722,r=tr,g=tg,b=tb)
    channels = []
    for old,texture in zip(original.split()[:3],[tr,tg,tb]):
        channels.append(ImageMath.lambda_eval(
            lambda a:a['convert'](a['C']*(a['L']*.28/255+.72)+(a['T']-a['L'])*.15+.5,'L'),
            C=old.convert('F'),T=texture,L=luminance))
    return Image.merge('RGBA',(*channels,original.getchannel('A')))


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for key in ['original-pack','reference','recipe','out']:
        parser.add_argument('--'+key,type=Path,required=True)
    parser.add_argument('--base-catalog',type=Path,default=ROOT/'remake/assets/themes/modern/catalog.json')
    args=parser.parse_args()
    out=args.out.resolve()
    if (not Path('/.dockerenv').exists() or out.exists() or not out.is_relative_to(ROOT)
            or out.parent.stat().st_uid!=os.getuid()):
        raise SystemExit('在Docker使用本人擁有的新候選目錄')
    recipe=json.loads(args.recipe.read_text())
    if (set(recipe)!={'schema_version','reference_sha256','resources','materials'} or recipe['schema_version']!=1
            or recipe['resources']!=RESOURCES or set(recipe['materials'])!={'fire','electric'}):
        raise SystemExit('未知普通效果配方或資源範圍')
    reference=json.loads(args.reference.read_text())
    if (digest(args.reference)!=recipe['reference_sha256'] or reference.get('status')!='PASS-PINNED-NATIVE-ASSET-REFERENCE'
            or digest(args.original_pack/'manifest.json')!=reference['pack_manifest_sha256']
            or reference['tool']['sha256']!=digest(Path(__file__).with_name('audit_fd2_ordinary_effect_source.py'))):
        raise SystemExit('固定效果來源盤點不同')
    pinned=next(a for a in json.loads((ROOT/'docs/data/fd2-reference-files.json').read_text())['files'] if a['file']=='FDOTHER.DAT')
    if reference['source_file']!=pinned:
        raise SystemExit('FDOTHER來源版本不同')
    base=validate(False,args.base_catalog)
    if any(a['role']=='battle_effect_hd_set' and a['source_resource'] in RESOURCES for a in base['assets']):
        raise SystemExit('普通效果已登錄，不覆寫')
    materials,source_records={},{}
    for key,record in recipe['materials'].items():
        if set(record)!={'file','sha256'}:
            raise SystemExit('未知材質欄位')
        path=(ROOT/record['file']).resolve()
        if not path.is_relative_to(ROOT) or path.stat().st_uid!=os.getuid() or digest(path)!=record['sha256']:
            raise SystemExit('材質路徑、SHA或擁有權不同')
        with Image.open(path) as im:
            if im.mode not in ['RGB','RGBA'] or not 512<=im.width<=4096 or not 512<=im.height<=4096:
                raise SystemExit('材質格式或形狀不同')
            if im.mode=='RGBA' and im.getchannel('A').getextrema()!=(255,255):
                raise SystemExit('材質須完整不透明')
            materials[key]=im.convert('RGB').resize((1920,1200),Image.Resampling.BICUBIC)
        source_records[key]={'file':str(path.relative_to(ROOT)),'sha256':digest(path)}
    rows={row['resource']:row for row in reference['resources']}
    natives={}
    inputs=[]
    # 所有原資料先驗完整，繪圖前不建立輸出目錄。
    for resource in RESOURCES:
        row=rows[resource]
        folder=args.original_pack/'animations'/f'FDOTHER_{resource:03d}'
        metadata=folder/'animation.json'
        if digest(metadata)!=row['metadata_sha256']:
            raise SystemExit('原版效果metadata不同')
        doc=json.loads(metadata.read_text())
        if (doc.get('source')!={'file':'FDOTHER.DAT','resource':resource} or doc['native_header']!=row['native_header']
                or len(doc['frames'])!=len(row['frames'])):
            raise SystemExit('原版效果header或幀數不同')
        for index,(frame,proof) in enumerate(zip(doc['frames'],row['frames'])):
            if any(frame[k]!=proof[k] for k in FIELDS):
                raise SystemExit('原版效果描述子不同')
            if (frame['width']<=0 or frame['height']<=0 or frame['x']<0 or frame['y']<0
                    or frame['x']+frame['width']>320 or frame['y']+frame['height']>200):
                raise SystemExit('原版效果超出固定材質平面')
            if any(Path(frame[k]).name!=frame[k] for k in ['path','mask_path']):
                raise SystemExit('原版效果圖源路徑越界')
            path,mask_path=folder/frame['path'],folder/frame['mask_path']
            if digest(path)!=proof['pixels_sha256'] or digest(mask_path)!=proof['mask_sha256']:
                raise SystemExit('原版效果PNG或mask SHA不同')
            with Image.open(path) as im,Image.open(mask_path) as mask:
                if im.mode!='P' or im.size!=mask.size or im.size!=(frame['width'],frame['height']):
                    raise SystemExit('原版效果PNG或mask形狀不同')
                alpha=mask.convert('L')
                if alpha.getextrema()[1]==0:
                    raise SystemExit('本批未允許空白效果來源')
                original=im.convert('RGBA');original.putalpha(alpha)
            inputs.append((resource,index,frame,original,alpha.getextrema()[0]!=0))
        natives[resource]=row
    if len(inputs)!=304:
        raise SystemExit('本批原版效果影格數不同')
    pending=[]
    for resource,index,frame,original,full in inputs:
        original=original.resize((frame['width']*6,frame['height']*6),Image.Resampling.LANCZOS)
        key='fire' if resource in [18,20] else 'electric'
        region=(frame['x']*6,frame['y']*6,(frame['x']+frame['width'])*6,(frame['y']+frame['height'])*6)
        picture=colour_frame(original,materials[key].crop(region))
        low,high=picture.getchannel('A').getextrema()
        if high!=255 or low!=0 and not full:
            raise SystemExit('普通效果成品缺少透明背景或255核心')
        pending.append((resource,index,frame,picture,key,full))
    private=ROOT/base['private_root']
    old_names=names(base)
    if any(Path(n).name!=n or not (private/n).is_file() for n in old_names):
        raise SystemExit('基底清冊缺件')
    out.mkdir();pack=out/'pack';pack.mkdir()
    shutil.copyfile(args.base_catalog,out/'base-catalog.json')
    for name in sorted(old_names):(pack/name).symlink_to(os.path.relpath(private/name,pack))
    source_names={}
    for key,record in source_records.items():
        name=f'ordinary-{key}-hd6-material-v1.png'
        if name in old_names:raise SystemExit('材質檔名與既有資產重複')
        shutil.copyfile(ROOT/record['file'],pack/name);source_names[key]=name
    records=[]
    for resource,index,frame,picture,key,full in pending:
        name=f'fdother-{resource:03d}-hd6-v1-f{index:02d}.png'
        picture.save(pack/name)
        records.append({'resource':resource,'frame':index,'file':name,'sha256':digest(pack/name),
                        'native_descriptor':{k:frame[k] for k in FIELDS},'material':key,'full_source_mask':full})
    candidate=copy.deepcopy(base);candidate['private_root']=str(pack.relative_to(ROOT))
    for resource in RESOURCES:
        row=natives[resource];selected=[v for v in records if v['resource']==resource]
        key=selected[0]['material']
        candidate['assets'].append({'asset_id':f'modern.fdother.{resource:03d}.hd6.v1','role':'battle_effect_hd_set',
            'status':'runtime_candidate','files':[v['file'] for v in selected],'width':1920,'height':1200,
            'frame_count':len(selected),'frame_sha256':[v['sha256'] for v in selected],
            'consumer_contract':'native_fdother_ordinary_rgba_hd6_v1','alpha_contract':'coverage',
            'source_refs':[f'asset:FDOTHER_{resource:03d}','docs/data/fd2-reference-files.json'],
            'source_archive':'FDOTHER.DAT','source_resource':resource,'source_metadata_sha256':row['metadata_sha256'],
            'native_header':[row['native_header'][k] for k in ['byte_1','byte_2','byte_4']],
            'native_frames':[v['native_descriptor'] for v in selected],
            'cycle_policy':'native_ordinary_frames','layout_policy':'native_descriptor_hd6',
            'source_files':[source_names[key]],'source_sha256':[source_records[key]['sha256']]})
    path=out/'catalog.json';path.write_text(json.dumps(candidate,ensure_ascii=False,indent=2)+'\n')
    validate(True,path)
    report={'status':'PASS-COMPILED-ORDINARY-EFFECT-CANDIDATE','runtime_registered':False,
        'base_catalog_sha256':digest(args.base_catalog),'catalog_sha256':digest(path),
        'recipe_sha256':digest(args.recipe),'reference_sha256':digest(args.reference),
        'compiler_sha256':digest(Path(__file__)),'sources':source_records,'frames':records,
        'scope':'原版遮罩與明暗重定向手繪材質，非304張獨立新畫、原版RGB parity或玩家驗證。'}
    (out/'compiled.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':report['status'],'resources':len(RESOURCES),'frames':len(records),'assets':len(candidate['assets'])}))


if __name__=='__main__':
    main()
