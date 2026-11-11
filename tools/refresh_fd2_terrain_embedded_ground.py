#!/usr/bin/env python3
"""以原生草地分區和既有手繪RGB，統一獨立圖集烘入的草地底色。"""
import argparse
from collections import deque
import copy
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageChops, ImageDraw, ImageFilter

from build_fd2_hd_ground import make_tile
from build_fd2_hd_ground_edges import composite_relief
from build_fd2_hd_terrain_reuse import native

RECIPE_REF = 'remake/assets/themes/modern/hd1-terrain-ground-consistency.recipe.json'
PROFILES = [f'hd1-terrain-r78-atlas{i:02d}.recipe.json' for i in (4, 5, 6)]
# Visual review rejects colour segmentation for leaves and moss. These source
# objects need separate painted-alpha evidence; they keep their admitted RGB.
EXCLUDED_CELLS = {
    PROFILES[0]: set(range(18, 26)) | set(range(86, 90)) | {95},
    PROFILES[1]: set(range(20)) | {24, 25} | set(range(66, 70)) | set(range(73, 81)),
    PROFILES[2]: set(range(7)) | {21, 22},
}


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def dump(path, value):
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2)+'\n')


def background_mask(painting, raw, policy):
    """只取接觸圖邊且接近原生草地的綠色，保護灰石與孤立綠色物件。"""
    grass = set(policy['index_groups']['grass'])
    zone = Image.frombytes('L', (24, 24), bytes(255 if v in grass else 0 for v in raw))
    zone = zone.resize((144, 144), Image.Resampling.NEAREST).filter(ImageFilter.MaxFilter(13))
    admitted = set()
    for y in range(144):
        for x in range(144):
            r, g, b, a = painting.getpixel((x, y))
            if a == 255 and zone.getpixel((x, y)) and g*100 >= r*95 and g*100 > b*165 and g-b > 20:
                admitted.add((x, y))
    pending = deque(p for p in admitted if p[0] in (0, 143) or p[1] in (0, 143))
    connected = set(pending)
    while pending:
        x, y = pending.popleft()
        for point in ((x-1, y), (x+1, y), (x, y-1), (x, y+1)):
            if point in admitted and point not in connected:
                connected.add(point)
                pending.append(point)
    mask = Image.new('L', painting.size)
    for point in connected:
        mask.putpixel(point, 255)
    return mask


