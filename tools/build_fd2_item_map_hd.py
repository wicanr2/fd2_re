#!/usr/bin/env python3
"""在Pillow Docker編製八張新物品HD光效；原版只讀，不匯出原像素。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil

from PIL import Image, ImageChops

ROOT = Path(__file__).resolve().parents[1]
BASE_SHA = '495dbae95c59dbfe25135684cfcb9ab1872cb4ba2cab11eab8bc6eb1b5913b17'
MASTER_SHA = '0d90cc18c6de7750d1092d55be1356ead94726110fce94f28bab69d3eeb38e1c'
MASTER_NAME = 'item-map-white-flash-master-oct25-r1.png'


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_json(path, value):
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n')


def components(mask):
    """Use the read-only source topology only while compiling new artwork."""
    w, h = mask.size
    remaining = {i for i, v in enumerate(mask.getdata()) if v}
    result = []
    while remaining:
        first = min(remaining)
        remaining.remove(first)
        queue, points = [first], []
        while queue:
            i = queue.pop()
            x, y = i % w, i // w
            points.append((x, y))
            for yy in range(max(0, y-1), min(h, y+2)):
                for xx in range(max(0, x-1), min(w, x+2)):
                    j = yy*w+xx
                    if j in remaining:
                        remaining.remove(j)
                        queue.append(j)
        xs, ys = zip(*points)
        box = min(xs), min(ys), max(xs)+1, max(ys)+1
        local = Image.new('L', (box[2]-box[0], box[3]-box[1]))
        for x, y in points:
            local.putpixel((x-box[0], y-box[1]), 255)
        result.append((box, local))
    return result


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['catalog', 'source', 'original-pack', 'master', 'out']:
        p.add_argument('--'+name, type=Path, required=True)
    a = p.parse_args()
    for name in ["catalog", "source", "original_pack", "master", "out"]:
        setattr(a, name, getattr(a, name).resolve())
    if not Path('/.dockerenv').exists() or a.out.exists() or not a.out.parent.is_dir():
        raise SystemExit('只在Docker寫入既有工作根的新候選；不覆寫')
    if sha(a.catalog) != BASE_SHA or sha(a.master) != MASTER_SHA:
        raise SystemExit('固定清冊或手繪母稿不同')
    catalog = json.loads(a.catalog.read_text())
    if len(catalog['assets']) != 768:
        raise SystemExit('需要已驗證的768項基底')
    inventory = json.loads((ROOT/'docs/data/fd2-battle-hd-source-inventory-20261019.json').read_text())
    if sha(a.original_pack/'manifest.json') != inventory['pack_manifest_sha256']:
        raise SystemExit('原包指紋不同')
    manifest = json.loads((a.original_pack/'manifest.json').read_text())
    originals = {row['path']: row for row in manifest['assets']}
    template = next(row for row in catalog['assets'] if row['role'] == 'map_overlay_hd_set'
                    and row['source_resource'] == 6 and row['frame_count'] == 7)
    metadata = a.original_pack/'effects/fdother_006_lmi1_opaque/bank.json'
    if sha(metadata) != template['source_metadata_sha256']:
        raise SystemExit('FDOTHER6 metadata不同')
    with Image.open(a.original_pack/'effects/fdother_006_lmi1_opaque/entry_057/frame.png') as anchor:
        if hashlib.sha256(bytes(anchor.getdata())).hexdigest() != template['native_lmi_frames'][0]['pixels_sha256']:
            raise SystemExit('PNG索引與正式原生LMI資料不同')
    a.out.mkdir()
    names = set()
    for row in catalog['assets']:
        names.update(row.get('files', []))
        names.update(row.get('source_files', []))
        names.update(row[k] for k in ['file', 'master_file'] if k in row)
    for name in sorted(names):
        if Path(name).name != name or not (a.source/name).is_file():
            raise SystemExit('舊白名單不完整或路徑不安全')
        (a.out/name).symlink_to(os.path.relpath(a.source/name, a.out))
    shutil.copyfile(a.master, a.out/MASTER_NAME)
    with Image.open(a.master) as im:
        if im.mode != 'RGBA' or im.size != (1268, 1240) or im.getchannel('A').getextrema() != (0, 254):
            raise SystemExit('母稿形狀或原始透明度不同')
        master = im.crop(im.getchannel('A').getbbox())
    files, descriptors, recipe = [], [], []
    for index in range(49, 57):
        path = f'effects/fdother_006_lmi1_opaque/entry_{index:03}/frame.png'
        source = a.original_pack/path
        row = originals[path]
        if (row.get('source_file') != 'FDOTHER.DAT' or row.get('source_resource') != 6
                or row.get('source_frame') != index or sha(source) != row['sha256']):
            raise SystemExit('原版來源定位或SHA不同')
        with Image.open(source) as native:
            if native.mode != 'P' or not 1 <= native.width <= 128 or not 1 <= native.height <= 128:
                raise SystemExit('原版LMI形狀不同')
            raw = bytes(native.getdata())
            pixel_sha = hashlib.sha256(raw).hexdigest()
            mask = native.point(lambda v: 255 if v else 0, 'L')
            out = Image.new('RGBA', (native.width*6, native.height*6))
            parts = components(mask)
            if not parts:
                raise SystemExit('已知可見物品光效變成空白')
            for box, local in parts:
                size = local.width*6, local.height*6
                painted = master.resize(size, Image.Resampling.LANCZOS)
                coverage = local.resize(size, Image.Resampling.LANCZOS)
                painted.putalpha(ImageChops.multiply(painted.getchannel('A'), coverage))
                out.alpha_composite(painted, (box[0]*6, box[1]*6))
            if out.getchannel('A').getextrema()[0] != 0 or out.getchannel('A').getextrema()[1] == 0:
                raise SystemExit('新光效coverage無效')
            name = f'item-map-fdother006-{index:03}-hd6-r1.png'
            out.save(a.out/name)
            files.append(name)
            descriptors.append(dict(index=index, width=native.width, height=native.height, pixels_sha256=pixel_sha))
            # Persist summaries and hashes only, never the original pixel array.
            recipe.append(dict(index=index, source_path=path, source_sha256=sha(source),
                               source_pixels_sha256=pixel_sha, native_size=list(native.size),
                               visible_pixels=sum(mask.histogram()[1:]), ink_box=mask.getbbox(),
                               component_count=len(parts), compiled_sha256=sha(a.out/name),
                               alpha_extrema=out.getchannel('A').getextrema()))
    asset = dict(template)
    asset.update(asset_id='item-map-fdother006-049-056-hd6', files=files, frame_count=8,
                 frame_sha256=[sha(a.out/name) for name in files], native_lmi_frames=descriptors,
                 source_files=[MASTER_NAME], source_sha256=[MASTER_SHA],
                 source_refs=['docs/knowledge-base/61-modern-theme-runtime-spec.md',
                              'docs/data/ida/fd2_ai_item_damage_1cd17_owner.txt',
                              'tools/build_fd2_item_map_hd.py'])
    catalog['assets'].append(asset)
    catalog['private_root'] = str(a.out.relative_to(ROOT))
    write_json(a.out/'catalog.json', catalog)
    write_json(a.out/'recipe.json', dict(status='DRAFT-COMPILED-ITEM-MAP-HD',
               tool_sha256=sha(Path(__file__)), base_catalog_sha256=BASE_SHA,
               original_manifest_sha256=sha(a.original_pack/'manifest.json'), master_sha256=MASTER_SHA,
               frames=recipe, new_entries=[asset], production_changed=False,
               art_policy='new painted master per connected component; native placement/dissolve topology at6x Lanczos; preserve authored alpha max254'))
    print(json.dumps(dict(status='DRAFT-COMPILED-ITEM-MAP-HD', assets=len(catalog['assets']), new_frames=8)))


if __name__ == '__main__':
    main()
