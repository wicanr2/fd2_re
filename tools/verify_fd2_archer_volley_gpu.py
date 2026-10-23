#!/usr/bin/env python3
"""核對固定來源的投射物、爪光、地塵與空拍；本體保持1000點契約。"""
import argparse
import json
from pathlib import Path

from PIL import Image
from build_fd2_character_battle_hd import DESCRIPTOR, load_native
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate
from verify_fd2_ares_battle_gpu import matches
from verify_fd2_compound_caster_gpu import make_native_occlusion_test, pinned_native_blank


INVENTORY_SHA256 = 'd019e2cb233d8bbe8d6b6d688221fdc2fecadc78bf39142e5e3c2f6b57375f51'
NATIVE_COUNTS = [77, 311, 422, 437, 443, 390, 123, 33, 0]
PROFILES = {
    23: {'resource': 70, 'header': {'byte_1': 1, 'byte_2': 16, 'byte_4': 1},
         'frames': 25, 'first': 16, 'blanks': [24], 'counts': NATIVE_COUNTS},
    40: {'resource': 121, 'header': {'byte_1': 1, 'byte_2': 11, 'byte_4': 1},
         'frames': 18, 'first': 11, 'blanks': [], 'counts': [124, 393, 481, 394, 355, 342, 342]},
    45: {'resource': 136, 'header': {'byte_1': 1, 'byte_2': 17, 'byte_4': 4},
         'frames': 22, 'first': 17, 'blanks': [17, 21], 'counts': [0, 5148, 8014, 1579, 0]},
    58: {'resource': 175, 'header': {'byte_1': 1, 'byte_2': 11, 'byte_4': 1},
         'frames': 18, 'first': 11, 'blanks': [], 'counts': [124, 393, 481, 394, 355, 342, 342]},
    63: {'resource': 190, 'header': {'byte_1': 1, 'byte_2': 17, 'byte_4': 4},
         'frames': 22, 'first': 17, 'blanks': [17, 21], 'counts': [0, 5148, 8014, 1579, 0]},
    72: {'resource': 217, 'header': {'byte_1': 1, 'byte_2': 12, 'byte_4': 4},
         'frames': 15, 'first': 12, 'blanks': [14], 'counts': [4331, 706, 0],
         'guard_missing_frames': [12, 13, 14]},
    88: {'resource': 265, 'header': {'byte_1': 1, 'byte_2': 13, 'byte_4': 4},
         'frames': 21, 'first': 13, 'blanks': [13, 20],
         'counts': [0, 330, 5868, 1630, 1014, 404, 252, 0],
         'guard_missing_frames': list(range(13, 21))},
    89: {'resource': 268, 'header': {'byte_1': 1, 'byte_2': 15, 'byte_4': 4},
         'frames': 20, 'first': 15, 'blanks': [19],
         'counts': [361, 4203, 5440, 850, 0],
         'guard_missing_frames': list(range(15, 20))},
    114: {'resource': 343, 'header': {'byte_1': 1, 'byte_2': 24, 'byte_4': 4},
          'frames': 35, 'first': 24, 'blanks': [34],
          'counts': [623, 5373, 7331, 7974, 8313, 3815, 1990, 976, 381, 143, 0],
          'guard_missing_frames': list(range(24, 35))},
    121: {'resource': 364, 'header': {'byte_1': 0, 'byte_2': 5, 'byte_4': 2},
          'frames': 5, 'first': 0, 'blanks': [], 'counts': [2553, 381, 135, 72, 6],
          'phases': ['prelude', 'body', 'counter'], 'guard_missing_frames': list(range(5)),
          'auxiliary_blanks': [{'resource': 363, 'frame': 0}]},
    124: {'resource': 373, 'header': {'byte_1': 0, 'byte_2': 22, 'byte_4': 2},
          'frames': 22, 'first': 6, 'selected_frames': [6, 7, 8], 'blanks': [],
          'counts': [1319, 592, 223], 'phases': ['prelude', 'body', 'counter'],
          'guard_missing_frames': [6, 7, 8]},
}


