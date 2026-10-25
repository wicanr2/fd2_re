#!/usr/bin/env python3
"""在素材 Docker 內核對 Game.Draw 收據的 HD 面板框線、槽與填條。

python3 tools/verify_fd2_battle_panel_chrome.py --receipt ABS/receipt.json \
  --original-pack ABS --out FRESH.json

只驗收據內獨立保存的面板RGBA，使用同次base PNG的色盤及唯讀原版UI。
文字區由既有字形／完整畫面驗證負責；本工具不宣稱原版RGB或正常玩家。
"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageChops, ImageDraw


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def checked_image(root, name, sha, size):
    if not isinstance(name, str) or Path(name).name != name:
        raise ValueError('圖像路徑須為收據目錄內的檔名')
    p = root / name
    if digest(p) != sha:
        raise ValueError('圖像SHA不同')
    with Image.open(p) as image:
        if image.size != size:
            raise ValueError('圖像尺寸不同')
        return image.copy()


def source_indices(pack):
    folder = pack / 'ui/fdother_005_item_panel'
    meta = json.loads((folder / 'resource.json').read_text())
    if meta.get('kind') != 'fdother_lmi1_item_panel' or meta.get('source', {}).get('resource') != 5:
        raise ValueError('原版面板來源不同')
    result = {}
    for index in (22, 24, 27, 29):
        entry = next(e for e in meta['entries'] if e['index'] == index)
        with Image.open(folder / entry['frame']) as im:
            size = (149, 42) if index == 22 else (1, 5)
            if im.mode != 'P' or im.size != size:
                raise ValueError('原始面板或bar尺寸不同')
            result[index] = im.copy()
    return result


def verify_picture(picture, values, palette, source):
    if (picture.size != (894, 252) or picture.mode != 'RGBA'
            or picture.getchannel('A').getextrema() != (255, 255)
            or set(values) != {'Level', 'HP', 'MaxHP', 'MP', 'MaxMP'}
            or any(type(v) is not int for v in values.values())
            or palette is None or len(palette) < 768):
        raise ValueError('面板凍結資料或色盤不完整')

    def colour(index):
        return tuple(palette[index * 3:index * 3 + 3]) + (255,)

    background = colour(source[22].getpixel((100, 10)))
    border = colour(source[22].getpixel((1, 1)))
    track = colour(source[29].getpixel((0, 1)))
    expected = Image.new('RGBA', (894, 252), background)
    d = ImageDraw.Draw(expected)
    d.rectangle((0, 0, 893, 251), outline=border, width=3)
    d.rectangle((24, 120, 869, 121), fill=border)
    for y, current, maximum, entry in (
        (132, values['HP'], values['MaxHP'], 24),
        (186, values['MP'], values['MaxMP'], 27),
    ):
        d.rectangle((126, y, 743, y + 35), fill=track)
        if current != 0 and maximum != 0:
            # Truncate toward zero as the indexed writer does; include both caps.
            quotient = abs(101 * current) // abs(maximum)
            if (current < 0) != (maximum < 0):
                quotient = -quotient
            span = min(103, max(1, quotient + 1) + 1) * 6
            d.rectangle((126, y, 125 + span, y + 35), fill=colour(source[entry].getpixel((0, 1))))
        d.rectangle((126, y, 743, y + 35), outline=border, width=1)

    mask = Image.new('L', expected.size, 255)
    m = ImageDraw.Draw(mask)
    for left, top, right, bottom in (
        (5, 4, 114, 20), (115, 4, 131, 13), (132, 4, 147, 13),
        (3, 21, 20, 30), (126, 21, 147, 30),
        (3, 30, 20, 39), (126, 30, 147, 39),
    ):
        m.rectangle((left * 6, top * 6, right * 6 - 1, bottom * 6 - 1), fill=0)
    channels = ImageChops.difference(picture, expected).split()
    rgb = ImageChops.lighter(ImageChops.lighter(channels[0], channels[1]), channels[2])
    different = ImageChops.multiply(rgb, mask)
    if different.getbbox() is not None:
        raise ValueError(f'獨立面板框／槽／填條不符：{different.getbbox()}')
    return mask.histogram()[255]


def verify(receipt, original_pack):
    document = json.loads(receipt.read_text())
    source = source_indices(original_pack)
    seen, phases, checked = set(), set(), 0
    for sample in document.get('source_shots', {}).values():
        panels = [layer for layer in sample.get('layers', []) if 'panel_file' in layer]
        if not panels:
            continue
        base = checked_image(receipt.parent, sample['base_file'], sample['base_sha256'], (320, 200))
        if base.mode != 'P':
            raise ValueError('同次底圖缺原始色盤')
        phases.add(sample['phase'])
        for layer in panels:
            if ([layer.get('width'), layer.get('height')] != [149, 42]
                    or layer.get('resource') != -1 or layer.get('preserve_palette') is not True):
                raise ValueError('面板來源身份不同')
            identity = (layer['panel_file'], layer['panel_sha256'], tuple(base.getpalette()),
                        json.dumps(layer.get('panel_values'), sort_keys=True))
            if identity in seen:
                continue
            picture = checked_image(receipt.parent, layer['panel_file'], layer['panel_sha256'], (894, 252)).convert('RGBA')
            checked += verify_picture(picture, layer['panel_values'], base.getpalette(), source)
            seen.add(identity)
    if len(seen) < 2 or not phases:
        raise ValueError('缺少實際保存的兩份面板快照')
    return {'status': 'PASS-INDEPENDENT-PANEL-CHROME', 'receipt_sha256': digest(receipt),
            'unique_panel_palette_pairs': len(seen), 'checked_chrome_pixels': checked,
            'phases': sorted(phases), 'scope': 'frozen panel RGBA only; text and complete GPU composition verified separately'}


def verify_gui(receipt, original_pack):
    document = json.loads(receipt.read_text())
    required = ('world_runtime_checked', 'battle_hd_consumer_checked', 'natural_attack_return_checked',
                'new_process_title_load', 'source_save_bytes_equal', 'theme_save_bytes_equal',
                'loaded_units_equal', 'cold_loaded_units_equal')
    if document.get('status') != 'PASS' or any(document.get(key) is not True for key in required):
        raise ValueError('正常GUI缺少攻擊／自然返回／主題或保存冷讀')
    name = 'world-hd-physical-scene.png'
    shot = document['screenshots'][name]
    state = shot['state']
    if (state.get('battle_phase') != 'body' or state.get('battle_scene') is not True
            or state.get('hd_battle') is not True or state.get('hd_theme') is not True):
        raise ValueError('需要正式HD物理戰鬥畫面')
    picture = checked_image(receipt.parent, name, shot['sha256'], (1920, 1200)).convert('RGBA')
    checked = 0
    for ox, oy in ((0, 924), (1026, 24)):
        # These left-edge/header points are unobscured in this recorded GUI
        # scene. Verify geometry only; do not infer a palette from screenshot RGB.
        border = picture.getpixel((ox + 1, oy + 60))
        background = picture.getpixel((ox + 4, oy + 60))
        if border == background:
            raise ValueError('未辨識三HD像素框與底板')
        points = [(x, y, border) for x in range(3) for y in (48, 54, 60)]
        points += [(x, y, background) for x in (3, 4, 12) for y in (48, 54, 60)]
        points += [(x, y, border) for x in (30, 48, 300) for y in (120, 121)]
        points += [(x, y, background) for x in (30, 48, 300) for y in (122, 123)]
        for y in (132, 186):
            points += [(126, y + 3, border), (300, y, border)]
            if picture.getpixel((ox + 127, oy + y + 3)) == border:
                raise ValueError('bar仍是低解析粗框')
        for x, y, want in points:
            if picture.getpixel((ox + x, oy + y)) != want:
                raise ValueError(f'正常GUI框／槽幾何不同：{ox+x},{oy+y}')
        checked += len(points) + 2
    return {'status': 'PASS-NORMAL-GUI-PANEL-CHROME', 'receipt_sha256': digest(receipt),
            'image_sha256': shot['sha256'], 'original_manifest_sha256': digest(original_pack / 'manifest.json'),
            'unoccluded_geometry_points': checked, 'panels': 2,
            'scope': 'normal keyboard attack, F2, natural return and existing save contract; sampled chrome geometry only; no chapter advancement or original RGB claim'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('receipt', 'original-pack', 'out'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--normal-gui', action='store_true', help='核對正常GUI收據及兩面板未遮擋的HD框／槽幾何')
    a = parser.parse_args()
    if a.out.exists():
        raise SystemExit('不覆寫既有驗證收據')
    result = verify_gui(a.receipt, a.original_pack) if a.normal_gui else verify(a.receipt, a.original_pack)
    a.out.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(result, ensure_ascii=False))


if __name__ == '__main__':
    main()
