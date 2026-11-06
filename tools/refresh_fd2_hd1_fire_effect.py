#!/usr/bin/env python3
"""依已選B配方重生熾炎刀六拍；只產生隔離候選，不登錄正式清冊。"""
import argparse
import copy
import json
import os
from pathlib import Path
import shutil

from PIL import Image, ImageDraw

from build_fd2_character_battle_hd import DESCRIPTOR, load_native
from build_fd2_gaia_battle_hd import digest, names
from validate_modern_theme_catalog import ROOT, validate


def bounds(im, threshold=8):
    return im.getchannel('A').point(lambda v: 255 if v >= threshold else 0).getbbox()


def place(stage, im, source, target, floor=True):
    scale = ((target[2]-target[0])/(source[2]-source[0]) if floor
             else (target[3]-target[1])/(source[3]-source[1]))
    fitted = im.resize((max(1, round(im.width*scale)), max(1, round(im.height*scale))), Image.Resampling.LANCZOS)
    sx, sy = fitted.width/im.width, fitted.height/im.height
    at = (round((target[0]+target[2])/2-(source[0]+source[2])*sx/2),
          round(target[3]-source[3]*sy))
    stage.alpha_composite(fitted, at)
    visible = fitted.getchannel('A').crop((-at[0], -at[1], 1920-at[0], 1200-at[1]))
    clipped = sum(v >= 192 for v in fitted.getchannel('A').getdata())-sum(v >= 192 for v in visible.getdata())
    return {'source_anchor': source, 'target_anchor': target, 'fit_size': list(fitted.size),
            'origin': list(at), 'viewport_clipped_high_coverage_pixels': clipped}


