#!/usr/bin/env python3
"""從固定原生分區及已登錄母稿刷新地表，連同全部草地相依圖。"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import tempfile

from PIL import Image, ImageDraw, ImageFilter

from build_fd2_hd_ground import ground_sources, make_tile
from build_fd2_hd_ground_edges import composite_materials, composite_relief, validate_sources
from build_fd2_hd_props import compose as compose_props, validate_sources as props_sources
from build_fd2_hd_terrain_atlas import catalog_files, masked_object_tile, rgba_object_tile
from build_fd2_hd_terrain_reuse import native, witness
from build_fd2_hd_terrain_object import fit, calibrate_repeat_edges
from refresh_fd2_hd1_ground import names, reject
from validate_modern_theme_catalog import validate
from verify_fd2_terrain_sources import check_binding, sha, verify

ROOT = Path(__file__).resolve().parent.parent
RECIPES = ROOT / 'remake/assets/themes/modern'
RECIPE_NAME = 'hd1-pixel-terrain-refresh.recipe.json'
MODES = {'native_grass_shade', 'atlas_masked_object', 'atlas_rgba_object'}


def read(path):
    return json.loads(path.read_text())


def write(path, value):
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n')


def profile(entry):
    reject(Path(entry['file']).name == entry['file'], '配方路徑越界')
    path = RECIPES / entry['file']
    reject(sha(path) == entry['sha256'], '來源配方SHA不同')
    return read(path)


def updated_catalog(base, recipe, replacements, new_hashes):
    """只允許五母稿與全部同名成品引用一起更新，其餘逐欄保持。"""
    out = copy.deepcopy(base)
    master_ids = {m['asset_id']: m for m in recipe['materials'].values()}
    ref = 'remake/assets/themes/modern/' + RECIPE_NAME
    consumed = set()
    changed = []
    for asset in out['assets']:
        old = copy.deepcopy(asset)
        if asset['asset_id'] in master_ids:
            m = master_ids[asset['asset_id']]
            reject((asset['file'], asset['sha256']) == (m['old_file'], m['old_sha256']), '基底母稿不同')
            asset.update(file=m['file'], sha256=m['sha256'], width=m['size'][0], height=m['size'][1])
        if asset['role'] == 'native_terrain_hd_set':
            for i, filename in enumerate(asset['files']):
                if filename in replacements:
                    replacement = replacements[filename]
                    asset['files'][i] = replacement
                    asset['frame_sha256'][i] = new_hashes[replacement]
                    consumed.add(filename)
        if asset != old:
            if ref not in asset.setdefault('source_refs', []):
                asset['source_refs'].append(ref)
            changed.append(asset['asset_id'])
    reject(consumed == set(replacements), '成品替換未完整消費')
    style = recipe['style']
    reject(style['asset_id'] not in {a['asset_id'] for a in base['assets']}, '概念圖身分已存在')
    for source in [style, *recipe['alpha_sources']]:
        reject(source['asset_id'] not in {a['asset_id'] for a in out['assets']}, '概念來源身分已存在')
        out['assets'].append(dict(asset_id=source['asset_id'], role='terrain_hd_master', status='concept',
                                  file=source['file'], sha256=source['sha256'], width=source['size'][0],
                                  height=source['size'][1], source_refs=[ref]))
    reject(len(out['assets']) == recipe['base_assets'] + 7 == len(base['assets']) + 7, '資產數量不同')
    reject(len({a['asset_id'] for a in out['assets']}) == len(out['assets']), '資產身分重複')
    return out, changed


def prepare(recipe_path, catalog_path, pack, original, sources, maps, masters, out):
    reject(Path('/.dockerenv').exists(), '只允許在Docker執行')
    reject(not os.path.lexists(out) and out.parent.is_dir(), '輸出必須為新目錄')
    r, base, source = read(recipe_path), read(catalog_path), read(sources / 'manifest.json')
    reject(recipe_path.name == RECIPE_NAME, '正式配方名稱不同')
    reject((r['schema_version'], r['kind']) == (1, 'fd2_hd1_pixel_terrain_refresh_v1'), '刷新契約不同')
    reject(sha(catalog_path) == r['base_catalog_sha256'], '基底清冊SHA不同')
    reject(sha(sources / 'manifest.json') == r['source_manifest_sha256'], '原生清冊SHA不同')
    reject(set(r['materials']) == {'grass', 'earth', 'stone', 'sand', 'water'}, '母稿白名單不同')
    reject(sha(masters / r['style']['file']) == r['style']['sha256'], '核定原型不同')
    baseline = names(base)
    # master_file is part of the archival allow-list even when a legacy entry
    # has no separate SHA field. Runtime-only names omit these source masters.
    for filename in catalog_files(base, pack):
        baseline.setdefault(filename, sha(pack / filename))
    for filename, digest in baseline.items():
        reject((pack / filename).is_file() and sha(pack / filename) == digest, '基底檔案不同：' + filename)
    proof = verify(original, maps, sources)
    by_id = {a['asset_id']: a for a in base['assets']}
    tiles, edges = {}, {}
    new_names = {r['style']['file']}
    for entry in r['alpha_sources']:
        reject(sha(masters / entry['file']) == entry['sha256'] and Path(entry['file']).name == entry['file'], '去背景來源不同')
        new_names.add(entry['file'])
    for key, m in r['materials'].items():
        reject(Path(m['file']).name == m['file'] and m['file'] not in baseline, '母稿名稱重複或越界')
        old = by_id[m['asset_id']]
        reject(old['role'] == 'terrain_hd_master' and (old['file'], old['sha256']) == (m['old_file'], m['old_sha256']), '母稿身分不同')
        reject(sha(masters / m['file']) == m['sha256'], '新母稿SHA不同')
        with Image.open(masters / m['file']) as im:
            reject(list(im.size) == m['size'], '母稿尺寸不同')
        tiles[key], edges[key] = make_tile(masters / m['file'], 144, 8)
        new_names.add(m['file'])
    reject(not new_names & set(baseline) and len(new_names) == 12, '來源檔名衝突')
    stage = Path(tempfile.mkdtemp(prefix='.pixel-terrain-', dir=out.parent))
    try:
        assets = stage / 'pack'
        assets.mkdir()
        for filename in baseline:
            (assets / filename).symlink_to((pack / filename).resolve())
        for filename in new_names:
            shutil.copyfile(masters / filename, assets / filename)
        replacements, frames, bindings, all_slots = {}, [], [], set()
        slots = {(a['map_id'], f): (filename, h) for a in base['assets'] if a['role'] == 'native_terrain_hd_set'
                 for f, filename, h in zip(a['frame_ids'], a['files'], a['frame_sha256'])}

        def emit(old, picture, anchor, content_hash, targets, recipe_name):
            reject(old in baseline and old not in replacements, '替換重複或沒有基底：' + old)
            filename = old[:-4] + '-pixelhand-v3.png'
            reject(filename not in baseline and filename not in new_names and Path(filename).name == filename, '成品名稱衝突')
            reject(picture.size == (144, 144) and picture.getextrema()[3] == (255, 255), '尺寸或alpha不同')
            reject(hashlib.sha256(native(original, source, *anchor)).hexdigest() == content_hash, '原生內容SHA不同')
            expected = {(t['map'], t['frame']) for t in targets}
            actual = {key for key, value in slots.items() if value[0] == old}
            reject(expected == actual and not expected & all_slots, '同源引用缺漏、錯號或重複')
            for target in targets:
                key = target['map'], target['frame']
                reject(hashlib.sha256(native(original, source, *key)).hexdigest() == content_hash, '目標原生內容不同')
                frame = source['maps'][key[0]]['frames'][key[1]]
                reject(frame['frame'] == key[1], '原生清冊索引不同')
                bindings.append(dict(map=key[0], frame=key[1], file=filename, input_layers=frame['input_layers']))
            picture.save(assets / filename)
            replacements[old] = filename
            all_slots.update(expected)
            frames.append(dict(old_file=old, file=filename, sha256=sha(assets / filename), anchor=anchor,
                               native_content_sha256=content_hash, targets=sorted(expected), profile=recipe_name))

        profiles = [profile(entry) for entry in r['profiles']]
        reject(tuple(e['file'] for e in r['profiles']) == ('ground-olive-hd.recipe.json', 'ground-grass-earth-hd.recipe.json',
               'ground-bases-hd.recipe.json', 'shore-hd.recipe.json', 'relief-hd.recipe.json'), '地表配方白名單不同')
        pure = ground_sources(original, source, profiles[0])
        batches = [(profiles[0], [dict(group=dict(anchor=profiles[0]['anchor'], file=profiles[0]['tile_file'],
                    native_content_sha256=profiles[0]['native_content_sha256']), bindings=pure, pixels=None)])]
        batches.extend((p, validate_sources(original, source, p)) for p in profiles[1:])
        seams = []
        for (p, rows), entry in zip(batches, r['profiles']):
            labels = {v: key for key, values in p.get('index_groups', {}).items() for v in values}
            for row in rows:
                group, raw = row['group'], row['pixels']
                if raw is None:
                    picture = tiles['grass'].copy()
                    classes = [('grass', 255)] * 576
                elif p['kind'] == 'fd2_relief_hd_v1':
                    picture, _ = composite_relief(raw, {k: tiles[k] for k in ('grass', 'earth', 'stone')}, p)
                    classes = [(labels[v], p['index_shades'][str(v)]) for v in raw]
                else:
                    pair = group.get('materials', ['grass', 'earth'])
                    if len(pair) == 1:
                        picture = tiles[pair[0]].copy()
                    else:
                        local = {**p, 'index_groups': dict(grass=p['index_groups'][pair[0]], earth=p['index_groups'][pair[1]])}
                        picture, _ = composite_materials(raw, tiles[pair[0]], tiles[pair[1]], local)
                    classes = [(labels[v], 255) for v in raw]
                old = group['file'].replace('-hd-v1.png', r['old_frame_suffix'])
                emit(old, picture, group['anchor'], group['native_content_sha256'], row['bindings'], entry['file'])
                seams.append((picture, classes))
        reject(len(frames) == 87 and len(all_slots) == 384, '基礎地表覆蓋數不同')
        for entry in r['dependent_profiles']:
            p = profile(entry)
            policy = p['shade_policy']
            ref = policy['reference']
            reject(sha(ROOT / ref['file']) == ref['sha256'], '草地明暗來源不同')
            known = read(ROOT / ref['file'])
            indices = known['index_groups']['grass']
            reject(policy['index_groups'] == {'grass': indices} and policy['index_shades'] ==
                   {str(v): known['index_shades'][str(v)] for v in indices}, '草地明暗索引不同')
            groups = [g for g in p['groups'] if g['compile_mode'] in MODES]
            reject(len(groups) == entry['groups'], '相依圖數量不同')
            for g in groups:
                raw = native(original, source, *g['anchor'])[:576]
                for target in g['targets']:
                    witness(read(check_binding(maps, source['maps'][target['map']]['map_data'])), target)
                if g['compile_mode'] == 'native_grass_shade':
                    picture, _ = composite_relief(raw, {'grass': tiles['grass']}, policy)
                else:
                    master_id = p['asset_id'] + f'.cell{g["cell"]:02d}.master'
                    m = by_id[master_id]
                    reject(m['file'] == g['master_file'] and sha(pack / m['file']) == m['sha256'], '已登錄物件母稿不同')
                    painting = Image.open(pack / m['file']).convert('RGBA').resize((144, 144), Image.Resampling.LANCZOS)
                    if g['compile_mode'] == 'atlas_masked_object':
                        picture, _, _ = masked_object_tile(raw, painting, tiles['grass'], policy)
                    else:
                        picture, _ = rgba_object_tile(raw, painting, tiles['grass'], policy)
                emit(g['file'], picture, g['anchor'], g['native_content_sha256'], g['targets'], entry['file'])
        props = profile(r['props_profile'])
        for row in props_sources(original, maps, source, props):
            g = row['group']
            m = props['masters'][g['master']]
            reject(sha(pack / m['file']) == m['sha256'], '道具母稿不同')
            painting = Image.open(pack / m['file']).convert('RGBA')
            picture = compose_props(row['pixels'], tiles['grass'], painting, g, props)
            emit(g['file'], picture, g['anchor'], g['native_content_sha256'], row['bindings'], r['props_profile']['file'])
        policy = profile(r['dependent_profiles'][0])['shade_policy']
        reject(set(policy['index_groups']['grass']) == {68, 69, 70, 71, 137, 138, 139, 140, 141, 142},
               '首章嵌入草地分區不同')
        alpha_tiles = {}
        masks = stage / 'masks'
        masks.mkdir()
        for entry in r['alpha_sources']:
            geometry = profile(entry['object_recipe'])
            picture = Image.open(masters / entry['file']).convert('RGBA')
            reject(list(picture.size) == entry['size'] == geometry['painted_master_size'], '去背景畫布或錨點不同')
            reject(picture.getextrema()[3][0] == 0 and picture.getextrema()[3][1] >= 200, '去背景來源缺少有效alpha')
            reject(entry['alpha_threshold'] == 128 and entry['outer_padding_hd_pixels'] == 2, 'alpha保護契約不同')
            alpha = picture.getchannel('A').point(lambda v: 255 if v >= 128 else 0)
            padded = Image.new('L', (alpha.width + 2, alpha.height + 2))
            padded.paste(alpha, (1, 1))
            ImageDraw.floodfill(padded, (0, 0), 128)
            alpha = padded.crop((1, 1, alpha.width + 1, alpha.height + 1)).point(lambda v: 0 if v == 128 else 255)
            # Treat the matte as opaque grayscale while using the existing
            # repeat-edge compiler, whose contract deliberately rejects RGBA.
            canvas = fit(alpha.convert('RGBA'), geometry)
            matte = canvas.getchannel('R').filter(ImageFilter.MaxFilter(5))
            canvas = Image.merge('RGBA', (matte, matte, matte, Image.new('L', matte.size, 255)))
            obj = next(obj for obj in source['objects'] if obj['id'] == geometry['object'])
            calibrate_repeat_edges(canvas, obj, geometry.get('repeat_edges', {}), 6)
            for cell in obj['cells']:
                rect = (cell['x'] * 144, cell['y'] * 144, (cell['x'] + 1) * 144, (cell['y'] + 1) * 144)
                mask = canvas.getchannel('R').crop(rect)
                for frame in [cell['frame'], cell.get('foreground_frame')]:
                    if frame is not None:
                        key = obj['map'], frame
                        reject(key not in alpha_tiles, '手繪alpha來源重複')
                        alpha_tiles[key] = mask
        for g in r['embedded_grass_groups']:
            raw = native(original, source, *g['anchor'])
            reject(all(raw[576:1152]) and not any(raw[1152:]), '嵌入草地只接受完整opaque來源')
            original_painting = Image.open(pack / g['file']).convert('RGBA')
            background_raw = bytes(v if v in policy['index_groups']['grass'] else 68 for v in raw[:576])
            background, _ = composite_relief(background_raw, {'grass': tiles['grass']}, policy)
            mask = alpha_tiles[tuple(g['anchor'])]
            picture = Image.composite(original_painting, background, mask)
            mask.save(masks / g['file'])
            emit(g['file'], picture, g['anchor'], g['native_content_sha256'], g['targets'], 'embedded-grass-painted-alpha')
            frames[-1]['painted_alpha_sha256'] = sha(masks / g['file'])
        equal = 0
        for a, ca in seams:
            for b, cb in seams:
                for i in range(144):
                    for pa, pb in [((143, i), (0, i)), ((i, 143), (i, 0))]:
                        if ca[pa[1] // 6 * 24 + pa[0] // 6] == cb[pb[1] // 6 * 24 + pb[0] // 6]:
                            reject(a.getpixel(pa) == b.getpixel(pb), '相同材質接縫不同')
                            equal += 1
        hashes = {f['file']: f['sha256'] for f in frames}
        candidate, changed = updated_catalog(base, r, replacements, hashes)
        candidate['private_root'] = str((out / 'pack').relative_to(ROOT))
        write(stage / 'catalog.json', candidate)
        required = {(mp['map'], f['frame']): f for mp in read(ROOT / 'docs/data/fd2-hd1-terrain-coverage.json')['maps']
                    for f in mp['required'] if f['role'] == 'terrain'}
        for b in bindings:
            b['sha256'] = hashes[b['file']]
            b['reuse_from'] = dict(map=b['map'], frame=b['frame'])
            v = required[b['map'], b['frame']]
            reject(v['source_pixels'] == 576 and v['remap_pixels'] == 0, '地表來源不是完整opaque')
            b.update({key: v[key] for key in ('role', 'source_pixels', 'remap_pixels', 'selector')})
        # Anchor every unique PNG, then one witness for every additional map.
        sample = [next(b for b in bindings if (b['map'], b['frame']) == tuple(f['anchor'])) for f in frames]
        for mp in sorted({b['map'] for b in bindings} - {b['map'] for b in sample}):
            sample.append(next(b for b in bindings if b['map'] == mp))
        write(stage / 'gpu-detail.json', dict(kind='fd2_painted_terrain_atlas_v1', refresh_kind=r['kind'],
                                             catalog_sha256=sha(stage / 'catalog.json'),
                                             new_sets=len({b['map'] for b in bindings}), bindings=sample))
        preview = Image.new('RGBA', (5 * 144, 2 * 144))
        for i, (key, m) in enumerate(r['materials'].items()):
            old, _ = make_tile(pack / m['old_file'], 144, 8)
            preview.paste(old, (i * 144, 0))
            preview.paste(tiles[key], (i * 144, 144))
        preview.save(stage / 'materials-preview.png')
        report = dict(status='PASS-HD1-PIXEL-TERRAIN-CANDIDATE', runtime_integrated=False,
                      recipe_sha256=sha(recipe_path), base_catalog_sha256=sha(catalog_path),
                      catalog_sha256=sha(stage / 'catalog.json'), source_manifest_sha256=r['source_manifest_sha256'],
                      source_verification=proof, tool_sha256=sha(Path(__file__)), frames=frames, bindings=bindings,
                      maps=len({b['map'] for b in bindings}), changed_assets=changed,
                      unchanged_assets=len(base['assets']) - len(changed), material_edges=edges,
                      equal_material_edge_pixels=equal, gpu_samples=len(sample), new_files=len(frames) + len(new_names))
        write(stage / 'manifest.json', report)
        validate(False, stage / 'catalog.json')
        stage.rename(out)
        return report
    except BaseException:
        shutil.rmtree(stage, ignore_errors=True)
        raise


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('recipe', 'catalog', 'pack', 'original', 'sources', 'maps', 'masters', 'out'):
        parser.add_argument('--' + name, type=lambda value: Path(value).resolve(), required=True)
    a = parser.parse_args()
    result = prepare(a.recipe, a.catalog, a.pack, a.original, a.sources, a.maps, a.masters, a.out)
    print(json.dumps({k: result[k] for k in ('status', 'maps', 'unchanged_assets', 'gpu_samples', 'new_files')}, ensure_ascii=False))


if __name__ == '__main__':
    main()
