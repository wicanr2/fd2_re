#!/usr/bin/env python3
"""由固定LMI1與原RGBA編製玩家9地圖效果；僅寫明示候選目錄。"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageChops, ImageDraw, ImageFont, ImageOps
from audit_fd2_player9_map_source import audit, RANGES, FOLDERS

MASTER_KEYS = {'hood-closed', 'hood-open', 'hood-silhouette', 'dark-ring', 'scythe-arc'}
FONT_SHA = 'b76b0433203017ca80401b2ee0dd69350349871c4b19d504c34dbdd80541690a'


def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()


def source_identity(report):
    # Canonical JSON values: Pillow bounding boxes are tuples in memory and
    # arrays in the persisted receipt. Preserve all values without coercion.
    return json.loads(json.dumps({k: report[k] for k in ['source_file', 'pack_manifest_sha256', 'palette_sha256', 'resources']}))


def checked_source(root, row):
    if not isinstance(row, dict) or set(row) != {'file', 'sha256'}:
        raise ValueError('invalid source descriptor')
    name = row['file']
    if not isinstance(name, str) or Path(name).name != name or Path(name).suffix != '.png':
        raise ValueError('unsafe master path')
    p = root/name
    if sha(p) != row['sha256']:
        raise ValueError('master sha256 mismatch')
    with Image.open(p) as im:
        if im.mode != 'RGBA' or im.getchannel('A').getextrema() != (0, 255):
            raise ValueError('master must retain genuine RGBA coverage')
        return im.copy()


def glyph(char, size, font_path):
    # Direct vector-font rasterization at output resolution, no pixel-font scale.
    for pixels in range(size[1], 1, -1):
        font = ImageFont.truetype(str(font_path), pixels, index=3)
        box = font.getbbox(char, stroke_width=2)
        if box[2]-box[0] <= size[0]-4 and box[3]-box[1] <= size[1]-4:
            break
    else:
        raise ValueError('font cannot fit the native result rectangle')
    result = Image.new('RGBA', size)
    x = (size[0]-(box[2]-box[0]))//2-box[0]
    y = (size[1]-(box[3]-box[1]))//2-box[1]
    ImageDraw.Draw(result).text((x,y), char, font=font, fill=(230,251,255,255),
                               stroke_width=2, stroke_fill=(0,9,15,255))
    return result


def components(mask):
    todo = {(x,y) for y in range(mask.height) for x in range(mask.width) if mask.getpixel((x,y))}
    groups = []
    while todo:
        seed = min(todo, key=lambda p:(p[1],p[0])); todo.remove(seed)
        group, stack = [seed], [seed]
        while stack:
            x,y = stack.pop()
            for dx in [-1,0,1]:
                for dy in [-1,0,1]:
                    p = (x+dx,y+dy)
                    if p in todo:
                        todo.remove(p); group.append(p); stack.append(p)
        groups.append((min(x for x,y in group),min(y for x,y in group),
                       max(x for x,y in group)+1,max(y for x,y in group)+1))
    return groups


def fitted(master, box, mirror=False):
    cropped = master.crop(master.getchannel('A').getbbox())
    if mirror:
        cropped = ImageOps.mirror(cropped)
    return cropped.resize(((box[2]-box[0])*6,(box[3]-box[1])*6),Image.Resampling.LANCZOS)


def effect(index, native, row, masters):
    size=(native.width*6,native.height*6)
    out=Image.new('RGBA',size)
    mask=native.getchannel('A')
    box=mask.getbbox()
    if box is None:
        raise ValueError('unexpected blank player9 native source')
    record={'index':index,'native_ink_box':list(box)}
    if index <= 96:
        # One fixed full-sized bust rises; it is not stretched to each early cap.
        full=fitted(masters['hood-closed'],(0,0,24,24))
        out.alpha_composite(full,(0,box[1]*6))
        record.update(source='hood-closed',policy='fixed_24px_bust_rise_native_crop',shift_y=box[1]*6)
    elif index <= 100:
        key={97:'hood-open',98:'hood-closed',99:'hood-open',100:'hood-silhouette'}[index]
        out.alpha_composite(fitted(masters[key],box),(box[0]*6,box[1]*6))
        record.update(source=key,policy='fit_native_ink_box')
    elif index <= 103:
        out.alpha_composite(fitted(masters['dark-ring'],box),(box[0]*6,box[1]*6))
        # Dissolve/reveal mask is original animation data, not an edited master.
        out.putalpha(ImageChops.multiply(out.getchannel('A'),mask.resize(size,Image.Resampling.LANCZOS)))
        record.update(source='dark-ring',policy='native_dissolve_clip')
    elif index <= 106:
        part=masters['dark-ring'].crop(masters['dark-ring'].getchannel('A').getbbox())
        part=part.crop((part.width//3,0,part.width*2//3,part.height//3))
        boxes=components(mask)
        for b in boxes:
            out.alpha_composite(fitted(part,b),(b[0]*6,b[1]*6))
        record.update(source='dark-ring',policy='native_particle_components',components=[list(b) for b in boxes])
    elif index in [108,109,110]:
        out.alpha_composite(fitted(masters['scythe-arc'],box,mirror=index==110),(box[0]*6,box[1]*6))
        record.update(source='scythe-arc',policy='native_arc_box',mirror_x=index==110)
    else:
        mother=masters['scythe-arc'].crop(masters['scythe-arc'].getchannel('A').getbbox())
        boxes=components(mask)
        # Native palette class selects red start or yellow tip of the same art.
        for b in boxes:
            colours=[native.convert('RGB').getpixel((x,y)) for y in range(b[1],b[3]) for x in range(b[0],b[2]) if mask.getpixel((x,y))]
            yellow=sum(c[1] for c in colours)>80*len(colours)
            crop=(mother.width*2//3,mother.height*2//3,mother.width,mother.height) if yellow else (0,0,mother.width//3,mother.height//3)
            out.alpha_composite(fitted(mother.crop(crop),b),(b[0]*6,b[1]*6))
        record.update(source='scythe-arc',policy='native_spark_components',components=[list(b) for b in boxes])
    low,high=out.getchannel('A').getextrema()
    if low!=0 or high==0:
        raise ValueError(f'compiled player9 entry {index} has no transparency or ink')
    record.update(alpha_extrema=[low,high],size=list(out.size))
    return out,record


def build(original, recipe_path, font_path, out, masters_root=None):
    recipe=json.loads(recipe_path.read_text())
    if set(recipe)!={'schema_version','native_source','masters','font_sha256'} or recipe['schema_version']!=1:
        raise ValueError('invalid player9 recipe')
    report,pictures=audit(original)
    if recipe['native_source']!=source_identity(report):
        raise ValueError('fixed native source identity changed')
    if not isinstance(recipe['masters'],dict) or set(recipe['masters'])!=MASTER_KEYS:
        raise ValueError('incomplete master set')
    if recipe['font_sha256']!=FONT_SHA or sha(font_path)!=FONT_SHA:
        raise ValueError('font sha256 mismatch')
    masters_root=masters_root or recipe_path.parent
    masters={key:checked_source(masters_root,row) for key,row in recipe['masters'].items()}
    prepared=[]; assets=[]; records=[]
    for resource in [6,5]:
        source=next(x for x in report['resources'] if x['resource']==resource)
        files=[]; descriptions=[]; glyph_sources=[]
        for index in RANGES[resource]:
            row=next(x for x in source['frames'] if x['index']==index)
            native=next(pic for res,fr,pic in pictures if res==resource and fr['index']==index)
            if resource==6:
                picture,record=effect(index,native,row,masters)
            else:
                char=str(index-94) if index<=103 else {116:'M',117:'I',118:'S'}[index]
                picture=glyph(char,(row['width']*6,row['height']*6),font_path)
                glyph_name=f'player9-noto-result-{char}-original-r1.png'
                glyph_sources.append(glyph_name); prepared.append((glyph_name,picture))
                record={'index':index,'font':FONT_SHA,'char':char,'policy':'direct_noto_raster','size':list(picture.size)}
            name=f'player9-fdother{resource:03d}-{index:03d}-hd6-r1.png'
            files.append(name); prepared.append((name,picture)); records.append({'resource':resource,**record})
            descriptions.append({k:row[k] for k in ['index','width','height','pixels_sha256']})
        source_files=[row['file'] for row in recipe['masters'].values()] if resource==6 else glyph_sources
        asset=dict(asset_id=f'player9-map-fdother{resource:03d}-hd6',role='map_overlay_hd_set',status='runtime_candidate',
                   files=files,width=1920,height=1200,frame_count=len(files),frame_sha256=[],
                   consumer_contract='native_lmi1_rgba_hd6_v1',alpha_contract='coverage',cycle_policy='native_lmi1_entries',
                   source_archive='FDOTHER.DAT',source_resource=resource,source_metadata_sha256=source['metadata_sha256'],
                   native_lmi_frames=descriptions,layout_policy='native_lmi1_rect_hd6',source_files=source_files,source_sha256=[],
                   source_refs=['docs/knowledge-base/61-modern-theme-runtime-spec.md','docs/data/fd2-modern-hd-battlefield-20261019.json','tools/build_fd2_player9_map_hd.py'])
        assets.append(asset)
    if out.exists():
        raise ValueError('candidate output already exists')
    out.mkdir(parents=True)
    for name,im in prepared: im.save(out/name)
    for row in recipe['masters'].values(): (out/row['file']).write_bytes((masters_root/row['file']).read_bytes())
    for asset in assets:
        asset['frame_sha256']=[sha(out/f) for f in asset['files']]
        asset['source_sha256']=[sha(out/f) for f in asset['source_files']]
        (out/f'asset-{asset["source_resource"]}.json').write_text(json.dumps(asset,ensure_ascii=False,indent=2)+'\n')
    (out/'compiled.json').write_text(json.dumps({'schema_version':1,'recipe_sha256':sha(recipe_path),'tool_sha256':sha(Path(__file__)),
        'font_sha256':FONT_SHA,'entries':records,'art_approximation':'Shared handpainted masters, native rise/crop/dissolve/particle boxes; silhouette and alpha retained. Not pixel parity or final human approval.'},ensure_ascii=False,indent=2)+'\n')
    sheet=Image.new('RGB',(1200,640),(40,42,48)); d=ImageDraw.Draw(sheet)
    frames=[(a,index,name) for a in assets for index,name in zip(RANGES[a['source_resource']],a['files'])]
    for i,(asset,index,name) in enumerate(frames):
        im=Image.open(out/name); im.thumbnail((104,112)); x=(i%10)*120;y=(i//10)*160
        bg=Image.new('RGBA',(112,120),(242,242,242,255));bg.alpha_composite(im,((112-im.width)//2,(120-im.height)//2));sheet.paste(bg,(x+4,y+22));d.text((x+4,y+4),f'{asset["source_resource"]}:{index}',fill='white')
    sheet.save(out/'review.png')
    return assets


if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--original-pack',type=Path,required=True);p.add_argument('--recipe',type=Path,required=True)
    p.add_argument('--font',type=Path,required=True);p.add_argument('--out',type=Path,required=True)
    p.add_argument('--masters-root',type=Path,help='母稿唯讀目錄；預設與配方同目錄')
    args=p.parse_args(); assets=build(args.original_pack,args.recipe,args.font,args.out,args.masters_root)
    print(json.dumps({'status':'CANDIDATE','resources':len(assets),'frames':sum(a['frame_count'] for a in assets),'out':str(args.out)}))
