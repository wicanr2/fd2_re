#!/usr/bin/env python3
"""從新手繪母稿編製已證實共用的純地形，沿既有144稀疏清冊。"""

import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import tempfile

from PIL import Image

from verify_fd2_terrain_sources import check_binding, sha, verify


def ground_sources(pack, manifest, recipe):
    """只共用實際被selector選取且三個原生圖層逐byte相同的來源。"""
    if (recipe.get('schema_version'), recipe.get('kind'), recipe.get('size')) != (1, 'fd2_shared_ground_hd_v1', 144):
        raise ValueError('草地配方契約不符')
    targets = recipe['targets']
    if (not targets or any(not isinstance(v, list) or len(v) != 2 or
                          any(type(n) is not int for n in v) for v in targets) or
            len({tuple(v) for v in targets}) != len(targets)):
        raise ValueError('草地來源重複或越界')
    expected = recipe['native_content_sha256']
    if not isinstance(expected, str) or len(expected) != 64:
        raise ValueError('缺少原生內容綁定')
    actual = []
    bindings = []
    for mp in manifest['maps']:
        for fr in mp['frames']:
            if fr['native_content_sha256'] != expected or fr['frame'] not in mp['selected_terrain_frames']:
                continue
            files = [check_binding(pack, e) for e in fr['input_layers']]
            images = [Image.open(p) for p in files]
            if len(images) != 3 or any(im.size != (24, 24) for im in images):
                raise ValueError('草地原生圖層不完整')
            raw, opaque, remap = [im.tobytes() for im in images]
            if len(raw) != 576 or len(opaque) != 576 or len(remap) != 576 or not all(opaque) or any(remap):
                raise ValueError('草地必須全opaque且沒有destination remap')
            content = raw + bytes(bool(v) for v in opaque) + bytes(bool(v) for v in remap)
            if hashlib.sha256(content).hexdigest() != expected:
                raise ValueError('共用草地的原生內容不同')
            actual.append([mp['map'], fr['frame']])
            bindings.append({'map': mp['map'], 'frame': fr['frame'], 'input_layers': fr['input_layers'],
                             'native_content_sha256': expected})
    if actual != targets or recipe['anchor'] not in actual:
        raise ValueError('配方與實際使用的共用草地來源不同')
    return bindings


def make_tile(master, size, blend):
    if type(blend) is not int or not 1 <= blend <= size // 8:
        raise ValueError('接縫校準範圍越界')
    source = Image.open(master).convert('RGBA')
    if source.width != source.height or source.width < size or source.getextrema()[3] != (255, 255):
        raise ValueError('草地母稿必須為不透明正方形')
    tile = source.resize((size, size), Image.Resampling.LANCZOS)
    before = hashlib.sha256(tile.tobytes()).hexdigest()
    # Only painted RGB participates. Opposite edge bands converge symmetrically;
    # no original pixel colour, palette shade, noise or state enters the painting.
    for axis in (0, 1):
        original = tile.copy()
        for cross in range(size):
            for offset in range(blend):
                a = (offset, cross) if axis == 0 else (cross, offset)
                b = (size - 1 - offset, cross) if axis == 0 else (cross, size - 1 - offset)
                x, y = original.getpixel(a), original.getpixel(b)
                weight = (blend - offset) / (2 * blend)
                tile.putpixel(a, tuple(round(v * (1 - weight) + w * weight) for v, w in zip(x[:3], y[:3])) + (255,))
                tile.putpixel(b, tuple(round(w * (1 - weight) + v * weight) for v, w in zip(x[:3], y[:3])) + (255,))
    if any(tile.getpixel((0, i)) != tile.getpixel((size - 1, i)) or
           tile.getpixel((i, 0)) != tile.getpixel((i, size - 1)) for i in range(size)):
        raise ValueError('草地相對邊緣不一致')
    return tile, {'unadjusted_rgba_sha256': before, 'edge_band': blend,
                  'opposite_edges_equal': True, 'rgb_source': 'new painted master only'}


