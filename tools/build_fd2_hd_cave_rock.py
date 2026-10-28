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


def prepare(recipe_path, catalog_path, pack, original, sources, maps, master_path, out):
    if not Path('/.dockerenv').exists() or out.exists() or not out.parent.is_dir():
        raise ValueError('需Docker、既有父目錄與全新候選')
    recipe, catalog, source = read(recipe_path), read(catalog_path), read(sources / 'manifest.json')
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


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--register', action='store_true')
    for name in ('recipe', 'catalog', 'pack', 'original', 'sources', 'maps', 'master', 'out', 'candidate', 'gpu-receipt'):
        p.add_argument('--' + name, type=lambda v: Path(v).resolve())
    a = p.parse_args()
    args = (a.catalog, a.candidate, a.gpu_receipt) if a.register else (a.recipe, a.catalog, a.pack, a.original, a.sources, a.maps, a.master, a.out)
    if any(v is None for v in args):
        p.error('missing explicit inputs')
    result = register(*args) if a.register else prepare(*args)
    print(json.dumps({k: v for k, v in result.items() if k != 'frames'}, ensure_ascii=False))


if __name__ == '__main__':
    main()
