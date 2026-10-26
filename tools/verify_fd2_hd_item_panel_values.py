#!/usr/bin/env python3
"""獨立核對共用狀態面板HD標籤、血條及替換區外的完整PNG。"""
import argparse
import copy
import hashlib
import json
import struct
from pathlib import Path

from PIL import Image, ImageChops
from verify_fd2_hd_item_panel_chrome import native_regions, validate as validate_chrome
from verify_fd2_hd_transient_expiry import require

LABELS = ((139, 33, 157, 41), (139, 44, 157, 52), (138, 55, 157, 63),
          (99, 55, 117, 63), (96, 67, 117, 75), (99, 79, 117, 87),
          (139, 67, 157, 75), (139, 79, 157, 87), (177, 32, 197, 40),
          (177, 51, 197, 59), (286, 41, 293, 49), (286, 59, 293, 67))
BARS = ((197, 32, 302, 39), (197, 51, 302, 58))


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def source_colors(pack):
    palette = json.loads((pack/'palette/fdother_000.json').read_text())
    require(palette['source']['sha256'] == 'a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce', '原始色盤版本不同')
    dac = palette['dac_6bit_components']
    require(len(dac) == 768 and all(type(v) is int and 0 <= v <= 63 for v in dac), 'DAC不合法')
    root = pack/'ui/fdother_005_item_panel'
    colors = {}
    for name, index, xy in [('background', 20, (100, 70)), ('border', 20, (1, 1)), ('hp', 24, (0, 1)), ('mp', 27, (0, 1)), ('track', 29, (0, 1))]:
        with Image.open(root/f'entry_{index:03d}/frame.png') as picture:
            require(picture.mode == 'P', '來源必須保留索引')
            pixel = picture.getpixel(xy)
        colors[name] = tuple((v << 2) | (v >> 4) for v in dac[pixel*3:pixel*3+3])+(255,)
    return colors


def regions_for(row, full):
    phase = row['phase']
    if full:
        if not phase.startswith('item-'):
            return []
        step = row['item_anim_step']
        return native_regions(step if row['item_closing'] else 11-step)
    frame = row['frame']
    if phase == 'return':
        return []
    if phase == 'opening':
        return native_regions(11-frame)
    if phase == 'closing':
        return native_regions(frame)
    if phase == 'transition':
        return native_regions(0, frame if frame < 7 else 13-frame)
    return native_regions(0)


def upper(regions):
    return next((r for r in regions if r['Enabled'] and r['SourceX'] >= 92 and r['SourceY'] >= 7 and
                 r['SourceX']+r['Width'] <= 315 and r['SourceY']+r['Height'] <= 93), None)


def area(rect, region):
    if region is None:
        return None
    x0, y0, x1, y1 = rect
    x0, y0 = max(x0, region['SourceX']), max(y0, region['SourceY'])
    x1, y1 = min(x1, region['SourceX']+region['Width']), min(y1, region['SourceY']+region['Height'])
    if x0 >= x1 or y0 >= y1:
        return None
    dx, dy = region['DestX']-region['SourceX'], region['DestY']-region['SourceY']
    return tuple(v*6 for v in (x0+dx, y0+dy, x1+dx, y1+dy))


def bar_count(region):
    boxes = [area(r, region) for r in BARS]
    return sum((b[2]-b[0])*(b[3]-b[1]) for b in boxes if b)


def check_picture(picture, previous, row, region, colors):
    require(picture.size == previous.size == (1920, 1200), '必須是完整HD畫布')
    picture, previous = picture.convert('RGBA'), previous.convert('RGBA')
    require(picture.getchannel('A').getextrema() == (255, 255), '完整畫布不是不透明')
    mask = Image.new('L', picture.size, 255)
    for rect in LABELS+BARS:
        box = area(rect, region)
        if box:
            mask.paste(0, box)
    diff = ImageChops.difference(picture, previous)
    require(all(ImageChops.multiply(channel, mask).getbbox() is None for channel in diff.split()), '替換區外畫面不同')
    unchanged = mask.histogram()[255]
    if region is None:
        return dict(bar_pixels=0, label_ink_pixels=0, unchanged_pixels=unchanged)
    pixels = picture.load()
    record = bytes.fromhex(row['item_values_record'])
    dx, dy = (region['DestX']-region['SourceX'])*6, (region['DestY']-region['SourceY'])*6
    count = ink = 0
    for i, (x0, y0, x1, y1) in enumerate(BARS):
        box = area((x0, y0, x1, y1), region)
        if box is None:
            continue
        current, maximum = struct.unpack_from('<hh', record, 64+i*4)
        length = 0
        if current and maximum:
            # Explicit truncation toward zero, including signed inputs.
            dividend = 101*current
            quotient = abs(dividend)//abs(maximum)
            if (dividend < 0) != (maximum < 0):
                quotient = -quotient
            length = min(103, max(1, quotient+1)+1)*6
        by = (33+19*i)*6
        left, top, right, bottom = 198*6, by, 301*6, by+30
        for y in range(box[1], box[3]):
            for x in range(box[0], box[2]):
                sx, sy = x-dx, y-dy
                want = colors['background']
                if left <= sx < right and top <= sy < bottom:
                    want = colors['hp' if i == 0 else 'mp'] if sx-left < length else colors['track']
                    if sx in (left, right-1) or sy in (top, bottom-1):
                        want = colors['border']
                require(all(abs(a-b) <= 1 for a, b in zip(pixels[x, y], want)), f'HD血條／舊邊帶不同：{x},{y}')
                count += 1
    for rect in LABELS:
        box = area(rect, region)
        if box is None or (box[2]-box[0], box[3]-box[1]) != ((rect[2]-rect[0])*6, (rect[3]-rect[1])*6):
            continue
        crop = picture.crop(box)
        require(crop.tobytes() != previous.crop(box).tobytes(), '固定標籤仍是舊像素圖')
        pixels = crop.getdata()
        label_ink = sum(1 for p in pixels if max(abs(a-b) for a, b in zip(p, colors['background'])) > 4)
        require(label_ink > 10, 'HD標籤沒有實際墨跡')
        ink += label_ink
    require(count == bar_count(region), '血條獨立核對範圍不足')
    return dict(bar_pixels=count, label_ink_pixels=ink, unchanged_pixels=unchanged)


