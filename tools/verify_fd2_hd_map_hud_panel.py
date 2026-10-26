#!/usr/bin/env python3
"""獨立驗原生HUD的3像素細框、直接HD繪字、舊PNG拒收與框外保持。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageChops


def require(condition, message):
    if not condition:
        raise ValueError(message)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def load(path):
    with Image.open(path) as pic:
        require(pic.size == (1920, 1200), 'PNG不是1920×1200')
        return pic.convert('RGBA')


def fine_panel(pic, rect):
    x, y, x1, y1 = rect
    require((x1-x, y1-y) == (414, 204), 'HUD矩形不同')
    require(x in (6, 30, 1452, 1476) and y in (942, 960, 966), 'HUD原點不同')
    # Above the labels and icon, the measured inner bevel has a three-pixel
    # stroke. A native image enlarged six times cannot satisfy this pattern.
    top = pic.crop((x+210, y+12, x+390, y+15))
    body = pic.crop((x+210, y+15, x+390, y+18))
    a, b = top.getpixel((0, 0)), body.getpixel((0, 0))
    require(a != b and top.getextrema() == tuple((c, c) for c in a)
            and body.getextrema() == tuple((c, c) for c in b), '未找到3像素HD細框')
    # A/D values occupy (43,8)..(63,27) in the source. Subpixel variation must
    # appear inside native six-pixel cells, away from the already-HD unit.
    subpixel = 0
    for yy in list(range(48, 96, 6))+list(range(114, 162, 6)):
        for xx in range(258, 378, 6):
            block = pic.crop((x+xx, y+yy, x+xx+6, y+yy+6))
            if any(lo != hi for lo, hi in block.getextrema()[:3]):
                subpixel += 1
    require(subpixel >= 12, 'A/D仍是六倍原版字模')
    return {'rect': rect, 'subpixel_numeric_cells': subpixel,
            'border_ink': a, 'body': b}


def rect_from_go(r):
    require(set(r) == {'Min', 'Max'}, '收據矩形缺項')
    return [r['Min']['X'], r['Min']['Y'], r['Max']['X'], r['Max']['Y']]


def validate(receipt, previous=None, normal=False, catalog=None, pack=None, override=None, clip_receipt=None):
    data = json.loads(receipt.read_text()) if override is None else override
    rows, rejected = [], []
    if normal:
        save_ok = data['source_save_bytes_equal'] is True or (
            data['natural_attack_return_checked'] is True and data['post_battle_save_checked'] is True
            and data['node_boundary_save_checked'] is True)
        require(data['status'].startswith('PASS') and data['new_process_title_load'] is True
                and save_ok and data['theme_save_bytes_equal'] is True
                and data['cold_loaded_units_equal'] is True, '正常玩家驗證未通過')
        for name in ['battle-hd-field.png', 'battle-hd-focused.png',
                     'battle-hd-focused-restored.png', 'world-hd-source-reloaded.png']:
            if name not in data['screenshots']:
                continue
            row = data['screenshots'][name]
            require(row['state']['hd_world'] is True and not row['state']['error'], '正常玩家未顯示HD世界')
            path = receipt.parent/name
            require(sha(path) == row['sha256'], '正常PNG雜湊不同')
            pic = load(path)
            matches = []
            for x in (30, 1476):
                try:
                    matches.append(fine_panel(pic, [x, 966, x+414, 1170]))
                except ValueError:
                    pass
            require(len(matches) == 1, '正常HUD未唯一定位')
            rows.append({'case': name, **matches[0]})
        require(len(rows) >= 3, '正常首章HUD抽樣缺項')
    else:
        require(data['status'] == 'PASS-HUD-PANEL-HD-RUNTIME-E1'
                and data['groups'] == 125 and data['source_states'] == 500,
                'HUD來源覆蓋未通過')
        require(len(data['cases']) == 18 and len({r['case'] for r in data['cases']}) == 18, 'GPU案例缺少或重複')
        for row in data['cases']:
            require(row['panel_hd'] is True, '仍是原版HUD')
            pic = load(receipt.parent/(row['case']+'.png'))
            require(hashlib.sha256(pic.tobytes()).hexdigest() == row['final_sha256'], 'GPU PNG雜湊不同')
            r = rect_from_go(row['rect'])
            # Event effects cover the unit, not this frame or A/D area.
            checked = fine_panel(pic, r)
            if row['case'].startswith('registered-terrain-'):
                require(catalog and pack and row['terrain_hd'] is True and row['group'] == -1, '已登錄縮圖來源缺項')
                tile = int(row['case'].rsplit('-', 1)[1])
                entries = [a for a in json.loads(catalog.read_text())['assets']
                           if a['role'] == 'native_terrain_hd_set' and a['map_id'] == 0 and tile in a['frame_ids']]
                require(len(entries) == 1, '已登錄縮圖清冊不唯一')
                asset = entries[0]; index = asset['frame_ids'].index(tile)
                name = asset['files'][index]
                require(Path(name).name == name and sha(pack/name) == asset['frame_sha256'][index], '已登錄縮圖檔案不同')
                with Image.open(pack/name) as source:
                    source = source.convert('RGBA')
                require(source.size == (144, 144) and source.getchannel('A').getextrema() == (255, 255), '縮圖形狀不同')
                actual = pic.crop((r[0]+36, r[1]+30, r[0]+180, r[1]+174))
                require(max(hi for lo, hi in ImageChops.difference(source, actual).getextrema()) <= 2, 'HUD未使用已登錄地形PNG')
                checked['registered_terrain_pixels'] = 144*144
            if previous and row['case'] in ('ring-restore', 'event-effect-above-hud'):
                old = load(previous/(row['case']+'.png'))
                # This older capture has an earlier terrain catalog. It can
                # reject the native HUD style, but is not an outside-world oracle.
                checked['previous_reference'] = 'HUD樣式反例，地形清冊較早，不比較框外'
                try:
                    fine_panel(old, r)
                except ValueError:
                    rejected.append(row['case'])
                else:
                    raise ValueError('舊原版HUD未拒收')
            rows.append({'case': row['case'], **checked})
        if previous:
            require(len(rejected) == 2, '舊圖拒收缺項')
    clip_check = None
    if clip_receipt:
        clip = json.loads(clip_receipt.read_text())
        require(clip['status'] == 'PASS-HUD-CLIP-GPU' and clip['unchanged_pixels'] == 2219544,
                '同來源裁切收據不同')
        r = rect_from_go(clip['rect'])
        native, modern = load(clip_receipt.parent/'native-control.png'), load(clip_receipt.parent/'hd-panel.png')
        fine_panel(modern, r)
        blank = Image.new('RGBA', (414, 204))
        native.paste(blank, tuple(r[:2])); modern.paste(blank, tuple(r[:2]))
        require(not any(c.getbbox() for c in ImageChops.difference(native, modern).split()), '同來源HUD框外畫面改變')
        clip_check = {'receipt_sha256': sha(clip_receipt), 'unchanged_pixels': clip['unchanged_pixels']}
    if previous:
        require(clip_check is not None, '舊樣式反例另須同來源框外核對')
    return {'status': 'PASS', 'receipt_sha256': sha(receipt), 'normal': normal,
            'cases': rows, 'old_images_rejected': rejected,
            'clip': clip_check,
            'scope': 'PNG形狀與範圍的獨立核對；文字格式／顏色與完整GPU像素另由Go來源測試核對'}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--receipt', type=Path, required=True)
    p.add_argument('--previous', type=Path)
    p.add_argument('--normal', action='store_true')
    p.add_argument('--catalog', type=Path)
    p.add_argument('--pack', type=Path)
    p.add_argument('--self-test', action='store_true')
    p.add_argument('--clip-receipt', type=Path)
    p.add_argument('--output', type=Path, required=True)
    a = p.parse_args()
    require(Path('/.dockerenv').exists() and not a.output.exists() and a.output.parent.is_dir(),
            '在Docker使用既有父目錄及全新輸出')
    result = validate(a.receipt, a.previous, a.normal, a.catalog, a.pack, clip_receipt=a.clip_receipt)
    if a.self_test:
        require(not a.normal, '收據反例使用受控GPU收據')
        original = json.loads(a.receipt.read_text())
        mutations = []
        for key, value in [('status', 'PASS'), ('groups', 124), ('source_states', 499)]:
            bad = copy.deepcopy(original); bad[key] = value; mutations.append(bad)
        bad = copy.deepcopy(original); bad['cases'].pop(); mutations.append(bad)
        for key, value in [('panel_hd', False), ('final_sha256', '0'*64),
                           ('rect', {'Min': {'X': 0, 'Y': 0}, 'Max': {'X': 414, 'Y': 204}})]:
            bad = copy.deepcopy(original); bad['cases'][0][key] = value; mutations.append(bad)
        for bad in mutations:
            try:
                validate(a.receipt, a.previous, False, a.catalog, a.pack, bad, a.clip_receipt)
            except ValueError:
                continue
            raise ValueError('變造收據未拒收')
        result['receipt_rejections'] = len(mutations)
    a.output.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': result['status'], 'cases': len(result['cases']),
                      'old_images_rejected': len(result['old_images_rejected'])}))


if __name__ == '__main__':
    main()