def verify(receipt, catalog, source, original_pack, inventory, figure=23):
    if figure not in PROFILES:
        raise ValueError('飛矢人物沒有固定來源契約')
    profile = PROFILES[figure]
    resource, first, last = profile['resource'], profile['first'], profile['frames']
    indices = profile.get('selected_frames', list(range(first, last)))
    if digest(inventory) != INVENTORY_SHA256:
        raise ValueError('飛矢固定盤點不同')
    proof = json.loads(inventory.read_text())
    if (proof.get('status') != 'PASS-SOURCE-INVENTORY'
        or proof.get('archive_reconstruction', {}).get('matches_pinned_original') is not True
        or digest(original_pack/'manifest.json') != proof['pack_manifest_sha256']):
        raise ValueError('飛矢原包或重建證據不同')
    native, originals = load_native(original_pack, proof, resource)
    if native['native_header'] != profile['header'] or len(originals) != last:
        raise ValueError('飛矢原始分段不同')
    counts = [sum(a != 0 for a in originals[i].getchannel('A').getdata()) for i in indices]
    if counts != profile['counts']:
        raise ValueError('原飛矢或透明空拍不符')
    doc = validate(False, catalog)
    entries = {a['source_resource']: a for a in doc['assets'] if a['role'] == 'battle_animation_hd_set'}
    effects = {a['source_resource']: a for a in doc['assets'] if a['role'] == 'battle_effect_hd_set'}
    asset = entries.get(resource)
    if (asset is None or asset['source_metadata_sha256'] != native['metadata_sha256']
        or asset['native_header'] != [native['native_header'][f'byte_{i}'] for i in (1, 2, 4)] or len(asset['files']) != last
        or any(any(f[k] != raw[k] for k in DESCRIPTOR) for f, raw in zip(asset['native_frames'], native['frames']))):
        raise ValueError('清冊不是固定原始弓手圖庫')
    d = json.loads(receipt.read_text())
    phases = profile.get('phases', ['prelude', 'departure', 'scroll-in', 'body', 'scroll-out', 'counter'])
    if (d.get('status') != 'PASS-CHARACTER-FIXTURE-RUNTIME-E1' or d.get('figure') != figure
        or d.get('catalog_sha256') != digest(catalog) or d.get('continuations') != 1
        or d.get('battle_dac_steps') != 64 or d.get('map_dac_steps') != 65
        or any(d.get('phase_f2', {}).get(p) != 1 or d.get('phase_hd_draws', {}).get(p, 0) <= 0 for p in phases)):
        raise ValueError('需要完整弓手Game.Draw／F2／返回收據')
    points, sources = {}, {}
    for index in indices:
        path = source/asset['files'][index]
        frame = native['frames'][index]
        if digest(path) != asset['frame_sha256'][index]:
            raise ValueError('飛矢成品SHA不同')
        sources[index] = {'file': asset['files'][index], 'sha256': digest(path)}
        with Image.open(path) as im:
            if im.mode != 'RGBA' or im.size != (frame['width']*6, frame['height']*6):
                raise ValueError('飛矢成品形狀不同')
            if index in profile['blanks']:
                sources[index]['blank'] = pinned_native_blank(original_pack, proof, resource, index, path)
            else:
                if im.getchannel('A').getextrema() != (0, 255):
                    raise ValueError('非空飛矢須保留透明背景與不透明核心')
                points[index] = [(i % im.width, i // im.width, *pixel) for i, pixel in enumerate(im.getdata()) if pixel[3] >= 250]
                if not points[index]:
                    raise ValueError('飛矢沒有可核對核心')
    auxiliary = {}
    for blank in profile.get('auxiliary_blanks', []):
        other, index = blank['resource'], blank['frame']
        row, _ = load_native(original_pack, proof, other)
        entry = entries.get(other)
        if (entry is None or entry['source_metadata_sha256'] != row['metadata_sha256']
            or entry['native_header'] != [row['native_header'][f'byte_{i}'] for i in (1, 2, 4)]
            or len(entry['files']) != len(row['frames'])
            or len(entry['native_frames']) != len(row['frames'])
            or any(any(f[k] != raw[k] for k in DESCRIPTOR) for f, raw in zip(entry['native_frames'], row['frames']))):
            raise ValueError('附屬空拍清冊不是固定原始圖庫')
        path = source/entry['files'][index]
        if digest(path) != entry['frame_sha256'][index]:
            raise ValueError('附屬空拍成品SHA不同')
        auxiliary[(other, index)] = {'file': entry['files'][index], 'sha256': digest(path),
            'blank': pinned_native_blank(original_pack, proof, other, index, path)}
    occluded = make_native_occlusion_test(entries, source, original_pack, proof, effects)
    found = {}
    blank_draws = {}
    for sample in d.get('source_shots', {}).values():
        if sample.get('phase') != 'body' and (not auxiliary or sample.get('phase') not in phases):
            continue
        name = sample['file']
        if Path(name).name != name or digest(receipt.parent/name) != sample['sha256']:
            raise ValueError('GPU圖片路徑或SHA不同')
        with Image.open(receipt.parent/name) as im:
            if im.size != (1920, 1200):
                raise ValueError('需要1920×1200最終GPU')
            rgb = im.convert('RGB').load()
            for state in (sample.get('before', {}), sample.get('after', {})):
                if not state.get('battle_scene') or not state.get('hd_battle'):
                    raise ValueError('飛矢不是HD戰鬥視窗')
                layers = state.get('battle_source_layers') or []
                for layer_index, layer in enumerate(layers):
                    index = layer['frame']
                    key = (layer['resource'], index)
                    if (key in auxiliary and layer.get('archive', '') in ('', 'FIGANI.DAT')
                        and layer.get('fill') == -1):
                        blank = auxiliary[key]
                        if (layer['width'], layer['height']) != (blank['blank']['width'], blank['blank']['height']):
                            raise ValueError('附屬空拍Draw描述子不同')
                        blank_draws.setdefault(key, {'resource': key[0], 'frame': key[1], 'source': blank,
                            'gpu_file': name, 'gpu_sha256': sample['sha256'], 'job': sample.get('job'),
                            'phase': sample['phase'], 'scope': '固定全透明來源與同次Draw圖層，無不透明核心聲明'})
                    if sample.get('phase') != 'body':
                        continue
                    if (layer['resource'] != resource or layer.get('archive', '') not in ('', 'FIGANI.DAT')
                        or layer.get('fill') != -1 or not isinstance(index, int) or index not in sources):
                        continue
                    frame = native['frames'][index]
                    if (layer['width'], layer['height']) != (frame['width'], frame['height']):
                        raise ValueError('飛矢Draw描述子不同')
                    result = {'resource': resource, 'frame': index, 'source': sources[index], 'gpu_file': name,
                              'gpu_sha256': sample['sha256'], 'job': sample.get('job'), 'phase': 'body'}
                    if index in profile['blanks']:
                        result['scope'] = '固定全透明來源與同次Draw圖層，無不透明核心聲明'
                    else:
                        x, y = layer['x']*6, layer['y']*6
                        clip = layer.get('clip')
                        visible = [p for p in points[index] if 0 <= p[0]+x < 1920 and 0 <= p[1]+y < 1200
                                   and (not clip or clip['Min']['X']*6 <= p[0]+x < clip['Max']['X']*6
                                        and clip['Min']['Y']*6 <= p[1]+y < clip['Max']['Y']*6)
                                   and not occluded(p[0]+x, p[1]+y, layers[layer_index+1:])]
                        if not visible or not matches(rgb, visible, x, y, 1920, 1200):
                            continue
                        result.update(eligible_pixels=len(points[index]), checked_pixels=len(visible),
                                      world_origin=[x, y], clip=clip, native_opaque_pixels=profile['counts'][indices.index(index)])
                    found.setdefault(index, result)
    if set(found) != set(indices):
        raise ValueError('飛矢或空拍最終Draw不完整：'+str(sorted(set(indices)-set(found))))
    if set(blank_draws) != set(auxiliary):
        raise ValueError('附屬空拍最終Draw不完整：'+str(sorted(set(auxiliary)-set(blank_draws))))
    return {'status': 'PASS-PINNED-ARCHER-VOLLEY-GPU', 'figure': figure, 'resource': resource,
            'receipt_sha256': digest(receipt), 'catalog_sha256': digest(catalog), 'inventory_sha256': digest(inventory),
            'original_manifest_sha256': digest(original_pack/'manifest.json'), 'verifier_sha256': digest(Path(__file__)),
            'native_metadata_sha256': native['metadata_sha256'], 'native_opaque_counts': profile['counts'],
            'matches': [found[i] for i in sorted(found)],
            'auxiliary_blank_draws': [blank_draws[k] for k in sorted(blank_draws)],
            'scope': f'固定FIGANI{resource}的明示幀集合{indices}逐幀全可見高覆蓋點；固定原空幀另驗透明來源／同次Draw。'
                     '不抽樣、不更動人物1000點契約。'
                     '沿已核來源排除clip及前景遮擋，非原版RGB parity、合法caller新增證據或PLAYER-E2。'}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--figure', type=int, choices=sorted(PROFILES), default=23)
    for name in ['receipt', 'catalog', 'source', 'original-pack', 'inventory', 'report']:
        p.add_argument('--'+name, required=True, type=Path)
    args = p.parse_args()
    if args.report.exists():
        raise SystemExit('不覆寫既有收據')
    try:
        report = verify(args.receipt, args.catalog, args.source, args.original_pack, args.inventory, args.figure)
    except (ValueError, KeyError, OSError) as error:
        raise SystemExit(str(error))
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'frames': len(report['matches'])}))


if __name__ == '__main__':
    main()