def prepare(args):
    assert Path('/.dockerenv').exists() and os.getuid() != 0
    assert not args.out.exists() and args.out.parent.stat().st_uid == os.getuid()
    base = json.loads(args.catalog.read_text())
    sources = json.loads((args.sources/'manifest.json').read_text())
    refresh = json.loads((args.repo/'remake/assets/themes/modern/hd1-pixel-terrain-refresh.recipe.json').read_text())
    material = refresh['materials']['grass']
    assert sha(args.pack/material['file']) == material['sha256']
    grass, _ = make_tile(args.pack/material['file'], 144, 8)
    profile_dir = args.repo/'remake/assets/themes/modern'
    profiles = [(name, json.loads((profile_dir/name).read_text())) for name in PROFILES]
    aliases = {}
    for asset in base['assets']:
        if asset['role'] != 'native_terrain_hd_set':
            continue
        for frame, name, h, role in zip(asset['frame_ids'], asset['files'], asset['frame_sha256'], asset['semantics']):
            aliases.setdefault(name, []).append(dict(map=asset['map_id'], frame=frame, role=role, old_sha256=h))
    args.out.mkdir()
    pack = args.out/'pack'
    pack.mkdir()
    names = set()
    for asset in base['assets']:
        for key in ('file', 'master_file'):
            if asset.get(key): names.add(asset[key])
        for key in ('files', 'source_files'):
            names.update(asset.get(key, []))
    for name in names:
        assert Path(name).name == name and (args.pack/name).is_file(), name
        (pack/name).symlink_to((args.pack/name).resolve())
    frames, bindings, replacements, source_checks = [], [], {}, []
    for profile_name, profile in profiles:
        # The current inventory adds later references. Revalidate every original
        # three-layer profile input instead of replacing its historical hash.
        checks = 0
        for group in profile['groups']:
            original = native(args.original, sources, *group['anchor'])
            assert hashlib.sha256(original).hexdigest() == group['native_content_sha256']
            for target in group['targets']:
                assert native(args.original, sources, target['map'], target['frame']) == original
                checks += 1
        source_checks.append(dict(profile=profile_name, historical_inventory_sha256=profile['source_manifest_sha256'],
                                  current_inventory_sha256=sha(args.sources/'manifest.json'),
                                  original_three_layer_inputs_revalidated=checks))
        policy = profile['shade_policy']
        indices = set(policy['index_groups']['grass'])
        for group in profile['groups']:
            old = group['file']
            if group['cell'] in EXCLUDED_CELLS[profile_name] or group['compile_mode'] != 'atlas_direct' or old not in aliases:
                continue
            content = native(args.original, sources, *group['anchor'])
            assert hashlib.sha256(content).hexdigest() == group['native_content_sha256']
            raw = content[:576]
            if sum(v in indices for v in raw) < 16:
                continue
            assert sha(args.pack/old) == aliases[old][0]['old_sha256']
            with Image.open(args.pack/old) as image:
                painting = image.convert('RGBA')
            assert painting.size == (144, 144) and painting.getextrema()[3] == (255, 255)
            mask = background_mask(painting, raw, policy)
            if not mask.getbbox() or sum(mask.tobytes())//255 < 72:
                continue
            background, _ = composite_relief(bytes(v if v in indices else 68 for v in raw), {'grass': grass}, policy)
            result = Image.composite(background, painting, mask)
            changed = ImageChops.difference(result, painting).convert('RGB')
            protected = Image.composite(changed, Image.new('RGB', changed.size), ImageChops.invert(mask))
            assert protected.getbbox() is None, '物件RGB被改動'
            name = old.removesuffix('.png')+'-groundfix-v1.png'
            assert name not in names and name not in replacements
            result.save(pack/name)
            row = dict(old_file=old, old_sha256=sha(args.pack/old), file=name, sha256=sha(pack/name),
                       anchor=group['anchor'], source_profile=profile_name,
                       native_content_sha256=group['native_content_sha256'],
                       background_pixels=sum(mask.tobytes())//255,
                       retained_pixels=144*144-sum(mask.tobytes())//255,
                       retained_rgb_unchanged=True, mask_l_sha256=hashlib.sha256(mask.tobytes()).hexdigest())
            replacements[old] = row
            frames.append(row)
            for target in aliases[old]:
                assert native(args.original, sources, target['map'], target['frame']) == content, '同名來源並非相同原生內容'
                witness = next(t for t in group['targets'] if (t['map'], t['frame']) == (target['map'], target['frame']))
                bindings.append({**target, 'reuse_from': dict(map=target['map'], frame=target['frame']),
                                 'selector': witness['selector'], 'input_layers': sources['maps'][target['map']]['frames'][target['frame']]['input_layers']})
    assert frames, '沒有可更新的背景'
    candidate = copy.deepcopy(base)
    changed_assets = []
    for asset in candidate['assets']:
        if asset['role'] != 'native_terrain_hd_set': continue
        changed = False
        for i, name in enumerate(asset['files']):
            if name in replacements:
                replacement = replacements[name]
                asset['files'][i] = replacement['file']
                asset['frame_sha256'][i] = replacement['sha256']
                changed = True
        if changed:
            asset['source_refs'] = list(dict.fromkeys([*asset['source_refs'], RECIPE_REF]))
            changed_assets.append(asset['asset_id'])
    candidate['private_root'] = str(pack.relative_to(args.repo))
    dump(args.out/'catalog.json', candidate)
    recipe = dict(schema_version=1, kind='fd2_embedded_ground_consistency_v1',
                  base_catalog_sha256=sha(args.catalog), source_manifest_sha256=sha(args.sources/'manifest.json'),
                  grass=material, source_profiles=[dict(file=n, sha256=sha(profile_dir/n)) for n in PROFILES],
                  original_source_checks=source_checks,
                  visually_excluded_cells={name: sorted(cells) for name, cells in EXCLUDED_CELLS.items()},
                  segmentation='edge-connected painted green within native-grass zone plus six HD pixels; opaque RGB outside mask unchanged',
                  frames=frames, changed_assets=changed_assets)
    dump(args.out/'recipe.json', recipe)
    dump(args.out/'manifest.json', dict(kind='fd2_painted_terrain_atlas_v1', catalog_sha256=sha(args.out/'catalog.json'),
         new_sets=len({r['map'] for r in bindings}), bindings=bindings, frames=frames,
         recipe_sha256=sha(args.out/'recipe.json'), scope='同一已登錄草地材質的編製候選；未改原生mask、selector或遊戲規則。'))
    review = Image.new('RGB', (8*288, ((len(frames)+7)//8)*168), '#182434')
    draw = ImageDraw.Draw(review)
    for i, row in enumerate(frames):
        x, y = i%8*288, i//8*168
        for dx, name in ((0, row['old_file']), (144, row['file'])):
            with Image.open(pack/name) as image: review.paste(image.convert('RGB'), (x+dx, y))
        draw.text((x, y+145), row['old_file'].replace('hd1-terrain-', '').replace('-hd.png', ''), fill='white')
    review.save(args.out/'comparison.png')
    print(json.dumps(dict(status='PREVIEW', changed_tiles=len(frames), bindings=len(bindings), maps=sorted({x['map'] for x in bindings}))))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', type=Path, default=Path(__file__).resolve().parents[1])
    for key in ('catalog', 'pack', 'original', 'sources', 'out'):
        parser.add_argument('--'+key, type=Path, required=True)
    prepare(parser.parse_args())