def validate(receipt, previous, colors, images=True, data=None):
    d = json.loads(receipt.read_text()) if data is None else data
    require(d.get('item_values') is True, '缺少HD數值旗標')
    validate_chrome(receipt, False, d)
    full = d['status'] == 'PASS-FULL-REWARD-HD-RUNTIME-E1'
    count = png_points = ink = unchanged = 0
    captures = []
    for row in d['rows']:
        region = upper(regions_for(row, full))
        points = bar_count(region)
        require(type(row.get('item_values_pixels')) is int and row['item_values_pixels'] == points, '逐拍血條點數不同')
        if region:
            require(type(row.get('item_values_record')) is str and len(row['item_values_record']) == 160, '封存record缺件')
            require(len(bytes.fromhex(row['item_values_record'])) == 80, 'record格式不同')
            expected = {k:dict(zip('RGBA', v)) for k, v in colors.items() if k != 'ink'}
            require(row.get('item_values_colors') == expected, '血條色盤不符原始來源')
        count += points
        if not images or not row.get('capture'):
            continue
        name = row['capture']
        require(Path(name).name == name and name.endswith('.png'), '危險截圖路徑')
        p = receipt.parent/name
        with Image.open(p) as pic, Image.open(previous/name) as old:
            require(hashlib.sha256(pic.convert('RGBA').tobytes()).hexdigest() == row['final_sha256'], '完整PNG雜湊不同')
            checks = check_picture(pic, old, row, region, colors)
        png_points += checks['bar_pixels']; ink += checks['label_ink_pixels']; unchanged += checks['unchanged_pixels']
        captures.append(dict(file=name, sha256=sha(p), **checks))
    require(count > 0 and d.get('item_values_pixels') == count, '血條總量不同')
    return dict(status='PASS-ITEM-PANEL-VALUES-INDEPENDENT', frames=len(d['rows']), go_bar_pixels=count,
                png_bar_pixels=png_points, label_ink_pixels=ink, unchanged_pixels=unchanged,
                captures=captures, receipt_sha256=sha(receipt), normal_player_verified=False)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--receipt', type=Path, required=True)
    p.add_argument('--previous', type=Path, required=True)
    p.add_argument('--original-pack', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--self-test', action='store_true')
    p.add_argument('--reject-picture', type=Path, action='append', default=[])
    a = p.parse_args()
    require(not a.output.exists() and a.output.parent.is_dir(), '不覆寫收據')
    colors = source_colors(a.original_pack)
    result = validate(a.receipt, a.previous, colors)
    if a.self_test:
        original = json.loads(a.receipt.read_text())
        selected = next(i for i, r in enumerate(original['rows']) if r.get('item_values_pixels', 0) > 0)
        mutators = [lambda d:d.update(item_values=False), lambda d:d.update(item_values_pixels=0),
                    lambda d:d['rows'][selected].update(item_values_pixels=0), lambda d:d['rows'][selected].update(item_values_record='00'),
                    lambda d:d['rows'][selected]['item_values_colors']['hp'].update(R=0),
                    lambda d:d.update(normal_player_verified=True), lambda d:d['rows'].pop(), lambda d:d.update(native_seed=8)]
        for mutate in mutators:
            changed = copy.deepcopy(original); mutate(changed)
            try:
                validate(a.receipt, a.previous, colors, False, changed)
            except (ValueError, KeyError):
                continue
            raise ValueError('變造數值收據未被拒收')
        result['rejection_checks'] = len(mutators)
        stable = next(r for r in original['rows'] if r.get('capture') and r['phase'] in ('item-choice', 'status'))
        old = a.previous/stable['capture']
        with Image.open(old) as pic:
            try:
                check_picture(pic, pic, stable, upper(regions_for(stable, original['status']=='PASS-FULL-REWARD-HD-RUNTIME-E1')), colors)
            except ValueError as error:
                result['old_picture_rejected'] = dict(file=str(old), sha256=sha(old), reason=str(error))
            else:
                raise ValueError('真實舊數值PNG未被拒收')
    if a.reject_picture:
        original = json.loads(a.receipt.read_text())
        full = original['status'] == 'PASS-FULL-REWARD-HD-RUNTIME-E1'
        stable = next(r for r in original['rows'] if r.get('capture') and r['phase'] in ('item-choice', 'status'))
        result['rejected_pictures'] = []
        for path in a.reject_picture:
            with Image.open(path) as picture:
                try:
                    check_picture(picture, picture, stable, upper(regions_for(stable, full)), colors)
                except ValueError as error:
                    result['rejected_pictures'].append(dict(file=str(path), sha256=sha(path), reason=str(error)))
                else:
                    raise ValueError('指定錯誤底色PNG未被拒收')
    a.output.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({k:v for k,v in result.items() if k!='captures'}, ensure_ascii=False))


if __name__ == '__main__':
    main()
