#!/usr/bin/env python3
"""依單位轉場契約編製11張HD候選，完整保留舊清冊及未修改母稿。"""
import argparse
import hashlib
import json
import os
import shutil
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter
from audit_fd2_player9_map_source import audit
from build_fd2_compound_tail_hd import extend_rgb
from validate_modern_theme_catalog import ROOT, validate

INDICES = list(range(114, 125))


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def stroke_coverage(mask):
    # Reconstruct the source's connected light strokes at HD resolution.
    # Four-times supersampling gives round joins rather than enlarged squares.
    scale, samples = 6, 4
    points={(x,y) for y in range(mask.height) for x in range(mask.width) if mask.getpixel((x,y))}
    canvas=Image.new('L',(mask.width*scale*samples,mask.height*scale*samples))
    paint=ImageDraw.Draw(canvas)
    width=round(scale*.9*samples)
    radius=width/2
    centre=lambda p:((p[0]+.5)*scale*samples,(p[1]+.5)*scale*samples)
    for x,y in sorted(points):
        a=centre((x,y))
        for dx,dy in [(1,0),(0,1),(1,1),(-1,1)]:
            if (x+dx,y+dy) in points:
                paint.line([a,centre((x+dx,y+dy))],fill=255,width=width)
        paint.ellipse((a[0]-radius,a[1]-radius,a[0]+radius,a[1]+radius),fill=255)
    return canvas.resize((mask.width*scale,mask.height*scale),Image.Resampling.LANCZOS).filter(ImageFilter.GaussianBlur(.45))


