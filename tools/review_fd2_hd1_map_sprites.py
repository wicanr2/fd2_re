#!/usr/bin/env python3
"""依正式144圖源建立全人物分頁審查板；不宣稱全部重繪完成。"""
import argparse
import hashlib
import html
import json
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont
from build_modern_map_sprite_overview import load_original_palette, LABELS
from validate_modern_theme_catalog import VISIBLE_FDICON_GROUPS

def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()

def build(catalog, modern, original, out):
    if not Path('/.dockerenv').exists() or out.exists() or not out.parent.is_dir():
        raise ValueError('fresh container output required')
    data=json.loads(catalog.read_text())
    assets={a['source_group']:a for a in data['assets'] if a['role']=='map_sprite_hd_set'}
    if set(assets)!=VISIBLE_FDICON_GROUPS:
        raise ValueError('complete visible HD collection required')
    palette_path=original/'palette/fdother_000.json'
    bank_path=original/'sprites/fdicon/bank.json'
    if sha(palette_path)!='404db1ca72ffc19fa7404fbc59bde98cdf239f63fe3375ec248a8142073a28bf':
        raise ValueError('native palette changed')
    bank=json.loads(bank_path.read_text())
    if bank['source']['sha256']!='7efb4448d05f19c1e17ebd53f3e3afead235f5c008d5167548d834c3686b1e44':
        raise ValueError('native bank changed')
    palette=load_original_palette(palette_path)
    records=[]
    # Inspect every admitted file, even when a review page samples six poses.
    for group,asset in sorted(assets.items()):
        inputs=[]
        for frame,(name,digest) in enumerate(zip(asset['files'],asset['frame_sha256'])):
            p=modern/name
            if sha(p)!=digest:
                raise ValueError('HD source changed: '+name)
            with Image.open(p) as image:
                if image.mode!='RGBA' or image.size!=(144,144) or not image.getchannel('A').getbbox():
                    raise ValueError('HD source shape/alpha changed')
            folder=original/'sprites/fdicon'/f'sprite_{group*12+frame:04d}'
            raw={key:sha(folder/key) for key in ('frame.png','mask.png','remap_mask.png')}
            inputs.append(dict(frame=frame,file=name,sha256=digest,native=raw))
        if len(inputs)!=asset['frame_count']:
            raise ValueError('incomplete HD frame table')
        frames=[f for f in (0,1,2,3,6,9) if f<len(inputs)]
        records.append(dict(group=group,label=LABELS.get(group,'Gaia base selector' if group==30 else 'selector-only; identity not inferred'),
                            frame_count=len(inputs),visual_frames=frames,inputs=inputs))
    if len(records)!=125 or sum(r['frame_count'] for r in records)!=1455:
        raise ValueError('native collection count changed')
    out.mkdir()
    font=ImageFont.load_default(size=17)
    small=ImageFont.load_default(size=13)
    pages=[]
    for start in range(0,len(records),10):
        rows=records[start:start+10]
        image=Image.new('RGB',(1376,68+250*len(rows)),'#101b2a')
        draw=ImageDraw.Draw(image)
        draw.text((20,12),'HD1 SOURCE REVIEW / ORIGINAL + ACTUAL 144px / NOT A COMPLETION GALLERY',font=font,fill='#ffe09c')
        draw.text((20,38),'Six sampled poses per walking group; all admitted frame files checked. Right: 144 / 72 / 48.',font=small,fill='#b1c5de')
        for i,row in enumerate(rows):
            y=68+i*250;group=row['group'];asset=assets[group]
            draw.line((16,y,1358,y),fill='#536376')
            draw.text((20,y+10),f'GROUP {group:03d}',font=font,fill='#ffe09c')
            draw.text((20,y+38),'ORIGINAL',font=small,fill='white')
            draw.text((20,y+121),'HD1',font=small,fill='white')
            for col,f in enumerate(row['visual_frames']):
                x=170+col*144;folder=original/'sprites/fdicon'/f'sprite_{group*12+f:04d}'
                with Image.open(folder/'frame.png') as indexed:
                    if indexed.mode!='P' or indexed.size!=(24,24):
                        raise ValueError('indexed native source required')
                    indexed.putpalette(palette);native=indexed.convert('RGBA')
                with Image.open(folder/'mask.png') as mask:
                    native.putalpha(mask.convert('L'))
                native=native.resize((72,72),Image.Resampling.NEAREST)
                image.paste(native,(x+36,y+26),native)
                with Image.open(modern/asset['files'][f]) as source:
                    hd=source.convert('RGBA')
                image.paste(hd,(x,y+102),hd)
                draw.text((x+4,y+5),f'f{f:02d}',font=small,fill='#a0b6d0')
            with Image.open(modern/asset['files'][0]) as source:
                front=source.convert('RGBA')
            for x,yy,size in [(1056,y+24,144),(1214,y+28,72),(1226,y+134,48)]:
                sample=front if size==144 else front.resize((size,size),Image.Resampling.LANCZOS)
                image.paste(sample,(x,yy),sample)
                draw.text((x,yy+size+1),str(size),font=small,fill='#b1c5de')
        path=out/f'page-{len(pages)+1:02d}.png';image.save(path)
        pages.append(dict(file=path.name,sha256=sha(path),groups=[r['group'] for r in rows],size=list(image.size)))
    links=''.join(f'<li><a href="#{i}">Groups {p["groups"][0]:03d}–{p["groups"][-1]:03d}</a></li>' for i,p in enumerate(pages))
    pictures=''.join(f'<section id="{i}"><h2>{html.escape(p["file"])}</h2><a href="{p["file"]}"><img src="{p["file"]}" alt="Source review page {i+1}"></a></section>' for i,p in enumerate(pages))
    (out/'index.html').write_text('<!doctype html><meta charset="utf-8"><title>HD1 人物圖源審查</title><style>body{background:#101b2a;color:#e8edf5;font:18px sans-serif;margin:24px}img{max-width:100%;height:auto}li{display:inline-block;margin:8px}a{color:#ffe09c}</style><h1>HD1 人物圖源審查</h1><p>原版與正式144圖源。這是檢查用分頁，尚非全人物重繪完成總覽。點圖可檢視原尺寸；行走組取四向中性與正面三拍，靜態組只顯示原有三拍。</p><ul>'+links+'</ul>'+pictures)
    report=dict(kind='fd2_hd1_map_sprite_review_sources',status='PASS-REVIEW-SOURCE-BOARDS',
                catalog_sha256=sha(catalog),tool_sha256=sha(Path(__file__)),palette_sha256=sha(palette_path),bank_sha256=sha(bank_path),
                reference_helper_sha256=sha(Path(__file__).with_name('build_modern_map_sprite_overview.py')),
                groups=records,pages=pages,groups_checked=125,frames_checked=1455,
                visual_review_complete=False,all_redrawn=False,original_parity_verified=False,
                scope='全圖源SHA／尺寸／alpha核對；分頁取四向中性及正面三拍，並列原版與144／72／48。逐組造型結論另記錄，不把來源通過當成造型通過。')
    (out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({k:report[k] for k in ('status','groups_checked','frames_checked','visual_review_complete','all_redrawn')},ensure_ascii=False))

if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('catalog','modern','original','out'):
        p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();build(a.catalog,a.modern,a.original,a.out)
