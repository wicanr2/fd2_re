#!/usr/bin/env python3
"""以普通攻防視窗PNG核對所選人物圖源的高覆蓋服裝核心。"""
import argparse
import json
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate
from verify_fd2_ares_battle_gpu import matches


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--figure', type=int, required=True)
    p.add_argument('--core-profile', choices=['blue', 'gold'], default='blue',
                   help='已知圖源服裝的取樣色域；只影響驗證，不影響遊戲')
    p.add_argument('--body-frames', help='只核對明示本體幀，格式為24:0,25:0-13；空白與投射物另驗')
    for name in ['receipt', 'catalog', 'source', 'report']:
        p.add_argument('--'+name, type=Path, required=True)
    args = p.parse_args()
    if not 0 <= args.figure <= 135 or args.report.exists(): raise SystemExit('人物越界或報告已存在')
    receipt = json.loads(args.receipt.read_text())
    boundary_save = (receipt.get('node_boundary_save_checked') is True
        and receipt.get('cold_loaded_units_equal') is True
        and bool(receipt.get('loaded_units'))
        and receipt['loaded_units'] == receipt.get('cold_units'))
    if (receipt.get('schema') != 'fd2-hd-battle-status-gui-v1' or receipt.get('status') != 'PASS'
        or not receipt.get('battle_hd_consumer_checked') or not receipt.get('new_process_title_load')
        or not (receipt.get('source_save_bytes_equal') or boundary_save) or not receipt.get('theme_save_bytes_equal')
        or digest(args.catalog) != receipt.get('catalog_sha256')):
        raise SystemExit('需要同份普通攻防完整GUI收據及清冊')
    catalog = validate(False, args.catalog); resources = [args.figure*3, args.figure*3+1]
    entries = [a for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set' and a['source_resource'] in resources]
    if sorted(a['source_resource'] for a in entries) != resources: raise SystemExit('需要完整待機及攻擊圖源')
    body_frames = None
    if args.body_frames:
        import re
        body_frames = set()
        for item in args.body_frames.split(','):
            m = re.fullmatch(r'(\d+):(\d+)(?:-(\d+))?', item)
            if not m: raise SystemExit('本體幀格式錯誤')
            resource, first, last = int(m[1]), int(m[2]), int(m[3] or m[2])
            asset = next((a for a in entries if a['source_resource'] == resource), None)
            if asset is None or not 0 <= first <= last < len(asset['files']): raise SystemExit('本體幀越界')
            body_frames.update((resource, i) for i in range(first, last+1))
    candidates = []
    for asset in entries:
        for index, (name, sha, frame) in enumerate(zip(asset['files'], asset['frame_sha256'], asset['native_frames'])):
            path = args.source/name
            if digest(path) != sha: raise SystemExit('圖源雜湊不同')
            if body_frames is not None and (asset['source_resource'], index) not in body_frames:
                continue
            with Image.open(path) as im:
                if im.mode != 'RGBA' or im.size != (frame['width']*6, frame['height']*6): raise SystemExit('圖源形狀不同')
                def selected(r, g, b, a):
                    if a < 250: return False
                    if args.core_profile == 'blue':
                        return b >= 70 and b*100 >= r*150 and b*100 >= g*130
                    return r >= 100 and g >= 70 and r >= g and g*100 >= b*150 and r*100 >= b*180
                points = [(i % im.width, i // im.width, r, g, b, a)
                          for i, (r, g, b, a) in enumerate(im.getdata()) if selected(r, g, b, a)]
            if len(points) < 1000: raise SystemExit('高覆蓋服裝核心像素不足')
            total = len(points); points = points[::max(1, total//2500)]
            candidates.append((asset['source_resource'], index, name, sha, frame, points, total))
    samples = list(receipt.get('battle_art_samples', []))
    for scene in ['world-hd', 'world-hd-replay']:
        name = scene+'-physical-scene.png'; shot = receipt['screenshots'][name]
        samples.append({'scene': scene, 'file': name, 'sha256': shot['sha256'], 'before': shot['gpu_before'], 'after': shot['state']})
    found, rejected = [], []
    for sample in samples:
        path = args.receipt.parent/sample['file']
        if Path(sample['file']).name != sample['file'] or digest(path) != sample['sha256']: raise SystemExit('GPU圖片路徑或雜湊不同')
        if not all(s.get('battle_scene') and s.get('hd_battle') for s in [sample['before'], sample['after']]):
            rejected.append({'file': path.name, 'reason': '擷取跨越戰鬥或主題邊界'}); continue
        with Image.open(path) as im:
            if im.size != (1920, 1200): raise SystemExit('需要1920×1200視窗PNG')
            rgb = im.convert('RGB').load()
            for resource, index, name, sha, frame, points, total in candidates:
                if matches(rgb, points, frame['x']*6, frame['y']*6, 1920, 1200):
                    found.append({'scene': sample['scene'], 'file': path.name, 'sha256': sample['sha256'],
                        'resource': resource, 'frame': index, 'source_file': name, 'source_sha256': sha,
                        'checked_pixels': len(points), 'eligible_pixels': total,
                        'world_origin': [frame['x']*6, frame['y']*6]})
    for scene in ['world-hd', 'world-hd-replay']:
        if not any(x['scene'] == scene and x['resource'] == resources[1] and x['frame'] > 0 for x in found):
            raise SystemExit('未在普通攻防GPU核對到所選人物的攻擊動作：'+scene)
    groups = {}
    for match in found: groups.setdefault(match['file'], []).append({'resource': match['resource'], 'frame': match['frame']})
    report = {'status': 'PASS-CHARACTER-NORMAL-ATTACK-GPU', 'figure': args.figure,
        'core_profile': args.core_profile,
        'body_frames': sorted(body_frames) if body_frames is not None else None,
        'node_boundary_save_checked': boundary_save,
        'receipt_sha256': digest(args.receipt), 'catalog_sha256': digest(args.catalog),
        'verifier_sha256': digest(Path(__file__)), 'pixel_matcher_sha256': digest(Path(__file__).with_name('verify_fd2_ares_battle_gpu.py')),
        'matches': found, 'sample_count': len(samples), 'rejected_boundary_samples': rejected,
        'ambiguous_source_candidates': {name: values for name, values in groups.items() if len(values) > 1},
        'scope': '普通兩次攻防的實際PNG與所選人物明示本體幀指定色域的高覆蓋服裝核心；原描述子位置。'
                 'alpha255一量化值，250..254依未知背景混合界限。多候選保留相容集合，不判定唯一相位；'
                 '未比較服裝核心以外的武器／FX，不宣稱全部幀GPU或原版對拍。'}
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'figure': args.figure, 'matched_pngs': len({x['file'] for x in found})}))


if __name__ == '__main__': main()
