#!/usr/bin/env python3
"""核對四召喚正式最終GPU中的施術者藍衣核心；fixture不升格普通玩家。"""
import argparse
import json
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate
from verify_fd2_ares_battle_gpu import matches


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--receipt', type=Path, action='append', required=True)
    p.add_argument('--figure', type=int, default=4)
    for name in ['catalog', 'source', 'report']:
        p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--original-pack', type=Path, required=True)
    p.add_argument('--inventory', type=Path, required=True)
    args = p.parse_args()
    if args.report.exists(): raise SystemExit('不覆寫既有報告')
    catalog = validate(False, args.catalog)
    inventory = json.loads(args.inventory.read_text())
    if digest(args.original_pack/'manifest.json') != inventory['pack_manifest_sha256']:
        raise SystemExit('遮擋來源包指紋不同')
    entries = {a['source_resource']: a for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set'}
    idle, actor = args.figure*3, args.figure*3+1
    if idle not in entries or actor not in entries: raise SystemExit('缺少完整施術者圖源')
    core = {}
    for resource in [idle, actor]:
        a = entries[resource]
        for index, (name, sha, frame) in enumerate(zip(a['files'], a['frame_sha256'], a['native_frames'])):
            path = args.source/name
            if digest(path) != sha: raise SystemExit('施術者圖源SHA不同')
            with Image.open(path) as im:
                if im.mode != 'RGBA' or im.size != (frame['width']*6, frame['height']*6): raise SystemExit('施術者圖源形狀不同')
                points = [(i % im.width, i // im.width, r, g, b, alpha)
                          for i, (r, g, b, alpha) in enumerate(im.getdata())
                          if alpha >= 250 and b >= 70 and b*100 >= r*150 and b*100 >= g*130]
            if len(points) < 1000: raise SystemExit('本圖源不適用藍衣核心核對')
            core[(resource, index)] = (name, sha, points, frame)
    native_masks = {}

    def occluded(x, y, layers):
        for layer in layers:
            ox, oy = x-layer['x']*6, y-layer['y']*6
            if not (0 <= ox < layer['width']*6 and 0 <= oy < layer['height']*6): continue
            if layer['resource'] < 0 or layer['archive'] not in ['', 'FIGANI.DAT']:
                raise SystemExit('不能解釋的前景圖層遮擋')
            key = (layer['resource'], layer['frame'])
            if key not in native_masks:
                resource, index = key
                asset = entries.get(resource)
                if asset is not None:
                    path = args.source/asset['files'][index]
                    if digest(path) != asset['frame_sha256'][index]: raise SystemExit('前景HD圖源SHA不同')
                    with Image.open(path) as im: mask = im.getchannel('A').copy()
                    native_masks[key] = (mask, True)
                else:
                    row = inventory['resources'][resource]
                    folder = args.original_pack/'animations'/f'FIGANI_{resource:03d}'
                    metadata = folder/'animation.json'
                    if row['resource'] != resource or digest(metadata) != row['metadata_sha256']: raise SystemExit('原生前景metadata不同')
                    frame = json.loads(metadata.read_text())['frames'][index]
                    path = folder/frame['mask_path']
                    if digest(path) != row['frames'][index]['mask_sha256']: raise SystemExit('原生前景mask SHA不同')
                    with Image.open(path) as im: mask = im.convert('L')
                    native_masks[key] = (mask, False)
            mask, hd = native_masks[key]
            if mask.size != (layer['width']*(6 if hd else 1), layer['height']*(6 if hd else 1)):
                raise SystemExit('前景遮罩形狀不同')
            if mask.getpixel((ox, oy) if hd else (ox//6, oy//6)) != 0: return True
        return False
    reports = []
    for receipt in args.receipt:
        d = json.loads(receipt.read_text())
        if not d.get('status', '').startswith('PASS') or d.get('continuations') != 1 or not d.get('go_rng_unchanged'):
            raise SystemExit('需要完整RunGame交易收據')
        found = []
        for name, sample in d.get('source_shots', {}).items():
            if sample['phase'] not in ['prelude', 'actor', 'actor-slide'] or sample['subtract'] != 0 or sample['mix'] != 0: continue
            if Path(name).name != name: raise SystemExit('GPU圖片路徑越界')
            path = receipt.parent/name
            with Image.open(path) as im:
                if im.size != (1920, 1200): raise SystemExit('需要1920×1200最終GPU')
                rgb = im.convert('RGB').load()
                for layer_index, layer in enumerate(sample['layers']):
                    key = (layer['resource'], layer['frame'])
                    if layer['archive'] not in ['', 'FIGANI.DAT'] or key not in core: continue
                    source, sha, all_points, frame = core[key]
                    if (layer['width'], layer['height']) != (frame['width'], frame['height']): raise SystemExit('GPU來源描述子不同')
                    points = [p for p in all_points if not occluded(p[0]+layer['x']*6, p[1]+layer['y']*6, sample['layers'][layer_index+1:])]
                    visible_count = len(points)
                    if visible_count < 1000: continue
                    points = points[::max(1, visible_count//2500)]
                    if matches(rgb, points, layer['x']*6, layer['y']*6, 1920, 1200):
                        found.append({'file': name, 'sha256': digest(path), 'phase': sample['phase'],
                                      'resource': key[0], 'frame': key[1], 'source_file': source,
                                      'source_sha256': sha, 'checked_pixels': len(points),
                                      'visible_eligible_pixels': visible_count, 'source_eligible_pixels': len(all_points),
                                      'excluded_by_source_occlusion': len(all_points)-visible_count})
        for phase in ['prelude', 'actor', 'actor-slide']:
            if not d.get('phase_f2_switches', {}).get(phase) or not any(x['phase'] == phase for x in found):
                raise SystemExit('缺少施術者階段的F2及實際HD核心：'+phase)
        if not any(x['phase'] == 'actor' and x['resource'] == actor and x['frame'] > 0 for x in found):
            raise SystemExit('沒有核對到非零施術者動作')
        reports.append({'receipt': str(receipt), 'receipt_sha256': digest(receipt), 'matches': found})
    report = {'status': 'PASS-COMPOUND-CASTER-GPU', 'figure': args.figure,
              'catalog_sha256': digest(args.catalog), 'verifier_sha256': digest(Path(__file__)),
              'pixel_matcher_sha256': digest(Path(__file__).with_name('verify_fd2_ares_battle_gpu.py')),
              'occlusion_inventory_sha256': digest(args.inventory),
              'reports': reports, 'scope': '正式RunGame最終GPU的prelude／actor／actor-slide高覆蓋藍衣核心，限明示fixture。'
              '只排除已核SHA的前景原生mask或HD alpha實際遮住的像素，剩餘須至少1000個核心。'
              '相近圖源不判唯一相位，武器／FX及其他施法者另驗；非普通玩家或原版oracle。'}
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'receipts': len(reports), 'matched_pngs': sum(len({x['file'] for x in r['matches']}) for r in reports)}))


if __name__ == '__main__': main()
