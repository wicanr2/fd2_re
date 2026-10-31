#!/usr/bin/env python3
"""依定版原稿更新HD1人物候選；保持完整圖庫、來源與相位契約。"""
import argparse
from copy import deepcopy
import hashlib
import json
import os
from pathlib import Path
import shutil
import tempfile

from PIL import Image
from build_modern_hd_map_sprites import master_components,compile_group
from validate_modern_theme_catalog import validate,VISIBLE_FDICON_GROUPS

ROOT=Path(__file__).resolve().parents[1]
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def read(p):return json.loads(p.read_text())
def write(p,d):p.write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
def safe(name):
    if not isinstance(name,str) or Path(name).name!=name or not name.endswith('.png'):raise ValueError('unsafe PNG name')
    return name

def asset_digest(assets):
    return hashlib.sha256(json.dumps(assets,ensure_ascii=False,sort_keys=True,separators=(',',':')).encode()).hexdigest()

def prepare(recipe_path,catalog_path,pack,original,masters,out):
    r,c=read(recipe_path),read(catalog_path)
    if r['kind']!='fd2_hd1_chibi_map_refresh' or r['base_catalog_sha256']!=sha(catalog_path):raise ValueError('base catalog changed')
    if r['approved_preview_sha256']!=sha(ROOT/r['approved_preview']):raise ValueError('approved design changed')
    bank=original/'sprites/fdicon';source=read(bank/'bank.json')['source']
    if source['sha256']!=r['native_bank_sha256'] or sha(original/'palette/fdother_000.json')!=r['palette_sha256']:raise ValueError('native source changed')
    if not Path('/.dockerenv').exists() or out.exists() or not out.parent.is_dir() or out.parent.stat().st_uid!=os.getuid():raise ValueError('fresh owned candidate required')
    parents={a['source_group']:a for a in c['assets'] if a['role']=='map_sprite_set'}
    children={a['source_group']:a for a in c['assets'] if a['role']=='map_sprite_hd_set'}
    if set(parents)!=VISIBLE_FDICON_GROUPS or set(children)!=VISIBLE_FDICON_GROUPS:raise ValueError('complete native collection required')
    groups=[a['group'] for a in r['groups']]
    if len(groups)!=len(set(groups)) or not groups or not set(groups)<=set(parents):raise ValueError('invalid or duplicate group')
    sources=[];reuses={}
    for entry in r['groups']:
        group=entry['group'];master=masters/safe(entry['master_file']);parent=parents[group]
        if sha(master)!=entry['master_sha256']:raise ValueError('approved master changed')
        if parent['frame_count']!=children[group]['frame_count']:raise ValueError('parent differs')
        native=[]
        for i in range(parent['frame_count']):
            folder=bank/f'sprite_{group*12+i:04d}'
            row={name:sha(folder/name) for name in ('frame.png','mask.png','remap_mask.png')}
            if not Image.open(folder/'mask.png').getbbox():raise ValueError('no source-write frame')
            native.append(row)
        with Image.open(master) as im:
            if im.mode!='RGBA' or im.getchannel('A').getextrema()!=(0,255):raise ValueError('master requires transparent alpha and opaque core')
            pieces=master_components(im,parent['frame_count'])
        if 'reuse_group' in entry:
            reference=entry['reuse_group']
            if type(reference)!=int or reference not in parents or reference in groups:raise ValueError('invalid reused group')
            reference_parent,reference_child=parents[reference],children[reference]
            if reference_parent['frame_count']!=parent['frame_count'] or reference_child['frame_count']!=parent['frame_count']:raise ValueError('reused frame count differs')
            if asset_digest([reference_parent,reference_child])!=entry.get('reuse_assets_sha256'):raise ValueError('reused asset source changed')
            if reference_parent.get('master_file')!=entry['master_file'] or reference_child.get('master_file')!=entry['master_file']:raise ValueError('reused master binding changed')
            reference_native=[{name:sha(bank/f'sprite_{reference*12+i:04d}'/name) for name in ('frame.png','mask.png','remap_mask.png')} for i in range(parent['frame_count'])]
            if native!=reference_native:raise ValueError('reused native layers differ')
            for asset in (reference_parent,reference_child):
                for name,digest in zip(asset['files'],asset['frame_sha256']):
                    if sha(pack/safe(name))!=digest:raise ValueError('reused registered frame changed')
            reuses[group]=(reference_parent,reference_child,reference_native)
        sources.append((entry,parent,children[group],master,pieces,native))
    # Validate the complete old pack before publishing even a disposable candidate.
    if read(pack/'catalog.json')!=c:raise ValueError('pack/catalog differ')
    scratch=Path(tempfile.mkdtemp(prefix='.chibi-',dir=out.parent))
    try:
        for p in pack.iterdir():
            if p.is_file() and p.name not in ('catalog.json','private-manifest.json'):(scratch/p.name).symlink_to(p.resolve())
        result=deepcopy(c);records=[]
        for entry,old_parent,old_child,master,pieces,native in sources:
            name=safe(entry['master_file'])
            if entry['group'] in reuses:
                reference_parent,reference_child,reference_native=reuses[entry['group']]
                if sha(scratch/name)!=entry['master_sha256']:raise ValueError('reused master changed')
                parent,child=deepcopy(old_parent),deepcopy(old_child)
                for key in ('master_file','files','frame_sha256'):parent[key]=deepcopy(reference_parent[key])
                for key in ('master_file','files','frame_sha256','frame_overrides','source_files','source_sha256'):child[key]=deepcopy(reference_child[key])
                hd_dir=scratch/'hd';hd_dir.mkdir()
                compiled=compile_group(parent,scratch,hd_dir,name,child['frame_overrides'],pieces)
                if [f['sha256'] for f in compiled['frames']]!=child['frame_sha256']:raise ValueError('reused HD rebuild differs')
                for i,frame in enumerate(compiled['frames']):frame['file']=child['files'][i]
                shutil.rmtree(hd_dir)
                for asset in (parent,child):result['assets'][next(i for i,v in enumerate(result['assets']) if v['asset_id']==asset['asset_id'])]=asset
                records.append(dict(group=entry['group'],master_file=name,master_sha256=sha(master),native_inputs=native,parent=parent,hd=child,compiled=compiled,reuse_group=entry['reuse_group'],reuse_native_inputs=reference_native,reuse_asset_ids=[reference_parent['asset_id'],reference_child['asset_id']]))
                continue
            if (scratch/name).exists():raise ValueError('new master name already exists')
            shutil.copyfile(master,scratch/name)
            parent=deepcopy(old_parent);parent['master_file']=name;files=[];hashes=[]
            prefix=entry['output_prefix'];safe(prefix+'.png')
            canonical=[old_parent['frame_sha256'].index(h) for h in old_parent['frame_sha256']]
            for i,(sprite,_) in enumerate(pieces):
                target=scratch/f'{prefix}-f{i:02d}.png'
                if target.exists():raise ValueError('new output already exists')
                if canonical[i]<i:target.write_bytes((scratch/files[canonical[i]]).read_bytes())
                else:
                    scale=min(22/sprite.width,22/sprite.height)
                    size=(max(1,round(sprite.width*scale)),max(1,round(sprite.height*scale)))
                    small=sprite.resize(size,Image.Resampling.NEAREST)
                    small.putalpha(small.getchannel('A').point(lambda v:255 if v>=128 else 0))
                    image=Image.new('RGBA',(24,24));image.alpha_composite(small,((24-size[0])//2,23-size[1]));image.save(target)
                files.append(target.name);hashes.append(sha(target))
            for i in range(len(files)):
                for j in range(i):
                    if (hashes[i]==hashes[j])!=(canonical[i]==canonical[j]):raise ValueError('24px phase collapse')
            parent['files'],parent['frame_sha256']=files,hashes
            # The legacy compiler writes fixed names; isolate those scratch files
            # from old pack aliases, then give every new frame a versioned name.
            hd_dir=scratch/'hd';hd_dir.mkdir()
            compiled=compile_group(parent,scratch,hd_dir,name,{},pieces)
            child=deepcopy(old_child);child['master_file']=name;child['frame_overrides']={};child['source_files']=[name];child['source_sha256']=[sha(master)]
            new_files=[]
            for i,frame in enumerate(compiled['frames']):
                target=scratch/f'{prefix}-hd6-f{i:02d}.png'
                if target.exists():raise ValueError('new HD output already exists')
                (hd_dir/frame['file']).rename(target);frame['file']=target.name;new_files.append(target.name)
            hd_dir.rmdir();child['files']=new_files;child['frame_sha256']=[f['sha256'] for f in compiled['frames']]
            for a in (parent,child):result['assets'][next(i for i,v in enumerate(result['assets']) if v['asset_id']==a['asset_id'])]=a
            records.append(dict(group=entry['group'],master_file=name,master_sha256=sha(master),native_inputs=native,parent=parent,hd=child,compiled=compiled))
        result['private_root']=str(out.relative_to(ROOT));write(scratch/'catalog.json',result)
        # validate() resolves private_root, so atomically move only complete output.
        scratch.rename(out)
        try:validate(True,out/'catalog.json')
        except BaseException:
            shutil.rmtree(out);raise
        manifest=dict(kind='fd2_hd1_chibi_refresh_candidate',schema_version=1,base_catalog_sha256=sha(catalog_path),catalog_sha256=sha(out/'catalog.json'),recipe_sha256=sha(recipe_path),compiler_sha256=sha(Path(__file__)),groups=records,other_assets_unchanged=sum(a==b for a,b in zip(c['assets'],result['assets'])),scope='候選編製；不是GPU、正常操作或原版動態驗收。')
        write(out/'manifest.json',manifest)
        if len(result['assets'])!=len(c['assets']) or manifest['other_assets_unchanged']!=len(c['assets'])-2*len(groups):raise ValueError('unrelated assets changed')
        return manifest
    except BaseException:
        if scratch.exists():shutil.rmtree(scratch)
        raise

def runtime_view(candidate,out):
    if out.exists() or not out.parent.is_dir():raise ValueError('fresh runtime view required')
    out.mkdir();assets=out/'assets';assets.mkdir()
    for p in (ROOT/'remake/assets').iterdir():
        if p.name!='themes':(assets/p.name).symlink_to(p.resolve())
    themes=assets/'themes';themes.mkdir()
    for p in (ROOT/'remake/assets/themes').iterdir():
        if p.name!='modern':(themes/p.name).symlink_to(p.resolve())
    modern=themes/'modern';modern.mkdir()
    for p in (ROOT/'remake/assets/themes/modern').iterdir():
        (modern/p.name).symlink_to((candidate/'catalog.json').resolve() if p.name=='catalog.json' else p.resolve())
    return dict(status='CANDIDATE-RUNTIME-VIEW',catalog_sha256=sha(modern/'catalog.json'),scope='相同正式程式與資料，僅改候選清冊；原始資產與規則未改。')

def main():
    p=argparse.ArgumentParser(description=__doc__);sub=p.add_subparsers(dest='mode',required=True)
    q=sub.add_parser('prepare')
    for name in ('recipe','catalog','pack','original','masters','out'):q.add_argument('--'+name,type=Path,required=True)
    q=sub.add_parser('runtime-view');q.add_argument('--candidate',type=Path,required=True);q.add_argument('--out',type=Path,required=True)
    a=p.parse_args();result=runtime_view(a.candidate,a.out) if a.mode=='runtime-view' else prepare(a.recipe,a.catalog,a.pack,a.original,a.masters,a.out)
    print(json.dumps(dict(status=result.get('status','CANDIDATE-PREPARED'),groups=len(result.get('groups',[])),catalog_sha256=result['catalog_sha256']),ensure_ascii=False))
if __name__=='__main__':main()
