#!/usr/bin/env python3
"""由已登錄九格手繪母稿及固定LMI支撐編製77份候選，不改正式圖包。"""
import argparse
import hashlib
import json
import os
import shutil
from collections import deque
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter
from audit_fd2_player9_map_source import audit
from validate_modern_theme_catalog import ROOT, validate

RANGES = list(range(64,76))+list(range(126,191))

def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()

def slot(index):
    if 64<=index<=69:return 0
    if 70<=index<=73:return 1
    if 74<=index<=75:return 2
    if 126<=index<=137:return 3
    if 138<=index<=146:return 4
    if 147<=index<=157:return 5
    if 158<=index<=169:return 6
    if 170<=index<=182:return 7
    if 183<=index<=190:return 8
    raise ValueError('未審查索引')

def painted_material(master, native_mask, size, rotate):
    choices=[]
    for angle in (range(0,360,45) if rotate else [0]):
        art=master.rotate(angle,Image.Resampling.BICUBIC,expand=True)
        box=art.getchannel('A').point(lambda v:255 if v>=32 else 0).getbbox()
        art=art.crop(box).resize(size,Image.Resampling.LANCZOS)
        guide=native_mask.resize((32,32),Image.Resampling.LANCZOS)
        alpha=art.getchannel('A').resize((32,32),Image.Resampling.LANCZOS)
        score=sum(v*(g/255-.35*(1-g/255)) for v,g in zip(alpha.getdata(),guide.getdata()))
        choices.append((score,angle,art))
    _,angle,art=max(choices,key=lambda v:v[0])
    return art,angle

def extend_rgb(art):
    width,height=art.size
    rgba=list(art.getdata());rgb=[None]*len(rgba);queue=deque()
    for i,(r,g,b,a) in enumerate(rgba):
        if a>=64:rgb[i]=(r,g,b);queue.append(i)
    if not queue:raise ValueError('母稿沒有可延拓的墨跡')
    while queue:
        i=queue.popleft();x,y=i%width,i//width
        for j in ([i-1] if x else [])+([i+1] if x+1<width else [])+([i-width] if y else [])+([i+width] if y+1<height else []):
            if rgb[j] is None:rgb[j]=rgb[i];queue.append(j)
    return rgb

