#!/usr/bin/env python3
"""獨立依原始record與物品列核對十個HD圖示、完整PNG及矩形外保持。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageChops
from verify_fd2_hd_item_panel_chrome import native_regions, validate as validate_chrome
from verify_fd2_hd_item_panel_values import regions_for
from verify_fd2_hd_transient_expiry import require

INDICES = {53, 54, 55, 56, 57, 59, 60, 61, 62, 63}
META_SHA = 'a547f2f593b980fd9c1b269d1ba95be8727562d7ed31b44015615775d991efd3'


def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()


def sources(catalog, pack, original, assets):
    require(sha(original/'ui/fdother_005_item_panel/resource.json') == META_SHA, '原包metadata版本不同')
    data = json.loads(catalog.read_text())
    entries = [a for a in data['assets'] if a['role'] == 'native_ui_icon_hd']
    require(len(entries) == 10 and {a['native_raw_cell']['index'] for a in entries} == INDICES, '十圖示清冊不完整')
    pictures = {}
    for a in entries:
        d = a['native_raw_cell'];index = d['index'];size = (29, 24) if index in (53, 54) else (24, 20)
        require((d['width'], d['height']) == size and a['source_metadata_sha256'] == META_SHA and a['consumer_contract'] == 'native_raw_ui_icon_hd6_v1', '圖示契約不同')
        require(Path(a['file']).name == a['file'], '圖示路徑不安全')
        with Image.open(original/f'ui/fdother_005_item_panel/entry_{index:03d}/frame.png') as native:
            require(native.mode == 'P' and native.size == size and hashlib.sha256(bytes(native.getdata())).hexdigest() == d['pixels_sha256'], '原始raw圖示不同')
        path = pack/a['file']
        require(sha(path) == a['sha256'], 'HD圖示SHA不同')
        with Image.open(path) as pic:
            rgba = pic.convert('RGBA')
        require(rgba.size == (size[0]*6, size[1]*6) and rgba.getchannel('A').getextrema() == (255, 255), 'HD圖示尺寸或alpha不同')
        pictures[index] = rgba
    rows = json.loads((assets/'data/native_item_effect_rows.json').read_text())
    item_rows = []
    for i, row in enumerate(rows):
        raw = bytes.fromhex(row['raw'])
        require(row['id'] == i and len(raw) == 23, '原始物品列格式不同')
        item_rows.append(raw)
    return pictures, item_rows


def expected_fields(row, items):
    fields = []
    encoded = row.get('row_label_record')
    require(type(row.get('row_label_commands')) is bool, '列模式缺項')
    if encoded is None:
        require(row['row_label_commands'] is False, '指令缺少record')
        return fields
    require(type(encoded) is str and len(encoded) == 160, 'record不是80 bytes')
    record = bytes.fromhex(encoded)
    fields.append(dict(source=53 if record[6] else 54, rect=[101, 30, 130, 54]))
    for i in range(3):
        if record[37+i]:
            x = 194+35*i
            fields.append(dict(source=55+i, rect=[x, 68, x+24, 88]))
    if not row['row_label_commands']:
        display = 0
        for slot in range(8):
            flag, item = record[10+2*slot:12+2*slot]
            if flag & 128:
                continue
            require(item < len(items), '物品ID超界')
            kind = items[item][0]
            source = (59 if kind < 0x15 else 60 if kind < 0x20 else 61)+(3 if flag & 64 else 0)
            if source in INDICES:
                x, y = 13+150*(display//4), 101+22*(display%4)
                fields.append(dict(source=source, rect=[x, y, x+24, y+20], raw_slot=slot, item_id=item))
            display += 1
    return fields


def placement(rect, regions):
    for r in regions:
        if not r['Enabled']:
            continue
        x0, y0 = max(rect[0], r['SourceX']), max(rect[1], r['SourceY'])
        x1, y1 = min(rect[2], r['SourceX']+r['Width']), min(rect[3], r['SourceY']+r['Height'])
        if x0 < x1 and y0 < y1:
            dx, dy = r['DestX']-r['SourceX'], r['DestY']-r['SourceY']
            return tuple(v*6 for v in (x0+dx, y0+dy, x1+dx, y1+dy)), tuple(v*6 for v in (x0-rect[0], y0-rect[1], x1-rect[0], y1-rect[1]))
    return None


def check_picture(picture, previous, fields, regions, icons):
    require(picture.size == previous.size == (1920, 1200), '必須是完整1920×1200')
    pic, old = picture.convert('RGBA'), previous.convert('RGBA')
    require(pic.getchannel('A').getextrema() == (255, 255), '完整PNG不透明度不同')
    mask = Image.new('L', pic.size, 255)
    points = 0
    for field in fields:
        areas = placement(field['rect'], regions)
        if areas is None:
            continue
        box, source = areas
        expected = icons[field['source']].crop(source)
        require(pic.crop(box).tobytes() == expected.tobytes(), 'HD圖示與已登錄圖源或裁切不同')
        mask.paste(0, box)
        points += expected.width*expected.height
    diff = ImageChops.difference(pic, old)
    require(all(ImageChops.multiply(c, mask).getbbox() is None for c in diff.split()), '圖示矩形外畫面不同')
    return dict(icon_pixels=points, unchanged_pixels=mask.histogram()[255])


def validate(receipt, previous, source, images=True, data=None):
    d = json.loads(receipt.read_text()) if data is None else data
    require(d.get('panel_icons') is True, '沒有HD圖示旗標')
    validate_chrome(receipt, False, d)
    full = d['status'] == 'PASS-FULL-REWARD-HD-RUNTIME-E1'
    icons, items = source
    captures = [];points = outside = 0
    for row in d['rows']:
        commands = not full and (row['phase'] in ('commands', 'closing') or row['phase'] == 'transition' and row['frame'] >= 7)
        require(row.get('row_label_commands') is commands, '物品／指令模式不同')
        fields = expected_fields(row, items)
        require(row.get('panel_icon_fields') == fields, '圖示未依原始record或原始緊密排列')
        if not images or not row.get('capture'):
            continue
        name = row['capture'];require(Path(name).name == name, '截圖路徑不合法')
        path = receipt.parent/name
        with Image.open(path) as pic, Image.open(previous/name) as old:
            require(hashlib.sha256(pic.convert('RGBA').tobytes()).hexdigest() == row['final_sha256'], 'PNG與完整畫布不同')
            check = check_picture(pic, old, fields, regions_for(row, full), icons)
        points += check['icon_pixels'];outside += check['unchanged_pixels']
        captures.append(dict(file=name, sha256=sha(path), **check))
    if not full:
        banks = d.get('row_label_banks')
        require(type(banks) is list and [r.get('phase') for r in banks] == ['bank-items', 'bank-commands', 'bank-empty'], '三補樣缺項')
        require({f['source'] for r in banks for f in r.get('panel_icon_fields', [])} == INDICES, '十圖示未全部補樣')
        for row in banks:
            fields = expected_fields(row, items)
            require(row.get('panel_icon_fields') == fields and row.get('control_kind') == 'HD-without-panel-icons', '補樣或控制組不同')
            require(row.get('regions') == native_regions(0) and row.get('f2_roundtrip') is True and row.get('native_control_unchanged') is True, '補樣F2或原生控制不同')
            require(row.get('capture') == row['phase']+'.png' and row.get('before_capture') == row['phase']+'-before.png', '補樣圖來源不同')
            if not images:
                continue
            path, before = receipt.parent/row['capture'], receipt.parent/row['before_capture']
            with Image.open(path) as pic, Image.open(before) as old:
                require(hashlib.sha256(pic.convert('RGBA').tobytes()).hexdigest() == row['final_sha256'] and hashlib.sha256(old.convert('RGBA').tobytes()).hexdigest() == row['before_sha256'], '補樣PNG不同')
                check = check_picture(pic, old, fields, native_regions(0), icons)
            points += check['icon_pixels'];outside += check['unchanged_pixels']
            captures.extend([dict(file=row['capture'], sha256=sha(path), **check), dict(file=row['before_capture'], sha256=sha(before), role='same-state HD without panel icons')])
    require(not images or points > 0, '沒有可見圖示正對照')
    return dict(status='PASS-HD-ITEM-PANEL-ICONS', frames=len(d['rows']), bank_samples=0 if full else 3, captures=captures, icon_pixels=points, unchanged_pixels=outside, receipt_sha256=sha(receipt), normal_player_verified=False)


def validate_normal(receipt, catalog, pack, icons):
    d = json.loads(receipt.read_text())
    require(d.get('status') == 'PASS' and d.get('new_process_title_load') is True
            and d.get('theme_save_bytes_equal') is True and d.get('loaded_units_equal') is True
            and d.get('cold_loaded_units_equal') is True and d.get('recording_created') is False,
            '正常玩家保存或新程序LOAD缺項')
    require(d.get('catalog_sha256') == sha(catalog), '正常流程清冊不同')
    data = json.loads(catalog.read_text())
    heads = {a['speaker_id']: a for a in data['assets'] if a['role'] == 'story_portrait_hd_frame'}
    rows = []
    for name in ('battle-hd-status.png', 'battle-hd-status-restored.png',
                 'battle-hd-command-unit-status.png', 'battle-hd-commands.png'):
        record = d['screenshots'][name]
        state = record['state']
        require(state.get('hd_theme') is True and state.get('error') == '', '正常面板未使用HD或有錯誤')
        require(state.get('player_status_phase') == ('commands' if name == 'battle-hd-commands.png' else 'status'), '正常面板階段不同')
        units = [u for u in state['units'] if u.get('on_field') is True and u.get('camp') == 0
                 and [u.get('x'), u.get('y')] == state['cursor']]
        require(len(units) == 1 and units[0].get('battle_fig') in heads, '正常游標沒有唯一DATO來源')
        speaker = units[0]['battle_fig']
        path = receipt.parent/name
        require(sha(path) == record['sha256'], '正常擷取SHA不同')
        head = heads[speaker]
        require(sha(pack/head['file']) == head['sha256'], '正常頭像來源不同')
        with Image.open(path) as pic, Image.open(pack/head['file']) as portrait:
            require(pic.size == (1920, 1200) and portrait.size == (480, 480), '正常完整畫布或頭像尺寸不同')
            rgba = pic.convert('RGBA')
            require(rgba.crop((48, 60, 528, 540)).tobytes() == portrait.convert('RGBA').tobytes(), '正常面板沒有使用已登錄480頭像')
            cell = rgba.crop((606, 180, 780, 324))
            matches = [index for index in (53, 54) if cell.tobytes() == icons[index].tobytes()]
            require(len(matches) == 1, '正常面板沒有使用已登錄HD頭徽')
        rows.append(dict(file=name, sha256=sha(path), speaker=speaker, icon=matches[0], portrait_pixels=480*480, icon_pixels=174*144))
    returned = d['screenshots']['world-hd-physical-return.png']['state']
    require(returned.get('hd_world') is True and returned.get('battle_scene') is False
            and returned.get('battle_event') is False, '普通攻防未自然返回HD戰場')
    return dict(status='PASS-NORMAL-GUI-HD-PANEL-ICONS', receipt_sha256=sha(receipt), captures=rows,
                source_save_sha256=d['source_save_sha256'], binary_sha256=d['binary_sha256'],
                source_save_bytes_equal=d['source_save_bytes_equal'], normal_player_scope=d['scope'])


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for key in ['receipt', 'previous', 'catalog', 'pack', 'original-pack', 'assets-root', 'output']:
        p.add_argument('--'+key, type=Path, required=True)
    p.add_argument('--self-test', action='store_true')
    p.add_argument('--normal-receipt', type=Path, help='另核對既有首章正常GUI收據的狀態／指令頭像及頭徽')
    a = p.parse_args();require(not a.output.exists(), '不覆寫報告')
    source = sources(a.catalog, a.pack, a.original_pack, a.assets_root)
    result = validate(a.receipt, a.previous, source)
    if a.normal_receipt:
        result['normal_gui'] = validate_normal(a.normal_receipt, a.catalog, a.pack, source[0])
    if a.self_test:
        d = json.loads(a.receipt.read_text());full = result['frames'] == 291
        selected = next(i for i, r in enumerate(d['rows']) if r.get('capture') and any(placement(f['rect'], regions_for(r, full)) for f in r['panel_icon_fields']))
        changes = [lambda x:x.update(panel_icons=False), lambda x:x['rows'].pop(), lambda x:x.update(native_seed=8),
                   lambda x:x['rows'][selected]['panel_icon_fields'][0].update(source=55),
                   lambda x:x['rows'][selected]['panel_icon_fields'][0]['rect'].__setitem__(0, 0),
                   lambda x:x['rows'][selected].update(row_label_record='00'),
                   lambda x:x['rows'][selected].update(row_label_commands='false'),
                   lambda x:x['rows'][selected].pop('panel_icon_fields')]
        if not full:
            changes += [lambda x:x['row_label_banks'].pop(), lambda x:x['row_label_banks'][0].update(control_kind='wrong'), lambda x:x['row_label_banks'][1].update(f2_roundtrip=False)]
        for change in changes:
            altered = copy.deepcopy(d);change(altered)
            try:
                validate(a.receipt, a.previous, source, False, altered)
            except (ValueError, KeyError):
                continue
            raise ValueError('變造圖示收據未被拒收')
        result['rejection_checks'] = len(changes)
        stable = next(r for r in d['rows'] if r.get('capture') and r['phase'] in ('item-choice', 'status'))
        path = a.previous/stable['capture']
        with Image.open(path) as old:
            try:
                check_picture(old, old, stable['panel_icon_fields'], regions_for(stable, full), source[0])
            except ValueError as error:
                result['old_picture_rejected'] = dict(file=str(path), sha256=sha(path), reason=str(error))
            else:
                raise ValueError('舊像素圖示實圖未被拒收')
    a.output.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({k:v for k,v in result.items() if k!='captures'}, ensure_ascii=False))


if __name__ == '__main__':
    main()
