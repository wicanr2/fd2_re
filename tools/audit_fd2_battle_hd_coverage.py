#!/usr/bin/env python3
"""核對正式戰鬥HD圖源覆蓋；資料登錄不提升為完整caller或玩家驗證。"""
import argparse
import json
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import ROOT, validate
from audit_fd2_player9_map_source import audit as audit_lmi_sources
from audit_fd2_battle_background_sources import audit as audit_background_sources


FIELDS = ['x', 'y', 'width', 'height', 'delay_native', 'raw_byte_4', 'raw_byte_5', 'raw_byte_7']


def check_pngs(asset, source, full_masks=None, empty_masks=None):
    for index, (name, sha, frame) in enumerate(zip(asset['files'], asset['frame_sha256'], asset['native_frames'])):
        path = source/name
        if digest(path) != sha: raise ValueError('已登錄成品SHA不同')
        with Image.open(path) as im:
            if im.mode != 'RGBA' or im.size != (frame['width']*6, frame['height']*6):
                raise ValueError('已登錄成品形狀或alpha不同')
            low, high = im.getchannel('A').getextrema()
            full_mask = asset['source_archive'] == 'FDOTHER.DAT' and full_masks is not None and full_masks[index]
            empty_allowed = asset['source_archive'] == 'FIGANI.DAT' or (
                asset['source_archive'] == 'FDOTHER.DAT' and asset['source_resource'] == 44
                and asset['frame_count'] == 31 and index == 30)
            native_empty = empty_allowed and empty_masks is not None and empty_masks[index]
            if native_empty and high != 0:
                raise ValueError('原生空白幀新增了墨跡')
            if not native_empty and (high != 255 or low != 0 and not full_mask):
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
    manifest = json.loads((args.original_pack/'manifest.json').read_text())
    files = {row['path']: row for row in manifest['assets']}
    pinned_effect = next(row for row in json.loads((ROOT/'docs/data/fd2-reference-files.json').read_text())['files'] if row['file'] == 'FDOTHER.DAT')
    if next(row for row in manifest['source_set'] if row['file'] == 'FDOTHER.DAT') != pinned_effect:
        raise SystemExit('共用效果來源版本不同')

    def checked_effect(path, resource, index=None):
        row = files.get(path.relative_to(args.original_pack).as_posix())
        if (row is None or row.get('source_file') != 'FDOTHER.DAT' or row.get('source_resource') != resource
                or digest(path) != row['sha256'] or index is not None and row.get('source_frame') != index):
            raise SystemExit('共用效果原版圖源SHA或定位不同')
        return row
    catalog = validate(False, args.catalog)
    stage = 'registered-worktree' if args.catalog.resolve() == ROOT/'remake/assets/themes/modern/catalog.json' else 'candidate'
    registered = {a['source_resource']: a for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set'}
    rows = []
    for row in inventory['resources']:
        resource = row['resource']; asset = registered.get(resource)
        empty_frames = []
        if asset is not None:
            if row['status'] != 'decoded' or asset['source_metadata_sha256'] != row['metadata_sha256']:
                raise SystemExit('已登錄資源不在原始解碼盤點或metadata不同')
            metadata = args.original_pack/'animations'/f'FIGANI_{resource:03d}'/'animation.json'
            if digest(metadata) != row['metadata_sha256']:
                raise SystemExit('目前原版metadata不同')
            expected = [{key: frame[key] for key in FIELDS} for frame in row['frames']]
            if asset['native_frames'] != expected or asset['frame_count'] != len(expected) or asset['native_header'] != [row['native_header'][k] for k in ['byte_1', 'byte_2', 'byte_4']]:
                raise SystemExit('已登錄描述子或header不同')
            doc = json.loads(metadata.read_text()); masks = []
            for index, frame in enumerate(doc['frames']):
                path = metadata.parent/frame['mask_path']
                if digest(path) != row['frames'][index]['mask_sha256']:
                    raise SystemExit('FIGANI原生遮罩SHA不同')
                with Image.open(path) as im:
                    if im.size != (frame['width'], frame['height']): raise SystemExit('FIGANI原生遮罩形狀不同')
                    empty = im.convert('L').getextrema() == (0, 0)
                masks.append(empty)
                if empty: empty_frames.append({'frame': index, 'mask_sha256': digest(path)})
            check_pngs(asset, args.source, empty_masks=masks)
        rows.append({'resource': resource, 'source_status': row['status'], 'native_frames': len(row['frames']),
            'registered_hd_frames': asset['frame_count'] if asset else 0,
            'asset_status': ('REGISTERED-ASSET' if stage == 'registered-worktree' else 'CATALOG-CANDIDATE') if asset else 'NOT-REGISTERED' if row['status'] == 'decoded' else 'SOURCE-EMPTY' if not row['frames'] and row['status'].startswith('empty') else 'SOURCE-BLOCKED',
            'common_background_header_eligible': row.get('native_header', {}).get('byte_1') == 0,
            'verified_empty_source_frames': empty_frames,
            'runtime_caller_status': 'NOT-INFERRED-FROM-ASSETS'})
    effects = []
    for asset in catalog['assets']:
        if asset['role'] != 'battle_effect_hd_set': continue
        resource = asset['source_resource']; metadata = args.original_pack/'animations'/f'FDOTHER_{resource:03d}'/'animation.json'
        checked_effect(metadata, resource)
        if digest(metadata) != asset['source_metadata_sha256']: raise SystemExit('共用效果原版metadata不同')
        doc = json.loads(metadata.read_text()); expected = [{key: frame[key] for key in FIELDS} for frame in doc['frames']]
        if (doc.get('source') != {'file': 'FDOTHER.DAT', 'resource': resource} or asset['native_frames'] != expected
                or asset['frame_count'] != len(expected)
                or asset['native_header'] != [doc['native_header'][k] for k in ['byte_1', 'byte_2', 'byte_4']]):
            raise SystemExit('共用效果定位、描述子、幀數或header不同')
        masks = []
        for index, frame in enumerate(doc['frames']):
            if any(Path(frame[key]).name != frame[key] for key in ['path', 'mask_path']):
                raise SystemExit('共用效果圖源路徑越界')
            path = metadata.parent/frame['mask_path']
            mask_row = checked_effect(path, resource, index)
            checked_effect(metadata.parent/frame['path'], resource, index)
            with Image.open(path) as im, Image.open(metadata.parent/frame['path']) as pixels:
                if im.size != (frame['width'], frame['height']) or pixels.mode != 'P' or pixels.size != im.size:
                    raise SystemExit('原生效果PNG或遮罩形狀不同')
                full = im.convert('L').getextrema()[0] != 0
                empty = im.convert('L').getextrema()[1] == 0
            masks.append({'sha256': mask_row['sha256'], 'full_source_mask': full, 'empty_source_mask': empty})
        check_pngs(asset, args.source, [m['full_source_mask'] for m in masks], [m['empty_source_mask'] for m in masks])
        effects.append({'resource': resource, 'registered_hd_frames': asset['frame_count'],
            'asset_status': 'REGISTERED-ASSET' if stage == 'registered-worktree' else 'CATALOG-CANDIDATE', 'native_masks': masks})
    lmi_assets=[a for a in catalog['assets'] if a['role']=='map_overlay_hd_set']
    lmi_overlays=[]
    if lmi_assets:
        ranges={}
        for asset in lmi_assets:
            ranges.setdefault(asset['source_resource'],set()).update(f['index'] for f in asset['native_lmi_frames'])
        source_report,_=audit_lmi_sources(args.original_pack,{res:sorted(indices) for res,indices in sorted(ranges.items())})
        banks={b['resource']:b for b in source_report['resources']}
        for asset in lmi_assets:
            bank=banks[asset['source_resource']]
            if bank['metadata_sha256']!=asset['source_metadata_sha256']:
                raise SystemExit('LMI原始metadata不同')
            native={f['index']:f for f in bank['frames']}
            for name,sha,frame in zip(asset['files'],asset['frame_sha256'],asset['native_lmi_frames']):
                expected={k:native[frame['index']][k] for k in ['index','width','height','pixels_sha256']}
                if frame!=expected or digest(args.source/name)!=sha:
                    raise SystemExit('LMI原始索引或已登錄SHA不同')
                with Image.open(args.source/name) as im:
                    if im.mode!='RGBA' or im.size!=(frame['width']*6,frame['height']*6):
                        raise SystemExit('LMI已登錄尺寸不同')
                    low,high=im.getchannel('A').getextrema()
                    if low!=0 or high==0:raise SystemExit('LMI已登錄缺少透明背景或墨跡')
            lmi_overlays.append({'asset_id':asset['asset_id'],'resource':asset['source_resource'],
                'registered_hd_frames':asset['frame_count'],'indices':[f['index'] for f in asset['native_lmi_frames']],
                'raw_sha256':bank['raw_sha256'],'runtime_caller_status':'NOT-INFERRED-FROM-ASSETS'})
    surfaces = []
    surface_assets = [a for a in catalog['assets'] if a['role'] == 'battle_surface_hd']
    if surface_assets:
        native_surfaces, _ = audit_background_sources(args.original_pack)
        for asset in surface_assets:
            archive, resource = asset['source_archive'], asset['source_resource']
            frame = native_surfaces['banks'][archive[:-4]]['entries'][resource]
            descriptor = dict(x=0,y=0,width=frame['width'],height=frame['height'],delay_native=0,raw_byte_4=0,raw_byte_5=0,raw_byte_7=0)
            if (frame.get('visible_pixels',0) == 0 or asset['source_metadata_sha256'] != frame['metadata_sha256']
                    or asset['native_frames'] != [descriptor] or digest(args.source/asset['file']) != asset['sha256']):
                raise SystemExit('背景或台座來源／原框／成品SHA不同')
            with Image.open(args.source/asset['file']) as picture:
                if picture.mode != 'RGBA' or picture.size != (frame['width']*6,frame['height']*6):
                    raise SystemExit('背景或台座HD尺寸不同')
                low, high = picture.getchannel('A').getextrema()
                full = frame['visible_pixels'] == frame['width']*frame['height']
                if high != 255 or (full and low != 255) or (not full and low != 0):
                    raise SystemExit('背景或台座HD透明度契約不同')
            surfaces.append({'asset_id':asset['asset_id'],'archive':archive,'resource':resource,
                'metadata_sha256':frame['metadata_sha256'],'native_size':[frame['width'],frame['height']],
                'compiled_sha256':asset['sha256'],'runtime_caller_status':'NOT-INFERRED-FROM-ASSETS'})
    decoded = [r for r in rows if r['source_status'] == 'decoded']
    report = {'status': 'PASS-BATTLE-HD-ASSET-COVERAGE', 'registered_battle_surfaces':surfaces,'registered_lmi_overlays':lmi_overlays,'inventory_sha256': digest(args.inventory),
        'catalog_sha256': digest(args.catalog), 'catalog_stage': stage, 'tool_sha256': digest(Path(__file__)), 'catalog_assets': len(catalog['assets']),
        'figani': {'resource_positions': len(rows), 'decoded_resources': len(decoded),
            'native_frames': sum(r['native_frames'] for r in decoded), 'registered_resources': len(registered),
            'registered_hd_frames': sum(r['registered_hd_frames'] for r in rows),
            'remaining_decoded_resources': [r['resource'] for r in decoded if not r['registered_hd_frames']],
            'source_blocked_resources': [r['resource'] for r in rows if r['asset_status'] == 'SOURCE-BLOCKED'], 'resources': rows},
        'registered_fdother_effects': effects,
        'opaque_effect_contract': '沿modern_hd_battle_assets.go：FDOTHER原生Mask每像素寫入時可無透明像素。FIGANI及FDOTHER44第30幀完整原生Mask全0時只能全透明；其餘幀仍須255核心，不以成品自承空白。',
        'scope': '完整FIGANI來源盤點對正式catalog、原版metadata及實際HD PNG的資料覆蓋；非原版oracle。'
                 'header資格不等於caller已接線，FDOTHER只核已登錄效果，不假定四種效果涵蓋全部共同效果。'
                 '一般玩家、所有caller、其他法術／死亡／背景、效能與平台須另有證據。',
        'full_battle_hd_complete': False}
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({key: value for key, value in report['figani'].items() if key not in ['resources', 'remaining_decoded_resources']}, ensure_ascii=False))


if __name__ == '__main__': main()
