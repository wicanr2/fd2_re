#!/usr/bin/env python3
"""核對固定原生空拍、HD透明來源與同次Game.Draw圖層，不宣稱空拍有核心像素。"""
import argparse
import json
from pathlib import Path

from PIL import Image
from build_fd2_character_battle_hd import DESCRIPTOR, load_native
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate
from verify_fd2_compound_caster_gpu import pinned_native_blank

INVENTORY_SHA256 = 'd019e2cb233d8bbe8d6b6d688221fdc2fecadc78bf39142e5e3c2f6b57375f51'


def verify(receipt, catalog, source, original_pack, inventory, resource):
    if digest(inventory) != INVENTORY_SHA256:
        raise ValueError('固定原始盤點不同')
    inv = json.loads(inventory.read_text())
    if digest(original_pack/'manifest.json') != inv['pack_manifest_sha256']:
        raise ValueError('原包指紋不同')
    native, images = load_native(original_pack, inv, resource)
    blanks = [i for i, im in enumerate(images) if im.getchannel('A').getbbox() is None]
    if not blanks:
        raise ValueError('本資源沒有原生空拍')
    assets = validate(False, catalog)['assets']
    asset = next((a for a in assets if a['role'] == 'battle_animation_hd_set'
                  and a['source_resource'] == resource), None)
    if (asset is None or asset['source_metadata_sha256'] != native['metadata_sha256']
        or len(asset['files']) != len(images)
        or asset['native_header'] != [native['native_header'][f'byte_{i}'] for i in (1, 2, 4)]
        or any(any(a[k] != b[k] for k in DESCRIPTOR)
               for a, b in zip(asset['native_frames'], native['frames']))):
        raise ValueError('清冊與原描述子不同')
    d = json.loads(receipt.read_text())
    phases = ['prelude', 'body', 'counter']
    if (d.get('status') != 'PASS-CHARACTER-FIXTURE-RUNTIME-E1' or d.get('figure') != resource//3
        or d.get('catalog_sha256') != digest(catalog) or d.get('continuations') != 1
        or d.get('battle_dac_steps') != 64 or d.get('map_dac_steps') != 65
        or any(d.get('phase_f2', {}).get(p) != 1 or d.get('phase_hd_draws', {}).get(p, 0) <= 0 for p in phases)):
        raise ValueError('需要完整Game.Draw／F2／返回收據')
    sources = {}
    for index in blanks:
        p = source/asset['files'][index]
        if digest(p) != asset['frame_sha256'][index]:
            raise ValueError('空拍成品SHA不同')
        sources[index] = pinned_native_blank(original_pack, inv, resource, index, p)
    found = {}
    for sample in d.get('source_shots', {}).values():
        if sample.get('phase') != 'body':
            continue
        name = sample['file']
        if Path(name).name != name or digest(receipt.parent/name) != sample['sha256']:
            raise ValueError('GPU圖片路徑或SHA不同')
        with Image.open(receipt.parent/name) as im:
            if im.size != (1920, 1200):
                raise ValueError('需要1920×1200最終GPU')
        for state in (sample.get('before', {}), sample.get('after', {})):
            if not state.get('battle_scene') or not state.get('hd_battle'):
                raise ValueError('需要HD戰鬥圖層')
            for layer in state.get('battle_source_layers') or []:
                index = layer.get('frame')
                if (layer.get('resource') != resource or layer.get('archive', '') not in ('', 'FIGANI.DAT')
                    or layer.get('fill') != -1 or index not in sources):
                    continue
                f = native['frames'][index]
                if (layer['width'], layer['height']) != (f['width'], f['height']):
                    raise ValueError('空拍Draw描述子不同')
                found[index] = {'resource': resource, 'frame': index, 'source': sources[index],
                                'gpu_file': name, 'gpu_sha256': sample['sha256'], 'job': sample.get('job')}
    if set(found) != set(blanks):
        raise ValueError('原生空拍同次Draw不完整')
    return {'status': 'PASS-NATIVE-BLANK-DRAW-GPU', 'resource': resource,
            'receipt_sha256': digest(receipt), 'catalog_sha256': digest(catalog),
            'inventory_sha256': digest(inventory), 'verifier_sha256': digest(Path(__file__)),
            'matches': [found[i] for i in blanks],
            'scope': '固定原包全部空拍、透明HD來源及同次body Draw圖層。無不透明核心聲明；非正常玩家或原版RGB對拍。'}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--resource', type=int, required=True)
    for name in ['receipt', 'catalog', 'source', 'original-pack', 'inventory', 'report']:
        p.add_argument('--'+name, type=Path, required=True)
    a = p.parse_args()
    if a.report.exists():
        raise SystemExit('不覆寫收據')
    try:
        result = verify(a.receipt, a.catalog, a.source, a.original_pack, a.inventory, a.resource)
    except (ValueError, KeyError, OSError, StopIteration) as error:
        raise SystemExit(str(error))
    a.report.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': result['status'], 'frames': len(result['matches'])}))


if __name__ == '__main__':
    main()
