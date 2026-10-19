#!/usr/bin/env python3
"""以原版像素建立隔離六倍診斷包，驗證HD戰鬥consumer；不製作新美術。"""

import argparse
import copy
import hashlib
import json
import os
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import names
from validate_modern_theme_catalog import ROOT, validate


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--pack', required=True, type=Path)
    parser.add_argument('--inventory', required=True, type=Path)
    parser.add_argument('--resources', required=True, help='逗號分隔的FIGANI資源編號')
    parser.add_argument('--out', required=True, type=Path)
    args = parser.parse_args()
    resources = [int(value) for value in args.resources.split(',')]
    if len(set(resources)) != len(resources) or not resources or any(value < 0 or value >= 409 for value in resources):
        raise SystemExit('資源編號重複或越界')
    out = args.out.resolve()
    if out.exists() or not out.is_relative_to(ROOT) or out.parent.stat().st_uid != os.getuid():
        raise SystemExit('使用儲存庫內、本人擁有的新診斷目錄')
    inventory = json.loads(args.inventory.read_text())
    if inventory.get('status') != 'PASS-SOURCE-INVENTORY' or not inventory['archive_reconstruction']['matches_pinned_original']:
        raise SystemExit('原始圖庫未通過核對')
    if digest(args.pack/'manifest.json') != inventory['pack_manifest_sha256']:
        raise SystemExit('分離包與盤點不同')
    palette_path = args.pack/'palette/fdother_000.json'
    palette_doc = json.loads(palette_path.read_text())
    # 固定版本已由盤點核對；palette仍保留原始來源指紋與逐通道範圍。
    if palette_doc['source']['sha256'] != 'a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce':
        raise SystemExit('色盤來源不符')
    dac = palette_doc['dac_6bit_components']
    if len(dac) != 768 or any(type(v) is not int or not 0 <= v <= 63 for v in dac):
        raise SystemExit('DAC色盤錯誤')
    rgb = [tuple((v << 2) | (v >> 4) for v in dac[i:i+3]) for i in range(0, 768, 3)]
    base_path = ROOT/'remake/assets/themes/modern/catalog.json'
    base = validate(False, base_path)
    if any(asset['role'] == 'battle_animation_hd_set' and asset['source_resource'] in resources for asset in base['assets']):
        raise SystemExit('所選診斷動畫已登錄，不覆寫')
    # 全部圖源先驗證，避免失敗時留下一份看似可用的清冊。
    inputs = []
    for resource in resources:
        row = next(value for value in inventory['resources'] if value['resource'] == resource)
        folder = args.pack/'animations'/f'FIGANI_{resource:03d}'
        metadata = folder/'animation.json'
        if row['status'] != 'decoded' or digest(metadata) != row['metadata_sha256']:
            raise SystemExit('動畫metadata與盤點不符')
        doc = json.loads(metadata.read_text())
        if len(doc['frames']) != len(row['frames']):
            raise SystemExit('動畫幀數不符')
        frames = []
        for descriptor, expected in zip(doc['frames'], row['frames']):
            with Image.open(folder/descriptor['path']) as source, Image.open(folder/descriptor['mask_path']) as mask:
                pixels, coverage = source.tobytes(), mask.convert('L').tobytes()
                if source.mode != 'P' or source.size != (expected['width'], expected['height']) or mask.size != source.size:
                    raise SystemExit('動畫像素形狀不符')
                # 盤點欄位保存原始PNG檔案雜湊，並非解碼後的像素buffer。
                if digest(folder/descriptor['path']) != expected['pixels_sha256'] or digest(folder/descriptor['mask_path']) != expected['mask_sha256']:
                    raise SystemExit('動畫像素／mask與盤點不符')
                picture = Image.new('RGBA', source.size)
                picture.putdata([rgb[index] + (255 if alpha else 0,) for index, alpha in zip(pixels, coverage)])
                frames.append(picture.resize((picture.width*6, picture.height*6), Image.Resampling.NEAREST))
        inputs.append((resource, row, frames))
    out.mkdir()
    pack = out/'pack'
    pack.mkdir()
    for name in sorted(names(base)):
        source = ROOT/base['private_root']/name
        if Path(name).name != name or not source.is_file():
            raise SystemExit('基底清冊缺件或路徑錯誤')
        (pack/name).symlink_to(os.path.relpath(source, pack))
    catalog = copy.deepcopy(base)
    catalog['private_root'] = str(pack.relative_to(ROOT))
    for resource, row, frames in inputs:
        files = []
        for index, picture in enumerate(frames):
            name = f'diagnostic-figani-{resource:03d}-f{index:02d}.png'
            picture.save(pack/name)
            files.append(name)
        catalog['assets'].append({
            'asset_id': f'diagnostic.figani.{resource:03d}.hd6', 'role': 'battle_animation_hd_set',
            'status': 'runtime_candidate', 'files': files, 'width': 1920, 'height': 1200,
            'frame_count': len(files), 'frame_sha256': [digest(pack/name) for name in files],
            'consumer_contract': 'native_figani_rgba_hd6_v1', 'alpha_contract': 'coverage',
            'source_refs': [f'asset:FIGANI_{resource:03d}', 'docs/data/fd2-reference-files.json'],
            'source_archive': 'FIGANI.DAT', 'source_resource': resource,
            'source_metadata_sha256': row['metadata_sha256'],
            'native_header': [row['native_header'][key] for key in ['byte_1', 'byte_2', 'byte_4']],
            'native_frames': [{key: frame[key] for key in ['x', 'y', 'width', 'height', 'delay_native', 'raw_byte_4', 'raw_byte_5', 'raw_byte_7']} for frame in row['frames']],
            'cycle_policy': 'native_descriptor_key_poses', 'layout_policy': 'contain_native_descriptor_hd6_candidate',
            'source_files': [files[0]], 'source_sha256': [digest(pack/files[0])],
        })
    path = out/'catalog.json'
    path.write_text(json.dumps(catalog, ensure_ascii=False, indent=2)+'\n')
    validate(True, path)
    runtime = out/'runtime'
    (runtime/'assets/themes/modern').mkdir(parents=True)
    for source in (ROOT/'remake/assets').iterdir():
        if source.name != 'themes':
            (runtime/'assets'/source.name).symlink_to(os.path.relpath(source, runtime/'assets'))
    (runtime/'assets/themes/modern/catalog.json').symlink_to(os.path.relpath(path, runtime/'assets/themes/modern'))
    report = {'status': 'PASS-DIAGNOSTIC-ONLY', 'resources': resources, 'frames': sum(len(frames) for _, _, frames in inputs),
              'scope': '原版像素最近鄰六倍，不是新增HD美術或原版oracle；不改正式清冊。',
              'catalog_sha256': digest(path), 'base_catalog_sha256': digest(base_path),
              'inventory_sha256': digest(args.inventory), 'palette_sha256': digest(palette_path),
              'tool_sha256': digest(Path(__file__))}
    (out/'diagnostic.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(report, ensure_ascii=False))


if __name__ == '__main__':
    main()