def compile_frame(native, material):
    size = native.width*6, native.height*6
    mask = native.getchannel('A')
    box = mask.getbbox()
    if box is None:
        raise ValueError('已審查的11張來源不得為空')
    box6 = tuple(v*6 for v in box)
    colour = Image.new('RGB', native.size)
    colour.putdata(extend_rgb(native))
    colour = colour.resize(size, Image.Resampling.LANCZOS)
    art = material.resize((box6[2]-box6[0], box6[3]-box6[1]), Image.Resampling.LANCZOS)
    coverage = stroke_coverage(mask)
    support = mask.resize(size, Image.Resampling.NEAREST).filter(ImageFilter.MaxFilter(5))
    out = Image.new('RGBA', size)
    for y in range(size[1]):
        for x in range(size[0]):
            if not support.getpixel((x,y)):
                continue
            alpha = coverage.getpixel((x,y))
            if alpha >= 224:
                alpha = 255
            if not alpha:
                continue
            base = colour.getpixel((x,y))
            rgb = art.getpixel((min(max(x-box6[0],0),art.width-1),min(max(y-box6[1],0),art.height-1)))
            light = (rgb[0]*.2126+rgb[1]*.7152+rgb[2]*.0722)/255
            # Preserve each source frame's yellow/red hue and dark release
            # values; the generated texture supplies the brush/highlight only.
            highlight = max(base)*.12*light**4
            value = tuple(min(255,round(v*(.72+.4*light)+highlight)) for v in base)
            out.putpixel((x,y),(*value,alpha))
    if out.getchannel('A').getextrema() != (0,255):
        raise ValueError('候選缺少透明邊界或不透明核心')
    return out,box


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ['original-pack','base-catalog','out']:
        p.add_argument('--'+name,type=Path,required=True)
    p.add_argument('--master-root',type=Path)
    a=p.parse_args()
    if not Path('/.dockerenv').exists() or a.out.exists() or not a.out.parent.is_dir() or a.out.parent.stat().st_uid!=os.getuid():
        raise ValueError('在Docker使用既有、本人擁有的工作根與全新候選目錄')
    state=json.loads((ROOT/'docs/data/fd2-modern-hd-battlefield-20261019.json').read_text())['unit_present_hd_followup_20261024']
    if state['status'] not in ['READY','CONFORMED-SCOPED-RUNTIME-E1'] or sha(a.base_catalog)!=state['base_catalog']['sha256']:
        raise ValueError('工程未READY或固定基底清冊不同')
    source=state['masters'][0]
    master=a.master_root/Path(source['file']).name if a.master_root else ROOT/source['file']
    if sha(master)!=source['sha256']:
        raise ValueError('未修改母稿SHA不同')
    with Image.open(master) as im:
        if im.mode!='RGBA' or list(im.size)!=source['size'] or list(im.getchannel('A').getextrema())!=source['alpha_extrema']:
            raise ValueError('母稿形狀或alpha不同')
        box=im.getchannel('A').point(lambda v:255 if v>=32 else 0).getbbox()
        if list(box)!=source['crop32']:
            raise ValueError('母稿材質裁切不同')
        crop=im.crop(box)
        material=Image.new('RGB',crop.size)
        material.putdata(extend_rgb(crop))
        material=material.filter(ImageFilter.GaussianBlur(1))
    fixed=state['fixed_sources']['report']
    report,pictures=audit(a.original_pack,{6:INDICES})
    if json.loads(json.dumps(report))!=fixed:
        raise ValueError('原版LMI來源與已審查收據不同')
    catalog=validate(True,a.base_catalog)
    if len(catalog['assets'])!=764:
        raise ValueError('不是已固定764項基底')
    old_root=ROOT/catalog['private_root']
    names=set()
    for asset in catalog['assets']:
        names.update(asset.get('files',[]));names.update(asset.get('source_files',[]))
        names.update(asset[k] for k in ['file','master_file'] if k in asset)
        if asset['role']=='map_overlay_hd_set' and asset['source_resource']==6 and any(f['index'] in INDICES for f in asset['native_lmi_frames']):
            raise ValueError('11張轉場來源已有登錄，不可覆寫')
    if any(Path(n).name!=n or not (old_root/n).is_file() for n in names):
        raise ValueError('既有圖源缺件或路徑不安全')
    a.out.mkdir()
    for name in sorted(names):
        (a.out/name).symlink_to(os.path.relpath(old_root/name,a.out))
    master_name=Path(source['file']).name
    if master_name in names:
        raise ValueError('新母稿名稱與舊素材衝突')
    shutil.copyfile(master,a.out/master_name)
    files,descriptors,records=[],[],[]
    for _,row,native in pictures:
        index=row['index'];out,box=compile_frame(native,material)
        name=f'unit-present-fdother006-{index:03d}-hd6-r2.png'
        out.save(a.out/name);files.append(name)
        descriptors.append({k:row[k] for k in ['index','width','height','pixels_sha256']})
        records.append({'index':index,'file':name,'sha256':sha(a.out/name),'native_ink_bbox':list(box),'size':list(out.size),'alpha_extrema':list(out.getchannel('A').getextrema())})
    asset=dict(asset_id='unit-present-fdother006-114-124-hd6',role='map_overlay_hd_set',status='runtime_candidate',files=files,width=1920,height=1200,frame_count=11,
               frame_sha256=[sha(a.out/n) for n in files],consumer_contract='native_lmi1_rgba_hd6_v1',alpha_contract='coverage',cycle_policy='native_lmi1_entries',source_archive='FDOTHER.DAT',source_resource=6,
               source_metadata_sha256=report['resources'][0]['metadata_sha256'],native_lmi_frames=descriptors,layout_policy='native_lmi1_rect_hd6',
               source_files=[master_name],source_sha256=[source['sha256']],source_refs=['docs/knowledge-base/61-modern-theme-runtime-spec.md','tools/build_fd2_unit_present_hd.py'])
    catalog['assets'].append(asset);catalog['private_root']=str(a.out.resolve().relative_to(ROOT))
    path=a.out/'catalog.json';path.write_text(json.dumps(catalog,ensure_ascii=False,indent=2)+'\n')
    validate(True,path)
    result=dict(status='PASS-UNIT-PRESENT-ART-CANDIDATE',base_catalog_sha256=sha(a.base_catalog),catalog_sha256=sha(path),tool_sha256=sha(Path(__file__)),records=records,
                original_sources=report,master=source,old_files=len(names),old_files_are_links=True,production_changed=False,
                policy='原生支撐、孔洞及各幀黃／紅／暗紅值控制輸出；四倍超取樣中心線、圓接、兩高解析像素支撐外緣、筆觸與高光為明示美術近似。')
    (a.out/'compiled.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    sheet=Image.new('RGB',(6*210,2*200),'#131b27');draw=ImageDraw.Draw(sheet)
    for i,row in enumerate(records):
        with Image.open(a.out/row['file']) as im:
            x,y=i%6*210,i//6*200;sheet.paste(im,(x+(210-im.width)//2,y+28),im);draw.text((x+8,y+7),str(row['index']),fill='white')
    sheet.save(a.out/'overview.png')
    print(json.dumps({'status':result['status'],'frames':len(records),'catalog_assets':len(catalog['assets']),'old_files':len(names)}))


if __name__=='__main__':
    main()
