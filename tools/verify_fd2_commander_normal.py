#!/usr/bin/env python3
"""核對第23章既有正常死亡路徑中的隊長待機及116故事頭像。Docker內執行。"""
import argparse
import json
from pathlib import Path

from PIL import Image, ImageChops, ImageOps
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate
from verify_fd2_ares_battle_gpu import matches


def require(condition, message):
    if not condition:
        raise ValueError(message)


def verify(receipt_path, catalog_path, source):
    receipt = json.loads(receipt_path.read_text())
    require(receipt.get('schema') == 'fd2-shared-death-normal-gui-v1'
            and receipt.get('status') == 'PASS' and receipt.get('commander_hd') is True
            and receipt.get('state_injection') is False and receipt.get('event_id') == 53
            and receipt.get('death_kind') == 'action'
            and receipt.get('final_node') == 'preparation_ch24', '需要完整正常隊長GUI收據')
    require(receipt.get('catalog_sha256') == digest(catalog_path), '清冊SHA不同')
    catalog = validate(False, catalog_path)
    original_source = Path(receipt['source_file'])
    require(digest(original_source) == receipt['source_sha256'], '來源槽已改變')
    saved = receipt_path.parent/'user-data/fd2_re/fd2_save.json'
    require(digest(saved) == receipt['saved_sha256']
            and json.loads(saved.read_text())['node'] == 'preparation_ch24', '保存邊界不同')
    shots = receipt['screenshots']
    for name, hd in [('death23-returned-preparation-faithful.png', False),
                     ('death23-returned-preparation-hd.png', True),
                     ('death23-cold-load-preparation.png', True)]:
        require(name in shots and shots[name]['state'].get('node') == 'preparation_ch24'
                and shots[name]['state'].get('hd_theme') is hd
                and digest(receipt_path.parent/name) == shots[name]['sha256'], '缺少F2或冷讀畫面')
    require(any(o.get('process_epoch') == 1 and o.get('key') == 'Return'
                and o.get('ack', {}).get('node') == 'preparation_ch24'
                for o in receipt['operations']), '沒有新程序正常LOAD紀錄')
    def picture(sample):
        name = sample['file']
        require(Path(name).name == name, '圖片路徑越界')
        path = receipt_path.parent/name
        require(digest(path) == sample['sha256'], '圖片SHA不同')
        image = Image.open(path).convert('RGB')
        require(image.size == (1920, 1200), '圖片尺寸不同')
        return image
    idle = next(a for a in catalog['assets']
                if a['role'] == 'battle_animation_hd_set' and a['source_resource'] == 348)
    require(idle['frame_count'] == 1 and digest(source/idle['files'][0]) == idle['frame_sha256'][0],
            '隊長待機來源不同')
    image = Image.open(source/idle['files'][0]).convert('RGBA')
    points = [(i % image.width, i // image.width, r, g, b, a)
              for i, (r, g, b, a) in enumerate(image.getdata())
              if a >= 250 and b >= 70 and b*100 >= r*150 and b*100 >= g*130]
    require(len(points) >= 1000, '待機核心不足1000點')
    points = points[::max(1, len(points)//2500)]
    found = []
    for sample in receipt.get('commander_samples', []):
        image = picture(sample)
        for state in [sample['before'], sample['after']]:
            require(not state.get('error'), '待機觀測含錯誤')
            if not (state.get('battle_scene') and state.get('hd_battle')
                    and state.get('battle_target_fig') == 116):
                continue
            for layer in state.get('battle_source_layers') or []:
                if (layer.get('archive', '') in ('', 'FIGANI.DAT') and layer['resource'] == 348
                        and layer['frame'] == 0 and layer['fill'] == -1
                        and (layer['width'], layer['height']) == (idle['native_frames'][0]['width'],
                                                                idle['native_frames'][0]['height'])
                        and matches(image.load(), points, layer['x']*6, layer['y']*6, 1920, 1200)):
                    found.append(dict(file=sample['file'], checked_pixels=len(points),
                                      origin=[layer['x']*6, layer['y']*6]))
                    break
    require(found, '沒有核對到普通攻擊目標348待機')
    portrait = next(a for a in catalog['assets']
                    if a['role'] == 'story_portrait_hd_frame' and a['speaker_id'] == 116)
    require(digest(source/portrait['file']) == portrait['sha256'], '頭像來源不同')
    base = Image.open(source/portrait['file']).convert('RGB')
    require(base.size == (480, 480), '需要480頭像')
    portraits = []
    for sample in receipt.get('portrait_samples', []):
        image = picture(sample)
        before, after = sample['before'], sample['after']
        require(all(s.get('dialogue_speaker') == 116 and s.get('dialogue_native') is True
                    and s.get('hd_theme') is True and not s.get('error') for s in [before, after]),
                '頭像擷取跨越speaker或HD邊界')
        rect = before.get('hd_overlay_rect')
        require(rect == after.get('hd_overlay_rect') and isinstance(rect, list) and len(rect) == 4
                and rect[2:] == [160, 160], '頭像觀測矩形不同')
        x, y, w, h = [v*3 for v in rect]
        require(0 <= x < x+w <= 1920 and 0 <= y < y+h <= 1200, '頭像越界')
        expected = ImageOps.mirror(base) if before.get('hd_overlay_mirrored') else base
        require(before.get('hd_overlay_mirrored') == after.get('hd_overlay_mirrored'), '鏡像邊界不同')
        difference = ImageChops.difference(image.crop((x, y, x+w, y+h)), expected)
        maximum = max(v[1] for v in difference.getextrema())
        require(maximum <= 1, '完整480頭像RGB不符')
        portraits.append(dict(file=sample['file'], pixels=w*h, maximum_rgb_error=maximum, rect=[x, y, w, h]))
    require(portraits, '缺少116正常故事頭像')
    return dict(status='PASS-COMMANDER-NORMAL-TARGET-AND-PORTRAIT',
                receipt_sha256=digest(receipt_path), catalog_sha256=digest(catalog_path),
                verifier_sha256=digest(Path(__file__)), idle=found, portraits=portraits,
                scope='章前強化槽、正常進軍與蓋亞攻擊的目標待機、戰後116頭像、F2保存及新程序LOAD。非隊長主動攻擊、自然難度、原版parity或整章E2。')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['receipt', 'catalog', 'source', 'report']:
        parser.add_argument('--'+name, type=Path, required=True)
    args = parser.parse_args()
    require(Path('/.dockerenv').exists() and not args.report.exists(), '在Docker使用新報告')
    report = verify(args.receipt, args.catalog, args.source)
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(dict(status=report['status'], idle=len(report['idle']), portraits=len(report['portraits']))))


if __name__ == '__main__':
    main()
