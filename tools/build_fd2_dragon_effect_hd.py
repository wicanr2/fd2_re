#!/usr/bin/env python3
"""依固定FDOTHER26/27及明示母稿矩陣編製36幀候選，不登錄正式清冊。"""
import argparse
import copy
import json
import os
from pathlib import Path
import shutil

from PIL import Image, ImageOps
from build_fd2_character_battle_hd import rectangle
from build_fd2_gaia_battle_hd import digest, names
from validate_modern_theme_catalog import ROOT, validate

FIELDS=['x','y','width','height','delay_native','raw_byte_4','raw_byte_5','raw_byte_7']


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for key in ['original-pack','reference','recipe','out']:
        parser.add_argument('--'+key,type=Path,required=True)
    parser.add_argument('--base-catalog',type=Path,default=ROOT/'remake/assets/themes/modern/catalog.json')
    args=parser.parse_args();out=args.out.resolve()
    if (not Path('/.dockerenv').exists() or out.exists() or not out.is_relative_to(ROOT)
            or out.parent.stat().st_uid!=os.getuid()):
        raise SystemExit('在Docker使用本人擁有的新候選目錄')
    recipe=json.loads(args.recipe.read_text());reference=json.loads(args.reference.read_text())
    if (set(recipe)!={'schema_version','reference_sha256','sources','resources'} or recipe['schema_version']!=1
            or set(recipe['sources'])!={'closed','open','wave'} or set(recipe['resources'])!={'26','27'}):
        raise SystemExit('未知火龍配方或圖庫範圍')
    if (recipe['reference_sha256']!=digest(args.reference) or reference.get('status')!='PASS-PINNED-NATIVE-ASSET-REFERENCE'
            or digest(args.original_pack/'manifest.json')!=reference['pack_manifest_sha256']
            or reference['tool']['sha256']!=digest(Path(__file__).with_name('audit_fd2_ordinary_effect_source.py'))):
        raise SystemExit('固定普通效果來源盤點不同')
    pinned=next(v for v in json.loads((ROOT/'docs/data/fd2-reference-files.json').read_text())['files'] if v['file']=='FDOTHER.DAT')
    if reference['source_file']!=pinned:raise SystemExit('FDOTHER來源版本不同')
    manifest=json.loads((args.original_pack/'manifest.json').read_text())
    native_files={v['path']:v for v in manifest['assets']}
    def checked(path,resource,index=None):
        row=native_files.get(path.relative_to(args.original_pack).as_posix())
        if (row is None or row.get('source_file')!='FDOTHER.DAT' or row.get('source_resource')!=resource
                or digest(path)!=row['sha256'] or index is not None and row.get('source_frame')!=index):
            raise SystemExit('火龍原圖源SHA或定位不同')
    base=validate(False,args.base_catalog)
    if any(v['role']=='battle_effect_hd_set' and v['source_resource'] in [26,27] for v in base['assets']):
        raise SystemExit('火龍已登錄，不覆寫')
    art,sources={},{}
    for key,value in recipe['sources'].items():
        if set(value)!={'file','sha256','anchor'}:raise SystemExit('未知母稿欄位')
        path=(ROOT/value['file']).resolve()
        if not path.is_relative_to(ROOT) or path.stat().st_uid!=os.getuid() or digest(path)!=value['sha256']:
            raise SystemExit('母稿路徑、SHA或擁有權不同')
        with Image.open(path) as im:
            if im.mode!='RGBA' or im.getchannel('A').getextrema()[0]!=0 or im.getchannel('A').getextrema()[1]<250:
                raise SystemExit('母稿缺少透明背景或高覆蓋核心')
            art[key]=im.copy()
        anchor=rectangle(value['anchor'],art[key].size)
        if tuple(anchor)!=art[key].getchannel('A').point(lambda a:255 if a>=192 else 0).getbbox():
            raise SystemExit('母稿高覆蓋錨點不同')
        sources[key]={'file':str(path.relative_to(ROOT)),'sha256':digest(path),'anchor':anchor}
    natives,pending,cache={},{},{}
    for resource in [26,27]:
        row=next(v for v in reference['resources'] if v['resource']==resource)
        folder=args.original_pack/'animations'/f'FDOTHER_{resource:03d}';metadata=folder/'animation.json'
        checked(metadata,resource)
        if digest(metadata)!=row['metadata_sha256']:raise SystemExit('原版火龍metadata不同')
        doc=json.loads(metadata.read_text());assignments=recipe['resources'][str(resource)]
        if (doc.get('source')!={'file':'FDOTHER.DAT','resource':resource} or doc['native_header']!=row['native_header']
                or len(doc['frames'])!=18 or len(row['frames'])!=18 or not isinstance(assignments,list) or len(assignments)!=18):
            raise SystemExit('火龍header或完整十八幀配方不同')
        natives[resource]=row
        for index,(frame,proof,assignment) in enumerate(zip(doc['frames'],row['frames'],assignments)):
            if any(frame[k]!=proof[k] for k in FIELDS):raise SystemExit('原版火龍描述子不同')
            if any(Path(frame[k]).name!=frame[k] for k in ['path','mask_path']):raise SystemExit('火龍圖源路徑越界')
            for key,sha in [('path','pixels_sha256'),('mask_path','mask_sha256')]:
                checked(folder/frame[key],resource,index)
                if digest(folder/frame[key])!=proof[sha]:raise SystemExit('原版火龍PNG或mask不同')
            with Image.open(folder/frame['path']) as im,Image.open(folder/frame['mask_path']) as mask:
                if (im.mode!='P' or im.size!=mask.size or im.size!=(frame['width'],frame['height'])
                        or mask.convert('L').getextrema()[1]==0):raise SystemExit('本批未允許空白或錯形狀火龍圖源')
            pose='wave' if index>=16 else 'open' if index in [9,15] else 'closed'
            if (not isinstance(assignment,dict) or set(assignment)!={'pose','size','origin','flip'}
                    or assignment['pose']!=pose or type(assignment['flip']) is not bool or assignment['flip']!=(resource==27)):
                raise SystemExit('火龍姿勢、矩陣欄位或方向不同')
            size,origin=assignment['size'],assignment['origin']
            if (not isinstance(size,list) or len(size)!=2 or any(type(v) is not int or not 1<=v<=4096 for v in size)
                    or not isinstance(origin,list) or len(origin)!=2 or any(type(v) is not int or abs(v)>4096 for v in origin)):
                raise SystemExit('火龍取樣矩陣超出有界整數範圍')
            key=(pose,*size,assignment['flip'])
            if key not in cache:
                fitted=art[pose].resize(tuple(size),Image.Resampling.LANCZOS)
                if assignment['flip']:fitted=ImageOps.mirror(fitted)
                cache[key]=fitted
            fitted=cache[key];canvas=Image.new('RGBA',(frame['width']*6,frame['height']*6))
            canvas.alpha_composite(fitted,tuple(origin));hist=canvas.getchannel('A').histogram()
            if canvas.getchannel('A').getextrema()!=(0,255) or sum(hist[250:])<1000:
                raise SystemExit('成品缺少透明背景、255核心或1000個高覆蓋像素')
            clipped=sum(fitted.getchannel('A').histogram()[192:])-sum(hist[192:])
            pending[(resource,index)]=({'resource':resource,'frame':index,**assignment,
                'native_descriptor':{k:frame[k] for k in FIELDS},'clipped_high_coverage_pixels':clipped,
                'high_coverage_pixels':sum(hist[250:])},canvas)
    private=ROOT/base['private_root'];old_names=names(base)
    source_names={key:f'fire-dragon-{key}-hd6-master-v1.png' for key in sources}
    if any(name in old_names for name in source_names.values()):raise SystemExit('母稿檔名與既有清冊重複')
    if any(Path(name).name!=name or not (private/name).is_file() for name in old_names):raise SystemExit('基底清冊缺件')
    out.mkdir();pack=out/'pack';pack.mkdir();shutil.copyfile(args.base_catalog,out/'base-catalog.json')
    for name in sorted(old_names):(pack/name).symlink_to(os.path.relpath(private/name,pack))
    for key,row in sources.items():shutil.copyfile(ROOT/row['file'],pack/source_names[key])
    records=[]
    for (resource,index),(record,picture) in pending.items():
        name=f'fdother-{resource:03d}-hd6-v1-f{index:02d}.png';picture.save(pack/name)
        records.append({**record,'file':name,'sha256':digest(pack/name)})
    candidate=copy.deepcopy(base);candidate['private_root']=str(pack.relative_to(ROOT))
    for resource in [26,27]:
        row=natives[resource];selected=[v for v in records if v['resource']==resource]
        candidate['assets'].append({'asset_id':f'modern.fdother.{resource:03d}.hd6.v1','role':'battle_effect_hd_set',
            'status':'runtime_candidate','files':[v['file'] for v in selected],'frame_sha256':[v['sha256'] for v in selected],
            'width':1920,'height':1200,'frame_count':18,'consumer_contract':'native_fdother_ordinary_rgba_hd6_v1','alpha_contract':'coverage',
            'source_refs':[f'asset:FDOTHER_{resource:03d}','docs/data/fd2-reference-files.json'],
            'source_archive':'FDOTHER.DAT','source_resource':resource,'source_metadata_sha256':row['metadata_sha256'],
            'native_header':[row['native_header'][k] for k in ['byte_1','byte_2','byte_4']],
            'native_frames':[v['native_descriptor'] for v in selected],'cycle_policy':'native_ordinary_frames',
            'layout_policy':'native_descriptor_hd6','source_files':[source_names[k] for k in ['closed','open','wave']],
            'source_sha256':[sources[k]['sha256'] for k in ['closed','open','wave']]})
    path=out/'catalog.json';path.write_text(json.dumps(candidate,ensure_ascii=False,indent=2)+'\n');validate(True,path)
    report={'status':'PASS-COMPILED-DRAGON-CANDIDATE','runtime_registered':False,'base_catalog_sha256':digest(args.base_catalog),
        'catalog_sha256':digest(path),'reference_sha256':digest(args.reference),'recipe_sha256':digest(args.recipe),
        'compiler_sha256':digest(Path(__file__)),'sources':sources,'frames':records,
        'scope':'三張母稿姿勢共用、27鏡射與原框裁切；美術幾何近似，非原版RGB或alpha parity，未驗正常操作或人工畫質。'}
    (out/'compiled.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':report['status'],'frames':len(records),'assets':len(candidate['assets'])}))


if __name__=='__main__':
    main()
