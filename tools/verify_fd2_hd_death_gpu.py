#!/usr/bin/env python3
"""獨立重組事件死亡fixture的實際GPU畫面，不作原版oracle聲明。"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageChops, ImageFilter, ImageMath


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def checked(root, record):
    name=record['file']
    if Path(name).name!=name or not name.endswith('.png') or digest(root/name)!=record['sha256']:
        raise ValueError('unsafe image or sha256 mismatch')
    return Image.open(root/name).convert('RGBA')


def rect(value):
    return (value['Min']['X'],value['Min']['Y'],value['Max']['X'],value['Max']['Y'])


def clipped(world, picture, position, viewport, go_rgba=False):
    x,y=position
    box=(max(x,viewport[0],0),max(y,viewport[1],0),
         min(x+picture.width,viewport[2],world.width),min(y+picture.height,viewport[3],world.height))
    if box[0]>=box[2] or box[1]>=box[3]: return
    source=picture.crop((box[0]-x,box[1]-y,box[2]-x,box[3]-y))
    if go_rgba:
        # Go1.22 image/draw.drawNRGBASrc stores premultiplied bytes as
        # ((channel * alpha * 0x101 / 0xff) >> 8). GPU SourceOver then blends
        # those quantized bytes. Pillow's straight-alpha composition alone
        # accumulates a different rounding error in overlapping smoke layers.
        alpha=source.getchannel('A').convert('I')
        destination=world.crop(box)
        channels=[]
        for name in ('R','G','B'):
            channels.append(ImageMath.unsafe_eval(
                "convert((s * a * 257 / 255) / 256 + (d * (255 - a) + 127) / 255, 'L')",
                s=source.getchannel(name).convert('I'),a=alpha,d=destination.getchannel(name).convert('I')))
        world.paste(Image.merge('RGBA',tuple(channels)+(Image.new('L',source.size,255),)),box[:2])
    else:
        world.alpha_composite(source,box[:2])


def validate(catalog_path, pack, root):
    catalog=json.loads(catalog_path.read_text())
    bank=next(a for a in catalog['assets'] if a['role']=='map_overlay_hd_set')
    if [f['index'] for f in bank['native_lmi_frames']]!=list(range(68,80)):
        raise ValueError('source identity mismatch')
    effects={}
    for desc,name,sha in zip(bank['native_lmi_frames'],bank['files'],bank['frame_sha256']):
        picture=checked(pack,dict(file=name,sha256=sha))
        if picture.size!=(desc['width']*6,desc['height']*6): raise ValueError('source geometry mismatch')
        effects[desc['index']]=picture
    units={a['source_group']:a for a in catalog['assets'] if a['role']=='map_sprite_hd_set'}
    report=json.loads((root/'receipt.json').read_text())
    if (report['status']!='PASS-EVENT-DEATH-FIXTURE-RUNTIME-E1' or report['diagnostic'] or
            report['hd_frame_count']!=26 or report['continuations']!=1 or report['faithful_draws']<4 or
            report['f2_phases']!=['pose','overlay_a','overlay_b','steady'] or
            [r['frame'] for r in report['frames']]!=list(range(26))):
        raise ValueError('incomplete lifecycle receipt')
    results=[]
    for record in report['frames']:
        index=record['frame']
        viewport=(0,0,1872,1152) if index<25 else (24,24,1896,1176)
        recorded_viewport=rect(record['viewport'])
        if recorded_viewport not in (viewport,(0,0,0,0)) or (index<25 and recorded_viewport!=viewport):
            raise ValueError('event viewport mismatch')
        if record['terrain_count'] or record['foreground_count']:
            raise ValueError('this fixture verifier requires explicit zero terrain/foreground HD layers')
        background=checked(root,record['background'])
        ui=checked(root,record['ui'])
        if background.size!=(320,200) or ui.size!=(320,200): raise ValueError('bad native layer geometry')
        world=background.resize((1920,1200),Image.Resampling.NEAREST)
        for unit in record['unit_draws'] or []:
            asset=units[unit['Group']]
            frame=unit['Frame']
            picture=checked(pack,dict(file=asset['files'][frame],sha256=asset['frame_sha256'][frame]))
            if unit['PaletteBand']:
                channels=picture.split()
                picture=Image.merge('RGBA',tuple(channels[i].point([round(v*f) for v in range(256)]) for i,f in enumerate((.55,.55,.6)))+(channels[3],))
            box=rect(unit['Rect'])
            if picture.size!=(box[2]-box[0],box[3]-box[1]): raise ValueError('bad unit geometry')
            clipped(world,picture,box[:2],viewport)
        expected=[]
        if 13<=index<19: entries=range(68,68+index-12)
        elif 19<=index<25: entries=[74+index-19]
        else: entries=[]
        for entry in entries:
            for target in report['targets']:
                origin=target['Origin']
                x=(origin%456-0x8088%456)*6
                y=(origin//456-0x8088//456)*6
                picture=effects[entry]
                expected.append((entry,(x,y,x+picture.width,y+picture.height)))
        actual=[(d['entry'],rect(d['rect'])) for d in record['overlay_draws'] or []]
        if actual!=expected or len(actual)!=record['overlays']:
            raise ValueError(f'phase{index} accumulation/reset/order/origin mismatch')
        for entry,box in expected: clipped(world,effects[entry],box[:2],viewport,go_rgba=True)
        actual=checked(root,record)
        if actual.size!=(1920,1200): raise ValueError('bad final screenshot geometry')
        # UI uses the existing final-screen filter. Compare world pixels away
        # from its support, rather than inventing a second UI filtering rule.
        mask=ui.getchannel('A').resize((1920,1200),Image.Resampling.NEAREST).filter(ImageFilter.MaxFilter(13))
        visible=mask.point([255 if v==0 else 0 for v in range(256)])
        diff=ImageChops.difference(actual.convert('RGB'),world.convert('RGB'))
        maximum=ImageChops.lighter(ImageChops.lighter(diff.getchannel('R'),diff.getchannel('G')),diff.getchannel('B'))
        errors=ImageChops.multiply(maximum.point([255 if v>3 else 0 for v in range(256)]),visible).histogram()[255]
        count=visible.histogram()[255]
        if count<1000000 or errors:
            raise ValueError(f'GPU frame{index}: {errors} RGB errors out of {count}')
        results.append(dict(frame=index,compared_pixels=count,rgb_errors_gt3=errors,overlay_layers=len(expected)))
    return dict(status='PASS-HD-DEATH-GPU-RECONSTRUCTION',frames=results,frame_count=26,
                catalog_sha256=digest(catalog_path),receipt_sha256=digest(root/'receipt.json'),
                scope='獨立Pillow重組來源PNG、原生origin與實際Game.Draw回讀；非原版oracle或PLAYER-E2。')


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--catalog',type=Path,required=True)
    parser.add_argument('--pack',type=Path,required=True)
    parser.add_argument('--run',type=Path,required=True)
    parser.add_argument('--out',type=Path,required=True)
    args=parser.parse_args()
    result=validate(args.catalog,args.pack,args.run)
    args.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(result['status'],result['frame_count'])
