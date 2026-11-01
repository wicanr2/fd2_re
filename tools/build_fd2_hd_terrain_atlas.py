#!/usr/bin/env python3
"""依原生三層及 selector 見證，從新繪圖集編製完整地表。"""
import argparse
from collections import defaultdict
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import tempfile

from PIL import Image

from build_fd2_hd_ground import make_tile
from build_fd2_hd_ground_edges import composite_relief
from build_fd2_hd_terrain_reuse import native, witness
from validate_modern_theme_catalog import ROOT, validate
from verify_fd2_terrain_sources import check_binding, sha, verify


def read(path):
    return json.loads(path.read_text())


def write(path, value):
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n')


def catalog_files(catalog, pack):
    names, hashes = set(), {}
    for asset in catalog['assets']:
        pairs = []
        if 'file' in asset:
            pairs.append((asset['file'], asset['sha256']))
        if 'master_file' in asset:
            names.add(asset['master_file'])
        for field, digest in [('files', 'frame_sha256'), ('source_files', 'source_sha256')]:
            if len(asset.get(field, [])) != len(asset.get(digest, [])):
                raise ValueError('基底來源長度不同')
            pairs.extend(zip(asset.get(field, []), asset.get(digest, [])))
        for name, digest in pairs:
            if name in hashes and hashes[name] != digest:
                raise ValueError('基底來源SHA衝突')
            names.add(name)
            hashes[name] = digest
    for name in names:
        if Path(name).name != name or not (pack / name).is_file():
            raise ValueError('基底素材缺少或路徑越界')
        if name in hashes and sha(pack / name) != hashes[name]:
            raise ValueError('基底素材SHA不同')
    return names


def source_groups(recipe, coverage, source, original, maps):
    groups = recipe['groups']
    if not groups or len(groups) != recipe['expected_cells']:
        raise ValueError('圖集來源格數不完整')
    if len({g['cell'] for g in groups}) != len(groups) or len({g['native_content_sha256'] for g in groups}) != len(groups):
        raise ValueError('圖集格或原生來源重複')
    if [m['map'] for m in coverage['maps']] != list(range(33)) or [m['map'] for m in source['maps']] != list(range(33)):
        raise ValueError('缺少33張原生地圖')
    by_hash = {g['native_content_sha256']: g for g in groups}
    bindings, contents = [], {}
    for mp in coverage['maps']:
        original_mp = source['maps'][mp['map']]
        if mp['bank'] != original_mp['bank'] or mp['map_data'] != original_mp['map_data']:
            raise ValueError('地圖來源綁定不同')
        check_binding(original, mp['bank'])
        data = read(check_binding(maps, mp['map_data']))
        for row in mp['required']:
            group = by_hash.get(row['native_content_sha256'])
            if group is None or row['status'] != 'missing-art':
                continue
            witness(data, row)
            content = native(original, source, mp['map'], row['frame'])
            if set(content[576:1152]) != {1} or any(content[1152:]):
                raise ValueError('本批只接受完整來源且無目的地重映射')
            anchor = tuple(group['anchor'])
            if anchor not in contents:
                contents[anchor] = native(original, source, *anchor)
            if content != contents[anchor] or hashlib.sha256(content).hexdigest() != group['native_content_sha256']:
                raise ValueError('不同原生三層不能共用新圖')
            bindings.append({'map': mp['map'], 'frame': row['frame'], 'role': row['role'],
                             'selector': row['selector'], 'native_content_sha256': row['native_content_sha256'],
                             'cell': group['cell'], 'reuse_from': {'map': anchor[0], 'frame': anchor[1]},
                             'bank': mp['bank'], 'map_data': mp['map_data']})
    for group in groups:
        actual = [{k: b[k] for k in ('map', 'frame', 'role', 'selector')} for b in bindings if b['cell'] == group['cell']]
        if actual != group['targets'] or not any([b['map'], b['frame']] == group['anchor'] for b in actual):
            raise ValueError('目標來源少項、錯角色或順序不同')
    if len(bindings) != recipe['expected_bindings'] or len({(b['map'], b['frame'], b['role']) for b in bindings}) != len(bindings):
        raise ValueError('目標槽不完整或重複')
    return bindings


