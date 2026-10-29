#!/usr/bin/env python3
"""以候選實際144圖源製作分組造型對照，不替代完整人物總覽。"""
import argparse
import hashlib
import json
from pathlib import Path
from PIL import Image,ImageDraw,ImageFont
from build_modern_map_sprite_overview import load_original_palette

def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def build(candidate,original,out,context=None):
    if out.exists() or not out.parent.is_dir():raise ValueError('fresh owned output required')
    m=json.loads((candidate/'manifest.json').read_text())
    groups=m['groups']
    if context is not None:
        catalog=json.loads((candidate/'catalog.json').read_text())
        by_role={(a.get('source_group'),a['role']):a for a in catalog['assets']}
        selected={g['group'] for g in groups};kept=[]
        for g in json.loads(context.read_text())['groups']:
            if g['group'] in selected:continue
            for role,key in (('parent','map_sprite_set'),('hd','map_sprite_hd_set')):
                if g[role]!=by_role.get((g['group'],key)):raise ValueError('context art differs from current candidate')
            kept.append(g)
        groups=sorted(kept+groups,key=lambda g:g['group'])
        if len({g['group'] for g in groups})!=len(groups):raise ValueError('duplicate context group')
    n=len(groups)
    if not n:raise ValueError('no groups')
    palette=load_original_palette(original/'palette/fdother_000.json')
    image=Image.new('RGB',(1300,65+510*n+50),'#091627');draw=ImageDraw.Draw(image)
    font=ImageFont.load_default(size=20);small=ImageFont.load_default(size=16)
    draw.text((24,14),'HD1 CHIBI REFRESH / ORIGINAL + HD1 / PARTIAL BATCH',font=font,fill='white')
    provenance=[]
    for row,g in enumerate(groups):
        y=65+row*510;group=g['group']
        draw.text((24,y),f'GROUP {group:03d}',font=font,fill='#ffe59b')
        draw.text((24,y+40),'ORIGINAL',font=font,fill='white')
        draw.text((24,y+195),'HD1 144px',font=font,fill='white')
        for col,f in enumerate((0,3,6,9)):
            x=230+col*210;p=original/'sprites/fdicon'/f'sprite_{group*12+f:04d}'
            for name,digest in g['native_inputs'][f].items():
                if sha(p/name)!=digest:raise ValueError('native frame changed')
            with Image.open(p/'frame.png') as im:
                if im.mode!='P':raise ValueError('expected original indexed frame')
                im.putpalette(palette);source=im.convert('RGBA')
            with Image.open(p/'mask.png') as im:source.putalpha(im.convert('L'))
            source=source.resize((144,144),Image.Resampling.NEAREST)
            image.paste(source,(x,y+35),source)
            path=candidate/g['hd']['files'][f]
            if sha(path)!=g['hd']['frame_sha256'][f]:raise ValueError('HD frame changed')
            with Image.open(path) as im:hd=im.convert('RGBA')
            if hd.size!=(144,144):raise ValueError('expected actual144 source')
            image.paste(hd,(x,y+190),hd)
            provenance.append(dict(group=group,frame=f,file=path.name,sha256=sha(path)))
        with Image.open(candidate/g['hd']['files'][0]) as im:source=im.convert('RGBA')
        for x,size in ((230,144),(460,72),(640,48)):
            hd=source.resize((size,size),Image.Resampling.LANCZOS)
            image.paste(hd,(x,y+350),hd)
            draw.text((x+size+6,y+355),str(size),font=font,fill='#aec5e0')
    draw.text((24,image.height-30),'Partial batch / art-source comparison / not a runtime screenshot or complete roster',font=small,fill='#aec5e0')
    image.save(out)
    return dict(kind='fd2_hd1_chibi_partial_comparison',groups=[g['group'] for g in groups],source_manifest_sha256=sha(candidate/'manifest.json'),context_manifest_sha256=sha(context) if context else None,tool_sha256=sha(Path(__file__)),image_sha256=sha(out),width=image.width,height=image.height,sources=provenance,scope='原版來源與實際HD1四向中性素材對照及144／72／48尺寸，非遊戲截圖或全人物完成。')

