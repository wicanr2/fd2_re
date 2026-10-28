#!/usr/bin/env python3
"""以完整三層內容相同的已登錄HD PNG補齊缺槽，不製造新圖。"""
import argparse
from collections import defaultdict
import hashlib
import json
import os
from pathlib import Path
import tempfile

from PIL import Image
from verify_fd2_terrain_sources import check_binding


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


def native(pack, manifest, map_id, frame):
    item = manifest['maps'][map_id]['frames'][frame]
    layers = [Image.open(check_binding(pack, p)) for p in item['input_layers']]
    if len(layers) != 3 or any(im.size != (24, 24) for im in layers):
        raise ValueError('原生三層形狀不符')
    if layers[0].mode != 'P' or any(set(im.getdata()) - {0, 255} for im in layers[1:]):
        raise ValueError('原生索引或二元遮罩不符')
    content = layers[0].tobytes() + bytes(bool(v) for v in layers[1].tobytes()) + bytes(bool(v) for v in layers[2].tobytes())
    if hashlib.sha256(content).hexdigest() != item['native_content_sha256']:
        raise ValueError('原生內容SHA不符')
    return content


def witness(data, row):
    s = row['selector']
    cell, passes = s['cell'], s['event_update_pass']
    if (type(cell) is not int or not 0 <= cell < len(data['tiles']) or
            type(passes) is not int or not 0 <= passes <= 1024 or
            s['flip'] not in (0, 1) or s['cycle'] not in range(4)):
        raise ValueError('缺少有效原生selector見證')
    word = data['tiles'][cell]
    controls = data['native_terrain_control']
    for _ in range(passes):
        tile = word & 1023
        if controls[tile * 4] & 0x60 == 0x20:
            word = (word + 1) & 65535
    if word != s['tile_word']:
        raise ValueError('事件更新見證不符')
    tile = word & 1023
    flags = controls[tile * 4]
    if row['role'] == 'terrain':
        frame = tile + (2 * s['flip'] if flags & 8 else s['cycle'] // 2 if flags & 16 else s['flip'] if flags & 4 else 0)
    elif row['role'] == 'foreground' and flags & 128:
        frame = tile + (2 * s['flip'] if flags & 8 else 0) + 1
    else:
        raise ValueError('沒有對應的原生consumer')
    if frame != row['frame']:
        raise ValueError('原生selector幀不符')


def prepare(catalog_path, pack, maps, sources, original, coverage_path, out):
    if not Path('/.dockerenv').exists() or out.exists() or not out.parent.is_dir():
        raise ValueError('只在Docker使用既有父目錄，拒絕覆寫候選')
    catalog, coverage, manifest = read(catalog_path), read(coverage_path), read(sources / 'manifest.json')
    if (coverage['schema_version'], coverage['kind'], coverage['catalog_sha256']) != (1, 'fd2_hd1_terrain_coverage', sha(catalog_path)):
        raise ValueError('覆蓋盤點與基底清冊不符')
    if [m['map'] for m in coverage['maps']] != list(range(33)) or [m['map'] for m in manifest['maps']] != list(range(33)):
        raise ValueError('33張原生地圖清單不完整')
    slots = {}
    for asset in catalog['assets']:
        if asset['role'] != 'native_terrain_hd_set':
            continue
        for frame, role, name, digest in zip(asset['frame_ids'], asset['semantics'], asset['files'], asset['frame_sha256']):
            for part in (('terrain', 'foreground') if role == 'both' else (role,)):
                key = asset['map_id'], frame, part
                if key in slots:
                    raise ValueError('基底有重複槽')
                slots[key] = {'asset_id': asset['asset_id'], 'map': asset['map_id'], 'frame': frame, 'file': name, 'sha256': digest}
    bindings, by_map, source_cache, seen = [], defaultdict(dict), {}, set()
    for mp in coverage['maps']:
        original_mp = manifest['maps'][mp['map']]
        if mp['bank'] != original_mp['bank'] or mp['map_data'] != original_mp['map_data']:
            raise ValueError('地圖或bank來源綁定不同')
        check_binding(original, mp['bank'])
        data = read(check_binding(maps, mp['map_data']))
        for fr in mp['required']:
            proof = fr.get('reuse')
            if proof is None:
                continue
            key = mp['map'], fr['frame'], fr['role']
            if key in seen:
                raise ValueError('重複的重用目標')
            seen.add(key)
            if fr['status'] != 'missing-art' or key in slots or proof != slots.get((proof['map'], proof['frame'], fr['role'])):
                raise ValueError('重用必須來自已登錄的相同角色，且目標尚缺')
            witness(data, fr)
            pair = []
            for identity in [(mp['map'], fr['frame']), (proof['map'], proof['frame'])]:
                if identity not in source_cache:
                    source_cache[identity] = native(original, manifest, *identity)
                pair.append(source_cache[identity])
            if pair[0] != pair[1] or hashlib.sha256(pair[0]).hexdigest() != fr['native_content_sha256']:
                raise ValueError('不同原生三層內容不得重用HD')
            name = proof['file']
            if Path(name).name != name or sha(pack / name) != proof['sha256']:
                raise ValueError('重用HD路徑或SHA不符')
            picture = Image.open(pack / name).convert('RGBA')
            if picture.size != (144, 144):
                raise ValueError('重用HD尺寸不符')
            mask = pair[0][576:1152]
            remap = pair[0][1152:]
            if any(v not in (0, 1) for v in mask + remap) or any(a and b for a, b in zip(mask, remap)):
                raise ValueError('來源與目的地遮罩必須二元且不重疊')
            if any(picture.getpixel((x, y))[3] != mask[(y // 6) * 24 + x // 6] * 255 for y in range(144) for x in range(144)):
                raise ValueError('HD來源覆蓋與原生遮罩不同')
            old = by_map[mp['map']].get(fr['frame'])
            if old and (old['file'], old['sha256']) != (name, proof['sha256']):
                raise ValueError('同槽兩角色的HD圖源衝突')
            item = {'map': mp['map'], 'frame': fr['frame'], 'role': fr['role'], 'native_content_sha256': fr['native_content_sha256'], 'file': name, 'sha256': proof['sha256'], 'reuse_from': proof, 'selector': fr['selector']}
            bindings.append(item)
            by_map[mp['map']][fr['frame']] = {**item, 'role': 'both' if old else fr['role']}
    if len(bindings) != coverage['summary'].get('reusable_slots', 0) or not bindings:
        raise ValueError('候選少於完整同源盤點')
    additions = []
    existing_ids = {a['asset_id'] for a in catalog['assets']}
    for map_id, frames in sorted(by_map.items()):
        ordered = [frames[i] for i in sorted(frames)]
        names, hashes = [f['file'] for f in ordered], [f['sha256'] for f in ordered]
        asset_id = f'modern.terrain.exact_reuse.map{map_id:02d}.native_hd'
        if asset_id in existing_ids:
            identity = json.dumps([(f['frame'], f['role'], f['native_content_sha256'], f['sha256']) for f in ordered], separators=(',', ':'))
            suffix = hashlib.sha256(identity.encode()).hexdigest()[:12]
            asset_id = f'modern.terrain.exact_reuse.batch{suffix}.map{map_id:02d}.native_hd'
        additions.append({'asset_id': asset_id, 'role': 'native_terrain_hd_set',
            'status': 'runtime_candidate', 'consumer_contract': 'fdshap_native_hd_sparse_v1', 'map_id': map_id,
            'width': 144, 'height': 144, 'frame_count': len(ordered), 'frame_ids': sorted(frames),
            'semantics': [f['role'] for f in ordered], 'files': names, 'frame_sha256': hashes,
            'alpha_contract': 'native_mask_nearest_6', 'cycle_policy': 'native_selector',
            'source_refs': ['docs/data/fd2-hd1-terrain-coverage.json', 'docs/knowledge-base/61-modern-theme-runtime-spec.md']})
    existing_ids = {a['asset_id'] for a in catalog['assets']}
    if any(a['asset_id'] in existing_ids for a in additions):
        raise ValueError('已有不同的同源候選登錄')
    names = set()
    for asset in catalog['assets']:
        for field in ('file', 'master_file'):
            if field in asset:
                names.add(asset[field])
        names.update(asset.get('files', []))
        names.update(asset.get('source_files', []))
    if any(Path(n).name != n or not (pack / n).is_file() for n in names):
        raise ValueError('基底包缺件或路徑不安全')
    out.mkdir()
    for name in sorted(names):
        (out / name).symlink_to(pack.resolve() / name)
    catalog['assets'].extend(additions)
    catalog['private_root'] = out.relative_to(Path(__file__).resolve().parents[1]).as_posix()
    (out / 'catalog.json').write_text(json.dumps(catalog, ensure_ascii=False, indent=2) + '\n')
    result = {'schema_version': 1, 'kind': 'fd2_hd1_exact_terrain_reuse', 'status': 'SOURCE-VERIFIED-CANDIDATE',
        'base_catalog_sha256': sha(catalog_path), 'coverage_sha256': sha(coverage_path),
        'native_manifest_sha256': sha(sources / 'manifest.json'), 'catalog_sha256': sha(out / 'catalog.json'),
        'bindings': bindings, 'new_catalog_sets': len(additions), 'new_image_files': 0,
        'tool_sha256': sha(Path(__file__)), 'scope': '原生三層及已登錄HD來源完全相同；GPU及正式登錄另驗。'}
    (out / 'reuse-manifest.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    return result


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--register', action='store_true')
    for name in ('catalog', 'pack', 'maps', 'sources', 'original', 'coverage', 'out', 'candidate', 'gpu-receipt'):
        p.add_argument('--' + name, type=lambda value: Path(value).resolve())
    a = p.parse_args()
    if a.register:
        if any(x is None for x in (a.catalog, a.candidate, a.gpu_receipt)):
            p.error('--register requires --catalog, --candidate and --gpu-receipt')
        result = register(a.catalog, a.candidate, a.gpu_receipt)
    else:
        if any(x is None for x in (a.catalog, a.pack, a.maps, a.sources, a.original, a.coverage, a.out)):
            p.error('prepare requires --catalog, --pack, --maps, --sources, --original, --coverage and --out')
        result = prepare(a.catalog, a.pack, a.maps, a.sources, a.original, a.coverage, a.out)
    print(json.dumps({k: v for k, v in result.items() if k != 'bindings'}, ensure_ascii=False))


def register(catalog_path, candidate, gpu_path):
    if not Path('/.dockerenv').exists():
        raise ValueError('只在Docker登錄')
    base, proposed, manifest, gpu = read(catalog_path), read(candidate / 'catalog.json'), read(candidate / 'reuse-manifest.json'), read(gpu_path)
    if (sha(catalog_path) != manifest['base_catalog_sha256'] or
            sha(candidate / 'catalog.json') != manifest['catalog_sha256'] or
            gpu.get('status') != 'PASS-EXACT-TERRAIN-REUSE-GPU' or
            gpu.get('catalog_sha256') != manifest['catalog_sha256'] or
            gpu.get('manifest_sha256') != sha(candidate / 'reuse-manifest.json')):
        raise ValueError('登錄來源或GPU收據不符')
    count = len(manifest['bindings'])
    cases = gpu.get('cases', [])
    if (gpu.get('bindings') != count or gpu.get('source_cases') != count * 4 or gpu.get('complete_game_cases') != 2 or
            len(cases) != count * 4 + 2 or sum(bool(c.get('game_draw')) for c in cases) != 2 or
            sum(bool(c.get('f2')) for c in cases) != 2 or gpu.get('save_bytes_unchanged', 0) <= 0):
        raise ValueError('GPU或保存檢查缺項')
    expected = {}
    for binding in manifest['bindings']:
        for mode in ('steady', 'lut', 'black', 'white'):
            expected[f"map{binding['map']:02d}-{binding['frame']:04d}-{binding['role']}-{mode}"] = binding['map'], binding['frame']
    source_cases = [c for c in cases if not c.get('game_draw')]
    if (len(expected) != count * 4 or {c['case'] for c in source_cases} != set(expected) or
            any(c.get('rgba_tolerance') != 2 for c in cases)):
        raise ValueError('GPU消費端或色盤案例缺項')
    if any((c.get('map'), c.get('source_frame')) != expected[c['case']] for c in source_cases):
        raise ValueError('GPU來源圖號不符')
    for c in cases:
        if c.get('file'):
            name = c['file']
            if Path(name).name != name or sha(gpu_path.parent / name) != c.get('sha256'):
                raise ValueError('GPU實圖不符')
    if proposed['assets'][:len(base['assets'])] != base['assets']:
        raise ValueError('舊清冊被改寫')
    additions = proposed['assets'][len(base['assets']):]
    if len(additions) != manifest['new_catalog_sets'] or any(a['role'] != 'native_terrain_hd_set' for a in additions):
        raise ValueError('登錄包含額外角色')
    wanted, actual = {}, {}
    for b in manifest['bindings']:
        wanted[b['map'], b['frame'], b['role']] = b['file'], b['sha256']
    for a in additions:
        for frame, role, name, digest in zip(a['frame_ids'], a['semantics'], a['files'], a['frame_sha256']):
            for part in (('terrain', 'foreground') if role == 'both' else (role,)):
                key = a['map_id'], frame, part
                if key in actual:
                    raise ValueError('重複登錄槽')
                actual[key] = name, digest
    if wanted != actual or len(wanted) != count:
        raise ValueError('登錄範圍與已驗同源槽不同')
    proposed['private_root'] = base['private_root']
    with tempfile.NamedTemporaryFile('w', dir=catalog_path.parent, delete=False, encoding='utf-8') as f:
        f.write(json.dumps(proposed, ensure_ascii=False, indent=2) + '\n')
        temporary = Path(f.name)
    os.chmod(temporary, catalog_path.stat().st_mode & 0o777)
    temporary.replace(catalog_path)
    return {'status': 'REGISTERED-EXACT-TERRAIN-REUSE', 'registered_slots': count,
            'catalog_assets': len(proposed['assets']), 'catalog_sha256': sha(catalog_path),
            'new_image_files': 0}


if __name__ == '__main__':
    main()
