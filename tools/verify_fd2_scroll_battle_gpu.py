#!/usr/bin/env python3
"""核對非共背景測試場景的最終GPU圖源，證據限於RUNTIME-E1。"""
import argparse
import json
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate
from verify_fd2_ares_battle_gpu import matches


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['receipt', 'catalog', 'source', 'report']:
        parser.add_argument('--'+name, type=Path, required=True)
    parser.add_argument('--projectiles', action='store_true', help='另核對FIGANI25的15..18四張飛矢')
    parser.add_argument('--target-figure', type=int, choices=[76,96], default=96)
    parser.add_argument('--target-frames', action='store_true', help='另核對明示敵方所有後續攻擊幀')
    args = parser.parse_args()
    if args.report.exists():
        raise SystemExit('不覆寫既有報告')
    receipt = json.loads(args.receipt.read_text())
    if receipt.get('target_figure',96) != args.target_figure:
        raise SystemExit('測試收據的敵方人物不同')
    if args.target_frames and receipt.get('target_attack_frames_captured') is not True:
        raise SystemExit('需要完整敵方攻擊幀擷取收據')
    phases = ['prelude', 'departure', 'scroll-out', 'scroll-in', 'body', 'counter']
    if (receipt.get('status') != 'PASS-SCOPED-RUNTIME-E1'
            or receipt.get('side') not in [0, 2]
            or receipt.get('real_registered_battle_bank') is not True
            or receipt.get('continuations') != 1
            or receipt.get('battle_dac_steps') != 64
            or receipt.get('map_dac_steps') != 65
            or any(receipt.get('phase_f2', {}).get(p) != 1 for p in phases)
            or any(receipt.get('phase_hd_draws', {}).get(p, 0) <= 0 for p in phases)):
        raise SystemExit('需要兩次射擊／反擊及六階段F2的完整正式圖源測試收據')
    catalog = validate(False, args.catalog)
    idle, attack = args.target_figure*3, args.target_figure*3+1
    resources = [25, idle, attack] if args.projectiles else [idle, attack]
    entries = {a['source_resource']: a for a in catalog['assets']
               if a['role'] == 'battle_animation_hd_set' and a['source_resource'] in resources}
    expected = [(idle,4),(attack,9 if args.target_figure==76 else 8)]
    if args.projectiles: expected = [(25,19)]+expected
    if sorted((r, a['frame_count']) for r, a in entries.items()) != expected:
        raise SystemExit('需要完整敵方待機及攻擊圖源')
    found, shots = [], []
    for phase, shot in receipt.get('source_shots', {}).items():
        name = shot['file']
        if Path(name).name != name:
            raise SystemExit('GPU圖片路徑越界')
        path = args.receipt.parent/name
        with Image.open(path) as im:
            if im.size != (1920, 1200):
                raise SystemExit('需要1920×1200最終GPU圖片')
            rgb = im.convert('RGB').load()
            for layer in shot['layers']:
                resource, index = layer['resource'], layer['frame']
                if resource not in entries or layer['fill'] != -1:
                    continue
                projectile = resource == 25 and index in range(15,19)
                if resource == 25 and not projectile: continue
                entry = entries[resource]
                if not isinstance(index, int) or not 0 <= index < entry['frame_count']:
                    raise SystemExit('圖源幀越界')
                frame = entry['native_frames'][index]
                rect = layer['rect']; origin = rect['Min']; end = rect['Max']
                if (end['X']-origin['X'], end['Y']-origin['Y'], origin['Y']) != (frame['width'], frame['height'], frame['y']):
                    raise SystemExit('原生描述子尺寸或垂直原點不同')
                source = args.source/entry['files'][index]
                sha = entry['frame_sha256'][index]
                if digest(source) != sha:
                    raise SystemExit('圖源雜湊不同')
                with Image.open(source) as art:
                    if art.mode != 'RGBA' or art.size != (frame['width']*6, frame['height']*6):
                        raise SystemExit('圖源形狀不同')
                    clip = layer['clip']
                    def eligible(x, y, pixel):
                        r, g, b, a = pixel
                        wx, wy = x+origin['X']*6, y+origin['Y']*6
                        if not (0 <= wx < 1920 and 0 <= wy < 1200):
                            return False
                        if clip and not (clip['Min']['X']*6 <= wx < clip['Max']['X']*6
                                         and clip['Min']['Y']*6 <= wy < clip['Max']['Y']*6):
                            return False
                        return a >= 250 and (projectile or b >= 70 and b*100 >= r*150 and b*100 >= g*130)
                    points = [(i % art.width, i // art.width, *pixel)
                              for i, pixel in enumerate(art.getdata())
                              if eligible(i % art.width, i // art.width, pixel)]
                if len(points) < (50 if projectile else 1000):
                    continue
                total = len(points)
                points = points[::max(1, total//2500)]
                if matches(rgb, points, origin['X']*6, origin['Y']*6, 1920, 1200):
                    found.append({'phase': phase, 'file': name, 'resource': resource, 'frame': index,
                                  'source_file': entry['files'][index], 'source_sha256': sha,
                                  'eligible_pixels': total, 'checked_pixels': len(points),
                                  'world_origin': [origin['X']*6, origin['Y']*6], 'clip': clip})
        shots.append({'file': name, 'sha256': digest(path)})
    for phase, resource in [('scroll-in',idle),('body',idle),('counter',attack)]:
        if not any(x['phase'] == phase and x['resource'] == resource for x in found):
            raise SystemExit('未核對到實際GPU圖源：'+phase)
    if args.projectiles:
        for index in range(15,19):
            if not any(x['resource']==25 and x['frame']==index and x['phase']=='body-bolt-'+str(index) for x in found):
                raise SystemExit('未核對到實際飛矢GPU圖源：'+str(index))
    if args.target_frames:
        for index in range(1,entries[attack]['frame_count']):
            if not any(x['resource']==attack and x['frame']==index and x['phase']==f'counter-target-{index:02d}' for x in found):
                raise SystemExit('未核對到實際敵方攻擊GPU圖源：'+str(index))
    report = {'status': 'PASS-SCROLL-FIXTURE-REGISTERED-GPU', 'side': receipt['side'],
              'receipt_sha256': digest(args.receipt), 'catalog_sha256': digest(args.catalog),
              'verifier_sha256': digest(Path(__file__)),
              'pixel_matcher_sha256': digest(Path(__file__).with_name('verify_fd2_ares_battle_gpu.py')),
              'screenshots': shots, 'matches': found,
              'four_projectile_frames_checked': args.projectiles,
              'target_figure':args.target_figure,'target_attack_frames_checked':args.target_frames,
              'scope': '修改fixture的正式Game.Draw／最終GPU，限非共背景scroll-in、body與counter已登錄敵方藍褲核心。'
                       '原描述子平移及裁切，人物至少1000點；明示飛矢模式另驗15..18各至少50個高覆蓋來源點。'
                       '不證明所有像素、普通玩家、原版oracle或平台效能。'}
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'side': receipt['side'], 'matches': len(found)}))


if __name__ == '__main__':
    main()
