#!/usr/bin/env python3
"""編製已核對的map9石壁或平面地板；保留原生覆蓋及背景契約。"""
import argparse
from collections import deque
import hashlib
import json
import os
from pathlib import Path
import shutil
import tempfile

from PIL import Image
from verify_fd2_terrain_sources import check_binding


def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()


def read(p):
    return json.loads(p.read_text())


def painted_atlas(master):
    image = master.convert('RGB').resize((576, 576), Image.Resampling.LANCZOS)
    pixels = list(image.getdata())
    owner = [-1] * len(pixels)
    queue = deque()
    for i, (r, g, b) in enumerate(pixels):
        if not (b > r + 8 and g > r):
            owner[i] = i
            queue.append(i)
    if not len(pixels) // 20 < len(queue) < len(pixels):
        raise ValueError('石材／深藍空洞分類不適用')
    while queue:
        i = queue.popleft()
        x, y = i % 576, i // 576
        for n in (i - 1 if x else -1, i + 1 if x < 575 else -1,
                  i - 576 if y else -1, i + 576 if y < 575 else -1):
            if n >= 0 and owner[n] == -1:
                owner[n] = owner[i]
                queue.append(n)
    padded = Image.new('RGB', image.size)
    padded.putdata([pixels[i] for i in owner])
    return padded, [i != source for i, source in enumerate(owner)]


