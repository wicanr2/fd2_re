#!/usr/bin/env python3
"""獨立解碼原始物品列，核對HD縮寫、實際墨跡及完整PNG區外保持。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageChops
from verify_fd2_hd_item_panel_chrome import native_regions, validate as validate_chrome
from verify_fd2_hd_item_panel_values import regions_for
from verify_fd2_hd_transient_expiry import require

LABELS = {64: '+AP', 65: '+DP', 66: '+HP', 67: '+MP', 92: '-MP'}


def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()


def sources(original, assets):
    palette = json.loads((original/'palette/fdother_000.json').read_text())
    require(palette['source']['sha256'] == 'a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce', '色盤來源版本不同')
    dac = palette['dac_6bit_components']
    require(len(dac) == 768 and all(type(x) is int and 0 <= x <= 63 for x in dac), 'DAC不合法')
    colors = [tuple((v << 2) | (v >> 4) for v in dac[i:i+3])+(255,) for i in range(0, 768, 3)]
    art = original/'ui/fdother_005_item_panel'
    with Image.open(art/'entry_021/frame.png') as pic:
        require(pic.mode == 'P' and pic.size == (310, 99), '下框來源形狀不同')
        background = pic.getpixel((100, 70))
    inks = {}
    for source in LABELS:
        with Image.open(art/f'entry_{source:03d}/frame.png') as pic:
            require(pic.mode == 'P' and pic.size == (20 if source == 92 else 21, 8) and pic.getpixel((0, 0)) == background, '字格來源／底色不同')
            indices = [v for v in pic.getdata() if v != background]
        require(indices, '來源字格沒有墨跡')
        inks[source] = colors[max(indices, key=lambda v: sum(colors[v][:3]))]
    encoded = json.loads((assets/'data/native_item_effect_rows.json').read_text())
    rows = []
    for i, row in enumerate(encoded):
        raw = bytes.fromhex(row['raw'])
        require(row['id'] == i and len(raw) == 23, '物品列不是連續23-byte原始資料')
        rows.append(raw)
    book = json.loads((assets/'spells.json').read_text())
    require(len(book) == 36 and all(row['id'] == i and len(bytes.fromhex(row['raw'])) == 7 for i, row in enumerate(book)), '指令來源表不同')
    return colors[background], inks, rows


def expected_fields(row, item_rows):
    fields = []
    encoded = row.get('row_label_record')
    require(type(row.get('row_label_commands')) is bool, '列模式不是布林值')
    if encoded is None:
        require(row['row_label_commands'] is False, '指令列缺少record')
        return fields
    require(type(encoded) is str and len(encoded) == 160, '列record形狀不同')
    record = bytes.fromhex(encoded)
    if row['row_label_commands']:
        ids = row.get('row_label_command_ids')
        require(type(ids) is list and len(ids) <= 12 and len(set(ids)) == len(ids), '指令ID清單不同')
        for i, command in enumerate(ids):
            require(type(command) is int and 0 <= command < 36, '指令ID超界')
            x, y = 68+100*(i//4), 108+22*(i%4)
            fields.append(dict(source=92, text='-MP', rect=[x, y, x+20, y+8], command_id=command))
    else:
        display = 0
        for slot in range(8):
            if record[10+slot*2] & 128:
                continue
            item = record[11+slot*2]
            require(item < len(item_rows), '物品ID超出原始資料')
            raw = item_rows[item]
            source = 64 if raw[0] < 0x15 else 65 if raw[0] < 0x20 else 66 if raw[0] == 0x20 and raw[13] == 5 else 67 if raw[0] == 0x20 and raw[13] == 11 else None
            if source:
                x, y = 110+150*(display//4), 107+22*(display%4)
                fields.append(dict(source=source, text=LABELS[source], rect=[x, y, x+21, y+8], raw_slot=slot, item_id=item))
            display += 1
    return fields


def field_area(rect, regions):
    r = next((x for x in regions if x['Enabled'] and x['SourceX'] == 5 and x['SourceY'] == 94), None)
    if r is None:
        return None
    x0, y0 = max(rect[0], r['SourceX']), max(rect[1], r['SourceY'])
    x1, y1 = min(rect[2], r['SourceX']+r['Width']), min(rect[3], r['SourceY']+r['Height'])
    if x0 >= x1 or y0 >= y1:
        return None
    dx, dy = r['DestX']-r['SourceX'], r['DestY']-r['SourceY']
    return tuple(v*6 for v in (x0+dx, y0+dy, x1+dx, y1+dy))


def check_picture(picture, previous, fields, regions, background, inks):
    require(picture.size == previous.size == (1920, 1200), '必須是完整1920×1200 PNG')
    picture, previous = picture.convert('RGBA'), previous.convert('RGBA')
    require(picture.getchannel('A').getextrema() == (255, 255), '完整畫布不是不透明')
    mask = Image.new('L', picture.size, 255)
    ink_count = solid_count = visible = 0
    for field in fields:
        box = field_area(field['rect'], regions)
        if box is None:
            continue
        mask.paste(0, box)
        visible += (box[2]-box[0])*(box[3]-box[1])
        crop = picture.crop(box)
        ink = inks[field['source']]
        # The one-color Noto glyph may only blend its real source ink and BG.
        for pixel in crop.getdata():
            require(all(min(a, b)-1 <= c <= max(a, b)+1 for a, b, c in zip(background, ink, pixel)), '字區出現非來源墨色或舊陰影')
            if max(abs(a-b) for a, b in zip(pixel, background)) > 4:
                ink_count += 1
            if max(abs(a-b) for a, b in zip(pixel, ink)) <= 1:
                solid_count += 1
        rect = field['rect']
        if (box[2]-box[0], box[3]-box[1]) == ((rect[2]-rect[0])*6, (rect[3]-rect[1])*6):
            require(crop.tobytes() != previous.crop(box).tobytes(), '實際縮寫仍是舊像素圖')
            require(any(max(abs(a-b) for a, b in zip(p, ink)) <= 1 for p in crop.getdata()), '完整字區缺少來源實色墨跡')
    diff = ImageChops.difference(picture, previous)
    require(all(ImageChops.multiply(c, mask).getbbox() is None for c in diff.split()), '縮寫區外畫面不同')
    return dict(visible_label_pixels=visible, label_ink_pixels=ink_count, solid_ink_pixels=solid_count, unchanged_pixels=mask.histogram()[255])


def validate(receipt, previous, source, images=True, data=None):
    d = json.loads(receipt.read_text()) if data is None else data
    require(d.get('row_labels') is True, '缺少HD列縮寫旗標')
    validate_chrome(receipt, False, d)
    full = d['status'] == 'PASS-FULL-REWARD-HD-RUNTIME-E1'
    background, inks, item_rows = source
    captures = []
    totals = dict(visible_label_pixels=0, label_ink_pixels=0, solid_ink_pixels=0, unchanged_pixels=0)
    for row in d['rows']:
        commands = not full and (row['phase'] in ('commands', 'closing') or row['phase'] == 'transition' and row['frame'] >= 7)
        require(row.get('row_label_commands') is commands, '逐拍物品／指令模式不同')
        fields = expected_fields(row, item_rows)
        require(row.get('row_label_fields') == fields, '原始資料導出的列縮寫或排列不同')
        if not images or not row.get('capture'):
            continue
        name = row['capture']
        require(Path(name).name == name and name.endswith('.png'), '截圖路徑不合法')
        p = receipt.parent/name
        with Image.open(p) as pic, Image.open(previous/name) as old:
            require(hashlib.sha256(pic.convert('RGBA').tobytes()).hexdigest() == row['final_sha256'], 'PNG與完整畫布雜湊不同')
            checks = check_picture(pic, old, fields, regions_for(row, full), background, inks)
        for key, value in checks.items():
            totals[key] += value
        captures.append(dict(file=name, sha256=sha(p), **checks))
    banks = d.get('row_label_banks')
    if not full:
        require(type(banks) is list and [r.get('phase') for r in banks] == ['bank-items', 'bank-commands', 'bank-empty'], '三種補樣缺項')
        for index, row in enumerate(banks):
            fields = expected_fields(row, item_rows)
            require(row.get('row_label_fields') == fields and len(fields) == [8, 12, 0][index], '補樣來源或數量不同')
            require(row.get('regions') == native_regions(0) and row.get('f2_roundtrip') is True and row.get('native_control_unchanged') is True, '補樣F2／原生控制不同')
            require(row.get('before_capture') == row['phase']+'-before.png' and row.get('capture') == row['phase']+'.png', '補樣圖來源不同')
            require(row.get('row_label_commands') is (index == 1), '補樣模式不同')
            if index == 0:
                require({f['source'] for f in fields} == {64, 65, 66, 67}, '四種真實物品列未齊')
            if not images:
                continue
            path, before = receipt.parent/row['capture'], receipt.parent/row['before_capture']
            with Image.open(path) as pic, Image.open(before) as old:
                require(hashlib.sha256(pic.convert('RGBA').tobytes()).hexdigest() == row['final_sha256'] and hashlib.sha256(old.convert('RGBA').tobytes()).hexdigest() == row['before_sha256'], '補樣PNG與完整畫布不同')
                checks = check_picture(pic, old, fields, native_regions(0), background, inks)
            for key, value in checks.items():
                totals[key] += value
            captures.extend([dict(file=row['capture'], sha256=sha(path), **checks), dict(file=row['before_capture'], sha256=sha(before), role='same-state HD without new row patch')])
    require(not images or totals['visible_label_pixels'] > 0 and totals['label_ink_pixels'] > 0, '新縮寫沒有可見正對照')
    return dict(status='PASS-HD-ITEM-ROW-LABELS', frames=len(d['rows']), bank_samples=0 if full else 3,
                captures=captures, receipt_sha256=sha(receipt), normal_player_verified=False, **totals)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for key in ['receipt', 'previous', 'original-pack', 'assets-root', 'output']:
        p.add_argument('--'+key, type=Path, required=True)
    p.add_argument('--self-test', action='store_true')
    a = p.parse_args()
    require(not a.output.exists() and a.output.parent.is_dir(), '不覆寫收據')
    source = sources(a.original_pack, a.assets_root)
    result = validate(a.receipt, a.previous, source)
    if a.self_test:
        d = json.loads(a.receipt.read_text())
        full = d['status'] == 'PASS-FULL-REWARD-HD-RUNTIME-E1'
        selected = next(i for i, row in enumerate(d['rows']) if any(field_area(f['rect'], regions_for(row, full)) for f in row['row_label_fields']))
        changes = [lambda x:x.update(row_labels=False), lambda x:x['rows'].pop(), lambda x:x.update(native_seed=8),
                   lambda x:x['rows'][selected]['row_label_fields'][0].update(source=65),
                   lambda x:x['rows'][selected]['row_label_fields'][0]['rect'].__setitem__(0, 0),
                   lambda x:x['rows'][selected].update(row_label_record='00'),
                   lambda x:x['rows'][selected].update(row_label_commands='false'),
                   lambda x:x['rows'][selected].pop('row_label_fields')]
        if not full:
            changes += [lambda x:x['row_label_banks'].pop(), lambda x:x['row_label_banks'][1].update(f2_roundtrip=False)]
        for change in changes:
            changed = copy.deepcopy(d);change(changed)
            try:
                validate(a.receipt, a.previous, source, False, changed)
            except (ValueError, KeyError):
                continue
            raise ValueError('變造列縮寫收據未被拒收')
        result['rejection_checks'] = len(changes)
        stable = next(row for row in d['rows'] if row.get('capture') and row['phase'] in ('item-choice', 'status'))
        path = a.previous/stable['capture']
        with Image.open(path) as old:
            try:
                check_picture(old, old, stable['row_label_fields'], regions_for(stable, full), *source[:2])
            except ValueError as error:
                result['old_picture_rejected'] = dict(file=str(path), sha256=sha(path), reason=str(error))
            else:
                raise ValueError('真實舊縮寫PNG未被拒收')
    a.output.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({k:v for k,v in result.items() if k!='captures'}, ensure_ascii=False))


if __name__ == '__main__':
    main()
