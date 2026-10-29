#!/usr/bin/env python3
"""以候選實際144圖源製作分組造型對照，不替代完整人物總覽。"""
import argparse
import hashlib
import json
from pathlib import Path
from PIL import Image,ImageDraw,ImageFont
from build_modern_map_sprite_overview import load_original_palette

def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def build(candidate,original,out):
    if out.exists() or not out.parent.is_dir():raise ValueError('fresh owned output required')
    m=json.loads((candidate/'manifest.json').read_text())
    groups=m['groups'];n=len(groups)
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
    return dict(kind='fd2_hd1_chibi_partial_comparison',groups=[g['group'] for g in groups],source_manifest_sha256=sha(candidate/'manifest.json'),tool_sha256=sha(Path(__file__)),image_sha256=sha(out),width=image.width,height=image.height,sources=provenance,scope='原版來源與實際HD1四向中性素材對照及144／72／48尺寸，非遊戲截圖或全人物完成。')

def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ('candidate','original','out'):p.add_argument('--'+n,type=Path,required=True)
    a=p.parse_args();report=a.out.with_suffix('.json')
    if report.exists():raise ValueError('fresh report required')
    d=build(a.candidate,a.original,a.out);report.write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n');print(json.dumps(d,ensure_ascii=False))
if __name__=='__main__':main()