def prepare(recipe_path, catalog_path, pack, original, sources, maps, master_path, out, coverage_path=None):
    if not Path('/.dockerenv').exists() or out.exists() or not out.parent.is_dir():
        raise ValueError('需Docker、既有父目錄與全新候選')
    recipe, catalog, source = read(recipe_path), read(catalog_path), read(sources / 'manifest.json')
    if recipe.get('kind') == 'fd2_cave_floor_details_hd_recipe':
        return prepare_details(recipe_path, catalog_path, pack, original, sources, maps, master_path, out, coverage_path)
    floor = recipe.get('kind') == 'fd2_cave_floor_hd_recipe'
    start = 80 if floor else 32
    prefix = 'cave-floor' if floor else 'cave-rock'
    asset = 'cave_floor' if floor else 'cave_rock_partial'
    if recipe.get('kind') not in ('fd2_cave_floor_hd_recipe', 'fd2_cave_rock_partial_hd_recipe') or (recipe['schema_version'], recipe['map_id'], recipe['frame_ids'], recipe['master_size']) != (1, 9, list(range(start, start + 16)), [1254, 1254]):
        raise ValueError('固定石壁配方不同')
    if sha(catalog_path) != recipe['base_catalog_sha256'] or sha(master_path) != recipe['master_sha256']:
        raise ValueError('基底或母稿SHA不同')
    with Image.open(master_path) as source_master:
        master = source_master.copy()
    if list(master.size) != recipe['master_size'] or master.convert('RGBA').getchannel('A').getextrema() != (255, 255):
        raise ValueError('母稿形狀或透明度不同')
    if any(a['role'] == 'native_terrain_hd_set' and a['map_id'] == 9 and set(a['frame_ids']) & set(recipe['frame_ids']) for a in catalog['assets']):
        raise ValueError('已有石壁槽登錄')
    mp = source['maps'][9]
    bank = read(check_binding(original, mp['bank']))
    check_binding(maps, mp['map_data'])
    if not set(recipe['frame_ids']) <= set(mp['selected_terrain_frames']):
        raise ValueError('石壁不在實際selector清單')
    painting, padded = (master.convert('RGB').resize((576, 576), Image.Resampling.LANCZOS), [False] * (576 * 576)) if floor else painted_atlas(master)
    pictures, frames = {}, []
    for i, frame in enumerate(recipe['frame_ids']):
        src = mp['frames'][frame]
        paths = [check_binding(original, p) for p in src['input_layers']]
        layers = [Image.open(p) for p in paths]
        if any(im.size != (24, 24) for im in layers):
            raise ValueError('原生三層尺寸不同')
        indices, mask, remap = [im.tobytes() for im in layers]
        if any(v not in (0, 255) for v in mask + remap) or any(a and b for a, b in zip(mask, remap)):
            raise ValueError('原生來源與目的地遮罩重疊')
        if floor and (set(mask) != {255} or any(remap)):
            raise ValueError('地板必須為完整來源且無目的地重映射')
        content = indices + bytes(bool(v) for v in mask) + bytes(bool(v) for v in remap)
        if hashlib.sha256(content).hexdigest() != src['native_content_sha256']:
            raise ValueError('原生三層內容SHA不同')
        x0, y0 = i % 4 * 144, i // 4 * 144
        image = painting.crop((x0, y0, x0 + 144, y0 + 144)).convert('RGBA')
        pixels = list(image.getdata())
        image.putdata([p[:3] + (255,) if mask[(y // 6) * 24 + x // 6] else (0, 0, 0, 0)
                       for y in range(144) for x, p in enumerate(pixels[y * 144:(y + 1) * 144])])
        count = sum(padded[(y0 + y) * 576 + x0 + x] and bool(mask[(y // 6) * 24 + x // 6]) for y in range(144) for x in range(144))
        name = f'map09-frame{frame:04d}-{prefix}-hd.png'
        pictures[name] = image
        frames.append({'map': 9, 'frame': frame, 'file': name, 'native_content_sha256': src['native_content_sha256'],
                       'source_pixels': sum(bool(v) for v in mask), 'remap_pixels': sum(bool(v) for v in remap),
                       'painted_void_edge_padding_pixels': count})
    names = set()
    for a in catalog['assets']:
        for field in ('file', 'master_file'):
            if field in a:
                names.add(a[field])
        names.update(a.get('files', []))
        names.update(a.get('source_files', []))
    if any(Path(n).name != n or not (pack / n).is_file() for n in names):
        raise ValueError('既有包缺件或路徑不安全')
    out.mkdir()
    for name in sorted(names):
        (out / name).symlink_to(pack.resolve() / name)
    shutil.copyfile(master_path, out / recipe['master_file'])
    for row in frames:
        pictures[row['file']].save(out / row['file'])
        row['sha256'] = sha(out / row['file'])
    refs = [f'remake/assets/themes/modern/{"cave-floor-hd" if floor else "cave-rock-partial-hd"}.recipe.json', 'docs/knowledge-base/61-modern-theme-runtime-spec.md', 'docs/data/fd2-reference-files.json']
    catalog['assets'].extend([
        {'asset_id': f'modern.terrain.{asset}.master', 'role': 'terrain_hd_master', 'status': 'concept',
         'file': recipe['master_file'], 'width': 1254, 'height': 1254, 'sha256': recipe['master_sha256'], 'source_refs': refs},
        {'asset_id': f'modern.terrain.{asset}.map09.native_hd', 'role': 'native_terrain_hd_set', 'status': 'runtime_candidate',
         'consumer_contract': 'fdshap_native_hd_sparse_v1', 'map_id': 9, 'width': 144, 'height': 144, 'frame_count': 16,
         'frame_ids': recipe['frame_ids'], 'semantics': ['terrain'] * 16, 'files': [f['file'] for f in frames],
         'frame_sha256': [f['sha256'] for f in frames], 'alpha_contract': 'native_mask_nearest_6', 'cycle_policy': 'native_selector', 'source_refs': refs}])
    root = Path(__file__).resolve().parents[1]
    catalog['private_root'] = out.relative_to(root).as_posix()
    (out / 'catalog.json').write_text(json.dumps(catalog, ensure_ascii=False, indent=2) + '\n')
    result = {'schema_version': 1, 'kind': 'fd2_cave_floor_hd' if floor else 'fd2_cave_rock_partial_hd', 'status': 'SOURCE-VERIFIED-CANDIDATE',
              'base_catalog_sha256': sha(catalog_path), 'catalog_sha256': sha(out / 'catalog.json'), 'recipe_sha256': sha(recipe_path),
              'master_sha256': recipe['master_sha256'], 'native_manifest_sha256': sha(sources / 'manifest.json'),
              'bank': mp['bank'], 'map_data': mp['map_data'], 'frames': frames, 'native_alpha': 'PASS',
              'tool_sha256': sha(Path(__file__)), 'source_rgb_used': False}
    (out / 'manifest.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    return result


def register(catalog_path, candidate, gpu_path):
    base, proposed, manifest, gpu = read(catalog_path), read(candidate / 'catalog.json'), read(candidate / 'manifest.json'), read(gpu_path)
    if manifest.get('kind') == 'fd2_cave_floor_details_hd':
        return register_details(catalog_path, candidate, gpu_path)
    floor = manifest.get('kind') == 'fd2_cave_floor_hd'
    if manifest.get('kind') not in ('fd2_cave_floor_hd', 'fd2_cave_rock_partial_hd'):
        raise ValueError('未知洞窟素材批次')
    frames = list(range(80, 96)) if floor else list(range(32, 48))
    status = 'PASS-CAVE-FLOOR-HD-GPU' if floor else 'PASS-CAVE-ROCK-HD-GPU'
    if not Path('/.dockerenv').exists() or sha(catalog_path) != manifest['base_catalog_sha256'] or sha(candidate / 'catalog.json') != manifest['catalog_sha256']:
        raise ValueError('正式或候選清冊已變動')
    if (gpu.get('status') != status or gpu.get('catalog_sha256') != manifest['catalog_sha256'] or
            gpu.get('manifest_sha256') != sha(candidate / 'manifest.json') or gpu.get('source_frames') != frames or
            gpu.get('source_cases') != 32 or gpu.get('complete_game_cases') != 2 or gpu.get('save_bytes_unchanged', 0) <= 0):
        raise ValueError('缺完整來源／Game.Draw／保存GPU收據')
    if len(gpu.get('cases', [])) != 34 or sum(bool(c.get('f2')) for c in gpu['cases']) != 2:
        raise ValueError('GPU案例或切換缺項')
    expected = {f'cave-{frame:04d}-raw{str(raw).lower()}': frame for frame in frames for raw in (False, True)}
    source_cases = [c for c in gpu['cases'] if not c.get('game_draw')]
    if ({c['case'] for c in source_cases} != set(expected) or
            any(c.get('map') != 9 or c.get('source_frame') != expected[c['case']] for c in source_cases) or
            any(c.get('rgba_tolerance') != 2 for c in gpu['cases'])):
        raise ValueError('GPU來源／原色／LUT案例不同')
    master_file = 'cave-floor-hd-v1-master.png' if floor else 'cave-rock-partial-hd-v1-master.png'
    if sha(candidate / master_file) != manifest['master_sha256']:
        raise ValueError('母稿已變動')
    for c in gpu['cases']:
        if c.get('file'):
            if Path(c['file']).name != c['file'] or sha(gpu_path.parent / c['file']) != c['sha256']:
                raise ValueError('GPU實圖不符')
    if proposed['assets'][:-2] != base['assets'] or len(proposed['assets']) != len(base['assets']) + 2:
        raise ValueError('既有清冊被改寫')
    for frame in manifest['frames']:
        if sha(candidate / frame['file']) != frame['sha256']:
            raise ValueError('成品變動')
    proposed['private_root'] = base['private_root']
    with tempfile.NamedTemporaryFile('w', dir=catalog_path.parent, delete=False, encoding='utf-8') as f:
        f.write(json.dumps(proposed, ensure_ascii=False, indent=2) + '\n')
        temporary = Path(f.name)
    os.chmod(temporary, catalog_path.stat().st_mode & 0o777)
    temporary.replace(catalog_path)
    return {'status': 'REGISTERED-CAVE-FLOOR-HD' if floor else 'REGISTERED-CAVE-ROCK-HD', 'frames': 16, 'catalog_assets': len(proposed['assets']), 'catalog_sha256': sha(catalog_path)}


def prepare_details(recipe_path, catalog_path, pack, original, sources, maps, master_path, out, coverage_path):
    from build_fd2_hd_terrain_reuse import native, witness
    if coverage_path is None:
        raise ValueError('明暗地板須明示基底覆蓋資料')
    recipe, catalog, source, coverage = read(recipe_path), read(catalog_path), read(sources / 'manifest.json'), read(coverage_path)
    if (coverage.get('schema_version'), coverage.get('kind')) != (1, 'fd2_hd1_terrain_coverage'):
        raise ValueError('明暗地板覆蓋資料格式不同')
    if (recipe['schema_version'], recipe['map_id'], recipe['frame_ids'], recipe['master_size'], recipe['master_file']) != (1, 9, [74, 75, 78, 79], [1254, 1254], 'cave-floor-details-hd-v1-master.png'):
        raise ValueError('固定明暗地板配方不同')
    if (sha(catalog_path) != recipe['base_catalog_sha256'] or sha(master_path) != recipe['master_sha256'] or
            sha(coverage_path) != recipe['coverage_sha256'] or coverage['catalog_sha256'] != sha(catalog_path) or
            sha(sources / 'manifest.json') != recipe['source_manifest_sha256']):
        raise ValueError('明暗地板基底、母稿或來源SHA不符')
    with Image.open(master_path) as opened:
        master = opened.convert('RGBA')
    if list(master.size) != [1254, 1254] or master.getchannel('A').getextrema() != (255, 255):
        raise ValueError('母稿尺寸或透明度不符')
    painting = master.convert('RGB').resize((288, 288), Image.Resampling.LANCZOS)
    anchors, pictures = {}, {}
    for i, frame in enumerate(recipe['frame_ids']):
        content = native(original, source, 9, frame)
        if set(content[576:1152]) != {1} or any(content[1152:]):
            raise ValueError('明暗地板須為完整來源且無目的地重映射')
        anchors[hashlib.sha256(content).hexdigest()] = frame, content
        pictures[frame] = painting.crop((i % 2 * 144, i // 2 * 144, i % 2 * 144 + 144, i // 2 * 144 + 144)).convert('RGBA')
    bindings, actual = [], []
    for mp in coverage['maps']:
        original_mp = source['maps'][mp['map']]
        if mp['bank'] != original_mp['bank'] or mp['map_data'] != original_mp['map_data']:
            raise ValueError('明暗地板地圖綁定不符')
        for row in mp['required']:
            if row['role'] != 'terrain' or row['status'] != 'missing-art' or row['native_content_sha256'] not in anchors:
                continue
            frame, content = anchors[row['native_content_sha256']]
            check_binding(original, mp['bank'])
            data = read(check_binding(maps, mp['map_data']))
            witness(data, row)
            if native(original, source, mp['map'], row['frame']) != content:
                raise ValueError('明暗地板三層內容不同')
            target = {'map': mp['map'], 'frame': row['frame'], 'source_frame': frame}
            actual.append(target)
            bindings.append({**target, 'role': 'terrain', 'selector': row['selector'],
                             'native_content_sha256': row['native_content_sha256'], 'bank': mp['bank'], 'map_data': mp['map_data']})
    if actual != recipe['targets'] or len(actual) != 12 or {r['map'] for r in actual} != {9, 24, 25, 26}:
        raise ValueError('明暗地板完整12槽範圍不同')
    names = set()
    for asset in catalog['assets']:
        names.update(asset.get('files', [])); names.update(asset.get('source_files', []))
        names.update(asset[k] for k in ('file', 'master_file') if k in asset)
    new_names = {recipe['master_file'], *(f'map09-frame{f:04d}-cave-floor-details-hd.png' for f in pictures)}
    if (names & new_names or any(Path(n).name != n or not (pack / n).is_file() for n in names) or
            Path(recipe['master_file']).name != recipe['master_file']):
        raise ValueError('基底缺件、路徑不安全或圖片已存在')
    existing = {(a['map_id'], f) for a in catalog['assets'] if a['role'] == 'native_terrain_hd_set' for f in a['frame_ids']}
    if existing & {(b['map'], b['frame']) for b in bindings}:
        raise ValueError('明暗地板目標已有登錄')
    stage = Path(tempfile.mkdtemp(prefix='.cave-details-', dir=out.parent))
    try:
        for name in sorted(names):
            (stage / name).symlink_to(pack.resolve() / name)
        shutil.copyfile(master_path, stage / recipe['master_file'])
        frames = []
        for frame, picture in pictures.items():
            name = f'map09-frame{frame:04d}-cave-floor-details-hd.png'
            picture.save(stage / name)
            frames.append({'frame': frame, 'file': name, 'sha256': sha(stage / name)})
        images = {f['frame']: f for f in frames}
        refs = ['remake/assets/themes/modern/cave-floor-details-hd.recipe.json', 'docs/knowledge-base/61-modern-theme-runtime-spec.md']
        additions = [{'asset_id': 'modern.terrain.cave_floor_details.master', 'role': 'terrain_hd_master', 'status': 'concept',
                      'file': recipe['master_file'], 'width': 1254, 'height': 1254, 'sha256': sha(master_path), 'source_refs': refs}]
        for map_id in sorted({b['map'] for b in bindings}):
            rows = [b for b in bindings if b['map'] == map_id]
            for row in rows:
                row.update(file=images[row['source_frame']]['file'], sha256=images[row['source_frame']]['sha256'])
            additions.append({'asset_id': f'modern.terrain.cave_floor_details.map{map_id:02d}.native_hd', 'role': 'native_terrain_hd_set',
                'status': 'runtime_candidate', 'consumer_contract': 'fdshap_native_hd_sparse_v1', 'map_id': map_id,
                'width': 144, 'height': 144, 'frame_count': len(rows), 'frame_ids': [b['frame'] for b in rows],
                'files': [b['file'] for b in rows], 'frame_sha256': [b['sha256'] for b in rows], 'semantics': ['terrain'] * len(rows),
                'alpha_contract': 'native_mask_nearest_6', 'cycle_policy': 'native_selector', 'source_refs': refs})
        if {a['asset_id'] for a in additions} & {a['asset_id'] for a in catalog['assets']}:
            raise ValueError('明暗地板集合身分已存在')
        catalog['assets'].extend(additions)
        catalog['private_root'] = out.relative_to(Path(__file__).resolve().parents[1]).as_posix()
        (stage / 'catalog.json').write_text(json.dumps(catalog, ensure_ascii=False, indent=2) + '\n')
        result = {'schema_version': 1, 'kind': 'fd2_cave_floor_details_hd', 'status': 'SOURCE-VERIFIED-CANDIDATE',
                  'base_catalog_sha256': sha(catalog_path), 'catalog_sha256': sha(stage / 'catalog.json'), 'coverage_sha256': sha(coverage_path),
                  'recipe_sha256': sha(recipe_path), 'master_sha256': sha(master_path), 'master_file': recipe['master_file'],
                  'native_manifest_sha256': sha(sources / 'manifest.json'), 'frames': frames, 'bindings': bindings, 'new_sets': 4,
                  'native_alpha': 'PASS', 'tool_sha256': sha(Path(__file__)), 'source_rgb_used': False}
        (stage / 'manifest.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
        stage.rename(out)
        return result
    except BaseException:
        shutil.rmtree(stage, ignore_errors=True)
        raise


def register_details(catalog_path, candidate, gpu_path):
    base, proposed, manifest, gpu = read(catalog_path), read(candidate / 'catalog.json'), read(candidate / 'manifest.json'), read(gpu_path)
    if [f['frame'] for f in manifest['frames']] != [74, 75, 78, 79] or manifest['master_file'] != 'cave-floor-details-hd-v1-master.png':
        raise ValueError('明暗地板固定成品範圍不同')
    if (not Path('/.dockerenv').exists() or sha(catalog_path) != manifest['base_catalog_sha256'] or
            sha(candidate / 'catalog.json') != manifest['catalog_sha256'] or gpu.get('catalog_sha256') != manifest['catalog_sha256'] or
            gpu.get('detail_manifest_sha256') != sha(candidate / 'manifest.json') or gpu.get('status') != 'PASS-HD-MAP-OVERVIEW-GPU'):
        raise ValueError('明暗地板登錄來源或GPU收據不符')
    rows = gpu.get('cases', [])
    wanted = {f'details-{b["map"]:02d}-{b["frame"]:04d}-{mode}': (b['map'], b['frame']) for b in manifest['bindings'] for mode in ('steady', 'lut', 'black', 'white')}
    source_rows = [c for c in rows if not c.get('game_draw')]
    game_rows = [c for c in rows if c.get('game_draw')]
    if (len(manifest['bindings']) != 12 or len(wanted) != 48 or len(source_rows) != 48 or gpu.get('detail_source_cases') != 48 or
            {c['case'] for c in source_rows} != set(wanted) or any((c['map'], c['source_frame']) != wanted[c['case']] for c in source_rows) or
            [c['map'] for c in game_rows] != [0, 1, 9, 24, 25, 26] or len(gpu.get('maps', [])) != 6 or
            any(not c.get('f2') for c in game_rows) or any(s.get('save_bytes', 0) <= 0 for s in gpu['maps']) or
            any(c.get('rgba_tolerance') != 2 for c in rows)):
        raise ValueError('明暗地板來源、六圖F2或保存檢查缺項')
    if proposed['assets'][:-5] != base['assets'] or len(proposed['assets']) != len(base['assets']) + 5:
        raise ValueError('明暗地板改寫舊清冊')
    master = proposed['assets'][-5]
    if (master['role'], master['file'], master['sha256'], master['width'], master['height']) != ('terrain_hd_master', manifest['master_file'], manifest['master_sha256'], 1254, 1254):
        raise ValueError('明暗地板母稿集合不符')
    expected = {(b['map'], b['frame']): (b['file'], b['sha256']) for b in manifest['bindings']}
    actual = {}
    for asset in proposed['assets'][-4:]:
        if asset['role'] != 'native_terrain_hd_set' or asset['semantics'] != ['terrain'] * asset['frame_count']:
            raise ValueError('明暗地板新增其他角色')
        for frame, name, digest in zip(asset['frame_ids'], asset['files'], asset['frame_sha256']):
            if (asset['map_id'], frame) in actual:
                raise ValueError('明暗地板重複槽')
            actual[asset['map_id'], frame] = name, digest
    if actual != expected or sha(candidate / manifest['master_file']) != manifest['master_sha256']:
        raise ValueError('明暗地板登錄與已驗來源不同')
    for row in manifest['frames']:
        if Path(row['file']).name != row['file'] or sha(candidate / row['file']) != row['sha256']:
            raise ValueError('明暗地板成品變動')
    for case in game_rows:
        if Path(case['file']).name != case['file'] or sha(gpu_path.parent / case['file']) != case['sha256']:
            raise ValueError('明暗地板遊戲PNG不符')
    proposed['private_root'] = base['private_root']
    with tempfile.NamedTemporaryFile('w', dir=catalog_path.parent, delete=False, encoding='utf-8') as target:
        target.write(json.dumps(proposed, ensure_ascii=False, indent=2) + '\n'); temporary = Path(target.name)
    os.chmod(temporary, catalog_path.stat().st_mode & 0o777)
    temporary.replace(catalog_path)
    return {'status': 'REGISTERED-CAVE-FLOOR-DETAILS-HD', 'bindings': 12, 'frames': 4, 'catalog_assets': len(proposed['assets']),
            'catalog_sha256': sha(catalog_path), 'candidate_catalog_sha256': manifest['catalog_sha256'], 'gpu_sha256': sha(gpu_path)}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--register', action='store_true')
    for name in ('recipe', 'catalog', 'pack', 'original', 'sources', 'maps', 'master', 'out', 'candidate', 'gpu-receipt', 'coverage'):
        p.add_argument('--' + name, type=lambda v: Path(v).resolve())
    a = p.parse_args()
    args = (a.catalog, a.candidate, a.gpu_receipt) if a.register else (a.recipe, a.catalog, a.pack, a.original, a.sources, a.maps, a.master, a.out)
    if any(v is None for v in args):
        p.error('missing explicit inputs')
    result = register(*args) if a.register else prepare(*args, coverage_path=a.coverage)
    print(json.dumps({k: v for k, v in result.items() if k != 'frames'}, ensure_ascii=False))


if __name__ == '__main__':
    main()