def build_revision(candidate,original,baseline_catalog,baseline_pack,out):
    if out.exists():raise ValueError('fresh revision image required')
    m=json.loads((candidate/'manifest.json').read_text())
    if len(m['groups'])!=1:raise ValueError('one revision group required')
    group=m['groups'][0];identity=group['group']
    catalog=json.loads(baseline_catalog.read_text())
    old=next(a for a in catalog['assets'] if a['role']=='map_sprite_hd_set' and a['source_group']==identity)
    if sha(baseline_catalog)!=m['base_catalog_sha256']:raise ValueError('baseline changed')
    palette=load_original_palette(original/'palette/fdother_000.json')
    image=Image.new('RGB',(1300,1000),'#091627');draw=ImageDraw.Draw(image)
    font=ImageFont.load_default(size=20);small=ImageFont.load_default(size=16)
    draw.text((24,18),'GAIA HEAVY REVISION / ORIGINAL + PREVIOUS + CANDIDATE / NOT REGISTERED',font=font,fill='#ffe59b')
    for y,label in [(82,'ORIGINAL'),(252,'PREVIOUS'),(422,'HEAVY CANDIDATE')]:
        draw.text((24,y+30),label,font=small,fill='white')
    records=[]
    for col,f in enumerate((0,3,6,9)):
        x=230+col*210;folder=original/'sprites/fdicon'/f'sprite_{identity*12+f:04d}'
        for name,digest in group['native_inputs'][f].items():
            if sha(folder/name)!=digest:raise ValueError('original source changed')
        with Image.open(folder/'frame.png') as source:
            source.putpalette(palette);native=source.convert('RGBA')
        with Image.open(folder/'mask.png') as mask:native.putalpha(mask.convert('L'))
        native=native.resize((144,144),Image.Resampling.NEAREST);image.paste(native,(x,82),native)
        for row,asset,root in [('previous',old,baseline_pack),('candidate',group['hd'],candidate)]:
            path=root/asset['files'][f]
            if sha(path)!=asset['frame_sha256'][f]:raise ValueError('revision source changed')
            with Image.open(path) as source:frame=source.convert('RGBA')
            if frame.size!=(144,144):raise ValueError('actual144 source required')
            image.paste(frame,(x,252 if row=='previous' else 422),frame)
            records.append(dict(row=row,frame=f,file=path.name,sha256=sha(path)))
    for row,asset,root,y in [('previous',old,baseline_pack,622),('candidate',group['hd'],candidate,792)]:
        draw.text((24,y+35),row.upper()+' SIZE',font=small,fill='white')
        with Image.open(root/asset['files'][0]) as source:front=source.convert('RGBA')
        for x,size in ((230,144),(460,72),(640,48)):
            frame=front if size==144 else front.resize((size,size),Image.Resampling.LANCZOS)
            image.paste(frame,(x,y),frame);draw.text((x+size+6,y+8),str(size),font=font,fill='#aec5e0')
    draw.text((24,968),'Design comparison only; existing game artwork remains unchanged until this revision is accepted.',font=small,fill='#aec5e0')
    image.save(out)
    return dict(kind='fd2_hd1_chibi_revision_prototype',status='DESIGN-CANDIDATE-NOT-REGISTERED',group=identity,
                catalog_sha256=sha(baseline_catalog),candidate_manifest_sha256=sha(candidate/'manifest.json'),
                tool_sha256=sha(Path(__file__)),image_sha256=sha(out),size=list(image.size),sources=records)

def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ('candidate','original','out'):p.add_argument('--'+n,type=Path,required=True)
    p.add_argument('--context-manifest',type=Path,help='沿用同批其他已登錄人物；必須與目前候選清冊逐欄一致')
    p.add_argument('--baseline-catalog',type=Path,help='另排原版／前版／候選，尚未登錄的造型對照')
    p.add_argument('--baseline-pack',type=Path)
    a=p.parse_args();report=a.out.with_suffix('.json')
    if report.exists():raise ValueError('fresh report required')
    if (a.baseline_catalog is None)!=(a.baseline_pack is None):raise ValueError('paired baseline arguments required')
    d=build_revision(a.candidate,a.original,a.baseline_catalog,a.baseline_pack,a.out) if a.baseline_catalog else build(a.candidate,a.original,a.out,a.context_manifest)
    report.write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n');print(json.dumps(d,ensure_ascii=False))
if __name__=='__main__':main()