def grass_policy(recipe, catalog, pack):
    policy = recipe['shade_policy']
    reference = ROOT / policy['reference']['file']
    if sha(reference) != policy['reference']['sha256']:
        raise ValueError('明暗配方來源SHA不同')
    approved = read(reference)
    indices = approved['index_groups']['grass']
    if (policy['index_groups'] != {'grass': indices} or
            policy['index_shades'] != {str(v): approved['index_shades'][str(v)] for v in indices} or
            policy['mask_blur_hd_pixels'] != 2 or policy['exact_outer_edge_hd_pixels'] != 1):
        raise ValueError('明暗索引或輪廓契約不同')
    material = policy['material']
    entries = [a for a in catalog['assets'] if a['asset_id'] == material['asset_id']]
    if (len(entries) != 1 or entries[0]['role'] != 'terrain_hd_master' or
            (entries[0]['file'], entries[0]['sha256']) != (material['file'], material['sha256']) or
            Path(material['file']).name != material['file'] or sha(pack / material['file']) != material['sha256']):
        raise ValueError('既定手繪材質綁定不同')
    picture, edge = make_tile(pack / material['file'], 144, 8)
    return policy, picture, edge


def prepare(recipe_path, catalog_path, pack, original, sources, maps, master_path, coverage_path, out):
    if not Path('/.dockerenv').exists() or os.path.lexists(out) or not out.parent.is_dir():
        raise ValueError('只在Docker寫入既有父目錄中的新候選')
    recipe, catalog, coverage, source = map(read, (recipe_path, catalog_path, coverage_path, sources / 'manifest.json'))
    if (recipe['schema_version'], recipe['kind'], recipe['master_size'], recipe['grid'], recipe['size'], recipe['edge_blend']) != (
            1, 'fd2_painted_terrain_atlas_v1', [1024, 1536], [4, 6], 144, 8):
        raise ValueError('完整地表圖集契約不同')
    for path, key in [(catalog_path, 'base_catalog_sha256'), (coverage_path, 'coverage_sha256'),
                      (master_path, 'master_sha256'), (sources / 'manifest.json', 'source_manifest_sha256')]:
        if sha(path) != recipe[key]:
            raise ValueError('基底、原生來源或母稿SHA不同：' + key)
    if coverage['catalog_sha256'] != sha(catalog_path):
        raise ValueError('覆蓋與基底清冊不同')
    with Image.open(master_path) as opened:
        master = opened.convert('RGBA')
    if list(master.size) != recipe['master_size'] or master.getchannel('A').getextrema() != (255, 255):
        raise ValueError('圖集母稿尺寸或透明度不同')
    bindings = source_groups(recipe, coverage, source, original, maps)
    policy, grass, grass_edge = grass_policy(recipe, catalog, pack)
    names = catalog_files(catalog, pack)
    new_names = [g[k] for g in recipe['groups'] for k in ('file', 'master_file') if k in g]
    if len(set(new_names)) != len(new_names) or names.intersection(new_names) or any(Path(n).name != n for n in new_names):
        raise ValueError('成品名稱重複、覆寫或路徑越界')
    existing = {(a['map_id'], f) for a in catalog['assets'] if a['role'] == 'native_terrain_hd_set' for f in a['frame_ids']}
    if existing.intersection((b['map'], b['frame']) for b in bindings):
        raise ValueError('拒絕覆寫已登錄來源槽')
    source_check = verify(original, maps, sources)
    stage = Path(tempfile.mkdtemp(prefix='.terrain-atlas-', dir=out.parent))
    try:
        for name in sorted(names):
            (stage / name).symlink_to((pack / name).resolve())
        frames, additions = [], []
        refs = ['remake/assets/themes/modern/' + recipe_path.name, 'docs/knowledge-base/61-modern-theme-runtime-spec.md']
        for group in recipe['groups']:
            cell = group['cell']
            if type(cell) is not int or not 0 <= cell < 24:
                raise ValueError('圖集來源格越界')
            frame = {'cell': cell, 'file': group['file'], 'compile_mode': group['compile_mode'],
                     'native_content_sha256': group['native_content_sha256']}
            if group['compile_mode'] == 'atlas_cell':
                crop = [cell % 4 * 256 + 4, cell // 4 * 256 + 4, cell % 4 * 256 + 252, cell // 4 * 256 + 252]
                if group['crop'] != crop:
                    raise ValueError('裁切越界或包含圖集邊線')
                master.crop(crop).save(stage / group['master_file'])
                tile, edge = make_tile(stage / group['master_file'], 144, 8)
                frame.update(master_file=group['master_file'], master_sha256=sha(stage / group['master_file']),
                             crop=crop, edge_calibration=edge)
                additions.append({'asset_id': recipe['asset_id'] + f'.cell{cell:02d}.master', 'role': 'terrain_hd_master',
                                  'status': 'concept', 'file': group['master_file'], 'width': 248, 'height': 248,
                                  'sha256': frame['master_sha256'], 'source_refs': refs})
            elif group['compile_mode'] == 'native_grass_shade':
                raw = native(original, source, *group['anchor'])[:576]
                if set(raw) - set(policy['index_groups']['grass']) or 'master_file' in group or 'crop' in group:
                    raise ValueError('陰影地表出現未知索引或未採圖集裁切')
                tile, mask = composite_relief(raw, {'grass': grass}, policy)
                frame.update(material=policy['material'], shade_mask_sha256=hashlib.sha256(mask.tobytes()).hexdigest(),
                             edge_calibration=grass_edge)
            else:
                raise ValueError('未知地表編製模式')
            tile.save(stage / group['file'])
            frame['sha256'] = sha(stage / group['file'])
            frames.append(frame)
        if len(additions) != recipe['expected_crop_masters']:
            raise ValueError('裁切母稿數不同')
        pictures = {f['cell']: f for f in frames}
        by_map = defaultdict(dict)
        for row in bindings:
            picture = pictures[row['cell']]
            row.update(file=picture['file'], sha256=picture['sha256'])
            prior = by_map[row['map']].get(row['frame'])
            by_map[row['map']][row['frame']] = {**row, 'role': 'both' if prior else row['role']}
        for map_id, slots in sorted(by_map.items()):
            rows = [slots[f] for f in sorted(slots)]
            additions.append({'asset_id': recipe['asset_id'] + f'.map{map_id:02d}.native_hd', 'role': 'native_terrain_hd_set',
                              'status': 'runtime_candidate', 'consumer_contract': 'fdshap_native_hd_sparse_v1',
                              'map_id': map_id, 'width': 144, 'height': 144, 'frame_count': len(rows),
                              'frame_ids': [b['frame'] for b in rows], 'files': [b['file'] for b in rows],
                              'frame_sha256': [b['sha256'] for b in rows], 'semantics': [b['role'] for b in rows],
                              'alpha_contract': 'native_mask_nearest_6', 'cycle_policy': 'native_selector', 'source_refs': refs})
        if {a['asset_id'] for a in additions} & {a['asset_id'] for a in catalog['assets']}:
            raise ValueError('新增清冊身分重複')
        proposed = copy.deepcopy(catalog)
        proposed['assets'].extend(additions)
        proposed['private_root'] = stage.relative_to(ROOT).as_posix()
        write(stage / 'catalog.json', proposed)
        check = validate(True, stage / 'catalog.json')
        proposed['private_root'] = out.relative_to(ROOT).as_posix()
        write(stage / 'catalog.json', proposed)
        result = {'schema_version': 1, 'kind': recipe['kind'], 'status': 'SOURCE-VERIFIED-CANDIDATE',
                  'base_catalog_sha256': sha(catalog_path), 'catalog_sha256': sha(stage / 'catalog.json'),
                  'recipe_sha256': sha(recipe_path), 'master_sha256': sha(master_path), 'coverage_sha256': sha(coverage_path),
                  'source_manifest_sha256': sha(sources / 'manifest.json'), 'source_verification': source_check,
                  'catalog_validation': check, 'frames': frames, 'bindings': bindings, 'new_assets': additions,
                  'new_image_files': len(frames) + recipe['expected_crop_masters'],
                  'new_master_files': recipe['expected_crop_masters'], 'shade_policy': policy,
                  'new_sets': len(by_map), 'source_rgb_used': False,
                  'tool_sha256': sha(Path(__file__))}
        write(stage / 'manifest.json', result)
        stage.rename(out)
        return result
    except BaseException:
        shutil.rmtree(stage, ignore_errors=True)
        raise


def register(catalog_path, candidate, gpu_path):
    base, proposed, manifest, gpu = map(read, (catalog_path, candidate / 'catalog.json', candidate / 'manifest.json', gpu_path))
    if (not Path('/.dockerenv').exists() or manifest['kind'] != 'fd2_painted_terrain_atlas_v1' or
            sha(catalog_path) != manifest['base_catalog_sha256'] or sha(candidate / 'catalog.json') != manifest['catalog_sha256'] or
            gpu.get('status') != 'PASS-PAINTED-TERRAIN-ATLAS-GPU' or gpu.get('catalog_sha256') != manifest['catalog_sha256'] or
            gpu.get('manifest_sha256') != sha(candidate / 'manifest.json')):
        raise ValueError('登錄基底、候選或GPU來源不同')
    wanted = {f'map{b["map"]:02d}-{b["frame"]:04d}-{b["role"]}-{mode}': (b['map'], b['frame'])
              for b in manifest['bindings'] for mode in ('steady', 'lut', 'black', 'white')}
    rows = gpu.get('cases', [])
    source_rows, game_rows = [r for r in rows if not r.get('game_draw')], [r for r in rows if r.get('game_draw')]
    first = manifest['bindings'][0]
    if (len(wanted) != len(manifest['bindings']) * 4 or len(source_rows) != len(wanted) or
            {r['case'] for r in source_rows} != set(wanted) or
            any((r['map'], r['source_frame']) != wanted[r['case']] for r in source_rows) or
            len(game_rows) != 2 or any(r['map'] != first['map'] or not r.get('f2') for r in game_rows) or
            any(r.get('rgba_tolerance') != 2 for r in rows) or gpu.get('save_bytes_unchanged', 0) <= 0):
        raise ValueError('GPU來源、F2或保存檢查缺項')
    if proposed['assets'] != base['assets'] + manifest['new_assets']:
        raise ValueError('候選改動既有清冊或新增未驗來源')
    if len([a for a in manifest['new_assets'] if a['role'] == 'terrain_hd_master']) != manifest['new_master_files']:
        raise ValueError('裁切母稿未完整登錄')
    for frame in manifest['frames']:
        pairs = [(frame['file'], frame['sha256'])]
        if 'master_file' in frame:
            pairs.append((frame['master_file'], frame['master_sha256']))
        for name, digest in pairs:
            if Path(name).name != name or sha(candidate / name) != digest:
                raise ValueError('已驗成品或裁切母稿變動')
    for row in rows:
        if row.get('file') and (Path(row['file']).name != row['file'] or sha(gpu_path.parent / row['file']) != row['sha256']):
            raise ValueError('GPU畫面SHA不同')
    validate(True, candidate / 'catalog.json')
    proposed['private_root'] = base['private_root']
    with tempfile.NamedTemporaryFile('w', dir=catalog_path.parent, delete=False) as stream:
        stream.write(json.dumps(proposed, ensure_ascii=False, indent=2) + '\n')
        temporary = Path(stream.name)
    os.chmod(temporary, catalog_path.stat().st_mode & 0o777)
    temporary.replace(catalog_path)
    return {'status': 'REGISTERED-PAINTED-TERRAIN-ATLAS', 'bindings': len(manifest['bindings']),
            'frames': len(manifest['frames']), 'catalog_assets': len(proposed['assets']), 'catalog_sha256': sha(catalog_path)}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--register', action='store_true')
    for name in ('recipe', 'catalog', 'pack', 'original', 'sources', 'maps', 'master', 'coverage', 'out', 'candidate', 'gpu-receipt'):
        p.add_argument('--' + name, type=lambda v: Path(v).resolve())
    a = p.parse_args()
    args = (a.catalog, a.candidate, a.gpu_receipt) if a.register else (a.recipe, a.catalog, a.pack, a.original, a.sources, a.maps, a.master, a.coverage, a.out)
    if any(v is None for v in args):
        p.error('missing explicit inputs')
    report = register(*args) if a.register else prepare(*args)
    print(json.dumps({k: v for k, v in report.items() if k not in ('frames', 'bindings', 'new_assets', 'source_verification', 'catalog_validation')}, ensure_ascii=False))


if __name__ == '__main__':
    main()
