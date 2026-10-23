#!/usr/bin/env python3
"""編製回復13～16的固定LMI圖源與完整候選，不改正式素材。"""
import argparse
import hashlib
import json
import os
import shutil
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont
from audit_fd2_player9_map_source import audit
from build_fd2_player9_map_hd import FONT_SHA
from validate_modern_theme_catalog import ROOT, validate

RANGES = {6: list(range(57, 64)), 5: list(range(105, 115))}


def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for key in ['original-pack', 'master', 'font', 'base-catalog', 'out']:
        p.add_argument('--'+key, type=Path, required=True)
    a = p.parse_args()
    if not Path('/.dockerenv').exists() or a.out.exists() or a.out.parent.stat().st_uid != os.getuid():
        raise ValueError('Docker內使用本人擁有的全新候選目錄')
    state = json.loads((ROOT/'docs/data/fd2-modern-hd-battlefield-20261019.json').read_text())['heal13_16_map_hd_followup_20261023']
    if sha(a.master) != state['master']['sha256'] or sha(a.font) != FONT_SHA:
        raise ValueError('原稿或字型不同')
    report, pictures = audit(a.original_pack, RANGES)
    pinned = json.loads((ROOT/state['source_audit']['file']).read_text())
    if report != pinned:
        # JSON stores bounding boxes as arrays.
        if json.loads(json.dumps(report)) != pinned:
            raise ValueError('固定原圖來源不同')
    catalog = validate(True, a.base_catalog)
    for asset in catalog['assets']:
        if asset['role'] == 'map_overlay_hd_set' and any(x['index'] in RANGES[asset['source_resource']] for x in asset['native_lmi_frames']):
            raise ValueError('來源已登錄')
    with Image.open(a.master) as im:
        if im.mode != 'RGBA' or im.size != (1254, 1254) or im.getchannel('A').getextrema() != (0, 254):
            raise ValueError('原稿RGBA或alpha不同')
        ring = im.crop(state['master']['compile_crop'])
    a.out.mkdir()
    old_names = set()
    for asset in catalog['assets']:
        old_names.update(asset.get('files', [])); old_names.update(asset.get('source_files', []))
        old_names.update(asset[k] for k in ['file', 'master_file'] if k in asset)
    old_root = ROOT/catalog['private_root']
    for name in sorted(old_names):
        if Path(name).name != name or not (old_root/name).is_file(): raise ValueError('舊檔缺件')
        (a.out/name).symlink_to(os.path.relpath(old_root/name, a.out))
    master_name = 'heal1316-ring-master-oct23-r1.png'
    shutil.copyfile(a.master, a.out/master_name)
    records = []
    for resource, indices in RANGES.items():
        bank = next(b for b in report['resources'] if b['resource'] == resource)
        files, descriptions, sources = [], [], [master_name] if resource == 6 else []
        for index in indices:
            row = next(f for f in bank['frames'] if f['index'] == index)
            native = next(im for res, f, im in pictures if res == resource and f['index'] == index)
            size = row['width']*6, row['height']*6
            picture = Image.new('RGBA', size)
            if resource == 6:
                box = native.getchannel('A').getbbox()
                if not box: raise ValueError('原環為空')
                art = ring.resize(((box[2]-box[0])*6, (box[3]-box[1])*6), Image.Resampling.LANCZOS)
                # Original first ring is dimmer; preserve source alpha otherwise.
                if index == 57: art.putalpha(art.getchannel('A').point(lambda v: round(v*0.7)))
                picture.alpha_composite(art, (box[0]*6, box[1]*6))
                record = {'index': index, 'native_ink_box': list(box), 'policy': 'shared_ring_crop_fit_native_ink; first_alpha70percent'}
            else:
                char = str(index-105)
                for pixels in range(size[1], 1, -1):
                    font = ImageFont.truetype(str(a.font), pixels, index=3)
                    box = font.getbbox(char, stroke_width=2)
                    if box[2]-box[0] <= size[0]-4 and box[3]-box[1] <= size[1]-4: break
                else: raise ValueError('字框放不下')
                ImageDraw.Draw(picture).text(((size[0]-box[2]+box[0])//2-box[0], (size[1]-box[3]+box[1])//2-box[1]), char, font=font, fill=(188,255,194,255), stroke_width=2, stroke_fill=(0,28,10,255))
                glyph = f'heal1316-noto-{char}-original-r1.png'; picture.save(a.out/glyph); sources.append(glyph)
                record = {'index': index, 'char': char, 'font_sha256': FONT_SHA, 'policy': 'direct_Noto_HD_raster_green'}
            low, high = picture.getchannel('A').getextrema()
            if low != 0 or high == 0: raise ValueError('透明背景或墨跡缺失')
            name = f'heal1316-fdother{resource:03d}-{index:03d}-hd6-r1.png'; picture.save(a.out/name); files.append(name)
            descriptions.append({k: row[k] for k in ['index','width','height','pixels_sha256']}); records.append({'resource':resource,**record,'alpha_extrema':[low,high]})
        catalog['assets'].append(dict(asset_id=f'heal1316-map-fdother{resource:03d}-hd6',role='map_overlay_hd_set',status='runtime_candidate',files=files,width=1920,height=1200,frame_count=len(files),frame_sha256=[sha(a.out/n) for n in files],consumer_contract='native_lmi1_rgba_hd6_v1',alpha_contract='coverage',cycle_policy='native_lmi1_entries',source_archive='FDOTHER.DAT',source_resource=resource,source_metadata_sha256=bank['metadata_sha256'],native_lmi_frames=descriptions,layout_policy='native_lmi1_rect_hd6',source_files=sources,source_sha256=[sha(a.out/n) for n in sources],source_refs=['docs/knowledge-base/61-modern-theme-runtime-spec.md','tools/build_fd2_heal_map_hd.py']))
    catalog['private_root'] = str(a.out.resolve().relative_to(ROOT))
    path = a.out/'catalog.json'; path.write_text(json.dumps(catalog,ensure_ascii=False,indent=2)+'\n'); validate(True,path)
    recipe = dict(status='PASS-HEAL1316-CANDIDATE',source=report,tool_sha256=sha(Path(__file__)),base_catalog_sha256=sha(a.base_catalog),catalog_sha256=sha(path),master_sha256=sha(a.master),font_sha256=FONT_SHA,records=records,old_files_are_links=True,old_file_count=len(old_names),production_changed=False)
    (a.out/'compiled.json').write_text(json.dumps(recipe,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':recipe['status'],'assets':len(catalog['assets']),'new_frames':17}))


if __name__ == '__main__': main()
