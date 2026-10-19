#!/usr/bin/env python3
"""核對普通首章攻擊PNG中的亞雷斯已編製圖源，不把HD布林標記當圖像證據。"""
import argparse
import json
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate


def matches(rgb, samples, ox, oy, width, height):
    def check(point):
        x, y, r, g, b, alpha = point
        x += ox; y += oy
        if not 0 <= x < width or not 0 <= y < height:
            return False
        actual = rgb[x, y]
        # Ebitengine預設src*a + dst*(1-a)，未知背景各通道0..255。
        return all(value*alpha/255-1.01 <= observed <= value*alpha/255+255-alpha+1.01
                   for value, observed in zip((r, g, b), actual))
    stride = max(1, len(samples)//20)
    return all(check(point) for point in samples[::stride]) and all(check(point) for point in samples)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['receipt', 'catalog', 'source', 'report']:
        p.add_argument('--'+name, type=Path, required=True)
    args = p.parse_args()
    if args.report.exists(): raise SystemExit('不覆寫既有收據')
    receipt = json.loads(args.receipt.read_text())
    if (receipt.get('schema') != 'fd2-hd-battle-status-gui-v1' or receipt.get('status') != 'PASS'
        or not receipt.get('battle_hd_consumer_checked') or not receipt.get('new_process_title_load')
        or not receipt.get('source_save_bytes_equal') or not receipt.get('theme_save_bytes_equal')
        or digest(args.catalog) != receipt.get('catalog_sha256')):
        raise SystemExit('需要同份普通首章完整GUI收據及清冊')
    catalog = validate(False, args.catalog)
    entries = [a for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set' and a['source_resource'] in [12, 13]]
    if sorted((a['source_resource'], a['frame_count']) for a in entries) != [(12, 4), (13, 15)]:
        raise SystemExit('需要亞雷斯完整兩組圖源')
    candidates = []
    for asset in entries:
        for index, (name, sha, frame) in enumerate(zip(asset['files'], asset['frame_sha256'], asset['native_frames'])):
            path = args.source/name
            if digest(path) != sha: raise SystemExit('圖源雜湊不同')
            with Image.open(path) as im:
                if im.mode != 'RGBA' or im.size != (frame['width']*6, frame['height']*6):
                    raise SystemExit('圖源形狀不同')
                samples = [(i % im.width, i // im.width, r, g, b, a)
                           for i, (r, g, b, a) in enumerate(im.getdata())
                           if a >= 250 and b >= 70 and b*100 >= r*150 and b*100 >= g*130]
            if len(samples) < 1000: raise SystemExit('高覆蓋藍衣像素不足')
            total = len(samples); samples = samples[::max(1, len(samples)//2500)]
            candidates.append((asset['source_resource'], index, name, sha, frame, samples, total))
    samples = list(receipt.get('battle_art_samples', []))
    for scene in ['world-hd', 'world-hd-replay']:
        name = scene+'-physical-scene.png'; shot = receipt['screenshots'][name]
        samples.append({'scene': scene, 'file': name, 'sha256': shot['sha256'],
                        'before': shot['gpu_before'], 'after': shot['state']})
    found = []; rejected = []
    for sample in samples:
        path = args.receipt.parent/sample['file']
        if Path(sample['file']).name != sample['file'] or digest(path) != sample['sha256']:
            raise SystemExit('GPU圖片路徑或雜湊不同')
        if not all(s.get('battle_scene') and s.get('hd_battle') for s in [sample['before'], sample['after']]):
            rejected.append({'file': path.name, 'reason': '擷取跨越戰鬥或主題邊界'}); continue
        with Image.open(path) as im:
            if im.size != (1920, 1200): raise SystemExit('需要原比例1920×1200視窗PNG')
            rgb = im.convert('RGB').load()
            for resource, index, name, sha, frame, points, total in candidates:
                if matches(rgb, points, frame['x']*6, frame['y']*6, 1920, 1200):
                    found.append({'scene': sample['scene'], 'file': path.name, 'sha256': sample['sha256'],
                        'resource': resource, 'frame': index, 'source_file': name, 'source_sha256': sha,
                        'checked_pixels': len(points), 'eligible_pixels': total,
                        'world_origin': [frame['x']*6, frame['y']*6]})
    for scene in ['world-hd', 'world-hd-replay']:
        if not any(x['scene'] == scene and x['resource'] == 13 and x['frame'] > 0 for x in found):
            raise SystemExit('未在普通攻擊GPU中核對到新增攻擊動作：'+scene)
    groups = {}
    for match in found:
        groups.setdefault(match['file'], []).append({'resource': match['resource'], 'frame': match['frame']})
    report = {'status': 'PASS-ARES-NORMAL-ATTACK-GPU', 'receipt_sha256': digest(args.receipt),
        'catalog_sha256': digest(args.catalog), 'verifier_sha256': digest(Path(__file__)),
        'matches': found, 'rejected_boundary_samples': rejected, 'sample_count': len(samples),
        'ambiguous_source_candidates': {name: values for name, values in groups.items() if len(values) > 1},
        'scope': '兩次普通首章攻擊的實際PNG與新增圖源高覆蓋藍衣像素；按原描述子放置。'
                 'alpha255精確至一量化值，250..254按未知背景混合界限。'
                 '多個候選相符時只列相容集合，不判定唯一相位；不比較非藍衣的武器與光效。非全部19幀GPU或原版對拍。'}
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'matched_samples': len(found),
                      'compatible_attack_frames': sorted({x['frame'] for x in found if x['resource'] == 13})}))


if __name__ == '__main__': main()
