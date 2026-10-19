#!/usr/bin/env python3
"""核對正式戰鬥HD圖源覆蓋；資料登錄不提升為完整caller或玩家驗證。"""
import argparse
import json
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import ROOT, validate


FIELDS = ['x', 'y', 'width', 'height', 'delay_native', 'raw_byte_4', 'raw_byte_5', 'raw_byte_7']


def check_pngs(asset, source, full_masks=None):
    for index, (name, sha, frame) in enumerate(zip(asset['files'], asset['frame_sha256'], asset['native_frames'])):
        path = source/name
        if digest(path) != sha: raise ValueError('已登錄成品SHA不同')
        with Image.open(path) as im:
            if im.mode != 'RGBA' or im.size != (frame['width']*6, frame['height']*6):
                raise ValueError('已登錄成品形狀或alpha不同')
            low, high = im.getchannel('A').getextrema()
            full_mask = asset['source_archive'] == 'FDOTHER.DAT' and full_masks is not None and full_masks[index]
            if high != 255 or low != 0 and not full_mask:
                raise ValueError('已登錄成品缺少透明背景或不透明核心')


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['inventory', 'original-pack', 'catalog', 'source', 'report']:
        p.add_argument('--'+name, required=True, type=Path)
    args = p.parse_args()
    if args.report.exists(): raise SystemExit('不覆寫既有收據')
    inventory = json.loads(args.inventory.read_text())
    if inventory.get('status') != 'PASS-SOURCE-INVENTORY' or digest(args.original_pack/'manifest.json') != inventory['pack_manifest_sha256']:
        raise SystemExit('原版盤點或包指紋不同')
    if [r['resource'] for r in inventory['resources']] != list(range(409)):
        raise SystemExit('盤點未完整涵蓋409個資源位置')
    catalog = validate(False, args.catalog)
    stage = 'registered-worktree' if args.catalog.resolve() == ROOT/'remake/assets/themes/modern/catalog.json' else 'candidate'
    registered = {a['source_resource']: a for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set'}
    rows = []
    for row in inventory['resources']:
        resource = row['resource']; asset = registered.get(resource)
        if asset is not None:
            if row['status'] != 'decoded' or asset['source_metadata_sha256'] != row['metadata_sha256']:
                raise SystemExit('已登錄資源不在原始解碼盤點或metadata不同')
            metadata = args.original_pack/'animations'/f'FIGANI_{resource:03d}'/'animation.json'
            if digest(metadata) != row['metadata_sha256']:
                raise SystemExit('目前原版metadata不同')
            expected = [{key: frame[key] for key in FIELDS} for frame in row['frames']]
            if asset['native_frames'] != expected or asset['frame_count'] != len(expected) or asset['native_header'] != [row['native_header'][k] for k in ['byte_1', 'byte_2', 'byte_4']]:
                raise SystemExit('已登錄描述子或header不同')
            check_pngs(asset, args.source)
        rows.append({'resource': resource, 'source_status': row['status'], 'native_frames': len(row['frames']),
            'registered_hd_frames': asset['frame_count'] if asset else 0,
            'asset_status': ('REGISTERED-ASSET' if stage == 'registered-worktree' else 'CATALOG-CANDIDATE') if asset else 'NOT-REGISTERED' if row['status'] == 'decoded' else 'SOURCE-EMPTY' if not row['frames'] and row['status'].startswith('empty') else 'SOURCE-BLOCKED',
            'common_background_header_eligible': row.get('native_header', {}).get('byte_1') == 0,
            'runtime_caller_status': 'NOT-INFERRED-FROM-ASSETS'})
    effects = []
    for asset in catalog['assets']:
        if asset['role'] != 'battle_effect_hd_set': continue
        resource = asset['source_resource']; metadata = args.original_pack/'animations'/f'FDOTHER_{resource:03d}'/'animation.json'
        if digest(metadata) != asset['source_metadata_sha256']: raise SystemExit('共用效果原版metadata不同')
        doc = json.loads(metadata.read_text()); expected = [{key: frame[key] for key in FIELDS} for frame in doc['frames']]
        if asset['native_frames'] != expected: raise SystemExit('共用效果描述子不同')
        masks = []
        for frame in doc['frames']:
            path = metadata.parent/frame['mask_path']
            with Image.open(path) as im:
                if im.size != (frame['width'], frame['height']): raise SystemExit('原生效果遮罩形狀不同')
                full = im.convert('L').getextrema()[0] != 0
            masks.append({'sha256': digest(path), 'full_source_mask': full})
        check_pngs(asset, args.source, [m['full_source_mask'] for m in masks])
        effects.append({'resource': resource, 'registered_hd_frames': asset['frame_count'],
            'asset_status': 'REGISTERED-ASSET' if stage == 'registered-worktree' else 'CATALOG-CANDIDATE', 'native_masks': masks})
    decoded = [r for r in rows if r['source_status'] == 'decoded']
    report = {'status': 'PASS-BATTLE-HD-ASSET-COVERAGE', 'inventory_sha256': digest(args.inventory),
        'catalog_sha256': digest(args.catalog), 'catalog_stage': stage, 'tool_sha256': digest(Path(__file__)), 'catalog_assets': len(catalog['assets']),
        'figani': {'resource_positions': len(rows), 'decoded_resources': len(decoded),
            'native_frames': sum(r['native_frames'] for r in decoded), 'registered_resources': len(registered),
            'registered_hd_frames': sum(r['registered_hd_frames'] for r in rows),
            'remaining_decoded_resources': [r['resource'] for r in decoded if not r['registered_hd_frames']],
            'source_blocked_resources': [r['resource'] for r in rows if r['asset_status'] == 'SOURCE-BLOCKED'], 'resources': rows},
        'registered_fdother_effects': effects,
        'opaque_effect_contract': '沿modern_hd_battle_assets.go：僅FDOTHER原生Mask每個像素均有來源寫入時，容許成品沒有透明像素；FIGANI仍須透明背景及255核心。',
        'scope': '完整FIGANI來源盤點對正式catalog、原版metadata及實際HD PNG的資料覆蓋；非原版oracle。'
                 'header資格不等於caller已接線，FDOTHER只核已登錄效果，不假定四種效果涵蓋全部共同效果。'
                 '一般玩家、所有caller、其他法術／死亡／背景、效能與平台須另有證據。',
        'full_battle_hd_complete': False}
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({key: value for key, value in report['figani'].items() if key not in ['resources', 'remaining_decoded_resources']}, ensure_ascii=False))


if __name__ == '__main__': main()