def build(pack, maps, sources, base_catalog, base_pack, master, recipe_path, out):
    if os.path.lexists(out) or not out.parent.is_dir():
        raise ValueError('輸出須為既有父目錄下的新目錄')
    recipe = json.loads(recipe_path.read_text())
    if sha(base_catalog) != recipe['base_catalog_sha256'] or sha(sources / 'manifest.json') != recipe['source_manifest_sha256']:
        raise ValueError('基底清冊或原生製圖來源綁定不同')
    source_check = verify(pack, maps, sources)
    manifest = json.loads((sources / 'manifest.json').read_text())
    bindings = ground_sources(pack, manifest, recipe)
    tile, edge = make_tile(master, recipe['size'], recipe['edge_blend'])
    catalog = json.loads(base_catalog.read_text())
    names, expected = set(), {}
    for a in catalog['assets']:
        if 'file' in a:
            names.add(a['file']); expected[a['file']] = a['sha256']
        if 'master_file' in a:
            names.add(a['master_file'])
        for key, hashes in [('files', 'frame_sha256'), ('source_files', 'source_sha256')]:
            if len(a.get(key, [])) != len(a.get(hashes, [])):
                raise ValueError('基底清冊圖源綁定不完整')
            for name, digest in zip(a.get(key, []), a.get(hashes, [])):
                if name in expected and expected[name] != digest:
                    raise ValueError('基底清冊重複檔案雜湊衝突')
                names.add(name); expected[name] = digest
    for name in [*names, recipe['master_file'], recipe['tile_file']]:
        if Path(name).name != name:
            raise ValueError('清冊檔名越界')
    for name in sorted(names):
        p = base_pack / name
        if not p.is_file() or name in expected and sha(p) != expected[name]:
            raise ValueError('基底素材缺少或雜湊不同：' + name)
    if set([recipe['tile_file'], recipe['master_file']]) & names:
        raise ValueError('拒絕覆寫既有素材')
    existing = {(a['map_id'], f) for a in catalog['assets'] if a['role'] == 'native_terrain_hd_set' for f in a['frame_ids']}
    if existing.intersection(map(tuple, recipe['targets'])):
        raise ValueError('拒絕覆寫既有地形槽')
    temporary = Path(tempfile.mkdtemp(prefix='.ground-', dir=out.parent))
    try:
        assets = temporary / 'pack'; assets.mkdir()
        for name in sorted(names):
            (assets / name).symlink_to((base_pack / name).resolve())
        tile.save(assets / recipe['tile_file'])
        shutil.copyfile(master, assets / recipe['master_file'])
        master_id = recipe['asset_id'] + '.master'
        additions = [{'asset_id': master_id, 'role': 'terrain_hd_master', 'status': 'concept',
                      'file': recipe['master_file'], 'width': Image.open(master).width,
                      'height': Image.open(master).height, 'sha256': sha(master), 'source_refs': recipe['source_refs']}]
        for map_id, frame in recipe['targets']:
            additions.append({'asset_id': recipe['asset_id'] + f'.map{map_id:02d}.native_hd',
                              'role': 'native_terrain_hd_set', 'status': 'runtime_candidate',
                              'consumer_contract': 'fdshap_native_hd_sparse_v1', 'map_id': map_id,
                              'width': 144, 'height': 144, 'frame_count': 1, 'frame_ids': [frame],
                              'files': [recipe['tile_file']], 'frame_sha256': [sha(assets / recipe['tile_file'])],
                              'semantics': ['terrain'], 'alpha_contract': 'native_mask_nearest_6',
                              'cycle_policy': 'native_selector', 'source_refs': ['asset:' + master_id, *recipe['source_refs']]})
        if {a['asset_id'] for a in catalog['assets']} & {a['asset_id'] for a in additions}:
            raise ValueError('清冊身分已存在')
        candidate = copy.deepcopy(catalog); candidate['assets'].extend(additions)
        (temporary / 'catalog.json').write_text(json.dumps(candidate, ensure_ascii=False, indent=2) + '\n')
        preview = Image.new('RGBA', (432, 432))
        for y in range(3):
            for x in range(3):preview.paste(tile, (x * 144, y * 144))
        preview.save(temporary / 'repeat-preview.png')
        report = {'status': 'PASS-GROUND-CANDIDATE', 'runtime_integrated': False,
                  'recipe_sha256': sha(recipe_path), 'base_catalog_sha256': sha(base_catalog),
                  'source_manifest_sha256': sha(sources / 'manifest.json'), 'source_verification': source_check,
                  'native_content_sha256': recipe['native_content_sha256'], 'targets': bindings,
                  'master': {'file': recipe['master_file'], 'sha256': sha(master)},
                  'tile': {'file': recipe['tile_file'], 'sha256': sha(assets / recipe['tile_file'])},
                  'edge_calibration': edge, 'catalog_assets': len(candidate['assets']), 'base_assets_unchanged': len(catalog['assets']),
                  'scope': '離線新手繪草地；三層原生內容相同才共用。不等於原版RGB、正常玩家或GPU驗收。'}
        (temporary / 'manifest.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
        temporary.rename(out)
    except BaseException:
        shutil.rmtree(temporary, ignore_errors=True)
        raise
    return report


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['original-pack', 'maps', 'sources', 'base-catalog', 'base-pack', 'master', 'recipe', 'out']:
        p.add_argument('--' + name, type=Path, required=True)
    a = p.parse_args()
    report = build(a.original_pack, a.maps, a.sources, a.base_catalog, a.base_pack, a.master, a.recipe, a.out)
    print(json.dumps({'status': report['status'], 'slots': len(report['targets']), 'catalog_assets': report['catalog_assets']}))


if __name__ == '__main__':main()
