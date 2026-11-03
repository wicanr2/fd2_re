#!/usr/bin/env python3
"""依已審原型及當前正式覆蓋，保存單份圖集的編製輸入與配方。"""
import argparse
import copy
import hashlib
import json
import re
from pathlib import Path

from build_fd2_hd_terrain_atlas import read, write

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def preserve(path, content):
    if path.exists():
        if path.read_bytes() != content:
            raise ValueError('已有不同輸入，不得覆寫：' + str(path))
    else:
        if path.parent.stat().st_uid != 1000:
            raise ValueError('輸出目錄擁有權不同')
        path.write_bytes(content)


def stage(batch_root, atlas, catalog_path, coverage_path):
    if not Path('/.dockerenv').exists() or atlas not in range(8):
        raise ValueError('需Docker與明示圖集編號0至7')
    batch_root.relative_to(ROOT / 'work')
    ready = read(batch_root / 'ready-source.json')
    batch_id = ready.get('batch_id', 'r74')
    rows = ready['atlases']
    def row_grid(a):
        grid = a.get('grid', ready.get('grid', [4, 6]))
        if (not isinstance(grid, list) or grid not in ([4, 6], [8, 12]) or
                any(type(v) is not int for v in grid)):
            raise ValueError('READY圖集形狀不同')
        return grid

    compile_mode = ready.get('compile_mode', 'atlas_direct')
    if (not re.fullmatch(r'r[1-9][0-9]{1,3}', batch_id) or ready['status'] != 'READY' or
            compile_mode not in ('atlas_direct', 'atlas_native_mask') or
            any(a.get('compile_mode', 'atlas_direct') != compile_mode for a in rows) or
            len(rows) != 8 or {a['atlas'] for a in rows} != set(range(8)) or
            any(type(a['accepted_cells']) is not int or not 1 <= a['accepted_cells'] <= row_grid(a)[0] * row_grid(a)[1] or
                type(a['bindings']) is not int or a['bindings'] < a['accepted_cells'] for a in rows) or
            ready['accepted_cells'] != sum(a['accepted_cells'] for a in rows) or
            ready['accepted_bindings'] != sum(a['bindings'] for a in rows)):
        raise ValueError('本批READY範圍不同')
    for name, key in [('source-pack.json', 'source_pack_sha256'), ('prototype-review.json', 'prototype_review_sha256')]:
        if sha(batch_root / name) != ready[key]:
            raise ValueError('READY圖源或審查不同')
    row = next(a for a in ready['atlases'] if a['atlas'] == atlas)
    grid = row_grid(row)
    if (not isinstance(grid, list) or len(grid) != 2 or any(type(v) is not int for v in grid) or
            grid not in ([4, 6], [8, 12])):
        raise ValueError('READY圖集形狀不同')
    columns, cell_size = grid[0], 1024 // grid[0]
    work = batch_root / f'atlas-{atlas:02d}'
    source_path, prototype_path = work / 'source-prototype.json', work / 'direct-prototype-r1/prototype.json'
    pack = read(batch_root / 'source-pack.json')
    source_entry = next(a for a in pack['atlases'] if a['atlas'] == atlas)
    if source_entry.get('grid', pack.get('grid', [4, 6])) != grid:
        raise ValueError('來源包與READY形狀不同')
    if sha(source_path) != source_entry['sha256']:
        raise ValueError('原型來源不同')
    for path, key in [(work / 'atlas-r1-master.png', 'raw_master_sha256'),
                      (prototype_path, 'prototype_sha256'), (work / 'imagegen-request-r1.json', 'request_sha256')]:
        if sha(path) != row[key]:
            raise ValueError('原稿、提示詞或已審原型不同')
    source, prototype, coverage = read(source_path), read(prototype_path), read(coverage_path)
    if (any(type(x.get('source_rgb_used', False)) is not bool for x in (row, source, prototype)) or
            len({x.get('source_rgb_used', False) for x in (row, source, prototype)}) != 1):
        raise ValueError('原色來源政策不同')
    if coverage['catalog_sha256'] != sha(catalog_path):
        raise ValueError('正式覆蓋與當前清冊不同')
    slots = {(m['map'], f['frame'], f['role']): f for m in coverage['maps'] for f in m['required']}
    accepted = set(prototype['accepted_cells'])
    groups = []
    for g in source['groups']:
        if g['cell'] not in accepted:
            continue
        for target in g['targets']:
            actual = slots[target['map'], target['frame'], target['role']]
            if (actual['status'] != 'missing-art' or actual['selector'] != target['selector'] or
                    actual['native_content_sha256'] != g['native_content_sha256']):
                raise ValueError('來源不再缺圖或selector不同')
        cell = g['cell']
        groups.append({'cell': cell, 'anchor': g['anchor'], 'native_content_sha256': g['native_content_sha256'],
                       'compile_mode': compile_mode, 'edge_pad_master_pixels': row['edge_pad_master_pixels'],
                       'crop': [cell % columns * cell_size, cell // columns * cell_size,
                                cell % columns * cell_size + cell_size, cell // columns * cell_size + cell_size],
                       'master_file': f'hd1-terrain-{batch_id}-a{atlas:02d}-cell{cell:02d}-master.png',
                       'file': f'hd1-terrain-{batch_id}-a{atlas:02d}-cell{cell:02d}-hd.png', 'targets': g['targets']})
    if (len(groups), sum(len(g['targets']) for g in groups)) != (row['accepted_cells'], row['bindings']):
        raise ValueError('已審格子或目標少項')
    template = read(ROOT / 'remake/assets/themes/modern/hd1-pale-terrain-r73.recipe.json')
    recipe = {k: copy.deepcopy(template[k]) for k in
              ['schema_version', 'kind', 'master_size', 'grid', 'size', 'edge_blend', 'shade_policy']}
    recipe.update(asset_id=f'modern.terrain.{batch_id}.atlas{atlas:02d}', base_catalog_sha256=sha(catalog_path),
                  coverage_sha256=sha(coverage_path), source_manifest_sha256=ready['source_manifest_sha256'],
                  master_sha256=row['raw_master_sha256'], expected_cells=len(groups), expected_crop_masters=len(groups),
                  expected_bindings=row['bindings'], master_alpha_contract='opaque',
                  generation=read(work / 'imagegen-request-r1.json'), groups=groups,
                  scope='依原可見色群及高低輪廓重繪，身份／材質未知；原native透明、selector、cycle及規則保持。',
                  direct_policy='完整256裁切母稿保持；編製複本明示0或2像素內側RGB外延後LANCZOS144，無裁邊、UV環繞或對邊混合。')
    recipe['grid'] = grid
    if cell_size == 128:
        recipe['master_cell_size'] = 128
        recipe['direct_policy'] = '完整128裁切母稿及alpha保持，只LANCZOS144；校準0，不裁邊、UV環繞或對邊混合。'
    if compile_mode == 'atlas_native_mask':
        recipe['direct_policy'] = '完整128不透明母稿保持，RGB只LANCZOS144；再依原nativeMask nearest6裁切，alpha0的RGBA歸零，目的地remap保持透明。'
        if cell_size == 256:
            recipe['direct_policy'] = recipe['direct_policy'].replace('128', '256')
    if row.get('source_rgb_used', False):
        recipe['source_rgb_used'] = True
    if len(accepted) != len(source['groups']):
        recipe['rejected_cells'] = [{'cell': g['cell'], 'anchor': g['anchor'], 'native_content_sha256': g['native_content_sha256'],
                                     'reason': '構件與圖案待原圖整體拼接核對，按格轉繪未接受'}
                                    for g in source['groups'] if g['cell'] not in accepted]
    recipe_path = ROOT / f'remake/assets/themes/modern/hd1-terrain-{batch_id}-atlas{atlas:02d}.recipe.json'
    preserve(work / 'base-catalog.json', catalog_path.read_bytes())
    preserve(work / 'base-coverage.json', coverage_path.read_bytes())
    preserve(recipe_path, (json.dumps(recipe, ensure_ascii=False, indent=2) + '\n').encode())
    result = {'status': 'STAGED-READY-ATLAS', 'atlas': atlas, 'cells': len(groups), 'bindings': row['bindings'],
              'recipe': str(recipe_path.relative_to(ROOT)), 'recipe_sha256': sha(recipe_path),
              'base_catalog_sha256': sha(catalog_path), 'coverage_sha256': sha(coverage_path),
              'master_sha256': row['raw_master_sha256'], 'ready_sha256': sha(batch_root / 'ready-source.json'),
              'tool_sha256': sha(Path(__file__))}
    preserve(work / 'staged-inputs.json', (json.dumps(result, ensure_ascii=False, indent=2) + '\n').encode())
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--batch-root', type=lambda p: Path(p).resolve(), required=True)
    parser.add_argument('--atlas', type=int, required=True)
    parser.add_argument('--catalog', type=lambda p: Path(p).resolve(), default=ROOT / 'remake/assets/themes/modern/catalog.json')
    parser.add_argument('--coverage', type=lambda p: Path(p).resolve(), default=ROOT / 'docs/data/fd2-hd1-terrain-coverage.json')
    args = parser.parse_args()
    print(json.dumps(stage(args.batch_root, args.atlas, args.catalog, args.coverage), ensure_ascii=False))


if __name__ == '__main__':
    main()