def compile_frame(native, master, intensity, which, index):
    size=native.width*6,native.height*6
    support=native.getchannel('A').resize(size,Image.Resampling.LANCZOS).filter(ImageFilter.GaussianBlur(1.1))
    box=native.getchannel('A').getbbox()
    if box is None:raise ValueError('固定來源為空')
    box=tuple(v*6 for v in box)
    material=Image.new('RGBA',size)
    mask=native.getchannel('A').crop(tuple(v//6 for v in box))
    art,angle=painted_material(master,mask,(box[2]-box[0],box[3]-box[1]),which in [3,6])
    material.alpha_composite(art,box[:2])
    if which==7 and index>=177:
        second=Image.new('RGBA',size)
        second.alpha_composite(art.transpose(Image.Transpose.FLIP_LEFT_RIGHT),box[:2])
        pixels=[]
        for first,other in zip(material.getdata(),second.getdata()):pixels.append(other if other[3]>first[3] else first)
        material.putdata(pixels)
    extended=extend_rgb(material)
    warm=Image.new('L',native.size)
    warm.putdata([255 if a and r>g and g>r*.2 and b<g*.6 else 0 for r,g,b,a in native.getdata()])
    warm=warm.resize(size,Image.Resampling.LANCZOS)
    output=Image.new('RGBA',size)
    values=[];used=0
    for y in range(size[1]):
        for x in range(size[0]):
            edge=min(1,min(x,y,size[0]-1-x,size[1]-1-y)/2)
            alpha=round(support.getpixel((x,y))*intensity*edge)
            if not alpha:values.append((0,0,0,0));continue
            rr,gg,bb,aa=material.getpixel((x,y))
            colour=list((rr,gg,bb) if aa>=64 else extended[y*size[0]+x])
            brown=warm.getpixel((x,y))/255
            if brown:
                lum=max(colour);tint=[round(lum*f) for f in [.65,.43,.12]]
                colour=[round(v*(1-brown)+tint[c]*brown) for c,v in enumerate(colour)]
            used+=aa>32
            values.append((*colour,alpha))
    output.putdata(values)
    if output.getchannel('A').getextrema()[0]!=0 or output.getchannel('A').getextrema()[1]==0 or used==0:
        raise ValueError('缺透明背景、墨跡或手繪材質')
    return output,used,angle

def main():
    p=argparse.ArgumentParser()
    for key in ['original-pack','master','base-catalog','out']:p.add_argument('--'+key,type=Path,required=True)
    a=p.parse_args()
    if not Path('/.dockerenv').exists() or a.out.exists() or a.out.parent.stat().st_uid!=os.getuid():raise ValueError('Docker內使用本人擁有的全新候選目錄')
    state=json.loads((ROOT/'docs/data/fd2-modern-hd-battlefield-20261019.json').read_text())['compound_tail_art_followup_20261024']
    if sha(a.master)!=state['master']['sha256']:raise ValueError('手繪原稿SHA不同')
    with Image.open(a.master) as raw:
        if raw.mode!='RGBA' or list(raw.size)!=state['master']['size'] or raw.getchannel('A').getextrema()!=(0,254):raise ValueError('原稿形狀或alpha不同')
        masters=[]
        for cell in state['cells']:
            picture=raw.crop(cell['cell']);box=picture.getchannel('A').point(lambda v:255 if v>=64 else 0).getbbox()
            if list(box)!=cell['core64']:raise ValueError('母稿裁切邊界不同')
            box=(max(0,box[0]-8),max(0,box[1]-8),min(picture.width,box[2]+8),min(picture.height,box[3]+8))
            masters.append(picture.crop(box))
    report,pictures=audit(a.original_pack,{6:RANGES})
    pinned=json.loads((ROOT/state['fixed_sources']['file']).read_text())
    fixed={f['index']:f for b in pinned['resources'] if b['resource']==6 for f in b['frames']}
    bank=report['resources'][0]
    if report['pack_manifest_sha256']!=pinned['pack_manifest_sha256'] or any(json.loads(json.dumps(f))!=fixed[f['index']] for f in bank['frames']):raise ValueError('固定LMI圖源不同')
    catalog=validate(True,a.base_catalog)
    if any(x['index'] in RANGES for asset in catalog['assets'] if asset['role']=='map_overlay_hd_set' and asset['source_resource']==6 for x in asset['native_lmi_frames']):raise ValueError('圖源已登錄')
    old_root=ROOT/catalog['private_root'];old_names=set()
    for asset in catalog['assets']:
        old_names.update(asset.get('files',[]));old_names.update(asset.get('source_files',[]))
        old_names.update(asset[k] for k in ['file','master_file'] if k in asset)
    for name in old_names:
        if Path(name).name!=name or not (old_root/name).is_file():raise ValueError('既有來源缺件')
    a.out.mkdir()
    for name in sorted(old_names):(a.out/name).symlink_to(os.path.relpath(old_root/name,a.out))
    master_name='compound-tail-nine-master-oct24-r2.png';shutil.copyfile(a.master,a.out/master_name)
    original={f['index']:im for resource,f,im in pictures}
    peaks={i:0 for i in range(9)}
    for index,im in original.items():peaks[slot(index)]=max(peaks[slot(index)],max(max(r,g,b) for r,g,b,a in im.getdata() if a))
    compiled=[];new_assets=[]
    for indices in [list(range(64,76)),list(range(126,191))]:
        files=[];descriptions=[]
        for index in indices:
            native=original[index];which=slot(index);peak=max(max(r,g,b) for r,g,b,a in native.getdata() if a)
            output,used,angle=compile_frame(native,masters[which],peak/peaks[which],which,index)
            name=f'compound-tail-fdother006-{index:03d}-hd6-r1.png';output.save(a.out/name);files.append(name)
            row=next(f for f in bank['frames'] if f['index']==index);descriptions.append({k:row[k] for k in ['index','width','height','pixels_sha256']})
            compiled.append({'index':index,'slot':which,'material_angle':angle,'native_ink_box':row['ink_box'],'intensity':peak/peaks[which],'material_pixels':used,'alpha_extrema':output.getchannel('A').getextrema(),'file':name,'sha256':sha(a.out/name)})
        asset=dict(asset_id=f'compound-tail-fdother006-{indices[0]:03d}-{indices[-1]:03d}-hd6',role='map_overlay_hd_set',status='runtime_candidate',files=files,width=1920,height=1200,frame_count=len(files),frame_sha256=[sha(a.out/n) for n in files],consumer_contract='native_lmi1_rgba_hd6_v1',alpha_contract='coverage',cycle_policy='native_lmi1_entries',source_archive='FDOTHER.DAT',source_resource=6,source_metadata_sha256=bank['metadata_sha256'],native_lmi_frames=descriptions,layout_policy='native_lmi1_rect_hd6',source_files=[master_name],source_sha256=[sha(a.master)],source_refs=['docs/knowledge-base/61-modern-theme-runtime-spec.md','tools/build_fd2_compound_tail_hd.py'])
        catalog['assets'].append(asset);new_assets.append(asset['asset_id'])
    catalog['private_root']=str(a.out.resolve().relative_to(ROOT));path=a.out/'catalog.json';path.write_text(json.dumps(catalog,ensure_ascii=False,indent=2)+'\n');validate(True,path)
    recipe={'status':'PASS-COMPOUND-TAIL-ART-CANDIDATE','base_catalog_sha256':sha(a.base_catalog),'catalog_sha256':sha(path),'tool_sha256':sha(Path(__file__)),'source':report,'master_sha256':sha(a.master),'new_assets':new_assets,'records':compiled,'old_file_count':len(old_names),'old_files_are_links':True,'policy':'new painted RGB, nearest ink extension, deterministic8-angle silhouette fit for shards/crescents, opposite slash for cross; native support6x Lanczos blur1.1HDpx/border2HDpx, native brightness ratio and brown region tint; artistic approximation, not raw RGB/alpha parity','production_changed':False}
    (a.out/'compiled.json').write_text(json.dumps(recipe,ensure_ascii=False,indent=2)+'\n')
    sheet=Image.new('RGB',(960,((len(compiled)+7)//8)*160),'#151721');draw=ImageDraw.Draw(sheet)
    for n,row in enumerate(compiled):
        im=Image.open(a.out/row['file']);im.thumbnail((112,124),Image.Resampling.LANCZOS);x,y=n%8*120,n//8*160;sheet.paste(im,(x+(120-im.width)//2,y+28),im);draw.text((x+5,y+5),str(row['index']),fill='white')
    sheet.save(a.out/'overview.png')
    print(json.dumps({'status':recipe['status'],'new_frames':len(compiled),'assets':len(catalog['assets'])}))

if __name__=='__main__':main()
