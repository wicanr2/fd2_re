#!/usr/bin/env python3
"""比較舊描述子框與腳底／身高對齊的可丟棄排版，不改圖源或正式清冊。"""
import argparse
import hashlib
import json
from pathlib import Path
from PIL import Image, ImageDraw


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--pack',required=True,type=Path)
    p.add_argument('--master',required=True,type=Path)
    p.add_argument('--out',required=True,type=Path)
    args=p.parse_args()
    root=args.pack/'animations/FIGANI_090'
    frame=json.loads((root/'animation.json').read_text())['frames'][0]
    native=Image.open(root/frame['path']).convert('RGBA')
    native.putalpha(Image.open(root/frame['mask_path']).convert('L'))
    master=Image.open(args.master).convert('RGBA')
    art=master.crop(master.getchannel('A').getbbox())
    w,h=frame['width']*6,frame['height']*6
    old_scale=min(w/art.width,h/art.height)
    old=art.resize((round(art.width*old_scale),round(art.height*old_scale)),Image.Resampling.LANCZOS)
    height_scale=h/art.height
    height_fit=art.resize((round(art.width*height_scale),h),Image.Resampling.LANCZOS)
    panels=[]
    for label,picture in [('Original decoded',native.resize((w,h),Image.Resampling.NEAREST)),('A: contain original box',old),('B: align height and feet',height_fit)]:
        canvas=Image.new('RGBA',(1920,1200),(35,41,50,255))
        x=frame['x']*6+(w-picture.width)//2
        y=(frame['y']+frame['height'])*6-picture.height
        canvas.alpha_composite(picture,(x,y))
        draw=ImageDraw.Draw(canvas)
        draw.rectangle((frame['x']*6,frame['y']*6,(frame['x']+frame['width'])*6-1,(frame['y']+frame['height'])*6-1),outline=(240,120,80),width=3)
        panel=canvas.convert('RGB').resize((640,400),Image.Resampling.LANCZOS)
        ImageDraw.Draw(panel).text((12,12),label,fill='white')
        panels.append(panel)
    sheet=Image.new('RGB',(1920,400))
    for index,panel in enumerate(panels):sheet.paste(panel,(index*640,0))
    sheet.save(args.out)
    args.out.with_suffix('.json').write_text(json.dumps({'scope':'layout-only; no GUI/parity; same unmodified HD master',
        'native_frame':[frame[k] for k in ['x','y','width','height']], 'contain_size':list(old.size),
        'height_fit_size':list(height_fit.size),'height_fit_extra_width':height_fit.width-w,
        'master_sha256':hashlib.sha256(args.master.read_bytes()).hexdigest()},indent=2)+'\n')


if __name__=='__main__':main()
