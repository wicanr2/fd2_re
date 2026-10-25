#!/usr/bin/env python3
"""將透明物件母稿依原生bbox接入既有稀疏地形，含事件更新後圖源。"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import tempfile

from PIL import Image, ImageChops, ImageFilter
from build_fd2_hd_ground import make_tile
from validate_modern_theme_catalog import validate as validate_catalog
from verify_fd2_terrain_sources import check_binding, sha, verify

ROOT = Path(__file__).resolve().parents[1]
FRAMES = {248, 249, 250, 251, 274, 275}


def eligible_frames(pack, maps, mp):
    """沿現有0x12263 writer的控制遮罩；不從相鄰圖片猜測開箱。"""
    bank = json.loads(check_binding(pack, mp['bank']).read_text())
    data = json.loads(check_binding(maps, mp['map_data']).read_text())
    opened = {(word & 1023) + 1 for word in data['tiles']
              if bank['controls'][(word & 1023) * 4] & 0x60 == 0x20}
    return set(mp['selected_terrain_frames']), opened


def validate_sources(pack, maps, manifest, recipe):
    if (recipe.get('schema_version'), recipe.get('kind'), recipe.get('size')) != (1, 'fd2_native_props_hd_v1', 144):
        raise ValueError('物件配方契約不同')
    if (recipe.get('grass_indices'), recipe.get('shadow_indices'), recipe.get('shadow_level'),
            recipe.get('shadow_blur_hd_pixels')) != ([68,69,70,71], [137,138,139,140,141,142], 153, 2):
        raise ValueError('物件草地與陰影契約不同')
    if len(recipe['groups']) != 6 or {tuple(g['anchor']) for g in recipe['groups']} != {(0, f) for f in FRAMES}:
        raise ValueError('物件來源身分不完整')
    if set(recipe['masters']) != {'chests', 'rocks'}:
        raise ValueError('物件母稿身分不同')
    eligible = {mp['map']: eligible_frames(pack, maps, mp) for mp in manifest['maps']}
    seen, result = set(), []
    background = set(recipe['grass_indices'] + recipe['shadow_indices'])
    for group in recipe['groups']:
        if group.get('master') != ('chests' if group['anchor'][1] < 274 else 'rocks'):
            raise ValueError('物件與母稿配對不同')
        if Path(group['file']).name != group['file'] or not group['file'].endswith('.png'):
            raise ValueError('物件檔名越界')
        targets, bindings, anchor = [], [], None
        for mp in manifest['maps']:
            static, opened = eligible[mp['map']]
            for fr in mp['frames']:
                if fr['native_content_sha256'] != group['native_content_sha256'] or fr['frame'] not in static | opened:
                    continue
                identity = [mp['map'], fr['frame']]
                if tuple(identity) in seen:
                    raise ValueError('物件槽位重複')
                seen.add(tuple(identity))
                paths = [check_binding(pack, e) for e in fr['input_layers']]
                raw, opaque, remap = [Image.open(p).tobytes() for p in paths]
                if len(raw) != 576 or len(opaque) != 576 or len(remap) != 576 or not all(opaque) or any(remap):
                    raise ValueError('物件原生遮罩不符')
                content = raw + bytes(bool(v) for v in opaque) + bytes(bool(v) for v in remap)
                if hashlib.sha256(content).hexdigest() != group['native_content_sha256']:
                    raise ValueError('物件原生內容不符')
                targets.append(identity)
                bindings.append({'map': mp['map'], 'frame': fr['frame'], 'input_layers': fr['input_layers'],
                                 'consumer': 'initial-map' if fr['frame'] in static else 'event-grid-word-plus-one'})
                if identity == group['anchor']:
                    anchor = raw
        if targets != group['targets'] or anchor is None:
            raise ValueError('物件實際consumer不同')
        mask = Image.frombytes('L', (24,24), bytes(0 if v in background else 255 for v in anchor))
        if any(type(v) is not int for v in group['native_bbox']) or list(mask.getbbox() or ()) != group['native_bbox']:
            raise ValueError('物件原生bbox不同')
        result.append({'group': group, 'pixels': anchor, 'bindings': bindings})
    if len(seen) != 10 or {m for m,f in seen} != {0,14}:
        raise ValueError('物件實際槽位不完整')
    return result


def painted_crop(master, group):
    crop = group['painted_crop']
    if (len(crop) != 4 or any(type(v) is not int for v in crop) or
            not (0 <= crop[0] < crop[2] <= master.width and 0 <= crop[1] < crop[3] <= master.height)):
        raise ValueError('物件裁切越界')
    frame = group['anchor'][1]
    column = (frame - 248) % 2 if frame < 274 else frame - 274
    row = (frame - 248) // 2 if frame < 274 else 0
    rows = 2 if frame < 274 else 1
    cell = (column * master.width // 2, row * master.height // rows,
            (column+1) * master.width // 2, (row+1) * master.height // rows)
    if not (cell[0] <= crop[0] < crop[2] <= cell[2] and cell[1] <= crop[1] < crop[3] <= cell[3]):
        raise ValueError('物件裁切跨圖組格子')
    high = master.getchannel('A').point(lambda v: 255 if v >= 250 else 0)
    box = high.crop(cell).getbbox()
    if box is None:
        raise ValueError('物件母稿缺主體')
    box = (box[0]+cell[0], box[1]+cell[1], box[2]+cell[0], box[3]+cell[1])
    if not (crop[0] <= box[0] < box[2] <= crop[2] and crop[1] <= box[1] < box[3] <= crop[3]):
        raise ValueError('物件高覆蓋主體裁切')
    return master.crop(crop)


def compose(raw, grass, master, group, recipe):
    shadow = Image.frombytes('L', (24,24), bytes(255 if v in recipe['shadow_indices'] else 0 for v in raw))
    smooth = shadow.resize((144,144), Image.Resampling.BICUBIC).filter(ImageFilter.GaussianBlur(2))
    level = smooth.point(lambda v: 255-round(v*102/255))
    for i in range(144):
        for x,y in [(i,0),(i,143),(0,i),(143,i)]:
            level.putpixel((x,y), 153 if raw[y//6*24+x//6] in recipe['shadow_indices'] else 255)
    background = Image.merge('RGBA', (*[ImageChops.multiply(b, level) for b in grass.split()[:3]], Image.new('L',(144,144),255)))
    picture = painted_crop(master, group)
    x,y,right,bottom = [v*6 for v in group['native_bbox']]
    picture = picture.resize((right-x,bottom-y), Image.Resampling.LANCZOS)
    background.alpha_composite(picture, (x,y))
    if background.getextrema()[3] != (255,255):
        raise ValueError('物件輸出alpha不同於原生opaque')
    return background


def build(pack, maps, sources, base_catalog, base_pack, recipe_path, master_root, out):
    if os.path.lexists(out) or not out.parent.is_dir():
        raise ValueError('輸出須為既有父目錄下的新目錄')
    recipe = json.loads(recipe_path.read_text())
    if sha(base_catalog) != recipe['base_catalog_sha256'] or sha(sources/'manifest.json') != recipe['source_manifest_sha256']:
        raise ValueError('物件基底來源綁定不同')
    source_report = verify(pack, maps, sources)
    checked = validate_sources(pack, maps, json.loads((sources/'manifest.json').read_text()), recipe)
    masters = {}
    for key, entry in recipe['masters'].items():
        p = master_root/entry['file']
        if Path(entry['file']).name != entry['file'] or sha(p) != entry['sha256']:
            raise ValueError('物件母稿來源不同')
        im = Image.open(p)
        if im.mode != 'RGBA' or list(im.size) != entry['size'] or im.getextrema()[3][0] != 0 or im.getextrema()[3][1] < 250:
            raise ValueError('物件母稿尺寸或透明契約不同')
        masters[key] = im
    grass_path = base_pack/recipe['grass_master_file']
    if sha(grass_path) != recipe['grass_master_sha256']:
        raise ValueError('既有草地母稿不同')
    grass, calibration = make_tile(grass_path, 144, 8)
    catalog = json.loads(base_catalog.read_text())
    names, hashes = set(), {}
    for a in catalog['assets']:
        for key in ['file','master_file']:
            if key in a: names.add(a[key])
        if 'file' in a:hashes[a['file']]=a['sha256']
        for key,hkey in [('files','frame_sha256'),('source_files','source_sha256')]:
            if len(a.get(key,[]))!=len(a.get(hkey,[])):
                raise ValueError('物件基底來源綁定缺少')
            for name,h in zip(a.get(key,[]),a.get(hkey,[])):
                if name in hashes and hashes[name]!=h:raise ValueError('物件基底來源衝突')
                names.add(name);hashes[name]=h
    fresh = {g['file'] for g in recipe['groups']} | {m['file'] for m in recipe['masters'].values()}
    old = {(a['map_id'], f) for a in catalog['assets'] if a['role']=='native_terrain_hd_set' for f in a['frame_ids']}
    if len(fresh) != 8 or names & fresh or old & {tuple(t) for g in recipe['groups'] for t in g['targets']}:
        raise ValueError('物件圖源或槽位身分衝突')
    temp = Path(tempfile.mkdtemp(prefix='.props-', dir=out.parent))
    try:
        dest = temp/'pack'; dest.mkdir()
        for name in names:
            if (Path(name).name != name or not (base_pack/name).is_file() or
                    name in hashes and sha(base_pack/name)!=hashes[name]):
                raise ValueError('物件基底缺圖源')
            (dest/name).symlink_to((base_pack/name).resolve())
        additions, frames, by_map = [], [], {}
        for key, entry in recipe['masters'].items():
            shutil.copyfile(master_root/entry['file'], dest/entry['file'])
            additions.append({'asset_id':recipe['asset_id']+'.'+key+'.master','role':'terrain_hd_master','status':'concept',
                              'file':entry['file'],'sha256':entry['sha256'],'width':entry['size'][0],'height':entry['size'][1],
                              'source_refs':recipe['source_refs']})
        preview = Image.new('RGBA',(3*144,2*144))
        for i,item in enumerate(checked):
            g = item['group']; im = compose(item['pixels'], grass, masters[g['master']], g, recipe)
            im.save(dest/g['file']); preview.paste(im,(i%3*144,i//3*144))
            frames.append({'anchor':g['anchor'],'file':g['file'],'sha256':sha(dest/g['file']),
                           'native_content_sha256':g['native_content_sha256'],'targets':item['bindings']})
            for map_id,frame in g['targets']:by_map.setdefault(map_id,[]).append((frame,g['file'],sha(dest/g['file'])))
        refs=['asset:'+recipe['asset_id']+'.'+key+'.master' for key in recipe['masters']]+['asset:modern.terrain.grass_olive.master']+recipe['source_refs']
        for map_id, entries in sorted(by_map.items()):
            entries.sort();additions.append({'asset_id':recipe['asset_id']+f'.map{map_id:02d}.native_hd','role':'native_terrain_hd_set',
                'status':'runtime_candidate','consumer_contract':'fdshap_native_hd_sparse_v1','map_id':map_id,'width':144,'height':144,
                'frame_count':len(entries),'frame_ids':[e[0] for e in entries],'files':[e[1] for e in entries],
                'frame_sha256':[e[2] for e in entries],'semantics':['terrain']*len(entries),
                'alpha_contract':'native_mask_nearest_6','cycle_policy':'native_selector','source_refs':refs})
        candidate=copy.deepcopy(catalog);candidate['assets'].extend(additions)
        (temp/'catalog.json').write_text(json.dumps(candidate,ensure_ascii=False,indent=2)+'\n')
        validate_catalog(False,temp/'catalog.json');preview.save(temp/'preview.png')
        proof={'status':'PASS-PROPS-CANDIDATE','runtime_integrated':False,'base_catalog_sha256':sha(base_catalog),
               'recipe_sha256':sha(recipe_path),'source_manifest_sha256':sha(sources/'manifest.json'),'source_verification':source_report,
               'source_groups':6,'target_slots':10,'maps':[0,14],'frames':frames,
               'masters':[{'file':e['file'],'sha256':e['sha256']} for e in recipe['masters'].values()],
               'catalog_assets':len(candidate['assets']),'base_assets_unchanged':len(catalog['assets']),
               'grass_master':{'file':recipe['grass_master_file'],'sha256':sha(grass_path)},'grass_edge_calibration':calibration,
               'scope':'透明新母稿依原bbox合成新草地；開箱依既有writer。不改規則，不是原版RGB或正常取寶證據。'}
        (temp/'manifest.json').write_text(json.dumps(proof,ensure_ascii=False,indent=2)+'\n');temp.rename(out)
        return proof
    except BaseException:
        shutil.rmtree(temp,ignore_errors=True)
        raise


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ['original-pack','maps','sources','base-catalog','base-pack','recipe','master-root','out']:
        p.add_argument('--'+n,type=Path,required=True)
    a=p.parse_args();proof=build(a.original_pack,a.maps,a.sources,a.base_catalog,a.base_pack,a.recipe,a.master_root,a.out)
    print(json.dumps({k:proof[k] for k in ['status','source_groups','target_slots','catalog_assets']}))


if __name__=='__main__':main()