def tip_runs(mask):
    result, pending = [], None
    for x in range(mask.width):
        ys = [y for y in range(mask.height) if mask.getpixel((x, y))]
        if ys:
            pending = ([x, min(ys), x+1, max(ys)+1] if pending is None else
                       [pending[0], min(pending[1], min(ys)), x+1, max(pending[3], max(ys)+1)])
        elif pending is not None:
            result.append(pending); pending = None
    if pending is not None:
        result.append(pending)
    return result


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for key in ['pack', 'inventory', 'recipe', 'masters', 'base-catalog', 'out']:
        p.add_argument('--'+key, required=True, type=Path)
    a = p.parse_args()
    out = a.out.resolve()
    if out.exists() or not out.is_relative_to(ROOT) or out.parent.stat().st_uid != os.getuid():
        raise SystemExit('使用本人擁有的全新候選目錄')
    recipe = json.loads(a.recipe.read_text())
    if (recipe['schema_version'] != 1 or recipe['choice'] != 'B' or recipe['resource'] != 104
            or digest(a.base_catalog) != recipe['base_catalog_sha256']
            or digest(a.inventory) != recipe['inventory_sha256']):
        raise SystemExit('配方、盤點或基底清冊不符')
    base = validate(False, a.base_catalog)
    old = next(x for x in base['assets'] if x.get('role') == 'battle_animation_hd_set' and x['source_resource'] == 104)
    inventory = json.loads(a.inventory.read_text())
    if (inventory['status'] != 'PASS-SOURCE-INVENTORY' or not inventory['archive_reconstruction']['matches_pinned_original']
            or digest(a.pack/'manifest.json') != inventory['pack_manifest_sha256']):
        raise SystemExit('原始固定包不符')
    row, native = load_native(a.pack, inventory, 104)
    if (len(native) != 18 or old['native_frames'] != [{k: f[k] for k in DESCRIPTOR} for f in row['frames']]
            or old['source_metadata_sha256'] != row['metadata_sha256'] or native[17].getchannel('A').getbbox() is not None):
        raise SystemExit('原始描述子或透明末拍不符')
    private = ROOT/base['private_root']
    old_names = names(base)
    if any(not (private/n).is_file() for n in old_names):
        raise SystemExit('基底素材缺件')
    art = {}
    for key, item in recipe['masters'].items():
        path = a.masters/item['file']
        if path.stat().st_uid != os.getuid() or digest(path) != item['sha256']:
            raise SystemExit('母稿擁有權或SHA不符')
        im = Image.open(path)
        if im.mode != 'RGBA' or im.getchannel('A').getextrema() != (0, 255):
            raise SystemExit('母稿透明端點不符')
        art[key] = im.copy()
    if set(art) != {'column', 'ring', 'spikes', 'tip'}:
        raise SystemExit('四母稿集合不符')
    for name, sha in zip(old['files'], old['frame_sha256']):
        if digest(private/name) != sha:
            raise SystemExit('既有動畫圖源SHA不同')
    prepared = []
    for index in range(11, 17):
        frame = row['frames'][index]
        stage = Image.new('RGBA', (1920, 1200))
        placements = []
        if index <= 13:
            width = frame['width']*6-4
            center, floor = (frame['x']+frame['width']/2)*6, (frame['y']+frame['height'])*6-1
            placements.append(place(stage, art['ring'], recipe['ring_anchor'], [center-width/2, floor-1, center+width/2, floor]))
        elif index == 14:
            placements.append(place(stage, art['column'], recipe['column_anchor'], recipe['column_target']))
        elif index == 15:
            placements.append(place(stage, art['spikes'], list(bounds(art['spikes'])), [38, 0, 766, 478]))
        else:
            for run in tip_runs(native[16].getchannel('A')):
                target = [(frame['x']+run[0])*6+1, 0, (frame['x']+run[2])*6-1, run[3]*6-2]
                placements.append(place(stage, art['tip'], list(bounds(art['tip'])), target, floor=False))
        if any(v['viewport_clipped_high_coverage_pixels'] for v in placements):
            raise SystemExit('火焰高覆蓋像素超出視窗，停止審查')
        box = stage.getchannel('A').getbbox()
        if box is None:
            raise SystemExit('可見效果缺失')
        box = (max(0, box[0]-1), max(0, box[1]-1), min(1920, box[2]+1), min(1200, box[3]+1))
        im = stage.crop(box)
        prepared.append((index, im, {'x': box[0], 'y': box[1], 'width': box[2]-box[0], 'height': box[3]-box[1]}, placements))
    out.mkdir(); pack = out/'pack'; pack.mkdir()
    for name in sorted(old_names):
        (pack/name).symlink_to(os.path.relpath(private/name, pack))
    shutil.copyfile(a.base_catalog, out/'base-catalog.json')
    candidate = copy.deepcopy(base)
    candidate['private_root'] = str(pack.relative_to(ROOT))
    effect = next(x for x in candidate['assets'] if x.get('role') == 'battle_animation_hd_set' and x['source_resource'] == 104)
    effect.update(asset_id='modern.figani.104.hd6.v2', consumer_contract='native_figani_rgba_hd6_v2',
                  layout_policy='explicit_hd_frame_rects_v2',
                  hd_frames=[{'x': f['x']*6, 'y': f['y']*6, 'width': f['width']*6, 'height': f['height']*6} for f in old['native_frames']])
    replaced = {'column': 'column', 'ring': 'disc', 'spikes': 'spikes', 'tip': 'tip'}
    for key, item in recipe['masters'].items():
        name = item['file']
        if (pack/name).exists():
            raise SystemExit('新母稿名稱與基底衝突')
        shutil.copyfile(a.masters/name, pack/name)
        before = f'figure-034-{replaced[key]}-battle-hd6-master-v1.png'
        at = effect['source_files'].index(before)
        effect['source_files'][at] = name; effect['source_sha256'][at] = item['sha256']
    records = []
    sheet = Image.new('RGB', (1280, 1200), (12, 24, 38)); draw = ImageDraw.Draw(sheet)
    for index, im, rect, placements in prepared:
        name = f'figani-104-hd6-v2-f{index:02d}.png'
        if (pack/name).exists():
            raise SystemExit('新成品名稱與基底衝突')
        im.save(pack/name)
        effect['files'][index] = name; effect['frame_sha256'][index] = digest(pack/name); effect['hd_frames'][index] = rect
        records.append({'frame': index, 'file': name, 'sha256': digest(pack/name), 'hd_rect': rect, 'placements': placements})
        x, y = (index-11)%2*640, (index-11)//2*400
        for dx, image, at in [(0, Image.open(private/old['files'][index]), (old['native_frames'][index]['x']*6, old['native_frames'][index]['y']*6)),
                              (320, im, (rect['x'], rect['y']))]:
            stage = Image.new('RGBA', (1920, 1200)); stage.alpha_composite(image, at)
            thumb = stage.resize((320, 200), Image.Resampling.LANCZOS)
            sheet.paste(thumb, (x+dx, y+36), thumb)
        draw.text((x+8, y+10), f'104 / {index:02d} Current / B', fill='white')
    unaffected = [x for x in candidate['assets'] if x['asset_id'] != effect['asset_id']]
    if unaffected != [x for x in base['assets'] if x['asset_id'] != old['asset_id']]:
        raise SystemExit('未選清冊項被更動')
    for i in [*range(11), 17]:
        if (effect['files'][i], effect['frame_sha256'][i]) != (old['files'][i], old['frame_sha256'][i]):
            raise SystemExit('人物或末拍被更動')
    path = out/'catalog.json'; path.write_text(json.dumps(candidate, ensure_ascii=False, indent=2)+'\n')
    validate(True, path); sheet.save(out/'review.png')
    report = {'status': 'PASS-FIRE-B-CANDIDATE', 'runtime_registered': False, 'scope': '六拍火焰與嵌入火星；無獨立粒子計時器或正常玩家聲明',
              'base_catalog_sha256': digest(a.base_catalog), 'catalog_sha256': digest(path), 'recipe_sha256': digest(a.recipe),
              'compiler_sha256': digest(Path(__file__)), 'inventory_sha256': digest(a.inventory), 'masters': recipe['masters'],
              'unchanged_assets': len(unaffected), 'unchanged_actor_and_blank_frames': [*range(11), 17], 'frames': records}
    (out/'compiled.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({k: report[k] for k in ['status', 'unchanged_assets']}, ensure_ascii=False))


if __name__ == '__main__':
    main()
